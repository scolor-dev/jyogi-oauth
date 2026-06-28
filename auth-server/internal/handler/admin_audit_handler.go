package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type AdminAuditHandler struct {
	auditStore *store.AuditStore
}

func NewAdminAuditHandler(auditStore *store.AuditStore) *AdminAuditHandler {
	return &AdminAuditHandler{auditStore: auditStore}
}

func (h *AdminAuditHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := parsePagination(r)
	q := r.URL.Query()

	var memberID *uuid.UUID
	if raw := q.Get("member_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "member_id must be a UUID")
			return
		}
		memberID = &id
	}

	from, ok := parseOptionalTime(w, q.Get("from"), "from")
	if !ok {
		return
	}
	to, ok := parseOptionalTime(w, q.Get("to"), "to")
	if !ok {
		return
	}

	items, total, err := h.auditStore.List(r.Context(), store.AuditLogFilter{
		Action:   q.Get("action"),
		MemberID: memberID,
		From:     from,
		To:       to,
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list audit logs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":    items,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func parseOptionalTime(w http.ResponseWriter, value, name string) (*time.Time, bool) {
	if value == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", name+" must be RFC3339")
		return nil, false
	}
	return &t, true
}
