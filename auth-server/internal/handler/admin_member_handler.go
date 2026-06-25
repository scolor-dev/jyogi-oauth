package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

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
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func (h *AdminMemberHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createMemberRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Username == "" || req.Password == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "username, password, and email are required")
		return
	}

	if err := oauth.ValidatePassword(req.Password, req.Username); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_password", err.Error())
		return
	}

	hash, err := oauth.HashPassword(req.Password, h.pwConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to hash password")
		return
	}

	member, err := h.memberStore.Create(r.Context(), req.Username, hash, req.Email)
	if err != nil {
		writeError(w, http.StatusConflict, "conflict", "Username or email already exists")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionMemberCreated, &member.ID, nil, r.RemoteAddr, r.UserAgent(), map[string]string{
		"username": member.Username,
	})

	writeJSON(w, http.StatusCreated, member)
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

	if err := h.memberStore.Deactivate(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to deactivate member")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionMemberDeactivated, &id, nil, r.RemoteAddr, r.UserAgent(), nil)

	w.WriteHeader(http.StatusNoContent)
}
