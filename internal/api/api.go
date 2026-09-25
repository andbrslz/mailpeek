package api

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andbrslz/mailpeek/internal/events"
	"github.com/andbrslz/mailpeek/internal/failures"
	"github.com/andbrslz/mailpeek/internal/store"
)

const (
	DefaultWaitTimeout = 10 * time.Second
	MaxWaitTimeout     = 30 * time.Second
)

type Info struct {
	Version        string `json:"version"`
	SMTPPort       int    `json:"smtpPort"`
	HTTPPort       int    `json:"httpPort"`
	MaxMessages    int    `json:"maxMessages"`
	MaxMessageSize int64  `json:"maxMessageSize"`
	MaxStoreSize   int64  `json:"maxStoreSize"`
	SMTPAuth       bool   `json:"smtpAuth"`
	SMTPTLS        bool   `json:"smtpTls"`
	UIAuth         bool   `json:"uiAuth"`
}

type Server struct {
	store    *store.MemoryStore
	broker   *events.Broker
	ui       fs.FS
	info     Info
	auth     func(user, password string) bool
	sessions *sessions
	mux      *http.ServeMux
	patterns []string
	failures *failures.Set

	localHostsOnly bool
}

//go:embed openapi.json
var openapi []byte

func New(st *store.MemoryStore, broker *events.Broker, ui fs.FS, info Info, auth func(user, password string) bool) *Server {
	s := &Server{store: st, broker: broker, ui: ui, info: info, auth: auth, sessions: newSessions(), mux: http.NewServeMux(), failures: &failures.Set{}}
	s.info.UIAuth = auth != nil
	s.routes()
	return s
}

func (s *Server) routes() {
	handle := func(pattern string, h http.HandlerFunc) {
		s.patterns = append(s.patterns, pattern)
		s.mux.HandleFunc(pattern, h)
	}
	handle("GET /api/v1/health", s.health)
	handle("GET /api/v1/info", s.getInfo)
	handle("GET /api/v1/openapi.json", s.openAPI)
	handle("GET /api/v1/events", s.events)
	handle("GET /api/v1/messages", s.listMessages)
	handle("DELETE /api/v1/messages", s.deleteMessages)
	handle("GET /api/v1/messages/count", s.countMessages)
	handle("GET /api/v1/messages/latest", s.latestMessage)
	handle("GET /api/v1/messages/wait", s.waitMessage)
	handle("GET /api/v1/messages/{id}", s.getMessage)
	handle("DELETE /api/v1/messages/{id}", s.deleteMessage)
	handle("GET /api/v1/messages/{id}/raw", s.rawMessage)
	handle("GET /api/v1/messages/{id}/attachments/{attachmentId}", s.attachment)
	handle("GET /api/v1/smtp/failures", s.listFailures)
	handle("POST /api/v1/smtp/failures", s.addFailure)
	handle("DELETE /api/v1/smtp/failures", s.clearFailures)
	handle("DELETE /api/v1/smtp/failures/{id}", s.deleteFailure)
	handle("GET /", s.serveUI)
}

func (s *Server) openAPI(w http.ResponseWriter, _ *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Cache-Control", "no-cache")
	_, _ = w.Write(openapi)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "same-origin")
	if !s.hostAllowed(w, r) {
		return
	}
	if s.auth != nil && !s.guard(w, r) {
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, struct {
		Info
		Store store.Stats `json:"store"`
	}{s.info, s.store.Stats()})
}

func parseFilter(q url.Values) (store.Filter, error) {
	f := store.Filter{
		To:      q.Get("to"),
		Address: q.Get("address"),
		From:    q.Get("from"),
		Subject: q.Get("subject"),
		Body:    q.Get("body"),
		Query:   q.Get("q"),
	}
	if v := q.Get("since"); v != "" {
		if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.Since = time.UnixMilli(ms)
		} else if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			f.Since = t
		} else {
			return f, errors.New("since must be RFC 3339 or Unix milliseconds")
		}
	}
	return f, nil
}

func parsePage(q url.Values) (cursor uint64, limit int, err error) {
	if v := q.Get("cursor"); v != "" {
		if cursor, err = strconv.ParseUint(v, 10, 64); err != nil || cursor == 0 {
			return 0, 0, errors.New("cursor must be a nextCursor returned by an earlier page")
		}
	}
	if v := q.Get("limit"); v != "" {
		if limit, err = strconv.Atoi(v); err != nil || limit < 1 {
			return 0, 0, errors.New("limit must be a positive number")
		}
	}
	return cursor, limit, nil
}

func parseTimeout(v string) (time.Duration, error) {
	if v == "" {
		return DefaultWaitTimeout, nil
	}
	ms, err := strconv.ParseInt(v, 10, 64)
	if err != nil || ms < 0 {
		return 0, errors.New("timeout must be a non-negative number of milliseconds")
	}
	return min(time.Duration(ms)*time.Millisecond, MaxWaitTimeout), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func isAPIPath(p string) bool { return strings.HasPrefix(p, "/api/") }
