package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mailpeek/mailpeek/internal/events"
	"github.com/mailpeek/mailpeek/internal/mail"
)

func msg(id, from, to, subject string) *mail.Message {
	return &mail.Message{
		ID:        id,
		From:      mail.Address{Address: from},
		To:        []mail.Address{{Address: to}},
		Subject:   subject,
		CreatedAt: time.Now(),
	}
}

func ids(list []*mail.Message) []string {
	out := make([]string, len(list))
	for i, m := range list {
		out[i] = m.ID
	}
	return out
}

func TestSaveGetListNewestFirst(t *testing.T) {
	s := New(10, nil)
	s.Save(msg("1", "a@x", "b@x", "one"), []byte("raw1"))
	s.Save(msg("2", "a@x", "b@x", "two"), []byte("raw2"))

	if got := ids(s.List(Filter{})); fmt.Sprint(got) != "[2 1]" {
		t.Fatalf("list = %v", got)
	}
	if m, ok := s.Get("1"); !ok || m.Subject != "one" {
		t.Fatalf("get = %v %v", m, ok)
	}
	if raw, ok := s.Raw("2"); !ok || string(raw) != "raw2" {
		t.Fatalf("raw = %q %v", raw, ok)
	}
	if m, ok := s.Latest(Filter{}); !ok || m.ID != "2" {
		t.Fatalf("latest = %v", m)
	}
	if _, ok := s.Get("missing"); ok {
		t.Fatal("unexpected hit")
	}
}

func TestEvictsOldest(t *testing.T) {
	b := events.NewBroker()
	sub := b.Subscribe()
	defer sub.Close()
	ch := sub.C
	s := New(2, b)
	for i := 1; i <= 3; i++ {
		s.Save(msg(fmt.Sprint(i), "a", "b", "s"), nil)
	}
	if got := ids(s.List(Filter{})); fmt.Sprint(got) != "[3 2]" {
		t.Fatalf("list = %v", got)
	}
	var types []string
	for len(ch) > 0 {
		e := <-ch
		types = append(types, e.Type+":"+e.ID)
	}
	want := "[message.created:1 message.created:2 message.deleted:1 message.created:3]"
	if fmt.Sprint(types) != want {
		t.Fatalf("events = %v", types)
	}
}

func TestDelete(t *testing.T) {
	s := New(10, nil)
	s.Save(msg("1", "a", "b", "s"), nil)
	if !s.Delete("1") || s.Delete("1") || s.Len() != 0 {
		t.Fatal("delete semantics broken")
	}
}

func TestDeleteMatchingAndClear(t *testing.T) {
	b := events.NewBroker()
	sub := b.Subscribe()
	defer sub.Close()
	ch := sub.C
	s := New(10, b)
	s.Save(msg("1", "a", "john@x", "s"), nil)
	s.Save(msg("2", "a", "jane@x", "s"), nil)
	s.Save(msg("3", "a", "john@x", "s"), nil)
	for len(ch) > 0 {
		<-ch
	}

	if n := s.DeleteMatching(Filter{To: "john@"}); n != 2 {
		t.Fatalf("removed %d", n)
	}
	if got := ids(s.List(Filter{})); fmt.Sprint(got) != "[2]" {
		t.Fatalf("list = %v", got)
	}
	if e := <-ch; e.Type != events.MessageDeleted {
		t.Fatalf("event = %+v", e)
	}
	<-ch

	if n := s.DeleteMatching(Filter{}); n != 1 || s.Len() != 0 {
		t.Fatalf("clear removed %d, len %d", n, s.Len())
	}
	if e := <-ch; e.Type != events.MessagesCleared {
		t.Fatalf("event = %+v", e)
	}
}

func TestFilter(t *testing.T) {
	m := &mail.Message{
		From:      mail.Address{Name: "Acme Team", Address: "no-reply@acme.com"},
		To:        []mail.Address{{Address: "john@example.com"}},
		Cc:        []mail.Address{{Name: "Boss", Address: "boss@example.com"}},
		Subject:   "Welcome to Acme",
		Envelope:  mail.Envelope{From: "bounce@acme.com", To: []string{"hidden-bcc@example.com"}},
		CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	cases := []struct {
		f    Filter
		want bool
	}{
		{Filter{}, true},
		{Filter{To: "JOHN@example.com"}, true},
		{Filter{To: "boss"}, true},
		{Filter{To: "hidden-bcc@"}, true},
		{Filter{To: "nobody"}, false},
		{Filter{From: "no-reply@acme.com"}, true},
		{Filter{From: "acme team"}, true},
		{Filter{From: "bounce@"}, true},
		{Filter{From: "john"}, false},
		{Filter{Subject: "welcome"}, true},
		{Filter{Subject: "welcome", To: "john"}, true},
		{Filter{Subject: "welcome", To: "jane"}, false},
		{Filter{Query: "acme"}, true},
		{Filter{Query: "john"}, true},
		{Filter{Query: "invoice"}, false},
		{Filter{Since: m.CreatedAt}, true},
		{Filter{Since: m.CreatedAt.Add(time.Millisecond)}, false},
	}
	for _, tc := range cases {
		if got := tc.f.Match(m); got != tc.want {
			t.Errorf("%+v: got %v want %v", tc.f, got, tc.want)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := New(50, events.NewBroker())
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				id := fmt.Sprintf("%d-%d", w, i)
				s.Save(msg(id, "a", "b", "s"), nil)
				s.List(Filter{To: "b"})
				s.Get(id)
				s.Latest(Filter{})
				if i%10 == 0 {
					s.Delete(id)
				}
				if i%50 == 0 {
					s.DeleteMatching(Filter{Subject: "nothing"})
				}
			}
		}(w)
	}
	wg.Wait()
	if s.Len() > 50 {
		t.Fatalf("len %d exceeds max", s.Len())
	}
}

func filledStore(n int) *MemoryStore {
	s := New(n, nil)
	for i := 0; i < n; i++ {
		s.Save(msg(fmt.Sprint(i), "no-reply@acme.com", fmt.Sprintf("user%d@example.com", i), "Welcome"), nil)
	}
	return s
}

func BenchmarkSave(b *testing.B) {
	s := New(100, events.NewBroker())
	m := msg("x", "a", "b", "s")
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		c := *m
		c.ID = fmt.Sprint(i)
		i++
		s.Save(&c, nil)
	}
}

func BenchmarkList(b *testing.B) {
	s := filledStore(100)
	b.ReportAllocs()
	for b.Loop() {
		s.List(Filter{})
	}
}

func BenchmarkFilter(b *testing.B) {
	s := filledStore(100)
	f := Filter{To: "user42@example.com", Subject: "welcome"}
	b.ReportAllocs()
	for b.Loop() {
		s.List(f)
	}
}

func TestAddressFilterIsExact(t *testing.T) {
	m := &mail.Message{
		To:       []mail.Address{{Name: "Ana", Address: "joana@example.com"}},
		Envelope: mail.Envelope{To: []string{"bcc@example.com"}},
	}
	cases := map[string]bool{
		"joana@example.com": true,
		"JOANA@EXAMPLE.COM": true,
		"ana@example.com":   false,
		"Ana":               false,
		"bcc@example.com":   true,
	}
	for address, want := range cases {
		if got := (Filter{Address: address}).Match(m); got != want {
			t.Errorf("Address %q: got %v want %v", address, got, want)
		}
	}
	if !(Filter{To: "ana@example.com"}).Match(m) {
		t.Error("To keeps substring semantics for search")
	}
}

func TestByteLimitEvictsOldestAndCounts(t *testing.T) {
	s := New(100, nil).WithMaxBytes(250)
	for i := 1; i <= 4; i++ {
		s.Save(msg(fmt.Sprint(i), "a", "b", "s"), make([]byte, 100))
	}
	st := s.Stats()
	if st.Messages != 2 || st.Bytes != 200 || st.Evicted != 2 {
		t.Fatalf("stats = %+v", st)
	}
	if got := ids(s.List(Filter{})); fmt.Sprint(got) != "[4 3]" {
		t.Fatalf("list = %v", got)
	}
	s.Save(msg("big", "a", "b", "s"), make([]byte, 1000))
	if st := s.Stats(); st.Messages != 1 || st.Bytes != 1000 {
		t.Fatalf("stats = %+v", st)
	}
	s.Delete("big")
	if st := s.Stats(); st.Bytes != 0 || st.Evicted != 4 {
		t.Fatalf("after delete: %+v", st)
	}
}

func TestPageFollowsCursor(t *testing.T) {
	s := New(10, nil)
	for i := 1; i <= 5; i++ {
		s.Save(msg(fmt.Sprint(i), "a@x", "b@x", "hello"), nil)
	}
	p := s.Page(Filter{}, 0, 2)
	if got := ids(p.Messages); fmt.Sprint(got) != "[5 4]" || p.Next == 0 {
		t.Fatalf("page 1 = %v next=%d", got, p.Next)
	}
	s.Delete("4")
	s.Save(msg("6", "a@x", "b@x", "hello"), nil)
	p = s.Page(Filter{}, p.Next, 2)
	if got := ids(p.Messages); fmt.Sprint(got) != "[3 2]" || p.Next == 0 {
		t.Fatalf("page 2 = %v next=%d", got, p.Next)
	}
	p = s.Page(Filter{}, p.Next, 2)
	if got := ids(p.Messages); fmt.Sprint(got) != "[1]" || p.Next != 0 {
		t.Fatalf("page 3 = %v next=%d", got, p.Next)
	}
}

func TestPageWithFilterAndExactFit(t *testing.T) {
	s := New(10, nil)
	for i := 1; i <= 6; i++ {
		subject := "other"
		if i%2 == 0 {
			subject = "match"
		}
		s.Save(msg(fmt.Sprint(i), "a@x", "b@x", subject), nil)
	}
	f := Filter{Subject: "match"}
	p := s.Page(f, 0, 2)
	if got := ids(p.Messages); fmt.Sprint(got) != "[6 4]" || p.Next == 0 {
		t.Fatalf("page 1 = %v next=%d", got, p.Next)
	}
	p = s.Page(f, p.Next, 1)
	if got := ids(p.Messages); fmt.Sprint(got) != "[2]" || p.Next != 0 {
		t.Fatalf("page 2 = %v next=%d", got, p.Next)
	}
}

func TestPageCursorBeforeEvictedMessages(t *testing.T) {
	s := New(3, nil)
	for i := 1; i <= 3; i++ {
		s.Save(msg(fmt.Sprint(i), "a@x", "b@x", "hello"), nil)
	}
	p := s.Page(Filter{}, 0, 2)
	s.Save(msg("4", "a@x", "b@x", "hello"), nil)
	s.Save(msg("5", "a@x", "b@x", "hello"), nil)
	if p = s.Page(Filter{}, p.Next, 2); len(p.Messages) != 0 || p.Next != 0 {
		t.Fatalf("after eviction = %v next=%d", ids(p.Messages), p.Next)
	}
}

func TestCount(t *testing.T) {
	s := New(10, nil)
	s.Save(msg("1", "a@x", "b@x", "one"), nil)
	s.Save(msg("2", "a@x", "c@x", "two"), nil)
	if n := s.Count(Filter{}); n != 2 {
		t.Fatalf("count = %d", n)
	}
	if n := s.Count(Filter{To: "c@x"}); n != 1 {
		t.Fatalf("filtered count = %d", n)
	}
}
