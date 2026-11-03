package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"obs-screen-overlay/internal/live"
)

const liveTestWait = 2 * time.Second

func openLiveTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return s
}

func createLiveTestPage(t *testing.T, s *Store, p Page) {
	t.Helper()
	if err := s.CreatePage(t.Context(), p); err != nil {
		t.Fatalf("CreatePage(%q): %v", p.ID, err)
	}
}

func subscribeLive(t *testing.T, s *Store, token string) (live.Snapshot, <-chan live.Snapshot, func()) {
	t.Helper()
	snap, ch, cancel, err := s.SubscribeLive(t.Context(), token)
	if err != nil {
		t.Fatalf("SubscribeLive(%q): %v", token, err)
	}
	t.Cleanup(cancel)
	return snap, ch, cancel
}

func assertSnapshotFields(t *testing.T, label string, snap live.Snapshot, want map[string]string) {
	t.Helper()
	if len(snap.Fields) != len(want) {
		t.Fatalf("%s: fields = %+v, istenen %+v", label, snap.Fields, want)
	}
	for name, value := range want {
		got, ok := snap.Fields[name]
		if !ok || got != value {
			t.Fatalf("%s: fields[%q] = %q (var=%v), istenen %q", label, name, got, ok, value)
		}
	}
}

func readLiveSnapshot(t *testing.T, ch <-chan live.Snapshot, label string) live.Snapshot {
	t.Helper()
	select {
	case snap, ok := <-ch:
		if !ok {
			t.Fatalf("%s: abonelik kanalı beklenmedik şekilde kapandı", label)
		}
		return snap
	case <-time.After(liveTestWait):
		t.Fatalf("%s: snapshot gelmedi", label)
		return live.Snapshot{}
	}
}

func assertNoLiveSnapshot(t *testing.T, ch <-chan live.Snapshot, label string) {
	t.Helper()
	select {
	case snap, ok := <-ch:
		if !ok {
			t.Fatalf("%s: abonelik kanalı beklenmedik şekilde kapandı", label)
		}
		t.Fatalf("%s: beklenmeyen snapshot: %+v", label, snap)
	default:
	}
}

// drainLiveRevisions reads every immediately available snapshot without
// sleeping. closed reports that the hub closed the channel (for example after
// the bounded queue overflowed).
func drainLiveRevisions(ch <-chan live.Snapshot) (revisions []int64, closed bool) {
	for {
		select {
		case snap, ok := <-ch:
			if !ok {
				return revisions, true
			}
			revisions = append(revisions, snap.Revision)
		default:
			return revisions, false
		}
	}
}

// TestSubscribeLiveSnapshotOnlyOpenFields verifies that both the initial and
// the reconnected snapshots carry the latest values but never fields whose SSE
// is disabled.
func TestSubscribeLiveSnapshotOnlyOpenFields(t *testing.T) {
	s := openLiveTestStore(t)
	createLiveTestPage(t, s, Page{
		ID:          "p",
		Name:        "Page",
		Type:        "widget",
		PublicToken: "tok",
		SourceHTML:  "<html></html>",
		Fields: []Field{
			{Name: "open", Value: "one", SSEEnabled: true},
			{Name: "closed", Value: "fixed", SSEEnabled: false, DisplayOrder: 1},
		},
	})

	snap, ch, _ := subscribeLive(t, s, "tok")
	if snap.Revision != 1 || snap.TemplateVersion != 1 {
		t.Fatalf("initial snapshot revision/template = %d/%d, istenen 1/1", snap.Revision, snap.TemplateVersion)
	}
	assertSnapshotFields(t, "initial snapshot", snap, map[string]string{"open": "one"})

	revision, err := s.UpdatePage(t.Context(), "p", 1, "Page", "widget", []Field{
		{Name: "open", Value: "two", SSEEnabled: true},
		{Name: "closed", Value: "fixed-2", SSEEnabled: false, DisplayOrder: 1},
	})
	if err != nil {
		t.Fatalf("UpdatePage: %v", err)
	}
	streamed := readLiveSnapshot(t, ch, "updated stream")
	if streamed.Revision != revision {
		t.Fatalf("streamed revision = %d, istenen %d", streamed.Revision, revision)
	}
	assertSnapshotFields(t, "updated stream", streamed, map[string]string{"open": "two"})

	// Reconnect: a fresh subscription must see the latest commit and still hide
	// the SSE-disabled field.
	reconnected, reconnectedCh, _ := subscribeLive(t, s, "tok")
	if reconnected.Revision != revision {
		t.Fatalf("reconnected revision = %d, istenen %d", reconnected.Revision, revision)
	}
	assertSnapshotFields(t, "reconnected snapshot", reconnected, map[string]string{"open": "two"})
	assertNoLiveSnapshot(t, reconnectedCh, "reconnected stream before any update")
}

// TestUpdatePageFailuresDoNotPublish verifies that validation errors, optimistic
// concurrency conflicts and a transaction rollback never publish a snapshot.
// The rollback is induced with the same trigger pattern used by the existing
// update/delete tests. A successful update afterwards proves the subscription
// is still healthy and only successful commits publish.
func TestUpdatePageFailuresDoNotPublish(t *testing.T) {
	s := openLiveTestStore(t)
	createLiveTestPage(t, s, Page{
		ID:          "p",
		Name:        "Page",
		Type:        "widget",
		PublicToken: "tok",
		SourceHTML:  "<html></html>",
		Fields: []Field{
			{Name: "open", Value: "one", SSEEnabled: true},
			{Name: "closed", Value: "fixed", SSEEnabled: false, DisplayOrder: 1},
		},
	})

	snap, ch, _ := subscribeLive(t, s, "tok")
	assertSnapshotFields(t, "initial snapshot", snap, map[string]string{"open": "one"})

	invalid := []struct {
		label  string
		name   string
		typ    string
		fields []Field
	}{
		{"empty name", "   ", "widget", []Field{{Name: "open", Value: "x", SSEEnabled: true}}},
		{"invalid field name", "Page", "widget", []Field{{Name: "bad-name", Value: "x", SSEEnabled: true}}},
	}
	for _, tc := range invalid {
		if _, err := s.UpdatePage(t.Context(), "p", 1, tc.name, tc.typ, tc.fields); !errors.Is(err, ErrValidation) {
			t.Fatalf("%s: err = %v, istenen ErrValidation", tc.label, err)
		}
		assertNoLiveSnapshot(t, ch, tc.label)
	}

	if _, err := s.UpdatePage(t.Context(), "p", 99, "Page", "widget", []Field{
		{Name: "open", Value: "stale", SSEEnabled: true},
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict: err = %v, istenen ErrConflict", err)
	}
	assertNoLiveSnapshot(t, ch, "conflict")

	if _, err := s.db.Exec(`
		CREATE TRIGGER live_no_insert BEFORE INSERT ON fields
		FOR EACH ROW
		BEGIN
			SELECT RAISE(ABORT, 'insert blocked by trigger');
		END
	`); err != nil {
		t.Fatalf("trigger oluşturulamadı: %v", err)
	}

	if _, err := s.UpdatePage(t.Context(), "p", 1, "Page", "widget", []Field{
		{Name: "open", Value: "rolled-back", SSEEnabled: true},
	}); err == nil {
		t.Fatal("rollback: trigger insert'i engellediği halde hata dönmedi")
	}
	assertNoLiveSnapshot(t, ch, "transaction rollback")

	page, err := s.GetPage(t.Context(), "p")
	if err != nil {
		t.Fatalf("GetPage after rollback: %v", err)
	}
	if page.Name != "Page" || page.Revision != 1 {
		t.Fatalf("rollback sonrası name/revision = %q/%d, istenen Page/1", page.Name, page.Revision)
	}
	if len(page.Fields) != 2 || page.Fields[0].Name != "open" || page.Fields[0].Value != "one" {
		t.Fatalf("rollback sonrası fields = %+v, özgün alanlar korunmalı", page.Fields)
	}

	if _, err := s.db.Exec(`DROP TRIGGER live_no_insert`); err != nil {
		t.Fatalf("trigger kaldırılamadı: %v", err)
	}
	revision, err := s.UpdatePage(t.Context(), "p", 1, "Page", "widget", []Field{
		{Name: "open", Value: "published", SSEEnabled: true},
		{Name: "closed", Value: "fixed", SSEEnabled: false, DisplayOrder: 1},
	})
	if err != nil {
		t.Fatalf("successful UpdatePage: %v", err)
	}
	published := readLiveSnapshot(t, ch, "successful update after failures")
	if published.Revision != revision {
		t.Fatalf("published revision = %d, istenen %d", published.Revision, revision)
	}
	assertSnapshotFields(t, "published snapshot", published, map[string]string{"open": "published"})
}

// TestSubscribeLiveConcurrentUpdate races SubscribeLive against UpdatePage.
// Holding liveMu across the read-and-subscribe window makes the pair
// consistent: the new revision must be either in the returned snapshot or in
// the streamed snapshots. The iteration count is small so the bounded hub
// queue should not overflow; if it nevertheless closes the channel, a fresh
// subscription is used for the final state instead of treating that as a
// failure.
func TestSubscribeLiveConcurrentUpdate(t *testing.T) {
	s := openLiveTestStore(t)
	createLiveTestPage(t, s, Page{
		ID:          "p",
		Name:        "Page",
		Type:        "widget",
		PublicToken: "tok",
		SourceHTML:  "<html></html>",
		Fields: []Field{
			{Name: "open", Value: "v0", SSEEnabled: true},
			{Name: "closed", Value: "fixed", SSEEnabled: false, DisplayOrder: 1},
		},
	})

	ctx := context.Background()
	revision := int64(1)
	const iterations = 12

	for i := 0; i < iterations; i++ {
		wantValue := fmt.Sprintf("v%d", i+1)
		start := make(chan struct{})
		var (
			snap        live.Snapshot
			ch          <-chan live.Snapshot
			cancel      func()
			subErr      error
			newRevision int64
			updateErr   error
			wg          sync.WaitGroup
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			snap, ch, cancel, subErr = s.SubscribeLive(ctx, "tok")
		}()
		go func() {
			defer wg.Done()
			<-start
			newRevision, updateErr = s.UpdatePage(ctx, "p", revision, "Page", "widget", []Field{
				{Name: "open", Value: wantValue, SSEEnabled: true},
				{Name: "closed", Value: "fixed", SSEEnabled: false, DisplayOrder: 1},
			})
		}()
		close(start)
		wg.Wait()

		if subErr != nil {
			t.Fatalf("iteration %d: SubscribeLive: %v", i, subErr)
		}
		if updateErr != nil {
			t.Fatalf("iteration %d: UpdatePage: %v", i, updateErr)
		}
		if ch == nil || cancel == nil {
			t.Fatalf("iteration %d: SubscribeLive boş kanal/cancel döndürdü", i)
		}

		streamRevisions, closed := drainLiveRevisions(ch)
		cancel()

		found := snap.Revision >= newRevision
		for _, streamRevision := range streamRevisions {
			if streamRevision >= newRevision {
				found = true
			}
		}
		if !found && closed {
			// Bounded hub queue overflow: reconnect for a fresh, consistent
			// snapshot instead of failing the race test.
			fresh, _, freshCancel, err := s.SubscribeLive(ctx, "tok")
			if err != nil {
				t.Fatalf("iteration %d: reconnect SubscribeLive: %v", i, err)
			}
			freshCancel()
			found = fresh.Revision >= newRevision
		}
		if !found {
			t.Fatalf("iteration %d: revision %d ne snapshot'ta ne stream'de (snapshot=%d stream=%v closed=%v)",
				i, newRevision, snap.Revision, streamRevisions, closed)
		}
		if snap.Revision >= newRevision {
			assertSnapshotFields(t, fmt.Sprintf("iteration %d snapshot", i), snap, map[string]string{"open": wantValue})
		}
		revision = newRevision
	}

	final, finalCh, _ := subscribeLive(t, s, "tok")
	if final.Revision != revision {
		t.Fatalf("final revision = %d, istenen %d", final.Revision, revision)
	}
	assertSnapshotFields(t, "final snapshot", final, map[string]string{"open": fmt.Sprintf("v%d", iterations)})
	assertNoLiveSnapshot(t, finalCh, "final idle stream")
}

// TestSubscribeLivePageIsolation verifies an update to one page is delivered
// only to that page's subscribers.
func TestSubscribeLivePageIsolation(t *testing.T) {
	s := openLiveTestStore(t)
	createLiveTestPage(t, s, Page{
		ID: "a", Name: "A", Type: "widget", PublicToken: "token-a",
		SourceHTML: "<html></html>",
		Fields:     []Field{{Name: "open", Value: "a1", SSEEnabled: true}},
	})
	createLiveTestPage(t, s, Page{
		ID: "b", Name: "B", Type: "widget", PublicToken: "token-b",
		SourceHTML: "<html></html>",
		Fields:     []Field{{Name: "open", Value: "b1", SSEEnabled: true}},
	})

	snapA, chA, _ := subscribeLive(t, s, "token-a")
	snapB, chB, _ := subscribeLive(t, s, "token-b")
	assertSnapshotFields(t, "page a snapshot", snapA, map[string]string{"open": "a1"})
	assertSnapshotFields(t, "page b snapshot", snapB, map[string]string{"open": "b1"})

	revisionA, err := s.UpdatePage(t.Context(), "a", 1, "A", "widget", []Field{
		{Name: "open", Value: "a2", SSEEnabled: true},
	})
	if err != nil {
		t.Fatalf("UpdatePage(a): %v", err)
	}
	streamedA := readLiveSnapshot(t, chA, "page a update")
	if streamedA.Revision != revisionA {
		t.Fatalf("page a revision = %d, istenen %d", streamedA.Revision, revisionA)
	}
	assertSnapshotFields(t, "page a stream", streamedA, map[string]string{"open": "a2"})
	assertNoLiveSnapshot(t, chB, "page b after a update")

	revisionB, err := s.UpdatePage(t.Context(), "b", 1, "B", "widget", []Field{
		{Name: "open", Value: "b2", SSEEnabled: true},
	})
	if err != nil {
		t.Fatalf("UpdatePage(b): %v", err)
	}
	streamedB := readLiveSnapshot(t, chB, "page b update")
	if streamedB.Revision != revisionB {
		t.Fatalf("page b revision = %d, istenen %d", streamedB.Revision, revisionB)
	}
	assertSnapshotFields(t, "page b stream", streamedB, map[string]string{"open": "b2"})
	assertNoLiveSnapshot(t, chA, "page a after b update")
}

// TestDeletePageClosesSubscriptions verifies the existing delete cleanup
// behavior: subscribers of the deleted page get their channel closed without a
// synthetic event, subscribers of other pages are unaffected, and cancelling an
// already-closed subscription is safe.
func TestDeletePageClosesSubscriptions(t *testing.T) {
	s := openLiveTestStore(t)
	createLiveTestPage(t, s, Page{
		ID: "a", Name: "A", Type: "widget", PublicToken: "token-a",
		SourceHTML: "<html></html>",
		Fields:     []Field{{Name: "open", Value: "a1", SSEEnabled: true}},
	})
	createLiveTestPage(t, s, Page{
		ID: "b", Name: "B", Type: "widget", PublicToken: "token-b",
		SourceHTML: "<html></html>",
		Fields:     []Field{{Name: "open", Value: "b1", SSEEnabled: true}},
	})

	_, chA, cancelA := subscribeLive(t, s, "token-a")
	_, chB, cancelB := subscribeLive(t, s, "token-b")

	if err := s.DeletePage(t.Context(), "a"); err != nil {
		t.Fatalf("DeletePage(a): %v", err)
	}

	// The deleted page's channel is closed; no page_deleted event is emitted.
	select {
	case snap, ok := <-chA:
		if ok {
			t.Fatalf("silinen sayfa için snapshot geldi: %+v", snap)
		}
	case <-time.After(liveTestWait):
		t.Fatal("silinen sayfanın abonelik kanalı kapanmadı")
	}

	cancelA()
	cancelA()

	// The other page is still subscribed and can receive its own update.
	assertNoLiveSnapshot(t, chB, "page b after page a delete")
	revisionB, err := s.UpdatePage(t.Context(), "b", 1, "B", "widget", []Field{
		{Name: "open", Value: "b2", SSEEnabled: true},
	})
	if err != nil {
		t.Fatalf("UpdatePage(b): %v", err)
	}
	streamedB := readLiveSnapshot(t, chB, "page b after page a delete")
	if streamedB.Revision != revisionB {
		t.Fatalf("page b revision = %d, istenen %d", streamedB.Revision, revisionB)
	}
	assertSnapshotFields(t, "page b stream after page a delete", streamedB, map[string]string{"open": "b2"})

	if err := s.DeletePage(t.Context(), "b"); err != nil {
		t.Fatalf("DeletePage(b): %v", err)
	}
	select {
	case snap, ok := <-chB:
		if ok {
			t.Fatalf("silinen sayfa b için snapshot geldi: %+v", snap)
		}
	case <-time.After(liveTestWait):
		t.Fatal("silinen sayfa b'nin abonelik kanalı kapanmadı")
	}
	cancelB()
}
