package store

import (
	"context"
	"database/sql"
	"testing"
)

func TestUpdatePage_Success(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	original := Page{
		ID:          "test-id",
		Name:        "old_name",
		Type:        "widget",
		SourceHTML:  "<html><body>as-is</body></html>",
		PublicToken: "token123",
		Fields: []Field{
			{Name: "stale_field", Value: "old", DisplayOrder: 0},
		},
	}
	if err := store.CreatePage(context.Background(), original); err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	revision, err := store.UpdatePage(context.Background(), "test-id", 1, "new_name", "screen_overlay", []Field{
		{Name: "score", Value: "10", SSEEnabled: true},
		{Name: "team", Value: "A", SSEEnabled: false},
	})
	if err != nil {
		t.Fatalf("failed to update page: %v", err)
	}
	if revision != 2 {
		t.Errorf("expected new revision 2, got %d", revision)
	}

	page, err := store.GetPage(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("failed to get page: %v", err)
	}

	if page.Name != "new_name" {
		t.Errorf("expected Name 'new_name', got '%s'", page.Name)
	}
	if page.Type != "screen_overlay" {
		t.Errorf("expected Type 'screen_overlay', got '%s'", page.Type)
	}
	if page.Revision != 2 {
		t.Errorf("expected Revision 2, got %d", page.Revision)
	}

	// Fields are replaced, ordered by the order of the given slice.
	want := []Field{
		{Name: "score", Value: "10", SSEEnabled: true, DisplayOrder: 0},
		{Name: "team", Value: "A", SSEEnabled: false, DisplayOrder: 1},
	}
	if len(page.Fields) != len(want) {
		t.Fatalf("expected %d fields after update, got %d", len(want), len(page.Fields))
	}
	for i := range want {
		if page.Fields[i] != want[i] {
			t.Errorf("field %d = %+v, want %+v", i, page.Fields[i], want[i])
		}
	}

	// source_html, public_token and template_version are left untouched.
	if page.SourceHTML != original.SourceHTML {
		t.Errorf("expected SourceHTML unchanged, got '%s'", page.SourceHTML)
	}
	if page.PublicToken != original.PublicToken {
		t.Errorf("expected PublicToken unchanged, got '%s'", page.PublicToken)
	}
	if page.TemplateVersion != 1 {
		t.Errorf("expected TemplateVersion 1, got %d", page.TemplateVersion)
	}
}

func TestUpdatePage_ConflictAndNotFound(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	page := Page{
		ID:          "test-id",
		Name:        "some_name",
		Type:        "widget",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
	}
	if err := store.CreatePage(context.Background(), page); err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	// Stale revision: the page exists (revision 1) but the expected one does not match.
	_, err = store.UpdatePage(context.Background(), "test-id", 99, "new_name", "widget", []Field{})
	if err != ErrConflict {
		t.Errorf("stale revision: expected ErrConflict, got %v", err)
	}

	// Unknown page: no row matches the id.
	_, err = store.UpdatePage(context.Background(), "missing-id", 1, "new_name", "widget", []Field{})
	if err != sql.ErrNoRows {
		t.Errorf("unknown id: expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdatePage_RollbackOnInsertAbort(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	original := Page{
		ID:          "test-id",
		Name:        "old_name",
		Type:        "widget",
		SourceHTML:  "<html><body>as-is</body></html>",
		PublicToken: "token123",
		Fields: []Field{
			{Name: "field1", Value: "value1", SSEEnabled: true, DisplayOrder: 0},
		},
	}
	if err := store.CreatePage(context.Background(), original); err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	// A trigger that aborts every fields insert, simulating an insert failure.
	if _, err := store.db.Exec(`
		CREATE TRIGGER no_insert BEFORE INSERT ON fields
		FOR EACH ROW
		BEGIN
			SELECT RAISE(ABORT, 'insert blocked by trigger');
		END
	`); err != nil {
		t.Fatalf("failed to create trigger: %v", err)
	}

	_, err = store.UpdatePage(context.Background(), "test-id", 1, "new_name", "screen_overlay", []Field{
		{Name: "fresh_field", Value: "v", SSEEnabled: false},
	})
	if err == nil {
		t.Fatal("expected an error when the trigger aborts the field insert")
	}

	// The failed transaction must roll back: name, type, revision and fields are restored.
	page, err := store.GetPage(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("failed to get page after rollback: %v", err)
	}
	if page.Name != original.Name {
		t.Errorf("expected Name reverted to '%s', got '%s'", original.Name, page.Name)
	}
	if page.Type != original.Type {
		t.Errorf("expected Type reverted to '%s', got '%s'", original.Type, page.Type)
	}
	if page.Revision != 1 {
		t.Errorf("expected Revision reverted to 1, got %d", page.Revision)
	}
	if len(page.Fields) != 1 || page.Fields[0].Name != "field1" {
		t.Errorf("expected original field 'field1' restored, got %+v", page.Fields)
	}
}
