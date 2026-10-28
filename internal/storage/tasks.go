package storage

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	v1 "github.com/UXCentury/confignostic/internal/contracts/v1"
	"github.com/UXCentury/confignostic/internal/domain"
	"github.com/jackc/pgx/v5"
)

type Audit struct {
	Actor       domain.Actor
	Correlation domain.RequestID
}

func (a Audit) valid() bool { return a.Actor.Validate() == nil && a.Correlation.Validate() == nil }

type Submission struct {
	Workspace    domain.WorkspaceID
	Conversation domain.ConversationID
	Connection   domain.ConnectionID
	Key          string
	Input        v1.Input
	Audit        Audit
}
type Accepted struct {
	Task     domain.TaskID     `json:"task_id"`
	Attempt  domain.AttemptID  `json:"attempt_id"`
	Message  domain.MessageID  `json:"message_id"`
	Dispatch domain.DispatchID `json:"dispatch_id"`
}

func (s *Store) Submit(ctx context.Context, in Submission) (Accepted, error) {
	var accepted Accepted
	err := s.transaction(ctx, func(tx pgx.Tx) error { var err error; accepted, err = s.submit(ctx, tx, in); return err })
	return accepted, err // Callers acknowledge only when err == nil.
}
func (s *Store) submit(ctx context.Context, tx pgx.Tx, in Submission) (Accepted, error) {
	var result Accepted
	if in.Workspace.Validate() != nil || in.Conversation.Validate() != nil || in.Connection.Validate() != nil || !bounded(in.Key, 128) || !bounded(in.Input.Message, v1.MaxMessageBytes) || !in.Audit.valid() || len(in.Input.ArtifactRefs) > 16 {
		return result, ErrInvalid
	}
	seen := map[domain.ArtifactID]bool{}
	for _, id := range in.Input.ArtifactRefs {
		if id.Validate() != nil || seen[id] {
			return result, ErrInvalid
		}
		seen[id] = true
	}
	if err := lockWorkspace(ctx, tx, in.Workspace); err != nil {
		return result, err
	}
	payload := in
	payload.Audit.Correlation = "req_00000000000000000000000000000000"
	hash := digest(payload) // A new request correlation does not change the operation.
	var prior string
	err := tx.QueryRow(ctx, `SELECT k.digest,t.id,a.id,t.message_id,d.id FROM confignostic.request_keys k JOIN confignostic.tasks t ON (t.workspace_id,t.id)=(k.workspace_id,k.task_id) JOIN confignostic.attempts a ON (a.workspace_id,a.task_id)=(t.workspace_id,t.id) AND a.number=1 JOIN confignostic.dispatches d ON (d.workspace_id,d.attempt_id)=(a.workspace_id,a.id) WHERE k.workspace_id=$1 AND k.key=$2`, in.Workspace.String(), in.Key).Scan(&prior, &result.Task, &result.Attempt, &result.Message, &result.Dispatch)
	if err == nil {
		if prior != hash {
			return Accepted{}, ErrConflict
		}
		return result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return result, dbError(err)
	}
	result.Task, err = domain.NewID[domain.TaskKind]()
	if err != nil {
		return result, ErrUnavailable
	}
	result.Attempt, err = domain.NewID[domain.AttemptKind]()
	if err != nil {
		return result, ErrUnavailable
	}
	result.Message, err = domain.NewID[domain.MessageKind]()
	if err != nil {
		return result, ErrUnavailable
	}
	result.Dispatch, err = domain.NewID[domain.DispatchKind]()
	if err != nil {
		return result, ErrUnavailable
	}
	ws := in.Workspace.String()
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO confignostic.messages VALUES($1,$2,$3,$4,$5)`, []any{ws, result.Message.String(), in.Conversation.String(), in.Audit.Actor.ID.String(), in.Input.Message}},
		{`INSERT INTO confignostic.tasks(workspace_id,id,conversation_id,connection_id,message_id) VALUES($1,$2,$3,$4,$5)`, []any{ws, result.Task.String(), in.Conversation.String(), in.Connection.String(), result.Message.String()}},
		{`INSERT INTO confignostic.attempts(workspace_id,id,task_id,number,state) VALUES($1,$2,$3,1,'queued')`, []any{ws, result.Attempt.String(), result.Task.String()}},
		{`INSERT INTO confignostic.dispatches(workspace_id,id,attempt_id) VALUES($1,$2,$3)`, []any{ws, result.Dispatch.String(), result.Attempt.String()}},
		{`INSERT INTO confignostic.request_keys VALUES($1,$2,$3,$4)`, []any{ws, in.Key, hash, result.Task.String()}},
	}
	for _, stmt := range statements {
		if _, err = tx.Exec(ctx, stmt.sql, stmt.args...); err != nil {
			return Accepted{}, dbError(err)
		}
	}
	for _, id := range in.Input.ArtifactRefs {
		if _, err = tx.Exec(ctx, `INSERT INTO confignostic.artifact_refs VALUES($1,$2,$3)`, ws, result.Attempt.String(), id.String()); err != nil {
			return Accepted{}, dbError(err)
		}
	}
	return result, nil
}

func readAttempt(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, ws domain.WorkspaceID, id domain.AttemptID) (domain.Attempt, error) {
	a := domain.Attempt{WorkspaceID: ws, ID: id}
	err := q.QueryRow(ctx, `SELECT task_id,number,state,COALESCE(dispatch_id,'') FROM confignostic.attempts WHERE workspace_id=$1 AND id=$2`, ws.String(), id.String()).Scan(&a.TaskID, &a.Number, &a.State, &a.DispatchID)
	if err != nil {
		return a, dbError(err)
	}
	if a.Validate() != nil {
		return a, ErrInvalid
	}
	return a, nil
}
func (s *Store) Attempt(ctx context.Context, ws domain.WorkspaceID, id domain.AttemptID) (domain.Attempt, error) {
	if ws.Validate() != nil || id.Validate() != nil {
		return domain.Attempt{}, ErrInvalid
	}
	return readAttempt(ctx, s.pool, ws, id)
}

type Lease struct {
	Workspace domain.WorkspaceID
	Dispatch  domain.DispatchID
	Attempt   domain.AttemptID
	Owner     string
	Epoch     domain.Counter
	Until     time.Time
	// Mode is send only if no durable send intent has ever been recorded.
	Mode string
}

func leaseDuration(ttl time.Duration) bool { return ttl >= time.Second && ttl <= 5*time.Minute }
func counterArg(c domain.Counter) string   { return strconv.FormatUint(uint64(c), 10) }
func parseCounter(s string) (domain.Counter, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	return domain.Counter(n), err
}

// Claim serializes within a workspace. Expired sent work is reconciled, never
// returned as a fresh send. Stage G must also enforce fencing at the runner.
func (s *Store) Claim(ctx context.Context, ws domain.WorkspaceID, owner string, ttl time.Duration, audit Audit) (Lease, error) {
	result := Lease{Workspace: ws, Owner: owner}
	if !bounded(owner, 128) || !leaseDuration(ttl) || !audit.valid() {
		return result, ErrInvalid
	}
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		if err := lockWorkspace(ctx, tx, ws); err != nil {
			return err
		}
		var epoch string
		var sent bool
		err := tx.QueryRow(ctx, `SELECT d.id,d.attempt_id,d.epoch::text,d.sent_at IS NOT NULL FROM confignostic.dispatches d JOIN confignostic.attempts a ON (a.workspace_id,a.id)=(d.workspace_id,d.attempt_id) WHERE d.workspace_id=$1 AND a.state NOT IN ('completed','failed','canceled') AND (a.state<>'awaiting_approval' OR d.sent_at IS NOT NULL) AND (d.lease_until IS NULL OR d.lease_until<=clock_timestamp()) ORDER BY d.id LIMIT 1 FOR UPDATE OF d`, ws.String()).Scan(&result.Dispatch, &result.Attempt, &epoch, &sent)
		if err != nil {
			return dbError(err)
		}
		old, err := parseCounter(epoch)
		if err != nil || old == ^domain.Counter(0) {
			return ErrLimit
		}
		result.Epoch = old + 1
		if err = tx.QueryRow(ctx, `UPDATE confignostic.dispatches SET owner=$3,epoch=$4,lease_until=clock_timestamp()+($5 * interval '1 millisecond') WHERE workspace_id=$1 AND id=$2 RETURNING lease_until`, ws.String(), result.Dispatch.String(), owner, counterArg(result.Epoch), ttl.Milliseconds()).Scan(&result.Until); err != nil {
			return dbError(err)
		}
		result.Mode = "send"
		if sent {
			result.Mode = "reconcile"
			a, err := readAttempt(ctx, tx, ws, result.Attempt)
			if err != nil {
				return err
			}
			if a.State != domain.Unknown {
				_, err = s.transition(ctx, tx, a, domain.Transition{To: domain.Unknown, Evidence: domain.OutcomeUncertain}, audit, nil)
				return err
			}
		}
		return nil
	})
	return result, err
}
func checkLease(ctx context.Context, tx pgx.Tx, l Lease) error {
	if l.Workspace.Validate() != nil || l.Dispatch.Validate() != nil || l.Attempt.Validate() != nil || !bounded(l.Owner, 128) || l.Epoch == 0 {
		return ErrInvalid
	}
	var ok bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM confignostic.dispatches WHERE workspace_id=$1 AND id=$2 AND attempt_id=$3 AND owner=$4 AND epoch=$5 AND lease_until>clock_timestamp())`, l.Workspace.String(), l.Dispatch.String(), l.Attempt.String(), l.Owner, counterArg(l.Epoch)).Scan(&ok)
	if err != nil {
		return dbError(err)
	}
	if !ok {
		return ErrLeaseLost
	}
	return nil
}
func (s *Store) fenced(ctx context.Context, l Lease, fn func(pgx.Tx) error) error {
	return s.transaction(ctx, func(tx pgx.Tx) error {
		if err := lockWorkspace(ctx, tx, l.Workspace); err != nil {
			return err
		}
		if err := checkLease(ctx, tx, l); err != nil {
			return err
		}
		if err := fn(tx); err != nil {
			return err
		}
		return checkLease(ctx, tx, l)
	})
}
func (s *Store) Renew(ctx context.Context, l Lease, ttl time.Duration) (Lease, error) {
	if !leaseDuration(ttl) {
		return l, ErrInvalid
	}
	err := s.fenced(ctx, l, func(tx pgx.Tx) error {
		return dbError(tx.QueryRow(ctx, `UPDATE confignostic.dispatches SET lease_until=clock_timestamp()+($3 * interval '1 millisecond') WHERE workspace_id=$1 AND id=$2 RETURNING lease_until`, l.Workspace.String(), l.Dispatch.String(), ttl.Milliseconds()).Scan(&l.Until))
	})
	return l, err
}
func (s *Store) BeginDispatch(ctx context.Context, l Lease, audit Audit) error {
	if !audit.valid() {
		return ErrInvalid
	}
	return s.fenced(ctx, l, func(tx pgx.Tx) error {
		a, err := readAttempt(ctx, tx, l.Workspace, l.Attempt)
		if err != nil {
			return err
		}
		if _, err = s.transition(ctx, tx, a, domain.Transition{To: domain.Dispatched, Evidence: domain.DispatchRecorded, DispatchID: l.Dispatch}, audit, nil); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE confignostic.dispatches SET sent_at=clock_timestamp() WHERE workspace_id=$1 AND id=$2 AND sent_at IS NULL`, l.Workspace.String(), l.Dispatch.String())
		return dbError(err)
	})
}

type Delivery struct {
	Source     string // Stable provider session/operation or runner-stream identity.
	ExternalID string
	Change     domain.Transition
	Audit      Audit
	Provider   *v1.ProviderReference
}

// Ingest acknowledges only committed receipts. Source/event identity reuse with
// different content fails even if the original event has since been compacted.
func (s *Store) Ingest(ctx context.Context, l Lease, in Delivery) (domain.Counter, error) {
	var sequence domain.Counter
	if !bounded(in.Source, 256) || !bounded(in.ExternalID, 512) || !in.Audit.valid() || in.Change.To == domain.Dispatched {
		return 0, ErrInvalid
	}
	err := s.fenced(ctx, l, func(tx pgx.Tx) error {
		var connection string
		if err := tx.QueryRow(ctx, `SELECT t.connection_id FROM confignostic.tasks t JOIN confignostic.attempts a ON (a.workspace_id,a.task_id)=(t.workspace_id,t.id) WHERE a.workspace_id=$1 AND a.id=$2`, l.Workspace.String(), l.Attempt.String()).Scan(&connection); err != nil {
			return dbError(err)
		}
		hash := digest(struct {
			Dispatch                                         domain.DispatchID
			Source, ExternalID, To, Evidence, ChangeDispatch string
			Audit                                            Audit
			Provider                                         *v1.ProviderReference
		}{l.Dispatch, in.Source, in.ExternalID, string(in.Change.To), string(in.Change.Evidence), in.Change.DispatchID.String(), in.Audit, in.Provider})
		var prior, seq string
		err := tx.QueryRow(ctx, `SELECT digest,sequence::text FROM confignostic.inbox WHERE workspace_id=$1 AND connection_id=$2 AND source=$3 AND external_id=$4`, l.Workspace.String(), connection, in.Source, in.ExternalID).Scan(&prior, &seq)
		if err == nil {
			if prior != hash {
				return ErrConflict
			}
			sequence, err = parseCounter(seq)
			return err
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return dbError(err)
		}
		a, err := readAttempt(ctx, tx, l.Workspace, l.Attempt)
		if err != nil {
			return err
		}
		sequence, err = s.transition(ctx, tx, a, in.Change, in.Audit, in.Provider)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO confignostic.inbox VALUES($1,$2,$3,$4,$5,$6,$7)`, l.Workspace.String(), connection, in.Source, in.ExternalID, hash, l.Dispatch.String(), counterArg(sequence))
		return dbError(err)
	})
	return sequence, err
}
func (s *Store) transition(ctx context.Context, tx pgx.Tx, a domain.Attempt, change domain.Transition, audit Audit, provider *v1.ProviderReference) (domain.Counter, error) {
	next, err := a.Apply(change)
	if err != nil {
		return 0, err
	}
	var sequence string
	if err = tx.QueryRow(ctx, `UPDATE confignostic.workspaces SET event_sequence=event_sequence+1 WHERE id=$1 RETURNING event_sequence::text`, a.WorkspaceID.String()).Scan(&sequence); err != nil {
		return 0, dbError(err)
	}
	seq, err := parseCounter(sequence)
	if err != nil {
		return 0, ErrLimit
	}
	id, err := domain.NewID[domain.EventKind]()
	if err != nil {
		return 0, ErrUnavailable
	}
	payload, _ := json.Marshal(v1.StateChange{From: a.State, To: next.State})
	event := v1.Event{SchemaVersion: v1.Version, ID: id, WorkspaceID: a.WorkspaceID, TaskID: a.TaskID, AttemptID: a.ID, Sequence: seq, OccurredAt: time.Now().UTC(), Actor: audit.Actor, CorrelationID: audit.Correlation, Provider: provider, Type: "task.state_changed", Payload: payload}
	data, err := v1.EncodeEvent(event)
	if err != nil {
		return 0, ErrInvalid
	}
	if _, err = tx.Exec(ctx, `UPDATE confignostic.attempts SET state=$3,dispatch_id=NULLIF($4,'') WHERE workspace_id=$1 AND id=$2`, a.WorkspaceID.String(), a.ID.String(), string(next.State), next.DispatchID.String()); err != nil {
		return 0, dbError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO confignostic.events(workspace_id,sequence,id,attempt_id,envelope) VALUES($1,$2,$3,$4,$5)`, a.WorkspaceID.String(), sequence, id.String(), a.ID.String(), data); err != nil {
		return 0, dbError(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO confignostic.event_outbox(workspace_id,sequence) VALUES($1,$2)`, a.WorkspaceID.String(), sequence)
	return seq, dbError(err)
}
