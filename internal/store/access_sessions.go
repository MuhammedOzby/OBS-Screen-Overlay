package store

import (
	"context"
	"errors"
	"time"
)

// AccessSession is a device session. TokenHash is supplied by the caller;
// the raw access token must not be passed to the store.
type AccessSession struct {
	TokenHash, CSRFToken, DeviceName string
	CreatedAt, ExpiresAt             time.Time
}

// CreateAccessSession stores a session with timestamps in Unix seconds.
func (s *Store) CreateAccessSession(ctx context.Context, tokenHash, csrfToken, deviceName string, expiresAt time.Time) error {
	now := time.Now()
	if tokenHash == "" || csrfToken == "" || deviceName == "" || expiresAt.Unix() <= now.Unix() {
		return errors.New("oturum alanları boş veya son kullanma zamanı geçersiz")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO access_sessions
		(token_hash, csrf_token, device_name, created_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
		tokenHash, csrfToken, deviceName, now.Unix(), expiresAt.Unix())
	return err
}

type sessionScanner interface {
	Scan(dest ...any) error
}

func scanAccessSession(row sessionScanner) (AccessSession, error) {
	var session AccessSession
	var created, expires int64
	if err := row.Scan(&session.TokenHash, &session.CSRFToken, &session.DeviceName, &created, &expires); err != nil {
		return AccessSession{}, err
	}
	session.CreatedAt = time.Unix(created, 0).UTC()
	session.ExpiresAt = time.Unix(expires, 0).UTC()
	return session, nil
}

// GetAccessSession returns sql.ErrNoRows for missing or expired sessions.
func (s *Store) GetAccessSession(ctx context.Context, tokenHash string) (AccessSession, error) {
	return scanAccessSession(s.db.QueryRowContext(ctx, `SELECT token_hash, csrf_token, device_name, created_at, expires_at
		FROM access_sessions WHERE token_hash = ? AND expires_at > ?`, tokenHash, time.Now().Unix()))
}

// ListAccessSessions returns only unexpired sessions.
func (s *Store) ListAccessSessions(ctx context.Context) ([]AccessSession, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT token_hash, csrf_token, device_name, created_at, expires_at
		FROM access_sessions WHERE expires_at > ? ORDER BY created_at, token_hash`, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sessions := []AccessSession{}
	for rows.Next() {
		session, err := scanAccessSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

// DeleteAccessSession revokes a session, including an expired one.
func (s *Store) DeleteAccessSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM access_sessions WHERE token_hash = ?", tokenHash)
	return err
}

// DeleteAllAccessSessions revokes all sessions, including expired ones.
func (s *Store) DeleteAllAccessSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM access_sessions")
	return err
}
