package store

import (
	"context"
	"time"

	"gopenid/internal/domain"
)

func (s *Store) CreateRefreshToken(ctx context.Context, in domain.RefreshToken) (domain.RefreshToken, error) {
	err := s.Pool.QueryRow(ctx, `INSERT INTO refresh_tokens(token_hash, user_id, client_id, scope, expires_at) VALUES($1,$2,$3,$4,$5) RETURNING id, created_at`,
		in.TokenHash, in.UserID, in.ClientID, in.Scope, in.ExpiresAt).Scan(&in.ID, &in.CreatedAt)
	return in, err
}

func (s *Store) GetRefreshTokenByHash(ctx context.Context, hash string) (domain.RefreshToken, error) {
	var row domain.RefreshToken
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, token_hash, user_id, client_id, scope, expires_at, revoked, revoked_at FROM refresh_tokens WHERE token_hash=$1`, hash).
		Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.TokenHash, &row.UserID, &row.ClientID, &row.Scope, &row.ExpiresAt, &row.Revoked, &row.RevokedAt)
	return row, normalizeErr(err)
}

func (s *Store) RevokeRefreshTokenByHash(ctx context.Context, hash string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE refresh_tokens SET revoked=true, revoked_at=now(), updated_at=now() WHERE token_hash=$1 AND revoked=false`, hash)
	return err
}

func (s *Store) RevokeAllUserRefreshTokens(ctx context.Context, userID int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE refresh_tokens SET revoked=true, revoked_at=now(), updated_at=now() WHERE user_id=$1 AND revoked=false`, userID)
	return err
}

func (s *Store) ListRefreshTokensForUser(ctx context.Context, userID int64) ([]domain.RefreshToken, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, created_at, updated_at, deleted_at, token_hash, user_id, client_id, scope, expires_at, revoked, revoked_at FROM refresh_tokens WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RefreshToken
	for rows.Next() {
		var row domain.RefreshToken
		if err := rows.Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.TokenHash, &row.UserID, &row.ClientID, &row.Scope, &row.ExpiresAt, &row.Revoked, &row.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// RevokeAccessToken blacklists an access token by its JTI until it expires.
func (s *Store) RevokeAccessToken(ctx context.Context, jti string, userID int64, reason string, expiresAt time.Time) error {
	if jti == "" {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `INSERT INTO revoked_tokens(jti, user_id, reason, expires_at) VALUES($1,$2,$3,$4)`, jti, userID, reason, expiresAt)
	return err
}

// IsAccessTokenRevoked reports whether an access token JTI has been revoked.
func (s *Store) IsAccessTokenRevoked(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	var count int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM revoked_tokens WHERE jti=$1`, jti).Scan(&count)
	return count > 0, err
}

// CleanupExpired removes expired refresh tokens, revoked-token records and
// stale authorization codes. Returns the number of rows removed in total.
func (s *Store) CleanupExpired(ctx context.Context) (int64, error) {
	var total int64
	statements := []string{
		`DELETE FROM refresh_tokens WHERE expires_at < now()`,
		`DELETE FROM revoked_tokens WHERE expires_at < now()`,
		`DELETE FROM auth_codes WHERE used = true OR created_at < now() - interval '10 minutes'`,
	}
	for _, q := range statements {
		tag, err := s.Pool.Exec(ctx, q)
		if err != nil {
			return total, err
		}
		total += tag.RowsAffected()
	}
	return total, nil
}
