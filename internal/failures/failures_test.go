package failures

import (
	"strings"
	"sync"
	"testing"
)

func TestDefaultsAndValidation(t *testing.T) {
	var s Set
	r, err := s.Add(Request{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Stage != StageData || r.Code != 451 || r.Remaining != 1 || !strings.HasPrefix(r.Message, "4.3.0") {
		t.Fatalf("defaults = %+v", r)
	}
	if r, _ := s.Add(Request{Code: 550}); !strings.HasPrefix(r.Message, "5.3.0") {
		t.Fatalf("5xx default message = %q", r.Message)
	}
	for _, bad := range []Request{
		{Stage: "connect"},
		{Code: 250},
		{Code: 600},
		{Message: "line\r\n250 OK"},
		{Count: -1},
		{Count: maxCount + 1},
	} {
		if _, err := s.Add(bad); err == nil {
			t.Errorf("%+v was accepted", bad)
		}
	}
}

func TestTakeConsumesMatchingRulesInOrder(t *testing.T) {
	var s Set
	_, _ = s.Add(Request{Stage: StageRcpt, Address: "Ana@Example.com", Count: 2, Code: 452})
	_, _ = s.Add(Request{Stage: StageData})

	if _, ok := s.Take(StageRcpt, []string{"bia@example.com"}); ok {
		t.Fatal("rule for another address matched")
	}
	for i := 0; i < 2; i++ {
		r, ok := s.Take(StageRcpt, []string{"ana@example.com"})
		if !ok || r.Code != 452 || r.Remaining != 1-i {
			t.Fatalf("take %d = %+v %v", i, r, ok)
		}
	}
	if _, ok := s.Take(StageRcpt, []string{"ana@example.com"}); ok {
		t.Fatal("exhausted rule still matches")
	}
	if _, ok := s.Take(StageData, []string{"anyone@example.com"}); !ok {
		t.Fatal("rule without address should match anyone")
	}
	if n := len(s.List()); n != 0 {
		t.Fatalf("%d rules left", n)
	}
}

func TestClearAndRemove(t *testing.T) {
	var s Set
	a, _ := s.Add(Request{Address: "ana@example.com"})
	_, _ = s.Add(Request{Address: "bia@example.com"})
	_, _ = s.Add(Request{})
	if n := s.Clear("ANA@example.com"); n != 1 {
		t.Fatalf("cleared %d", n)
	}
	if s.Remove(a.ID) {
		t.Fatal("removed a rule that was already cleared")
	}
	if n := s.Clear(""); n != 2 {
		t.Fatalf("cleared %d", n)
	}
}

func TestConcurrentTake(t *testing.T) {
	var s Set
	_, _ = s.Add(Request{Count: 50})
	var wg sync.WaitGroup
	var mu sync.Mutex
	taken := 0
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := s.Take(StageData, nil); ok {
				mu.Lock()
				taken++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if taken != 50 {
		t.Fatalf("taken %d times, want 50", taken)
	}
}
