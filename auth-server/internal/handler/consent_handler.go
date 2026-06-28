package handler

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/jyogi-oauth/auth-server/internal/config"
	"github.com/jyogi-oauth/auth-server/internal/middleware"
	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type ConsentHandler struct {
	sessionStore  *store.SessionStore
	consentStore  *store.ConsentStore
	authCodeStore *store.AuthCodeStore
	clientStore   *store.ClientStore
	scopeStore    *store.ScopeStore
	auditStore    *store.AuditStore
	cfg           *config.Config
}

func NewConsentHandler(
	sessionStore *store.SessionStore,
	consentStore *store.ConsentStore,
	authCodeStore *store.AuthCodeStore,
	clientStore *store.ClientStore,
	scopeStore *store.ScopeStore,
	auditStore *store.AuditStore,
	cfg *config.Config,
) *ConsentHandler {
	return &ConsentHandler{
		sessionStore:  sessionStore,
		consentStore:  consentStore,
		authCodeStore: authCodeStore,
		clientStore:   clientStore,
		scopeStore:    scopeStore,
		auditStore:    auditStore,
		cfg:           cfg,
	}
}

func (h *ConsentHandler) Info(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSession(r.Context())
	if session == nil || session.OAuthParams == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "No active authorization request")
		return
	}

	client, err := h.clientStore.GetByClientID(r.Context(), session.OAuthParams.ClientID)
	if err != nil || client == nil {
		writeError(w, http.StatusBadRequest, "invalid_client", "Client not found")
		return
	}

	var scopeNames []string
	if session.OAuthParams.Scope != "" {
		scopeNames = splitScopes(session.OAuthParams.Scope)
	}
	scopes, _ := h.scopeStore.GetByNames(r.Context(), scopeNames)

	type scopeInfo struct {
		Name        string  `json:"name"`
		Description *string `json:"description,omitempty"`
	}
	var scopeInfos []scopeInfo
	for _, s := range scopes {
		scopeInfos = append(scopeInfos, scopeInfo{Name: s.Name, Description: s.Description})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"client_name":        client.Name,
		"client_icon_url":    client.IconURL,
		"client_description": client.Description,
		"requested_scopes":   scopeInfos,
	})
}

type consentRequest struct {
	Approved bool     `json:"approved"`
	Scopes   []string `json:"scopes"`
}

func (h *ConsentHandler) Process(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSession(r.Context())
	if session == nil || session.OAuthParams == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "No active authorization request")
		return
	}

	memberID, ok := middleware.GetMemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not logged in")
		return
	}

	if session.MustChangePassword {
		writeError(w, http.StatusForbidden, "password_change_required", "You must change your password before continuing")
		return
	}

	var req consentRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	params := session.OAuthParams

	if !req.Approved {
		u, err := url.Parse(params.RedirectURI)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid redirect_uri")
			return
		}
		q := u.Query()
		q.Set("error", "access_denied")
		q.Set("error_description", "User denied the request")
		q.Set("state", params.State)
		u.RawQuery = q.Encode()

		sessionID := middleware.GetSessionID(r.Context())
		session.OAuthParams = nil
		h.sessionStore.Update(r.Context(), sessionID, session)

		writeJSON(w, http.StatusOK, map[string]string{
			"redirect_to": u.String(),
		})
		return
	}

	client, err := h.clientStore.GetByClientID(r.Context(), params.ClientID)
	if err != nil || client == nil {
		writeError(w, http.StatusBadRequest, "invalid_client", "Client not found")
		return
	}

	requestedScopes := splitScopes(params.Scope)
	approvedScopes := requestedScopes
	if len(req.Scopes) > 0 {
		approvedScopes = normalizeScopes(req.Scopes)
	}
	if len(approvedScopes) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_scope", "At least one scope must be approved")
		return
	}
	if !scopesCovered(requestedScopes, approvedScopes) {
		writeError(w, http.StatusBadRequest, "invalid_scope", "Approved scopes must be a subset of the original request")
		return
	}
	if ok, err := h.scopesExist(r.Context(), approvedScopes); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to validate scopes")
		return
	} else if !ok {
		writeError(w, http.StatusBadRequest, "invalid_scope", "Unknown scope requested")
		return
	}

	if err := h.consentStore.Upsert(r.Context(), memberID, client.ID, approvedScopes); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to save consent")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionConsentGranted, &memberID, &params.ClientID, r.RemoteAddr, r.UserAgent(), map[string]any{
		"scopes": approvedScopes,
	})

	code, err := oauth.GenerateRandomString(h.cfg.CodeLength)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to generate code")
		return
	}

	scope := params.Scope
	if len(approvedScopes) > 0 {
		scope = joinScopes(approvedScopes)
	}

	var authTime int64
	if session.CreatedAt > 0 {
		authTime = session.CreatedAt
	}

	err = h.authCodeStore.Save(r.Context(), code, &store.AuthCodeData{
		ClientID:            params.ClientID,
		MemberID:            memberID.String(),
		RedirectURI:         params.RedirectURI,
		Scope:               scope,
		CodeChallenge:       params.CodeChallenge,
		CodeChallengeMethod: params.CodeChallengeMethod,
		Nonce:               params.Nonce,
		AuthTime:            authTime,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to store code")
		return
	}

	sessionID := middleware.GetSessionID(r.Context())
	session.OAuthParams = nil
	h.sessionStore.Update(r.Context(), sessionID, session)

	u, err := url.Parse(params.RedirectURI)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid redirect_uri")
		return
	}
	q := u.Query()
	q.Set("code", code)
	q.Set("state", params.State)
	u.RawQuery = q.Encode()
	writeJSON(w, http.StatusOK, map[string]string{
		"redirect_to": u.String(),
	})
}

func splitScopes(scope string) []string {
	return strings.Fields(scope)
}

func normalizeScopes(scopes []string) []string {
	seen := make(map[string]bool, len(scopes))
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		for _, s := range strings.Fields(scope) {
			if !seen[s] {
				seen[s] = true
				result = append(result, s)
			}
		}
	}
	return result
}

func joinScopes(scopes []string) string {
	return strings.Join(scopes, " ")
}

func (h *ConsentHandler) scopesExist(ctx context.Context, scopes []string) (bool, error) {
	found, err := h.scopeStore.GetByNames(ctx, scopes)
	if err != nil {
		return false, err
	}
	return len(found) == len(scopes), nil
}
