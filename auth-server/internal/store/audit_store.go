package store

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditStore struct {
	pool *pgxpool.Pool
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
