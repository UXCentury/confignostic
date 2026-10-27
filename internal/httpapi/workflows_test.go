package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	v1 "github.com/UXCentury/confignostic/internal/contracts/v1"
)

func workflowJSON(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../contracts/v1/testdata/workflow.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func TestWorkflowValidationIsStatelessAndDoesNotClaimExecution(t *testing.T) {
	handler := NewHandler()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", bytes.NewReader(workflowJSON(t)))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		SchemaVersion       string `json:"schema_version"`
		Valid               bool   `json:"valid"`
		CapabilitiesChecked bool   `json:"capabilities_checked"`
		Executable          bool   `json:"executable"`
		RequestID           string `json:"request_id"`
	}
	if json.Unmarshal(response.Body.Bytes(), &body) != nil || !body.Valid || body.Executable || body.CapabilitiesChecked || body.SchemaVersion != "1" {
		t.Fatalf("misleading validation result: %s", response.Body)
	}
	if body.RequestID == "" || response.Header().Get("X-Request-ID") != body.RequestID || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing request correlation/cache policy")
	}
	listing := httptest.NewRecorder()
	handler.ServeHTTP(listing, httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil))
	if strings.TrimSpace(listing.Body.String()) != "[]" {
		t.Fatal("validation persisted a workflow")
	}
}
func TestWorkflowHTTPFailuresUseSafeVersionedErrors(t *testing.T) {
	base := workflowJSON(t)
	var cyclic map[string]any
	if err := json.Unmarshal(base, &cyclic); err != nil {
		t.Fatal(err)
	}
	cyclic["edges"] = append(cyclic["edges"].([]any), map[string]any{"from": map[string]any{"node_id": "review"}, "to": map[string]any{"node_id": "implement"}})
	cycle, _ := json.Marshal(cyclic)
	unknown := append([]byte(`{"credentials":"synthetic-secret-canary",`), bytes.TrimSpace(base)[1:]...)
	for _, tc := range []struct {
		name, method, contentType string
		data                      []byte
		status                    int
		code                      string
	}{
		{"method", http.MethodGet, "application/json", base, 405, "unsupported_operation"},
		{"media type", http.MethodPost, "text/plain", base, 415, "invalid_schema"},
		{"malformed", http.MethodPost, "application/json", []byte("{"), 400, "invalid_json"},
		{"unknown version", http.MethodPost, "application/json", bytes.Replace(base, []byte(`"schema_version": "1"`), []byte(`"schema_version": "2"`), 1), 400, "unsupported_version"},
		{"unknown field", http.MethodPost, "application/json", unknown, 400, "invalid_schema"},
		{"graph cycle", http.MethodPost, "application/json", cycle, 422, "invalid_workflow"},
		{"oversize", http.MethodPost, "application/json", bytes.Repeat([]byte(" "), v1.MaxRequestBytes+1), 413, "payload_too_large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, "/api/v1/workflows/validate", bytes.NewReader(tc.data))
			request.Header.Set("Content-Type", tc.contentType)
			response := httptest.NewRecorder()
			NewHandler().ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d: %s", response.Code, tc.status, response.Body.String())
			}
			problem, err := v1.DecodeError(response.Body.Bytes())
			if err != nil || problem.Error.Code != tc.code || problem.Error.Retry != v1.Never {
				t.Fatalf("error contract: %s (%v)", response.Body.String(), err)
			}
			if strings.Contains(response.Body.String(), "synthetic-secret-canary") {
				t.Fatal("request input leaked")
			}
			if tc.status == 405 && response.Header().Get("Allow") != "POST" {
				t.Fatal("Allow header missing")
			}
			if tc.status == 422 && len(problem.Issues) == 0 {
				t.Fatal("graph diagnostics missing")
			}
		})
	}
}
