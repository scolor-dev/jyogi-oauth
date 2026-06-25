package handler

import (
	"net/http"

	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type MePasswordHandler struct {
	memberStore *store.MemberStore
	auditStore  *store.AuditStore
	pwConfig    *oauth.PasswordConfig
}

func NewMePasswordHandler(memberStore *store.MemberStore, auditStore *store.AuditStore, pwConfig *oauth.PasswordConfig) *MePasswordHandler {
	return &MePasswordHandler{memberStore: memberStore, auditStore: auditStore, pwConfig: pwConfig}
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *MePasswordHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	memberID, ok := requireSession(w, r)
	if !ok {
		return
	}

	var req changePasswordRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "current_password and new_password are required")
		return
	}

	member, err := h.memberStore.GetByID(r.Context(), memberID)
	if err != nil || member == nil {
		writeError(w, http.StatusNotFound, "not_found", "Member not found")
		return
	}

	match, err := oauth.VerifyPassword(req.CurrentPassword, member.PasswordHash)
	if err != nil || !match {
		writeError(w, http.StatusBadRequest, "invalid_password", "Current password is incorrect")
		return
	}

	if err := oauth.ValidatePassword(req.NewPassword, member.Username); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_password", err.Error())
		return
	}

	hash, err := oauth.HashPassword(req.NewPassword, h.pwConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to hash password")
		return
	}

	if err := h.memberStore.UpdatePassword(r.Context(), memberID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update password")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionMemberUpdated, &memberID, nil, r.RemoteAddr, r.UserAgent(), map[string]any{"field": "password"})
	writeJSON(w, http.StatusOK, map[string]string{"status": "password_changed"})
}
