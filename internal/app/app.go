package app

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mailpeek/mailpeek/internal/api"
	"github.com/mailpeek/mailpeek/internal/config"
	"github.com/mailpeek/mailpeek/internal/events"
	"github.com/mailpeek/mailpeek/internal/failures"
	"github.com/mailpeek/mailpeek/internal/mail"
	"github.com/mailpeek/mailpeek/internal/smtp"
	"github.com/mailpeek/mailpeek/internal/store"
)

type App struct {
	cfg    config.Config
	store  *store.MemoryStore
	broker *events.Broker
	smtp   *smtp.Server
	http   *http.Server

	smtpLn, httpLn net.Listener
	cancelBase     context.CancelFunc
	errs           chan error
	parsing        chan struct{}
	activity       *log.Logger
	failures       *failures.Set
}

func New(cfg config.Config, ui fs.FS, version string, logger *log.Logger) *App {
	broker := events.NewBroker()
	st := store.New(cfg.MaxMessages, broker).WithMaxBytes(cfg.MaxStoreSize)
	a := &App{
		cfg:     cfg,
		store:   st,
		broker:  broker,
		errs:    make(chan error, 2),
		parsing: make(chan struct{}, max(2, runtime.NumCPU())),
	}

	a.failures = &failures.Set{}
	a.smtp = &smtp.Server{
		Handler:        a.receive,
		Reject:         a.reject,
		MaxMessageSize: cfg.MaxMessageSize,
		ErrorLog:       logger,
	}
	if cfg.SMTPAuth.Enabled() {
		a.smtp.Auth = cfg.SMTPAuth.Match
	}
	var uiAuth func(user, password string) bool
	if cfg.UIAuth.Enabled() {
		uiAuth = cfg.UIAuth.Match
	}

	base, cancel := context.WithCancel(context.Background())
	a.cancelBase = cancel
	a.http = &http.Server{
		Handler: api.New(st, broker, ui, api.Info{
			Version:        version,
			SMTPPort:       cfg.SMTPPort,
			HTTPPort:       cfg.HTTPPort,
			MaxMessages:    cfg.MaxMessages,
			MaxMessageSize: cfg.MaxMessageSize,
			MaxStoreSize:   cfg.MaxStoreSize,
			SMTPAuth:       cfg.SMTPAuth.Enabled(),
			SMTPTLS:        cfg.SMTPTLS,
		}, uiAuth).WithFailures(a.failures),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
		ErrorLog:          logger,
		BaseContext:       func(net.Listener) context.Context { return base },
	}
	return a
}

func (a *App) receive(env smtp.Envelope, data []byte) (string, error) {
	a.parsing <- struct{}{}
	m := mail.Parse(data, mail.Envelope{From: env.From, To: env.To})
	<-a.parsing
	a.store.Save(m, data)
	if a.activity != nil {
		secure := ""
		if env.TLS {
			secure = ", TLS"
		}
		a.activity.Printf("received %s from %s to %s %q (%s%s)",
			m.ID, orNone(env.From), strings.Join(env.To, ", "), m.Subject, humanSize(len(data)), secure)
	}
	return m.ID, nil
}

func (a *App) reject(stage string, to []string) (int, string, bool) {
	r, ok := a.failures.Take(stage, to)
	if !ok {
		return 0, "", false
	}
	if a.activity != nil {
		a.activity.Printf("simulated failure %d at %s for %s (rule %s, %d left)",
			r.Code, strings.ToUpper(stage), strings.Join(to, ", "), r.ID, r.Remaining)
	}
	return r.Code, r.Message, true
}

func (a *App) tlsConfig() (*tls.Config, error) {
	if a.cfg.SMTPTLSCert != "" {
		cert, err := tls.LoadX509KeyPair(a.cfg.SMTPTLSCert, a.cfg.SMTPTLSKey)
		if err != nil {
			return nil, fmt.Errorf("smtp-tls-cert/smtp-tls-key: %w", err)
		}
		return smtp.TLSConfig(cert), nil
	}
	hostname, _ := os.Hostname()
	return smtp.SelfSignedTLS("localhost", "127.0.0.1", "::1", "mailpeek", hostname, a.cfg.Host)
}

func (a *App) SetActivityLog(l *log.Logger) {
	a.activity = l
	a.smtp.ActivityLog = l
}

func orNone(s string) string {
	if s == "" {
		return "<>"
	}
	return s
}

func humanSize(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func (a *App) Start() error {
	var err error
	if a.cfg.SMTPTLS {
		if a.smtp.TLSConfig, err = a.tlsConfig(); err != nil {
			return err
		}
	}
	if a.smtpLn, err = listen(a.cfg.Host, a.cfg.SMTPPort); err != nil {
		return listenError("SMTP", a.cfg.SMTPPort, "--smtp-port", "MAILPEEK_SMTP_PORT", err)
	}
	if a.httpLn, err = listen(a.cfg.Host, a.cfg.HTTPPort); err != nil {
		_ = a.smtpLn.Close()
		return listenError("HTTP", a.cfg.HTTPPort, "--http-port", "MAILPEEK_HTTP_PORT", err)
	}
	go func() {
		if err := a.smtp.Serve(a.smtpLn); !errors.Is(err, smtp.ErrServerClosed) {
			a.errs <- err
		}
	}()
	go func() {
		if err := a.http.Serve(a.httpLn); !errors.Is(err, http.ErrServerClosed) {
			a.errs <- err
		}
	}()
	return nil
}

func (a *App) Err() <-chan error { return a.errs }

func (a *App) SMTPAddr() net.Addr { return a.smtpLn.Addr() }

func (a *App) HTTPAddr() net.Addr { return a.httpLn.Addr() }

func (a *App) Store() *store.MemoryStore { return a.store }

func (a *App) Shutdown(ctx context.Context) error {
	smtpErr := a.smtp.Shutdown(ctx)
	a.broker.Close()
	a.cancelBase()
	httpErr := a.http.Shutdown(ctx)
	return errors.Join(smtpErr, httpErr)
}

func listenError(name string, port int, flag, env string, err error) error {
	switch {
	case errors.Is(err, syscall.EADDRINUSE):
		return fmt.Errorf("%s port %d is already in use (another Mailpeek or mail catcher may be running); "+
			"stop it or choose another port with %s or %s: %w", name, port, flag, env, err)
	case errors.Is(err, syscall.EACCES):
		return fmt.Errorf("not allowed to listen on %s port %d (ports below 1024 need extra privileges); "+
			"choose another port with %s or %s: %w", name, port, flag, env, err)
	}
	return fmt.Errorf("cannot listen on %s port %d: %w", name, port, err)
}
