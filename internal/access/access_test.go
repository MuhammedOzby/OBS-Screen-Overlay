package access

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"obs-screen-overlay/internal/store"
)

const origin = "http://localhost:8080"

func setup(t *testing.T) (*Manager, *store.Store, http.Handler, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, e := store.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	m, e := New(s, Config{AdminOrigins: []string{origin, "http://192.168.1.2:8080"}})
	if e != nil {
		t.Fatal(e)
	}
	return m, s, m.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })), path
}
func req(h http.Handler, method, path, remote, body string, c *http.Cookie, csrf, orig string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, origin+path, strings.NewReader(body))
	r.RemoteAddr = remote
	if orig != "" {
		r.Header.Set("Origin", orig)
	}
	if csrf != "" {
		r.Header.Set("X-CSRF-Token", csrf)
	}
	r.Header.Set("X-Overlay-Request", "1")
	if c != nil {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func payload(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	return v
}
func bootstrap(t *testing.T, h http.Handler) (*http.Cookie, string) {
	t.Helper()
	w := req(h, "POST", "/access/local", "127.0.0.1:1234", "", nil, "", origin)
	if w.Code != 200 {
		t.Fatalf("bootstrap %d %s", w.Code, w.Body)
	}
	return w.Result().Cookies()[0], payload(t, w)["csrf_token"].(string)
}
func TestGateAndCookies(t *testing.T) {
	_, _, h, _ := setup(t)
	for _, method := range []string{"GET", "POST", "PATCH", "DELETE"} {
		w := req(h, method, "/api/pages", "127.0.0.1:22", "", nil, "", origin)
		if w.Code != 401 {
			t.Fatalf("%s: %d", method, w.Code)
		}
	}
	for _, remote := range []string{"192.0.2.1:100", "bad"} {
		if w := req(h, "POST", "/access/local", remote, "", nil, "", origin); w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
	r := httptest.NewRequest("POST", origin+"/access/local", nil)
	r.RemoteAddr = "192.0.2.1:123"
	r.Header.Set("Origin", origin)
	r.Header.Set("X-Overlay-Request", "1")
	r.Header.Set("X-Forwarded-For", "127.0.0.1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	c, csrf := bootstrap(t, h)
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.Domain != "" || c.Secure || c.MaxAge != int(lifetime.Seconds()) {
		t.Fatalf("cookie: %+v", c)
	}
	for _, test := range []struct {
		csrf, origin string
		want         int
	}{{csrf, origin, 204}, {"wrong", origin, 403}, {csrf, "http://localhost:8081", 403}, {csrf, "", 403}} {
		if w := req(h, "PATCH", "/api/pages/1", "127.0.0.1:2", "", c, test.csrf, test.origin); w.Code != test.want {
			t.Fatalf("got %d want %d", w.Code, test.want)
		}
	}
	if w := req(h, "POST", "/access/logout", "127.0.0.1:2", "", c, csrf, origin); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := req(h, "GET", "/api/pages", "127.0.0.1:2", "", c, "", origin); w.Code != 401 {
		t.Fatal(w.Code)
	}
}
func TestPairingLifecycleAndOrigin(t *testing.T) {
	m, _, h, _ := setup(t)
	c, csrf := bootstrap(t, h)
	create := func(body string) *httptest.ResponseRecorder {
		return req(h, "POST", "/access/pairing", "127.0.0.1:33", body, c, csrf, origin)
	}
	if w := create(`{"origin":"http://evil:80"}`); w.Code != 400 {
		t.Fatal(w.Code)
	}
	w := create(`{}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	p := payload(t, w)
	code := p["code"].(string)
	token := p["qr_token"].(string)
	if len(code) != 6 || !strings.HasPrefix(p["qr_url"].(string), "http://192.168.1.2:8080/access#code=") || !strings.HasPrefix(p["qr_image"].(string), "data:image/png;base64,") {
		t.Fatal("bad QR response")
	}
	pair := func(body string) *httptest.ResponseRecorder {
		return req(h, "POST", "/access/pair", "192.0.2.1:44", body, nil, "", origin)
	}
	old := code
	w = create(`{"origin":"http://localhost:8080"}`)
	p = payload(t, w)
	if !strings.HasPrefix(p["qr_url"].(string), origin+"/access#") {
		t.Fatal("target")
	}
	if pair(`{"qr_token":"`+token+`","device_name":"phone"}`).Code != 403 {
		t.Fatal("old token accepted")
	}
	_ = old
	code = p["code"].(string)
	w = pair(`{"code":"` + code + `","device_name":"phone"}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	if pair(`{"code":"`+code+`","device_name":"phone"}`).Code != 403 {
		t.Fatal("reused")
	}
	w = create(`{}`)
	p = payload(t, w)
	m.mu.Lock()
	m.challenge.expires = time.Now().Add(-time.Second)
	m.mu.Unlock()
	if pair(`{"qr_token":"`+p["qr_token"].(string)+`","device_name":"phone"}`).Code != 403 {
		t.Fatal("expired")
	}
	for i := 0; i < 5; i++ {
		pair(`{"code":"invalid","device_name":"phone"}`)
	}
	if pair(`{"code":"invalid","device_name":"phone"}`).Code != 429 {
		t.Fatal("limit")
	}
}
func TestPersistenceDevicesAndHeaders(t *testing.T) {
	_, s, h, path := setup(t)
	c, csrf := bootstrap(t, h)
	w := req(h, "GET", "/access/devices", "127.0.0.1:3", "", c, "", origin)
	if w.Code != 200 || bytes.Contains(w.Body.Bytes(), []byte(csrf)) || bytes.Contains(w.Body.Bytes(), []byte(c.Value)) {
		t.Fatal("device leak")
	}
	for k, v := range map[string]string{"Cache-Control": "no-store", "X-Frame-Options": "DENY", "Referrer-Policy": "no-referrer", "X-Content-Type-Options": "nosniff"} {
		if w.Header().Get(k) != v {
			t.Fatal(k)
		}
	}
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	s2, e := store.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s2.Close()
	m, e := New(s2, Config{AdminOrigins: []string{origin}})
	if e != nil {
		t.Fatal(e)
	}
	h = m.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	if req(h, "GET", "/api/pages", "127.0.0.1:3", "", c, "", origin).Code != 204 {
		t.Fatal("persistence")
	}
	if req(h, "POST", "/access/devices/revoke", "127.0.0.1:3", `{"all":true}`, c, csrf, origin).Code != 200 {
		t.Fatal("revoke")
	}
	if req(h, "GET", "/api/pages", "127.0.0.1:3", "", c, "", origin).Code != 401 {
		t.Fatal("revoked")
	}
}
func TestInvalidConfig(t *testing.T) {
	s, e := store.Open(":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for _, o := range []string{"http://user@localhost", "http://localhost/", "http://localhost?x=1", "https://localhost:bad", "http://localhost#x"} {
		if _, e := New(s, Config{AdminOrigins: []string{o}}); e == nil {
			t.Errorf("accepted %q", o)
		}
	}
}

func TestHostTLSAndRemoteSession(t *testing.T) {
	_, s, h, _ := setup(t)
	c, _ := bootstrap(t, h)
	r := httptest.NewRequest("GET", "http://evil.example/api/pages", nil)
	r.AddCookie(c)
	r.RemoteAddr = "127.0.0.1:123"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("unknown host", w.Code)
	}
	w = req(h, "GET", "/access/session", "192.0.2.5:12", "", c, "", origin)
	if w.Code != 200 || bytes.Contains(w.Body.Bytes(), []byte("admin_origins")) {
		t.Fatal("remote session config", w.Code)
	}
	m, e := New(s, Config{AdminOrigins: []string{"https://localhost:8443"}})
	if e != nil {
		t.Fatal(e)
	}
	r = httptest.NewRequest("POST", "https://localhost:8443/access/local", nil)
	r.RemoteAddr = "127.0.0.1:2"
	r.TLS = &tls.ConnectionState{}
	r.Header.Set("Origin", "https://localhost:8443")
	r.Header.Set("X-Overlay-Request", "1")
	w = httptest.NewRecorder()
	m.Wrap(http.NotFoundHandler()).ServeHTTP(w, r)
	if w.Code != 200 || !w.Result().Cookies()[0].Secure {
		t.Fatal("secure cookie", w.Code)
	}
}

func TestPairConcurrentSingleUse(t *testing.T) {
	_, _, h, _ := setup(t)
	c, csrf := bootstrap(t, h)
	w := req(h, "POST", "/access/pairing", "127.0.0.1:1", `{}`, c, csrf, origin)
	token := payload(t, w)["qr_token"].(string)
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- req(h, "POST", "/access/pair", "192.0.2.10:1", `{"qr_token":"`+token+`","device_name":"phone"}`, nil, "", origin).Code
		}()
	}
	wg.Wait()
	close(statuses)
	success := 0
	for status := range statuses {
		if status == 200 {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("successes %d", success)
	}
}

func TestPairStoreFailurePreservesChallenge(t *testing.T) {
	m, s, h, path := setup(t)
	c, csrf := bootstrap(t, h)
	w := req(h, "POST", "/access/pairing", "127.0.0.1:1", `{}`, c, csrf, origin)
	token := payload(t, w)["qr_token"].(string)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	body := `{"qr_token":"` + token + `","device_name":"   "}`
	w = req(h, "POST", "/access/pair", "192.0.2.10:1", body, nil, "", origin)
	if w.Code != 500 || len(w.Result().Cookies()) != 0 {
		t.Fatalf("failed insert: %d cookies=%v", w.Code, w.Result().Cookies())
	}
	if m.challenge.token != token {
		t.Fatal("challenge consumed on store failure")
	}
	reopened, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	sessions, err := reopened.ListAccessSessions(t.Context())
	if err != nil || len(sessions) != 1 {
		t.Fatalf("unexpected sessions: %v, %v", sessions, err)
	}
	m.store = reopened
	w = req(h, "POST", "/access/pair", "192.0.2.10:1", body, nil, "", origin)
	if w.Code != 200 {
		t.Fatalf("retry: %d %s", w.Code, w.Body)
	}
	sessions, err = reopened.ListAccessSessions(t.Context())
	if err != nil || len(sessions) != 2 {
		t.Fatalf("sessions after retry: %v, %v", sessions, err)
	}
	found := false
	for _, session := range sessions {
		if session.DeviceName == "Eşleşmiş cihaz" {
			found = true
		}
	}
	if !found {
		t.Fatal("default device name missing")
	}
}

func TestDevicesResponseObject(t *testing.T) {
	_, _, h, _ := setup(t)
	c, _ := bootstrap(t, h)
	w := req(h, "GET", "/access/devices", "127.0.0.1:1", "", c, "", origin)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var body struct {
		Devices []struct {
			ID         string `json:"id"`
			DeviceName string `json:"device_name"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || len(body.Devices) != 1 || body.Devices[0].ID == "" || body.Devices[0].DeviceName == "" {
		t.Fatalf("invalid response %s: %v", w.Body, err)
	}
}

func TestPairingLocalOnlyAndNameLimit(t *testing.T) {
	_, _, h, _ := setup(t)
	c, csrf := bootstrap(t, h)
	if w := req(h, "POST", "/access/pairing", "192.0.2.1:1", `{}`, c, csrf, origin); w.Code != 403 {
		t.Fatalf("remote pairing creation: %d", w.Code)
	}
	w := req(h, "POST", "/access/pairing", "127.0.0.1:1", `{}`, c, csrf, origin)
	token := payload(t, w)["qr_token"].(string)
	body := `{"qr_token":"` + token + `","device_name":"` + strings.Repeat("x", 129) + `"}`
	if w = req(h, "POST", "/access/pair", "192.0.2.1:1", body, nil, "", origin); w.Code != 403 {
		t.Fatalf("oversized name: %d", w.Code)
	}
	body = `{"qr_token":"` + token + `"}`
	if w = req(h, "POST", "/access/pair", "192.0.2.1:1", body, nil, "", origin); w.Code != 200 {
		t.Fatalf("optional name: %d", w.Code)
	}
}

func TestRevokeAllInvalidatesPendingAndPaired(t *testing.T) {
	_, _, h, _ := setup(t)
	c, csrf := bootstrap(t, h)
	newToken := func() string {
		w := req(h, "POST", "/access/pairing", "127.0.0.1:1", `{}`, c, csrf, origin)
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
		return payload(t, w)["qr_token"].(string)
	}
	token := newToken()
	w := req(h, "POST", "/access/pair", "192.0.2.1:1", `{"qr_token":"`+token+`"}`, nil, "", origin)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	paired := w.Result().Cookies()[0]
	token = newToken()
	// Revoking a single unrelated device must not invalidate a challenge.
	w = req(h, "POST", "/access/devices/revoke", "127.0.0.1:1", `{"id":"nonexistent"}`, c, csrf, origin)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	w = req(h, "POST", "/access/devices/revoke", "127.0.0.1:1", `{"all":true}`, c, csrf, origin)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = req(h, "GET", "/access/session", "192.0.2.1:1", "", paired, "", origin); w.Code != 401 {
		t.Fatalf("paired session: %d", w.Code)
	}
	if w = req(h, "POST", "/access/pair", "192.0.2.1:1", `{"qr_token":"`+token+`"}`, nil, "", origin); w.Code != 403 {
		t.Fatalf("pending challenge: %d", w.Code)
	}
}

func TestSingleRevokePreservesChallenge(t *testing.T) {
	_, _, h, _ := setup(t)
	c, csrf := bootstrap(t, h)
	w := req(h, "POST", "/access/pairing", "127.0.0.1:1", `{}`, c, csrf, origin)
	token := payload(t, w)["qr_token"].(string)
	if w = req(h, "POST", "/access/devices/revoke", "127.0.0.1:1", `{"id":"nonexistent"}`, c, csrf, origin); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = req(h, "POST", "/access/pair", "192.0.2.1:1", `{"qr_token":"`+token+`"}`, nil, "", origin); w.Code != 200 {
		t.Fatalf("challenge lost: %d", w.Code)
	}
}

func TestRevokeAllStoreFailurePreservesChallenge(t *testing.T) {
	m, s, h, _ := setup(t)
	c, csrf := bootstrap(t, h)
	w := req(h, "POST", "/access/pairing", "127.0.0.1:1", `{}`, c, csrf, origin)
	token := payload(t, w)["qr_token"].(string)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", origin+"/access/devices/revoke", strings.NewReader(`{"all":true}`))
	w = httptest.NewRecorder()
	m.revoke(w, r, store.AccessSession{})
	if w.Code != 500 || m.challenge.token != token || len(w.Result().Cookies()) != 0 {
		t.Fatalf("failed revoke: %d", w.Code)
	}
}

func TestPairAndRevokeAllSerialize(t *testing.T) {
	m, s, h, _ := setup(t)
	c, csrf := bootstrap(t, h)
	w := req(h, "POST", "/access/pairing", "127.0.0.1:1", `{}`, c, csrf, origin)
	token := payload(t, w)["qr_token"].(string)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		req(h, "POST", "/access/pair", "192.0.2.1:1", `{"qr_token":"`+token+`"}`, nil, "", origin)
	}()
	go func() {
		defer wg.Done()
		r := httptest.NewRequest("POST", origin+"/access/devices/revoke", strings.NewReader(`{"all":true}`))
		m.revoke(httptest.NewRecorder(), r, store.AccessSession{})
	}()
	wg.Wait()
	sessions, err := s.ListAccessSessions(t.Context())
	if err != nil || len(sessions) != 0 || m.challenge.token != "" {
		t.Fatalf("sessions=%v challenge=%v err=%v", len(sessions), m.challenge.token != "", err)
	}
}

func TestChallengeCreationRejectsRevokedSession(t *testing.T) {
	m, s, h, _ := setup(t)
	c, _ := bootstrap(t, h)
	session, err := s.GetAccessSession(t.Context(), hash(c.Value))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteAllAccessSessions(t.Context()); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", origin+"/access/pairing", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	m.newChallenge(w, r, session)
	if w.Code != 401 || m.challenge.token != "" {
		t.Fatalf("stale session created challenge: %d", w.Code)
	}
}

func TestGlobalPairLimitAndMapBound(t *testing.T) {
	m, _, h, _ := setup(t)
	for i := 1; i <= 50; i++ {
		w := req(h, "POST", "/access/pair", fmt.Sprintf("192.0.2.%d:1", i), `{"code":"wrong"}`, nil, "", origin)
		if w.Code != 403 {
			t.Fatalf("attempt %d: %d", i, w.Code)
		}
	}
	if w := req(h, "POST", "/access/pair", "192.0.2.51:1", `{"code":"wrong"}`, nil, "", origin); w.Code != 429 {
		t.Fatalf("global limit: %d", w.Code)
	}
	// The global limit normally prevents 1024 distinct entries; fill it directly
	// to exercise the independent defensive bound without changing the limiter.
	m.mu.Lock()
	m.global = attempts{}
	m.ips = make(map[string]attempts)
	for i := 0; i < 1024; i++ {
		m.ips[fmt.Sprintf("198.18.%d.%d", i/256, i%256)] = attempts{count: 1, until: time.Now().Add(time.Minute)}
	}
	m.mu.Unlock()
	if w := req(h, "POST", "/access/pair", "203.0.113.1:1", `{"code":"wrong"}`, nil, "", origin); w.Code != 429 {
		t.Fatalf("map bound: %d", w.Code)
	}
	if len(m.ips) != 1024 {
		t.Fatalf("map size: %d", len(m.ips))
	}
}

func TestMissingRequestHeaderAndCrossSite(t *testing.T) {
	_, _, h, _ := setup(t)
	for _, tc := range []struct{ method, path, overlay, fetch string }{
		{"POST", "/access/local", "", ""}, {"POST", "/access/pair", "", ""},
		{"POST", "/access/local", "1", "cross-site"}, {"POST", "/access/pair", "1", "cross-site"}, {"GET", "/access", "", "cross-site"},
	} {
		r := httptest.NewRequest(tc.method, origin+tc.path, strings.NewReader(`{}`))
		r.RemoteAddr = "127.0.0.1:1"
		r.Header.Set("Origin", origin)
		if tc.overlay != "" {
			r.Header.Set("X-Overlay-Request", tc.overlay)
		}
		if tc.fetch != "" {
			r.Header.Set("Sec-Fetch-Site", tc.fetch)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Errorf("%s %s overlay=%q fetch=%q: %d", tc.method, tc.path, tc.overlay, tc.fetch, w.Code)
		}
	}
}
