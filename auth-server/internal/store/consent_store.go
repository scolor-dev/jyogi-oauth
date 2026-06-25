package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jyogi-oauth/auth-server/internal/model"
)

type ConsentStore struct {
	pool *pgxpool.Pool
}

func NewConsentStore(pool *pgxpool.Pool) *ConsentStore {
	return &ConsentStore{pool: pool}
}

func (s *ConsentStore) GetByMemberAndClient(ctx context.Context, memberID, clientID uuid.UUID) (*model.ConsentRecord, error) {
	c := &model.ConsentRecord{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, member_id, client_id, scopes, granted_at, revoked_at
		 FROM auth.consent_records
		 WHERE member_id = $1 AND client_id = $2 AND revoked_at IS NULL`,
		memberID, clientID,
	).Scan(&c.ID, &c.MemberID, &c.ClientID, &c.Scopes, &c.GrantedAt, &c.RevokedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get consent: %w", err)
	}
	return c, nil
}

func (s *ConsentStore) Upsert(ctx context.Context, memberID, clientID uuid.UUID, scopes []string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO auth.consent_records (member_id, client_id, scopes)
		 VALUES ($1, $2, $3)
		 ON CONFLICT ON CONSTRAINT uq_consent_member_client
		 DO UPDATE SET scopes = $3, granted_at = now(), revoked_at = NULL`,
		memberID, clientID, scopes,
	)
	if err != nil {
		return fmt.Errorf("upsert consent: %w", err)
	}
	return nil
}

func (s *ConsentStore) Revoke(ctx context.Context, memberID, clientID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE auth.consent_records SET revoked_at = now()
		 WHERE member_id = $1 AND client_id = $2 AND revoked_at IS NULL`,
		memberID, clientID,
	)
	if err != nil {
		return fmt.Errorf("revoke consent: %w", err)
	}
	return nil
}
