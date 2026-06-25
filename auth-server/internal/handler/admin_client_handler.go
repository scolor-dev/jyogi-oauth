package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type AdminClientHandler struct {
	clientStore *store.ClientStore
	auditStore  *store.AuditStore
}

func NewAdminClientHandler(clientStore *store.ClientStore, auditStore *store.AuditStore) *AdminClientHandler {
	return &AdminClientHandler{clientStore: clientStore, auditStore: auditStore}
}

func (h *AdminClientHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := parsePagination(r)
	clients, total, err := h.clientStore.List(r.Context(), page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list clients")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clients":  clients,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

type createClientRequest struct {
	ClientID          string   `json:"client_id"`
	Name              string   `json:"name"`
	ClientType        string   `json:"client_type"`
	RedirectURIs      []string `json:"redirect_uris"`
	AllowedGrantTypes []string `json:"allowed_grant_types"`
	Description       *string  `json:"description,omitempty"`
}

type createClientResponse struct {
	*model.Client
	ClientSecret string `json:"client_secret,omitempty"`
}

func (h *AdminClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createClientRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Name == "" || len(req.RedirectURIs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "name and redirect_uris are required")
		return
	}

	if req.ClientType == "" {
		req.ClientType = "confidential"
	}
	if len(req.AllowedGrantTypes) == 0 {
		req.AllowedGrantTypes = []string{"authorization_code"}
	}
	if req.ClientID == "" {
		req.ClientID = uuid.New().String()[:8]
	}

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
		req.ClientID, req.Name, req.ClientType,
		secretHash, req.Description,
		req.RedirectURIs, req.AllowedGrantTypes, nil,
	)
	if err != nil {
		writeError(w, http.StatusConflict, "conflict", "Client ID already exists")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionClientCreated, nil, &client.ClientID, r.RemoteAddr, r.UserAgent(), nil)

	resp := createClientResponse{Client: client, ClientSecret: plainSecret}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *AdminClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid client ID")
		return
	}

	client, err := h.clientStore.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get client")
		return
	}
	if client == nil {
		writeError(w, http.StatusNotFound, "not_found", "Client not found")
		return
	}

	writeJSON(w, http.StatusOK, client)
}

func (h *AdminClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid client ID")
		return
	}

	var req struct {
		Name              *string  `json:"name,omitempty"`
		RedirectURIs      []string `json:"redirect_uris,omitempty"`
		AllowedGrantTypes []string `json:"allowed_grant_types,omitempty"`
		IsActive          *bool    `json:"is_active,omitempty"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if err := h.clientStore.Update(r.Context(), id, req.Name, req.RedirectURIs, req.AllowedGrantTypes, req.IsActive); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update client")
		return
	}

	client, _ := h.clientStore.GetByID(r.Context(), id)
	writeJSON(w, http.StatusOK, client)
}

func (h *AdminClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	if err := h.clientStore.Deactivate(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete client")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionClientDeleted, nil, &client.ClientID, r.RemoteAddr, r.UserAgent(), nil)
	w.WriteHeader(http.StatusNoContent)
}
