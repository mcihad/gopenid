package store

import (
	"context"
	"strconv"
	"strings"

	"gopenid/internal/domain"
)

func (s *Store) ListPolicies(ctx context.Context) ([]domain.Policy, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, created_at, updated_at, deleted_at, name, description, type, effect, ip_cidrs, days_of_week, start_time, end_time FROM policies WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Policy
	for rows.Next() {
		row, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) CreatePolicy(ctx context.Context, in domain.Policy) (domain.Policy, error) {
	row, err := scanPolicy(s.Pool.QueryRow(ctx, `INSERT INTO policies(name, description, type, effect, ip_cidrs, days_of_week, start_time, end_time) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at, updated_at, deleted_at, name, description, type, effect, ip_cidrs, days_of_week, start_time, end_time`,
		in.Name, in.Description, in.Type, in.Effect, in.IPCIDRs, marshalDays(in.DaysOfWeek), in.StartTime, in.EndTime))
	return row, err
}

func (s *Store) GetPolicy(ctx context.Context, id int64) (domain.Policy, error) {
	row, err := scanPolicy(s.Pool.QueryRow(ctx, `SELECT id, created_at, updated_at, deleted_at, name, description, type, effect, ip_cidrs, days_of_week, start_time, end_time FROM policies WHERE id=$1 AND deleted_at IS NULL`, id))
	return row, normalizeErr(err)
}

func (s *Store) UpdatePolicy(ctx context.Context, id int64, in domain.Policy) (domain.Policy, error) {
	row, err := scanPolicy(s.Pool.QueryRow(ctx, `UPDATE policies SET name=$2, description=$3, type=$4, effect=$5, ip_cidrs=$6, days_of_week=$7, start_time=$8, end_time=$9, updated_at=now() WHERE id=$1 AND deleted_at IS NULL RETURNING id, created_at, updated_at, deleted_at, name, description, type, effect, ip_cidrs, days_of_week, start_time, end_time`,
		id, in.Name, in.Description, in.Type, in.Effect, in.IPCIDRs, marshalDays(in.DaysOfWeek), in.StartTime, in.EndTime))
	return row, normalizeErr(err)
}

func (s *Store) DeletePolicy(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE policies SET deleted_at=now(), updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	return err
}

func (s *Store) AssignPolicy(ctx context.Context, policyID int64, subjectType domain.PolicySubject, subjectID int64) (domain.PolicyAssignment, error) {
	var row domain.PolicyAssignment
	err := s.Pool.QueryRow(ctx, `INSERT INTO policy_assignments(policy_id, subject_type, subject_id) VALUES($1,$2,$3) RETURNING id, created_at, updated_at, deleted_at, policy_id, subject_type, subject_id`,
		policyID, subjectType, subjectID).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.PolicyID, &row.SubjectType, &row.SubjectID)
	return row, err
}

func (s *Store) UnassignPolicy(ctx context.Context, assignmentID int64) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM policy_assignments WHERE id=$1`, assignmentID)
	return err
}

func (s *Store) ListPolicyAssignments(ctx context.Context, policyID int64) ([]domain.PolicyAssignment, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, created_at, updated_at, deleted_at, policy_id, subject_type, subject_id FROM policy_assignments WHERE policy_id=$1 ORDER BY id`, policyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PolicyAssignment
	for rows.Next() {
		var row domain.PolicyAssignment
		if err := rows.Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.PolicyID, &row.SubjectType, &row.SubjectID); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// PoliciesForSubjects returns active policies attached to any of the given
// subjects of a single subject type.
func (s *Store) PoliciesForSubjects(ctx context.Context, subjectType domain.PolicySubject, subjectIDs []int64) ([]domain.Policy, error) {
	if len(subjectIDs) == 0 {
		return nil, nil
	}
	query := `SELECT p.id, p.created_at, p.updated_at, p.deleted_at, p.name, p.description, p.type, p.effect, p.ip_cidrs, p.days_of_week, p.start_time, p.end_time
		FROM policies p
		JOIN policy_assignments pa ON pa.policy_id=p.id
		WHERE p.deleted_at IS NULL AND pa.subject_type=$1 AND pa.subject_id IN (` + placeholders(2, subjectIDs) + `)`
	args := make([]any, 0, len(subjectIDs)+1)
	args = append(args, subjectType)
	for _, id := range subjectIDs {
		args = append(args, id)
	}
	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Policy
	for rows.Next() {
		row, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanPolicy(row scannable) (domain.Policy, error) {
	var p domain.Policy
	var days string
	if err := row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.Name, &p.Description, &p.Type, &p.Effect, &p.IPCIDRs, &days, &p.StartTime, &p.EndTime); err != nil {
		return p, err
	}
	p.DaysOfWeek = unmarshalDays(days)
	return p, nil
}

func marshalDays(days []int) string {
	if len(days) == 0 {
		return ""
	}
	parts := make([]string, 0, len(days))
	for _, d := range days {
		if d >= 0 && d <= 6 {
			parts = append(parts, strconv.Itoa(d))
		}
	}
	return strings.Join(parts, ",")
}

func unmarshalDays(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []int
	for _, part := range strings.Split(raw, ",") {
		if d, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			out = append(out, d)
		}
	}
	return out
}
