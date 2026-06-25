package model

import (
	"time"

	"github.com/google/uuid"
)

type ConsentRecord struct {
	ID        uuid.UUID  `json:"id"`
	MemberID  uuid.UUID  `json:"member_id"`
	ClientID  uuid.UUID  `json:"client_id"`
	Scopes    []string   `json:"scopes"`
	GrantedAt time.Time  `json:"granted_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}
