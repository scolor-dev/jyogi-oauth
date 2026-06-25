package handler

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/jyogi-oauth/auth-server/internal/config"
	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type TokenHandler struct {
	authCodeStore *store.AuthCodeStore
	refreshStore  *store.RefreshStore
	clientStore   *store.ClientStore
	memberStore   *store.MemberStore
	jwtService    *oauth.JWTService
	auditStore    *store.AuditStore
	cfg           *config.Config
}

func NewTokenHandler(
	authCodeStore *store.AuthCodeStore,
	refreshStore *store.RefreshStore,
	clientStore *store.ClientStore,
	memberStore *store.MemberStore,
	jwtService *oauth.JWTService,
	auditStore *store.AuditStore,
	cfg *config.Config,
) *TokenHandler {
	return &TokenHandler{
		authCodeStore: authCodeStore,
		refreshStore:  refreshStore,
		clientStore:   clientStore,
		memberStore:   memberStore,
		jwtService:    jwtService,
		auditStore:    auditStore,
		cfg:           cfg,
	}
}

func (h *TokenHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Cannot parse form")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	grantType := r.FormValue("grant_type")
	switch grantType {
	case "authorization_code":
		h.handleAuthorizationCode(w, r)
	case "refresh_token":
		h.handleRefreshToken(w, r)
	case "client_credentials":
		h.handleClientCredentials(w, r)
	default:
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "Unsupported grant_type")
	}
}

func (h *TokenHandler) handleAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	codeVerifier := r.FormValue("code_verifier")
	clientID := r.FormValue("client_id")

	if code == "" || redirectURI == "" || codeVerifier == "" || clientID == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "code, redirect_uri, code_verifier, and client_id are required")
		return
	}

	client, _, err := oauth.AuthenticateClient(r.Context(), r, h.clientStore)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal error")
		return
	}
	if client == nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "Client authentication failed")
		return
	}
	if client.ClientID != clientID {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "client_id mismatch")
		return
	}

	codeData, err := h.authCodeStore.Consume(r.Context(), code)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal error")
		return
	}
	if codeData == nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "Authorization code is invalid or expired")
		return
	}

	if codeData.ClientID != clientID {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "client_id does not match")
		return
	}
	if codeData.RedirectURI != redirectURI {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "redirect_uri does not match")
		return
	}

	if !oauth.VerifyPKCE(codeVerifier, codeData.CodeChallenge, codeData.CodeChallengeMethod) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
		return
	}

	member, err := h.memberStore.GetByID(r.Context(), mustParseUUID(codeData.MemberID))
	if err != nil || member == nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "Member not found")
		return
	}

	accessToken, err := h.jwtService.SignAccessToken(oauth.AccessTokenClaims{
		MemberID: codeData.MemberID,
		ClientID: clientID,
		Scope:    codeData.Scope,
		Username: member.Username,
	})
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Failed to sign token")
		return
	}

	refreshToken, err := oauth.GenerateRandomString(h.cfg.RefreshTokenLength)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Failed to generate refresh token")
		return
	}

	err = h.refreshStore.Save(r.Context(), refreshToken, &store.RefreshTokenData{
		MemberID: codeData.MemberID,
		ClientID: clientID,
		Scope:    codeData.Scope,
	})
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Failed to save refresh token")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionTokenIssued, &member.ID, &clientID, r.RemoteAddr, r.UserAgent(), map[string]string{
		"grant_type": "authorization_code",
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(h.cfg.AccessTokenTTL.Seconds()),
		"refresh_token": refreshToken,
		"scope":         codeData.Scope,
	})
}

func (h *TokenHandler) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")

	if refreshToken == "" || clientID == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "refresh_token and client_id are required")
		return
	}

	client, _, err := oauth.AuthenticateClient(r.Context(), r, h.clientStore)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal error")
		return
	}
	if client == nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "Client authentication failed")
		return
	}

	tokenData, err := h.refreshStore.Get(r.Context(), refreshToken)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal error")
		return
	}
	if tokenData == nil {
		log.Printf("Potential refresh token reuse detected for client %s", clientID)
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "Refresh token is invalid or expired")
		return
	}

	if tokenData.ClientID != clientID {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "client_id does not match")
		return
	}

	if err := h.refreshStore.Delete(r.Context(), refreshToken, tokenData.MemberID); err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal error")
		return
	}

	scope := tokenData.Scope
	if reqScope := r.FormValue("scope"); reqScope != "" {
		scope = reqScope
	}

	member, err := h.memberStore.GetByID(r.Context(), mustParseUUID(tokenData.MemberID))
	if err != nil || member == nil || !member.IsActive {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "Member not found or inactive")
		return
	}

	accessToken, err := h.jwtService.SignAccessToken(oauth.AccessTokenClaims{
		MemberID: tokenData.MemberID,
		ClientID: clientID,
		Scope:    scope,
		Username: member.Username,
	})
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Failed to sign token")
		return
	}

	newRefreshToken, err := oauth.GenerateRandomString(h.cfg.RefreshTokenLength)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Failed to generate refresh token")
		return
	}

	err = h.refreshStore.Save(r.Context(), newRefreshToken, &store.RefreshTokenData{
		MemberID: tokenData.MemberID,
		ClientID: clientID,
		Scope:    scope,
	})
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Failed to save refresh token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(h.cfg.AccessTokenTTL.Seconds()),
		"refresh_token": newRefreshToken,
		"scope":         scope,
	})
}

func (h *TokenHandler) handleClientCredentials(w http.ResponseWriter, r *http.Request) {
	client, _, err := oauth.AuthenticateClient(r.Context(), r, h.clientStore)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal error")
		return
	}
	if client == nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "Client authentication failed")
		return
	}
	if client.ClientType != "confidential" {
		writeOAuthError(w, http.StatusBadRequest, "unauthorized_client", "client_credentials requires confidential client")
		return
	}

	hasGrant := false
	for _, g := range client.AllowedGrantTypes {
		if g == "client_credentials" {
			hasGrant = true
			break
		}
	}
	if !hasGrant {
		writeOAuthError(w, http.StatusBadRequest, "unauthorized_client", "client_credentials grant not allowed")
		return
	}

	scope := r.FormValue("scope")
	if scope == "" {
		scope = "read"
	}

	accessToken, err := h.jwtService.SignClientCredentialsToken(client.ClientID, scope)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Failed to sign token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(h.cfg.AccessTokenTTL.Seconds()),
		"scope":        scope,
	})
}

func writeOAuthError(w http.ResponseWriter, status int, errorCode, description string) {
	writeJSON(w, status, map[string]string{
		"error":             errorCode,
		"error_description": description,
	})
}

func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}
	}
	return id
}
