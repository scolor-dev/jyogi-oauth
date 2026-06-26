package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleMember    = "member"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

const ActionPasswordReset = "password_reset"

type Member struct {
	ID                 uuid.UUID `json:"id"`
	Username           string    `json:"username"`
	PasswordHash       string    `json:"-"`
	Email              string    `json:"email"`
	Role               string    `json:"role"`
	MustChangePassword bool      `json:"must_change_password"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (m *Member) IsAdmin() bool     { return m.Role == RoleAdmin }
func (m *Member) IsModerator() bool { return m.Role == RoleModerator || m.Role == RoleAdmin }
