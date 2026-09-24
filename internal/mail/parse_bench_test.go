package mail

import (
	"strings"
	"testing"
)

func benchMessage() []byte {
	html := strings.Repeat(`<p>Hello John, <a href="https://example.com/x">a link</a></p>`, 100)
	return crlf(`From: "Acme" <hello@acme.com>
To: john@example.com
Subject: =?UTF-8?Q?Welcome_to_Acme?=
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary=outer

--outer
Content-Type: multipart/alternative; boundary=alt

--alt
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable

` + strings.Repeat("Hello John, welcome to Acme =E2=9C=93\n", 100) + `
--alt
Content-Type: text/html; charset=utf-8

` + html + `
--alt--
--outer
Content-Type: application/pdf
Content-Disposition: attachment; filename="invoice.pdf"
Content-Transfer-Encoding: base64

` + strings.Repeat("JVBERi0xLjQKJcfsj6IKNSAwIG9iago8PC9MZW5ndGggNiAwIFI+PgpzdHJlYW0K\n", 200) + `
--outer--
`)
}

func BenchmarkParse(b *testing.B) {
	raw := benchMessage()
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for b.Loop() {
		Parse(raw, Envelope{})
	}
}

func BenchmarkExtractLinks(b *testing.B) {
	html := strings.Repeat(`<p>Hello <a href="https://example.com/x">a <b>link</b></a></p>`, 200)
	b.ReportAllocs()
	for b.Loop() {
		ExtractLinks(html, "")
	}
}
