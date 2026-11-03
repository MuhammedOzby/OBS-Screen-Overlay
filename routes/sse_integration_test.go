package routes

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"obs-screen-overlay/internal/live"
	"obs-screen-overlay/internal/store"
)

const (
	sseIntegrationWait    = 2 * time.Second
	sseIntegrationNoEvent = 250 * time.Millisecond
)

type sseTestFrame struct {
	Event string
	ID    string
	Data  string
}

type sseTestStream struct {
	events chan sseTestFrame
}

// openSSETestStream opens a real streaming request against the public router.
// Frames are parsed in a background goroutine so tests can wait with a timeout
// instead of sleeping for an arbitrary duration.
func openSSETestStream(t *testing.T, baseURL, path string) *sseTestStream {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		cancel()
		t.Fatalf("SSE isteği oluşturulamadı: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		t.Fatalf("SSE bağlantısı açılamadı: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		cancel()
		t.Fatalf("SSE status = %d, body = %q", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		resp.Body.Close()
		cancel()
		t.Fatalf("SSE Content-Type = %q", got)
	}

	stream := &sseTestStream{
		events: make(chan sseTestFrame, 16),
	}
	go func() {
		defer close(stream.events)
		scanner := bufio.NewScanner(resp.Body)
		frame := sseTestFrame{}
		hasData := false
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				if hasData {
					select {
					case stream.events <- frame:
					case <-ctx.Done():
						return
					}
				}
				frame = sseTestFrame{}
				hasData = false
				continue
			}
			if strings.HasPrefix(line, ":") {
				continue
			}
			field, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			value = strings.TrimPrefix(value, " ")
			switch field {
			case "event":
				frame.Event = value
			case "id":
				frame.ID = value
			case "data":
				if hasData {
					frame.Data += "\n"
				}
				frame.Data += value
				hasData = true
			}
		}
	}()

	t.Cleanup(func() {
		cancel()
		resp.Body.Close()
	})
	return stream
}

// next returns the next frame, or false if no frame arrives within timeout.
// An unexpectedly closed stream is a test failure.
func (s *sseTestStream) next(t *testing.T, timeout time.Duration) (sseTestFrame, bool) {
	t.Helper()
	select {
	case frame, ok := <-s.events:
		if !ok {
			t.Fatal("SSE akışı beklenmedik şekilde kapandı")
		}
		return frame, true
	case <-time.After(timeout):
		return sseTestFrame{}, false
	}
}

func stateFromFrame(t *testing.T, frame sseTestFrame) live.Snapshot {
	t.Helper()
	if frame.Event != "state" {
		t.Fatalf("SSE event = %q, istenen state", frame.Event)
	}
	var snap live.Snapshot
	if err := json.Unmarshal([]byte(frame.Data), &snap); err != nil {
		t.Fatalf("SSE data JSON okunamadı: %v (data=%q)", err, frame.Data)
	}
	if frame.ID != strconv.FormatInt(snap.Revision, 10) {
		t.Fatalf("SSE id = %q, revision = %d", frame.ID, snap.Revision)
	}
	return snap
}

func expectState(t *testing.T, stream *sseTestStream, wantRevision int64) live.Snapshot {
	t.Helper()
	frame, ok := stream.next(t, sseIntegrationWait)
	if !ok {
		t.Fatalf("SSE state olayı gelmedi (beklenen revision %d)", wantRevision)
	}
	snap := stateFromFrame(t, frame)
	if snap.Revision != wantRevision {
		t.Fatalf("revision = %d, istenen %d", snap.Revision, wantRevision)
	}
	return snap
}

func assertField(t *testing.T, snap live.Snapshot, name, want string) {
	t.Helper()
	got, ok := snap.Fields[name]
	if !ok {
		t.Fatalf("alan %q payload'da yok: %+v", name, snap.Fields)
	}
	if got != want {
		t.Fatalf("alan %q = %q, istenen %q", name, got, want)
	}
}

func assertNoField(t *testing.T, snap live.Snapshot, name string) {
	t.Helper()
	if got, ok := snap.Fields[name]; ok {
		t.Fatalf("SSE kapalı alan %q payload'da olmamalı, değer = %q", name, got)
	}
}

func assertNoEvent(t *testing.T, stream *sseTestStream, context string) {
	t.Helper()
	if frame, ok := stream.next(t, sseIntegrationNoEvent); ok {
		t.Fatalf("%s: beklenmeyen SSE olayı: %+v", context, frame)
	}
}

type sseTestField struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	SSEEnabled bool   `json:"sse_enabled"`
}

type sseTestPatch struct {
	ExpectedRevision int64          `json:"expected_revision"`
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	Fields           []sseTestField `json:"fields"`
}

func patchPageHTTP(t *testing.T, adminURL, id string, expectedRevision int64, name, typ string, fields []sseTestField) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(sseTestPatch{
		ExpectedRevision: expectedRevision,
		Name:             name,
		Type:             typ,
		Fields:           fields,
	})
	if err != nil {
		t.Fatalf("PATCH gövdesi hazırlanamadı: %v", err)
	}
	req, err := http.NewRequest(http.MethodPatch, adminURL+"/api/pages/"+id, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("PATCH isteği oluşturulamadı: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH isteği başarısız: %v", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("PATCH yanıtı okunamadı: %v", err)
	}
	return resp.StatusCode, respBody
}

func mustPatch(t *testing.T, adminURL, id string, expectedRevision int64, name, typ string, fields []sseTestField) int64 {
	t.Helper()
	status, body := patchPageHTTP(t, adminURL, id, expectedRevision, name, typ, fields)
	if status != http.StatusOK {
		t.Fatalf("PATCH status = %d, body = %s", status, body)
	}
	var out struct {
		Revision int64 `json:"revision"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("PATCH yanıtı JSON okunamadı: %v (body=%s)", err, body)
	}
	return out.Revision
}

// TestSSEPatchIntegration exercises the real admin and public route
// registration over httptest servers: PATCH commits, publish happens after
// commit, and only SSE-enabled fields reach the event stream.
func TestSSEPatchIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// RegisterRoutes loads panel templates with paths relative to the
	// repository root. Run this test from the root and restore the package
	// working directory automatically afterwards.
	t.Chdir("..")

	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	if err := s.CreatePage(t.Context(), store.Page{
		ID:          "p1",
		Name:        "Birinci",
		Type:        "widget",
		PublicToken: "token-a",
		SourceHTML:  `<!doctype html><html><body>{{open}} {{closed}}</body></html>`,
		Fields: []store.Field{
			{Name: "open", Value: "one", SSEEnabled: true},
			{Name: "closed", Value: "fixed", SSEEnabled: false},
		},
	}); err != nil {
		t.Fatalf("CreatePage p1: %v", err)
	}
	if err := s.CreatePage(t.Context(), store.Page{
		ID:          "p2",
		Name:        "İkinci",
		Type:        "widget",
		PublicToken: "token-b",
		SourceHTML:  `<!doctype html><html><body>{{other}}</body></html>`,
		Fields:      []store.Field{{Name: "other", Value: "only", SSEEnabled: true}},
	}); err != nil {
		t.Fatalf("CreatePage p2: %v", err)
	}

	admin := gin.New()
	if err := RegisterRoutes(admin, s, "http://public.test"); err != nil {
		t.Fatalf("RegisterRoutes: %v", err)
	}
	adminServer := httptest.NewServer(admin)
	t.Cleanup(adminServer.Close)

	public := gin.New()
	RegisterPublicRoutes(public, s)
	publicServer := httptest.NewServer(public)
	t.Cleanup(publicServer.Close)

	streamA := openSSETestStream(t, publicServer.URL, "/view/token-a/events")
	streamB := openSSETestStream(t, publicServer.URL, "/view/token-a/events")
	streamOther := openSSETestStream(t, publicServer.URL, "/view/token-b/events")

	// Initial subscription snapshot: both clients see the same revision and
	// only the SSE-enabled field; the closed field must not be present.
	initialA := expectState(t, streamA, 1)
	initialB := expectState(t, streamB, 1)
	assertField(t, initialA, "open", "one")
	assertField(t, initialB, "open", "one")
	assertNoField(t, initialA, "closed")
	assertNoField(t, initialB, "closed")
	initialOther := expectState(t, streamOther, 1)
	assertField(t, initialOther, "other", "only")

	allFields := []sseTestField{
		{Name: "open", Value: "two", SSEEnabled: true},
		{Name: "closed", Value: "fixed-2", SSEEnabled: false},
	}

	// A single PATCH commits both fields and publishes once. Both subscribers
	// receive the same revision; the closed field is not in the payload.
	if got := mustPatch(t, adminServer.URL, "p1", 1, "Birinci", "widget", allFields); got != 2 {
		t.Fatalf("PATCH revision = %d, istenen 2", got)
	}
	stateA2 := expectState(t, streamA, 2)
	stateB2 := expectState(t, streamB, 2)
	assertField(t, stateA2, "open", "two")
	assertField(t, stateB2, "open", "two")
	assertNoField(t, stateA2, "closed")
	assertNoField(t, stateB2, "closed")

	// Turning SSE off for a field removes it from subsequent state payloads;
	// the empty map is valid and does not mean "clear the DOM".
	offFields := []sseTestField{
		{Name: "open", Value: "three", SSEEnabled: false},
		{Name: "closed", Value: "fixed-2", SSEEnabled: false},
	}
	if got := mustPatch(t, adminServer.URL, "p1", 2, "Birinci", "widget", offFields); got != 3 {
		t.Fatalf("PATCH revision = %d, istenen 3", got)
	}
	stateA3 := expectState(t, streamA, 3)
	stateB3 := expectState(t, streamB, 3)
	if len(stateA3.Fields) != 0 || len(stateB3.Fields) != 0 {
		t.Fatalf("SSE kapalıyken fields boş olmalı: A=%+v B=%+v", stateA3.Fields, stateB3.Fields)
	}

	// Turning SSE back on applies the current stored value to both clients.
	onFields := []sseTestField{
		{Name: "open", Value: "four", SSEEnabled: true},
		{Name: "closed", Value: "fixed-2", SSEEnabled: false},
	}
	if got := mustPatch(t, adminServer.URL, "p1", 3, "Birinci", "widget", onFields); got != 4 {
		t.Fatalf("PATCH revision = %d, istenen 4", got)
	}
	stateA4 := expectState(t, streamA, 4)
	stateB4 := expectState(t, streamB, 4)
	assertField(t, stateA4, "open", "four")
	assertField(t, stateB4, "open", "four")
	assertNoField(t, stateA4, "closed")
	assertNoField(t, stateB4, "closed")

	// Page isolation: none of p1's updates may leak to p2's subscriber.
	assertNoEvent(t, streamOther, "p1 güncellemesi p2'ye sızdı")

	// A stale revision is rejected with 409 and must not publish anything.
	staleFields := []sseTestField{
		{Name: "open", Value: "stale", SSEEnabled: true},
		{Name: "closed", Value: "fixed-2", SSEEnabled: false},
	}
	status, body := patchPageHTTP(t, adminServer.URL, "p1", 3, "Birinci", "widget", staleFields)
	if status != http.StatusConflict {
		t.Fatalf("stale PATCH status = %d, istenen 409, body = %s", status, body)
	}
	assertNoEvent(t, streamA, "stale PATCH yayın yaptı (A)")
	assertNoEvent(t, streamB, "stale PATCH yayın yaptı (B)")

	// The stream must still be usable after the rejected request.
	finalFields := []sseTestField{
		{Name: "open", Value: "five", SSEEnabled: true},
		{Name: "closed", Value: "fixed-2", SSEEnabled: false},
	}
	if got := mustPatch(t, adminServer.URL, "p1", 4, "Birinci", "widget", finalFields); got != 5 {
		t.Fatalf("PATCH revision = %d, istenen 5", got)
	}
	stateA5 := expectState(t, streamA, 5)
	stateB5 := expectState(t, streamB, 5)
	assertField(t, stateA5, "open", "five")
	assertField(t, stateB5, "open", "five")
	assertNoField(t, stateA5, "closed")
	assertNoField(t, stateB5, "closed")
	assertNoEvent(t, streamOther, "son p1 güncellemesi p2'ye sızdı")
}
