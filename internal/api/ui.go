package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const uiPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' *; " +
	"img-src 'self' * data: blob:; font-src 'self' * data:; media-src 'self' * data:; " +
	"connect-src 'self'; frame-src 'self'; object-src 'none'; base-uri 'self'; " +
	"form-action 'none'; frame-ancestors 'none'"

func (s *Server) serveUI(w http.ResponseWriter, r *http.Request) {
	if isAPIPath(r.URL.Path) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if s.ui == nil {
		http.Error(w, "Web UI not built. Run `make web` and rebuild, or use the API at /api/v1.", http.StatusNotFound)
		return
	}
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if info, err := fs.Stat(s.ui, name); name == "" || err != nil || info.IsDir() {
		name = "index.html"
	}

	h := w.Header()
	if strings.HasPrefix(name, "assets/") {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "no-cache")
	}
	if name == "index.html" {
		h.Set("Content-Security-Policy", uiPolicy)
		h.Set("X-Frame-Options", "DENY")
	}
	http.ServeFileFS(w, r, s.ui, name)
}
