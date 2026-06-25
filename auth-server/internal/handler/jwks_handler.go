package handler

import (
	"net/http"

	"github.com/jyogi-oauth/auth-server/internal/oauth"
)

type JWKSHandler struct {
	jwtService *oauth.JWTService
}

func NewJWKSHandler(jwtService *oauth.JWTService) *JWKSHandler {
	return &JWKSHandler{jwtService: jwtService}
}

func (h *JWKSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, h.jwtService.GetJWKS())
}
