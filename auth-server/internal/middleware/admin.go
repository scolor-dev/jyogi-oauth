package middleware

import (
	"context"
	"net/http"

	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type AdminMiddleware struct {
	memberStore *store.MemberStore
}

func NewAdminMiddleware(memberStore *store.MemberStore) *AdminMiddleware {
	return &AdminMiddleware{memberStore: memberStore}
}

func (m *AdminMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return m.requireRole(next, "admin")
}

func (m *AdminMiddleware) RequireModerator(next http.Handler) http.Handler {
	return m.requireRole(next, "moderator")
}

func (m *AdminMiddleware) requireRole(next http.Handler, minRole string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		memberID, ok := GetMemberID(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error": map[string]string{"code": "unauthorized", "message": "Not logged in"},
			})
			return
		}

		session := GetSession(r.Context())
		if session != nil && session.MustChangePassword {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"error": map[string]string{"code": "password_change_required", "message": "You must change your password before continuing"},
			})
			return
		}

		member, err := m.memberStore.GetByID(r.Context(), memberID)
		if err != nil || member == nil || !member.IsActive {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error": map[string]string{"code": "unauthorized", "message": "Member not found"},
			})
			return
		}

		allowed := false
		switch minRole {
		case "admin":
			allowed = member.IsAdmin()
		case "moderator":
			allowed = member.IsModerator()
		}

		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"error": map[string]string{"code": "forbidden", "message": "Insufficient permissions"},
			})
			return
		}

		ctx := context.WithValue(r.Context(), adminMemberKey, member)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type adminContextKey string

const adminMemberKey adminContextKey = "admin_member"

func GetAdminMember(ctx context.Context) *model.Member {
	m, _ := ctx.Value(adminMemberKey).(*model.Member)
	return m
}
