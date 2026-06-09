package store

import (
	"context"

	"gopenid/internal/domain"

	"github.com/jackc/pgx/v5"
)

const userColumns = `id, created_at, updated_at, deleted_at, email, name, password_hash, active, blocked, blocked_reason, phone, title, avatar_url, last_login_at, department_id`

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+userColumns+` FROM users WHERE deleted_at IS NULL ORDER BY email`)
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
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range users {
		if err := s.LoadUserRelations(ctx, &users[i]); err != nil {
			return nil, err
		}
	}
	return users, nil
}

func (s *Store) GetUser(ctx context.Context, id int64) (domain.User, error) {
	var user domain.User
	err := s.Pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id=$1 AND deleted_at IS NULL`, id).Scan(userScanDest(&user)...)
	if err != nil {
		return user, normalizeErr(err)
	}
	return user, s.LoadUserRelations(ctx, &user)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User
	err := s.Pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email=$1 AND deleted_at IS NULL`, email).Scan(userScanDest(&user)...)
	if err != nil {
		return user, normalizeErr(err)
	}
	return user, s.LoadUserRelations(ctx, &user)
}

func (s *Store) CreateUser(ctx context.Context, user domain.User, roleIDs, clientIDs, clientRoleIDs, departmentIDs, groupIDs []int64) (domain.User, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return user, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `INSERT INTO users(email, name, password_hash, active, blocked, blocked_reason, phone, title, avatar_url, department_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		user.Email, user.Name, user.PasswordHash, user.Active, user.Blocked, user.BlockedReason, user.Phone, user.Title, user.AvatarURL, user.DepartmentID).Scan(&user.ID)
	if err != nil {
		return user, err
	}
	if err := replaceUserRelations(ctx, tx, user.ID, roleIDs, clientIDs, clientRoleIDs, departmentIDs, groupIDs); err != nil {
		return user, err
	}
	if err := tx.Commit(ctx); err != nil {
		return user, err
	}
	return s.GetUser(ctx, user.ID)
}

func (s *Store) UpdateUser(ctx context.Context, id int64, user domain.User, roleIDs, clientIDs, clientRoleIDs, departmentIDs, groupIDs []int64) (domain.User, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return user, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE users SET email=$2, name=$3, password_hash=COALESCE(NULLIF($4,''), password_hash), active=$5, blocked=$6, blocked_reason=$7, phone=$8, title=$9, avatar_url=$10, department_id=$11, updated_at=now() WHERE id=$1 AND deleted_at IS NULL`,
		id, user.Email, user.Name, user.PasswordHash, user.Active, user.Blocked, user.BlockedReason, user.Phone, user.Title, user.AvatarURL, user.DepartmentID)
	if err != nil {
		return user, err
	}
	if tag.RowsAffected() == 0 {
		return user, ErrNotFound
	}
	if err := replaceUserRelations(ctx, tx, id, roleIDs, clientIDs, clientRoleIDs, departmentIDs, groupIDs); err != nil {
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

// UpdateProfile lets a user edit a limited set of their own fields.
func (s *Store) UpdateProfile(ctx context.Context, id int64, name, phone, title, avatarURL string) (domain.User, error) {
	tag, err := s.Pool.Exec(ctx, `UPDATE users SET name=$2, phone=$3, title=$4, avatar_url=$5, updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, name, phone, title, avatarURL)
	if err != nil {
		return domain.User{}, err
	}
	if tag.RowsAffected() == 0 {
		return domain.User{}, ErrNotFound
	}
	return s.GetUser(ctx, id)
}

// ChangePassword updates the password hash for a user.
func (s *Store) ChangePassword(ctx context.Context, id int64, passwordHash string) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE users SET password_hash=$2, updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetBlocked toggles a user's blocked flag with an optional reason.
func (s *Store) SetBlocked(ctx context.Context, id int64, blocked bool, reason string) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE users SET blocked=$2, blocked_reason=$3, updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, blocked, reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchLastLogin records the time of a successful login.
func (s *Store) TouchLastLogin(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET last_login_at=now() WHERE id=$1`, id)
	return err
}

func (s *Store) LoadUserRelations(ctx context.Context, user *domain.User) error {
	if user.DepartmentID != nil {
		dept, err := s.GetDepartment(ctx, *user.DepartmentID)
		if err == nil {
			user.Department = dept
		}
	}
	departments, err := s.userDepartments(ctx, user.ID)
	if err != nil {
		return err
	}
	user.Departments = departments
	groups, err := s.userGroups(ctx, user.ID)
	if err != nil {
		return err
	}
	user.Groups = groups
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

func (s *Store) userDepartments(ctx context.Context, userID int64) ([]domain.Department, error) {
	rows, err := s.Pool.Query(ctx, `SELECT d.id, d.created_at, d.updated_at, d.deleted_at, d.name, d.description FROM departments d JOIN user_departments ud ON ud.department_id=d.id WHERE ud.user_id=$1 AND d.deleted_at IS NULL ORDER BY d.name`, userID)
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

func (s *Store) userGroups(ctx context.Context, userID int64) ([]domain.Group, error) {
	rows, err := s.Pool.Query(ctx, `SELECT g.id, g.created_at, g.updated_at, g.deleted_at, g.name, g.description FROM groups g JOIN user_groups ug ON ug.group_id=g.id WHERE ug.user_id=$1 AND g.deleted_at IS NULL ORDER BY g.name`, userID)
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

func (s *Store) userClients(ctx context.Context, userID int64) ([]domain.Client, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+clientColumnsPrefixed("c")+` FROM clients c JOIN user_authorized_clients uac ON uac.client_id=c.id WHERE uac.user_id=$1 AND c.deleted_at IS NULL ORDER BY c.name`, userID)
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

func replaceUserRelations(ctx context.Context, tx pgx.Tx, userID int64, roleIDs, clientIDs, clientRoleIDs, departmentIDs, groupIDs []int64) error {
	resets := []string{
		`DELETE FROM user_roles WHERE user_id=$1`,
		`DELETE FROM user_authorized_clients WHERE user_id=$1`,
		`DELETE FROM user_client_roles WHERE user_id=$1`,
		`DELETE FROM user_departments WHERE user_id=$1`,
		`DELETE FROM user_groups WHERE user_id=$1`,
	}
	for _, q := range resets {
		if _, err := tx.Exec(ctx, q, userID); err != nil {
			return err
		}
	}
	inserts := []struct {
		query string
		ids   []int64
	}{
		{`INSERT INTO user_roles(user_id, role_id) VALUES($1,$2)`, roleIDs},
		{`INSERT INTO user_authorized_clients(user_id, client_id) VALUES($1,$2)`, clientIDs},
		{`INSERT INTO user_client_roles(user_id, client_role_id) VALUES($1,$2)`, clientRoleIDs},
		{`INSERT INTO user_departments(user_id, department_id) VALUES($1,$2)`, departmentIDs},
		{`INSERT INTO user_groups(user_id, group_id) VALUES($1,$2)`, groupIDs},
	}
	for _, ins := range inserts {
		for _, id := range ins.ids {
			if _, err := tx.Exec(ctx, ins.query, userID, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func userScanDest(user *domain.User) []any {
	return []any{
		&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&user.Email, &user.Name, &user.PasswordHash, &user.Active,
		&user.Blocked, &user.BlockedReason, &user.Phone, &user.Title,
		&user.AvatarURL, &user.LastLoginAt, &user.DepartmentID,
	}
}

func scanUser(rows pgx.Rows, user *domain.User) error {
	return rows.Scan(userScanDest(user)...)
}
