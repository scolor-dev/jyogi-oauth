package model

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID                uuid.UUID  `json:"id"`
	ClientID          string     `json:"client_id"`
	ClientSecretHash  *string    `json:"-"`
	Name              string     `json:"name"`
	Description       *string    `json:"description,omitempty"`
	IconURL           *string    `json:"icon_url,omitempty"`
	ClientType        string     `json:"client_type"`
	RedirectURIs      []string   `json:"redirect_uris"`
	AllowedGrantTypes []string   `json:"allowed_grant_types"`
	IsActive          bool       `json:"is_active"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
