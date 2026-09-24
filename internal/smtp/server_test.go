package smtp

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	netsmtp "net/smtp"
	"strings"
	"sync"
	"testing"
	"time"
)

type received struct {
	env  Envelope
	data []byte
}

type recorder struct {
	mu   sync.Mutex
	msgs []received
}

func (r *recorder) handle(env Envelope, data []byte) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, received{env, append([]byte(nil), data...)})
	return fmt.Sprintf("id%d", len(r.msgs)), nil
}

func (r *recorder) all() []received {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]received(nil), r.msgs...)
}

func start(t testing.TB, srv *Server) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(l) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		if err := <-done; err != ErrServerClosed {
			t.Errorf("Serve returned %v", err)
		}
	})
	return l.Addr().String()
}

type client struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
}

func dial(t *testing.T, addr string) *client {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	c := &client{t: t, conn: conn, r: bufio.NewReader(conn)}
	c.expect("220")
	return c
}

func (c *client) read() string {
	c.t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var lines []string
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			c.t.Fatalf("read: %v (got %q)", err, lines)
		}
		lines = append(lines, strings.TrimRight(line, "\r\n"))
		if len(line) < 4 || line[3] != '-' {
			return strings.Join(lines, "\n")
		}
	}
}

func (c *client) expect(code string) string {
	c.t.Helper()
	reply := c.read()
	if !strings.HasPrefix(reply, code) {
		c.t.Fatalf("expected %s, got %q", code, reply)
	}
	return reply
}

func (c *client) send(line, code string) string {
	c.t.Helper()
	if _, err := fmt.Fprintf(c.conn, "%s\r\n", line); err != nil {
		c.t.Fatal(err)
	}
	return c.expect(code)
}

func TestSendMailMultipleRecipients(t *testing.T) {
	rec := &recorder{}
	addr := start(t, &Server{Handler: rec.handle})
	body := "From: a@x.test\r\nTo: b@x.test\r\nSubject: Hi\r\n\r\nHello\r\n.leading dot\r\n"
	err := netsmtp.SendMail(addr, nil, "a@x.test", []string{"b@x.test", "c@x.test"}, []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	msgs := rec.all()
	if len(msgs) != 1 {
		t.Fatalf("got %d messages", len(msgs))
	}
	m := msgs[0]
	if m.env.From != "a@x.test" || strings.Join(m.env.To, ",") != "b@x.test,c@x.test" {
		t.Errorf("envelope = %+v", m.env)
	}
	if string(m.data) != body {
		t.Errorf("data = %q; want %q (CRLF and dots preserved)", m.data, body)
	}
}

func TestProtocol(t *testing.T) {
	addr := start(t, &Server{Handler: (&recorder{}).handle})
	c := dial(t, addr)
	c.send("MAIL FROM:<a@x>", "503")
	ehlo := c.send("EHLO client", "250")
	for _, ext := range []string{"SIZE 10485760", "8BITMIME", "PIPELINING", "AUTH PLAIN LOGIN"} {
		if !strings.Contains(ehlo, ext) {
			t.Errorf("EHLO missing %q: %s", ext, ehlo)
		}
	}
	c.send("RCPT TO:<b@x>", "503")
	c.send("DATA", "503")
	c.send("MAIL FROM:<>", "250")
	c.send("MAIL FROM:<a@x>", "503")
	c.send("RCPT TO:<>", "501")
	c.send("RCPT TO:<b@x>", "250")
	c.send("RSET", "250")
	c.send("DATA", "503")
	c.send("NOOP", "250")
	c.send("VRFY john", "252")
	c.send("HELP", "214")
	c.send("STARTTLS", "502")
	c.send("BOGUS", "500")
	c.send("HELO", "501")
	c.send("HELO client", "250")
	c.send("QUIT", "221")
}

func TestAuthAcceptsAnythingByDefault(t *testing.T) {
	addr := start(t, &Server{Handler: (&recorder{}).handle})
	session := func(steps ...[2]string) {
		t.Helper()
		c := dial(t, addr)
		c.send("EHLO x", "250")
		for _, s := range steps {
			c.send(s[0], s[1])
		}
	}
	session([2]string{"AUTH PLAIN AHVzZXIAcGFzcw==", "235"}, [2]string{"AUTH PLAIN", "503"})
	session([2]string{"AUTH PLAIN", "334"}, [2]string{"AHVzZXIAcGFzcw==", "235"})
	session([2]string{"AUTH LOGIN", "334 VXNlcm5hbWU6"}, [2]string{"dXNlcg==", "334 UGFzc3dvcmQ6"}, [2]string{"cGFzcw==", "235"})
	session([2]string{"AUTH LOGIN dXNlcg==", "334 UGFzc3dvcmQ6"}, [2]string{"cGFzcw==", "235"})
	session([2]string{"AUTH LOGIN", "334"}, [2]string{"*", "501"})
	session([2]string{"AUTH PLAIN not-base64!", "501"})
	session([2]string{"AUTH CRAM-MD5", "504"})
	session([2]string{"MAIL FROM:<a@x>", "250"})

	_, port, _ := net.SplitHostPort(addr)
	auth := netsmtp.PlainAuth("", "user", "pass", "localhost")
	if err := netsmtp.SendMail("localhost:"+port, auth, "a@x", []string{"b@x"}, []byte("Subject: x\r\n\r\nx\r\n")); err != nil {
		t.Fatal(err)
	}
}

func TestAuthRequired(t *testing.T) {
	rec := &recorder{}
	check := func(user, password string) bool { return user == "app" && password == "s3cret:x" }
	addr := start(t, &Server{Handler: rec.handle, Auth: check})

	c := dial(t, addr)
	c.send("EHLO x", "250")
	c.send("MAIL FROM:<a@x>", "530")
	c.send("AUTH PLAIN "+base64.StdEncoding.EncodeToString([]byte("\x00app\x00wrong")), "535")
	c.send("AUTH LOGIN", "334")
	c.send(base64.StdEncoding.EncodeToString([]byte("app")), "334")
	c.send(base64.StdEncoding.EncodeToString([]byte("s3cret:x")), "235")
	c.send("MAIL FROM:<a@x>", "250")

	_, port, _ := net.SplitHostPort(addr)
	good := netsmtp.PlainAuth("", "app", "s3cret:x", "localhost")
	if err := netsmtp.SendMail("localhost:"+port, good, "a@x", []string{"b@x"}, []byte("Subject: x\r\n\r\nx\r\n")); err != nil {
		t.Fatal(err)
	}
	bad := netsmtp.PlainAuth("", "app", "nope", "localhost")
	if err := netsmtp.SendMail("localhost:"+port, bad, "a@x", []string{"b@x"}, []byte("x")); err == nil || !strings.Contains(err.Error(), "535") {
		t.Fatalf("expected 535, got %v", err)
	}
	if err := netsmtp.SendMail("localhost:"+port, nil, "a@x", []string{"b@x"}, []byte("x")); err == nil || !strings.Contains(err.Error(), "530") {
		t.Fatalf("expected 530 without login, got %v", err)
	}
	if n := len(rec.all()); n != 1 {
		t.Fatalf("stored %d messages", n)
	}
}

func TestMessageTooLarge(t *testing.T) {
	rec := &recorder{}
	addr := start(t, &Server{Handler: rec.handle, MaxMessageSize: 100})
	c := dial(t, addr)
	c.send("EHLO x", "250")
	c.send("MAIL FROM:<a@x> SIZE=1000", "552")
	c.send("MAIL FROM:<a@x>", "250")
	c.send("RCPT TO:<b@x>", "250")
	c.send("DATA", "354")
	fmt.Fprintf(c.conn, "%s\r\n.\r\n", strings.Repeat("x", 500))
	c.expect("552")
	c.send("MAIL FROM:<a@x>", "250")
	c.send("RCPT TO:<b@x>", "250")
	c.send("DATA", "354")
	c.send("small\r\n.", "250")
	if n := len(rec.all()); n != 1 {
		t.Fatalf("stored %d messages", n)
	}
}

func TestLongLinesAndPipelining(t *testing.T) {
	rec := &recorder{}
	addr := start(t, &Server{Handler: rec.handle})
	c := dial(t, addr)
	c.send("NOOP "+strings.Repeat("x", 10000), "500")

	long := strings.Repeat("y", 9000)
	fmt.Fprintf(c.conn, "EHLO x\r\nMAIL FROM:<a@x>\r\nRCPT TO:<b@x>\r\nRCPT TO:<c@x>\r\nDATA\r\n")
	for _, code := range []string{"250", "250", "250", "250", "354"} {
		c.expect(code)
	}
	fmt.Fprintf(c.conn, "Subject: long\r\n\r\n%s\r\n..dot\r\n.\r\nQUIT\r\n", long)
	c.expect("250")
	c.expect("221")

	msgs := rec.all()
	if len(msgs) != 1 || !strings.Contains(string(msgs[0].data), long+"\r\n.dot\r\n") {
		t.Fatalf("unexpected data")
	}
}

func TestTooManyRecipientsAndErrors(t *testing.T) {
	addr := start(t, &Server{Handler: (&recorder{}).handle, MaxRecipients: 2})
	c := dial(t, addr)
	c.send("EHLO x", "250")
	c.send("MAIL FROM:<a@x>", "250")
	c.send("RCPT TO:<1@x>", "250")
	c.send("RCPT TO:<2@x>", "250")
	c.send("RCPT TO:<3@x>", "452")
	for i := 0; i < maxErrors; i++ {
		c.send("BOGUS", "500")
	}
	c.send("BOGUS", "421")
}

func TestIdleTimeout(t *testing.T) {
	addr := start(t, &Server{Handler: (&recorder{}).handle, ReadTimeout: 50 * time.Millisecond})
	c := dial(t, addr)
	_ = c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := c.r.ReadString('\n'); err == nil {
		t.Fatal("expected connection to be closed after idle timeout")
	}
}

func TestConcurrentConnections(t *testing.T) {
	rec := &recorder{}
	addr := start(t, &Server{Handler: rec.handle})
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			msg := fmt.Sprintf("Subject: %d\r\n\r\nbody\r\n", i)
			if err := netsmtp.SendMail(addr, nil, "a@x", []string{fmt.Sprintf("%d@x", i)}, []byte(msg)); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if n := len(rec.all()); n != 25 {
		t.Fatalf("stored %d messages", n)
	}
}

func TestShutdownClosesIdleAndFinishesInFlight(t *testing.T) {
	release := make(chan struct{})
	stored := make(chan struct{}, 1)
	srv := &Server{Handler: func(Envelope, []byte) (string, error) {
		<-release
		stored <- struct{}{}
		return "x", nil
	}}
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	serveDone := make(chan error, 1)
	go func() { serveDone <- srv.Serve(l) }()
	addr := l.Addr().String()

	idle := dial(t, addr)
	busy := dial(t, addr)
	busy.send("HELO x", "250")
	busy.send("MAIL FROM:<a@x>", "250")
	busy.send("RCPT TO:<b@x>", "250")
	busy.send("DATA", "354")
	fmt.Fprint(busy.conn, "Subject: x\r\n\r\nx\r\n.\r\n")
	time.Sleep(50 * time.Millisecond)

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		shutdownDone <- srv.Shutdown(ctx)
	}()

	idle.expect("421")
	close(release)
	<-stored
	busy.expect("250")
	busy.expect("421")

	if err := <-shutdownDone; err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := <-serveDone; err != ErrServerClosed {
		t.Fatalf("serve: %v", err)
	}
	if _, err := net.DialTimeout("tcp", addr, 200*time.Millisecond); err == nil {
		t.Fatal("listener still accepting")
	}
}

func TestShutdownForceClosesAfterDeadline(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	srv := &Server{Handler: func(Envelope, []byte) (string, error) { <-block; return "", nil }}
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	go func() { _ = srv.Serve(l) }()
	c := dial(t, l.Addr().String())
	c.send("HELO x", "250")
	c.send("MAIL FROM:<a@x>", "250")
	c.send("RCPT TO:<b@x>", "250")
	c.send("DATA", "354")
	fmt.Fprint(c.conn, "x\r\n.\r\n")
	time.Sleep(20 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	go func() { time.Sleep(100 * time.Millisecond); block <- struct{}{} }()
	if err := srv.Shutdown(ctx); err != context.DeadlineExceeded {
		t.Fatalf("shutdown = %v", err)
	}
}

func TestParsePath(t *testing.T) {
	cases := []struct {
		in, prefix, addr string
		params           int
		ok               bool
	}{
		{"FROM:<a@b.c>", "FROM:", "a@b.c", 0, true},
		{"from: <a@b.c> SIZE=10 BODY=8BITMIME", "FROM:", "a@b.c", 2, true},
		{"FROM:<>", "FROM:", "", 0, true},
		{"FROM:a@b.c", "FROM:", "a@b.c", 0, true},
		{"TO:<@relay.x,@r2.y:user@z.w>", "TO:", "user@z.w", 0, true},
		{"FROM:<a@b.c", "FROM:", "", 0, false},
		{"TO:<a@b.c>", "FROM:", "", 0, false},
		{"FROM:", "FROM:", "", 0, false},
	}
	for _, tc := range cases {
		addr, params, ok := parsePath(tc.in, tc.prefix)
		if ok != tc.ok || addr != tc.addr || len(params) != tc.params {
			t.Errorf("parsePath(%q) = %q %v %v", tc.in, addr, params, ok)
		}
	}
}

func BenchmarkReceive(b *testing.B) {
	srv := &Server{Handler: func(Envelope, []byte) (string, error) { return "x", nil }}
	addr := start(b, srv)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	readReply := func() {
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				b.Fatal(err)
			}
			if len(line) < 4 || line[3] != '-' {
				return
			}
		}
	}
	readReply()
	fmt.Fprint(conn, "EHLO bench\r\n")
	readReply()
	body := "From: a@x\r\nTo: b@x\r\nSubject: bench\r\n\r\n" + strings.Repeat("Hello world line\r\n", 200) + ".\r\n"
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	for b.Loop() {
		fmt.Fprint(conn, "MAIL FROM:<a@x>\r\nRCPT TO:<b@x>\r\nDATA\r\n")
		readReply()
		readReply()
		readReply()
		fmt.Fprint(conn, body)
		readReply()
	}
}

func TestMaxConnections(t *testing.T) {
	addr := start(t, &Server{Handler: (&recorder{}).handle, MaxConnections: 2})
	a, b := dial(t, addr), dial(t, addr)
	defer a.conn.Close()
	defer b.conn.Close()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	third := &client{t: t, conn: conn, r: bufio.NewReader(conn)}
	third.expect("421")
	a.send("QUIT", "221")
	time.Sleep(50 * time.Millisecond)
	dial(t, addr).send("NOOP", "250")
}
