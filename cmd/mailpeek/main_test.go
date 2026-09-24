package main

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/andbrslz/mailpeek/internal/config"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestVersion(t *testing.T) {
	var out syncBuffer
	if err := run(context.Background(), []string{"--version"}, func(string) string { return "" }, &out, time.Now()); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "mailpeek ") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunPrintsBannerAndStopsOnSignal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var out syncBuffer
	done := make(chan error, 1)
	getenv := func(k string) string {
		return map[string]string{"MAILPEEK_HOST": "127.0.0.1"}[k]
	}
	go func() {
		done <- run(ctx, []string{"--smtp-port", "0", "--http-port", "0"}, getenv, &out, time.Now())
	}()

	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(out.String(), "Ready in") {
		if time.Now().After(deadline) {
			t.Fatalf("banner not printed: %q", out.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
	banner := out.String()
	for _, want := range []string{"Mailpeek\n", "SMTP  smtp://127.0.0.1:", "Web   http://127.0.0.1:"} {
		if !strings.Contains(banner, want) {
			t.Errorf("banner missing %q:\n%s", want, banner)
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("run did not return after cancel")
	}
}

func TestRunRejectsBadConfig(t *testing.T) {
	var out syncBuffer
	err := run(context.Background(), []string{"--max-messages", "0"}, func(string) string { return "" }, &out, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBannerWarnsWithoutWebUI(t *testing.T) {
	a, _ := startMailpeek(t, "")
	cfg := config.Config{Host: "127.0.0.1"}
	var withUI, withoutUI syncBuffer
	printBanner(&withUI, cfg, a, true, 0)
	printBanner(&withoutUI, cfg, a, false, 0)
	if strings.Contains(withUI.String(), "API only") {
		t.Errorf("banner with UI mentions API only:\n%s", withUI.String())
	}
	if !strings.Contains(withoutUI.String(), "API only") || !strings.Contains(withoutUI.String(), "go install") {
		t.Errorf("banner without UI does not explain it:\n%s", withoutUI.String())
	}
}

func TestBannerShowsDataDir(t *testing.T) {
	a, _ := startMailpeek(t, "")
	var memory, disk syncBuffer
	printBanner(&memory, config.Config{Host: "127.0.0.1"}, a, true, 0)
	printBanner(&disk, config.Config{Host: "127.0.0.1", DataDir: "/data"}, a, true, 0)
	if strings.Contains(memory.String(), "Data  ") {
		t.Errorf("banner without a data dir mentions one:\n%s", memory.String())
	}
	if !strings.Contains(disk.String(), "Data  /data  (0 messages kept across restarts)\n") {
		t.Errorf("banner does not show the data dir:\n%s", disk.String())
	}
}
