package store

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
)

// Page represents a page in the OBS Screen Overlay system.
type Page struct {
	ID              string
	Name            string
	Type            string
	SourceHTML      string
	PublicToken     string
	Revision        int64
	TemplateVersion int64
	CreatedAt       string
	UpdatedAt       string
	Fields          []Field
}

// Field represents a field associated with a page.
type Field struct {
	Name         string
	Value        string
	SSEEnabled   bool
	DisplayOrder int
}

// nameRegex validates page and field names.
var nameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// CreatePage creates a new page in the database.
// It validates the page and field names, and inserts the page with its fields in a transaction.
// If any step fails, the transaction is rolled back and no data is committed.
func (s *Store) CreatePage(ctx context.Context, p Page) error {
	// Validate ID is not empty
	if p.ID == "" {
		return fmt.Errorf("id boş olamaz")
	}

	// Validate PublicToken is not empty
	if p.PublicToken == "" {
		return fmt.Errorf("public_token boş olamaz")
	}

	// Validate Name is not empty
	if p.Name == "" {
		return fmt.Errorf("name boş olamaz")
	}

	// Validate field names
	for _, field := range p.Fields {
		if !nameRegex.MatchString(field.Name) {
			return fmt.Errorf("field name geçersiz formatta: %s", field.Name)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("transaction başlatılamadı: %w", err)
	}
	defer tx.Rollback()

	// Insert page
	_, err = tx.ExecContext(ctx,
		"INSERT INTO pages (id, name, type, source_html, public_token) VALUES (?, ?, ?, ?, ?)",
		p.ID, p.Name, p.Type, p.SourceHTML, p.PublicToken,
	)
	if err != nil {
		return fmt.Errorf("page insert edilemedi: %w", err)
	}

	// Insert fields one by one
	stmt, err := tx.PrepareContext(ctx,
		"INSERT INTO fields (page_id, name, value, sse_enabled, display_order) VALUES (?, ?, ?, ?, ?)",
	)
	if err != nil {
		return fmt.Errorf("field prepared statement'i hazırlanamadı: %w", err)
	}
	defer stmt.Close()

	for _, field := range p.Fields {
		sseEnabled := 0
		if field.SSEEnabled {
			sseEnabled = 1
		}
		_, err := stmt.ExecContext(ctx, p.ID, field.Name, field.Value, sseEnabled, field.DisplayOrder)
		if err != nil {
			return fmt.Errorf("field insert edilemedi: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("transaction commit edilemedi: %w", err)
	}

	return nil
}

// GetPage retrieves a page by its ID along with all its fields.
// It uses a transaction for a consistent snapshot.
// Returns sql.ErrNoRows if the page does not exist.
func (s *Store) GetPage(ctx context.Context, id string) (Page, error) {
	return s.getPage(ctx, "id", id)
}

// GetPageByPublicToken retrieves the page and all fields in one snapshot.
func (s *Store) GetPageByPublicToken(ctx context.Context, token string) (Page, error) {
	return s.getPage(ctx, "public_token", token)
}

func (s *Store) getPage(ctx context.Context, column, value string) (Page, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Page{}, fmt.Errorf("transaction başlatılamadı: %w", err)
	}
	defer tx.Rollback()

	var page Page
	err = tx.QueryRowContext(ctx,
		"SELECT id, name, type, source_html, public_token, revision, template_version, created_at, updated_at FROM pages WHERE "+column+" = ?",
		value,
	).Scan(&page.ID, &page.Name, &page.Type, &page.SourceHTML, &page.PublicToken, &page.Revision, &page.TemplateVersion, &page.CreatedAt, &page.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return Page{}, sql.ErrNoRows
		}
		return Page{}, fmt.Errorf("page getirilemedi: %w", err)
	}

	// Query fields ordered by display_order, then by name
	rows, err := tx.QueryContext(ctx,
		"SELECT name, value, sse_enabled, display_order FROM fields WHERE page_id = ? ORDER BY display_order, name",
		page.ID,
	)
	if err != nil {
		return Page{}, fmt.Errorf("fields getirilemedi: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var field Field
		var sseEnabled int
		err := rows.Scan(&field.Name, &field.Value, &sseEnabled, &field.DisplayOrder)
		if err != nil {
			return Page{}, fmt.Errorf("field scan edilemedi: %w", err)
		}
		field.SSEEnabled = sseEnabled != 0
		page.Fields = append(page.Fields, field)
	}

	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("rows okunamadı: %w", err)
	}

	if err := rows.Close(); err != nil {
		return Page{}, err
	}

	if err := tx.Commit(); err != nil {
		return Page{}, fmt.Errorf("transaction commit edilemedi: %w", err)
	}

	return page, nil
}
