package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	sessionCookie = "mailpeek_session"
	sessionTTL    = 7 * 24 * time.Hour
	failedDelay   = 400 * time.Millisecond
)

type sessions struct {
	mu     sync.Mutex
	expiry map[string]time.Time
}

func newSessions() *sessions { return &sessions{expiry: make(map[string]time.Time)} }

func (s *sessions) create() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for t, exp := range s.expiry {
		if now.After(exp) {
			delete(s.expiry, t)
		}
	}
	s.expiry[token] = now.Add(sessionTTL)
	return token
}

func (s *sessions) valid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.expiry[token]
	return ok && time.Now().Before(exp)
}

func (s *sessions) delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.expiry, token)
}

func (s *Server) guard(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case "/api/v1/health", "/favicon.svg":
		return true
	case "/login":
		s.login(w, r)
		return false
	case "/logout":
		s.logout(w, r)
		return false
	}
	if s.authenticated(r) {
		return true
	}
	if isAPIPath(r.URL.Path) {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return false
	}
	http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
	return false
}

func (s *Server) authenticated(r *http.Request) bool {
	if c, err := r.Cookie(sessionCookie); err == nil && s.sessions.valid(c.Value) {
		return true
	}
	user, password, ok := r.BasicAuth()
	return ok && s.auth(user, password)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	next := safeNext(r.FormValue("next"))
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if s.authenticated(r) {
			http.Redirect(w, r, next, http.StatusSeeOther)
			return
		}
		renderLogin(w, r, http.StatusOK, loginView{Next: next})
	case http.MethodPost:
		if !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "cross-origin login refused")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		user, password := r.PostFormValue("username"), r.PostFormValue("password")
		if !s.auth(user, password) {
			time.Sleep(failedDelay)
			renderLogin(w, r, http.StatusUnauthorized, loginView{Next: next, Username: user, Failed: true})
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    s.sessions.create(),
			Path:     "/",
			MaxAge:   int(sessionTTL.Seconds()),
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, next, http.StatusSeeOther)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "cross-origin logout refused")
		return
	}
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.sessions.delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") ||
		strings.HasPrefix(next, "/login") {
		return "/"
	}
	return next
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == r.Host
}
