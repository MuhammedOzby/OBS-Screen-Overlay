package store

import (
	"context"
	"database/sql"
	"fmt"
)

// DeletePage removes a page and, through the foreign key cascade, its fields.
// It executes a single DELETE statement and reports sql.ErrNoRows when no
// page with the given ID exists.
func (s *Store) DeletePage(ctx context.Context, id string) error {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	res, err := s.db.ExecContext(ctx, "DELETE FROM pages WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("page silinemedi: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("etkilenen satır sayısı alınamadı: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}

	s.hub.ClosePage(id)
	return nil
}
