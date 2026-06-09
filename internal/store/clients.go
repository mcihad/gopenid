package store

import (
	"context"

	"gopenid/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListClients(ctx context.Context) ([]domain.Client, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, created_at, updated_at, deleted_at, client_id, client_secret, name, redirect_uris FROM clients WHERE deleted_at IS NULL ORDER BY client_id`)
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
		roles, err := s.ListClientRoles(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		row.Roles = roles
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) CreateClient(ctx context.Context, in domain.Client) (domain.Client, error) {
	var row domain.Client
	err := s.Pool.QueryRow(ctx, `INSERT INTO clients(client_id, client_secret, name, redirect_uris) VALUES($1,$2,$3,$4) RETURNING id, created_at, updated_at, deleted_at, client_id, client_secret, name, redirect_uris`, in.ClientID, in.ClientSecret, in.Name, in.RedirectURIs).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.ClientID, &row.ClientSecret, &row.Name, &row.RedirectURIs)
	return row, err
}

func (s *Store) GetClient(ctx context.Context, id int64) (domain.Client, error) {
	var row domain.Client
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, client_id, client_secret, name, redirect_uris FROM clients WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.ClientID, &row.ClientSecret, &row.Name, &row.RedirectURIs)
	if err != nil {
		return row, normalizeErr(err)
	}
	roles, err := s.ListClientRoles(ctx, row.ID)
	row.Roles = roles
	return row, err
}

func (s *Store) GetClientByClientID(ctx context.Context, clientID string) (domain.Client, error) {
	var row domain.Client
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, client_id, client_secret, name, redirect_uris FROM clients WHERE client_id=$1 AND deleted_at IS NULL`, clientID).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.ClientID, &row.ClientSecret, &row.Name, &row.RedirectURIs)
	if err != nil {
		return row, normalizeErr(err)
	}
	roles, err := s.ListClientRoles(ctx, row.ID)
	row.Roles = roles
	return row, err
}

func (s *Store) UpdateClient(ctx context.Context, id int64, in domain.Client) (domain.Client, error) {
	var row domain.Client
	err := s.Pool.QueryRow(ctx, `UPDATE clients SET client_id=$2, client_secret=$3, name=$4, redirect_uris=$5, updated_at=now() WHERE id=$1 AND deleted_at IS NULL RETURNING id, created_at, updated_at, deleted_at, client_id, client_secret, name, redirect_uris`, id, in.ClientID, in.ClientSecret, in.Name, in.RedirectURIs).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.ClientID, &row.ClientSecret, &row.Name, &row.RedirectURIs)
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

func scanClient(rows pgx.Rows, row *domain.Client) error {
	return rows.Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.ClientID, &row.ClientSecret, &row.Name, &row.RedirectURIs)
}

func scanClientRole(rows pgx.Rows, row *domain.ClientRole) error {
	return rows.Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.ClientID, &row.Name, &row.Description)
}
