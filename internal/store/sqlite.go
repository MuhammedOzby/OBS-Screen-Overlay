package store

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"sync"

	"obs-screen-overlay/internal/live"

	_ "github.com/mattn/go-sqlite3"
)

// Store represents a SQLite database connection for the OBS Screen Overlay.
type Store struct {
	db *sql.DB

	// liveMu serializes snapshot-read and subscribe per page so no change
	// is lost between the read and the subscription (see PLAN.md).
	liveMu sync.Mutex
	// hub is the in-memory subscriber hub for live (SSE) snapshots.
	hub *live.Hub
}

// Open creates a new Store with a connection to the SQLite database at the given path.
// It rejects empty paths. A path of ":memory:" creates an in-memory database.
// The connection is configured with foreign keys, busy timeout, and WAL journal mode.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("veritabanı yolu boş olamaz")
	}

	// Use filepath.Abs for file paths, but keep :memory: as-is for in-memory databases
	var absPath string
	if path == ":memory:" {
		absPath = path
	} else {
		var err error
		absPath, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
	}

	// Build DSN with configuration options
	dsn := buildDSN(absPath)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Configure connection
	db.SetMaxOpenConns(1)

	store := &Store{
		db:  db,
		hub: live.NewHub(16),
	}

	// Ping to verify connection
	if err := store.db.Ping(); err != nil {
		store.db.Close()
		return nil, err
	}

	// Run migrations
	if err := store.migrate(); err != nil {
		store.db.Close()
		return nil, err
	}

	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// buildDSN creates a SQLite DSN with the specified path and configuration options.
func buildDSN(path string) string {
	// :memory: databases don't need URI escaping
	if path == ":memory:" {
		return ":memory:" + "?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL"
	}

	// Use url.URL for proper URI escaping of the path
	u := url.URL{
		Scheme: "file",
		Path:   path,
	}

	// Add DSN parameters
	return u.String() + "?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL"
}

// migrate runs database migrations. It checks PRAGMA user_version:
// - 0: fresh database - creates pages, fields and access_sessions, then sets user_version to 2
// - 1: existing v1 database - preserves pages and fields, creates access_sessions, then sets user_version to 2
// - 2: current schema version - nothing to do
// - >2: returns ErrUnsupportedMigration
func (s *Store) migrate() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var version int
	err = tx.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		return err
	}

	switch version {
	case 0:
		if err := s.createBaseSchema(tx); err != nil {
			return err
		}
		if err := s.createAccessSessions(tx); err != nil {
			return err
		}
		if err := s.setUserVersion(tx, 2); err != nil {
			return err
		}
	case 1:
		// Pages and fields already exist and must be preserved.
		if err := s.createAccessSessions(tx); err != nil {
			return err
		}
		if err := s.setUserVersion(tx, 2); err != nil {
			return err
		}
	case 2:
		// Current schema version - nothing to do.
	default:
		// Version > 2 is not supported by this binary.
		return ErrUnsupportedMigration
	}

	return tx.Commit()
}

// createBaseSchema creates the pages and fields tables and the fields
// ordering index. It does not touch PRAGMA user_version.
func (s *Store) createBaseSchema(tx *sql.Tx) error {
	// Create pages table
	_, err := tx.Exec(`
		CREATE TABLE pages (
			id TEXT PRIMARY KEY NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('screen_overlay','widget')),
			source_html TEXT NOT NULL,
			public_token TEXT NOT NULL UNIQUE,
			revision INTEGER NOT NULL DEFAULT 1 CHECK(revision>=1),
			template_version INTEGER NOT NULL DEFAULT 1 CHECK(template_version>=1),
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create fields table
	_, err = tx.Exec(`
		CREATE TABLE fields (
			page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			value TEXT NOT NULL DEFAULT '',
			sse_enabled INTEGER NOT NULL DEFAULT 1 CHECK(sse_enabled IN (0,1)),
			display_order INTEGER NOT NULL DEFAULT 0 CHECK(display_order>=0),
			PRIMARY KEY (page_id, name)
		)
	`)
	if err != nil {
		return err
	}

	// Create index on page_id and display_order for fields table
	_, err = tx.Exec(`
		CREATE INDEX idx_fields_page_order ON fields(page_id, display_order)
	`)
	if err != nil {
		return err
	}

	return nil
}

// createAccessSessions creates the access_sessions table for device-level
// authentication tokens. The token_hash column stores the hash of the secret
// access token; the raw token is never persisted. Uses IF NOT EXISTS to be
// idempotent.
func (s *Store) createAccessSessions(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS access_sessions (
			token_hash TEXT PRIMARY KEY NOT NULL,
			csrf_token TEXT NOT NULL,
			device_name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL
		)
	`)
	return err
}

// setUserVersion sets PRAGMA user_version to the given schema version.
func (s *Store) setUserVersion(tx *sql.Tx, version int) error {
	if version <= 0 {
		return fmt.Errorf("geçersiz şema sürümü: %d", version)
	}
	_, err := tx.Exec("PRAGMA user_version = " + strconv.Itoa(version))
	return err
}

// ErrUnsupportedMigration is returned when the database version is greater than expected.
var ErrUnsupportedMigration = errors.New("desteklenmeyen veritabanı şema sürümü")
