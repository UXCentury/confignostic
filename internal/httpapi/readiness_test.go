package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadinessIsSeparateFromLiveness(t *testing.T) {
	for _, tc := range []struct {
		name   string
		probe  func(context.Context) error
		status int
	}{
		{"not configured", nil, 503},
		{"ready", func(ctx context.Context) error {
			if _, ok := ctx.Deadline(); !ok {
				t.Error("readiness has no deadline")
			}
			return nil
		}, 200},
		{"unavailable", func(context.Context) error { return errors.New("synthetic-secret-canary") }, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandlerWithReadiness(tc.probe)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if response.Code != tc.status || strings.Contains(response.Body.String(), "secret") {
				t.Fatalf("unsafe readiness response: %d %s", response.Code, response.Body)
			}
			live := httptest.NewRecorder()
			handler.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/healthz", nil))
			if live.Code != 200 {
				t.Fatal("liveness depends on storage")
			}
		})
	}
}
