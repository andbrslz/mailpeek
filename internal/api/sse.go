package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	sseHeartbeat    = 25 * time.Second
	sseWriteTimeout = 10 * time.Second
)

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")

	sub := s.broker.Subscribe()
	defer sub.Close()

	send := func(format string, args ...any) bool {
		_ = rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout))
		if _, err := fmt.Fprintf(w, format, args...); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	if !send("retry: 2000\n: connected\n\n") {
		return
	}

	heartbeat := time.NewTicker(sseHeartbeat)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if !send(": ping\n\n") {
				return
			}
		case e, ok := <-sub.C:
			if !ok {
				return
			}
			data := map[string]string{}
			if e.ID != "" {
				data["id"] = e.ID
			}
			payload, _ := json.Marshal(data)
			if !send("event: %s\ndata: %s\n\n", e.Type, payload) {
				return
			}
			if sub.Missed() > 0 && !send("event: resync\ndata: {}\n\n") {
				return
			}
		}
	}
}
