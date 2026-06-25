package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jyogi-oauth/auth-server/internal/middleware"
	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type MeClientHandler struct {
	clientStore *store.ClientStore
	auditStore  *store.AuditStore
}

func NewMeClientHandler(clientStore *store.ClientStore, auditStore *store.AuditStore) *MeClientHandler {
	return &MeClientHandler{clientStore: clientStore, auditStore: auditStore}
}

func (h *MeClientHandler) requireMember(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	memberID, ok := middleware.GetMemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not logged in")
		return uuid.UUID{}, false
	}
	return memberID, true
}

func (h *MeClientHandler) List(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.requireMember(w, r)
	if !ok {
		return
	}

	clients, total, err := h.clientStore.ListByCreator(r.Context(), memberID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list clients")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"clients": clients,
		"total":   total,
	})
}

type createMyClientRequest struct {
	Name              string   `json:"name"`
	ClientType        string   `json:"client_type"`
	RedirectURIs      []string `json:"redirect_uris"`
	AllowedGrantTypes []string `json:"allowed_grant_types"`
	Description       *string  `json:"description,omitempty"`
}

func (h *MeClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.requireMember(w, r)
	if !ok {
		return
	}

	var req createMyClientRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Name == "" || len(req.RedirectURIs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "name and redirect_uris are required")
		return
	}
	if req.ClientType == "" {
		req.ClientType = "public"
	}
	if len(req.AllowedGrantTypes) == 0 {
		req.AllowedGrantTypes = []string{"authorization_code"}
	}

	clientID := uuid.New().String()[:8]

	var secretHash *string
	var plainSecret string
	if req.ClientType == "confidential" {
		secret, err := oauth.GenerateRandomString(32)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to generate secret")
			return
		}
		plainSecret = secret
		h := oauth.HashClientSecret(secret)
		secretHash = &h
	}

	client, err := h.clientStore.Create(r.Context(),
		clientID, req.Name, req.ClientType,
		secretHash, req.Description,
		req.RedirectURIs, req.AllowedGrantTypes, &memberID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create client")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionClientCreated, &memberID, &client.ClientID, r.RemoteAddr, r.UserAgent(), nil)

	resp := map[string]any{"client": client}
	if plainSecret != "" {
		resp["client_secret"] = plainSecret
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *MeClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.requireMember(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid client ID")
		return
	}

	client, err := h.clientStore.GetByID(r.Context(), id)
	if err != nil || client == nil {
		writeError(w, http.StatusNotFound, "not_found", "Client not found")
		return
	}
	if client.CreatedBy == nil || *client.CreatedBy != memberID {
		writeError(w, http.StatusForbidden, "forbidden", "You can only edit your own clients")
		return
	}

	var req struct {
		Name              *string  `json:"name,omitempty"`
		RedirectURIs      []string `json:"redirect_uris,omitempty"`
		AllowedGrantTypes []string `json:"allowed_grant_types,omitempty"`
		Description       *string  `json:"description,omitempty"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if err := h.clientStore.Update(r.Context(), id, req.Name, req.RedirectURIs, req.AllowedGrantTypes, nil); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update client")
		return
	}

	updated, _ := h.clientStore.GetByID(r.Context(), id)
	writeJSON(w, http.StatusOK, updated)
}

func (h *MeClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.requireMember(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid client ID")
		return
	}

	client, err := h.clientStore.GetByID(r.Context(), id)
	if err != nil || client == nil {
		writeError(w, http.StatusNotFound, "not_found", "Client not found")
		return
	}
	if client.CreatedBy == nil || *client.CreatedBy != memberID {
		writeError(w, http.StatusForbidden, "forbidden", "You can only delete your own clients")
		return
	}

	if err := h.clientStore.Deactivate(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete client")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionClientDeleted, &memberID, &client.ClientID, r.RemoteAddr, r.UserAgent(), nil)
	w.WriteHeader(http.StatusNoContent)
}
