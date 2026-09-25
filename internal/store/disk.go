package store

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/andbrslz/mailpeek/internal/mail"
)

type meta struct {
	Envelope  mail.Envelope `json:"envelope"`
	CreatedAt time.Time     `json:"createdAt"`
}

type disk struct {
	dir     string
	onError func(error)

	mu      sync.Mutex
	turn    sync.Cond
	next    uint64
	serving uint64
}

func newDisk(dir string, onError func(error)) *disk {
	d := &disk{dir: dir, onError: onError}
	d.turn.L = &d.mu
	return d
}

func (d *disk) schedule(op func()) func() {
	d.mu.Lock()
	ticket := d.next
	d.next++
	d.mu.Unlock()
	return func() {
		d.mu.Lock()
		for d.serving != ticket {
			d.turn.Wait()
		}
		d.mu.Unlock()
		op()
		d.mu.Lock()
		d.serving++
		d.turn.Broadcast()
		d.mu.Unlock()
	}
}

func (d *disk) write(m *mail.Message, raw []byte) {
	data, err := json.Marshal(meta{Envelope: m.Envelope, CreatedAt: m.CreatedAt})
	if err == nil {
		err = writeFile(filepath.Join(d.dir, m.ID+".eml"), raw)
	}
	if err == nil {
		err = writeFile(filepath.Join(d.dir, m.ID+".json"), data)
	}
	if err != nil {
		d.onError(fmt.Errorf("saving message %s: %w", m.ID, err))
	}
}

func (d *disk) remove(ids []string) {
	for _, id := range ids {
		for _, ext := range []string{".json", ".eml"} {
			if err := os.Remove(filepath.Join(d.dir, id+ext)); err != nil && !errors.Is(err, fs.ErrNotExist) {
				d.onError(fmt.Errorf("removing message %s: %w", id, err))
			}
		}
	}
}

func writeFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

type stored struct {
	id   string
	meta meta
	raw  []byte
}

func (s *MemoryStore) Persist(dir string, onError func(error)) (int, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, fmt.Errorf("data dir: %w", err)
	}
	probe := filepath.Join(dir, ".mailpeek-write-test")
	if err := os.WriteFile(probe, nil, 0o600); err != nil {
		return 0, fmt.Errorf("data dir %s is not writable: %w", dir, err)
	}
	_ = os.Remove(probe)

	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return 0, fmt.Errorf("data dir: %w", err)
	}
	var list []stored
	for _, p := range paths {
		e, err := readStored(p)
		if err != nil {
			onError(err)
			continue
		}
		list = append(list, e)
	}
	slices.SortFunc(list, func(a, b stored) int {
		return cmp.Or(a.meta.CreatedAt.Compare(b.meta.CreatedAt), strings.Compare(a.id, b.id))
	})

	removeOrphans(dir, onError)

	s.mu.Lock()
	s.disk = newDisk(dir, onError)
	s.mu.Unlock()
	for _, e := range list {
		m := mail.Parse(e.raw, e.meta.Envelope)
		m.ID = e.id
		m.CreatedAt = e.meta.CreatedAt
		s.save(m, e.raw, false)
	}
	s.mu.Lock()
	s.evicted = 0
	s.mu.Unlock()
	return s.Len(), nil
}

func readStored(path string) (stored, error) {
	id := strings.TrimSuffix(filepath.Base(path), ".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return stored{}, fmt.Errorf("loading message %s: %w", id, err)
	}
	var m meta
	if err := json.Unmarshal(data, &m); err != nil {
		return stored{}, fmt.Errorf("loading message %s: %w", id, err)
	}
	raw, err := os.ReadFile(strings.TrimSuffix(path, ".json") + ".eml")
	if err != nil {
		return stored{}, fmt.Errorf("loading message %s: %w", id, err)
	}
	if m.Envelope.To == nil {
		m.Envelope.To = []string{}
	}
	return stored{id: id, meta: m, raw: raw}, nil
}

var storedName = regexp.MustCompile(`^[0-9a-f]{16}\.(eml|json)(\.tmp)?$`)

func removeOrphans(dir string, onError func(error)) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		onError(fmt.Errorf("data dir: %w", err))
		return
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name()] = true
	}
	for name := range names {
		if !storedName.MatchString(name) {
			continue
		}
		orphan := strings.HasSuffix(name, ".tmp") ||
			strings.HasSuffix(name, ".eml") && !names[strings.TrimSuffix(name, ".eml")+".json"]
		if !orphan {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			onError(fmt.Errorf("removing unfinished file %s: %w", name, err))
		}
	}
}
