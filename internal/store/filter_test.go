package store

import (
	"fmt"
	"strings"
	"testing"
)

func FuzzContains(f *testing.F) {
	for _, c := range [][2]string{
		{"Olá João, confirme a sua conta", "joão"},
		{"CONFIRMAÇÃO DE REGISTO", "confirmação"},
		{"Temperature: 5 K", "5 k"},
		{"İstanbul", "istanbul"},
		{"Straße", "STRASSE"},
		{"ǅemal", "ǆ"},
		{"broken \xff\xfe utf-8", "�"},
		{"aaaaaaaaab", "aab"},
		{"", ""},
		{"abc", ""},
		{"", "a"},
	} {
		f.Add(c[0], c[1])
	}
	f.Fuzz(func(t *testing.T, s, needle string) {
		needle = strings.ToLower(needle)
		want := strings.Contains(strings.ToLower(s), needle)
		if got := contains(s, needle); got != want {
			t.Fatalf("contains(%q, %q) = %v, want %v", s, needle, got, want)
		}
	})
}

func bodyStore(n int) *MemoryStore {
	s := New(n, nil)
	body := strings.Repeat("Olá Ana, obrigado por se registar na Acme. Confirme a sua conta. ", 1500)
	for i := range n {
		m := msg(fmt.Sprint(i), "no-reply@acme.com", fmt.Sprintf("user%d@example.com", i), "Welcome")
		m.Text = body
		m.HTML = "<p>" + body + "</p>"
		s.Save(m, nil)
	}
	return s
}

func BenchmarkSearchBody(b *testing.B) {
	s := bodyStore(100)
	f := Filter{Query: "palavra que não existe"}
	b.ReportAllocs()
	for b.Loop() {
		s.List(f)
	}
}
