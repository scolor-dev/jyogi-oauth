package handler

import (
	"net/http"
	"strings"

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
	memberID := mustParseUUID(sub)

	member, err := h.memberStore.GetByID(r.Context(), memberID)
	if err != nil || member == nil {
		writeError(w, http.StatusNotFound, "not_found", "Member not found")
		return
	}

	resp := map[string]any{
		"sub": member.ID.String(),
	}

	scopes := strings.Split(scope, " ")
	scopeSet := make(map[string]bool)
	for _, s := range scopes {
		scopeSet[s] = true
	}

	if scopeSet["profile"] {
		resp["name"] = member.Username
		resp["preferred_username"] = member.Username
		resp["updated_at"] = member.UpdatedAt.Unix()
	}

	if scopeSet["identity"] {
		var displayName, avatarURL, themeColor, tagline *string
		err := h.pool.QueryRow(r.Context(),
			`SELECT display_name, avatar_url, theme_color, tagline FROM resource.member_identities WHERE member_id = $1`,
			memberID,
		).Scan(&displayName, &avatarURL, &themeColor, &tagline)
		if err == nil {
			if displayName != nil {
				resp["name"] = *displayName
			}
			if avatarURL != nil {
				resp["picture"] = *avatarURL
			}
			if themeColor != nil {
				resp["color"] = *themeColor
			}
			if tagline != nil {
				resp["tagline"] = *tagline
			}
		}
	}

	if !scopeSet["profile"] && !scopeSet["identity"] {
		resp["preferred_username"] = member.Username
		resp["email"] = member.Email
	}

	writeJSON(w, http.StatusOK, resp)
}
