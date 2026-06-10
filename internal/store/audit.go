package store

import (
	"context"
	"strconv"
	"strings"
	"time"

	"gopenid/internal/domain"
)

type AuditFilter struct {
	UserID   int64
	Event    string
	Email    string
	ClientID string
	IP       string
	Success  *bool
	From     *time.Time
	To       *time.Time
	Page     int
	PageSize int
}

type AuditPage struct {
	Items    []domain.AuditLog `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

func (s *Store) WriteAudit(ctx context.Context, in domain.AuditLog) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO audit_logs(user_id, email, client_id, event, success, message, ip, user_agent, device, browser, os) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		in.UserID, in.Email, in.ClientID, in.Event, in.Success, in.Message, in.IP, in.UserAgent, in.Device, in.Browser, in.OS)
	return err
}

func (s *Store) ListAuditLogs(ctx context.Context, filter AuditFilter) (AuditPage, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 500 {
		filter.PageSize = 100
	}
	where, args := auditWhere(filter)
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs`+where, args...).Scan(&total); err != nil {
		return AuditPage{}, err
	}
	query := `SELECT id, created_at, updated_at, deleted_at, user_id, email, client_id, event, success, message, ip, user_agent, device, browser, os FROM audit_logs` + where
	idx := len(args) + 1
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(idx) + ` OFFSET $` + strconv.Itoa(idx+1)
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return AuditPage{}, err
	}
	defer rows.Close()
	var out []domain.AuditLog
	for rows.Next() {
		var row domain.AuditLog
		if err := rows.Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt, &row.DeletedAt, &row.UserID, &row.Email, &row.ClientID, &row.Event, &row.Success, &row.Message, &row.IP, &row.UserAgent, &row.Device, &row.Browser, &row.OS); err != nil {
			return AuditPage{}, err
		}
		out = append(out, row)
	}
	return AuditPage{Items: out, Total: total, Page: filter.Page, PageSize: filter.PageSize}, rows.Err()
}

func auditWhere(filter AuditFilter) (string, []any) {
	query := ` WHERE 1=1`
	args := []any{}
	idx := 1
	if filter.UserID > 0 {
		query += ` AND user_id=$` + strconv.Itoa(idx)
		args = append(args, filter.UserID)
		idx++
	}
	if filter.Event != "" {
		query += ` AND event=$` + strconv.Itoa(idx)
		args = append(args, filter.Event)
		idx++
	}
	if filter.Email != "" {
		query += ` AND lower(email) LIKE $` + strconv.Itoa(idx)
		args = append(args, "%"+strings.ToLower(filter.Email)+"%")
		idx++
	}
	if filter.ClientID != "" {
		query += ` AND lower(client_id) LIKE $` + strconv.Itoa(idx)
		args = append(args, "%"+strings.ToLower(filter.ClientID)+"%")
		idx++
	}
	if filter.IP != "" {
		query += ` AND ip LIKE $` + strconv.Itoa(idx)
		args = append(args, "%"+filter.IP+"%")
		idx++
	}
	if filter.Success != nil {
		query += ` AND success=$` + strconv.Itoa(idx)
		args = append(args, *filter.Success)
		idx++
	}
	if filter.From != nil {
		query += ` AND created_at >= $` + strconv.Itoa(idx)
		args = append(args, *filter.From)
		idx++
	}
	if filter.To != nil {
		query += ` AND created_at <= $` + strconv.Itoa(idx)
		args = append(args, *filter.To)
	}
	return query, args
}
