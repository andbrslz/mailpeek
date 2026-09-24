package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"runtime"
	"syscall"
	"time"

	"github.com/mailpeek/mailpeek/internal/api"
	"github.com/mailpeek/mailpeek/internal/config"
	"github.com/mailpeek/mailpeek/internal/events"
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

	a.smtp = &smtp.Server{
		Handler:        a.receive,
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
		}, uiAuth),
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
	return m.ID, nil
}

func (a *App) Start() error {
	var err error
	if a.smtpLn, err = net.Listen("tcp", a.cfg.SMTPAddr()); err != nil {
		return listenError("SMTP", a.cfg.SMTPPort, "--smtp-port", "MAILPEEK_SMTP_PORT", err)
	}
	if a.httpLn, err = net.Listen("tcp", a.cfg.HTTPAddr()); err != nil {
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
