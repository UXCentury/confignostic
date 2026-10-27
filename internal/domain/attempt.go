package domain

import (
	"errors"
	"math"
)

type State string

const (
	Queued           State = "queued"
	AwaitingApproval State = "awaiting_approval"
	Dispatched       State = "dispatched"
	Running          State = "running"
	AwaitingInput    State = "awaiting_input"
	CancelRequested  State = "cancel_requested"
	Unknown          State = "unknown"
	Completed        State = "completed"
	Failed           State = "failed"
	Canceled         State = "canceled"
)

func (s State) Valid() bool {
	switch s {
	case Queued, AwaitingApproval, Dispatched, Running, AwaitingInput, CancelRequested, Unknown, Completed, Failed, Canceled:
		return true
	}
	return false
}
func (s State) Terminal() bool { return s == Completed || s == Failed || s == Canceled }

// CanTransition checks the state pair only; Apply also checks execution evidence.
func (s State) CanTransition(to State) bool {
	if !s.Valid() || !to.Valid() || s == to || s.Terminal() {
		return false
	}
	switch s {
	case Queued:
		return to == AwaitingApproval || to == Dispatched || to == Failed || to == Canceled
	case AwaitingApproval:
		return to == Queued || to == Running || to == CancelRequested || to == Unknown || to.Terminal()
	case Dispatched, Running:
		return to == Running || to == AwaitingInput || to == AwaitingApproval || to == CancelRequested || to == Unknown || to.Terminal()
	case AwaitingInput:
		return to == Running || to == AwaitingApproval || to == CancelRequested || to == Unknown || to.Terminal()
	case CancelRequested:
		return to == Running || to == Unknown || to.Terminal()
	case Unknown:
		return to == Running || to == AwaitingInput || to == AwaitingApproval || to.Terminal()
	}
	return false
}

type Evidence string

const (
	LocalDecision        Evidence = "local_decision"
	DispatchRecorded     Evidence = "dispatch_recorded"
	ProviderObserved     Evidence = "provider_observed"
	ProviderStopped      Evidence = "provider_stopped"
	OutcomeUncertain     Evidence = "outcome_uncertain"
	Reconciled           Evidence = "reconciled"
	CancellationRejected Evidence = "cancellation_rejected"
)

var (
	ErrInvalidAttempt    = errors.New("invalid attempt")
	ErrInvalidTransition = errors.New("invalid attempt transition")
	ErrRetryDenied       = errors.New("retry requires authorization and a new attempt identity")
)

type Attempt struct {
	ID          AttemptID   `json:"id"`
	WorkspaceID WorkspaceID `json:"workspace_id"`
	TaskID      TaskID      `json:"task_id"`
	Number      uint32      `json:"number"`
	State       State       `json:"state"`
	DispatchID  DispatchID  `json:"dispatch_id,omitempty"`
}

func (a Attempt) Validate() error {
	if a.ID.Validate() != nil || a.WorkspaceID.Validate() != nil || a.TaskID.Validate() != nil || a.Number == 0 || !a.State.Valid() {
		return ErrInvalidAttempt
	}
	if a.DispatchID != "" && a.DispatchID.Validate() != nil {
		return ErrInvalidAttempt
	}
	if a.State == Queued && a.DispatchID != "" {
		return ErrInvalidAttempt
	}
	if a.DispatchID == "" {
		switch a.State {
		case Queued, AwaitingApproval, Failed, Canceled:
		default:
			return ErrInvalidAttempt
		}
	}
	return nil
}

type Transition struct {
	To       State
	Evidence Evidence
	// Set only for queued -> dispatched, when durable dispatch intent is recorded.
	DispatchID DispatchID
}

// Apply returns a new value; it never reopens a terminal attempt. Evidence must
// come from trusted policy/connector/reconciliation code, not an API caller.
// Persistence must atomically compare the prior attempt revision before saving.
func (a Attempt) Apply(change Transition) (Attempt, error) {
	if a.Validate() != nil || !a.State.CanTransition(change.To) {
		return a, ErrInvalidTransition
	}
	external := a.DispatchID != ""
	allowed := false
	switch {
	case a.State == Unknown:
		allowed = change.Evidence == Reconciled && (change.To == Running || change.To == AwaitingInput || change.To == AwaitingApproval || change.To.Terminal())
	case change.To == Dispatched:
		allowed = a.State == Queued && change.Evidence == DispatchRecorded && change.DispatchID.Validate() == nil
	case change.To == Unknown:
		allowed = external && change.Evidence == OutcomeUncertain
	case change.To.Terminal():
		if external {
			allowed = change.Evidence == ProviderStopped
		} else {
			allowed = change.Evidence == LocalDecision && (change.To == Failed || change.To == Canceled)
		}
	case change.To == CancelRequested:
		allowed = external && change.Evidence == LocalDecision
	case change.To == AwaitingApproval:
		allowed = (!external && a.State == Queued && change.Evidence == LocalDecision) ||
			(external && (a.State == Dispatched || a.State == Running || a.State == AwaitingInput) && change.Evidence == ProviderObserved)
	case change.To == Queued:
		allowed = a.State == AwaitingApproval && !external && change.Evidence == LocalDecision
	case change.To == Running:
		allowed = external && ((a.State == Dispatched || a.State == AwaitingApproval || a.State == AwaitingInput) && change.Evidence == ProviderObserved ||
			a.State == CancelRequested && change.Evidence == CancellationRejected)
	case change.To == AwaitingInput:
		allowed = (a.State == Dispatched || a.State == Running) && change.Evidence == ProviderObserved
	}
	if change.To != Dispatched && change.DispatchID != "" {
		allowed = false
	}
	if !allowed {
		return a, ErrInvalidTransition
	}
	next := a
	next.State = change.To
	if change.To == Dispatched {
		next.DispatchID = change.DispatchID
	}
	if next.Validate() != nil {
		return a, ErrInvalidTransition
	}
	return next, nil
}

func Retry(previous Attempt, nextID AttemptID, authorized bool) (Attempt, error) {
	if previous.Validate() != nil || !authorized || (previous.State != Failed && previous.State != Canceled) ||
		nextID.Validate() != nil || nextID == previous.ID || previous.Number == math.MaxUint32 {
		return Attempt{}, ErrRetryDenied
	}
	return Attempt{ID: nextID, WorkspaceID: previous.WorkspaceID, TaskID: previous.TaskID, Number: previous.Number + 1, State: Queued}, nil
}
