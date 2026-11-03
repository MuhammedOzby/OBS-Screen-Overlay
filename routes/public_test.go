package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"obs-screen-overlay/internal/store"
)

func TestPublicView(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	source := `<!doctype html><html><body>{{on}} {{off}} {{on}}<script>const x = 1 < 2;</script><svg><circle cx="2"/></svg></body></html>`
	p := store.Page{ID: "one", Name: "one", Type: "widget", PublicToken: "secret", SourceHTML: source, Fields: []store.Field{{Name: "on", Value: `<img src=x>&"`, SSEEnabled: true}, {Name: "off", Value: "disabled saved"}}}
	if err := s.CreatePage(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	r := NewPublicRouter(s)
	request := func(method, path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
		return w
	}
	w := request(http.MethodGet, "/view/secret")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	escaped := `<span data-overlay-field="on">&lt;img src=x&gt;&amp;&#34;</span>`
	if strings.Count(body, escaped) != 2 || !strings.Contains(body, `<span data-overlay-field="off">disabled saved</span>`) || !strings.Contains(body, `<script>const x = 1 < 2;</script><svg><circle cx="2"/></svg>`) {
		t.Fatalf("unexpected HTML: %s", body)
	}
	if !strings.Contains(w.Header().Get("Cache-Control"), "no-store") || w.Header().Get("Referrer-Policy") != "no-referrer" || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatal(w.Header())
	}
	for _, path := range []string{"/view/wrong", "/view/one", "/", "/yeni", "/duzelt/one", "/api/pages", "/assets/test", "/view/wrong/events"} {
		w = request(http.MethodGet, path)
		if w.Code != http.StatusNotFound {
			t.Errorf("GET %s: %d", path, w.Code)
		}
	}
	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodDelete} {
		if w := request(method, "/api/pages/one"); w.Code != http.StatusNotFound {
			t.Errorf("%s: %d", method, w.Code)
		}
	}
}
