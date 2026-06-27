package handler

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/jyogi-oauth/auth-server/internal/config"
	"github.com/jyogi-oauth/auth-server/internal/middleware"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type AuthorizeHandler struct {
	clientStore   *store.ClientStore
	consentStore  *store.ConsentStore
	sessionStore  *store.SessionStore
	authCodeStore *store.AuthCodeStore
	scopeStore    *store.ScopeStore
	auditStore    *store.AuditStore
	cfg           *config.Config
}

func NewAuthorizeHandler(
	clientStore *store.ClientStore,
	consentStore *store.ConsentStore,
	sessionStore *store.SessionStore,
	authCodeStore *store.AuthCodeStore,
	scopeStore *store.ScopeStore,
	auditStore *store.AuditStore,
	cfg *config.Config,
) *AuthorizeHandler {
	return &AuthorizeHandler{
		clientStore:   clientStore,
		consentStore:  consentStore,
		sessionStore:  sessionStore,
		authCodeStore: authCodeStore,
		scopeStore:    scopeStore,
		auditStore:    auditStore,
		cfg:           cfg,
	}
}

func (h *AuthorizeHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	responseType := q.Get("response_type")
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	scope := q.Get("scope")
	state := q.Get("state")
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")
	nonce := q.Get("nonce")

	if responseType != "code" {
		writeError(w, http.StatusBadRequest, "unsupported_response_type", "Only 'code' is supported")
		return
	}
	if clientID == "" || redirectURI == "" || state == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "client_id, redirect_uri, and state are required")
		return
	}
	if codeChallenge == "" || codeChallengeMethod != "S256" {
		writeError(w, http.StatusBadRequest, "invalid_request", "code_challenge (S256) is required")
		return
	}

	client, err := h.clientStore.GetByClientID(r.Context(), clientID)
	if err != nil || client == nil || !client.IsActive {
		writeError(w, http.StatusBadRequest, "invalid_client", "Unknown or inactive client")
		return
	}

	validRedirect := false
	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			validRedirect = true
			break
		}
	}
	if !validRedirect {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid redirect_uri")
		return
	}

	session := middleware.GetSession(r.Context())
	oauthParams := &store.OAuthFlowParams{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		State:               state,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		Nonce:               nonce,
	}

	// Not logged in: save OAuth params and redirect to login
	if session == nil {
		tempSession := &store.SessionData{
			OAuthParams: oauthParams,
		}
		sessionID, err := h.sessionStore.Create(r.Context(), tempSession)
		if err != nil {
			httpRedirectWithError(w, r, redirectURI, state, "server_error", "Failed to create session")
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     h.cfg.SessionCookieName,
			Value:    sessionID,
			Path:     "/oauth",
			Domain:   h.cfg.SessionCookieDomain,
			MaxAge:   int(h.cfg.SessionTTL.Seconds()),
			HttpOnly: true,
			Secure:   h.cfg.SessionCookieSecure,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Session exists but not authenticated
	memberID, ok := middleware.GetMemberID(r.Context())
	if !ok {
		sessionID := middleware.GetSessionID(r.Context())
		session.OAuthParams = oauthParams
		h.sessionStore.Update(r.Context(), sessionID, session)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if session.MustChangePassword {
		http.Redirect(w, r, "/change-password", http.StatusFound)
		return
	}

	// Check existing consent
	consent, err := h.consentStore.GetByMemberAndClient(r.Context(), memberID, client.ID)
	if err != nil {
		httpRedirectWithError(w, r, redirectURI, state, "server_error", "Failed to check consent")
		return
	}

	requestedScopes := strings.Split(scope, " ")
	if consent != nil && scopesCovered(consent.Scopes, requestedScopes) {
		// Already consented: issue auth code and redirect to client
		h.issueAuthCodeAndRedirect(w, r, client.ClientID, memberID.String(), redirectURI, scope, codeChallenge, codeChallengeMethod, state, nonce)
		return
	}

	// Need consent: save params and redirect to consent page
	sessionID := middleware.GetSessionID(r.Context())
	session.OAuthParams = oauthParams
	h.sessionStore.Update(r.Context(), sessionID, session)
	http.Redirect(w, r, "/consent", http.StatusFound)
}

func (h *AuthorizeHandler) issueAuthCodeAndRedirect(w http.ResponseWriter, r *http.Request, clientID, memberID, redirectURI, scope, codeChallenge, codeChallengeMethod, state, nonce string) {
	code, err := oauth.GenerateRandomString(h.cfg.CodeLength)
	if err != nil {
		httpRedirectWithError(w, r, redirectURI, state, "server_error", "Failed to generate code")
		return
	}

	var authTime int64
	if session := middleware.GetSession(r.Context()); session != nil {
		authTime = session.CreatedAt
	}

	err = h.authCodeStore.Save(r.Context(), code, &store.AuthCodeData{
		ClientID:            clientID,
		MemberID:            memberID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		CodeChallenge:       codeChallenge,
		Nonce:               nonce,
		CodeChallengeMethod: codeChallengeMethod,
		AuthTime:            authTime,
	})
	if err != nil {
		httpRedirectWithError(w, r, redirectURI, state, "server_error", "Failed to store code")
		return
	}

	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("code", code)
	q.Set("state", state)
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}

func httpRedirectWithError(w http.ResponseWriter, r *http.Request, redirectURI, state, errorCode, description string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		writeError(w, http.StatusBadRequest, errorCode, description)
		return
	}
	q := u.Query()
	q.Set("error", errorCode)
	q.Set("error_description", description)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func scopesCovered(consented, requested []string) bool {
	set := make(map[string]bool, len(consented))
	for _, s := range consented {
		set[s] = true
	}
	for _, s := range requested {
		if !set[s] {
			return false
		}
	}
	return true
}
