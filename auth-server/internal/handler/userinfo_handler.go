package handler

import (
	"net/http"
	"strings"

	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type UserInfoHandler struct {
	memberStore *store.MemberStore
	jwtService  *oauth.JWTService
}

func NewUserInfoHandler(memberStore *store.MemberStore, jwtService *oauth.JWTService) *UserInfoHandler {
	return &UserInfoHandler{memberStore: memberStore, jwtService: jwtService}
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
	memberID := mustParseUUID(sub)

	member, err := h.memberStore.GetByID(r.Context(), memberID)
	if err != nil || member == nil {
		writeError(w, http.StatusNotFound, "not_found", "Member not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"sub":      member.ID.String(),
		"username": member.Username,
		"email":    member.Email,
	})
}
