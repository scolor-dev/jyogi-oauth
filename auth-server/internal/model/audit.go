package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	ActionLoginSuccess     = "login_success"
	ActionLoginFailure     = "login_failure"
	ActionConsentGranted   = "consent_granted"
	ActionConsentRevoked   = "consent_revoked"
	ActionTokenIssued      = "token_issued"
	ActionTokenRevoked     = "token_revoked"
	ActionClientCreated    = "client_created"
	ActionClientUpdated    = "client_updated"
	ActionClientDeleted    = "client_deleted"
	ActionMemberCreated    = "member_created"
	ActionMemberUpdated    = "member_updated"
	ActionMemberDeactivated = "member_deactivated"
	ActionSessionRevoked    = "session_revoked"
)

type AuditLog struct {
	ID        uuid.UUID       `json:"id"`
	MemberID  *uuid.UUID      `json:"member_id,omitempty"`
	ClientID  *string         `json:"client_id,omitempty"`
	Action    string          `json:"action"`
	IPAddress *string         `json:"ip_address,omitempty"`
	UserAgent *string         `json:"user_agent,omitempty"`
	Details   json.RawMessage `json:"details,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}
