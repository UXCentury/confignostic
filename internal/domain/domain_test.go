package domain

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
)

func TestResourceIdentitiesAndPrecision(t *testing.T) {
	workspace, err := NewID[WorkspaceKind]()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(workspace)
	if err != nil {
		t.Fatal(err)
	}
	var decoded WorkspaceID
	if err = json.Unmarshal(encoded, &decoded); err != nil || decoded != workspace {
		t.Fatalf("ID round trip: %v", err)
	}
	var task TaskID
	if json.Unmarshal(encoded, &task) == nil {
		t.Fatal("workspace ID accepted as task ID")
	}
	for _, value := range []string{"", "provider-session-42", "wsp_" + strings.Repeat("A", 32), "wsp_" + strings.Repeat("0", 31)} {
		if _, err := ParseID[WorkspaceKind](value); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("invalid ID accepted: %q", value)
		}
	}
	if json.Unmarshal([]byte("null"), &decoded) == nil {
		t.Fatal("null ID accepted")
	}
	if _, err := json.Marshal(WorkspaceID("")); err == nil {
		t.Fatal("empty ID serialized")
	}
	for _, number := range []Counter{0, 9007199254740993, math.MaxUint64} {
		encoded, err := json.Marshal(number)
		if err != nil {
			t.Fatal(err)
		}
		var roundTrip Counter
		if json.Unmarshal(encoded, &roundTrip) != nil || roundTrip != number || encoded[0] != '"' {
			t.Fatalf("counter precision lost: %s", encoded)
		}
	}
	for _, value := range []string{`1`, `"01"`, `"-1"`, `"18446744073709551616"`, `null`, `"1.0"`} {
		var number Counter
		if json.Unmarshal([]byte(value), &number) == nil {
			t.Fatalf("invalid counter accepted: %s", value)
		}
	}
}

func attempt(state State, external bool) Attempt {
	result := Attempt{ID: "atm_00000000000000000000000000000001", WorkspaceID: "wsp_00000000000000000000000000000001", TaskID: "tsk_00000000000000000000000000000001", Number: 1, State: state}
	if external {
		result.DispatchID = "dsp_00000000000000000000000000000001"
	}
	return result
}
func TestAttemptEvidenceGuards(t *testing.T) {
	for _, tc := range []struct {
		name     string
		from     State
		external bool
		to       State
		evidence Evidence
		valid    bool
	}{
		{"local approval", Queued, false, AwaitingApproval, LocalDecision, true},
		{"local denial", AwaitingApproval, false, Canceled, LocalDecision, true},
		{"pre-dispatch cannot complete", Queued, false, Completed, ProviderStopped, false},
		{"request cancellation", Running, true, CancelRequested, LocalDecision, true},
		{"cancel is not stopped", CancelRequested, true, Canceled, LocalDecision, false},
		{"confirmed cancel", CancelRequested, true, Canceled, ProviderStopped, true},
		{"unconfirmed timeout", Running, true, Failed, OutcomeUncertain, false},
		{"timeout remains unknown", Running, true, Unknown, OutcomeUncertain, true},
		{"must reconcile unknown", Unknown, true, Completed, ProviderStopped, false},
		{"reconciled completion", Unknown, true, Completed, Reconciled, true},
		{"remote denial cannot imply stop", AwaitingApproval, true, Canceled, LocalDecision, false},
		{"confirmed remote denial", AwaitingApproval, true, Canceled, ProviderStopped, true},
		{"approval resumes remote", AwaitingApproval, true, Running, ProviderObserved, true},
		{"remote approval never requeues", AwaitingApproval, true, Queued, LocalDecision, false},
		{"ordinary progress cannot retract cancel", CancelRequested, true, Running, ProviderObserved, false},
		{"explicit rejected cancellation", CancelRequested, true, Running, CancellationRejected, true},
		{"late completion while awaiting input", AwaitingInput, true, Completed, ProviderStopped, true},
		{"terminal immutable", Completed, true, Running, Reconciled, false},
		{"no unknown before dispatch", Queued, false, Unknown, OutcomeUncertain, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := attempt(tc.from, tc.external)
			after, err := before.Apply(Transition{To: tc.to, Evidence: tc.evidence})
			if (err == nil) != tc.valid {
				t.Fatalf("transition result: %v", err)
			}
			if tc.valid && (after.State != tc.to || after.TaskID != before.TaskID || after.ID != before.ID) {
				t.Fatal("identity/state changed incorrectly")
			}
			if before.State != tc.from {
				t.Fatal("input attempt mutated")
			}
		})
	}
}
func TestDispatchAndRetryIdentities(t *testing.T) {
	before := attempt(Queued, false)
	if _, err := before.Apply(Transition{To: Dispatched, Evidence: DispatchRecorded}); err == nil {
		t.Fatal("dispatch without identity accepted")
	}
	dispatched, err := before.Apply(Transition{To: Dispatched, Evidence: DispatchRecorded, DispatchID: "dsp_00000000000000000000000000000001"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dispatched.Apply(Transition{To: Running, Evidence: ProviderObserved, DispatchID: "dsp_00000000000000000000000000000002"}); err == nil {
		t.Fatal("dispatch identity replaced")
	}
	failed, err := dispatched.Apply(Transition{To: Failed, Evidence: ProviderStopped})
	if err != nil {
		t.Fatal(err)
	}
	nextID := AttemptID("atm_00000000000000000000000000000002")
	retry, err := Retry(failed, nextID, true)
	if err != nil || retry.TaskID != failed.TaskID || retry.ID == failed.ID || retry.Number != 2 || retry.State != Queued || retry.DispatchID != "" {
		t.Fatalf("retry: %+v %v", retry, err)
	}
	if _, err = Retry(failed, nextID, false); !errors.Is(err, ErrRetryDenied) {
		t.Fatal("unauthorized retry accepted")
	}
	if _, err = Retry(failed, failed.ID, true); !errors.Is(err, ErrRetryDenied) {
		t.Fatal("attempt identity reused")
	}
	if _, err = Retry(attempt(Unknown, true), nextID, true); !errors.Is(err, ErrRetryDenied) {
		t.Fatal("uncertain execution retried blindly")
	}
	if _, err = Retry(attempt(Completed, true), nextID, true); !errors.Is(err, ErrRetryDenied) {
		t.Fatal("completed task reopened")
	}
	failed.Number = math.MaxUint32
	if _, err = Retry(failed, nextID, true); err == nil {
		t.Fatal("attempt counter overflow")
	}
}
