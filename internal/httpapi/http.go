package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// NewHandler exposes the developer scaffold and stateless graph validation.
// It does not execute agents or persist workflows.
func NewHandler() http.Handler {
	return NewHandlerWithReadiness(nil)
}

func NewHandlerWithReadiness(ready func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if ready == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "storage_not_configured"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if ready(ctx) != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1/agents", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, []struct{}{})
	})
	mux.HandleFunc("GET /api/v1/workflows", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, []struct{}{})
	})
	mux.HandleFunc("/api/v1/workflows/validate", validateWorkflow)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
