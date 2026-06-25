package handler

import (
	"net/http"

	"github.com/jyogi-oauth/auth-server/internal/oauth"
)

type IntrospectHandler struct {
	jwtService *oauth.JWTService
}

func NewIntrospectHandler(jwtService *oauth.JWTService) *IntrospectHandler {
	return &IntrospectHandler{jwtService: jwtService}
}

func (h *IntrospectHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusOK, map[string]bool{"active": false})
		return
	}

	token := r.FormValue("token")
	if token == "" {
		writeJSON(w, http.StatusOK, map[string]bool{"active": false})
		return
	}

	claims, err := h.jwtService.VerifyToken(token)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]bool{"active": false})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"active":     true,
		"scope":      claims["scope"],
		"client_id":  claims["client_id"],
		"sub":        claims["sub"],
		"exp":        claims["exp"],
		"iat":        claims["iat"],
		"iss":        claims["iss"],
		"token_type": "Bearer",
	})
}
