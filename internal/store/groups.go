package store

import (
	"context"

	"gopenid/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListGroups(ctx context.Context) ([]domain.Group, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, created_at, updated_at, deleted_at, name, description FROM groups WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Group
	for rows.Next() {
		var row domain.Group
		if err := scanGroup(rows, &row); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) CreateGroup(ctx context.Context, in domain.Group) (domain.Group, error) {
	var row domain.Group
	err := s.Pool.QueryRow(ctx, `INSERT INTO groups(name, description) VALUES($1,$2) RETURNING id, created_at, updated_at, deleted_at, name, description`, in.Name, in.Description).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
	return row, err
}

func (s *Store) GetGroup(ctx context.Context, id int64) (domain.Group, error) {
	var row domain.Group
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, name, description FROM groups WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
	return row, normalizeErr(err)
}

func (s *Store) UpdateGroup(ctx context.Context, id int64, in domain.Group) (domain.Group, error) {
	var row domain.Group
	err := s.Pool.QueryRow(ctx, `UPDATE groups SET name=$2, description=$3, updated_at=now() WHERE id=$1 AND deleted_at IS NULL RETURNING id, created_at, updated_at, deleted_at, name, description`, id, in.Name, in.Description).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
	return row, normalizeErr(err)
}

func (s *Store) DeleteGroup(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE groups SET deleted_at=now(), updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	return err
}

// GroupIDsForUser returns the group ids a user belongs to.
func (s *Store) GroupIDsForUser(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := s.Pool.Query(ctx, `SELECT group_id FROM user_groups WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func scanGroup(rows pgx.Rows, row *domain.Group) error {
	return rows.Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
}
