package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/mailpeek/mailpeek/internal/events"
	"github.com/mailpeek/mailpeek/internal/mail"
	"github.com/mailpeek/mailpeek/internal/store"
)

type env struct {
	t      *testing.T
	store  *store.MemoryStore
	broker *events.Broker
	srv    *httptest.Server
}

func newEnv(t *testing.T, ui fs.FS) *env {
	t.Helper()
	b := events.NewBroker()
	st := store.New(100, b)
	srv := httptest.NewServer(New(st, b, ui, Info{Version: "test", SMTPPort: 1026}, nil))
	t.Cleanup(func() {
		b.Close()
		srv.Close()
	})
	return &env{t: t, store: st, broker: b, srv: srv}
}

func (e *env) add(raw string, rcpts ...string) *mail.Message {
	data := []byte(strings.ReplaceAll(raw, "\n", "\r\n"))
	m := mail.Parse(data, mail.Envelope{From: "sender@x", To: rcpts})
	e.store.Save(m, data)
	return m
}

func (e *env) do(method, path string) *http.Response {
	e.t.Helper()
	req, _ := http.NewRequest(method, e.srv.URL+path, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	e.t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func (e *env) json(method, path string, wantStatus int, v any) {
	e.t.Helper()
	resp := e.do(method, path)
	if resp.StatusCode != wantStatus {
		body, _ := io.ReadAll(resp.Body)
		e.t.Fatalf("%s %s: status %d, want %d: %s", method, path, resp.StatusCode, wantStatus, body)
	}
	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			e.t.Fatal(err)
		}
	}
}

const welcome = `From: Acme <hello@acme.com>
To: john@example.com
Subject: Welcome to Acme
Content-Type: text/html

<p>Hello John</p><a href="http://localhost:3000/activate/abc">Activate account</a>
`

const invoice = `From: billing@acme.com
To: jane@example.com
Subject: Invoice #123
Content-Type: multipart/mixed; boundary=b

--b
Content-Type: text/plain

See attached.
--b
Content-Type: application/pdf
Content-Disposition: attachment; filename="../../etc/in\"voice.pdf"
Content-Transfer-Encoding: base64

JVBERi0xLjQK
--b--
`

func TestHealthAndInfo(t *testing.T) {
	e := newEnv(t, nil)
	var health map[string]string
	e.json("GET", "/api/v1/health", 200, &health)
	if health["status"] != "ok" {
		t.Fatalf("health = %v", health)
	}
	var info Info
	e.json("GET", "/api/v1/info", 200, &info)
	if info.Version != "test" || info.SMTPPort != 1026 {
		t.Fatalf("info = %+v", info)
	}
}

func TestListAndFilters(t *testing.T) {
	e := newEnv(t, nil)
	e.add(welcome)
	e.add(invoice)

	var list struct {
		Messages []map[string]any `json:"messages"`
		Count    int              `json:"count"`
	}
	e.json("GET", "/api/v1/messages", 200, &list)
	if list.Count != 2 || list.Messages[0]["subject"] != "Invoice #123" {
		t.Fatalf("list = %+v", list)
	}
	if _, hasHTML := list.Messages[0]["html"]; hasHTML {
		t.Fatal("listing should not include bodies")
	}
	if list.Messages[0]["attachments"] != float64(1) {
		t.Fatalf("attachment count = %v", list.Messages[0]["attachments"])
	}

	cases := map[string]int{
		"?to=john@example.com":                 1,
		"?to=JOHN@EXAMPLE.COM&subject=welcome": 1,
		"?to=john@example.com&subject=invoice": 0,
		"?from=billing@":                       1,
		"?subject=acme":                        1,
		"?q=acme":                              2,
		"?since=" + fmt.Sprint(time.Now().Add(time.Hour).UnixMilli()): 0,
		"?since=2000-01-01T00:00:00Z":                                 2,
	}
	for query, want := range cases {
		e.json("GET", "/api/v1/messages"+query, 200, &list)
		if list.Count != want {
			t.Errorf("%s: count %d, want %d", query, list.Count, want)
		}
	}
	e.json("GET", "/api/v1/messages?since=yesterday", 400, nil)
}

func TestListPages(t *testing.T) {
	e := newEnv(t, nil)
	for i := range 5 {
		e.add(fmt.Sprintf("From: a@x\nTo: b@x\nSubject: m%d\n\nbody\n", i))
	}
	type page struct {
		Messages []struct {
			Subject string `json:"subject"`
		} `json:"messages"`
		Count      int    `json:"count"`
		NextCursor string `json:"nextCursor"`
	}
	var got []string
	path := "/api/v1/messages?limit=2"
	for range 5 {
		var p page
		e.json("GET", path, 200, &p)
		for _, m := range p.Messages {
			got = append(got, m.Subject)
		}
		if p.NextCursor == "" {
			break
		}
		path = "/api/v1/messages?limit=2&cursor=" + p.NextCursor
	}
	if fmt.Sprint(got) != "[m4 m3 m2 m1 m0]" {
		t.Fatalf("pages = %v", got)
	}

	for _, bad := range []string{"limit=0", "limit=x", "cursor=0", "cursor=abc"} {
		e.json("GET", "/api/v1/messages?"+bad, 400, nil)
	}
}

func TestCountMessages(t *testing.T) {
	e := newEnv(t, nil)
	e.add(welcome)
	e.add(invoice)
	var res map[string]int
	e.json("GET", "/api/v1/messages/count", 200, &res)
	if res["count"] != 2 {
		t.Fatalf("count = %v", res)
	}
	e.json("GET", "/api/v1/messages/count?q=invoice", 200, &res)
	if res["count"] != 1 {
		t.Fatalf("filtered count = %v", res)
	}
	e.json("GET", "/api/v1/messages/count?since=yesterday", 400, nil)
}

func TestGetLatestAndDelete(t *testing.T) {
	e := newEnv(t, nil)
	w := e.add(welcome)
	e.add(invoice)

	var m mail.Message
	e.json("GET", "/api/v1/messages/"+w.ID, 200, &m)
	if m.Subject != "Welcome to Acme" || !strings.Contains(m.HTML, "Hello John") {
		t.Fatalf("message = %+v", m)
	}
	if len(m.Links) != 1 || m.Links[0].Text != "Activate account" {
		t.Fatalf("links = %+v", m.Links)
	}
	e.json("GET", "/api/v1/messages/latest", 200, &m)
	if m.Subject != "Invoice #123" {
		t.Fatalf("latest = %q", m.Subject)
	}
	e.json("GET", "/api/v1/messages/latest?to=john", 200, &m)
	if m.ID != w.ID {
		t.Fatalf("filtered latest = %q", m.Subject)
	}
	e.json("GET", "/api/v1/messages/latest?to=nobody", 404, nil)
	e.json("GET", "/api/v1/messages/missing", 404, nil)

	if resp := e.do("DELETE", "/api/v1/messages/"+w.ID); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status %d", resp.StatusCode)
	}
	if resp := e.do("DELETE", "/api/v1/messages/"+w.ID); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete status %d", resp.StatusCode)
	}
}

func TestDeleteMessagesWithFilter(t *testing.T) {
	e := newEnv(t, nil)
	e.add(welcome)
	e.add(invoice)
	var res map[string]int
	e.json("DELETE", "/api/v1/messages?to=john@example.com", 200, &res)
	if res["deleted"] != 1 || e.store.Len() != 1 {
		t.Fatalf("filtered delete: %v, len %d", res, e.store.Len())
	}
	e.json("DELETE", "/api/v1/messages", 200, &res)
	if res["deleted"] != 1 || e.store.Len() != 0 {
		t.Fatalf("clear: %v, len %d", res, e.store.Len())
	}
}

func TestRawAndAttachment(t *testing.T) {
	e := newEnv(t, nil)
	m := e.add(invoice)

	resp := e.do("GET", "/api/v1/messages/"+m.ID+"/raw")
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(string(body), "From: billing@acme.com\r\n") {
		t.Fatalf("raw = %d %q", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("raw content type %q", ct)
	}
	resp = e.do("GET", "/api/v1/messages/"+m.ID+"/raw?download=1")
	if cd := resp.Header.Get("Content-Disposition"); cd != `attachment; filename=`+m.ID+`.eml` {
		t.Fatalf("raw disposition %q", cd)
	}

	resp = e.do("GET", "/api/v1/messages/"+m.ID+"/attachments/1")
	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "%PDF-1.4\n" {
		t.Fatalf("attachment = %d %q", resp.StatusCode, body)
	}
	h := resp.Header
	if h.Get("Content-Type") != "application/pdf" || h.Get("Content-Security-Policy") != "sandbox" ||
		h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("attachment headers = %v", h)
	}
	if cd := h.Get("Content-Disposition"); cd != `attachment; filename=in_voice.pdf` {
		t.Fatalf("disposition = %q", cd)
	}
	for _, p := range []string{"/attachments/2", "/attachments/..%2F..%2Fetc%2Fpasswd", "/attachments/0"} {
		if resp := e.do("GET", "/api/v1/messages/"+m.ID+p); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status %d", p, resp.StatusCode)
		}
	}
	if resp := e.do("GET", "/api/v1/messages/nope/attachments/1"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing message: status %d", resp.StatusCode)
	}
}

func TestWaitReturnsExistingMatch(t *testing.T) {
	e := newEnv(t, nil)
	e.add(welcome)
	var m mail.Message
	e.json("GET", "/api/v1/messages/wait?to=john@example.com&subject=Welcome&timeout=1000", 200, &m)
	if m.Subject != "Welcome to Acme" {
		t.Fatalf("subject = %q", m.Subject)
	}
}

func TestWaitBlocksUntilMessageArrives(t *testing.T) {
	e := newEnv(t, nil)
	e.add(invoice)
	go func() {
		for e.broker.Subscribers() == 0 {
			time.Sleep(time.Millisecond)
		}
		time.Sleep(20 * time.Millisecond)
		e.add(strings.Replace(welcome, "john@", "other@", 1))
		e.add(welcome)
	}()
	start := time.Now()
	var m mail.Message
	e.json("GET", "/api/v1/messages/wait?to=john@example.com&timeout=5000", 200, &m)
	if m.Subject != "Welcome to Acme" || m.To[0].Address != "john@example.com" {
		t.Fatalf("got %+v", m)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("wait took too long")
	}
}

func TestWaitSinceIgnoresOlderMessages(t *testing.T) {
	e := newEnv(t, nil)
	e.add(welcome)
	since := time.Now().Add(time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	resp := e.do("GET", "/api/v1/messages/wait?to=john&timeout=50&since="+since.Format(time.RFC3339Nano))
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestWaitTimeoutAndValidation(t *testing.T) {
	e := newEnv(t, nil)
	start := time.Now()
	resp := e.do("GET", "/api/v1/messages/wait?to=nobody&timeout=100")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if d := time.Since(start); d < 100*time.Millisecond || d > 2*time.Second {
		t.Fatalf("timeout took %v", d)
	}
	e.json("GET", "/api/v1/messages/wait?timeout=abc", 400, nil)
	e.json("GET", "/api/v1/messages/wait?timeout=-1", 400, nil)
	if e.broker.Subscribers() != 0 {
		t.Fatalf("leaked %d subscribers", e.broker.Subscribers())
	}
}

func TestParseTimeoutClamps(t *testing.T) {
	cases := map[string]time.Duration{"": DefaultWaitTimeout, "0": 0, "500": 500 * time.Millisecond, "999999": MaxWaitTimeout}
	for in, want := range cases {
		if got, err := parseTimeout(in); err != nil || got != want {
			t.Errorf("parseTimeout(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
}

func TestWaitEndsWhenBrokerCloses(t *testing.T) {
	e := newEnv(t, nil)
	go func() {
		for e.broker.Subscribers() == 0 {
			time.Sleep(time.Millisecond)
		}
		e.broker.Close()
	}()
	e.json("GET", "/api/v1/messages/wait?timeout=10000", 503, nil)
}

func TestSSE(t *testing.T) {
	e := newEnv(t, nil)
	resp := e.do("GET", "/api/v1/events")
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	lines := make(chan string, 32)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	next := func(prefix string) string {
		t.Helper()
		timeout := time.After(2 * time.Second)
		for {
			select {
			case l, ok := <-lines:
				if !ok {
					t.Fatal("stream closed")
				}
				if strings.HasPrefix(l, prefix) {
					return l
				}
			case <-timeout:
				t.Fatalf("no line with prefix %q", prefix)
			}
		}
	}
	next(": connected")

	m := e.add(welcome)
	if l := next("event:"); l != "event: message.created" {
		t.Fatalf("got %q", l)
	}
	if l := next("data:"); l != fmt.Sprintf(`data: {"id":"%s"}`, m.ID) {
		t.Fatalf("got %q", l)
	}
	e.store.Delete(m.ID)
	if l := next("event:"); l != "event: message.deleted" {
		t.Fatalf("got %q", l)
	}
	e.store.DeleteMatching(store.Filter{})
	if l := next("event:"); l != "event: messages.cleared" {
		t.Fatalf("got %q", l)
	}
	if l := next("data:"); l != "data: {}" {
		t.Fatalf("got %q", l)
	}
}

func TestUI(t *testing.T) {
	ui := fstest.MapFS{
		"index.html":    {Data: []byte("<html>app</html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	e := newEnv(t, ui)

	resp := e.do("GET", "/")
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "<html>app</html>" || !strings.Contains(resp.Header.Get("Content-Security-Policy"), "script-src 'self'") {
		t.Fatalf("index = %q %v", body, resp.Header)
	}
	resp = e.do("GET", "/assets/app.js")
	if !strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("asset cache = %q", resp.Header.Get("Cache-Control"))
	}
	resp = e.do("GET", "/some/client/route")
	body, _ = io.ReadAll(resp.Body)
	if string(body) != "<html>app</html>" {
		t.Fatalf("SPA fallback = %q", body)
	}
	resp = e.do("GET", "/api/v2/nope")
	if resp.StatusCode != http.StatusNotFound || resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("api 404 = %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
}

func TestUINotBuilt(t *testing.T) {
	e := newEnv(t, nil)
	if resp := e.do("GET", "/"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestSafeFilename(t *testing.T) {
	cases := map[string]string{
		"invoice.pdf":            "invoice.pdf",
		"../../etc/passwd":       "passwd",
		`..\..\windows\win.ini`:  "win.ini",
		".htaccess":              "htaccess",
		"a\x00b\r\nc.txt":        "a_b__c.txt",
		`we"ird<>|?*:.txt`:       "we_ird______.txt",
		"":                       "attachment",
		"..":                     "attachment",
		"/":                      "attachment",
		"relatório final.pdf":    "relatório final.pdf",
		strings.Repeat("é", 300): strings.Repeat("é", 200),
	}
	for in, want := range cases {
		if got := SafeFilename(in); got != want {
			t.Errorf("SafeFilename(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestUIAuth(t *testing.T) {
	b := events.NewBroker()
	st := store.New(10, b)
	check := func(u, p string) bool { return u == "admin" && p == "secret" }
	srv := httptest.NewServer(New(st, b, fstest.MapFS{"index.html": {Data: []byte("ui")}}, Info{}, check))
	t.Cleanup(func() { b.Close(); srv.Close() })

	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	do := func(method, path string, form url.Values, setup func(*http.Request)) (*http.Response, string) {
		t.Helper()
		var body io.Reader
		if form != nil {
			body = strings.NewReader(form.Encode())
		}
		req, _ := http.NewRequest(method, srv.URL+path, body)
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if setup != nil {
			setup(req)
		}
		resp, err := noRedirect.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return resp, string(data)
	}

	resp, _ := do("GET", "/some/route?x=1", nil, nil)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login?next=%2Fsome%2Froute%3Fx%3D1" {
		t.Fatalf("redirect = %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	for _, path := range []string{"/api/v1/messages", "/api/v1/events", "/api/v1/info"} {
		resp, _ := do("GET", path, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("WWW-Authenticate") != "" {
			t.Errorf("%s: %d %q", path, resp.StatusCode, resp.Header.Get("WWW-Authenticate"))
		}
	}
	for _, open := range []string{"/api/v1/health", "/favicon.svg"} {
		if resp, _ := do("GET", open, nil, nil); resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusSeeOther {
			t.Errorf("%s should not require a login: %d", open, resp.StatusCode)
		}
	}

	resp, page := do("GET", "/login?next=/some/route", nil, nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(page, `name="password"`) || !strings.Contains(page, `value="/some/route"`) {
		t.Fatalf("login page: %d", resp.StatusCode)
	}
	resp, page = do("POST", "/login", url.Values{"username": {"admin"}, "password": {"nope"}, "next": {"/x"}}, nil)
	if resp.StatusCode != http.StatusUnauthorized || !strings.Contains(page, "Wrong username or password") || !strings.Contains(page, `value="admin"`) {
		t.Fatalf("failed login: %d", resp.StatusCode)
	}
	resp, _ = do("POST", "/login", url.Values{"username": {"admin"}, "password": {"secret"}}, func(r *http.Request) {
		r.Header.Set("Origin", "http://evil.test")
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin login: %d", resp.StatusCode)
	}
	resp, _ = do("POST", "/login", url.Values{"username": {"admin"}, "password": {"secret"}, "next": {"/some/route"}}, nil)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/some/route" {
		t.Fatalf("login: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	var session *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookie {
			session = c
		}
	}
	if session == nil || !session.HttpOnly || session.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie = %+v", session)
	}
	withSession := func(r *http.Request) { r.AddCookie(session) }
	if resp, body := do("GET", "/", nil, withSession); resp.StatusCode != http.StatusOK || body != "ui" {
		t.Fatalf("UI with session: %d %q", resp.StatusCode, body)
	}
	if resp, _ := do("GET", "/api/v1/messages", nil, withSession); resp.StatusCode != http.StatusOK {
		t.Fatalf("API with session: %d", resp.StatusCode)
	}
	if resp, _ := do("GET", "/login", nil, withSession); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login page while signed in should redirect: %d", resp.StatusCode)
	}

	basic := func(r *http.Request) { r.SetBasicAuth("admin", "secret") }
	if resp, _ := do("GET", "/api/v1/messages", nil, basic); resp.StatusCode != http.StatusOK {
		t.Fatalf("basic auth: %d", resp.StatusCode)
	}

	resp, _ = do("POST", "/logout", nil, withSession)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("logout: %d", resp.StatusCode)
	}
	if resp, _ := do("GET", "/api/v1/messages", nil, withSession); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("session should be gone: %d", resp.StatusCode)
	}
}

func TestSafeNext(t *testing.T) {
	cases := map[string]string{
		"/":                   "/",
		"/inbox?q=1":          "/inbox?q=1",
		"":                    "/",
		"//evil.test":         "/",
		"/\\evil.test":        "/",
		"https://evil.test":   "/",
		"javascript:alert(1)": "/",
		"/login?next=/x":      "/",
	}
	for in, want := range cases {
		if got := safeNext(in); got != want {
			t.Errorf("safeNext(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestWaitFindsMessageWhoseEventWasLost(t *testing.T) {
	b := events.NewBroker()
	quiet := store.New(10, nil)
	srv := httptest.NewServer(New(quiet, b, nil, Info{}, nil))
	t.Cleanup(func() { b.Close(); srv.Close() })

	go func() {
		time.Sleep(50 * time.Millisecond)
		data := []byte("To: lost@example.com\r\nSubject: Lost\r\n\r\nx\r\n")
		quiet.Save(mail.Parse(data, mail.Envelope{}), data)
	}()
	resp, err := http.Get(srv.URL + "/api/v1/messages/wait?address=lost@example.com&timeout=300")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d; the stored message should be returned", resp.StatusCode)
	}
}

func TestWaitSurvivesEventBurst(t *testing.T) {
	e := newEnv(t, nil)
	done := make(chan int, 1)
	go func() {
		resp, err := http.Get(e.srv.URL + "/api/v1/messages/wait?address=target@example.com&timeout=5000")
		if err != nil {
			done <- 0
			return
		}
		resp.Body.Close()
		done <- resp.StatusCode
	}()
	for e.broker.Subscribers() == 0 {
		time.Sleep(time.Millisecond)
	}
	for i := 0; i < 500; i++ {
		e.add(fmt.Sprintf("To: other%d@example.com\nSubject: noise\n\nx\n", i))
	}
	e.add("To: target@example.com\nSubject: wanted\n\nx\n")
	select {
	case status := <-done:
		if status != http.StatusOK {
			t.Fatalf("status %d", status)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("wait did not return")
	}
}

func TestInfoIncludesStoreStats(t *testing.T) {
	e := newEnv(t, nil)
	e.add(welcome)
	var info struct {
		Store store.Stats `json:"store"`
	}
	e.json("GET", "/api/v1/info", 200, &info)
	if info.Store.Messages != 1 || info.Store.Bytes == 0 {
		t.Fatalf("store stats = %+v", info.Store)
	}
}

func TestLoginTextsComplete(t *testing.T) {
	for lang, text := range loginTexts {
		v := reflect.ValueOf(text)
		for i := range v.NumField() {
			if v.Field(i).String() == "" && v.Type().Field(i).Name != "Lang" {
				t.Errorf("%s: %s is empty", lang, v.Type().Field(i).Name)
			}
		}
	}
}

func TestLoginPageLanguage(t *testing.T) {
	b := events.NewBroker()
	srv := httptest.NewServer(New(store.New(1, b), b, nil, Info{}, func(string, string) bool { return false }))
	t.Cleanup(func() { b.Close(); srv.Close() })
	get := func(setup func(*http.Request)) string {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/login", nil)
		setup(req)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return string(body)
	}
	cases := []struct {
		name  string
		setup func(*http.Request)
		want  string
	}{
		{"default", func(*http.Request) {}, `lang="en-US"`},
		{"accept pt-BR", func(r *http.Request) { r.Header.Set("Accept-Language", "pt-BR,pt;q=0.9") }, "Digite sua senha"},
		{"accept pt-PT", func(r *http.Request) { r.Header.Set("Accept-Language", "pt-PT") }, "Palavra-passe"},
		{"q ordering", func(r *http.Request) { r.Header.Set("Accept-Language", "de;q=0.9, fr-CA;q=0.8, es;q=0.1") }, "Mot de passe"},
		{"en-AU is British", func(r *http.Request) { r.Header.Set("Accept-Language", "en-AU") }, `lang="en-GB"`},
		{"cookie wins", func(r *http.Request) {
			r.Header.Set("Accept-Language", "fr-FR")
			r.AddCookie(&http.Cookie{Name: "mailpeek_lang", Value: "es-ES"})
		}, "Contraseña"},
		{"unknown falls back", func(r *http.Request) { r.Header.Set("Accept-Language", "ja") }, `lang="en-US"`},
	}
	for _, tc := range cases {
		if body := get(tc.setup); !strings.Contains(body, tc.want) {
			t.Errorf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestOpenAPIDescribesEveryRoute(t *testing.T) {
	e := newEnv(t, nil)
	var spec struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	e.json("GET", "/api/v1/openapi.json", 200, &spec)
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		t.Fatalf("openapi = %q", spec.OpenAPI)
	}

	registered := map[string]bool{}
	srv := New(e.store, e.broker, nil, Info{}, nil)
	for _, pattern := range srv.patterns {
		method, path, _ := strings.Cut(pattern, " ")
		if !isAPIPath(path) {
			continue
		}
		op := strings.ToLower(method) + " " + path
		registered[op] = true
		if _, ok := spec.Paths[path][strings.ToLower(method)]; !ok {
			t.Errorf("route %s is missing from openapi.json", pattern)
		}
	}
	for path, ops := range spec.Paths {
		for method := range ops {
			if !registered[method+" "+path] {
				t.Errorf("openapi.json documents %s %s, which is not a route", strings.ToUpper(method), path)
			}
		}
	}
}
