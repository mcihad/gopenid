package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gopenid/internal/config"
	"gopenid/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool}
}

func (s *Store) Seed(ctx context.Context, cfg config.Config) error {
	if !cfg.DevSeed {
		return nil
	}
	var count int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE deleted_at IS NULL`).Scan(&count); err != nil || count > 0 {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var deptID, roleID int64
	if err := tx.QueryRow(ctx, `INSERT INTO departments(name, description) VALUES($1,$2) RETURNING id`, "Platform", "Identity and platform operations").Scan(&deptID); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO roles(name, description) VALUES($1,$2) RETURNING id`, "admin", "Full administration access").Scan(&roleID); err != nil {
		return err
	}
	var userID int64
	if err := tx.QueryRow(ctx, `INSERT INTO users(email, name, password_hash, active, department_id) VALUES($1,$2,$3,true,$4) RETURNING id`, cfg.AdminEmail, "System Admin", string(hash), deptID).Scan(&userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_roles(user_id, role_id) VALUES($1,$2)`, userID, roleID); err != nil {
		return err
	}
	var clientID, clientRoleID int64
	if err := tx.QueryRow(ctx, `INSERT INTO clients(client_id, client_secret, name, redirect_uris) VALUES($1,$2,$3,$4) RETURNING id`,
		"gopen-dotnet",
		"dotnet-secret",
		"ASP.NET Core MVC Example",
		"http://localhost:5048/signin-oidc,https://localhost:7284/signin-oidc",
	).Scan(&clientID); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO client_roles(client_id, name, description) VALUES($1,$2,$3) RETURNING id`,
		clientID,
		"reader",
		"Can open the sample reader page",
	).Scan(&clientRoleID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_authorized_clients(user_id, client_id) VALUES($1,$2)`, userID, clientID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_client_roles(user_id, client_role_id) VALUES($1,$2)`, userID, clientRoleID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListDepartments(ctx context.Context) ([]domain.Department, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, created_at, updated_at, deleted_at, name, description FROM departments WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Department
	for rows.Next() {
		var row domain.Department
		if err := scanDepartment(rows, &row); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) CreateDepartment(ctx context.Context, in domain.Department) (domain.Department, error) {
	var row domain.Department
	err := s.Pool.QueryRow(ctx, `INSERT INTO departments(name, description) VALUES($1,$2) RETURNING id, created_at, updated_at, deleted_at, name, description`, in.Name, in.Description).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
	return row, err
}

func (s *Store) GetDepartment(ctx context.Context, id int64) (domain.Department, error) {
	var row domain.Department
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, name, description FROM departments WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
	return row, normalizeErr(err)
}

func (s *Store) UpdateDepartment(ctx context.Context, id int64, in domain.Department) (domain.Department, error) {
	var row domain.Department
	err := s.Pool.QueryRow(ctx, `UPDATE departments SET name=$2, description=$3, updated_at=now() WHERE id=$1 AND deleted_at IS NULL RETURNING id, created_at, updated_at, deleted_at, name, description`, id, in.Name, in.Description).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
	return row, normalizeErr(err)
}

func (s *Store) DeleteDepartment(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE departments SET deleted_at=now(), updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	return err
}

func (s *Store) ListRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, created_at, updated_at, deleted_at, name, description FROM roles WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Role
	for rows.Next() {
		var row domain.Role
		if err := scanRole(rows, &row); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) CreateRole(ctx context.Context, in domain.Role) (domain.Role, error) {
	var row domain.Role
	err := s.Pool.QueryRow(ctx, `INSERT INTO roles(name, description) VALUES($1,$2) RETURNING id, created_at, updated_at, deleted_at, name, description`, in.Name, in.Description).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
	return row, err
}

func (s *Store) GetRole(ctx context.Context, id int64) (domain.Role, error) {
	var row domain.Role
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, name, description FROM roles WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
	return row, normalizeErr(err)
}

func (s *Store) UpdateRole(ctx context.Context, id int64, in domain.Role) (domain.Role, error) {
	var row domain.Role
	err := s.Pool.QueryRow(ctx, `UPDATE roles SET name=$2, description=$3, updated_at=now() WHERE id=$1 AND deleted_at IS NULL RETURNING id, created_at, updated_at, deleted_at, name, description`, id, in.Name, in.Description).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
	return row, normalizeErr(err)
}

func (s *Store) DeleteRole(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE roles SET deleted_at=now(), updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	return err
}

func normalizeErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func scanDepartment(rows pgx.Rows, row *domain.Department) error {
	return rows.Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
}

func scanRole(rows pgx.Rows, row *domain.Role) error {
	return rows.Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.Name, &row.Description)
}

func placeholders(start int, ids []int64) string {
	parts := make([]string, len(ids))
	for i := range ids {
		parts[i] = fmt.Sprintf("$%d", start+i)
	}
	return strings.Join(parts, ",")
}
