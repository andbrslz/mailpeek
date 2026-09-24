package events

import (
	"sync"
	"sync/atomic"

	"github.com/andbrslz/mailpeek/internal/mail"
)

const (
	MessageCreated  = "message.created"
	MessageDeleted  = "message.deleted"
	MessagesCleared = "messages.cleared"
)

type Event struct {
	Type    string
	ID      string
	Message *mail.Message
}

const bufferSize = 64

type Subscription struct {
	C      <-chan Event
	ch     chan Event
	missed atomic.Int64
	broker *Broker
}

func (s *Subscription) Missed() int64 { return s.missed.Swap(0) }

func (s *Subscription) Close() { s.broker.remove(s) }

type Broker struct {
	mu     sync.Mutex
	subs   map[*Subscription]struct{}
	closed bool
}

func NewBroker() *Broker {
	return &Broker{subs: make(map[*Subscription]struct{})}
}

func (b *Broker) Subscribe() *Subscription {
	ch := make(chan Event, bufferSize)
	s := &Subscription{C: ch, ch: ch, broker: b}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		close(ch)
	} else {
		b.subs[s] = struct{}{}
	}
	return s
}

func (b *Broker) remove(s *Subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subs[s]; ok {
		delete(b.subs, s)
		close(s.ch)
	}
}

func (b *Broker) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for s := range b.subs {
		select {
		case s.ch <- e:
		default:
			s.missed.Add(1)
		}
	}
}

func (b *Broker) Subscribers() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}

func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	for s := range b.subs {
		delete(b.subs, s)
		close(s.ch)
	}
}
