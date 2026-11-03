package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestAccessSessionsLifecycle(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "sessions.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(time.Hour).Truncate(time.Second)
	before := time.Now().Unix()
	if err := s.CreateAccessSession(ctx, "hash-a", "csrf-a", "Cihaz", expires); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAccessSession(ctx, "hash-b", "csrf-b", "Diğer", expires); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAccessSession(ctx, "hash-a", "csrf", "duplicate", expires); err == nil {
		t.Fatal("duplicate hash accepted")
	}
	got, err := s.GetAccessSession(ctx, "hash-a")
	if err != nil {
		t.Fatal(err)
	}
	if got.TokenHash != "hash-a" || got.CSRFToken != "csrf-a" || got.DeviceName != "Cihaz" || !got.ExpiresAt.Equal(expires) || got.CreatedAt.Unix() < before || got.CreatedAt.Unix() > time.Now().Unix() {
		t.Fatalf("unexpected session: %+v", got)
	}
	var created, expiry int64
	if err := s.db.QueryRow("SELECT created_at, expires_at FROM access_sessions WHERE token_hash = ?", "hash-a").Scan(&created, &expiry); err != nil {
		t.Fatal(err)
	}
	if created != got.CreatedAt.Unix() || expiry != expires.Unix() {
		t.Fatal("timestamps not stored as Unix seconds")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	reopened, err := s.GetAccessSession(ctx, "hash-a")
	if err != nil || !reflect.DeepEqual(got, reopened) {
		t.Fatalf("reopen: %+v, %v", reopened, err)
	}
	// Expire one row directly so the test never sleeps or depends on timing.
	if _, err := s.db.Exec("UPDATE access_sessions SET expires_at = ? WHERE token_hash = ?", time.Now().Unix()-1, "hash-b"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAccessSession(ctx, "hash-b"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expired: %v", err)
	}
	if _, err := s.GetAccessSession(ctx, "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing: %v", err)
	}
	list, err := s.ListAccessSessions(ctx)
	if err != nil || len(list) != 1 || list[0].TokenHash != "hash-a" {
		t.Fatalf("list: %+v, %v", list, err)
	}
	if err := s.DeleteAccessSession(ctx, "hash-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAccessSession(ctx, "hash-a"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted: %v", err)
	}
	if err := s.CreateAccessSession(ctx, "hash-c", "csrf", "device", expires); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteAllAccessSessions(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM access_sessions").Scan(&count); err != nil || count != 0 {
		t.Fatalf("delete all count %d: %v", count, err)
	}
}

func TestAccessSessionValidation(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	future := time.Now().Add(time.Hour)
	for _, tc := range []struct {
		hash, csrf, device string
		expiry             time.Time
	}{
		{"", "csrf", "device", future}, {"hash", "", "device", future},
		{"hash", "csrf", "", future}, {"hash", "csrf", "device", time.Now().Add(-time.Second)},
		{"hash", "csrf", "device", time.Unix(time.Now().Unix(), 0)},
	} {
		if err := s.CreateAccessSession(ctx, tc.hash, tc.csrf, tc.device, tc.expiry); err == nil {
			t.Errorf("accepted invalid session: %+v", tc)
		}
	}
}

func TestMigrationV1PreservesPagesAndFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.db")
	db, err := sql.Open("sqlite3", buildDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	// Construct a genuine v1 schema without invoking the current migration.
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := (&Store{}).createBaseSchema(tx); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO pages (id, name, type, source_html, public_token, revision, template_version) VALUES ('p', 'Sayfa', 'widget', '<html></html>', 'public', 4, 3);
		INSERT INTO fields (page_id, name, value, sse_enabled, display_order) VALUES ('p', 'field', 'değer', 0, 7);
		PRAGMA user_version = 1`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 2 {
		t.Fatalf("version %d: %v", version, err)
	}
	p, err := s.GetPage(context.Background(), "p")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Sayfa" || p.Type != "widget" || p.SourceHTML != "<html></html>" || p.PublicToken != "public" || p.Revision != 4 || p.TemplateVersion != 3 || len(p.Fields) != 1 || p.Fields[0] != (Field{Name: "field", Value: "değer", SSEEnabled: false, DisplayOrder: 7}) {
		t.Fatalf("migration lost data: %+v", p)
	}
	if err := s.CreateAccessSession(context.Background(), "hash", "csrf", "device", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.GetAccessSession(context.Background(), "hash"); err != nil {
		t.Fatal(err)
	}
	reopened, err := s.GetPage(context.Background(), "p")
	if err != nil || !reflect.DeepEqual(p, reopened) {
		t.Fatalf("reopened page: %+v, %v", reopened, err)
	}
}
