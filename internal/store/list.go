package store

import (
	"context"
	"fmt"
)

// pageTypeWhitelist holds the page types accepted by ListPages as a filter.
var pageTypeWhitelist = map[string]bool{
	"screen_overlay": true,
	"widget":         true,
}

// ListPages lists pages in the database. An empty pageType returns all pages;
// "screen_overlay" and "widget" return only pages of that type. Any other
// value returns an error. Pages are ordered by created_at DESC, id DESC for
// deterministic output. Fields are not loaded.
func (s *Store) ListPages(ctx context.Context, pageType string) ([]Page, error) {
	var args []any

	switch {
	case pageType == "":
		// return all pages
	case pageTypeWhitelist[pageType]:
		args = append(args, pageType)
	default:
		return nil, fmt.Errorf("geçersiz page type: %s", pageType)
	}

	query := "SELECT id, name, type, revision, template_version, created_at, updated_at FROM pages"
	if len(args) > 0 {
		query += " WHERE type = ?"
	}
	query += " ORDER BY created_at DESC, id DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pages listelenemedi: %w", err)
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var page Page
		if err := rows.Scan(
			&page.ID, &page.Name, &page.Type,
			&page.Revision, &page.TemplateVersion, &page.CreatedAt, &page.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("page scan edilemedi: %w", err)
		}
		pages = append(pages, page)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows okunamadı: %w", err)
	}

	return pages, nil
}
