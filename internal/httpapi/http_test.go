package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadOnlyEndpoints(t *testing.T) {
	for _, tc := range []struct{ path, want string }{
		{"/healthz", `{"status":"ok"}`},
		{"/api/v1/agents", `[]`},
		{"/api/v1/workflows", `[]`},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d, expected 200", rec.Code)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestUnsupportedMethod(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/agents", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d, expected 405", rec.Code)
	}
}
