package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestDeletePage_Success(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	target := Page{
		ID:          "test-id",
		Name:        "valid_name",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
		Fields: []Field{
			{Name: "field1", Value: "value1", DisplayOrder: 0},
			{Name: "field2", Value: "value2", DisplayOrder: 1},
		},
	}
	if err := store.CreatePage(context.Background(), target); err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	other := Page{
		ID:          "other-id",
		Name:        "other_name",
		Type:        "widget",
		SourceHTML:  "<html></html>",
		PublicToken: "token456",
		Fields:      []Field{{Name: "keep_field", Value: "kept", DisplayOrder: 0}},
	}
	if err := store.CreatePage(context.Background(), other); err != nil {
		t.Fatalf("failed to create other page: %v", err)
	}

	if err := store.DeletePage(context.Background(), "test-id"); err != nil {
		t.Fatalf("failed to delete page: %v", err)
	}

	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM pages WHERE id = ?", "test-id").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query pages: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 pages after delete, got %d", count)
	}

	// Fields are cascade deleted.
	err = store.db.QueryRow("SELECT COUNT(*) FROM fields WHERE page_id = ?", "test-id").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query fields: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 fields after cascade delete, got %d", count)
	}

	// The other page and its fields are untouched.
	page, err := store.GetPage(context.Background(), "other-id")
	if err != nil {
		t.Fatalf("failed to get other page: %v", err)
	}
	if len(page.Fields) != 1 || page.Fields[0].Name != "keep_field" {
		t.Errorf("expected other page field 'keep_field' preserved, got %+v", page.Fields)
	}
}

func TestDeletePage_NotFound(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	page := Page{
		ID:          "test-id",
		Name:        "valid_name",
		Type:        "widget",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
	}
	if err := store.CreatePage(context.Background(), page); err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	if err := store.DeletePage(context.Background(), "test-id"); err != nil {
		t.Fatalf("failed to delete page: %v", err)
	}

	if _, err := store.GetPage(context.Background(), "test-id"); err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for deleted page, got: %v", err)
	}
	if err := store.DeletePage(context.Background(), "test-id"); err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for repeated delete, got: %v", err)
	}
}

func TestDeletePage_FileDBPersists(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "overlay.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	page := Page{
		ID:          "test-id",
		Name:        "valid_name",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
		Fields:      []Field{{Name: "field1", Value: "value1", DisplayOrder: 0}},
	}
	if err := store.CreatePage(context.Background(), page); err != nil {
		t.Fatalf("failed to create page: %v", err)
	}
	if err := store.DeletePage(context.Background(), "test-id"); err != nil {
		t.Fatalf("failed to delete page: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}

	store, err = Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}
	defer store.Close()

	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM pages WHERE id = ?", "test-id").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query pages: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 pages after reopen, got %d", count)
	}
	err = store.db.QueryRow("SELECT COUNT(*) FROM fields WHERE page_id = ?", "test-id").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query fields: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 fields after reopen, got %d", count)
	}
}

func TestDeletePage_RollbackOnCascadeAbort(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	page := Page{
		ID:          "test-id",
		Name:        "valid_name",
		Type:        "widget",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
		Fields:      []Field{{Name: "field1", Value: "value1", DisplayOrder: 0}},
	}
	if err := store.CreatePage(context.Background(), page); err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	// A trigger that aborts every cascade delete from fields.
	if _, err := store.db.Exec(`
		CREATE TRIGGER no_delete BEFORE DELETE ON fields
		FOR EACH ROW
		BEGIN
			SELECT RAISE(ABORT, 'delete blocked by trigger');
		END
	`); err != nil {
		t.Fatalf("failed to create trigger: %v", err)
	}

	if err := store.DeletePage(context.Background(), "test-id"); err == nil {
		t.Fatal("expected an error when the trigger aborts the cascade delete")
	}

	// The parent page and its fields must be preserved.
	got, err := store.GetPage(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("failed to get page after aborted delete: %v", err)
	}
	if len(got.Fields) != 1 || got.Fields[0].Name != "field1" {
		t.Errorf("expected field 'field1' preserved, got %+v", got.Fields)
	}
}
