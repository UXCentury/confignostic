package v1

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/UXCentury/confignostic/internal/domain"
)

var (
	ErrProtocolMismatch       = errors.New("no supported runner protocol")
	ErrAcknowledgmentMismatch = errors.New("acknowledgment does not match dispatch")
)

type Hello struct {
	Type             string             `json:"type"`
	SchemaVersion    string             `json:"schema_version"`
	WorkspaceID      domain.WorkspaceID `json:"workspace_id"`
	RunnerID         domain.RunnerID    `json:"runner_id"`
	RuntimeVersion   string             `json:"runtime_version"`
	ProtocolVersions []string           `json:"protocol_versions"`
	MaxMessageBytes  uint32             `json:"max_message_bytes"`
}
type Welcome struct {
	Type            string `json:"type"`
	SchemaVersion   string `json:"schema_version"`
	ProtocolVersion string `json:"protocol_version"`
	MaxMessageBytes uint32 `json:"max_message_bytes"`
}

func DecodeHello(data []byte) (Hello, error) {
	var hello Hello
	err := decode(data, MaxRunnerBytes, "hello.schema.json", &hello)
	return hello, err
}

func DecodeWelcome(data []byte) (Welcome, error) {
	var welcome Welcome
	err := decode(data, MaxRunnerBytes, "welcome.schema.json", &welcome)
	return welcome, err
}

// Negotiate requires the identities from an already authenticated enrollment.
// A matching JSON identity is never a substitute for authentication.
func Negotiate(hello Hello, workspace domain.WorkspaceID, runner domain.RunnerID) (Welcome, error) {
	data, err := json.Marshal(hello)
	if err != nil {
		return Welcome{}, err
	}
	if _, err = DecodeHello(data); err != nil {
		return Welcome{}, err
	}
	if workspace.Validate() != nil || runner.Validate() != nil || workspace != hello.WorkspaceID || runner != hello.RunnerID {
		return Welcome{}, ErrScopeMismatch
	}
	if !slices.Contains(hello.ProtocolVersions, Version) {
		return Welcome{}, ErrProtocolMismatch
	}
	return Welcome{Type: "runner.welcome", SchemaVersion: Version, ProtocolVersion: Version, MaxMessageBytes: min(hello.MaxMessageBytes, MaxRunnerBytes)}, nil
}

type Input struct {
	Message      string              `json:"message"`
	ArtifactRefs []domain.ArtifactID `json:"artifact_refs,omitempty"`
}
type Dispatch struct {
	Type              string                   `json:"type"`
	SchemaVersion     string                   `json:"schema_version"`
	WorkspaceID       domain.WorkspaceID       `json:"workspace_id"`
	TaskID            domain.TaskID            `json:"task_id"`
	AttemptID         domain.AttemptID         `json:"attempt_id"`
	DispatchID        domain.DispatchID        `json:"dispatch_id"`
	ConnectionID      domain.ConnectionID      `json:"connection_id"`
	LeaseEpoch        domain.Counter           `json:"lease_epoch"`
	CorrelationID     domain.RequestID         `json:"correlation_id"`
	Actor             domain.Actor             `json:"actor"`
	IdempotencyKey    string                   `json:"idempotency_key"`
	PolicyVersion     string                   `json:"policy_version"`
	ApprovalRefs      []domain.ApprovalID      `json:"approval_refs,omitempty"`
	Input             Input                    `json:"input"`
	WorkflowRunID     domain.WorkflowRunID     `json:"workflow_run_id,omitempty"`
	WorkflowVersionID domain.WorkflowVersionID `json:"workflow_version_id,omitempty"`
	WorkflowNodeID    string                   `json:"workflow_node_id,omitempty"`
}
type Acknowledgment struct {
	Type          string             `json:"type"`
	SchemaVersion string             `json:"schema_version"`
	WorkspaceID   domain.WorkspaceID `json:"workspace_id"`
	RunnerID      domain.RunnerID    `json:"runner_id"`
	DispatchID    domain.DispatchID  `json:"dispatch_id"`
	AttemptID     domain.AttemptID   `json:"attempt_id"`
	LeaseEpoch    domain.Counter     `json:"lease_epoch"`
	Status        string             `json:"status"`
	Error         *Problem           `json:"error,omitempty"`
}

func DecodeDispatch(data []byte, workspace domain.WorkspaceID) (Dispatch, error) {
	var result Dispatch
	if err := decode(data, MaxRunnerBytes, "dispatch.schema.json", &result); err != nil {
		return result, err
	}
	if workspace.Validate() != nil || result.WorkspaceID != workspace {
		return result, ErrScopeMismatch
	}
	if len(result.Input.Message) > MaxMessageBytes || strings.TrimSpace(result.Input.Message) == "" {
		return result, ErrInvalidSchema
	}
	return result, nil
}
func DecodeAcknowledgment(data []byte, workspace domain.WorkspaceID) (Acknowledgment, error) {
	var result Acknowledgment
	if err := decode(data, MaxRunnerBytes, "ack.schema.json", &result); err != nil {
		return result, err
	}
	if workspace.Validate() != nil || result.WorkspaceID != workspace {
		return result, ErrScopeMismatch
	}
	if result.Error != nil && result.Error.Retry != RetryFor(result.Error.Code) {
		return result, ErrInvalidSchema
	}
	// A rejection is an admission decision. An uncertain provider outcome
	// belongs to an accepted attempt's event history, never a rejection.
	if result.Error != nil && result.Error.Code == "uncertain_outcome" {
		return result, ErrInvalidSchema
	}
	return result, nil
}
func MatchAcknowledgment(dispatch Dispatch, ack Acknowledgment, runner domain.RunnerID) error {
	data, err := json.Marshal(dispatch)
	if err != nil {
		return err
	}
	if _, err = DecodeDispatch(data, dispatch.WorkspaceID); err != nil {
		return err
	}
	data, err = json.Marshal(ack)
	if err != nil {
		return err
	}
	if _, err = DecodeAcknowledgment(data, dispatch.WorkspaceID); err != nil {
		return err
	}
	if runner.Validate() != nil || ack.RunnerID != runner || ack.DispatchID != dispatch.DispatchID || ack.AttemptID != dispatch.AttemptID || ack.LeaseEpoch != dispatch.LeaseEpoch {
		return ErrAcknowledgmentMismatch
	}
	return nil
}
