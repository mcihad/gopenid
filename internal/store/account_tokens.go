package store

import (
	"context"
	"time"

	"gopenid/internal/domain"
)

func (s *Store) CreateAccountToken(ctx context.Context, token domain.AccountToken) (domain.AccountToken, error) {
	err := s.Pool.QueryRow(ctx, `INSERT INTO account_tokens(user_id, token_hash, type, expires_at) VALUES($1,$2,$3,$4) RETURNING id, created_at, updated_at, deleted_at`,
		token.UserID, token.TokenHash, token.Type, token.ExpiresAt).Scan(&token.ID, &token.CreatedAt, &token.UpdatedAt, &token.DeletedAt)
	return token, err
}

func (s *Store) ConsumeAccountToken(ctx context.Context, tokenHash, tokenType string) (domain.AccountToken, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return domain.AccountToken{}, err
	}
	defer tx.Rollback(ctx)
	var row domain.AccountToken
	err = tx.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, user_id, token_hash, type, expires_at, used_at FROM account_tokens WHERE token_hash=$1 AND type=$2 AND used_at IS NULL AND deleted_at IS NULL`,
		tokenHash, tokenType).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.UserID, &row.TokenHash, &row.Type, &row.ExpiresAt, &row.UsedAt)
	if err != nil {
		return row, normalizeErr(err)
	}
	if time.Now().After(row.ExpiresAt) {
		return row, ErrNotFound
	}
	now := time.Now()
	if _, err := tx.Exec(ctx, `UPDATE account_tokens SET used_at=$2, updated_at=$2 WHERE id=$1`, row.ID, now); err != nil {
		return row, err
	}
	row.UsedAt = &now
	if err := tx.Commit(ctx); err != nil {
		return row, err
	}
	return row, nil
}
