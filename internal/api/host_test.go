package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andbrslz/mailpeek/internal/events"
	"github.com/andbrslz/mailpeek/internal/store"
)

func TestLocalHost(t *testing.T) {
	for host, want := range map[string]bool{
		"":                    true,
		"localhost":           true,
		"localhost:8026":      true,
		"LOCALHOST.:8026":     true,
		"app.localhost:8026":  true,
		"127.0.0.1:8026":      true,
		"[::1]:8026":          true,
		"[::1]":               true,
		"192.168.1.20:8026":   true,
		"evil.example":        false,
		"evil.example:8026":   false,
		"localhost.evil.test": false,
		"mailpeek:8026":       false,
	} {
		if got := localHost(host); got != want {
			t.Errorf("localHost(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestLocalHostsOnly(t *testing.T) {
	b := events.NewBroker()
	defer b.Close()
	srv := New(store.New(10, b), b, nil, Info{}, nil)

	get := func(host, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = host
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec
	}

	if rec := get("rebound.example:8026", "/api/v1/messages"); rec.Code != http.StatusOK {
		t.Fatalf("without the check: status %d", rec.Code)
	}

	srv.WithLocalHostsOnly()
	if rec := get("localhost:8026", "/api/v1/messages"); rec.Code != http.StatusOK {
		t.Fatalf("localhost: status %d", rec.Code)
	}
	rec := get("rebound.example:8026", "/api/v1/messages")
	body, _ := io.ReadAll(rec.Body)
	if rec.Code != http.StatusForbidden || rec.Header().Get("Content-Type") != "application/json" ||
		!strings.Contains(string(body), "--host 0.0.0.0") {
		t.Fatalf("API from another host: status %d, %s", rec.Code, body)
	}
	if rec := get("rebound.example:8026", "/"); rec.Code != http.StatusForbidden ||
		!strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("UI from another host: status %d", rec.Code)
	}
}
