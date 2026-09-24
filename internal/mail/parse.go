package mail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

const (
	maxDepth = 16
	maxParts = 500
)

var wordDecoder = &mime.WordDecoder{CharsetReader: charsetReader}

func Parse(raw []byte, env Envelope) *Message {
	msg := &Message{
		ID:          NewID(),
		To:          []Address{},
		Cc:          []Address{},
		ReplyTo:     []Address{},
		Headers:     map[string][]string{},
		Attachments: []Attachment{},
		Envelope:    env,
		Size:        len(raw),
		CreatedAt:   time.Now().UTC(),
	}
	if msg.Envelope.To == nil {
		msg.Envelope.To = []string{}
	}

	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		msg.From = Address{Address: env.From}
		msg.Text = toUTF8("", raw)
		msg.Links = ExtractLinks("", msg.Text)
		return msg
	}

	parseHeaders(msg, m.Header)
	p := parser{msg: msg}
	p.walk(textproto.MIMEHeader(m.Header), m.Body, 0)
	msg.Links = ExtractLinks(msg.HTML, msg.Text)
	return msg
}

func parseHeaders(msg *Message, h mail.Header) {
	for k, v := range h {
		msg.Headers[k] = append([]string(nil), v...)
	}
	msg.Subject = decodeHeader(h.Get("Subject"))
	msg.MessageID = strings.TrimSpace(h.Get("Message-Id"))
	if from := parseAddresses(h.Get("From")); len(from) > 0 {
		msg.From = from[0]
	} else {
		msg.From = Address{Address: msg.Envelope.From}
	}
	msg.To = parseAddresses(h.Get("To"))
	msg.Cc = parseAddresses(h.Get("Cc"))
	msg.ReplyTo = parseAddresses(h.Get("Reply-To"))
	if d, err := mail.ParseDate(h.Get("Date")); err == nil {
		d = d.UTC()
		msg.Date = &d
	}
}

type parser struct {
	msg   *Message
	parts int
}

func (p *parser) walk(h textproto.MIMEHeader, body io.Reader, depth int) {
	p.parts++
	if p.parts > maxParts {
		return
	}
	mediaType, params, err := mime.ParseMediaType(h.Get("Content-Type"))
	if err != nil || mediaType == "" {
		mediaType, params = "text/plain", map[string]string{}
	}

	if strings.HasPrefix(mediaType, "multipart/") && params["boundary"] != "" && depth < maxDepth {
		mr := multipart.NewReader(body, params["boundary"])
		for {
			part, err := mr.NextRawPart()
			if err != nil {
				return
			}
			p.walk(part.Header, part, depth+1)
		}
	}

	data := decodeTransfer(h.Get("Content-Transfer-Encoding"), body)
	disposition, dparams, _ := mime.ParseMediaType(h.Get("Content-Disposition"))
	filename := dparams["filename"]
	if filename == "" {
		filename = params["name"]
	}
	filename = decodeHeader(filename)

	if disposition != "attachment" && filename == "" && p.setBody(mediaType, params["charset"], data) {
		return
	}
	p.addAttachment(h, mediaType, disposition, filename, data)
}

func (p *parser) setBody(mediaType, charset string, data []byte) bool {
	switch {
	case mediaType == "text/plain" && p.msg.Text == "":
		p.msg.Text = toUTF8(charset, data)
	case mediaType == "text/html" && p.msg.HTML == "":
		p.msg.HTML = toUTF8(charset, data)
	default:
		return false
	}
	return true
}

func (p *parser) addAttachment(h textproto.MIMEHeader, mediaType, disposition, filename string, data []byte) {
	id := strconv.Itoa(len(p.msg.Attachments) + 1)
	contentID := strings.Trim(strings.TrimSpace(h.Get("Content-Id")), "<>")
	if filename == "" {
		filename = "attachment-" + id + extensionFor(mediaType)
	}
	p.msg.Attachments = append(p.msg.Attachments, Attachment{
		ID:          id,
		Filename:    filename,
		ContentType: mediaType,
		ContentID:   contentID,
		Inline:      disposition == "inline" || (disposition == "" && contentID != ""),
		Size:        len(data),
		Data:        data,
	})
}

func extensionFor(mediaType string) string {
	if mediaType == "message/rfc822" {
		return ".eml"
	}
	if exts, _ := mime.ExtensionsByType(mediaType); len(exts) > 0 {
		return exts[0]
	}
	return ".bin"
}

func decodeTransfer(encoding string, body io.Reader) []byte {
	raw, _ := io.ReadAll(body)
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		clean := bytes.Map(func(r rune) rune {
			if r == ' ' || r == '\t' || r == '\r' || r == '\n' {
				return -1
			}
			return r
		}, raw)
		out := make([]byte, base64.StdEncoding.DecodedLen(len(clean)))
		n, err := base64.StdEncoding.Decode(out, clean)
		if err != nil {
			n, _ = base64.RawStdEncoding.Decode(out, bytes.TrimRight(clean, "="))
		}
		return out[:n]
	case "quoted-printable":
		out, _ := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(raw)))
		return out
	}
	return raw
}

func decodeHeader(v string) string {
	if !strings.Contains(v, "=?") {
		return strings.TrimSpace(v)
	}
	decoded, err := wordDecoder.DecodeHeader(v)
	if err != nil {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(decoded)
}

func parseAddresses(v string) []Address {
	out := []Address{}
	if strings.TrimSpace(v) == "" {
		return out
	}
	parser := mail.AddressParser{WordDecoder: wordDecoder}
	list, err := parser.ParseList(v)
	if err != nil {
		for _, part := range strings.Split(decodeHeader(v), ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, Address{Address: part})
			}
		}
		return out
	}
	for _, a := range list {
		out = append(out, Address{Name: a.Name, Address: a.Address})
	}
	return out
}

func (a Address) String() string {
	if a.Name == "" {
		return a.Address
	}
	return fmt.Sprintf("%s <%s>", a.Name, a.Address)
}
