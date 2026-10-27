package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"

	v1 "github.com/UXCentury/confignostic/internal/contracts/v1"
	"github.com/UXCentury/confignostic/internal/domain"
	"github.com/UXCentury/confignostic/internal/workflows"
)

func validateWorkflow(w http.ResponseWriter, r *http.Request) {
	requestID, err := domain.NewID[domain.RequestKind]()
	if err != nil {
		http.Error(w, "request identity unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Request-ID", requestID.String())
	respondError := func(status int, code, message string, issues []workflows.Issue) {
		writeJSON(w, status, v1.ErrorResponse{SchemaVersion: v1.Version, RequestID: requestID, Error: v1.NewProblem(code, message), Issues: issues})
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		respondError(http.StatusMethodNotAllowed, "unsupported_operation", "Use POST for workflow validation.", nil)
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		respondError(http.StatusUnsupportedMediaType, "invalid_schema", "Content-Type must be application/json.", nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, v1.MaxRequestBytes)
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondError(http.StatusRequestEntityTooLarge, "payload_too_large", "Workflow request exceeds 256 KiB.", nil)
		} else {
			respondError(http.StatusBadRequest, "invalid_json", "Unable to read the JSON request.", nil)
		}
		return
	}
	_, err = v1.DecodeWorkflow(data)
	if err != nil {
		var invalid *workflows.ValidationError
		switch {
		case errors.As(err, &invalid):
			respondError(http.StatusUnprocessableEntity, "invalid_workflow", "Workflow graph validation failed.", invalid.Issues)
		case errors.Is(err, v1.ErrUnsupportedVersion):
			respondError(http.StatusBadRequest, "unsupported_version", "Only schema version 1 is supported.", nil)
		case errors.Is(err, v1.ErrInvalidJSON):
			respondError(http.StatusBadRequest, "invalid_json", "Expected one strict JSON object without duplicate keys or nulls.", nil)
		case errors.Is(err, v1.ErrTooLarge):
			respondError(http.StatusRequestEntityTooLarge, "payload_too_large", "Workflow exceeds 256 KiB when serialized.", nil)
		case errors.Is(err, v1.ErrSchemaUnavailable):
			respondError(http.StatusInternalServerError, "internal_error", "Workflow validation is unavailable.", nil)
		default:
			respondError(http.StatusBadRequest, "invalid_schema", "Request does not match workflow schema v1.", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, struct {
		SchemaVersion       string           `json:"schema_version"`
		RequestID           domain.RequestID `json:"request_id"`
		Valid               bool             `json:"valid"`
		CapabilitiesChecked bool             `json:"capabilities_checked"`
		Executable          bool             `json:"executable"`
	}{v1.Version, requestID, true, false, false})
}
