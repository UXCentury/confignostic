package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"strings"
	"time"

	"github.com/UXCentury/confignostic/internal/domain"
)

type ProviderReference struct {
	SessionID   string `json:"session_id,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	EventID     string `json:"event_id,omitempty"`
}
type Event struct {
	SchemaVersion string             `json:"schema_version"`
	ID            domain.EventID     `json:"id"`
	WorkspaceID   domain.WorkspaceID `json:"workspace_id"`
	TaskID        domain.TaskID      `json:"task_id"`
	AttemptID     domain.AttemptID   `json:"attempt_id"`
	Sequence      domain.Counter     `json:"sequence"`
	OccurredAt    time.Time          `json:"occurred_at"`
	Actor         domain.Actor       `json:"actor"`
	CorrelationID domain.RequestID   `json:"correlation_id"`
	Provider      *ProviderReference `json:"provider,omitempty"`
	Type          string             `json:"type"`
	Payload       json.RawMessage    `json:"payload"`
}
type StateChange struct {
	From domain.State `json:"from"`
	To   domain.State `json:"to"`
}
type Output struct {
	Text string `json:"text"`
}
type ApprovalRequest struct {
	ApprovalID    domain.ApprovalID `json:"approval_id"`
	ActionDigest  string            `json:"action_digest"`
	PolicyVersion string            `json:"policy_version"`
	Scope         string            `json:"scope"`
	ExpiresAt     time.Time         `json:"expires_at"`
}
type ArtifactCreated struct {
	ArtifactID domain.ArtifactID `json:"artifact_id"`
	SHA256     string            `json:"sha256"`
	MediaType  string            `json:"media_type"`
	SizeBytes  domain.Counter    `json:"size_bytes"`
}

func DecodeEvent(data []byte, workspace domain.WorkspaceID) (Event, error) {
	var event Event
	if err := decode(data, MaxEventBytes, "event.schema.json", &event); err != nil {
		return event, err
	}
	if workspace.Validate() != nil || event.WorkspaceID != workspace {
		return event, ErrScopeMismatch
	}
	if event.OccurredAt.IsZero() {
		return event, ErrInvalidSchema
	}
	switch event.Type {
	case "task.state_changed":
		var change StateChange
		if json.Unmarshal(event.Payload, &change) != nil || !change.From.CanTransition(change.To) {
			return event, ErrInvalidSchema
		}
		// This is a notification, not authority to apply a transition. The task
		// aggregate must already have validated the corresponding evidence.
	case "task.output":
		var output Output
		if json.Unmarshal(event.Payload, &output) != nil || len(output.Text) > MaxMessageBytes {
			return event, ErrInvalidSchema
		}
	case "approval.requested":
		var approval ApprovalRequest
		if json.Unmarshal(event.Payload, &approval) != nil || !approval.ExpiresAt.After(event.OccurredAt) {
			return event, ErrInvalidSchema
		}
	case "artifact.created":
		var artifact ArtifactCreated
		if json.Unmarshal(event.Payload, &artifact) != nil || artifact.SizeBytes > MaxArtifactBytes {
			return event, ErrInvalidSchema
		}
		if mediaType, _, err := mime.ParseMediaType(artifact.MediaType); err != nil || !strings.Contains(mediaType, "/") {
			return event, ErrInvalidSchema
		}
	}
	return event, nil
}
func EncodeEvent(event Event) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	if _, err = DecodeEvent(data, event.WorkspaceID); err != nil {
		return nil, err
	}
	return data, nil
}

// WriteSSE writes a complete validated SSE frame, without introducing a live
// subscription or claiming persistence. JSON escaping prevents frame injection.
func WriteSSE(w io.Writer, event Event) error {
	data, err := EncodeEvent(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.Sequence, event.Type, data)
	return err
}
func ParseCursor(value string) (domain.Counter, error) {
	if value == "" {
		return 0, nil
	}
	if len(value) > 20 || strings.TrimSpace(value) != value {
		return 0, domain.ErrInvalidCounter
	}
	encoded, _ := json.Marshal(value)
	var cursor domain.Counter
	err := json.Unmarshal(encoded, &cursor)
	return cursor, err
}
