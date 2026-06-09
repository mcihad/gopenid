package store

import (
	"context"
	"strconv"

	"gopenid/internal/domain"
)

func (s *Store) WriteAudit(ctx context.Context, in domain.AuditLog) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO audit_logs(user_id, email, client_id, event, success, message, ip, user_agent, device, browser, os) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		in.UserID, in.Email, in.ClientID, in.Event, in.Success, in.Message, in.IP, in.UserAgent, in.Device, in.Browser, in.OS)
	return err
}

// ListAuditLogs returns the most recent audit entries, optionally filtered by
// user id (0 = all) and event (empty = all), capped by limit.
func (s *Store) ListAuditLogs(ctx context.Context, userID int64, event string, limit int) ([]domain.AuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT id, created_at, updated_at, deleted_at, user_id, email, client_id, event, success, message, ip, user_agent, device, browser, os FROM audit_logs WHERE 1=1`
	args := []any{}
	idx := 1
	if userID > 0 {
		query += ` AND user_id=$` + strconv.Itoa(idx)
		args = append(args, userID)
		idx++
	}
	if event != "" {
		query += ` AND event=$` + strconv.Itoa(idx)
		args = append(args, event)
		idx++
	}
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(idx)
	args = append(args, limit)

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AuditLog
	for rows.Next() {
		var row domain.AuditLog
		if err := rows.Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.UserID, &row.Email, &row.ClientID, &row.Event, &row.Success, &row.Message, &row.IP, &row.UserAgent, &row.Device, &row.Browser, &row.OS); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
