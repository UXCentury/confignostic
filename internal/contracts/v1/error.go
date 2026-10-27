package v1

import (
	"github.com/UXCentury/confignostic/internal/domain"
	"github.com/UXCentury/confignostic/internal/workflows"
)

type RetryDisposition string

const (
	Never          RetryDisposition = "never"
	Backoff        RetryDisposition = "backoff"
	ReconcileFirst RetryDisposition = "reconcile"
)

type Problem struct {
	Code    string           `json:"code"`
	Message string           `json:"message"`
	Retry   RetryDisposition `json:"retry"`
}
type ErrorResponse struct {
	SchemaVersion string            `json:"schema_version"`
	RequestID     domain.RequestID  `json:"request_id"`
	Error         Problem           `json:"error"`
	Issues        []workflows.Issue `json:"issues,omitempty"`
}

func RetryFor(code string) RetryDisposition {
	switch code {
	case "unavailable", "rate_limited":
		return Backoff
	case "uncertain_outcome":
		return ReconcileFirst
	default:
		return Never
	}
}
func NewProblem(code, message string) Problem { return Problem{code, message, RetryFor(code)} }
func DecodeError(data []byte) (ErrorResponse, error) {
	var result ErrorResponse
	if err := decode(data, MaxEventBytes, "error.schema.json", &result); err != nil {
		return result, err
	}
	if result.Error.Retry != RetryFor(result.Error.Code) {
		return result, ErrInvalidSchema
	}
	return result, nil
}
