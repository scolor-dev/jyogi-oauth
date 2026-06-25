package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type contextKey string

const (
	sessionDataKey contextKey = "session_data"
	sessionIDKey   contextKey = "session_id"
)

type SessionMiddleware struct {
	sessionStore *store.SessionStore
	cookieName   string
}

func NewSessionMiddleware(sessionStore *store.SessionStore, cookieName string) *SessionMiddleware {
	return &SessionMiddleware{sessionStore: sessionStore, cookieName: cookieName}
}

func (m *SessionMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(m.cookieName)
		if err == nil && cookie.Value != "" {
			data, err := m.sessionStore.Get(r.Context(), cookie.Value)
			if err == nil && data != nil {
				ctx := context.WithValue(r.Context(), sessionDataKey, data)
				ctx = context.WithValue(ctx, sessionIDKey, cookie.Value)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func GetSession(ctx context.Context) *store.SessionData {
	data, _ := ctx.Value(sessionDataKey).(*store.SessionData)
	return data
}

func GetSessionID(ctx context.Context) string {
	id, _ := ctx.Value(sessionIDKey).(string)
	return id
}

func GetMemberID(ctx context.Context) (uuid.UUID, bool) {
	session := GetSession(ctx)
	if session == nil {
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(session.MemberID)
	if err != nil {
		return uuid.UUID{}, false
	}
	return id, true
}
