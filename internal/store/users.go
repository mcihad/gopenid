package store

import (
	"context"

	"gopenid/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, created_at, updated_at, deleted_at, email, name, password_hash, active, department_id FROM users WHERE deleted_at IS NULL ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		var user domain.User
		if err := scanUser(rows, &user); err != nil {
			return nil, err
		}
		if err := s.LoadUserRelations(ctx, &user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) GetUser(ctx context.Context, id int64) (domain.User, error) {
	var user domain.User
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, email, name, password_hash, active, department_id FROM users WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.Email, &user.Name, &user.PasswordHash, &user.Active, &user.DepartmentID)
	if err != nil {
		return user, normalizeErr(err)
	}
	return user, s.LoadUserRelations(ctx, &user)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User
	err := s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, email, name, password_hash, active, department_id FROM users WHERE email=$1 AND active=true AND deleted_at IS NULL`, email).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.Email, &user.Name, &user.PasswordHash, &user.Active, &user.DepartmentID)
	if err != nil {
		return user, normalizeErr(err)
	}
	return user, s.LoadUserRelations(ctx, &user)
}

func (s *Store) CreateUser(ctx context.Context, user domain.User, roleIDs, clientIDs, clientRoleIDs []int64) (domain.User, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return user, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `INSERT INTO users(email, name, password_hash, active, department_id) VALUES($1,$2,$3,$4,$5) RETURNING id`, user.Email, user.Name, user.PasswordHash, user.Active, user.DepartmentID).Scan(&user.ID)
	if err != nil {
		return user, err
	}
	if err := replaceUserRelations(ctx, tx, user.ID, roleIDs, clientIDs, clientRoleIDs); err != nil {
		return user, err
	}
	if err := tx.Commit(ctx); err != nil {
		return user, err
	}
	return s.GetUser(ctx, user.ID)
}

func (s *Store) UpdateUser(ctx context.Context, id int64, user domain.User, roleIDs, clientIDs, clientRoleIDs []int64) (domain.User, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return user, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE users SET email=$2, name=$3, password_hash=COALESCE(NULLIF($4,''), password_hash), active=$5, department_id=$6, updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, user.Email, user.Name, user.PasswordHash, user.Active, user.DepartmentID)
	if err != nil {
		return user, err
	}
	if tag.RowsAffected() == 0 {
		return user, ErrNotFound
	}
	if err := replaceUserRelations(ctx, tx, id, roleIDs, clientIDs, clientRoleIDs); err != nil {
		return user, err
	}
	if err := tx.Commit(ctx); err != nil {
		return user, err
	}
	return s.GetUser(ctx, id)
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET deleted_at=now(), updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	return err
}

func (s *Store) LoadUserRelations(ctx context.Context, user *domain.User) error {
	if user.DepartmentID != nil {
		dept, err := s.GetDepartment(ctx, *user.DepartmentID)
		if err == nil {
			user.Department = dept
		}
	}
	roles, err := s.userRoles(ctx, user.ID)
	if err != nil {
		return err
	}
	user.Roles = roles
	clients, err := s.userClients(ctx, user.ID)
	if err != nil {
		return err
	}
	user.AuthorizedClients = clients
	clientRoles, err := s.userClientRoles(ctx, user.ID)
	if err != nil {
		return err
	}
	user.ClientRoles = clientRoles
	return nil
}

func (s *Store) userRoles(ctx context.Context, userID int64) ([]domain.Role, error) {
	rows, err := s.Pool.Query(ctx, `SELECT r.id, r.created_at, r.updated_at, r.deleted_at, r.name, r.description FROM roles r JOIN user_roles ur ON ur.role_id=r.id WHERE ur.user_id=$1 AND r.deleted_at IS NULL ORDER BY r.name`, userID)
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

func (s *Store) userClients(ctx context.Context, userID int64) ([]domain.Client, error) {
	rows, err := s.Pool.Query(ctx, `SELECT c.id, c.created_at, c.updated_at, c.deleted_at, c.client_id, c.client_secret, c.name, c.redirect_uris FROM clients c JOIN user_authorized_clients uac ON uac.client_id=c.id WHERE uac.user_id=$1 AND c.deleted_at IS NULL ORDER BY c.name`, userID)
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
	return out, rows.Err()
}

func (s *Store) userClientRoles(ctx context.Context, userID int64) ([]domain.ClientRole, error) {
	rows, err := s.Pool.Query(ctx, `SELECT cr.id, cr.created_at, cr.updated_at, cr.deleted_at, cr.client_id, cr.name, cr.description FROM client_roles cr JOIN user_client_roles ucr ON ucr.client_role_id=cr.id WHERE ucr.user_id=$1 AND cr.deleted_at IS NULL ORDER BY cr.name`, userID)
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

func replaceUserRelations(ctx context.Context, tx pgx.Tx, userID int64, roleIDs, clientIDs, clientRoleIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id=$1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_authorized_clients WHERE user_id=$1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_client_roles WHERE user_id=$1`, userID); err != nil {
		return err
	}
	for _, id := range roleIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles(user_id, role_id) VALUES($1,$2)`, userID, id); err != nil {
			return err
		}
	}
	for _, id := range clientIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO user_authorized_clients(user_id, client_id) VALUES($1,$2)`, userID, id); err != nil {
			return err
		}
	}
	for _, id := range clientRoleIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO user_client_roles(user_id, client_role_id) VALUES($1,$2)`, userID, id); err != nil {
			return err
		}
	}
	return nil
}

func scanUser(rows pgx.Rows, user *domain.User) error {
	return rows.Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.Email, &user.Name, &user.PasswordHash, &user.Active, &user.DepartmentID)
}
