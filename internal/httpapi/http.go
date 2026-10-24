package httpapi

import (
	"encoding/json"
	"net/http"
)

// NewHandler exposes the read-only developer scaffold, not real agent execution.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/agents", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, []struct{}{})
	})
	mux.HandleFunc("GET /api/v1/workflows", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, []struct{}{})
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
