package api

import (
	"encoding/json"
	"mime"
	"net/http"

	"github.com/mailpeek/mailpeek/internal/failures"
)

func (s *Server) WithFailures(f *failures.Set) *Server {
	s.failures = f
	return s
}

func (s *Server) listFailures(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string][]failures.Rule{"failures": s.failures.List()})
}

func (s *Server) addFailure(w http.ResponseWriter, r *http.Request) {
	if mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	var req failures.Request
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	rule, err := s.failures.Add(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) clearFailures(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]int{"deleted": s.failures.Clear(r.URL.Query().Get("address"))})
}

func (s *Server) deleteFailure(w http.ResponseWriter, r *http.Request) {
	if !s.failures.Remove(r.PathValue("id")) {
		writeError(w, http.StatusNotFound, "failure rule not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
