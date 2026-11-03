package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestManyFieldsPersistence tests persistence with a large number of fields.
// It validates field creation, retrieval, and integrity with Turkish characters in filename.
func TestManyFieldsPersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use t.TempDir with Turkish characters/space/?/# in filename
	tmpDir := t.TempDir()
	dbPath := strings.Join([]string{tmpDir, "yayın ?# deneme.db"}, string(filepath.Separator))

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Use t.Cleanup for close (will close the reopened store)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("failed to close database: %v", err)
		}
	})

	// Create a page with 10000 fields checking all properties
	page := Page{
		ID:          "p1",
		Name:        "Canlı Yayın Çerçevisi",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "t1",
		Fields:      make([]Field, 10000),
	}

	for i := 0; i < 10000; i++ {
		page.Fields[i] = Field{
			Name:         fmt.Sprintf("alan_%d", i),
			Value:        fmt.Sprintf("Özbay <&> %d", i),
			SSEEnabled:   i%2 == 0,
			DisplayOrder: i,
		}
	}

	// Create the page
	err = store.CreatePage(ctx, page)
	if err != nil {
		t.Fatalf("failed to create page with 10000 fields: %v", err)
	}

	// Close the store and reopen to verify persistence
	if err := store.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}
	store, err = Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}

	// Retrieve the page and verify all fields
	retrieved, err := store.GetPage(ctx, "p1")
	if err != nil {
		t.Fatalf("failed to get page: %v", err)
	}

	// Verify page properties
	if retrieved.ID != "p1" {
		t.Errorf("expected ID 'p1', got '%s'", retrieved.ID)
	}
	if retrieved.Name != "Canlı Yayın Çerçevisi" {
		t.Errorf("expected Name 'Canlı Yayın Çerçevisi', got '%s'", retrieved.Name)
	}
	if retrieved.Type != "screen_overlay" {
		t.Errorf("expected Type 'screen_overlay', got '%s'", retrieved.Type)
	}
	if retrieved.PublicToken != "t1" {
		t.Errorf("expected PublicToken 't1', got '%s'", retrieved.PublicToken)
	}
	if retrieved.SourceHTML != page.SourceHTML {
		t.Errorf("expected SourceHTML '%s', got '%s'", page.SourceHTML, retrieved.SourceHTML)
	}
	if retrieved.Revision != 1 {
		t.Errorf("expected Revision 1, got %d", retrieved.Revision)
	}
	if retrieved.TemplateVersion != 1 {
		t.Errorf("expected TemplateVersion 1, got %d", retrieved.TemplateVersion)
	}

	// Verify all 10000 fields
	if len(retrieved.Fields) != 10000 {
		t.Fatalf("expected 10000 fields, got %d", len(retrieved.Fields))
	}

	for i := 0; i < 10000; i++ {
		expectedName := fmt.Sprintf("alan_%d", i)
		expectedValue := fmt.Sprintf("Özbay <&> %d", i)
		expectedSSE := i%2 == 0
		expectedOrder := i

		if retrieved.Fields[i].Name != expectedName {
			t.Errorf("field %d: expected name '%s', got '%s'", i, expectedName, retrieved.Fields[i].Name)
		}
		if retrieved.Fields[i].Value != expectedValue {
			t.Errorf("field %d: expected value '%s', got '%s'", i, expectedValue, retrieved.Fields[i].Value)
		}
		if retrieved.Fields[i].SSEEnabled != expectedSSE {
			t.Errorf("field %d: expected SSEEnabled %v, got %v", i, expectedSSE, retrieved.Fields[i].SSEEnabled)
		}
		if retrieved.Fields[i].DisplayOrder != expectedOrder {
			t.Errorf("field %d: expected DisplayOrder %d, got %d", i, expectedOrder, retrieved.Fields[i].DisplayOrder)
		}
	}

	// Verify DELETE works correctly (should delete all fields due to CASCADE)
	_, err = store.db.Exec("DELETE FROM pages WHERE id = ?", "p1")
	if err != nil {
		t.Fatalf("failed to delete page: %v", err)
	}

	// Verify page is deleted
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM pages WHERE id = ?", "p1").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query pages: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 pages after delete, got %d", count)
	}

	// Verify all fields are deleted due to CASCADE delete
	err = store.db.QueryRow("SELECT COUNT(*) FROM fields WHERE page_id = ?", "p1").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query fields: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 fields after page delete (cascade), got %d", count)
	}
}

// TestFieldSchemaIntegrity tests various field schema constraints and error conditions.
func TestFieldSchemaIntegrity(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	t.Run("orphan_field_insert_error", func(t *testing.T) {
		// Try to insert a field for a non-existent page
		_, err := store.db.Exec(
			"INSERT INTO fields (page_id, name, value, sse_enabled, display_order) VALUES (?, ?, ?, ?, ?)",
			"nonexistent_page", "field1", "value1", 1, 0,
		)
		if err == nil {
			t.Error("expected error for orphan field (foreign key constraint)")
		}
		// SQLite returns SQLITE_CONSTRAINT_FOREIGN for foreign key violations
	})

	t.Run("duplicate_public_token_error", func(t *testing.T) {
		// First, create a page with a public_token
		_, err = store.db.Exec(
			"INSERT INTO pages (id, name, type, source_html, public_token) VALUES (?, ?, ?, ?, ?)",
			"page1", "Page 1", "screen_overlay", "", "duplicate_token",
		)
		if err != nil {
			t.Fatalf("failed to insert first page: %v", err)
		}

		// Try to insert another page with the same public_token (should fail due to UNIQUE constraint)
		_, err := store.db.Exec(
			"INSERT INTO pages (id, name, type, source_html, public_token) VALUES (?, ?, ?, ?, ?)",
			"page2", "Page 2", "widget", "", "duplicate_token",
		)
		if err == nil {
			t.Error("expected error for duplicate public_token (UNIQUE constraint)")
		}
	})

	t.Run("invalid_type_error", func(t *testing.T) {
		// Try to insert a page with invalid type (not 'screen_overlay' or 'widget')
		_, err := store.db.Exec(
			"INSERT INTO pages (id, name, type, source_html, public_token) VALUES (?, ?, ?, ?, ?)",
			"page_invalid_type", "Page Invalid", "invalid_type", "", "token_invalid",
		)
		if err == nil {
			t.Error("expected error for invalid type (CHECK constraint)")
		}
	})

	t.Run("invalid_sse_enabled_error", func(t *testing.T) {
		// First create a valid page
		_, err = store.db.Exec(
			"INSERT INTO pages (id, name, type, source_html, public_token) VALUES (?, ?, ?, ?, ?)",
			"page_sse_test", "Page SSE", "screen_overlay", "", "token_sse",
		)
		if err != nil {
			t.Fatalf("failed to insert page: %v", err)
		}

		// Try to insert a field with sse_enabled = 2 (must be 0 or 1)
		_, err := store.db.Exec(
			"INSERT INTO fields (page_id, name, value, sse_enabled, display_order) VALUES (?, ?, ?, ?, ?)",
			"page_sse_test", "field_invalid_sse", "value", 2, 0,
		)
		if err == nil {
			t.Error("expected error for invalid sse_enabled value (must be 0 or 1)")
		}
	})

	t.Run("negative_display_order_error", func(t *testing.T) {
		// First create a valid page
		_, err = store.db.Exec(
			"INSERT INTO pages (id, name, type, source_html, public_token) VALUES (?, ?, ?, ?, ?)",
			"page_order_test", "Page Order", "screen_overlay", "", "token_order",
		)
		if err != nil {
			t.Fatalf("failed to insert page: %v", err)
		}

		// Try to insert a field with negative display_order
		_, err := store.db.Exec(
			"INSERT INTO fields (page_id, name, value, sse_enabled, display_order) VALUES (?, ?, ?, ?, ?)",
			"page_order_test", "field_neg_order", "value", 1, -1,
		)
		if err == nil {
			t.Error("expected error for negative display_order (CHECK constraint)")
		}
	})

	t.Run("negative_revision_update_error", func(t *testing.T) {
		// First create a valid page
		_, err = store.db.Exec(
			"INSERT INTO pages (id, name, type, source_html, public_token) VALUES (?, ?, ?, ?, ?)",
			"page_rev_test", "Page Rev", "screen_overlay", "", "token_rev",
		)
		if err != nil {
			t.Fatalf("failed to insert page: %v", err)
		}

		// Try to update revision to 0 (must be >= 1)
		_, err := store.db.Exec(
			"UPDATE pages SET revision = ? WHERE id = ?",
			0, "page_rev_test",
		)
		if err == nil {
			t.Error("expected error for revision 0 (CHECK constraint)")
		}
	})

	t.Run("page_and_field_counts", func(t *testing.T) {
		// Create a page with fields
		_, err = store.db.Exec(
			"INSERT INTO pages (id, name, type, source_html, public_token) VALUES (?, ?, ?, ?, ?)",
			"count_test", "Count Test", "screen_overlay", "", "token_count",
		)
		if err != nil {
			t.Fatalf("failed to insert page: %v", err)
		}

		// Insert multiple fields
		for i := 0; i < 5; i++ {
			_, err = store.db.Exec(
				"INSERT INTO fields (page_id, name, value, sse_enabled, display_order) VALUES (?, ?, ?, ?, ?)",
				"count_test", fmt.Sprintf("field_%d", i), "value", 1, i,
			)
			if err != nil {
				t.Fatalf("failed to insert field %d: %v", i, err)
			}
		}

		// Verify counts
		var pageCount, fieldCount int
		err = store.db.QueryRow("SELECT COUNT(*) FROM pages").Scan(&pageCount)
		if err != nil {
			t.Fatalf("failed to query page count: %v", err)
		}
		if pageCount < 1 {
			t.Errorf("expected at least 1 page, got %d", pageCount)
		}

		err = store.db.QueryRow("SELECT COUNT(*) FROM fields WHERE page_id = ?", "count_test").Scan(&fieldCount)
		if err != nil {
			t.Fatalf("failed to query field count: %v", err)
		}
		if fieldCount != 5 {
			t.Errorf("expected 5 fields, got %d", fieldCount)
		}
	})

	t.Run("delete_cascade_behavior", func(t *testing.T) {
		// Create a page with fields
		_, err = store.db.Exec(
			"INSERT INTO pages (id, name, type, source_html, public_token) VALUES (?, ?, ?, ?, ?)",
			"cascade_test", "Cascade Test", "screen_overlay", "", "token_cascade",
		)
		if err != nil {
			t.Fatalf("failed to insert page: %v", err)
		}

		// Insert fields
		for i := 0; i < 3; i++ {
			_, err = store.db.Exec(
				"INSERT INTO fields (page_id, name, value, sse_enabled, display_order) VALUES (?, ?, ?, ?, ?)",
				"cascade_test", fmt.Sprintf("cf_field_%d", i), "value", 1, i,
			)
			if err != nil {
				t.Fatalf("failed to insert field %d: %v", i, err)
			}
		}

		// Verify fields exist
		var fieldCount int
		err = store.db.QueryRow("SELECT COUNT(*) FROM fields WHERE page_id = ?", "cascade_test").Scan(&fieldCount)
		if err != nil {
			t.Fatalf("failed to query field count: %v", err)
		}
		if fieldCount != 3 {
			t.Fatalf("expected 3 fields before delete, got %d", fieldCount)
		}

		// Delete the page
		result, err := store.db.Exec("DELETE FROM pages WHERE id = ?", "cascade_test")
		if err != nil {
			t.Fatalf("failed to delete page: %v", err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected != 1 {
			t.Errorf("expected 1 row affected by delete, got %d", rowsAffected)
		}

		// Verify fields are cascade deleted
		err = store.db.QueryRow("SELECT COUNT(*) FROM fields WHERE page_id = ?", "cascade_test").Scan(&fieldCount)
		if err != nil {
			t.Fatalf("failed to query field count after delete: %v", err)
		}
		if fieldCount != 0 {
			t.Errorf("expected 0 fields after cascade delete, got %d", fieldCount)
		}

		// Verify page is deleted
		var pageCount int
		err = store.db.QueryRow("SELECT COUNT(*) FROM pages WHERE id = ?", "cascade_test").Scan(&pageCount)
		if err != nil {
			t.Fatalf("failed to query page count: %v", err)
		}
		if pageCount != 0 {
			t.Errorf("expected 0 pages after delete, got %d", pageCount)
		}
	})

	// Handle sql.ErrNoRows case for GetPage tests
	_, err = store.GetPage(context.Background(), "nonexistent_page_id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for nonexistent page, got: %v", err)
	}
}
