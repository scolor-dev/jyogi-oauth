package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jyogi-oauth/auth-server/internal/middleware"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func readJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB
	return json.NewDecoder(r.Body).Decode(v)
}

func parsePagination(r *http.Request) (page, perPage int) {
	page = queryInt(r, "page", 1)
	perPage = queryInt(r, "per_page", 20)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return
}

func requireSession(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	memberID, ok := middleware.GetMemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not logged in")
		return uuid.UUID{}, false
	}
	return memberID, true
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
