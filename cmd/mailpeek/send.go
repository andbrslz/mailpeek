package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/andbrslz/mailpeek/internal/config"
	"github.com/andbrslz/mailpeek/internal/mail"
)

type sendOptions struct {
	host, from, subject, text, html, auth string
	port                                  int
	to                                    []string
}

func send(args []string, getenv func(string) string, out io.Writer) error {
	opts := sendOptions{
		host:    "localhost",
		port:    config.DefaultSMTPPort,
		from:    "mailpeek@localhost",
		subject: "Mailpeek test",
		text:    "This is a test email sent by mailpeek send.",
		auth:    getenv("MAILPEEK_SMTP_AUTH"),
	}
	if v := getenv("MAILPEEK_SMTP_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("MAILPEEK_SMTP_PORT must be a number, got %q", v)
		}
		opts.port = p
	}

	fs := flag.NewFlagSet("mailpeek send", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: mailpeek send --to address [flags]\n\nSends a test email to a running Mailpeek.")
		fs.PrintDefaults()
	}
	fs.Func("to", "recipient; repeat or separate with commas (required)", func(v string) error {
		for _, a := range strings.Split(v, ",") {
			if a = strings.TrimSpace(a); a != "" {
				opts.to = append(opts.to, a)
			}
		}
		return nil
	})
	fs.StringVar(&opts.from, "from", opts.from, "sender address")
	fs.StringVar(&opts.subject, "subject", opts.subject, "subject")
	fs.StringVar(&opts.text, "text", opts.text, "plain-text body")
	fs.StringVar(&opts.html, "html", "", "HTML body (sent together with the text body)")
	fs.StringVar(&opts.host, "host", opts.host, "Mailpeek SMTP host")
	fs.IntVar(&opts.port, "port", opts.port, "Mailpeek SMTP port (env MAILPEEK_SMTP_PORT)")
	fs.StringVar(&opts.auth, "auth", opts.auth, "SMTP login as user:password, when Mailpeek runs with --smtp-auth (env MAILPEEK_SMTP_AUTH)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q (use --to, --subject, --text...)", fs.Arg(0))
	}
	if len(opts.to) == 0 {
		return errors.New("--to is required, e.g. mailpeek send --to ana@example.com")
	}

	id, err := deliver(opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Sent %q to %s (id %s)\n", opts.subject, strings.Join(opts.to, ", "), id)
	return nil
}

func deliver(opts sendOptions) (string, error) {
	addr := net.JoinHostPort(opts.host, strconv.Itoa(opts.port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("cannot connect to SMTP at %s: is Mailpeek running, and on this port "+
			"(--port or MAILPEEK_SMTP_PORT)? %w", addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	c, err := smtp.NewClient(conn, opts.host)
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("SMTP at %s: %w", addr, err)
	}
	defer c.Close()

	if err := c.Hello("localhost"); err != nil {
		return "", err
	}
	if opts.auth != "" {
		creds, err := config.ParseCredentials(opts.auth)
		if err != nil {
			return "", fmt.Errorf("--auth: %w", err)
		}
		if err := c.Auth(plainAuth(creds)); err != nil {
			return "", fmt.Errorf("SMTP login failed: %w", err)
		}
	}
	if err := c.Mail(opts.from); err != nil {
		return "", withHint(err)
	}
	for _, rcpt := range opts.to {
		if err := c.Rcpt(rcpt); err != nil {
			return "", withHint(err)
		}
	}
	msg, err := compose(opts)
	if err != nil {
		return "", err
	}
	id, err := data(c.Text, msg)
	if err != nil {
		return "", err
	}
	_ = c.Quit()
	return id, nil
}

func data(t *textproto.Conn, msg []byte) (string, error) {
	cmd, err := t.Cmd("DATA")
	if err != nil {
		return "", err
	}
	t.StartResponse(cmd)
	_, _, err = t.ReadResponse(354)
	t.EndResponse(cmd)
	if err != nil {
		return "", err
	}
	w := t.DotWriter()
	if _, err := w.Write(msg); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	_, reply, err := t.ReadResponse(250)
	if err != nil {
		return "", err
	}
	if _, id, ok := strings.Cut(reply, "queued as "); ok {
		return strings.TrimSpace(id), nil
	}
	return "", nil
}

func withHint(err error) error {
	var tp *textproto.Error
	if errors.As(err, &tp) && tp.Code == 530 {
		return fmt.Errorf("%w (Mailpeek requires an SMTP login: pass --auth user:password or set MAILPEEK_SMTP_AUTH)", err)
	}
	return err
}

type plainAuth config.Credentials

func (a plainAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + a.User + "\x00" + a.Password), nil
}

func (a plainAuth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("unexpected SMTP login challenge")
	}
	return nil, nil
}

func compose(opts sendOptions) ([]byte, error) {
	var b bytes.Buffer
	header := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	header("From", opts.from)
	header("To", strings.Join(opts.to, ", "))
	header("Subject", mime.QEncoding.Encode("utf-8", opts.subject))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("Message-ID", "<"+mail.NewID()+"@mailpeek.send>")
	header("MIME-Version", "1.0")

	if opts.html == "" {
		header("Content-Type", "text/plain; charset=utf-8")
		header("Content-Transfer-Encoding", "quoted-printable")
		b.WriteString("\r\n")
		if err := writeQP(&b, opts.text); err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	}

	mw := multipart.NewWriter(&b)
	header("Content-Type", mime.FormatMediaType("multipart/alternative", map[string]string{"boundary": mw.Boundary()}))
	b.WriteString("\r\n")
	for _, part := range []struct{ mediaType, body string }{{"text/plain", opts.text}, {"text/html", opts.html}} {
		w, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {part.mediaType + "; charset=utf-8"},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return nil, err
		}
		if err := writeQP(w, part.body); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func writeQP(w io.Writer, s string) error {
	qp := quotedprintable.NewWriter(w)
	if _, err := io.WriteString(qp, s); err != nil {
		return err
	}
	return qp.Close()
}
