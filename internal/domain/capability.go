package domain

import "errors"

type Capability string

const (
	Invoke           Capability = "invoke"
	StreamEvents     Capability = "stream_events"
	Cancel           Capability = "cancel"
	Resume           Capability = "resume"
	Reconcile        Capability = "reconcile"
	Artifacts        Capability = "artifacts"
	StructuredOutput Capability = "structured_output"
	Usage            Capability = "usage"
	Approvals        Capability = "approvals"
	UserInput        Capability = "user_input"
	ListSessions     Capability = "list_sessions"
	CreateSession    Capability = "create_session"
	LoadSession      Capability = "load_session"
	IdempotentInvoke Capability = "idempotent_invoke"
)

func (c Capability) Valid() bool {
	switch c {
	case Invoke, StreamEvents, Cancel, Resume, Reconcile, Artifacts, StructuredOutput, Usage, Approvals, UserInput, ListSessions, CreateSession, LoadSession, IdempotentInvoke:
		return true
	}
	return false
}

func ValidateCapabilities(caps []Capability) error {
	if len(caps) > 16 {
		return errors.New("too many capabilities")
	}
	seen := map[Capability]bool{}
	for _, c := range caps {
		if !c.Valid() || seen[c] {
			return errors.New("invalid or duplicate capability")
		}
		seen[c] = true
	}
	return nil
}

type ActorKind string

const (
	Human      ActorKind = "human"
	Planner    ActorKind = "planner"
	AgentActor ActorKind = "agent"
	System     ActorKind = "system"
)

// Actor describes provenance. It never authenticates the sender or grants rights.
type Actor struct {
	Kind ActorKind `json:"kind"`
	ID   ActorID   `json:"id"`
}

func (a Actor) Validate() error {
	if err := a.ID.Validate(); err != nil {
		return err
	}
	switch a.Kind {
	case Human, Planner, AgentActor, System:
		return nil
	}
	return errors.New("invalid actor kind")
}
