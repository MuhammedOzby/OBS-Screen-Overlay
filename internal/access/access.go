// Package access implements the admin session and pairing gate.
package access

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	"obs-screen-overlay/internal/store"
)

const cookieName = "overlay_session"
const lifetime = 180 * 24 * time.Hour
const challengeTTL = 5 * time.Minute

type Config struct{ AdminOrigins []string }
type challenge struct {
	code, token string
	expires     time.Time
}
type attempts struct {
	count int
	until time.Time
}
type Manager struct {
	store      *store.Store
	origins    map[string]bool
	originList []string
	mu         sync.Mutex
	challenge  challenge
	ips        map[string]attempts
	global     attempts
	now        func() time.Time
}

// New validates the complete set of trusted admin origins. Reverse proxy headers
// are deliberately ignored; TLS must terminate at this server.
func New(s *store.Store, cfg Config) (*Manager, error) {
	if s == nil || len(cfg.AdminOrigins) == 0 {
		return nil, errors.New("store and admin origins required")
	}
	m := &Manager{store: s, origins: map[string]bool{}, ips: map[string]attempts{}, now: time.Now}
	for _, origin := range cfg.AdminOrigins {
		u, err := url.Parse(origin)
		if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.String() != origin || strings.ToLower(u.Host) != u.Host {
			return nil, errors.New("invalid admin origin")
		}
		if u.Hostname() == "" || strings.ContainsAny(u.Host, "% \\ ") {
			return nil, errors.New("invalid admin host")
		}
		if p := u.Port(); p != "" {
			n, err := net.LookupPort("tcp", p)
			if err != nil || n < 1 || n > 65535 || fmt.Sprint(n) != p {
				return nil, errors.New("invalid admin port")
			}
		} else if strings.HasSuffix(u.Host, ":") {
			return nil, errors.New("invalid admin port")
		}
		if !m.origins[origin] {
			m.originList = append(m.originList, origin)
		}
		m.origins[origin] = true
	}
	return m, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func hash(s string) string   { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func equal(a, b string) bool { return a != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
func local(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || !ip.Unmap().IsLoopback() {
		return false
	}
	h := r.Host
	if strings.HasPrefix(h, "[") || strings.Contains(h, ":") {
		h, _, err = net.SplitHostPort(h)
		if err != nil {
			if strings.HasPrefix(r.Host, "[") && strings.HasSuffix(r.Host, "]") {
				h = strings.Trim(r.Host, "[]")
			} else {
				return false
			}
		}
	}
	if h == "localhost" {
		return true
	}
	hip, err := netip.ParseAddr(h)
	return err == nil && hip.Unmap().IsLoopback()
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int) {
	writeJSON(w, status, map[string]string{"error": http.StatusText(status)})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return false
	}
	var extra any
	return d.Decode(&extra) == io.EOF
}
func mutation(method string) bool { return method != "GET" && method != "HEAD" && method != "OPTIONS" }

// Wrap protects every admin route and serves the /access API. GET /access and
// public assets are passed to next; integration must not expose secrets there.
func (m *Manager) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		origin := requestOrigin(r)
		if !m.origins[origin] || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			fail(w, http.StatusForbidden)
			return
		}
		supplied := r.Header.Values("Origin")
		if len(supplied) > 1 || (len(supplied) == 1 && supplied[0] != origin) {
			fail(w, http.StatusForbidden)
			return
		}
		if r.Method == "OPTIONS" {
			fail(w, http.StatusForbidden)
			return
		}
		if mutation(r.Method) && (len(supplied) != 1 || supplied[0] != origin) {
			fail(w, http.StatusForbidden)
			return
		}
		if r.Method == "GET" && (r.URL.Path == "/access" || strings.HasPrefix(r.URL.Path, "/assets/")) {
			next.ServeHTTP(w, r)
			return
		}
		if (r.URL.Path == "/access/local" || r.URL.Path == "/access/pair") && r.Method == "POST" {
			if r.Header.Get("X-Overlay-Request") != "1" {
				fail(w, http.StatusForbidden)
				return
			}
			if r.URL.Path == "/access/local" {
				if !local(r) {
					fail(w, http.StatusForbidden)
					return
				}
				m.issue(w, r, "Local browser")
				return
			}
			m.pair(w, r)
			return
		}
		var session store.AccessSession
		c, err := r.Cookie(cookieName)
		if err == nil && len(c.Value) == 43 {
			session, err = m.store.GetAccessSession(r.Context(), hash(c.Value))
		}
		if err != nil || session.TokenHash == "" || !session.ExpiresAt.After(m.now()) {
			if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/access/") {
				fail(w, http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/access", http.StatusSeeOther)
			}
			return
		}
		if mutation(r.Method) && !equal(r.Header.Get("X-CSRF-Token"), session.CSRFToken) {
			fail(w, http.StatusForbidden)
			return
		}
		switch r.URL.Path {
		case "/access/session":
			if r.Method != "GET" {
				fail(w, 405)
				return
			}
			info := map[string]any{"csrf_token": session.CSRFToken, "local": local(r), "expires_at": session.ExpiresAt}
			if local(r) {
				info["admin_origins"] = m.originList
			}
			writeJSON(w, 200, info)
		case "/access/pairing":
			if r.Method != "POST" {
				fail(w, 405)
				return
			}
			if !local(r) {
				fail(w, 403)
				return
			}
			m.newChallenge(w, r, session)
		case "/access/logout":
			if r.Method != "POST" {
				fail(w, 405)
				return
			}
			if m.store.DeleteAccessSession(r.Context(), session.TokenHash) != nil {
				fail(w, 500)
				return
			}
			m.clearCookie(w, r)
			writeJSON(w, 200, map[string]bool{"ok": true})
		case "/access/devices":
			if r.Method != "GET" {
				fail(w, 405)
				return
			}
			if !local(r) {
				fail(w, 403)
				return
			}
			m.devices(w, r)
		case "/access/devices/revoke":
			if r.Method != "POST" {
				fail(w, 405)
				return
			}
			if !local(r) {
				fail(w, 403)
				return
			}
			m.revoke(w, r, session)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func (m *Manager) setCookie(w http.ResponseWriter, r *http.Request, token string, age int) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: age})
}
func (m *Manager) clearCookie(w http.ResponseWriter, r *http.Request) { m.setCookie(w, r, "", -1) }
func (m *Manager) create(r *http.Request, name string) (string, string, time.Time, error) {
	token, err := randomToken()
	if err != nil {
		return "", "", time.Time{}, err
	}
	csrf, err := randomToken()
	if err != nil {
		return "", "", time.Time{}, err
	}
	expires := m.now().Add(lifetime).UTC().Truncate(time.Second)
	err = m.store.CreateAccessSession(r.Context(), hash(token), csrf, name, expires)
	return token, csrf, expires, err
}
func (m *Manager) issue(w http.ResponseWriter, r *http.Request, name string) {
	token, csrf, expires, err := m.create(r, name)
	if err != nil {
		fail(w, 500)
		return
	}
	m.setCookie(w, r, token, int(lifetime.Seconds()))
	writeJSON(w, 200, map[string]any{"csrf_token": csrf, "local": local(r), "expires_at": expires})
}
func (m *Manager) newChallenge(w http.ResponseWriter, r *http.Request, session store.AccessSession) {
	var body struct {
		Origin string `json:"origin"`
	}
	if r.ContentLength != 0 && !decode(w, r, &body) {
		fail(w, 400)
		return
	}
	target := body.Origin
	if target == "" {
		target = m.originList[0]
		for _, o := range m.originList {
			u, _ := url.Parse(o)
			h := u.Hostname()
			ip, e := netip.ParseAddr(h)
			if h != "localhost" && (e != nil || !ip.Unmap().IsLoopback()) {
				target = o
				break
			}
		}
	}
	if !m.origins[target] {
		fail(w, 400)
		return
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		fail(w, 500)
		return
	}
	token, err := randomToken()
	if err != nil {
		fail(w, 500)
		return
	}
	code := fmt.Sprintf("%06d", n.Int64())
	qrURL := target + "/access#code=" + code + "&qr_token=" + token
	png, err := qrcode.Encode(qrURL, qrcode.Medium, 256)
	if err != nil {
		fail(w, 500)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// The request may have authenticated before a concurrent revoke-all.
	if _, err := m.store.GetAccessSession(r.Context(), session.TokenHash); err != nil {
		fail(w, http.StatusUnauthorized)
		return
	}
	m.challenge = challenge{code, token, m.now().Add(challengeTTL).UTC()}
	writeJSON(w, 200, map[string]any{"code": code, "qr_token": token, "expires_at": m.challenge.expires, "qr_url": qrURL, "qr_image": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)})
}
func (m *Manager) pair(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code       string `json:"code"`
		QRToken    string `json:"qr_token"`
		DeviceName string `json:"device_name"`
	}
	ok := decode(w, r, &body)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		fail(w, 403)
		return
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		fail(w, 403)
		return
	}
	ip = addr.Unmap().String()
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if !now.Before(m.global.until) {
		m.global = attempts{until: now.Add(challengeTTL)}
	}
	for k, v := range m.ips {
		if !now.Before(v.until) {
			delete(m.ips, k)
		}
	}
	a, exists := m.ips[ip]
	if !exists {
		if len(m.ips) >= 1024 {
			fail(w, 429)
			return
		}
		a.until = now.Add(challengeTTL)
	}
	if a.count >= 5 || m.global.count >= 50 {
		fail(w, 429)
		return
	}
	valid := ok && len(body.DeviceName) <= 128 && ((body.Code != "") != (body.QRToken != "")) && now.Before(m.challenge.expires) && (equal(body.Code, m.challenge.code) || equal(body.QRToken, m.challenge.token))
	if !valid {
		a.count++
		m.ips[ip] = a
		m.global.count++
		fail(w, 403)
		return
	}
	name := strings.TrimSpace(body.DeviceName)
	if name == "" {
		name = "Eşleşmiş cihaz"
	}
	token, csrf, expires, err := m.create(r, name)
	if err != nil {
		fail(w, 500)
		return
	}
	m.challenge = challenge{}
	m.setCookie(w, r, token, int(lifetime.Seconds()))
	writeJSON(w, 200, map[string]any{"csrf_token": csrf, "local": local(r), "expires_at": expires})
}
func (m *Manager) devices(w http.ResponseWriter, r *http.Request) {
	sessions, err := m.store.ListAccessSessions(r.Context())
	if err != nil {
		fail(w, 500)
		return
	}
	type device struct {
		ID         string    `json:"id"`
		DeviceName string    `json:"device_name"`
		CreatedAt  time.Time `json:"created_at"`
		ExpiresAt  time.Time `json:"expires_at"`
	}
	out := make([]device, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, device{s.TokenHash, s.DeviceName, s.CreatedAt, s.ExpiresAt})
	}
	writeJSON(w, 200, map[string]any{"devices": out})
}
func (m *Manager) revoke(w http.ResponseWriter, r *http.Request, s store.AccessSession) {
	var b struct {
		ID  string `json:"id"`
		All bool   `json:"all"`
	}
	if !decode(w, r, &b) || ((b.ID != "") == b.All) {
		fail(w, 400)
		return
	}
	var err error
	if b.All {
		// Pair uses the same lock around its session insert and challenge
		// consumption, so it cannot insert between deletion and invalidation.
		m.mu.Lock()
		err = m.store.DeleteAllAccessSessions(r.Context())
		if err == nil {
			m.challenge = challenge{}
		}
		m.mu.Unlock()
	} else {
		err = m.store.DeleteAccessSession(r.Context(), b.ID)
	}
	if err != nil {
		fail(w, 500)
		return
	}
	if b.All || b.ID == s.TokenHash {
		m.clearCookie(w, r)
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
