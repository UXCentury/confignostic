//go:build integration

package storage

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/UXCentury/confignostic/internal/domain"
)

func TestLostIngestAcknowledgementRecovery(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		name := "active"
		if terminal {
			name = "terminal"
		}
		t.Run(name, func(t *testing.T) {
			s, dsn, _ := testDB(t, true)
			p := seed(t, s, 1)
			other := seed(t, s, 2)
			accepted := accept(t, s, p, "lost-ack")
			l := claim(t, s, p, "original-worker")
			if err := s.BeginDispatch(t.Context(), l, audit()); err != nil {
				t.Fatal(err)
			}
			delivery := Delivery{Source: "provider-operation-1", ExternalID: "running", Change: domain.Transition{To: domain.Running, Evidence: domain.ProviderObserved}, Audit: audit()}
			if terminal {
				delivery.ExternalID = "completed"
				delivery.Change = domain.Transition{To: domain.Completed, Evidence: domain.ProviderStopped}
			}
			if seq, err := s.DeliveryReceipt(t.Context(), p.Workspace, l.Dispatch, delivery); !errors.Is(err, ErrNotFound) || seq != 0 {
				t.Fatal("uncommitted delivery has a receipt", seq, err)
			}
			// The commit succeeds, but its acknowledgement is discarded by the caller.
			sequence, err := s.Ingest(t.Context(), l, delivery)
			if err != nil {
				t.Fatal(err)
			}
			execute(t, s, `UPDATE confignostic.dispatches SET lease_until=clock_timestamp()-interval '1 second' WHERE workspace_id=$1 AND id=$2`, p.Workspace.String(), l.Dispatch.String())
			expectedState := domain.Completed
			if terminal {
				if _, err := s.Claim(t.Context(), p.Workspace, "replacement-worker", time.Minute, audit()); !errors.Is(err, ErrNotFound) {
					t.Fatal("terminal attempt reclaimed", err)
				}
			} else {
				next := claim(t, s, p, "replacement-worker")
				if next.Mode != "reconcile" || next.Dispatch != l.Dispatch || next.Epoch <= l.Epoch {
					t.Fatal(next)
				}
				expectedState = domain.Unknown
			}
			if _, err := s.Ingest(t.Context(), l, delivery); !errors.Is(err, ErrLeaseLost) {
				t.Fatal("expired ingestion bypassed fencing", err)
			}
			if _, err := s.Renew(t.Context(), l, time.Minute); !errors.Is(err, ErrLeaseLost) {
				t.Fatal(err)
			}
			if err := s.BeginDispatch(t.Context(), l, audit()); !errors.Is(err, ErrLeaseLost) {
				t.Fatal(err)
			}
			// A new connection can recover the committed sequence after lease loss.
			u, err := url.Parse(dsn)
			if err != nil {
				t.Fatal(err)
			}
			query := u.Query()
			query.Set("default_transaction_read_only", "on")
			u.RawQuery = query.Encode()
			reopened, err := Open(t.Context(), u.String())
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			if seq, err := reopened.DeliveryReceipt(t.Context(), p.Workspace, l.Dispatch, delivery); err != nil || seq != sequence {
				t.Fatal("lost acknowledgement was not recoverable", seq, err)
			}
			changed := delivery
			changed.Change.To = domain.Failed
			if seq, err := s.DeliveryReceipt(t.Context(), p.Workspace, l.Dispatch, changed); !errors.Is(err, ErrConflict) || seq != 0 {
				t.Fatal("changed delivery matched receipt", seq, err)
			}
			if seq, err := s.DeliveryReceipt(t.Context(), other.Workspace, l.Dispatch, delivery); !errors.Is(err, ErrNotFound) || seq != 0 {
				t.Fatal("receipt crossed workspace", seq, err)
			}
			missing := delivery
			missing.ExternalID = "not-committed"
			if _, err := s.DeliveryReceipt(t.Context(), p.Workspace, l.Dispatch, missing); !errors.Is(err, ErrNotFound) {
				t.Fatal(err)
			}
			invalid := delivery
			invalid.Source = ""
			if _, err := s.DeliveryReceipt(t.Context(), p.Workspace, l.Dispatch, invalid); !errors.Is(err, ErrInvalid) {
				t.Fatal(err)
			}
			if _, err := s.DeliveryReceipt(t.Context(), "bad", l.Dispatch, delivery); !errors.Is(err, ErrInvalid) {
				t.Fatal(err)
			}
			if _, err := s.DeliveryReceipt(t.Context(), p.Workspace, "bad", delivery); !errors.Is(err, ErrInvalid) {
				t.Fatal(err)
			}
			// The same connection cannot reuse an external identity for another dispatch.
			accept(t, s, p, "other-task")
			var another domain.DispatchID
			if err := s.pool.QueryRow(t.Context(), `SELECT id FROM confignostic.dispatches WHERE workspace_id=$1 AND id<>$2`, p.Workspace.String(), l.Dispatch.String()).Scan(&another); err != nil {
				t.Fatal(err)
			}
			if _, err := s.DeliveryReceipt(t.Context(), p.Workspace, another, delivery); !errors.Is(err, ErrConflict) {
				t.Fatal("receipt crossed dispatch", err)
			}
			// Compact the original event, retaining the receipt and original sequence.
			if terminal {
				// Add a later transition so the completed event can be pruned.
				next := claim(t, s, p, "other-worker")
				if err := s.BeginDispatch(t.Context(), next, audit()); err != nil {
					t.Fatal(err)
				}
			}
			pending, err := s.PendingEvents(t.Context(), p.Workspace, 0, 100)
			if err != nil {
				t.Fatal(err)
			}
			for _, event := range pending.Events {
				if err := s.Published(t.Context(), p.Workspace, event.Sequence); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Compact(t.Context(), p.Workspace, 1); err != nil {
				t.Fatal(err)
			}
			var count int
			if err := s.pool.QueryRow(t.Context(), `SELECT count(*) FROM confignostic.events WHERE workspace_id=$1 AND sequence=$2`, p.Workspace.String(), counterArg(sequence)).Scan(&count); err != nil || count != 0 {
				t.Fatal("fixture did not prune original event", count, err)
			}
			if seq, err := reopened.DeliveryReceipt(t.Context(), p.Workspace, l.Dispatch, delivery); err != nil || seq != sequence {
				t.Fatal("compaction lost recovery receipt", seq, err)
			}
			a, err := s.Attempt(t.Context(), p.Workspace, accepted.Attempt)
			if err != nil || a.State != expectedState {
				t.Fatal("receipt lookup mutated attempt", a, err)
			}
			var total int
			if err := s.pool.QueryRow(t.Context(), `SELECT count(*) FROM confignostic.inbox WHERE workspace_id=$1`, p.Workspace.String()).Scan(&total); err != nil || total != 1 {
				t.Fatal("receipt lookup inserted delivery", total, err)
			}
		})
	}
}
