package store

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

const templateTestSource = `<!doctype html><html><body>{{konuk_adi}}</body></html>`

func templatePage(t *testing.T, s *Store) Page {
	t.Helper()
	p, err := s.GetPage(t.Context(), "p")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func assertTemplatePageUnchanged(t *testing.T, s *Store, before Page) {
	t.Helper()
	if after := templatePage(t, s); !reflect.DeepEqual(after, before) {
		t.Fatalf("page changed after failed publish:\nbefore: %+v\nafter: %+v", before, after)
	}
}

// createTemplateTestPage creates a page with one SSE-enabled field and one
// SSE-disabled field; the disabled one must never reach published snapshots.
func createTemplateTestPage(t *testing.T, s *Store) {
	t.Helper()
	createLiveTestPage(t, s, Page{
		ID:          "p",
		Name:        "Page",
		Type:        "widget",
		PublicToken: "tok",
		SourceHTML:  templateTestSource,
		Fields: []Field{
			{Name: "konuk_adi", Value: "ÖZBAY", SSEEnabled: true, DisplayOrder: 0},
			{Name: "sayac_baslangic", Value: "300", SSEEnabled: false, DisplayOrder: 1},
		},
	})
}

// TestPublishTemplate_SuccessPreservesFields verifies that a successful design
// publish advances source/revision/template_version and leaves the existing
// user-defined field set (list, values, SSE flags, order) untouched.
func TestPublishTemplate_SuccessPreservesFields(t *testing.T) {
	s := openLiveTestStore(t)
	createTemplateTestPage(t, s)

	newRev, err := s.PublishTemplate(t.Context(), "p", 1, `<!doctype html><html><body>{{sayac_baslangic}}</body></html>`)
	if err != nil {
		t.Fatalf("PublishTemplate: %v", err)
	}
	if newRev != 2 {
		t.Fatalf("new revision = %d, istenen 2", newRev)
	}

	page, err := s.GetPage(t.Context(), "p")
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	wantSource := `<!doctype html><html><body>{{sayac_baslangic}}</body></html>`
	if page.SourceHTML != wantSource {
		t.Fatalf("source = %q, istenen %q", page.SourceHTML, wantSource)
	}
	if page.Revision != 2 {
		t.Fatalf("revision = %d, istenen 2", page.Revision)
	}
	if page.TemplateVersion != 2 {
		t.Fatalf("template_version = %d, istenen 2", page.TemplateVersion)
	}

	want := []Field{
		{Name: "konuk_adi", Value: "ÖZBAY", SSEEnabled: true, DisplayOrder: 0},
		{Name: "sayac_baslangic", Value: "300", SSEEnabled: false, DisplayOrder: 1},
	}
	if len(page.Fields) != len(want) {
		t.Fatalf("fields = %+v, istenen %+v", page.Fields, want)
	}
	for i := range want {
		if page.Fields[i] != want[i] {
			t.Fatalf("field %d = %+v, istenen %+v", i, page.Fields[i], want[i])
		}
	}
}

// TestPublishTemplate_PublishesOnlyOpenFieldsAfterCommit verifies that after a
// successful commit the published snapshot carries the new revision and template
// version and only the SSE-enabled field; the SSE-disabled field stays out.
func TestPublishTemplate_PublishesOnlyOpenFieldsAfterCommit(t *testing.T) {
	s := openLiveTestStore(t)
	createTemplateTestPage(t, s)

	snap, ch, _ := subscribeLive(t, s, "tok")
	if snap.Revision != 1 || snap.TemplateVersion != 1 {
		t.Fatalf("initial snapshot: %+v", snap)
	}
	assertSnapshotFields(t, "initial snapshot", snap, map[string]string{"konuk_adi": "ÖZBAY"})

	newRev, err := s.PublishTemplate(t.Context(), "p", 1, templateTestSource)
	if err != nil {
		t.Fatalf("PublishTemplate: %v", err)
	}

	streamed := readLiveSnapshot(t, ch, "published snapshot")
	if streamed.Revision != newRev {
		t.Fatalf("streamed revision = %d, istenen %d", streamed.Revision, newRev)
	}
	if streamed.TemplateVersion != 2 {
		t.Fatalf("streamed template_version = %d, istenen 2", streamed.TemplateVersion)
	}
	assertSnapshotFields(t, "published snapshot", streamed, map[string]string{"konuk_adi": "ÖZBAY"})
}

// TestPublishTemplate_FailuresDoNotMutateOrPublish verifies that a stale
// revision, an invalid source or a missing page leaves the stored page and the
// live stream untouched (no mutation, no event).
func TestPublishTemplate_FailuresDoNotMutateOrPublish(t *testing.T) {
	for _, test := range []struct {
		name, id, source string
		revision         int64
		want             error
	}{
		{"stale", "p", templateTestSource, 99, ErrConflict},
		{"invalid", "p", "<html></html>", 1, ErrValidation},
		{"missing", "missing", templateTestSource, 1, sql.ErrNoRows},
		{"negative", "p", templateTestSource, -1, ErrValidation},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := openLiveTestStore(t)
			createTemplateTestPage(t, s)
			before := templatePage(t, s)
			_, ch, _ := subscribeLive(t, s, "tok")
			if _, err := s.PublishTemplate(t.Context(), test.id, test.revision, test.source); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			assertNoLiveSnapshot(t, ch, test.name)
			assertTemplatePageUnchanged(t, s, before)
		})
	}
}

// TestPublishTemplate_RollbackOnUpdateAbort verifies that an aborted update
// rolls back the transaction and therefore does not publish a snapshot.
func TestPublishTemplate_RollbackOnUpdateAbort(t *testing.T) {
	s := openLiveTestStore(t)
	createTemplateTestPage(t, s)
	before := templatePage(t, s)
	_, ch, _ := subscribeLive(t, s, "tok")

	if _, err := s.db.Exec(`
		CREATE TRIGGER no_update BEFORE UPDATE ON pages
		FOR EACH ROW
		BEGIN
			SELECT RAISE(ABORT, 'update blocked by trigger');
		END
	`); err != nil {
		t.Fatalf("trigger oluşturulamadı: %v", err)
	}

	if _, err := s.PublishTemplate(t.Context(), "p", 1, `<!doctype html><html><body>Blocked design</body></html>`); err == nil {
		t.Fatal("rollback: update trigger'i engellediği halde hata dönmedi")
	}

	assertTemplatePageUnchanged(t, s, before)
	assertNoLiveSnapshot(t, ch, "aborted UPDATE")
}

// TestPublishTemplate_ConcurrentSameRevisionExactlyOneSucceeds verifies that
// two concurrent publishes for the same expected revision result in exactly
// one success and one conflict.
func TestPublishTemplate_ConcurrentSameRevisionExactlyOneSucceeds(t *testing.T) {
	s := openLiveTestStore(t)
	createTemplateTestPage(t, s)
	_, ch, _ := subscribeLive(t, s, "tok")

	type publishOutcome struct {
		rev    int64
		err    error
		source string
	}

	outcomes := make(chan publishOutcome, 2)
	start := make(chan struct{})
	for _, source := range []string{
		`<!doctype html><html><body>Design A</body></html>`,
		`<!doctype html><html><body>Design B</body></html>`,
	} {
		go func(source string) {
			<-start
			rev, err := s.PublishTemplate(t.Context(), "p", 1, source)
			outcomes <- publishOutcome{rev, err, source}
		}(source)
	}
	close(start)
	deadline := time.NewTimer(liveTestWait)
	defer deadline.Stop()

	successes := 0
	var conflictErr error
	var winner string
	for i := 0; i < 2; i++ {
		var out publishOutcome
		select {
		case out = <-outcomes:
		case <-deadline.C:
			t.Fatal("concurrent publishes did not finish")
		}
		if out.err == nil {
			successes++
			winner = out.source
			if out.rev != 2 {
				t.Fatalf("success revision = %d, istenen 2", out.rev)
			}
		} else if errors.Is(out.err, ErrConflict) {
			conflictErr = out.err
		} else {
			t.Fatalf("unexpected publish error: %v", out.err)
		}
	}

	if successes != 1 || conflictErr == nil {
		t.Fatalf("concurrent publish sonuçları: successes=%d, conflict=%v", successes, conflictErr)
	}
	page := templatePage(t, s)
	if page.SourceHTML != winner || page.Revision != 2 || page.TemplateVersion != 2 {
		t.Fatalf("stored page does not match winner: %+v", page)
	}
	snapshot := readLiveSnapshot(t, ch, "concurrent winner")
	if snapshot.Revision != 2 || snapshot.TemplateVersion != 2 {
		t.Fatalf("winner snapshot: %+v", snapshot)
	}
	assertSnapshotFields(t, "concurrent winner", snapshot, map[string]string{"konuk_adi": "ÖZBAY"})
	assertNoLiveSnapshot(t, ch, "only one concurrent publish may succeed")
}

func TestPublishTemplate_CommitFailureDoesNotMutateOrPublish(t *testing.T) {
	s := openLiveTestStore(t)
	createTemplateTestPage(t, s)
	before := templatePage(t, s)
	_, ch, _ := subscribeLive(t, s, "tok")
	// The UPDATE succeeds, but its deferred foreign-key violation fails COMMIT.
	if _, err := s.db.Exec(`
		CREATE TABLE deferred_publish_failure (
			page_id TEXT REFERENCES pages(id) DEFERRABLE INITIALLY DEFERRED
		);
		CREATE TRIGGER fail_publish_commit AFTER UPDATE ON pages BEGIN
			INSERT INTO deferred_publish_failure(page_id) VALUES ('missing-parent');
		END;
	`); err != nil {
		t.Fatal(err)
	}
	_, err := s.PublishTemplate(t.Context(), "p", 1, `<!doctype html><html><body>Uncommitted design</body></html>`)
	if err == nil || !strings.Contains(err.Error(), "commit") {
		t.Fatalf("expected COMMIT failure, got %v", err)
	}
	assertTemplatePageUnchanged(t, s, before)
	assertNoLiveSnapshot(t, ch, "failed COMMIT")
}
