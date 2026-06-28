package handler

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MeIdentityHandler struct {
	pool *pgxpool.Pool
}

func NewMeIdentityHandler(pool *pgxpool.Pool) *MeIdentityHandler {
	return &MeIdentityHandler{pool: pool}
}

type identityResponse struct {
	MemberID    uuid.UUID `json:"member_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	ThemeColor  string    `json:"theme_color"`
	Tagline     *string   `json:"tagline"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (h *MeIdentityHandler) getIdentity(ctx context.Context, memberID uuid.UUID) (*identityResponse, error) {
	var resp identityResponse
	err := h.pool.QueryRow(ctx,
		`SELECT member_id, display_name, avatar_url, theme_color, tagline, updated_at
		 FROM resource.member_identities WHERE member_id = $1`, memberID,
	).Scan(&resp.MemberID, &resp.DisplayName, &resp.AvatarURL, &resp.ThemeColor, &resp.Tagline, &resp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (h *MeIdentityHandler) Get(w http.ResponseWriter, r *http.Request) {
	memberID, ok := requireSession(w, r)
	if !ok {
		return
	}

	identity, err := h.getIdentity(r.Context(), memberID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"identity": nil})
		return
	}
	writeJSON(w, http.StatusOK, identity)
}

type updateIdentityRequest struct {
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	ThemeColor  string  `json:"theme_color"`
	Tagline     *string `json:"tagline"`
}

func (h *MeIdentityHandler) Update(w http.ResponseWriter, r *http.Request) {
	memberID, ok := requireSession(w, r)
	if !ok {
		return
	}

	var req updateIdentityRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.DisplayName == "" || len([]rune(req.DisplayName)) > 100 {
		writeError(w, http.StatusBadRequest, "validation_error", "display_name must be 1-100 characters")
		return
	}

	if req.Tagline != nil && len([]rune(*req.Tagline)) > 8 {
		writeError(w, http.StatusBadRequest, "validation_error", "tagline must be 1-8 characters")
		return
	}

	if !isValidHexColor(req.ThemeColor) {
		writeError(w, http.StatusBadRequest, "validation_error", "theme_color must be a hex color code (#RRGGBB)")
		return
	}

	_, err := h.pool.Exec(r.Context(),
		`INSERT INTO resource.member_identities (member_id, display_name, avatar_url, theme_color, tagline)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (member_id) DO UPDATE SET
		   display_name = EXCLUDED.display_name,
		   avatar_url = EXCLUDED.avatar_url,
		   tagline = EXCLUDED.tagline,
		   theme_color = EXCLUDED.theme_color`,
		memberID, req.DisplayName, req.AvatarURL, req.ThemeColor, req.Tagline,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("Failed to update identity: %v", err))
		return
	}

	identity, _ := h.getIdentity(r.Context(), memberID)
	writeJSON(w, http.StatusOK, identity)
}

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func isValidHexColor(s string) bool {
	return hexColorRegex.MatchString(s)
}
