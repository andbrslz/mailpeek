package store

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/andbrslz/mailpeek/internal/mail"
)

type Filter struct {
	To      string
	Address string
	From    string
	Subject string
	Body    string
	Query   string
	Since   time.Time
}

func (f Filter) IsZero() bool { return f == Filter{} }

func (f Filter) Match(m *mail.Message) bool {
	if !f.Since.IsZero() && m.CreatedAt.Before(f.Since) {
		return false
	}
	if f.To != "" && !matchTo(m, lower(f.To)) {
		return false
	}
	if f.Address != "" && !hasRecipient(m, f.Address) {
		return false
	}
	if f.From != "" && !matchFrom(m, lower(f.From)) {
		return false
	}
	if f.Subject != "" && !contains(m.Subject, lower(f.Subject)) {
		return false
	}
	if f.Body != "" && !matchBody(m, lower(f.Body)) {
		return false
	}
	if f.Query != "" {
		q := lower(f.Query)
		if !contains(m.Subject, q) && !matchFrom(m, q) && !matchTo(m, q) && !matchBody(m, q) {
			return false
		}
	}
	return true
}

func matchTo(m *mail.Message, needle string) bool {
	for _, list := range [][]mail.Address{m.To, m.Cc} {
		for _, a := range list {
			if matchAddress(a, needle) {
				return true
			}
		}
	}
	for _, rcpt := range m.Envelope.To {
		if contains(rcpt, needle) {
			return true
		}
	}
	return false
}

func hasRecipient(m *mail.Message, address string) bool {
	address = strings.TrimSpace(address)
	for _, list := range [][]mail.Address{m.To, m.Cc} {
		for _, a := range list {
			if strings.EqualFold(a.Address, address) {
				return true
			}
		}
	}
	for _, rcpt := range m.Envelope.To {
		if strings.EqualFold(rcpt, address) {
			return true
		}
	}
	return false
}

func matchBody(m *mail.Message, needle string) bool {
	return contains(m.Text, needle) || contains(m.HTML, needle)
}

func matchFrom(m *mail.Message, needle string) bool {
	return matchAddress(m.From, needle) || contains(m.Envelope.From, needle)
}

func matchAddress(a mail.Address, needle string) bool {
	return contains(a.Address, needle) || contains(a.Name, needle)
}

func contains(s, lowerNeedle string) bool {
	if lowerNeedle == "" {
		return true
	}
	first := lowerNeedle[0]
	for i := 0; i < len(s); {
		if c := s[i]; c < utf8.RuneSelf {
			if lowerASCII(c) == first && hasLowerPrefix(s[i:], lowerNeedle) {
				return true
			}
			i++
			continue
		}
		if hasLowerPrefix(s[i:], lowerNeedle) {
			return true
		}
		_, n := utf8.DecodeRuneInString(s[i:])
		i += n
	}
	return false
}

func lowerASCII(c byte) byte {
	if 'A' <= c && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

func hasLowerPrefix(s, prefix string) bool {
	var buf [utf8.UTFMax]byte
	for prefix != "" {
		if s == "" {
			return false
		}
		if c := s[0]; c < utf8.RuneSelf {
			if lowerASCII(c) != prefix[0] {
				return false
			}
			s, prefix = s[1:], prefix[1:]
			continue
		}
		r, n := utf8.DecodeRuneInString(s)
		m := utf8.EncodeRune(buf[:], unicode.ToLower(r))
		if len(prefix) < m || prefix[:m] != string(buf[:m]) {
			return false
		}
		s, prefix = s[n:], prefix[m:]
	}
	return true
}

func lower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
