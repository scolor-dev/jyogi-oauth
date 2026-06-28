package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jyogi-oauth/auth-server/internal/middleware"
	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type AdminScopeHandler struct {
	scopeStore *store.ScopeStore
	auditStore *store.AuditStore
}

func NewAdminScopeHandler(scopeStore *store.ScopeStore, auditStore *store.AuditStore) *AdminScopeHandler {
	return &AdminScopeHandler{scopeStore: scopeStore, auditStore: auditStore}
}

func (h *AdminScopeHandler) List(w http.ResponseWriter, r *http.Request) {
	scopes, err := h.scopeStore.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list scopes")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scopes": scopes, "total": len(scopes)})
}

type scopeRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsDefault   *bool   `json:"is_default,omitempty"`
}

func (h *AdminScopeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req scopeRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Name == nil || *req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}

	scope, err := h.scopeStore.Create(r.Context(), *req.Name, req.Description, isDefault)
	if err != nil {
		writeError(w, http.StatusConflict, "conflict", "Scope name already exists")
		return
	}

	operatorID := adminOperatorID(r)
	h.auditStore.Log(r.Context(), model.ActionScopeCreated, operatorID, nil, r.RemoteAddr, r.UserAgent(), map[string]string{"scope": scope.Name})
	writeJSON(w, http.StatusCreated, scope)
}

func (h *AdminScopeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid scope ID")
		return
	}

	var req scopeRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if err := h.scopeStore.Update(r.Context(), id, req.Name, req.Description, req.IsDefault); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update scope")
		return
	}
	scope, err := h.scopeStore.GetByID(r.Context(), id)
	if err != nil || scope == nil {
		writeError(w, http.StatusNotFound, "not_found", "Scope not found")
		return
	}

	operatorID := adminOperatorID(r)
	h.auditStore.Log(r.Context(), model.ActionScopeUpdated, operatorID, nil, r.RemoteAddr, r.UserAgent(), map[string]string{"scope": scope.Name})
	writeJSON(w, http.StatusOK, scope)
}

func (h *AdminScopeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid scope ID")
		return
	}

	scope, err := h.scopeStore.GetByID(r.Context(), id)
	if err != nil || scope == nil {
		writeError(w, http.StatusNotFound, "not_found", "Scope not found")
		return
	}
	if err := h.scopeStore.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete scope")
		return
	}

	operatorID := adminOperatorID(r)
	h.auditStore.Log(r.Context(), model.ActionScopeDeleted, operatorID, nil, r.RemoteAddr, r.UserAgent(), map[string]string{"scope": scope.Name})
	w.WriteHeader(http.StatusNoContent)
}

func adminOperatorID(r *http.Request) *uuid.UUID {
	operator := middleware.GetAdminMember(r.Context())
	if operator == nil {
		return nil
	}
	return &operator.ID
}
