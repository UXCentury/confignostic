// Package domain contains provider-independent identities and execution rules.
package domain

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

var ErrInvalidID = errors.New("invalid resource identifier")

type resourceKind interface{ prefix() string }

// ID's type parameter prevents mixing identities of different resources in Go.
// The wire form is a resource prefix plus 128 random bits in lowercase hex.
type ID[K resourceKind] string

func NewID[K resourceKind]() (ID[K], error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	var kind K
	return ID[K](kind.prefix() + "_" + hex.EncodeToString(bytes[:])), nil
}

func ParseID[K resourceKind](value string) (ID[K], error) {
	id := ID[K](value)
	return id, id.Validate()
}

func (id ID[K]) Validate() error {
	var kind K
	prefix := kind.prefix() + "_"
	value := string(id)
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+32 {
		return ErrInvalidID
	}
	for _, c := range value[len(prefix):] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return ErrInvalidID
		}
	}
	return nil
}

func (id ID[K]) String() string { return string(id) }
func (id ID[K]) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(string(id))
}
func (id *ID[K]) UnmarshalJSON(data []byte) error {
	var value string
	if string(data) == "null" || json.Unmarshal(data, &value) != nil {
		return ErrInvalidID
	}
	parsed, err := ParseID[K](value)
	if err == nil {
		*id = parsed
	}
	return err
}

type WorkspaceKind struct{}

func (WorkspaceKind) prefix() string { return "wsp" }

type WorkspaceID = ID[WorkspaceKind]

type ProjectKind struct{}

func (ProjectKind) prefix() string { return "prj" }

type ProjectID = ID[ProjectKind]

type AgentKind struct{}

func (AgentKind) prefix() string { return "agt" }

type AgentID = ID[AgentKind]

type ConnectionKind struct{}

func (ConnectionKind) prefix() string { return "con" }

type ConnectionID = ID[ConnectionKind]

type ConversationKind struct{}

func (ConversationKind) prefix() string { return "cnv" }

type ConversationID = ID[ConversationKind]

type MessageKind struct{}

func (MessageKind) prefix() string { return "msg" }

type MessageID = ID[MessageKind]

type TaskKind struct{}

func (TaskKind) prefix() string { return "tsk" }

type TaskID = ID[TaskKind]

type AttemptKind struct{}

func (AttemptKind) prefix() string { return "atm" }

type AttemptID = ID[AttemptKind]

type DispatchKind struct{}

func (DispatchKind) prefix() string { return "dsp" }

type DispatchID = ID[DispatchKind]

type EventKind struct{}

func (EventKind) prefix() string { return "evt" }

type EventID = ID[EventKind]

type ApprovalKind struct{}

func (ApprovalKind) prefix() string { return "apr" }

type ApprovalID = ID[ApprovalKind]

type ArtifactKind struct{}

func (ArtifactKind) prefix() string { return "art" }

type ArtifactID = ID[ArtifactKind]

type WorkflowKind struct{}

func (WorkflowKind) prefix() string { return "wfl" }

type WorkflowID = ID[WorkflowKind]

type WorkflowVersionKind struct{}

func (WorkflowVersionKind) prefix() string { return "wfv" }

type WorkflowVersionID = ID[WorkflowVersionKind]

type WorkflowRunKind struct{}

func (WorkflowRunKind) prefix() string { return "run" }

type WorkflowRunID = ID[WorkflowRunKind]

type RunnerKind struct{}

func (RunnerKind) prefix() string { return "rnr" }

type RunnerID = ID[RunnerKind]

type ActorIDKind struct{}

func (ActorIDKind) prefix() string { return "act" }

type ActorID = ID[ActorIDKind]

type RequestKind struct{}

func (RequestKind) prefix() string { return "req" }

type RequestID = ID[RequestKind]
