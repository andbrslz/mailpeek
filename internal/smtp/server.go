package smtp

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Envelope struct {
	From       string
	To         []string
	RemoteAddr string
}

type Handler func(env Envelope, data []byte) (id string, err error)

var ErrServerClosed = errors.New("smtp: server closed")

type Server struct {
	Hostname       string
	Handler        Handler
	MaxMessageSize int64
	MaxRecipients  int
	MaxConnections int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	DataTimeout    time.Duration
	Auth           func(user, password string) bool
	ErrorLog       *log.Logger

	mu       sync.Mutex
	listener net.Listener
	sessions map[*session]struct{}
	wg       sync.WaitGroup
	closing  atomic.Bool
}

func (s *Server) Serve(l net.Listener) error {
	s.mu.Lock()
	if s.closing.Load() {
		s.mu.Unlock()
		return ErrServerClosed
	}
	s.listener = l
	s.mu.Unlock()

	backoff := 5 * time.Millisecond
	for {
		conn, err := l.Accept()
		if err != nil {
			if s.closing.Load() || errors.Is(err, net.ErrClosed) {
				return ErrServerClosed
			}
			s.logf("smtp: accept: %v; retrying in %v", err, backoff)
			time.Sleep(backoff)
			backoff = min(backoff*2, time.Second)
			continue
		}
		backoff = 5 * time.Millisecond
		sess := newSession(s, conn)
		if !s.track(sess) {
			_ = conn.Close()
			continue
		}
		if s.active() > s.maxConnections() {
			go func() {
				defer s.wg.Done()
				defer s.untrack(sess)
				sess.reply(421, "4.7.0 Too many connections, try again later")
				_ = conn.Close()
			}()
			continue
		}
		go func() {
			defer s.wg.Done()
			defer s.untrack(sess)
			sess.serve()
		}()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.closing.Store(true)
	s.mu.Lock()
	if s.listener != nil {
		_ = s.listener.Close()
	}
	for sess := range s.sessions {
		if sess.idle.Load() {
			sess.interrupt()
		}
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.mu.Lock()
		for sess := range s.sessions {
			_ = sess.conn.Close()
		}
		s.mu.Unlock()
		<-done
		return ctx.Err()
	}
}

func (s *Server) track(sess *session) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing.Load() {
		return false
	}
	if s.sessions == nil {
		s.sessions = make(map[*session]struct{})
	}
	s.sessions[sess] = struct{}{}
	s.wg.Add(1)
	return true
}

func (s *Server) active() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

func (s *Server) untrack(sess *session) {
	s.mu.Lock()
	delete(s.sessions, sess)
	s.mu.Unlock()
}

func (s *Server) logf(format string, args ...any) {
	if s.ErrorLog != nil {
		s.ErrorLog.Printf(format, args...)
	}
}

func (s *Server) hostname() string            { return orDefault(s.Hostname, "mailpeek") }
func (s *Server) maxSize() int64              { return orDefault(s.MaxMessageSize, 10<<20) }
func (s *Server) maxRecipients() int          { return orDefault(s.MaxRecipients, 100) }
func (s *Server) maxConnections() int         { return orDefault(s.MaxConnections, 100) }
func (s *Server) readTimeout() time.Duration  { return orDefault(s.ReadTimeout, 60*time.Second) }
func (s *Server) writeTimeout() time.Duration { return orDefault(s.WriteTimeout, 30*time.Second) }
func (s *Server) dataTimeout() time.Duration  { return orDefault(s.DataTimeout, 5*time.Minute) }

func orDefault[T comparable](v, def T) T {
	var zero T
	if v == zero {
		return def
	}
	return v
}

func newSession(s *Server, conn net.Conn) *session {
	return &session{
		srv:  s,
		conn: conn,
		r:    bufio.NewReaderSize(conn, 4096),
		w:    bufio.NewWriter(conn),
	}
}
