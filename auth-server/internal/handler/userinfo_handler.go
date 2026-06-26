package handler

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type UserInfoHandler struct {
	memberStore *store.MemberStore
	jwtService  *oauth.JWTService
	pool        *pgxpool.Pool
}

func NewUserInfoHandler(memberStore *store.MemberStore, jwtService *oauth.JWTService, pool *pgxpool.Pool) *UserInfoHandler {
	return &UserInfoHandler{memberStore: memberStore, jwtService: jwtService, pool: pool}
}

func (h *UserInfoHandler) UserInfo(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		writeError(w, http.StatusUnauthorized, "unauthorized", "Bearer token required")
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := h.jwtService.VerifyToken(tokenStr)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return
	}

	sub, _ := claims["sub"].(string)
	scope, _ := claims["scope"].(string)

	if !oauth.HasScope(scope, "openid") {
		w.Header().Set("WWW-Authenticate", `Bearer error="insufficient_scope"`)
		writeError(w, http.StatusForbidden, "insufficient_scope", "openid scope is required for userinfo")
		return
	}

	if grantType, _ := claims["grant_type"].(string); grantType == "client_credentials" {
		writeError(w, http.StatusForbidden, "forbidden", "UserInfo is not available for client_credentials tokens")
		return
	}

	memberID := mustParseUUID(sub)
	if memberID == (uuid.UUID{}) {
		writeError(w, http.StatusBadRequest, "invalid_token", "Invalid subject in token")
		return
	}

	member, err := h.memberStore.GetByID(r.Context(), memberID)
	if err != nil || member == nil {
		writeError(w, http.StatusNotFound, "not_found", "Member not found")
		return
	}

	resp := map[string]any{
		"sub": member.ID.String(),
	}

	if oauth.HasScope(scope, "profile") {
		resp["name"] = member.Username
		resp["preferred_username"] = member.Username
		resp["username"] = member.Username
		resp["updated_at"] = member.UpdatedAt.Unix()
	}

	if oauth.HasScope(scope, "identity") {
		ic := getIdentityClaims(r.Context(), h.pool, memberID)
		if ic != nil {
			if ic.DisplayName != nil {
				resp["name"] = *ic.DisplayName
			}
			if ic.AvatarURL != nil {
				resp["picture"] = *ic.AvatarURL
			}
			if ic.ThemeColor != nil {
				resp["color"] = *ic.ThemeColor
			}
			if ic.Tagline != nil {
				resp["tagline"] = *ic.Tagline
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
