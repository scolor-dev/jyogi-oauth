package handler

import (
	"net/http"

	"github.com/jyogi-oauth/auth-server/internal/model"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

type RevokeHandler struct {
	refreshStore *store.RefreshStore
	clientStore  *store.ClientStore
	auditStore   *store.AuditStore
}

func NewRevokeHandler(refreshStore *store.RefreshStore, clientStore *store.ClientStore, auditStore *store.AuditStore) *RevokeHandler {
	return &RevokeHandler{refreshStore: refreshStore, clientStore: clientStore, auditStore: auditStore}
}

func (h *RevokeHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	token := r.FormValue("token")
	if token == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	client, _, _ := oauth.AuthenticateClient(r.Context(), r, h.clientStore)

	tokenData, _ := h.refreshStore.Get(r.Context(), token)
	if tokenData != nil {
		h.refreshStore.Delete(r.Context(), token, tokenData.MemberID)

		memberID := mustParseUUID(tokenData.MemberID)
		var clientIDStr *string
		if client != nil {
			clientIDStr = &client.ClientID
		}
		h.auditStore.Log(r.Context(), model.ActionTokenRevoked, &memberID, clientIDStr, r.RemoteAddr, r.UserAgent(), nil)
	}

	w.WriteHeader(http.StatusOK)
}
