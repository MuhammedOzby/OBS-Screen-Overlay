package store

import (
	"context"
	"database/sql"
	"testing"
)

func TestCreatePage_EmptyID(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	page := Page{
		ID:          "",
		Name:        "test",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
	}

	err = store.CreatePage(context.Background(), page)
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestCreatePage_EmptyToken(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	page := Page{
		ID:          "test-id",
		Name:        "test",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "",
	}

	err = store.CreatePage(context.Background(), page)
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestCreatePage_EmptyName(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	page := Page{
		ID:          "test-id",
		Name:        "",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
	}

	err = store.CreatePage(context.Background(), page)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreatePage_InvalidName(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Page names are now free-form, only empty validation is done
	// "invalid-name!" should be accepted as a valid page name
	page := Page{
		ID:          "test-id",
		Name:        "invalid-name!",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
	}

	err = store.CreatePage(context.Background(), page)
	if err != nil {
		t.Errorf("expected no error for page name with special characters, got: %v", err)
	}
}

func TestCreatePage_InvalidFieldName(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	page := Page{
		ID:          "test-id",
		Name:        "valid_name",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
		Fields: []Field{
			{Name: "invalid field name"},
		},
	}

	err = store.CreatePage(context.Background(), page)
	if err == nil {
		t.Error("expected error for invalid field name")
	}
}

func TestCreatePage_Success(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	page := Page{
		ID:          "test-id",
		Name:        "valid_name",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
		Fields: []Field{
			{Name: "field1", Value: "value1", SSEEnabled: true, DisplayOrder: 0},
			{Name: "field2", Value: "value2", SSEEnabled: false, DisplayOrder: 1},
		},
	}

	err = store.CreatePage(context.Background(), page)
	if err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	// Verify page was created
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM pages WHERE id = ?", "test-id").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query page: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 page, got %d", count)
	}
}

func TestCreatePage_EmptyFields(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	page := Page{
		ID:          "test-id",
		Name:        "valid_name",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
		Fields:      []Field{},
	}

	err = store.CreatePage(context.Background(), page)
	if err != nil {
		t.Fatalf("failed to create page with empty fields: %v", err)
	}
}

func TestCreatePage_DuplicateFieldName(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// First insert a page successfully
	page := Page{
		ID:          "test-id",
		Name:        "valid_name",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
		Fields: []Field{
			{Name: "field1", Value: "value1"},
		},
	}

	err = store.CreatePage(context.Background(), page)
	if err != nil {
		t.Fatalf("failed to create first page: %v", err)
	}

	// Try to insert another field with the same name for the same page (should succeed if different page_id)
	page2 := Page{
		ID:          "test-id2",
		Name:        "valid_name2",
		Type:        "widget",
		SourceHTML:  "<html></html>",
		PublicToken: "token456",
		Fields: []Field{
			{Name: "field1", Value: "value1"},
		},
	}

	err = store.CreatePage(context.Background(), page2)
	if err != nil {
		t.Fatalf("failed to create second page: %v", err)
	}
}

func TestGetPage_NotFound(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	_, err = store.GetPage(context.Background(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got: %v", err)
	}
}

func TestGetPage_Success(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Create a page
	page := Page{
		ID:              "test-id",
		Name:            "valid_name",
		Type:            "screen_overlay",
		SourceHTML:      "<html></html>",
		PublicToken:     "token123",
		TemplateVersion: 5,
		Fields: []Field{
			{Name: "field1", Value: "value1", SSEEnabled: true, DisplayOrder: 0},
			{Name: "field2", Value: "value2", SSEEnabled: false, DisplayOrder: 1},
		},
	}

	err = store.CreatePage(context.Background(), page)
	if err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	// Get the page
	retrieved, err := store.GetPage(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("failed to get page: %v", err)
	}

	if retrieved.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", retrieved.ID)
	}
	if retrieved.Name != "valid_name" {
		t.Errorf("expected Name 'valid_name', got '%s'", retrieved.Name)
	}
	if retrieved.Type != "screen_overlay" {
		t.Errorf("expected Type 'screen_overlay', got '%s'", retrieved.Type)
	}
	if retrieved.PublicToken != "token123" {
		t.Errorf("expected PublicToken 'token123', got '%s'", retrieved.PublicToken)
	}
	if len(retrieved.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(retrieved.Fields))
	}

	// Verify fields are ordered by display_order
	if retrieved.Fields[0].Name != "field1" {
		t.Errorf("expected first field 'field1', got '%s'", retrieved.Fields[0].Name)
	}
	if retrieved.Fields[1].Name != "field2" {
		t.Errorf("expected second field 'field2', got '%s'", retrieved.Fields[1].Name)
	}

	// Verify SSEEnabled
	if !retrieved.Fields[0].SSEEnabled {
		t.Error("expected field1 SSEEnabled to be true")
	}
	if retrieved.Fields[1].SSEEnabled {
		t.Error("expected field2 SSEEnabled to be false")
	}
}

func TestGetPage_FieldsOrdered(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Create a page with fields not in order
	page := Page{
		ID:          "test-id",
		Name:        "valid_name",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
		Fields: []Field{
			{Name: "zfield", Value: "value_z", DisplayOrder: 2},
			{Name: "afield", Value: "value_a", DisplayOrder: 0},
			{Name: "mfield", Value: "value_m", DisplayOrder: 1},
		},
	}

	err = store.CreatePage(context.Background(), page)
	if err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	// Get the page and verify fields are ordered
	retrieved, err := store.GetPage(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("failed to get page: %v", err)
	}

	// Fields should be ordered by display_order
	expectedOrder := []string{"afield", "mfield", "zfield"}
	for i, field := range retrieved.Fields {
		if field.Name != expectedOrder[i] {
			t.Errorf("field at position %d: expected '%s', got '%s'", i, expectedOrder[i], field.Name)
		}
	}
}

func TestCreatePage_RollbackOnFieldError(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Create a page with multiple fields that include a duplicate field name
	// This will trigger a UNIQUE constraint violation on fields(page_id, name)
	page := Page{
		ID:          "test-id",
		Name:        "valid_name",
		Type:        "screen_overlay",
		SourceHTML:  "<html></html>",
		PublicToken: "token123",
		Fields: []Field{
			{Name: "field1", Value: "value1", DisplayOrder: 0},
			{Name: "field1", Value: "value2", DisplayOrder: 1}, // Duplicate field name within same page
		},
	}

	// CreatePage should fail because of the duplicate field name
	err = store.CreatePage(context.Background(), page)
	if err == nil {
		t.Error("expected error for duplicate field name within same page")
	}

	// Verify the page was not created (transaction was rolled back)
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM pages WHERE id = ?", "test-id").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query pages: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 pages after rollback, got %d", count)
	}

	// Verify no fields were inserted (transaction was rolled back)
	err = store.db.QueryRow("SELECT COUNT(*) FROM fields WHERE page_id = ?", "test-id").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query fields: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 fields after rollback, got %d", count)
	}
}
