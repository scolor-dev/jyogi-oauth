package handler

import (
	"net/http"
	"net/url"
	"time"

	"github.com/jyogi-oauth/auth-server/internal/middleware"
	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type LoginHandler struct {
	memberStore  *store.MemberStore
	sessionStore *store.SessionStore
	auditStore   *store.AuditStore
	rateLimiter  *middleware.RateLimiter
	cookieName   string
	cookieSecure bool
	cookieDomain string
	sessionTTL   time.Duration
	loginLimit   int
}

func NewLoginHandler(
	memberStore *store.MemberStore,
	sessionStore *store.SessionStore,
	auditStore *store.AuditStore,
	rateLimiter *middleware.RateLimiter,
	cookieName string,
	cookieSecure bool,
	cookieDomain string,
	sessionTTL time.Duration,
	loginLimit int,
) *LoginHandler {
	return &LoginHandler{
		memberStore:  memberStore,
		sessionStore: sessionStore,
		auditStore:   auditStore,
		rateLimiter:  rateLimiter,
		cookieName:   cookieName,
		cookieSecure: cookieSecure,
		cookieDomain: cookieDomain,
		sessionTTL:   sessionTTL,
		loginLimit:   loginLimit,
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "username and password are required")
		return
	}

	allowed, _, err := h.rateLimiter.Check(r.Context(), "ratelimit:login:"+req.Username, h.loginLimit, 5*time.Minute)
	if err == nil && !allowed {
		writeError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many login attempts. Try again later.")
		return
	}

	member, err := h.memberStore.GetByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to look up member")
		return
	}

	if member == nil || !member.IsActive {
		h.auditStore.Log(r.Context(), model.ActionLoginFailure, nil, nil, r.RemoteAddr, r.UserAgent(), map[string]string{
			"username": req.Username,
			"reason":   "not_found",
		})
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Username or password is incorrect")
		return
	}

	match, err := oauth.VerifyPassword(req.Password, member.PasswordHash)
	if err != nil || !match {
		h.auditStore.Log(r.Context(), model.ActionLoginFailure, &member.ID, nil, r.RemoteAddr, r.UserAgent(), map[string]string{
			"reason": "bad_password",
		})
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Username or password is incorrect")
		return
	}

	existingSession := middleware.GetSession(r.Context())
	var oauthParams *store.OAuthFlowParams
	if existingSession != nil {
		oauthParams = existingSession.OAuthParams
	}

	sessionData := &store.SessionData{
		MemberID:    member.ID.String(),
		Username:    member.Username,
		IPAddress:   r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		OAuthParams: oauthParams,
	}

	sessionID, err := h.sessionStore.Create(r.Context(), sessionData)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    sessionID,
		Path:     "/oauth",
		Domain:   h.cookieDomain,
		MaxAge:   int(h.sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	h.auditStore.Log(r.Context(), model.ActionLoginSuccess, &member.ID, nil, r.RemoteAddr, r.UserAgent(), nil)

	redirectTo := "/dashboard"
	if oauthParams != nil {
		v := url.Values{}
		v.Set("response_type", "code")
		v.Set("client_id", oauthParams.ClientID)
		v.Set("redirect_uri", oauthParams.RedirectURI)
		v.Set("scope", oauthParams.Scope)
		v.Set("state", oauthParams.State)
		v.Set("code_challenge", oauthParams.CodeChallenge)
		v.Set("code_challenge_method", oauthParams.CodeChallengeMethod)
		redirectTo = "/oauth/authorize?" + v.Encode()
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"redirect_to": redirectTo,
	})
}
