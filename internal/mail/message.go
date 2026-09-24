package mail

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	ContentID   string `json:"contentId,omitempty"`
	Inline      bool   `json:"inline"`
	Size        int    `json:"size"`
	Data        []byte `json:"-"`
}

type Link struct {
	Text string `json:"text"`
	Href string `json:"href"`
}

type Envelope struct {
	From string   `json:"from"`
	To   []string `json:"to"`
}

type Message struct {
	ID          string              `json:"id"`
	MessageID   string              `json:"messageId,omitempty"`
	From        Address             `json:"from"`
	To          []Address           `json:"to"`
	Cc          []Address           `json:"cc"`
	ReplyTo     []Address           `json:"replyTo"`
	Subject     string              `json:"subject"`
	Date        *time.Time          `json:"date,omitempty"`
	Text        string              `json:"text"`
	HTML        string              `json:"html"`
	Headers     map[string][]string `json:"headers"`
	Attachments []Attachment        `json:"attachments"`
	Links       []Link              `json:"links"`
	Envelope    Envelope            `json:"envelope"`
	Size        int                 `json:"size"`
	CreatedAt   time.Time           `json:"createdAt"`
}

type Summary struct {
	ID          string    `json:"id"`
	From        Address   `json:"from"`
	To          []Address `json:"to"`
	Cc          []Address `json:"cc"`
	Subject     string    `json:"subject"`
	Envelope    Envelope  `json:"envelope"`
	Attachments int       `json:"attachments"`
	Size        int       `json:"size"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (m *Message) Summary() Summary {
	return Summary{
		ID:          m.ID,
		From:        m.From,
		To:          m.To,
		Cc:          m.Cc,
		Subject:     m.Subject,
		Envelope:    m.Envelope,
		Attachments: len(m.Attachments),
		Size:        m.Size,
		CreatedAt:   m.CreatedAt,
	}
}

func (m *Message) Attachment(id string) (Attachment, bool) {
	for _, a := range m.Attachments {
		if a.ID == id {
			return a, true
		}
	}
	return Attachment{}, false
}

func NewID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
