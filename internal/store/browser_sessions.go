package store

import (
	"context"
	"time"

	"gopenid/internal/domain"
)

func (s *Store) CreateBrowserSession(ctx context.Context, row domain.BrowserSession) error {
	return s.Pool.QueryRow(ctx, `INSERT INTO browser_sessions(token_hash, user_id, auth_time, expires_at, revoked) VALUES($1,$2,$3,$4,false) RETURNING id`,
		row.TokenHash, row.UserID, row.AuthTime, row.ExpiresAt).Scan(&row.ID)
}

func (s *Store) GetBrowserSession(ctx context.Context, tokenHash string) (domain.BrowserSession, error) {
	var row domain.BrowserSession
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, token_hash, user_id, auth_time, expires_at, revoked FROM browser_sessions WHERE token_hash=$1 AND deleted_at IS NULL`,
		tokenHash).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.TokenHash, &row.UserID, &row.AuthTime, &row.ExpiresAt, &row.Revoked)
	if err != nil {
		return row, normalizeErr(err)
	}
	if row.Revoked || time.Now().After(row.ExpiresAt) {
		return row, ErrNotFound
	}
	return row, nil
}

func (s *Store) RevokeBrowserSession(ctx context.Context, tokenHash string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE browser_sessions SET revoked=true, updated_at=now() WHERE token_hash=$1`, tokenHash)
	return err
}
