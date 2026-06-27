package handler

import (
	"net/http"
	"net/url"

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
		"client_name":      client.Name,
		"requested_scopes": scopeInfos,
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
		u, _ := url.Parse(params.RedirectURI)
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

	if err := h.consentStore.Upsert(r.Context(), memberID, client.ID, req.Scopes); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to save consent")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionConsentGranted, &memberID, &params.ClientID, r.RemoteAddr, r.UserAgent(), map[string]any{
		"scopes": req.Scopes,
	})

	code, err := oauth.GenerateRandomString(h.cfg.CodeLength)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to generate code")
		return
	}

	scope := params.Scope
	if len(req.Scopes) > 0 {
		scope = joinScopes(req.Scopes)
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

	redirectURI := params.RedirectURI + "?code=" + code + "&state=" + params.State
	writeJSON(w, http.StatusOK, map[string]string{
		"redirect_to": redirectURI,
	})
}

func splitScopes(scope string) []string {
	var scopes []string
	for _, s := range splitBySpace(scope) {
		if s != "" {
			scopes = append(scopes, s)
		}
	}
	return scopes
}

func splitBySpace(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ' ' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func joinScopes(scopes []string) string {
	result := ""
	for i, s := range scopes {
		if i > 0 {
			result += " "
		}
		result += s
	}
	return result
}
