package handler

import (
	"net/http"
)

type DiscoveryHandler struct {
	issuer string
}

func NewDiscoveryHandler(issuer string) *DiscoveryHandler {
	return &DiscoveryHandler{issuer: issuer}
}

func (h *DiscoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=86400")
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                h.issuer,
		"authorization_endpoint":                h.issuer + "/oauth/authorize",
		"token_endpoint":                        h.issuer + "/oauth/token",
		"userinfo_endpoint":                     h.issuer + "/oauth/userinfo",
		"jwks_uri":                              h.issuer + "/oauth/jwks",
		"scopes_supported":                      []string{"openid", "profile", "identity", "identity:write", "read", "write"},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "client_credentials"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"ES256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
	})
}
