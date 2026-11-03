package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"obs-screen-overlay/internal/live"
)

// ErrValidation is returned when any input to an update operation fails validation.
var ErrValidation = errors.New("geçersiz girdi")

// ErrConflict is returned when the expected revision does not match the stored revision.
var ErrConflict = errors.New("revizyon çatışması")

// UpdatePage atomically updates a page's name, type and fields within a single
// transaction, guarded by optimistic concurrency control.
//
// name is trimmed; it must be non-empty. typ must be one of the accepted page
// types. Field names must match the field name pattern and must not be
// duplicated. expectedRevision must not be negative.
//
// The page row is updated only when its stored revision equals
// expectedRevision; revision is then incremented by one and updated_at is
// refreshed. source_html, public_token and template_version are left unchanged.
// The page's fields are deleted and re-inserted from the slice, whose order
// determines display order.
//
// Returns the new revision (expectedRevision+1) on success. If the page does
// not exist it returns sql.ErrNoRows; if the page exists but its revision does
// not match it returns ErrConflict. All failures roll back the transaction
// and no live snapshot is published.
//
// On a successful commit the store publishes a live snapshot for the page via
// the in-memory hub: Revision is expectedRevision+1, TemplateVersion is the
// stored template version read inside the transaction, and Fields holds the
// latest values of the SSE-enabled fields of the given slice. Publishing is
// serialized with SubscribeLive by holding liveMu for the entire call; no
// database read is performed after the commit.
func (s *Store) UpdatePage(ctx context.Context, id string, expectedRevision int64, name, typ string, fields []Field) (int64, error) {
	// liveMu keeps the commit-and-publish window consistent with
	// SubscribeLive's read-and-subscribe window, so no committed change
	// is lost between reading the snapshot and subscribing to it.
	s.liveMu.Lock()
	defer s.liveMu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("%w: name boş olamaz", ErrValidation)
	}
	if !pageTypeWhitelist[typ] {
		return 0, fmt.Errorf("%w: geçersiz type: %s", ErrValidation, typ)
	}
	if expectedRevision < 0 {
		return 0, fmt.Errorf("%w: beklenen revision negatif olamaz: %d", ErrValidation, expectedRevision)
	}
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		if !nameRegex.MatchString(field.Name) {
			return 0, fmt.Errorf("%w: field name geçersiz formatta: %s", ErrValidation, field.Name)
		}
		if seen[field.Name] {
			return 0, fmt.Errorf("%w: tekrarlanan field name: %s", ErrValidation, field.Name)
		}
		seen[field.Name] = true
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("transaction başlatılamadı: %w", err)
	}
	defer tx.Rollback()

	// Read the stored template version inside the transaction so the post-commit
	// snapshot carries the same generation without any database read after commit.
	var templateVersion int64
	switch err := tx.QueryRowContext(ctx, "SELECT template_version FROM pages WHERE id = ?", id).Scan(&templateVersion); {
	case errors.Is(err, sql.ErrNoRows):
		return 0, sql.ErrNoRows
	case err != nil:
		return 0, fmt.Errorf("template_version okunamadı: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		"UPDATE pages SET name = ?, type = ?, revision = revision + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND revision = ?",
		name, typ, id, expectedRevision,
	)
	if err != nil {
		return 0, fmt.Errorf("page güncellenemedi: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var exists int
		err := tx.QueryRowContext(ctx, "SELECT 1 FROM pages WHERE id = ?", id).Scan(&exists)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return 0, sql.ErrNoRows
		case err != nil:
			return 0, fmt.Errorf("page varlığı kontrol edilemedi: %w", err)
		default:
			return 0, ErrConflict
		}
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM fields WHERE page_id = ?", id); err != nil {
		return 0, fmt.Errorf("fields silinemedi: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx,
		"INSERT INTO fields (page_id, name, value, sse_enabled, display_order) VALUES (?, ?, ?, ?, ?)",
	)
	if err != nil {
		return 0, fmt.Errorf("field prepared statement'i hazırlanamadı: %w", err)
	}
	defer stmt.Close()

	for i, field := range fields {
		sseEnabled := 0
		if field.SSEEnabled {
			sseEnabled = 1
		}
		if _, err := stmt.ExecContext(ctx, id, field.Name, field.Value, sseEnabled, i); err != nil {
			return 0, fmt.Errorf("field insert edilemedi: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("transaction commit edilemedi: %w", err)
	}

	// Publish only after a successful commit, from the in-memory field data:
	// SSE-enabled fields only, never a database re-read.
	if s.hub != nil {
		snapFields := make(map[string]string)
		for _, field := range fields {
			if field.SSEEnabled {
				snapFields[field.Name] = field.Value
			}
		}
		s.hub.Publish(id, live.Snapshot{
			Revision:        expectedRevision + 1,
			TemplateVersion: templateVersion,
			Fields:          snapFields,
		})
	}

	return expectedRevision + 1, nil
}
