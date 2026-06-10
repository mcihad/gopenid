package store

import (
	"context"
	"time"

	"gopenid/internal/domain"
)

func (s *Store) CreateAuthCode(ctx context.Context, code domain.AuthCode) error {
	err := s.Pool.QueryRow(ctx, `INSERT INTO auth_codes(code, user_id, client_id, redirect_uri, scope, nonce, code_challenge, code_challenge_method, used) VALUES($1,$2,$3,$4,$5,$6,$7,$8,false) RETURNING id, created_at`, code.Code, code.UserID, code.ClientID, code.RedirectURI, code.Scope, code.Nonce, code.CodeChallenge, code.CodeChallengeMethod).Scan(&code.ID, &code.CreatedAt)
	return err
}

func (s *Store) GetUnusedAuthCode(ctx context.Context, codeText string) (domain.AuthCode, error) {
	var code domain.AuthCode
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, code, user_id, client_id, redirect_uri, scope, nonce, code_challenge, code_challenge_method, used FROM auth_codes WHERE code=$1 AND used=false AND deleted_at IS NULL`, codeText).Scan(&code.ID, &code.CreatedAt, &code.UpdatedAt, &code.DeletedAt, &code.Code, &code.UserID, &code.ClientID, &code.RedirectURI, &code.Scope, &code.Nonce, &code.CodeChallenge, &code.CodeChallengeMethod, &code.Used)
	return code, normalizeErr(err)
}

func (s *Store) MarkAuthCodeUsed(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE auth_codes SET used=true, updated_at=$2 WHERE id=$1`, id, time.Now())
	return err
}

func (s *Store) GetSigningKey(ctx context.Context, keyID string) (domain.SigningKey, error) {
	var key domain.SigningKey
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, key_id, private_pem, active FROM signing_keys WHERE key_id=$1 AND active=true AND deleted_at IS NULL`, keyID).Scan(&key.ID, &key.CreatedAt, &key.UpdatedAt, &key.DeletedAt, &key.KeyID, &key.PrivatePEM, &key.Active)
	return key, normalizeErr(err)
}

func (s *Store) ListSigningKeys(ctx context.Context) ([]domain.SigningKey, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, created_at, updated_at, deleted_at, key_id, private_pem, active FROM signing_keys WHERE active=true AND deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SigningKey
	for rows.Next() {
		var key domain.SigningKey
		if err := rows.Scan(&key.ID, &key.CreatedAt, &key.UpdatedAt, &key.DeletedAt, &key.KeyID, &key.PrivatePEM, &key.Active); err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

func (s *Store) CreateSigningKey(ctx context.Context, key domain.SigningKey) error {
	return s.Pool.QueryRow(ctx, `INSERT INTO signing_keys(key_id, private_pem, active) VALUES($1,$2,$3) RETURNING id`, key.KeyID, key.PrivatePEM, key.Active).Scan(&key.ID)
}
