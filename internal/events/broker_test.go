package events

import (
	"sync"
	"testing"
	"time"
)

func receive(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}

func TestPublishSubscribe(t *testing.T) {
	b := NewBroker()
	a, c := b.Subscribe(), b.Subscribe()
	defer a.Close()
	defer c.Close()

	b.Publish(Event{Type: MessageCreated, ID: "1"})
	if e := receive(t, a.C); e.ID != "1" || e.Type != MessageCreated {
		t.Errorf("a got %+v", e)
	}
	if e := receive(t, c.C); e.ID != "1" {
		t.Errorf("c got %+v", e)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	b := NewBroker()
	s := b.Subscribe()
	s.Close()
	s.Close()
	if _, ok := <-s.C; ok {
		t.Fatal("channel should be closed")
	}
	if b.Subscribers() != 0 {
		t.Fatalf("subscribers = %d", b.Subscribers())
	}
	b.Publish(Event{Type: MessageDeleted})
}

func TestSlowSubscriberDoesNotBlockAndCountsMissed(t *testing.T) {
	b := NewBroker()
	s := b.Subscribe()
	defer s.Close()
	done := make(chan struct{})
	go func() {
		for i := 0; i < bufferSize+10; i++ {
			b.Publish(Event{Type: MessageCreated})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked on a slow subscriber")
	}
	if n := s.Missed(); n != 10 {
		t.Fatalf("missed = %d, want 10", n)
	}
	if n := s.Missed(); n != 0 {
		t.Fatalf("Missed should reset, got %d", n)
	}
}

func TestBrokerClose(t *testing.T) {
	b := NewBroker()
	s := b.Subscribe()
	b.Close()
	if _, ok := <-s.C; ok {
		t.Fatal("channel should be closed")
	}
	s.Close()
	if _, ok := <-b.Subscribe().C; ok {
		t.Fatal("subscription after close should be closed")
	}
}

func TestConcurrentUse(t *testing.T) {
	b := NewBroker()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			s := b.Subscribe()
			defer s.Close()
			select {
			case <-s.C:
			case <-time.After(10 * time.Millisecond):
			}
		}()
		go func() {
			defer wg.Done()
			b.Publish(Event{Type: MessageCreated})
		}()
	}
	wg.Wait()
}

func BenchmarkPublish(b *testing.B) {
	br := NewBroker()
	for i := 0; i < 10; i++ {
		s := br.Subscribe()
		defer s.Close()
		go func() {
			for range s.C { //nolint:revive
			}
		}()
	}
	e := Event{Type: MessageCreated, ID: "x"}
	b.ReportAllocs()
	for b.Loop() {
		br.Publish(e)
	}
}
