package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type MeConsentHandler struct {
	consentStore *store.ConsentStore
	auditStore   *store.AuditStore
	pool         *pgxpool.Pool
}

func NewMeConsentHandler(consentStore *store.ConsentStore, auditStore *store.AuditStore, pool *pgxpool.Pool) *MeConsentHandler {
	return &MeConsentHandler{consentStore: consentStore, auditStore: auditStore, pool: pool}
}

type consentInfo struct {
	ClientID   string    `json:"client_id"`
	ClientName string    `json:"client_name"`
	Scopes     []string  `json:"scopes"`
	GrantedAt  time.Time `json:"granted_at"`
}

func (h *MeConsentHandler) List(w http.ResponseWriter, r *http.Request) {
	memberID, ok := requireSession(w, r)
	if !ok {
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT c.client_id, cl.name, c.scopes, c.granted_at
		 FROM auth.consent_records c
		 JOIN auth.clients cl ON cl.id = c.client_id
		 WHERE c.member_id = $1 AND c.revoked_at IS NULL
		 ORDER BY c.granted_at DESC`, memberID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list consents")
		return
	}
	defer rows.Close()

	var consents []consentInfo
	for rows.Next() {
		var ci consentInfo
		if err := rows.Scan(&ci.ClientID, &ci.ClientName, &ci.Scopes, &ci.GrantedAt); err != nil {
			continue
		}
		consents = append(consents, ci)
	}

	writeJSON(w, http.StatusOK, map[string]any{"consents": consents})
}

func (h *MeConsentHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	memberID, ok := requireSession(w, r)
	if !ok {
		return
	}

	clientUUID, err := uuid.Parse(r.PathValue("client_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid client ID")
		return
	}

	if err := h.consentStore.Revoke(r.Context(), memberID, clientUUID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to revoke consent")
		return
	}

	h.auditStore.Log(r.Context(), model.ActionConsentRevoked, &memberID, nil, r.RemoteAddr, r.UserAgent(), nil)
	w.WriteHeader(http.StatusNoContent)
}
