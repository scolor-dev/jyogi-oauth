package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jyogi-oauth/auth-server/internal/middleware"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type MeHandler struct {
	memberStore     *store.MemberStore
	sessionStore    *store.SessionStore
	identityHandler *MeIdentityHandler
	cookieName      string
	cookieSecure    bool
	cookieDomain    string
}

func NewMeHandler(memberStore *store.MemberStore, sessionStore *store.SessionStore, pool *pgxpool.Pool, cookieName string, cookieSecure bool, cookieDomain string) *MeHandler {
	return &MeHandler{
		memberStore:     memberStore,
		sessionStore:    sessionStore,
		identityHandler: NewMeIdentityHandler(pool),
		cookieName:      cookieName,
		cookieSecure:    cookieSecure,
		cookieDomain:    cookieDomain,
	}
}

func (h *MeHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	memberID, ok := requireSessionAllowForceChange(w, r)
	if !ok {
		return
	}

	member, err := h.memberStore.GetByID(r.Context(), memberID)
	if err != nil || member == nil {
		writeError(w, http.StatusNotFound, "not_found", "Member not found")
		return
	}

	identity, _ := h.identityHandler.getIdentity(r.Context(), memberID)

	writeJSON(w, http.StatusOK, map[string]any{
		"member":   member,
		"identity": identity,
	})
}

func (h *MeHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID := middleware.GetSessionID(r.Context())
	if sessionID != "" {
		h.sessionStore.Delete(r.Context(), sessionID)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     "/oauth",
		Domain:   h.cookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}
