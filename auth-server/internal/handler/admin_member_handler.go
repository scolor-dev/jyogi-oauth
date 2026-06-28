package handler

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jyogi-oauth/auth-server/internal/middleware"
	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

func (h *AdminMemberHandler) checkPrivilege(w http.ResponseWriter, r *http.Request, targetID uuid.UUID) (*model.Member, bool) {
	operator := middleware.GetAdminMember(r.Context())
	if operator == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not logged in")
		return nil, false
	}

	if operator.ID == targetID {
		writeError(w, http.StatusForbidden, "forbidden", "Cannot modify yourself via Admin API")
		return nil, false
	}

	target, err := h.memberStore.GetByID(r.Context(), targetID)
	if err != nil || target == nil {
		writeError(w, http.StatusNotFound, "not_found", "Member not found")
		return nil, false
	}

	if target.IsRoot() {
		writeError(w, http.StatusForbidden, "forbidden", "Cannot modify root user")
		return nil, false
	}

	if !model.CanManage(operator.Role, target.Role) {
		writeError(w, http.StatusForbidden, "forbidden", "Insufficient permissions to manage this member")
		return nil, false
	}

	return target, true
}

type AdminMemberHandler struct {
	memberStore *store.MemberStore
	auditStore  *store.AuditStore
	pwConfig    *oauth.PasswordConfig
}

func NewAdminMemberHandler(memberStore *store.MemberStore, auditStore *store.AuditStore, pwConfig *oauth.PasswordConfig) *AdminMemberHandler {
	return &AdminMemberHandler{memberStore: memberStore, auditStore: auditStore, pwConfig: pwConfig}
}

func (h *AdminMemberHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := parsePagination(r)
	members, total, err := h.memberStore.List(r.Context(), page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list members")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"members":  members,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

type createMemberRequest struct {
	Username           string `json:"username"`
	Password           string `json:"password,omitempty"`
	Email              string `json:"email"`
	MustChangePassword *bool  `json:"must_change_password,omitempty"`
}

func (h *AdminMemberHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createMemberRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Username == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "username and email are required")
		return
	}

	mustChange := true
	if req.MustChangePassword != nil {
		mustChange = *req.MustChangePassword
	}

	var password string
	var tempPassword string

	if req.Password != "" {
		password = req.Password
	} else {
		generated, err := oauth.GenerateRandomString(9)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to generate password")
			return
		}
		password = generated
		tempPassword = generated
	}

	if err := oauth.ValidatePassword(password, req.Username); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_password", err.Error())
		return
	}

	hash, err := oauth.HashPassword(password, h.pwConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to hash password")
		return
	}

	member, err := h.memberStore.Create(r.Context(), req.Username, hash, req.Email)
	if err != nil {
		writeError(w, http.StatusConflict, "conflict", "Username or email already exists")
		return
	}

	if mustChange {
		if err := h.memberStore.SetMustChangePassword(r.Context(), member.ID, true); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to set password change flag")
			return
		}
		member.MustChangePassword = true
	}

	h.auditStore.Log(r.Context(), model.ActionMemberCreated, &member.ID, nil, r.RemoteAddr, r.UserAgent(), map[string]string{
		"username":             member.Username,
		"must_change_password": fmt.Sprintf("%v", mustChange),
	})

	resp := map[string]any{"member": member}
	if tempPassword != "" {
		resp["temporary_password"] = tempPassword
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *AdminMemberHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid member ID")
		return
	}

	member, err := h.memberStore.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get member")
		return
	}
	if member == nil {
		writeError(w, http.StatusNotFound, "not_found", "Member not found")
		return
	}

	writeJSON(w, http.StatusOK, member)
}

type updateMemberRequest struct {
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
	Role     *string `json:"role,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

func (h *AdminMemberHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid member ID")
		return
	}

	if _, ok := h.checkPrivilege(w, r, id); !ok {
		return
	}

	operator := middleware.GetAdminMember(r.Context())

	var req updateMemberRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Password != nil {
		username := ""
		if req.Username != nil {
			username = *req.Username
		}
		if err := oauth.ValidatePassword(*req.Password, username); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_password", err.Error())
			return
		}
		hash, err := oauth.HashPassword(*req.Password, h.pwConfig)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to hash password")
			return
		}
		if err := h.memberStore.UpdatePassword(r.Context(), id, hash); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update password")
			return
		}
	}

	if req.Role != nil {
		if !operator.IsAdmin() {
			writeError(w, http.StatusForbidden, "forbidden", "Only admin can change roles")
			return
		}
		valid := *req.Role == model.RoleMember || *req.Role == model.RoleModerator || *req.Role == model.RoleAdmin
		if !valid {
			writeError(w, http.StatusBadRequest, "invalid_request", "role must be member, moderator, or admin")
			return
		}
		if err := h.memberStore.UpdateRole(r.Context(), id, *req.Role); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update role")
			return
		}
	}

	if err := h.memberStore.Update(r.Context(), id, req.Username, req.Email, req.IsActive); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update member")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionMemberUpdated, &id, nil, r.RemoteAddr, r.UserAgent(), nil)

	member, _ := h.memberStore.GetByID(r.Context(), id)
	writeJSON(w, http.StatusOK, member)
}

func (h *AdminMemberHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid member ID")
		return
	}

	if _, ok := h.checkPrivilege(w, r, id); !ok {
		return
	}

	if err := h.memberStore.Deactivate(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to deactivate member")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionMemberDeactivated, &id, nil, r.RemoteAddr, r.UserAgent(), nil)

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminMemberHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid member ID")
		return
	}

	member, ok := h.checkPrivilege(w, r, id)
	if !ok {
		return
	}

	tempPassword, err := oauth.GenerateRandomString(9)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to generate password")
		return
	}

	hash, err := oauth.HashPassword(tempPassword, h.pwConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to hash password")
		return
	}

	if err := h.memberStore.UpdatePassword(r.Context(), id, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update password")
		return
	}

	if err := h.memberStore.SetMustChangePassword(r.Context(), id, true); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to set password reset flag")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionPasswordReset, &id, nil, r.RemoteAddr, r.UserAgent(), map[string]string{
		"target_username": member.Username,
	})

	writeJSON(w, http.StatusOK, map[string]string{
		"temporary_password": tempPassword,
		"message":            "Password has been reset. Member must change it on next login.",
	})
}
