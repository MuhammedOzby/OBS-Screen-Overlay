package store

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListPages_EmptyDatabase(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	pages, err := store.ListPages(context.Background(), "")
	if err != nil {
		t.Fatalf("failed to list pages: %v", err)
	}
	if len(pages) != 0 {
		t.Errorf("expected 0 pages, got %d", len(pages))
	}
}

func TestListPages_FilterByType(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	overlay := Page{
		ID:          "overlay-1",
		Name:        "overlay",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token-overlay",
	}
	if err := store.CreatePage(ctx, overlay); err != nil {
		t.Fatalf("failed to create overlay page: %v", err)
	}

	widget := Page{
		ID:          "widget-1",
		Name:        "widget",
		Type:        "widget",
		SourceHTML:  "<html></html>",
		PublicToken: "token-widget",
	}
	if err := store.CreatePage(ctx, widget); err != nil {
		t.Fatalf("failed to create widget page: %v", err)
	}

	// Empty type returns all pages
	all, err := store.ListPages(ctx, "")
	if err != nil {
		t.Fatalf("failed to list all pages: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(all))
	}

	// Fields must not be loaded for list results
	for _, p := range all {
		if len(p.Fields) != 0 {
			t.Errorf("page %s: expected no fields, got %d", p.ID, len(p.Fields))
		}
	}

	// screen_overlay filter returns only the overlay page
	overlays, err := store.ListPages(ctx, "screen_overlay")
	if err != nil {
		t.Fatalf("failed to list screen_overlay pages: %v", err)
	}
	if len(overlays) != 1 {
		t.Fatalf("expected 1 screen_overlay page, got %d", len(overlays))
	}
	if overlays[0].ID != "overlay-1" {
		t.Errorf("expected ID 'overlay-1', got '%s'", overlays[0].ID)
	}

	// widget filter returns only the widget page
	widgets, err := store.ListPages(ctx, "widget")
	if err != nil {
		t.Fatalf("failed to list widget pages: %v", err)
	}
	if len(widgets) != 1 {
		t.Fatalf("expected 1 widget page, got %d", len(widgets))
	}
	if widgets[0].ID != "widget-1" {
		t.Errorf("expected ID 'widget-1', got '%s'", widgets[0].ID)
	}
}

func TestListPages_InvalidType(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	for _, invalid := range []string{"dashboard", "Screen_Overlay", "screen overlay"} {
		pages, err := store.ListPages(context.Background(), invalid)
		if err == nil {
			t.Errorf("expected error for invalid type %q, got pages: %v", invalid, pages)
		}
	}
}

func TestListPages_Ordering(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	seed := func(id, createdAt, updatedAt string) {
		if _, err := store.db.ExecContext(ctx,
			"INSERT INTO pages (id, name, type, source_html, public_token, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			id, "name-"+id, "screen_overlay", "<html></html>", "token-"+id, createdAt, updatedAt,
		); err != nil {
			t.Fatalf("failed to seed page %s: %v", id, err)
		}
	}

	// a1 is oldest; m1 and z1 share the newest created_at and must be
	// ordered by id DESC among themselves.
	seed("a1", "2025-01-01 00:00:00", "2025-01-01 00:00:00")
	seed("m1", "2025-01-01 00:00:02", "2025-01-01 00:00:02")
	seed("z1", "2025-01-01 00:00:02", "2025-01-01 00:00:02")

	pages, err := store.ListPages(ctx, "")
	if err != nil {
		t.Fatalf("failed to list pages: %v", err)
	}

	var ids []string
	for _, p := range pages {
		ids = append(ids, p.ID)
	}
	expected := []string{"z1", "m1", "a1"}
	if !reflect.DeepEqual(ids, expected) {
		t.Errorf("expected order %v, got %v", expected, ids)
	}
}

func TestListPages_PersistsAcrossReopen(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "list_test.db")

	store1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	ctx := context.Background()
	for _, id := range []string{"page-1", "page-2"} {
		page := Page{
			ID:          id,
			Name:        "name-" + id,
			Type:        "screen_overlay",
			SourceHTML:  "<html></html>",
			PublicToken: "token-" + id,
		}
		if err := store1.CreatePage(ctx, page); err != nil {
			t.Fatalf("failed to create page %s: %v", id, err)
		}
	}
	if err := store1.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}

	store2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}
	defer store2.Close()

	pages, err := store2.ListPages(ctx, "")
	if err != nil {
		t.Fatalf("failed to list pages after reopen: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages after reopen, got %d", len(pages))
	}

	for i, id := range []string{"page-2", "page-1"} {
		if pages[i].ID != id {
			t.Errorf("position %d: expected ID %q, got %q", i, id, pages[i].ID)
		}
	}
}
