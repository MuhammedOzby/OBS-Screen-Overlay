package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"obs-screen-overlay/internal/live"
	"obs-screen-overlay/internal/upload"
)

// PublishTemplate atomically publishes a new design version for a page and
// notifies its live subscribers. It is guarded by liveMu so the commit-and-
// publish window stays consistent with SubscribeLive's read-and-subscribe
// window (see PLAN.md §5).
//
// The source is validated with upload.ValidatePlaceholders before the live
// lock is acquired; validation is the expensive part and must not hold the
// snapshot consistency lock while running. The expected revision must match
// the stored revision or ErrConflict is returned.
//
// On success the page's source, revision and template_version are advanced
// inside a single transaction, the transaction is committed, and only then a
// live snapshot is published from the in-memory field data. No database read
// happens after the commit, and the field rows themselves are not modified, so
// the existing field list, values, SSE flags and order are preserved exactly.
func (s *Store) PublishTemplate(ctx context.Context, id string, expectedRevision int64, source string) (int64, error) {
	if err := upload.ValidatePlaceholders([]byte(source)); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if expectedRevision < 0 {
		return 0, ErrValidation
	}

	s.liveMu.Lock()
	defer s.liveMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("transaction başlatılamadı: %w", err)
	}
	defer tx.Rollback()

	var currentRevision int64
	var templateVersion int64
	if err := tx.QueryRowContext(ctx, "SELECT revision, template_version FROM pages WHERE id = ?", id).Scan(&currentRevision, &templateVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, sql.ErrNoRows
		}
		return 0, fmt.Errorf("page metadata okunamadı: %w", err)
	}

	if expectedRevision != currentRevision {
		return 0, ErrConflict
	}

	// Collect the current SSE-enabled field values for the published snapshot.
	// Only values are read; the field rows are intentionally left untouched.
	fieldsMap := make(map[string]string)
	rows, err := tx.QueryContext(ctx, "SELECT name, value FROM fields WHERE page_id = ? AND sse_enabled = 1", id)
	if err != nil {
		return 0, fmt.Errorf("fields listelenemedi: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return 0, fmt.Errorf("field scan edilemedi: %w", err)
		}
		fieldsMap[name] = value
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("fields okunamadı: %w", err)
	}

	newRevision := currentRevision + 1
	newTemplateVersion := templateVersion + 1

	result, err := tx.ExecContext(ctx,
		"UPDATE pages SET source_html = ?, revision = ?, template_version = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND revision = ?",
		source,
		newRevision,
		newTemplateVersion,
		id,
		currentRevision,
	)
	if err != nil {
		return 0, fmt.Errorf("page güncellenemedi: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("güncellenen satır sayısı alınamadı: %w", err)
	}
	if updated == 0 {
		return 0, ErrConflict
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("transaction commit edilemedi: %w", err)
	}

	s.hub.Publish(id, live.Snapshot{
		Revision:        newRevision,
		TemplateVersion: newTemplateVersion,
		Fields:          fieldsMap,
	})

	return newRevision, nil
}
