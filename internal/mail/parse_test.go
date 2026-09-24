package mail

import (
	"strings"
	"testing"
)

func crlf(s string) []byte { return []byte(strings.ReplaceAll(s, "\n", "\r\n")) }

func TestParsePlainText(t *testing.T) {
	raw := crlf(`From: "Acme" <hello@acme.com>
To: john@example.com, Jane <jane@example.com>
Cc: boss@example.com
Reply-To: support@acme.com
Subject: Welcome
Date: Mon, 02 Jan 2006 15:04:05 +0000
Message-ID: <abc@acme.com>
X-Custom: one
X-Custom: two

Hello John,
Visit https://example.com/start.
`)
	m := Parse(raw, Envelope{From: "bounce@acme.com", To: []string{"john@example.com"}})

	if m.From != (Address{Name: "Acme", Address: "hello@acme.com"}) {
		t.Errorf("from = %+v", m.From)
	}
	if len(m.To) != 2 || m.To[1] != (Address{Name: "Jane", Address: "jane@example.com"}) {
		t.Errorf("to = %+v", m.To)
	}
	if len(m.Cc) != 1 || m.Cc[0].Address != "boss@example.com" {
		t.Errorf("cc = %+v", m.Cc)
	}
	if len(m.ReplyTo) != 1 || m.ReplyTo[0].Address != "support@acme.com" {
		t.Errorf("replyTo = %+v", m.ReplyTo)
	}
	if m.Subject != "Welcome" || m.MessageID != "<abc@acme.com>" {
		t.Errorf("subject/messageId = %q %q", m.Subject, m.MessageID)
	}
	if m.Date == nil || m.Date.Year() != 2006 {
		t.Errorf("date = %v", m.Date)
	}
	if got := m.Headers["X-Custom"]; len(got) != 2 || got[1] != "two" {
		t.Errorf("custom headers = %v", got)
	}
	if !strings.Contains(m.Text, "Hello John,") || m.HTML != "" {
		t.Errorf("text = %q html = %q", m.Text, m.HTML)
	}
	if len(m.Links) != 1 || m.Links[0].Href != "https://example.com/start" {
		t.Errorf("text links = %+v", m.Links)
	}
	if m.Size != len(raw) || m.ID == "" || m.CreatedAt.IsZero() {
		t.Errorf("metadata not set: %+v", m)
	}
}

func TestParseMultipartAlternative(t *testing.T) {
	raw := crlf(`From: hello@acme.com
To: john@example.com
Subject: =?UTF-8?B?V2VsY29tZSDwn5GL?=
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="b1"

--b1
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable

Hello John, caf=C3=A9 =
continues
--b1
Content-Type: text/html; charset=utf-8

<p>Hello <b>John</b></p><a href="http://localhost:3000/activate/abc?x=1&amp;y=2">Activate <span>account</span></a>
--b1--
`)
	m := Parse(raw, Envelope{})
	if m.Subject != "Welcome \U0001F44B" {
		t.Errorf("subject = %q", m.Subject)
	}
	if !strings.Contains(m.Text, "Hello John, café continues") {
		t.Errorf("text = %q", m.Text)
	}
	if !strings.Contains(m.HTML, "<b>John</b>") {
		t.Errorf("html = %q", m.HTML)
	}
	want := Link{Text: "Activate account", Href: "http://localhost:3000/activate/abc?x=1&y=2"}
	if len(m.Links) != 1 || m.Links[0] != want {
		t.Errorf("links = %+v", m.Links)
	}
	if len(m.Attachments) != 0 {
		t.Errorf("unexpected attachments: %+v", m.Attachments)
	}
}

func TestParseAttachments(t *testing.T) {
	raw := crlf(`From: billing@acme.com
To: john@example.com
Subject: Invoice
Content-Type: multipart/mixed; boundary=outer

--outer
Content-Type: multipart/related; boundary=inner

--inner
Content-Type: text/html

<img src="cid:logo@acme">
--inner
Content-Type: image/png
Content-ID: <logo@acme>
Content-Transfer-Encoding: base64

iVBORw0K
GgoAAAA=
--inner--
--outer
Content-Type: application/pdf; name="ignored.pdf"
Content-Disposition: attachment; filename="=?UTF-8?Q?fatura_n=C2=BA1.pdf?="
Content-Transfer-Encoding: base64

JVBERi0xLjQK
--outer
Content-Type: text/plain
Content-Disposition: attachment; filename="notes.txt"

plain attachment, not the body
--outer--
`)
	m := Parse(raw, Envelope{})
	if m.Text != "" {
		t.Errorf("text attachment was used as body: %q", m.Text)
	}
	if len(m.Attachments) != 3 {
		t.Fatalf("attachments = %+v", m.Attachments)
	}
	logo, pdf, notes := m.Attachments[0], m.Attachments[1], m.Attachments[2]
	if !logo.Inline || logo.ContentID != "logo@acme" || logo.ContentType != "image/png" || logo.Size != 11 {
		t.Errorf("inline image = %+v", logo)
	}
	if !strings.HasPrefix(logo.Filename, "attachment-1") {
		t.Errorf("generated filename = %q", logo.Filename)
	}
	if pdf.Filename != "fatura nº1.pdf" || pdf.Inline || string(pdf.Data) != "%PDF-1.4\n" {
		t.Errorf("pdf = %+v %q", pdf, pdf.Data)
	}
	if notes.Filename != "notes.txt" || notes.ID != "3" {
		t.Errorf("notes = %+v", notes)
	}
	if a, ok := m.Attachment("2"); !ok || a.Filename != pdf.Filename {
		t.Errorf("Attachment lookup failed")
	}
	if _, ok := m.Attachment("../../etc/passwd"); ok {
		t.Errorf("unexpected attachment match")
	}
}

func TestParseCharsets(t *testing.T) {
	raw := []byte("From: a@b.c\r\nSubject: =?ISO-8859-1?Q?Ol=E1?=\r\nContent-Type: text/plain; charset=windows-1252\r\n\r\nCaf\xe9 \x93quoted\x94\r\n")
	m := Parse(raw, Envelope{})
	if m.Subject != "Olá" {
		t.Errorf("subject = %q", m.Subject)
	}
	if !strings.Contains(m.Text, "Café “quoted”") {
		t.Errorf("text = %q", m.Text)
	}
}

func TestParseMalformedFallsBack(t *testing.T) {
	m := Parse([]byte("this is not\x00 a valid header\r\n\r\nbody"), Envelope{From: "env@x.y"})
	if m.From.Address != "env@x.y" || !strings.Contains(m.Text, "body") {
		t.Errorf("fallback = %+v", m)
	}
}

func TestParseMissingFromUsesEnvelope(t *testing.T) {
	m := Parse(crlf("To: x@y.z\n\nhi\n"), Envelope{From: "env@x.y"})
	if m.From.Address != "env@x.y" {
		t.Errorf("from = %+v", m.From)
	}
}

func TestParseUnparseableAddressKeepsRaw(t *testing.T) {
	m := Parse(crlf("From: a@b.c\nTo: not an address, other\n\nhi\n"), Envelope{})
	if len(m.To) != 2 || m.To[0].Address != "not an address" {
		t.Errorf("to = %+v", m.To)
	}
}

func TestExtractLinks(t *testing.T) {
	htmlBody := `
<a href="https://a.test/1">  First
   link </a>
<a href="https://a.test/2"><img src="x.png" alt="Logo"></a>
<a href="https://a.test/3" aria-label="Labelled"></a>
<a name="anchor-without-href">skip</a>
<a href="https://a.test/1">First link</a>
<a href="mailto:x@y.z">Mail<br>us</a>
<a href='https://a.test/4'>Unclosed`
	got := ExtractLinks(htmlBody, "ignored https://text.test")
	want := []Link{
		{"First link", "https://a.test/1"},
		{"Logo", "https://a.test/2"},
		{"Labelled", "https://a.test/3"},
		{"Mail us", "mailto:x@y.z"},
		{"Unclosed", "https://a.test/4"},
	}
	if len(got) != len(want) {
		t.Fatalf("links = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("link %d = %+v; want %+v", i, got[i], want[i])
		}
	}
}

func TestTextLinksTrimPunctuation(t *testing.T) {
	got := ExtractLinks("", "Go to (https://x.test/a?b=1). Or https://x.test/a?b=1, again.")
	if len(got) != 1 || got[0].Href != "https://x.test/a?b=1" {
		t.Errorf("links = %+v", got)
	}
}

func TestSummary(t *testing.T) {
	m := Parse(crlf("From: a@b.c\nTo: x@y.z\nSubject: Hi\n\nbody\n"), Envelope{})
	s := m.Summary()
	if s.ID != m.ID || s.Subject != "Hi" || s.Attachments != 0 || s.Size != m.Size {
		t.Errorf("summary = %+v", s)
	}
}
