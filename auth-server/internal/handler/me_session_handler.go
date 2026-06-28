package handler

import (
	"net/http"

	"github.com/jyogi-oauth/auth-server/internal/middleware"
	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type MeSessionHandler struct {
	sessionStore *store.SessionStore
	auditStore   *store.AuditStore
}

func NewMeSessionHandler(sessionStore *store.SessionStore, auditStore *store.AuditStore) *MeSessionHandler {
	return &MeSessionHandler{sessionStore: sessionStore, auditStore: auditStore}
}

func (h *MeSessionHandler) List(w http.ResponseWriter, r *http.Request) {
	memberID, ok := middleware.GetMemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not logged in")
		return
	}

	currentHashedID := store.HashSessionID(middleware.GetSessionID(r.Context()))

	sessions, err := h.sessionStore.ListByMember(r.Context(), memberID.String())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list sessions")
		return
	}

	for _, s := range sessions {
		s["is_current"] = s["session_id"] == currentHashedID
	}

	if sessions == nil {
		sessions = []map[string]any{}
	}

	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (h *MeSessionHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	memberID, ok := middleware.GetMemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not logged in")
		return
	}

	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session_id is required")
		return
	}

	currentHashedID := store.HashSessionID(middleware.GetSessionID(r.Context()))
	if sessionID == currentHashedID {
		writeError(w, http.StatusBadRequest, "invalid_request", "Cannot revoke current session")
		return
	}

	if err := h.sessionStore.DeleteByHashedID(r.Context(), sessionID, memberID.String()); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Session not found")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionSessionRevoked, &memberID, nil, r.RemoteAddr, r.UserAgent(), map[string]string{
		"revoked_session_id": sessionID,
	})

	w.WriteHeader(http.StatusNoContent)
}
