package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"obs-screen-overlay/internal/store"
)

type initialFlushRecorder struct {
	*httptest.ResponseRecorder
	flushed chan struct{}
	once    sync.Once
}

func (r *initialFlushRecorder) Flush() {
	r.ResponseRecorder.Flush()
	r.once.Do(func() { close(r.flushed) })
}

// TestPublicEvents verifies the SSE endpoint: unknown token is 404, known
// token streams the initial snapshot as an `event: state` frame, the stream
// ends when the request context is cancelled, and the response carries the
// no-buffering headers.
func TestPublicEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p := store.Page{
		ID:          "one",
		Name:        "one",
		Type:        "widget",
		PublicToken: "secret",
		SourceHTML:  `<!doctype html><html><body>{{a}}</body></html>`,
		Fields:      []store.Field{{Name: "a", Value: "1", SSEEnabled: true}},
	}
	if err := s.CreatePage(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	r := NewPublicRouter(s)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/view/wrong/events", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /view/wrong/events: %d", w.Code)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w = httptest.NewRecorder()
	stream := &initialFlushRecorder{ResponseRecorder: w, flushed: make(chan struct{})}
	done := make(chan struct{})
	go func() {
		r.ServeHTTP(stream, httptest.NewRequestWithContext(ctx, http.MethodGet, "/view/secret/events", nil))
		close(done)
	}()
	select {
	case <-stream.flushed:
	case <-time.After(2 * time.Second):
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("SSE handler ilk snapshot beklenirken sonlanmadı")
		}
		t.Fatal("SSE ilk snapshot flush edilmedi")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SSE akışı ctx iptalinde sonlanmadı")
	}

	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.HasPrefix(body, "event: state\nid: 1\n") || !strings.Contains(body, `"revision":1`) || !strings.Contains(body, `"a":"1"`) || !strings.HasSuffix(body, "\n\n") {
		t.Fatalf("beklenmeyen SSE gövdesi: %q", body)
	}
	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type %q", got)
	}
	if got := w.Header().Get("Cache-Control"); !strings.Contains(got, "no-cache") {
		t.Fatalf("Cache-Control %q", got)
	}
	if w.Header().Get("X-Accel-Buffering") != "no" {
		t.Fatal("X-Accel-Buffering eksik")
	}
}
