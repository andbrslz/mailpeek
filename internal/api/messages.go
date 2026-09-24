package api

import (
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/andbrslz/mailpeek/internal/events"
	"github.com/andbrslz/mailpeek/internal/mail"
)

type listResponse struct {
	Messages   []mail.Summary `json:"messages"`
	Count      int            `json:"count"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, err := parseFilter(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cursor, limit, err := parsePage(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page := s.store.Page(f, cursor, limit)
	resp := listResponse{Messages: make([]mail.Summary, len(page.Messages)), Count: len(page.Messages)}
	for i, m := range page.Messages {
		resp.Messages[i] = m.Summary()
	}
	if page.Next > 0 {
		resp.NextCursor = strconv.FormatUint(page.Next, 10)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) countMessages(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": s.store.Count(f)})
}

func (s *Server) latestMessage(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	m, ok := s.store.Latest(f)
	if !ok {
		writeError(w, http.StatusNotFound, "no matching message")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) waitMessage(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	timeout, err := parseTimeout(r.URL.Query().Get("timeout"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sub := s.broker.Subscribe()
	defer sub.Close()
	if m, ok := s.store.Latest(f); ok {
		writeJSON(w, http.StatusOK, m)
		return
	}

	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(timeout + 10*time.Second))
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-r.Context().Done():
			writeError(w, http.StatusServiceUnavailable, "request cancelled")
			return
		case <-timer.C:
			if m, ok := s.store.Latest(f); ok {
				writeJSON(w, http.StatusOK, m)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusNoContent)
			return
		case e, ok := <-sub.C:
			if !ok {
				writeError(w, http.StatusServiceUnavailable, "server shutting down")
				return
			}
			created := e.Type == events.MessageCreated
			for drained := false; !drained; {
				select {
				case e, ok := <-sub.C:
					if !ok {
						writeError(w, http.StatusServiceUnavailable, "server shutting down")
						return
					}
					created = created || e.Type == events.MessageCreated
				default:
					drained = true
				}
			}
			if !created && sub.Missed() == 0 {
				continue
			}
			if m, ok := s.store.Latest(f); ok {
				writeJSON(w, http.StatusOK, m)
				return
			}
		}
	}
}

func (s *Server) getMessage(w http.ResponseWriter, r *http.Request) {
	m, ok := s.store.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) deleteMessage(w http.ResponseWriter, r *http.Request) {
	if !s.store.Delete(r.PathValue("id")) {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteMessages(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	n := s.store.DeleteMatching(f)
	writeJSON(w, http.StatusOK, map[string]int{"deleted": n})
}

func (s *Server) rawMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	raw, ok := s.store.Raw(id)
	if !ok {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}
	h := w.Header()
	if r.URL.Query().Get("download") != "" {
		h.Set("Content-Type", "message/rfc822")
		h.Set("Content-Disposition", contentDisposition(id+".eml"))
	} else {
		h.Set("Content-Type", "text/plain; charset=utf-8")
	}
	h.Set("Content-Length", strconv.Itoa(len(raw)))
	h.Set("Content-Security-Policy", "sandbox")
	_, _ = w.Write(raw)
}

func (s *Server) attachment(w http.ResponseWriter, r *http.Request) {
	m, ok := s.store.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}
	a, ok := m.Attachment(r.PathValue("attachmentId"))
	if !ok {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	h := w.Header()
	contentType := safeContentType(a.ContentType)
	h.Set("Content-Type", contentType)
	if r.URL.Query().Get("inline") != "" && previewable[contentType] {
		h.Set("Content-Disposition", disposition("inline", a.Filename))
	} else {
		h.Set("Content-Disposition", contentDisposition(a.Filename))
	}
	h.Set("Content-Length", strconv.Itoa(len(a.Data)))
	h.Set("Content-Security-Policy", "sandbox")
	h.Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(a.Data)
}

func safeContentType(ct string) string {
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || !strings.Contains(mediaType, "/") {
		return "application/octet-stream"
	}
	return mediaType
}

var previewable = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
	"image/avif": true,
	"image/bmp":  true,
}

func contentDisposition(filename string) string { return disposition("attachment", filename) }

func disposition(kind, filename string) string {
	if v := mime.FormatMediaType(kind, map[string]string{"filename": SafeFilename(filename)}); v != "" {
		return v
	}
	return kind
}

func SafeFilename(name string) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	if name == "/" || name == "." {
		name = ""
	}
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, name)
	name = strings.TrimLeft(strings.TrimSpace(name), ".")
	for utf8.RuneCountInString(name) > 200 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	if name == "" {
		return "attachment"
	}
	return name
}
