package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	netsmtp "net/smtp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mailpeek/mailpeek/internal/config"
	"github.com/mailpeek/mailpeek/internal/mail"
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
