package failures

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const (
	StageRcpt = "rcpt"
	StageData = "data"
	maxCount  = 10000
	maxRules  = 1000
)

type Rule struct {
	ID        string `json:"id"`
	Stage     string `json:"stage"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Address   string `json:"address,omitempty"`
	Remaining int    `json:"remaining"`
}

type Request struct {
	Stage   string `json:"stage"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Address string `json:"address"`
	Count   int    `json:"count"`
}

type Set struct {
	mu    sync.Mutex
	rules []*Rule
	next  int
}

func (s *Set) Add(req Request) (Rule, error) {
	r := Rule{
		Stage:     strings.ToLower(strings.TrimSpace(req.Stage)),
		Code:      req.Code,
		Message:   strings.TrimSpace(req.Message),
		Address:   strings.TrimSpace(req.Address),
		Remaining: req.Count,
	}
	if r.Stage == "" {
		r.Stage = StageData
	}
	if r.Code == 0 {
		r.Code = 451
	}
	if r.Remaining == 0 {
		r.Remaining = 1
	}
	if err := r.validate(); err != nil {
		return Rule{}, err
	}
	if r.Message == "" {
		r.Message = defaultMessage(r.Code)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.rules) >= maxRules {
		return Rule{}, fmt.Errorf("at most %d failure rules can be pending", maxRules)
	}
	s.next++
	r.ID = strconv.Itoa(s.next)
	s.rules = append(s.rules, &r)
	return r, nil
}

func (r Rule) validate() error {
	var errs []error
	if r.Stage != StageRcpt && r.Stage != StageData {
		errs = append(errs, fmt.Errorf("stage must be %q or %q", StageRcpt, StageData))
	}
	if r.Code < 400 || r.Code > 599 {
		errs = append(errs, errors.New("code must be an SMTP error code between 400 and 599"))
	}
	if strings.ContainsAny(r.Message, "\r\n") {
		errs = append(errs, errors.New("message must be a single line"))
	}
	if r.Remaining < 1 || r.Remaining > maxCount {
		errs = append(errs, fmt.Errorf("count must be between 1 and %d", maxCount))
	}
	return errors.Join(errs...)
}

func defaultMessage(code int) string {
	if code < 500 {
		return "4.3.0 Temporary failure simulated by Mailpeek"
	}
	return "5.3.0 Permanent failure simulated by Mailpeek"
}

func (s *Set) List() []Rule {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Rule, len(s.rules))
	for i, r := range s.rules {
		out[i] = *r
	}
	return out
}

func (s *Set) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.rules {
		if r.ID == id {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			return true
		}
	}
	return false
}

func (s *Set) Clear(address string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.rules[:0]
	for _, r := range s.rules {
		if address != "" && !strings.EqualFold(r.Address, strings.TrimSpace(address)) {
			kept = append(kept, r)
		}
	}
	removed := len(s.rules) - len(kept)
	clear(s.rules[len(kept):])
	s.rules = kept
	return removed
}

func (s *Set) Take(stage string, recipients []string) (Rule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.rules {
		if r.Stage != stage || !r.matches(recipients) {
			continue
		}
		r.Remaining--
		taken := *r
		if r.Remaining == 0 {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
		}
		return taken, true
	}
	return Rule{}, false
}

func (r *Rule) matches(recipients []string) bool {
	if r.Address == "" {
		return true
	}
	for _, rcpt := range recipients {
		if strings.EqualFold(rcpt, r.Address) {
			return true
		}
	}
	return false
}
