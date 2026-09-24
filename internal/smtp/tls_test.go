package smtp

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	netsmtp "net/smtp"
	"strings"
	"testing"
)

func trustedBy(t *testing.T, cfg *tls.Config, serverName string) *tls.Config {
	t.Helper()
	leaf, err := x509.ParseCertificate(cfg.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return &tls.Config{RootCAs: pool, ServerName: serverName}
}

func TestStartTLS(t *testing.T) {
	cfg, err := SelfSignedTLS("localhost", "127.0.0.1", "::1")
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	addr := start(t, &Server{Handler: rec.handle, TLSConfig: cfg, Auth: func(u, p string) bool { return u == "app" && p == "secret" }})

	c, err := netsmtp.Dial(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); !ok {
		t.Fatal("STARTTLS is not advertised")
	}
	if err := c.StartTLS(trustedBy(t, cfg, "127.0.0.1")); err != nil {
		t.Fatal(err)
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		t.Fatal("STARTTLS is still advertised inside TLS")
	}
	if err := c.Auth(netsmtp.PlainAuth("", "app", "secret", "127.0.0.1")); err != nil {
		t.Fatal(err)
	}
	if err := c.Mail("app@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := c.Rcpt("ana@example.com"); err != nil {
		t.Fatal(err)
	}
	w, err := c.Data()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(w, "Subject: Secure\r\n\r\nhello\r\n")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	_ = c.Quit()

	msgs := rec.all()
	if len(msgs) != 1 || !msgs[0].env.TLS || !strings.Contains(string(msgs[0].data), "Subject: Secure") {
		t.Fatalf("received = %+v", msgs)
	}
}

func TestStartTLSVerifiesForLocalhostNames(t *testing.T) {
	cfg, err := SelfSignedTLS("localhost", "127.0.0.1", "::1", "mailpeek")
	if err != nil {
		t.Fatal(err)
	}
	leaf, _ := x509.ParseCertificate(cfg.Certificates[0].Certificate[0])
	for _, name := range []string{"localhost", "127.0.0.1", "::1", "mailpeek"} {
		if err := leaf.VerifyHostname(name); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestStartTLSNotOfferedByDefault(t *testing.T) {
	addr := start(t, &Server{Handler: (&recorder{}).handle})
	c := dial(t, addr)
	if reply := c.send("EHLO x", "250"); strings.Contains(reply, "STARTTLS") {
		t.Fatalf("STARTTLS advertised without TLS: %q", reply)
	}
	c.send("STARTTLS", "502")
}

func TestStartTLSDiscardsPipelinedPlaintext(t *testing.T) {
	cfg, err := SelfSignedTLS("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	addr := start(t, &Server{Handler: (&recorder{}).handle, TLSConfig: cfg})
	c := dial(t, addr)
	c.send("EHLO x", "250")
	fmt.Fprint(c.conn, "STARTTLS\r\nMAIL FROM:<injected@evil.test>\r\n")
	c.expect("220")

	conn := tls.Client(c.conn, trustedBy(t, cfg, "127.0.0.1"))
	if err := conn.Handshake(); err != nil {
		t.Fatal(err)
	}
	secure := &client{t: t, conn: conn, r: bufio.NewReader(conn)}
	secure.send("EHLO x", "250")
	secure.send("RCPT TO:<ana@example.com>", "503")
}
