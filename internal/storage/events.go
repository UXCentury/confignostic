package storage

import (
	"context"
	"encoding/json"

	v1 "github.com/UXCentury/confignostic/internal/contracts/v1"
	"github.com/UXCentury/confignostic/internal/domain"
	"github.com/jackc/pgx/v5"
)

type Snapshot struct {
	Cursor   domain.Counter   `json:"cursor"`
	Attempts []domain.Attempt `json:"attempts"`
}
type EventPage struct {
	Events   []v1.Event
	Snapshot *Snapshot // Present with ErrCursorExpired; resume after its Cursor.
}

// Events and PendingEvents page by committed workspace sequence, never offset.
func (s *Store) Events(ctx context.Context, ws domain.WorkspaceID, after domain.Counter, limit int) (EventPage, error) {
	return s.events(ctx, ws, after, limit, false)
}
func (s *Store) PendingEvents(ctx context.Context, ws domain.WorkspaceID, after domain.Counter, limit int) (EventPage, error) {
	return s.events(ctx, ws, after, limit, true)
}
func (s *Store) events(ctx context.Context, ws domain.WorkspaceID, after domain.Counter, limit int, pending bool) (EventPage, error) {
	page := EventPage{Events: []v1.Event{}}
	if limit < 1 || limit > 1000 {
		return page, ErrLimit
	}
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		if err := lockWorkspace(ctx, tx, ws); err != nil {
			return err
		}
		var floor, head string
		if err := tx.QueryRow(ctx, `SELECT retained_after::text,event_sequence::text FROM confignostic.workspaces WHERE id=$1`, ws.String()).Scan(&floor, &head); err != nil {
			return dbError(err)
		}
		low, err := parseCounter(floor)
		if err != nil {
			return ErrInvalid
		}
		high, err := parseCounter(head)
		if err != nil {
			return ErrInvalid
		}
		if after > high {
			return ErrInvalid
		}
		if after < low {
			var data []byte
			if err := tx.QueryRow(ctx, `SELECT snapshot FROM confignostic.snapshots WHERE workspace_id=$1`, ws.String()).Scan(&data); err != nil {
				return dbError(err)
			}
			var snap Snapshot
			if json.Unmarshal(data, &snap) != nil {
				return ErrInvalid
			}
			page.Snapshot = &snap
			return ErrCursorExpired
		}
		rows, err := tx.Query(ctx, `SELECT e.envelope FROM confignostic.events e JOIN confignostic.event_outbox o USING(workspace_id,sequence) WHERE e.workspace_id=$1 AND e.sequence>$2 AND (NOT $4 OR NOT o.delivered) ORDER BY e.sequence LIMIT $3`, ws.String(), counterArg(after), limit, pending)
		if err != nil {
			return dbError(err)
		}
		defer rows.Close()
		for rows.Next() {
			var data []byte
			if rows.Scan(&data) != nil {
				return ErrUnavailable
			}
			event, err := v1.DecodeEvent(compactJSON(data), ws)
			if err != nil {
				return ErrInvalid
			}
			page.Events = append(page.Events, event)
		}
		return dbError(rows.Err())
	})
	return page, err
}

// Published is called only after durable downstream acceptance. Delivery is
// at-least-once; downstream consumers must deduplicate the stable event ID.
func (s *Store) Published(ctx context.Context, ws domain.WorkspaceID, sequence domain.Counter) error {
	return s.transaction(ctx, func(tx pgx.Tx) error {
		if err := lockWorkspace(ctx, tx, ws); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE confignostic.event_outbox SET delivered=true WHERE workspace_id=$1 AND sequence=$2`, ws.String(), counterArg(sequence))
		if err != nil {
			return dbError(err)
		}
		if tag.RowsAffected() != 1 {
			return ErrNotFound
		}
		return nil
	})
}

// Compact deletes at most 1,000 published events per call and keeps at least
// keep recent events. A consistent current-state snapshot is committed with the
// new floor. Large workspaces fail closed rather than emit a partial snapshot.
func (s *Store) Compact(ctx context.Context, ws domain.WorkspaceID, keep uint32) error {
	if keep < 1 || keep > 10000 {
		return ErrLimit
	}
	return s.transaction(ctx, func(tx pgx.Tx) error {
		if err := lockWorkspace(ctx, tx, ws); err != nil {
			return err
		}
		var head, floor string
		if err := tx.QueryRow(ctx, `SELECT event_sequence::text,retained_after::text FROM confignostic.workspaces WHERE id=$1`, ws.String()).Scan(&head, &floor); err != nil {
			return dbError(err)
		}
		cursor, err := parseCounter(head)
		if err != nil {
			return ErrInvalid
		}
		old, err := parseCounter(floor)
		if err != nil {
			return ErrInvalid
		}
		if cursor <= domain.Counter(keep) || cursor-domain.Counter(keep) <= old {
			return nil
		}
		through := cursor - domain.Counter(keep)
		if through-old > 1000 {
			through = old + 1000
		}
		var pending bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM confignostic.event_outbox WHERE workspace_id=$1 AND sequence<=$2 AND NOT delivered)`, ws.String(), counterArg(through)).Scan(&pending); err != nil {
			return dbError(err)
		}
		if pending {
			return ErrConflict
		}
		rows, err := tx.Query(ctx, `SELECT DISTINCT ON(task_id) id,task_id,number,state,COALESCE(dispatch_id,'') FROM confignostic.attempts WHERE workspace_id=$1 ORDER BY task_id,number DESC LIMIT 1001`, ws.String())
		if err != nil {
			return dbError(err)
		}
		snap := Snapshot{Cursor: cursor, Attempts: []domain.Attempt{}}
		for rows.Next() {
			a := domain.Attempt{WorkspaceID: ws}
			if rows.Scan(&a.ID, &a.TaskID, &a.Number, &a.State, &a.DispatchID) != nil {
				rows.Close()
				return ErrUnavailable
			}
			snap.Attempts = append(snap.Attempts, a)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return dbError(err)
		}
		if len(snap.Attempts) > 1000 {
			return ErrLimit
		}
		data, err := json.Marshal(snap)
		if err != nil || len(data) > 1<<20 {
			return ErrLimit
		}
		if _, err = tx.Exec(ctx, `INSERT INTO confignostic.snapshots VALUES($1,$2,$3) ON CONFLICT(workspace_id) DO UPDATE SET cursor=EXCLUDED.cursor,snapshot=EXCLUDED.snapshot`, ws.String(), head, data); err != nil {
			return dbError(err)
		}
		if _, err = tx.Exec(ctx, `DELETE FROM confignostic.events WHERE workspace_id=$1 AND sequence<=$2`, ws.String(), counterArg(through)); err != nil {
			return dbError(err)
		}
		_, err = tx.Exec(ctx, `UPDATE confignostic.workspaces SET retained_after=$2 WHERE id=$1`, ws.String(), counterArg(through))
		return dbError(err)
	})
}
