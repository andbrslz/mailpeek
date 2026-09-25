package api

import (
	"net"
	"net/http"
	"strings"
)

func (s *Server) WithLocalHostsOnly() *Server {
	s.localHostsOnly = true
	return s
}

func localHost(host string) bool {
	if host == "" {
		return true
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.ToLower(strings.Trim(host, "[]")), ".")
	return host == "localhost" || strings.HasSuffix(host, ".localhost") || net.ParseIP(host) != nil
}

func (s *Server) hostAllowed(w http.ResponseWriter, r *http.Request) bool {
	if !s.localHostsOnly || localHost(r.Host) {
		return true
	}
	const msg = "host not allowed: Mailpeek listens on this machine only; open it as localhost or 127.0.0.1, " +
		"or start it with --host 0.0.0.0 to use another name"
	if isAPIPath(r.URL.Path) {
		writeError(w, http.StatusForbidden, msg)
	} else {
		http.Error(w, msg, http.StatusForbidden)
	}
	return false
}
