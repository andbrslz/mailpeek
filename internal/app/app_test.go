package app

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	netsmtp "net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mailpeek/mailpeek/internal/config"
	"github.com/mailpeek/mailpeek/internal/mail"
	"github.com/mailpeek/mailpeek/internal/smtp"
	"github.com/mailpeek/mailpeek/internal/store"
)

func startApp(t *testing.T) (*App, string, string) {
	t.Helper()
	cfg := config.Config{Host: "127.0.0.1", MaxMessages: 10, MaxMessageSize: 1 << 20}
	a := New(cfg, nil, "test", log.New(io.Discard, "", 0))
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	return a, a.SMTPAddr().String(), "http://" + a.HTTPAddr().String()
}

func send(t *testing.T, addr, to, subject string) {
	t.Helper()
	msg := "From: app@example.com\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nContent-Type: text/html\r\n\r\n<a href=\"http://localhost/activate\">Activate account</a>\r\n"
	if err := netsmtp.SendMail(addr, nil, "app@example.com", []string{to, "bcc@example.com"}, []byte(msg)); err != nil {
		t.Fatal(err)
	}
}

func TestEndToEnd(t *testing.T) {
	a, smtpAddr, base := startApp(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := a.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	}()

	sse, err := http.Get(base + "/api/v1/events")
	if err != nil {
		t.Fatal(err)
	}
	defer sse.Body.Close()
	events := bufio.NewReader(sse.Body)

	waitDone := make(chan mail.Message, 1)
	go func() {
		resp, err := http.Get(base + "/api/v1/messages/wait?to=john@example.com&subject=Welcome&timeout=5000")
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		var m mail.Message
		_ = json.NewDecoder(resp.Body).Decode(&m)
		waitDone <- m
	}()
	time.Sleep(50 * time.Millisecond)
	send(t, smtpAddr, "john@example.com", "Welcome")

	select {
	case m := <-waitDone:
		if m.Subject != "Welcome" || len(m.Links) != 1 || m.Links[0].Href != "http://localhost/activate" {
			t.Fatalf("wait returned %+v", m)
		}
		if len(m.Envelope.To) != 2 || m.Envelope.To[1] != "bcc@example.com" {
			t.Fatalf("envelope = %+v", m.Envelope)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("wait did not return")
	}

	for {
		line, err := events.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(line, "event: message.created") {
			break
		}
	}
	if a.Store().Len() != 1 {
		t.Fatalf("store len %d", a.Store().Len())
	}
}

func TestGracefulShutdown(t *testing.T) {
	before := runtime.NumGoroutine()
	a, _, base := startApp(t)
	client := &http.Client{Transport: &http.Transport{}}

	sse, err := client.Get(base + "/api/v1/events")
	if err != nil {
		t.Fatal(err)
	}
	waitStatus := make(chan int, 1)
	go func() {
		resp, err := client.Get(base + "/api/v1/messages/wait?timeout=30000")
		if err != nil {
			waitStatus <- 0
			return
		}
		resp.Body.Close()
		waitStatus <- resp.StatusCode
	}()
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	if err := a.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("shutdown took %v", d)
	}
	if status := <-waitStatus; status != http.StatusServiceUnavailable {
		t.Fatalf("pending wait got status %d", status)
	}
	_, _ = io.Copy(io.Discard, sse.Body)
	sse.Body.Close()
	client.CloseIdleConnections()

	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > before {
		buf := make([]byte, 1<<16)
		t.Fatalf("goroutines: before %d, after %d\n%s", before, n, buf[:runtime.Stack(buf, true)])
	}
}

func TestStartExplainsPortInUse(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	port := busy.Addr().(*net.TCPAddr).Port

	cfg := config.Config{Host: "127.0.0.1", SMTPPort: port, MaxMessages: 10, MaxMessageSize: 1 << 20}
	err = New(cfg, nil, "test", log.New(io.Discard, "", 0)).Start()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{fmt.Sprintf("SMTP port %d is already in use", port), "--smtp-port", "MAILPEEK_SMTP_PORT"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func TestActivityLog(t *testing.T) {
	creds, _ := config.ParseCredentials("app:secret")
	cfg := config.Config{Host: "127.0.0.1", MaxMessages: 10, MaxMessageSize: 1 << 10, SMTPAuth: creds}
	a := New(cfg, nil, "test", log.New(io.Discard, "", 0))
	var activity lockedBuffer
	a.SetActivityLog(log.New(&activity, "", 0))
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	defer a.Shutdown(context.Background())
	addr := a.SMTPAddr().String()
	auth := netsmtp.PlainAuth("", "app", "secret", "127.0.0.1")

	msg := "From: app@example.com\r\nTo: ana@example.com\r\nSubject: Welcome\r\n\r\nhi\r\n"
	if err := netsmtp.SendMail(addr, auth, "app@example.com", []string{"ana@example.com"}, []byte(msg)); err != nil {
		t.Fatal(err)
	}
	big := "Subject: Big\r\n\r\n" + strings.Repeat("x", 2<<10) + "\r\n"
	if err := netsmtp.SendMail(addr, auth, "app@example.com", []string{"bia@example.com"}, []byte(big)); err == nil {
		t.Fatal("expected the big message to be rejected")
	}
	if err := netsmtp.SendMail(addr, nil, "app@example.com", []string{"ana@example.com"}, []byte(msg)); err == nil {
		t.Fatal("expected a login to be required")
	}
	wrong := netsmtp.PlainAuth("", "app", "nope", "127.0.0.1")
	_ = netsmtp.SendMail(addr, wrong, "app@example.com", []string{"ana@example.com"}, []byte(msg))

	id := a.Store().List(store.Filter{})[0].ID
	for _, want := range []string{
		"received " + id + ` from app@example.com to ana@example.com "Welcome" (`,
		"rejected message from 127.0.0.1:",
		"to bia@example.com: over --max-message-size (1KB)",
		"SMTP login required",
		`SMTP login failed for user "app"`,
	} {
		if !strings.Contains(activity.String(), want) {
			t.Errorf("activity log misses %q:\n%s", want, activity.String())
		}
	}
}

func sendOverTLS(t *testing.T, addr string) {
	t.Helper()
	c, err := netsmtp.Dial(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		t.Fatal(err)
	}
	if err := c.Mail("app@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := c.Rcpt("ana@example.com"); err != nil {
		t.Fatal(err)
	}
	w, err := c.Data()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(w, "Subject: Secure\r\n\r\nhi\r\n")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	_ = c.Quit()
}

func TestSMTPTLSSelfSignedAndFromFiles(t *testing.T) {
	generated, err := smtp.SelfSignedTLS("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	cert := generated.Certificates[0]
	keyDER, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0o600)
	_ = os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600)

	for name, cfg := range map[string]config.Config{
		"self-signed": {SMTPTLS: true},
		"from files":  {SMTPTLS: true, SMTPTLSCert: certFile, SMTPTLSKey: keyFile},
	} {
		t.Run(name, func(t *testing.T) {
			cfg.Host, cfg.MaxMessages, cfg.MaxMessageSize = "127.0.0.1", 10, 1<<20
			a := New(cfg, nil, "test", log.New(io.Discard, "", 0))
			if err := a.Start(); err != nil {
				t.Fatal(err)
			}
			defer a.Shutdown(context.Background())
			sendOverTLS(t, a.SMTPAddr().String())
			if a.Store().Len() != 1 {
				t.Fatalf("stored %d messages", a.Store().Len())
			}
		})
	}

	bad := config.Config{Host: "127.0.0.1", MaxMessages: 10, MaxMessageSize: 1 << 20, SMTPTLS: true,
		SMTPTLSCert: filepath.Join(dir, "missing.pem"), SMTPTLSKey: keyFile}
	if err := New(bad, nil, "test", log.New(io.Discard, "", 0)).Start(); err == nil || !strings.Contains(err.Error(), "smtp-tls-cert") {
		t.Fatalf("missing certificate: %v", err)
	}
}

func TestSimulatedSMTPFailures(t *testing.T) {
	a, smtpAddr, base := startApp(t)
	defer a.Shutdown(context.Background())
	var activity lockedBuffer
	a.SetActivityLog(log.New(&activity, "", 0))

	addRule := func(body string) {
		t.Helper()
		resp, err := http.Post(base+"/api/v1/smtp/failures", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("add rule: %d", resp.StatusCode)
		}
	}
	deliver := func(to string) error {
		msg := "From: app@example.com\r\nTo: " + to + "\r\nSubject: Retry me\r\n\r\nhi\r\n"
		return netsmtp.SendMail(smtpAddr, nil, "app@example.com", []string{to}, []byte(msg))
	}
	code := func(err error) int {
		var tp *textproto.Error
		if errors.As(err, &tp) {
			return tp.Code
		}
		return 0
	}

	addRule(`{"stage":"data","code":451,"address":"ana@example.com"}`)
	if err := deliver("bia@example.com"); err != nil {
		t.Fatalf("another address must not be affected: %v", err)
	}
	if err := deliver("ana@example.com"); code(err) != 451 {
		t.Fatalf("first delivery to ana: %v", err)
	}
	if err := deliver("ana@example.com"); err != nil {
		t.Fatalf("retry after the rule was used up: %v", err)
	}

	addRule(`{"stage":"rcpt","code":550,"message":"5.1.1 No such user"}`)
	err := deliver("carl@example.com")
	if code(err) != 550 || !strings.Contains(err.Error(), "No such user") {
		t.Fatalf("rcpt failure: %v", err)
	}

	if n := a.Store().Len(); n != 2 {
		t.Fatalf("stored %d messages, want 2 (failed deliveries are not stored)", n)
	}
	for _, want := range []string{"simulated failure 451 at DATA for ana@example.com", "simulated failure 550 at RCPT for carl@example.com"} {
		if !strings.Contains(activity.String(), want) {
			t.Errorf("activity log misses %q:\n%s", want, activity.String())
		}
	}
}
