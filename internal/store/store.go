package store

import (
	"cmp"
	"slices"
	"sync"

	"github.com/andbrslz/mailpeek/internal/events"
	"github.com/andbrslz/mailpeek/internal/mail"
)

type entry struct {
	msg  *mail.Message
	raw  []byte
	size int64
	seq  uint64
}

func sizeOf(m *mail.Message, raw []byte) int64 {
	n := len(raw) + len(m.Text) + len(m.HTML)
	for _, a := range m.Attachments {
		n += len(a.Data)
	}
	return int64(n)
}

type Stats struct {
	Messages int   `json:"messages"`
	Bytes    int64 `json:"bytes"`
	Evicted  int64 `json:"evicted"`
}

type MemoryStore struct {
	mu       sync.RWMutex
	max      int
	maxBytes int64
	bytes    int64
	evicted  int64
	seq      uint64
	order    []string
	items    map[string]entry
	broker   *events.Broker
	disk     *disk
}

func New(max int, broker *events.Broker) *MemoryStore {
	if max < 1 {
		max = 1
	}
	return &MemoryStore{max: max, items: make(map[string]entry), broker: broker}
}

func (s *MemoryStore) WithMaxBytes(n int64) *MemoryStore {
	s.maxBytes = n
	return s
}

func (s *MemoryStore) Save(m *mail.Message, raw []byte) {
	s.save(m, raw, true)
}

func (s *MemoryStore) save(m *mail.Message, raw []byte, persist bool) {
	s.mu.Lock()
	s.seq++
	e := entry{msg: m, raw: raw, size: sizeOf(m, raw), seq: s.seq}
	s.items[m.ID] = e
	s.bytes += e.size
	s.order = append(s.order, m.ID)
	var evicted []string
	for len(s.order) > s.max || (s.maxBytes > 0 && s.bytes > s.maxBytes && len(s.order) > 1) {
		oldest := s.order[0]
		evicted = append(evicted, oldest)
		s.bytes -= s.items[oldest].size
		delete(s.items, oldest)
		s.order = slices.Delete(s.order, 0, 1)
	}
	s.evicted += int64(len(evicted))
	flush := s.onDisk(func(d *disk) {
		if persist {
			d.write(m, raw)
		}
		d.remove(evicted)
	})
	s.mu.Unlock()
	flush()

	for _, id := range evicted {
		s.publish(events.Event{Type: events.MessageDeleted, ID: id})
	}
	s.publish(events.Event{Type: events.MessageCreated, ID: m.ID, Message: m})
}

func (s *MemoryStore) Get(id string) (*mail.Message, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.items[id]
	return e.msg, ok
}

func (s *MemoryStore) Raw(id string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.items[id]
	return e.raw, ok
}

func (s *MemoryStore) List(f Filter) []*mail.Message {
	return s.Page(f, 0, 0).Messages
}

type Page struct {
	Messages []*mail.Message
	Next     uint64
}

func (s *MemoryStore) Page(f Filter, cursor uint64, limit int) Page {
	s.mu.RLock()
	defer s.mu.RUnlock()
	end := len(s.order)
	if cursor > 0 {
		end, _ = slices.BinarySearchFunc(s.order, cursor, func(id string, seq uint64) int {
			return cmp.Compare(s.items[id].seq, seq)
		})
	}
	var p Page
	var last uint64
	for i := end - 1; i >= 0; i-- {
		e := s.items[s.order[i]]
		if !f.Match(e.msg) {
			continue
		}
		if limit > 0 && len(p.Messages) == limit {
			p.Next = last
			break
		}
		p.Messages = append(p.Messages, e.msg)
		last = e.seq
	}
	return p
}

func (s *MemoryStore) Count(f Filter) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if f.IsZero() {
		return len(s.order)
	}
	n := 0
	for _, id := range s.order {
		if f.Match(s.items[id].msg) {
			n++
		}
	}
	return n
}

func (s *MemoryStore) Latest(f Filter) (*mail.Message, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := len(s.order) - 1; i >= 0; i-- {
		if m := s.items[s.order[i]].msg; f.Match(m) {
			return m, true
		}
	}
	return nil, false
}

func (s *MemoryStore) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{Messages: len(s.order), Bytes: s.bytes, Evicted: s.evicted}
}

func (s *MemoryStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.order)
}

func (s *MemoryStore) Delete(id string) bool {
	s.mu.Lock()
	e, ok := s.items[id]
	flush := func() {}
	if ok {
		s.bytes -= e.size
		delete(s.items, id)
		s.order = slices.DeleteFunc(s.order, func(v string) bool { return v == id })
		flush = s.onDisk(func(d *disk) { d.remove([]string{id}) })
	}
	s.mu.Unlock()
	flush()

	if ok {
		s.publish(events.Event{Type: events.MessageDeleted, ID: id})
	}
	return ok
}

func (s *MemoryStore) DeleteMatching(f Filter) int {
	s.mu.Lock()
	var removed []string
	if f.IsZero() {
		removed = s.order
		s.order = nil
		s.items = make(map[string]entry)
		s.bytes = 0
	} else {
		s.order = slices.DeleteFunc(s.order, func(id string) bool {
			if e := s.items[id]; f.Match(e.msg) {
				removed = append(removed, id)
				s.bytes -= e.size
				delete(s.items, id)
				return true
			}
			return false
		})
	}
	flush := s.onDisk(func(d *disk) { d.remove(removed) })
	s.mu.Unlock()
	flush()

	if f.IsZero() {
		s.publish(events.Event{Type: events.MessagesCleared})
		return len(removed)
	}
	for _, id := range removed {
		s.publish(events.Event{Type: events.MessageDeleted, ID: id})
	}
	return len(removed)
}

func (s *MemoryStore) onDisk(op func(d *disk)) func() {
	if s.disk == nil {
		return func() {}
	}
	d := s.disk
	return d.schedule(func() { op(d) })
}

func (s *MemoryStore) publish(e events.Event) {
	if s.broker != nil {
		s.broker.Publish(e)
	}
}
