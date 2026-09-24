package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andbrslz/mailpeek/internal/mail"
)

func parsed(subject string) (*mail.Message, []byte) {
	raw := []byte("From: app@acme.test\r\nTo: ana@example.com\r\nSubject: " + subject +
		"\r\nContent-Type: text/plain\r\n\r\nHello " + subject + "\r\n")
	return mail.Parse(raw, mail.Envelope{From: "bounce@acme.test", To: []string{"ana@example.com"}}), raw
}

func persisted(t *testing.T, max int, dir string) *MemoryStore {
	t.Helper()
	s := New(max, nil)
	if _, err := s.Persist(dir, func(err error) { t.Errorf("storage error: %v", err) }); err != nil {
		t.Fatal(err)
	}
	return s
}

func files(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestPersistSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	s := persisted(t, 10, dir)
	one, raw1 := parsed("one")
	two, raw2 := parsed("two")
	three, raw3 := parsed("three")
	s.Save(one, raw1)
	s.Save(two, raw2)
	s.Save(three, raw3)
	s.Delete(two.ID)

	again := persisted(t, 10, dir)
	if got := ids(again.List(Filter{})); fmt.Sprint(got) != fmt.Sprint([]string{three.ID, one.ID}) {
		t.Fatalf("reloaded ids = %v", got)
	}
	m, _ := again.Get(one.ID)
	if m.Subject != "one" || !m.CreatedAt.Equal(one.CreatedAt) || m.Envelope.From != "bounce@acme.test" ||
		fmt.Sprint(m.Envelope.To) != "[ana@example.com]" || !strings.Contains(m.Text, "Hello one") {
		t.Fatalf("reloaded message = %+v", m)
	}
	if r, _ := again.Raw(three.ID); string(r) != string(raw3) {
		t.Fatalf("raw = %q", r)
	}
	if n := len(files(t, dir)); n != 4 {
		t.Fatalf("files = %v", files(t, dir))
	}
}

func TestPersistRemovesEvictedAndClearedFiles(t *testing.T) {
	dir := t.TempDir()
	s := persisted(t, 2, dir)
	for _, subject := range []string{"a", "b", "c"} {
		s.Save(parsed(subject))
	}
	if got := files(t, dir); len(got) != 4 {
		t.Fatalf("after eviction files = %v", got)
	}
	s.DeleteMatching(Filter{})
	if got := files(t, dir); len(got) != 0 {
		t.Fatalf("after clear files = %v", got)
	}
}

func TestPersistLoadKeepsNewestWithinLimit(t *testing.T) {
	dir := t.TempDir()
	s := persisted(t, 10, dir)
	var saved []string
	for i := range 5 {
		m, raw := parsed(fmt.Sprint(i))
		s.Save(m, raw)
		saved = append(saved, m.ID)
	}

	small := persisted(t, 2, dir)
	if got := ids(small.List(Filter{})); fmt.Sprint(got) != fmt.Sprint([]string{saved[4], saved[3]}) {
		t.Fatalf("kept = %v, saved = %v", got, saved)
	}
	if st := small.Stats(); st.Evicted != 0 {
		t.Fatalf("stats after load = %+v", st)
	}
	if got := files(t, dir); len(got) != 4 {
		t.Fatalf("files = %v", got)
	}
}

func TestPersistSkipsBrokenMessages(t *testing.T) {
	dir := t.TempDir()
	s := persisted(t, 10, dir)
	good, raw := parsed("good")
	s.Save(good, raw)
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "orphan.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	var errs []error
	again := New(10, nil)
	n, err := again.Persist(dir, func(err error) { errs = append(errs, err) })
	if err != nil || n != 1 {
		t.Fatalf("loaded %d, err %v", n, err)
	}
	if len(errs) != 2 {
		t.Fatalf("errors = %v", errs)
	}
}

func TestPersistRejectsUnusableDir(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(10, nil).Persist(file, func(error) {}); err == nil {
		t.Fatal("expected an error for a data dir that is a file")
	}
}
