package smtp

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/andbrslz/mailpeek/internal/config"
)

const maxErrors = 10

var (
	errLineTooLong   = errors.New("line too long")
	errTooLarge      = errors.New("message too large")
	errAuthCancelled = errors.New("authentication cancelled")
	errBadBase64     = errors.New("invalid base64")
)

type session struct {
	srv  *Server
	conn net.Conn
	r    *bufio.Reader
	w    *bufio.Writer
	idle atomic.Bool

	helo     string
	authed   bool
	tls      bool
	hasFrom  bool
	from     string
	rcpts    []string
	errCount int
}

func (c *session) serve() {
	defer c.conn.Close()
	c.reply(220, c.srv.hostname()+" ESMTP Mailpeek")
	for {
		line, err := c.readCommand()
		if errors.Is(err, errLineTooLong) {
			if !c.fail(500, "5.5.2 Line too long") {
				return
			}
			continue
		}
		if err != nil {
			if c.srv.closing.Load() {
				c.reply(421, "4.3.2 Service shutting down")
			}
			return
		}
		verb, arg, _ := strings.Cut(line, " ")
		if !c.handle(strings.ToUpper(verb), strings.TrimSpace(arg)) {
			return
		}
		if c.srv.closing.Load() {
			c.reply(421, "4.3.2 Service shutting down")
			return
		}
	}
}

func (c *session) handle(verb, arg string) bool {
	switch verb {
	case "HELO", "EHLO":
		return c.hello(verb, arg)
	case "MAIL":
		return c.mail(arg)
	case "RCPT":
		return c.rcpt(arg)
	case "DATA":
		return c.data()
	case "RSET":
		c.reset()
		return c.reply(250, "2.0.0 OK")
	case "NOOP":
		return c.reply(250, "2.0.0 OK")
	case "QUIT":
		c.reply(221, "2.0.0 Bye")
		return false
	case "VRFY":
		return c.reply(252, "2.5.0 Cannot verify user, but will accept message")
	case "HELP":
		return c.reply(214, "2.0.0 Mailpeek accepts all mail for inspection")
	case "AUTH":
		return c.auth(arg)
	case "STARTTLS":
		return c.startTLS(arg)
	case "EXPN", "TURN", "ETRN", "BDAT":
		return c.fail(502, "5.5.1 Command not implemented")
	}
	return c.fail(500, "5.5.2 Command not recognized")
}

func (c *session) hello(verb, arg string) bool {
	if arg == "" {
		return c.fail(501, "5.5.4 Syntax: "+verb+" hostname")
	}
	c.reset()
	c.helo = arg
	if verb == "HELO" {
		return c.reply(250, c.srv.hostname())
	}
	lines := []string{
		c.srv.hostname() + " greets " + arg,
		"SIZE " + strconv.FormatInt(c.srv.maxSize(), 10),
		"8BITMIME",
		"SMTPUTF8",
		"PIPELINING",
	}
	if c.srv.TLSConfig != nil && !c.tls {
		lines = append(lines, "STARTTLS")
	}
	return c.reply(250, append(lines, "AUTH PLAIN LOGIN", "HELP")...)
}

func (c *session) startTLS(arg string) bool {
	switch {
	case c.srv.TLSConfig == nil:
		return c.fail(502, "5.5.1 Command not implemented")
	case c.tls:
		return c.fail(503, "5.5.1 TLS already active")
	case arg != "":
		return c.fail(501, "5.5.4 Syntax: STARTTLS")
	}
	if !c.reply(220, "2.0.0 Ready to start TLS") {
		return false
	}
	conn := tls.Server(c.conn, c.srv.TLSConfig)
	_ = c.conn.SetDeadline(time.Now().Add(c.srv.readTimeout()))
	if err := conn.Handshake(); err != nil {
		c.srv.note("TLS handshake with %s failed (a client that verifies certificates needs --smtp-tls-cert, "+
			"or certificate verification turned off): %v", c.remote(), err)
		return false
	}
	c.r = bufio.NewReaderSize(conn, 4096)
	c.w = bufio.NewWriter(conn)
	c.tls = true
	c.helo, c.authed = "", false
	c.reset()
	return true
}

func (c *session) mail(arg string) bool {
	if c.helo == "" {
		return c.fail(503, "5.5.1 Send HELO/EHLO first")
	}
	if c.srv.Auth != nil && !c.authed {
		c.srv.note("rejected mail from %s: SMTP login required (Mailpeek runs with --smtp-auth)", c.remote())
		return c.fail(530, "5.7.0 Authentication required")
	}
	if c.hasFrom {
		return c.fail(503, "5.5.1 Sender already specified")
	}
	addr, params, ok := parsePath(arg, "FROM:")
	if !ok {
		return c.fail(501, "5.5.4 Syntax: MAIL FROM:<address>")
	}
	for _, p := range params {
		k, v, _ := strings.Cut(p, "=")
		if strings.EqualFold(k, "SIZE") {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > c.srv.maxSize() {
				c.srv.note("rejected mail from %s: announced size %d bytes is over --max-message-size (%s)",
					c.remote(), n, config.FormatSize(c.srv.maxSize()))
				return c.fail(552, "5.3.4 Message size exceeds fixed limit")
			}
		}
	}
	c.hasFrom, c.from = true, addr
	return c.reply(250, "2.1.0 OK")
}

func (c *session) rcpt(arg string) bool {
	if !c.hasFrom {
		return c.fail(503, "5.5.1 Send MAIL first")
	}
	addr, _, ok := parsePath(arg, "TO:")
	if !ok || addr == "" {
		return c.fail(501, "5.5.4 Syntax: RCPT TO:<address>")
	}
	if len(c.rcpts) >= c.srv.maxRecipients() {
		return c.reply(452, "4.5.3 Too many recipients")
	}
	if code, msg, ok := c.rejected("rcpt", []string{addr}); ok {
		return c.reply(code, msg)
	}
	c.rcpts = append(c.rcpts, addr)
	return c.reply(250, "2.1.5 OK")
}

func (c *session) data() bool {
	if len(c.rcpts) == 0 {
		return c.fail(503, "5.5.1 Send RCPT first")
	}
	if !c.reply(354, "End data with <CR><LF>.<CR><LF>") {
		return false
	}
	_ = c.conn.SetReadDeadline(time.Now().Add(c.srv.dataTimeout()))
	body, err := readData(c.r, c.srv.maxSize())
	if errors.Is(err, errTooLarge) {
		c.srv.note("rejected message from %s to %s: over --max-message-size (%s)",
			c.remote(), strings.Join(c.rcpts, ", "), config.FormatSize(c.srv.maxSize()))
		c.reset()
		return c.reply(552, "5.3.4 Message size exceeds fixed limit")
	}
	if err != nil {
		return false
	}

	env := Envelope{From: c.from, To: c.rcpts, RemoteAddr: c.conn.RemoteAddr().String(), TLS: c.tls}
	c.reset()
	if code, msg, ok := c.rejected("data", env.To); ok {
		return c.reply(code, msg)
	}
	id, err := c.srv.Handler(env, body)
	if err != nil {
		c.srv.logf("smtp: handler: %v", err)
		return c.reply(451, "4.3.0 Failed to store message")
	}
	return c.reply(250, "2.0.0 OK: queued as "+id)
}

func (c *session) auth(arg string) bool {
	if c.helo == "" {
		return c.fail(503, "5.5.1 Send HELO/EHLO first")
	}
	if c.authed {
		return c.fail(503, "5.5.1 Already authenticated")
	}
	mech, initial, _ := strings.Cut(arg, " ")
	var user, password string
	var err error
	switch strings.ToUpper(mech) {
	case "PLAIN":
		var b []byte
		if b, err = c.challenge(initial, ""); err == nil {
			if parts := strings.SplitN(string(b), "\x00", 3); len(parts) == 3 {
				user, password = parts[1], parts[2]
			}
		}
	case "LOGIN":
		var u, p []byte
		if u, err = c.challenge(initial, "VXNlcm5hbWU6"); err == nil {
			if p, err = c.challenge("", "UGFzc3dvcmQ6"); err == nil {
				user, password = string(u), string(p)
			}
		}
	default:
		return c.fail(504, "5.5.4 Unrecognized authentication type")
	}
	switch {
	case errors.Is(err, errAuthCancelled):
		return c.fail(501, "5.0.0 Authentication cancelled")
	case errors.Is(err, errBadBase64):
		return c.fail(501, "5.5.2 Cannot decode response")
	case err != nil:
		return false
	}
	if c.srv.Auth != nil && !c.srv.Auth(user, password) {
		c.srv.note("SMTP login failed for user %q from %s: wrong username or password", user, c.remote())
		return c.fail(535, "5.7.8 Authentication credentials invalid")
	}
	c.authed = true
	return c.reply(235, "2.7.0 Authentication successful")
}

func (c *session) challenge(initial, prompt string) ([]byte, error) {
	line := initial
	if line == "" {
		if !c.reply(334, prompt) {
			return nil, net.ErrClosed
		}
		var err error
		if line, err = c.readCommand(); err != nil {
			return nil, err
		}
	}
	switch line {
	case "*":
		return nil, errAuthCancelled
	case "=":
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(line)
	if err != nil {
		return nil, errBadBase64
	}
	return b, nil
}

func (c *session) rejected(stage string, to []string) (int, string, bool) {
	if c.srv.Reject == nil {
		return 0, "", false
	}
	return c.srv.Reject(stage, to)
}

func (c *session) remote() string { return c.conn.RemoteAddr().String() }

func (c *session) reset() {
	c.hasFrom, c.from, c.rcpts = false, "", nil
}

func (c *session) fail(code int, msg string) bool {
	c.errCount++
	if c.errCount > maxErrors {
		c.reply(421, "4.7.0 Too many errors")
		return false
	}
	return c.reply(code, msg)
}

func (c *session) reply(code int, lines ...string) bool {
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.srv.writeTimeout()))
	for i, l := range lines {
		sep := "-"
		if i == len(lines)-1 {
			sep = " "
		}
		fmt.Fprintf(c.w, "%d%s%s\r\n", code, sep, l)
	}
	if err := c.w.Flush(); err != nil {
		_ = c.conn.Close()
		return false
	}
	return true
}

func (c *session) readCommand() (string, error) {
	_ = c.conn.SetReadDeadline(time.Now().Add(c.srv.readTimeout()))
	c.idle.Store(true)
	defer c.idle.Store(false)
	if c.srv.closing.Load() {
		return "", ErrServerClosed
	}
	line, err := c.r.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		for errors.Is(err, bufio.ErrBufferFull) {
			_, err = c.r.ReadSlice('\n')
		}
		if err != nil {
			return "", err
		}
		return "", errLineTooLong
	}
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(line), "\r\n"), nil
}

func (c *session) interrupt() { _ = c.conn.SetReadDeadline(time.Now()) }

func readData(r *bufio.Reader, max int64) ([]byte, error) {
	var buf bytes.Buffer
	tooLarge := false
	atLineStart := true
	for {
		chunk, err := r.ReadSlice('\n')
		if err != nil && !errors.Is(err, bufio.ErrBufferFull) {
			return nil, err
		}
		complete := err == nil
		if atLineStart {
			if complete && (bytes.Equal(chunk, []byte(".\r\n")) || bytes.Equal(chunk, []byte(".\n"))) {
				break
			}
			if len(chunk) > 0 && chunk[0] == '.' {
				chunk = chunk[1:]
			}
		}
		if !tooLarge {
			if int64(buf.Len()+len(chunk)) > max {
				tooLarge = true
				buf = bytes.Buffer{}
			} else {
				buf.Write(chunk)
			}
		}
		atLineStart = complete
	}
	if tooLarge {
		return nil, errTooLarge
	}
	return buf.Bytes(), nil
}

func parsePath(arg, prefix string) (addr string, params []string, ok bool) {
	if len(arg) < len(prefix) || !strings.EqualFold(arg[:len(prefix)], prefix) {
		return "", nil, false
	}
	rest := strings.TrimSpace(arg[len(prefix):])
	if strings.HasPrefix(rest, "<") {
		end := strings.IndexByte(rest, '>')
		if end < 0 {
			return "", nil, false
		}
		addr, rest = rest[1:end], rest[end+1:]
	} else {
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return "", nil, false
		}
		addr, rest = fields[0], strings.Join(fields[1:], " ")
	}
	if i := strings.LastIndexByte(addr, ':'); i >= 0 && strings.HasPrefix(addr, "@") {
		addr = addr[i+1:]
	}
	if strings.ContainsAny(addr, " \t\r\n") {
		return "", nil, false
	}
	return addr, strings.Fields(rest), true
}
