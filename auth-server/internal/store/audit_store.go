package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jyogi-oauth/auth-server/internal/model"
)

type AuditStore struct {
	pool *pgxpool.Pool
}

type AuditLogFilter struct {
	Action   string
	MemberID *uuid.UUID
	From     *time.Time
	To       *time.Time
	Page     int
	PerPage  int
}

func (s *AuditStore) List(ctx context.Context, filter AuditLogFilter) ([]model.AuditLog, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}
	if filter.PerPage > 100 {
		filter.PerPage = 100
	}

	where := `WHERE ($1 = '' OR action = $1)
		AND ($2::uuid IS NULL OR member_id = $2)
		AND ($3::timestamptz IS NULL OR created_at >= $3)
		AND ($4::timestamptz IS NULL OR created_at <= $4)`

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM auth.audit_logs `+where, filter.Action, filter.MemberID, filter.From, filter.To).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, member_id, client_id, action, ip_address::text, user_agent, details, created_at
		 FROM auth.audit_logs `+where+`
		 ORDER BY created_at DESC
		 LIMIT $5 OFFSET $6`,
		filter.Action, filter.MemberID, filter.From, filter.To, filter.PerPage, (filter.Page-1)*filter.PerPage,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var item model.AuditLog
		if err := rows.Scan(&item.ID, &item.MemberID, &item.ClientID, &item.Action, &item.IPAddress, &item.UserAgent, &item.Details, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		logs = append(logs, item)
	}
	if err := rows.Err(); err != nil && err != pgx.ErrNoRows {
		return nil, 0, fmt.Errorf("iterate audit logs: %w", err)
	}
	return logs, total, nil
}

func NewAuditStore(pool *pgxpool.Pool) *AuditStore {
	return &AuditStore{pool: pool}
}

func (s *AuditStore) Log(ctx context.Context, action string, memberID *uuid.UUID, clientID *string, ipAddress, userAgent string, details any) {
	var detailsJSON json.RawMessage
	if details != nil {
		b, err := json.Marshal(details)
		if err == nil {
			detailsJSON = b
		}
	}

	var ipAddr *string
	if ipAddress != "" {
		ipAddr = &ipAddress
	}
	var ua *string
	if userAgent != "" {
		ua = &userAgent
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO auth.audit_logs (member_id, client_id, action, ip_address, user_agent, details)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		memberID, clientID, action, ipAddr, ua, detailsJSON,
	)
	if err != nil {
		log.Printf("audit log error: %v", err)
	}
}
