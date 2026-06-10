package store

import (
	"context"
	"strings"

	"gopenid/internal/domain"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func clientColumnsPrefixed(p string) string {
	cols := []string{"id", "created_at", "updated_at", "deleted_at", "client_id", "client_secret", "name", "description", "home_url", "logo_url", "redirect_uris", "token_ttl_seconds", "refresh_ttl_seconds", "allow_password_grant"}
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = p + "." + c
	}
	return joinComma(out)
}

const clientColumns = `id, created_at, updated_at, deleted_at, client_id, client_secret, name, description, home_url, logo_url, redirect_uris, token_ttl_seconds, refresh_ttl_seconds, allow_password_grant`

func (s *Store) ListClients(ctx context.Context) ([]domain.Client, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+clientColumns+` FROM clients WHERE deleted_at IS NULL ORDER BY client_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Client
	for rows.Next() {
		var row domain.Client
		if err := scanClient(rows, &row); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		roles, err := s.ListClientRoles(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Roles = roles
	}
	return out, nil
}

func (s *Store) CreateClient(ctx context.Context, in domain.Client) (domain.Client, error) {
	var row domain.Client
	err := s.Pool.QueryRow(ctx, `INSERT INTO clients(client_id, client_secret, name, description, home_url, logo_url, redirect_uris, token_ttl_seconds, refresh_ttl_seconds, allow_password_grant) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING `+clientColumns,
		in.ClientID, in.ClientSecret, in.Name, in.Description, in.HomeURL, in.LogoURL, in.RedirectURIs, in.TokenTTLSeconds, in.RefreshTTLSeconds, in.AllowPasswordGrant).Scan(clientScanDest(&row)...)
	normalizeClient(&row)
	return row, err
}

func (s *Store) GetClient(ctx context.Context, id int64) (domain.Client, error) {
	var row domain.Client
	err := s.Pool.QueryRow(ctx, `SELECT `+clientColumns+` FROM clients WHERE id=$1 AND deleted_at IS NULL`, id).Scan(clientScanDest(&row)...)
	if err != nil {
		return row, normalizeErr(err)
	}
	normalizeClient(&row)
	roles, err := s.ListClientRoles(ctx, row.ID)
	row.Roles = roles
	return row, err
}

func (s *Store) GetClientByClientID(ctx context.Context, clientID string) (domain.Client, error) {
	var row domain.Client
	err := s.Pool.QueryRow(ctx, `SELECT `+clientColumns+` FROM clients WHERE client_id=$1 AND deleted_at IS NULL`, clientID).Scan(clientScanDest(&row)...)
	if err != nil {
		return row, normalizeErr(err)
	}
	normalizeClient(&row)
	roles, err := s.ListClientRoles(ctx, row.ID)
	row.Roles = roles
	return row, err
}

func (s *Store) UpdateClient(ctx context.Context, id int64, in domain.Client) (domain.Client, error) {
	var row domain.Client
	err := s.Pool.QueryRow(ctx, `UPDATE clients SET client_id=$2, client_secret=COALESCE(NULLIF($3,''), client_secret), name=$4, description=$5, home_url=$6, logo_url=$7, redirect_uris=$8, token_ttl_seconds=$9, refresh_ttl_seconds=$10, allow_password_grant=$11, updated_at=now() WHERE id=$1 AND deleted_at IS NULL RETURNING `+clientColumns,
		id, in.ClientID, in.ClientSecret, in.Name, in.Description, in.HomeURL, in.LogoURL, in.RedirectURIs, in.TokenTTLSeconds, in.RefreshTTLSeconds, in.AllowPasswordGrant).Scan(clientScanDest(&row)...)
	normalizeClient(&row)
	return row, normalizeErr(err)
}

func (s *Store) DeleteClient(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE clients SET deleted_at=now(), updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	return err
}

func (s *Store) ListClientRoles(ctx context.Context, clientID int64) ([]domain.ClientRole, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, created_at, updated_at, deleted_at, client_id, name, description FROM client_roles WHERE client_id=$1 AND deleted_at IS NULL ORDER BY name`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ClientRole
	for rows.Next() {
		var row domain.ClientRole
		if err := scanClientRole(rows, &row); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) CreateClientRole(ctx context.Context, clientID int64, in domain.ClientRole) (domain.ClientRole, error) {
	var row domain.ClientRole
	err := s.Pool.QueryRow(ctx, `INSERT INTO client_roles(client_id, name, description) VALUES($1,$2,$3) RETURNING id, created_at, updated_at, deleted_at, client_id, name, description`, clientID, in.Name, in.Description).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.ClientID, &row.Name, &row.Description)
	return row, err
}

func (s *Store) UpdateClientRole(ctx context.Context, clientID, roleID int64, in domain.ClientRole) (domain.ClientRole, error) {
	var row domain.ClientRole
	err := s.Pool.QueryRow(ctx, `UPDATE client_roles SET name=$3, description=$4, updated_at=now() WHERE id=$2 AND client_id=$1 AND deleted_at IS NULL RETURNING id, created_at, updated_at, deleted_at, client_id, name, description`, clientID, roleID, in.Name, in.Description).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.ClientID, &row.Name, &row.Description)
	return row, normalizeErr(err)
}

func (s *Store) DeleteClientRole(ctx context.Context, clientID, roleID int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE client_roles SET deleted_at=now(), updated_at=now() WHERE id=$2 AND client_id=$1 AND deleted_at IS NULL`, clientID, roleID)
	return err
}

func (s *Store) UserAuthorizedForClient(ctx context.Context, userID, clientID int64) (bool, error) {
	var count int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM user_authorized_clients WHERE user_id=$1 AND client_id=$2`, userID, clientID).Scan(&count)
	return count > 0, err
}

func clientScanDest(row *domain.Client) []any {
	return []any{
		&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt,
		&row.ClientID, &row.ClientSecret, &row.Name, &row.Description,
		&row.HomeURL, &row.LogoURL, &row.RedirectURIs,
		&row.TokenTTLSeconds, &row.RefreshTTLSeconds, &row.AllowPasswordGrant,
	}
}

func scanClient(rows pgx.Rows, row *domain.Client) error {
	if err := rows.Scan(clientScanDest(row)...); err != nil {
		return err
	}
	row.HasClientSecret = row.ClientSecret != ""
	return nil
}

func scanClientRole(rows pgx.Rows, row *domain.ClientRole) error {
	return rows.Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.ClientID, &row.Name, &row.Description)
}

func normalizeClient(row *domain.Client) {
	row.HasClientSecret = row.ClientSecret != ""
}

func HashClientSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	return string(hash), err
}

func VerifyClientSecret(stored, presented string) bool {
	if stored == "" {
		return presented == ""
	}
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(presented)) == nil
	}
	return stored == presented
}
