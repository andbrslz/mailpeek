package main

import (
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/andbrslz/mailpeek/internal/app"
	"github.com/andbrslz/mailpeek/internal/config"
	"github.com/andbrslz/mailpeek/internal/store"
)

func startMailpeek(t *testing.T, smtpAuth string) (*app.App, func(string) string) {
	t.Helper()
	cfg := config.Config{Host: "127.0.0.1", MaxMessages: 10, MaxMessageSize: 1 << 20, MaxStoreSize: 1 << 20}
	if smtpAuth != "" {
		creds, err := config.ParseCredentials(smtpAuth)
		if err != nil {
			t.Fatal(err)
		}
		cfg.SMTPAuth = creds
	}
	a := app.New(cfg, nil, "test", log.New(io.Discard, "", 0))
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Shutdown(t.Context()) })
	port := strconv.Itoa(a.SMTPAddr().(*net.TCPAddr).Port)
	return a, func(k string) string { return map[string]string{"MAILPEEK_SMTP_PORT": port}[k] }
}

func TestSendPlainText(t *testing.T) {
	a, getenv := startMailpeek(t, "")
	var out syncBuffer
	err := send([]string{"--to", "ana@example.com,bia@example.com", "--subject", "Olá, Ana", "--text", "Linha 1\n.começa com ponto"}, getenv, &out)
	if err != nil {
		t.Fatal(err)
	}
	msgs := a.Store().List(storeAll)
	if len(msgs) != 1 {
		t.Fatalf("stored %d messages", len(msgs))
	}
	m := msgs[0]
	if m.Subject != "Olá, Ana" || len(m.Envelope.To) != 2 || !strings.Contains(m.Text, ".começa com ponto") {
		t.Fatalf("message = %+v", m)
	}
	if want := "(id " + m.ID + ")"; !strings.Contains(out.String(), want) {
		t.Fatalf("output %q does not contain %q", out.String(), want)
	}
}

func TestSendHTML(t *testing.T) {
	a, getenv := startMailpeek(t, "")
	err := send([]string{"--to", "ana@example.com", "--html", `<a href="http://localhost/x">Go</a>`}, getenv, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	m := a.Store().List(storeAll)[0]
	if m.Text == "" || len(m.Links) != 1 || m.Links[0].Href != "http://localhost/x" {
		t.Fatalf("message = %+v", m)
	}
}

func TestSendWithLogin(t *testing.T) {
	a, getenv := startMailpeek(t, "app:secret")
	err := send([]string{"--to", "ana@example.com"}, getenv, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "--auth") {
		t.Fatalf("without login: %v", err)
	}
	if err := send([]string{"--to", "ana@example.com", "--auth", "app:secret"}, getenv, io.Discard); err != nil {
		t.Fatal(err)
	}
	if n := a.Store().Len(); n != 1 {
		t.Fatalf("stored %d messages", n)
	}
}

func TestSendErrors(t *testing.T) {
	if err := send(nil, func(string) string { return "" }, io.Discard); err == nil || !strings.Contains(err.Error(), "--to is required") {
		t.Fatalf("missing --to: %v", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	ln.Close()
	err = send([]string{"--to", "a@b", "--port", port}, func(string) string { return "" }, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "is Mailpeek running") {
		t.Fatalf("closed port: %v", err)
	}
}

var storeAll = store.Filter{}
