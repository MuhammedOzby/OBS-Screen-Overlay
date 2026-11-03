package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_EmptyPath(t *testing.T) {
	_, err := Open("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestOpen_MemoryDatabase(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open memory database: %v", err)
	}
	defer store.Close()

	// Verify connection is working
	if err := store.db.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestOpen_FileDatabase(t *testing.T) {
	// Create a temporary file for the database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Verify connection is working
	if err := store.db.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}

	// Verify the database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestOpen_Reopen(t *testing.T) {
	// Create a temporary directory for the database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// First open creates the database and runs migrations
	store1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Create a page to verify schema
	_, err = store1.db.Exec("INSERT INTO pages (id, name, type, source_html, public_token) VALUES ('test-id', 'Test', 'screen_overlay', X'', 'token123')")
	if err != nil {
		t.Fatalf("failed to insert page: %v", err)
	}
	store1.Close()

	// Reopen should not fail and should preserve data
	store2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}
	defer store2.Close()

	var count int
	err = store2.db.QueryRow("SELECT COUNT(*) FROM pages").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query pages: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 page, got %d", count)
	}
}

func TestMigration_Version(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	var version int
	err = store.db.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		t.Fatalf("failed to get user_version: %v", err)
	}
	if version != 2 {
		t.Errorf("expected user_version 2, got %d", version)
	}
}

func TestMigration_TablesExist(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Verify pages table exists
	var pageCount int
	err = store.db.QueryRow("SELECT COUNT(*) FROM pages").Scan(&pageCount)
	if err != nil {
		t.Errorf("failed to query pages table: %v", err)
	}

	// Verify fields table exists
	var fieldCount int
	err = store.db.QueryRow("SELECT COUNT(*) FROM fields").Scan(&fieldCount)
	if err != nil {
		t.Errorf("failed to query fields table: %v", err)
	}

	// Verify index exists
	var indexCount int
	err = store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_fields_page_order'").Scan(&indexCount)
	if err != nil {
		t.Errorf("failed to check index: %v", err)
	}
	if indexCount != 1 {
		t.Errorf("expected index to exist")
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	_, err := Open("/nonexistent/directory/that/does/not/exist/test.db")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestStore_Close(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Errorf("close failed: %v", err)
	}

	// Closing again should not panic
	if err := store.Close(); err != nil {
		t.Errorf("second close failed: %v", err)
	}
}

func TestOpen_MultipleInserts(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Insert multiple pages
	result, err := store.db.Exec("INSERT INTO pages (id, name, type, source_html, public_token) VALUES ('id1', 'Page 1', 'screen_overlay', X'', 'token1')")
	if err != nil {
		t.Fatalf("failed to insert page: %v", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}

	// Insert fields for a page
	_, err = store.db.Exec("INSERT INTO fields (page_id, name, value, sse_enabled, display_order) VALUES ('id1', 'field1', 'value1', 1, 0)")
	if err != nil {
		t.Fatalf("failed to insert field: %v", err)
	}

	// Verify field was inserted
	var value string
	err = store.db.QueryRow("SELECT value FROM fields WHERE page_id = ? AND name = ?", "id1", "field1").Scan(&value)
	if err != nil {
		t.Fatalf("failed to query field: %v", err)
	}
	if value != "value1" {
		t.Errorf("expected value1, got %s", value)
	}
}

func TestMigration_UnsupportedVersion(t *testing.T) {
	// Create a temporary file for the database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open and create the database
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Manually set user_version to 3 to simulate unsupported migration
	_, err = store.db.Exec("PRAGMA user_version = 3")
	if err != nil {
		t.Fatalf("failed to set user_version: %v", err)
	}

	// Close the store
	store.Close()

	// Reopen should fail with ErrUnsupportedMigration
	_, err = Open(dbPath)
	if err == nil {
		t.Error("expected error for unsupported migration version")
	}
	if err != ErrUnsupportedMigration {
		t.Errorf("expected ErrUnsupportedMigration, got: %v", err)
	}
}

func TestOpen_PathsWithSpaces(t *testing.T) {
	// Create a temporary directory with a path containing spaces
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test db.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database with spaces in path: %v", err)
	}
	defer store.Close()

	// Verify connection is working
	if err := store.db.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}

	// Verify the database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}
