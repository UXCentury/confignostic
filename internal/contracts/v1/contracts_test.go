package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/UXCentury/confignostic/internal/domain"
)

const testWorkspace = domain.WorkspaceID("wsp_00000000000000000000000000000001")
const testRunner = domain.RunnerID("rnr_00000000000000000000000000000001")

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func changed(t *testing.T, data []byte, edit func(map[string]any)) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	edit(value)
	result, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func encoded(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func TestWireFixturesRoundTripAndCompileOffline(t *testing.T) {
	workflow, err := DecodeWorkflow(fixture(t, "workflow"))
	if err != nil {
		t.Fatalf("workflow: %v", err)
	}
	if _, err = DecodeWorkflow(encoded(t, workflow)); err != nil {
		t.Fatal(err)
	}
	event, err := DecodeEvent(fixture(t, "event"), testWorkspace)
	if err != nil {
		t.Fatalf("event: %v", err)
	}
	if event.Sequence != 9007199254740993 || event.Provider.SessionID != "provider-session-42" {
		t.Fatal("external identity or sequence changed")
	}
	data, err := EncodeEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeEvent(data, testWorkspace)
	if err != nil || !bytes.Equal(encoded(t, event), encoded(t, again)) {
		t.Fatalf("event round trip: %v", err)
	}
	dispatch, err := DecodeDispatch(fixture(t, "dispatch"), testWorkspace)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if _, err = DecodeDispatch(encoded(t, dispatch), testWorkspace); err != nil {
		t.Fatal(err)
	}
	ack, err := DecodeAcknowledgment(fixture(t, "ack"), testWorkspace)
	if err != nil {
		t.Fatalf("ack: %v", err)
	}
	if err = MatchAcknowledgment(dispatch, ack, testRunner); err != nil {
		t.Fatal(err)
	}
	hello, err := DecodeHello(fixture(t, "hello"))
	if err != nil {
		t.Fatal(err)
	}
	welcome, err := Negotiate(hello, testWorkspace, testRunner)
	if err != nil || welcome.ProtocolVersion != "1" || welcome.MaxMessageBytes != MaxRunnerBytes {
		t.Fatalf("negotiation: %+v %v", welcome, err)
	}
	if _, err = DecodeWelcome(encoded(t, welcome)); err != nil {
		t.Fatal(err)
	}
	problem, err := DecodeError(fixture(t, "error"))
	if err != nil || problem.Error.Retry != ReconcileFirst {
		t.Fatal("uncertain error lost reconciliation semantics")
	}
	if _, err = DecodeError(encoded(t, problem)); err != nil {
		t.Fatal(err)
	}
	if compileErr != nil || len(compiled) != 8 {
		t.Fatalf("embedded schemas: %v", compileErr)
	}
}
func TestStrictWorkflowBoundary(t *testing.T) {
	baseline := fixture(t, "workflow")
	cases := map[string][]byte{
		"duplicate key":         append([]byte(`{"schema_version":"1",`), bytes.TrimSpace(baseline)[1:]...),
		"escaped duplicate key": []byte(`{"schema_version":"1","\u0073chema_version":"1"}`),
		"trailing document":     append(append([]byte{}, baseline...), []byte("{}")...),
		"unsupported version":   changed(t, baseline, func(d map[string]any) { d["schema_version"] = "2" }),
		"case alias":            changed(t, baseline, func(d map[string]any) { d["SCHEMA_VERSION"] = d["schema_version"]; delete(d, "schema_version") }),
		"unknown field":         changed(t, baseline, func(d map[string]any) { d["secret"] = "synthetic-canary" }),
		"null optional":         changed(t, baseline, func(d map[string]any) { d["edges"] = nil }),
		"numeric revision":      changed(t, baseline, func(d map[string]any) { d["revision"] = float64(1) }),
		"counter overflow":      changed(t, baseline, func(d map[string]any) { d["revision"] = "18446744073709551616" }),
		"zero revision":         changed(t, baseline, func(d map[string]any) { d["revision"] = "0" }),
		"invalid UTF-8":         append(append([]byte{}, baseline...), 0xff),
		"deep nesting":          []byte(strings.Repeat("[", MaxJSONDepth+2) + "true" + strings.Repeat("]", MaxJSONDepth+2)),
		"numeric exponent bomb": []byte(`{"schema_version":"1","unknown":1e999999999}`),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeWorkflow(data); err == nil {
				t.Fatal("invalid document accepted")
			}
		})
	}
	if _, err := DecodeWorkflow(bytes.Repeat([]byte(" "), MaxRequestBytes+1)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("size boundary: %v", err)
	}
}
func TestEventPayloadsAndSSE(t *testing.T) {
	event, err := DecodeEvent(fixture(t, "event"), testWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	event.Payload = encoded(t, Output{Text: "line one\n\nevent: forged\ndata: injected"})
	var frame bytes.Buffer
	if err = WriteSSE(&frame, event); err != nil {
		t.Fatal(err)
	}
	if strings.Count(frame.String(), "\nevent:") != 1 || strings.Count(frame.String(), "\ndata:") != 1 || !strings.HasPrefix(frame.String(), "id: 9007199254740993\n") {
		t.Fatalf("unsafe frame: %s", frame.String())
	}
	event.Provider = &ProviderReference{OperationID: "remote-operation-42"}
	if _, err = EncodeEvent(event); err != nil {
		t.Fatal("operation-only provider reference rejected:", err)
	}
	event.Provider = &ProviderReference{EventID: "event-without-execution-reference"}
	if _, err = EncodeEvent(event); err == nil {
		t.Fatal("provider reference without session/operation accepted")
	}
	event.Provider = nil
	if _, err = DecodeEvent(encoded(t, event), "wsp_00000000000000000000000000000002"); !errors.Is(err, ErrScopeMismatch) {
		t.Fatal("cross-workspace event accepted")
	}
	event.Type = "task.state_changed"
	event.Payload = encoded(t, StateChange{From: domain.Queued, To: domain.Completed})
	if _, err = EncodeEvent(event); err == nil {
		t.Fatal("impossible state notification accepted")
	}
	event.Payload = encoded(t, StateChange{From: domain.Running, To: domain.Completed})
	if _, err = EncodeEvent(event); err != nil {
		t.Fatal(err)
	}
	event.Type = "approval.requested"
	request := ApprovalRequest{ApprovalID: "apr_00000000000000000000000000000001", ActionDigest: strings.Repeat("a", 64), PolicyVersion: "policy-v1", Scope: "fixture workspace", ExpiresAt: event.OccurredAt.Add(time.Minute)}
	event.Payload = encoded(t, request)
	if _, err = EncodeEvent(event); err != nil {
		t.Fatal(err)
	}
	request.ExpiresAt = event.OccurredAt
	event.Payload = encoded(t, request)
	if _, err = EncodeEvent(event); err == nil {
		t.Fatal("already expired approval request accepted")
	}
	event.Type = "artifact.created"
	artifact := ArtifactCreated{ArtifactID: "art_00000000000000000000000000000001", SHA256: strings.Repeat("b", 64), MediaType: "application/json", SizeBytes: 123}
	event.Payload = encoded(t, artifact)
	if _, err = EncodeEvent(event); err != nil {
		t.Fatal(err)
	}
	artifact.SizeBytes = MaxArtifactBytes + 1
	event.Payload = encoded(t, artifact)
	if _, err = EncodeEvent(event); err == nil {
		t.Fatal("oversized artifact metadata accepted")
	}
	artifact.SizeBytes = 0
	artifact.MediaType = "attachment"
	event.Payload = encoded(t, artifact)
	if _, err = EncodeEvent(event); err == nil {
		t.Fatal("invalid media type accepted")
	}
	event.Type = "task.output"
	event.Payload = encoded(t, Output{Text: strings.Repeat("界", MaxMessageBytes/3+1)})
	if _, err = EncodeEvent(event); err == nil {
		t.Fatal("UTF-8 byte ceiling ignored")
	}
	// A compact peer can send HTML-sensitive characters without escaping.
	// Reject values that cannot fit in our own serialized envelope.
	peer := changed(t, fixture(t, "event"), func(value map[string]any) {
		value["payload"] = map[string]any{"text": strings.Repeat("<", MaxMessageBytes)}
	})
	var value any
	if err = json.Unmarshal(peer, &value); err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	encoder := json.NewEncoder(&compact)
	encoder.SetEscapeHTML(false)
	if err = encoder.Encode(value); err != nil {
		t.Fatal(err)
	}
	if compact.Len() >= MaxEventBytes {
		t.Fatal("test input must fit before encoding")
	}
	if _, err = DecodeEvent(compact.Bytes(), testWorkspace); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expanded event size ignored: %v", err)
	}
}
func TestRunnerIdentityFencingAndNegotiation(t *testing.T) {
	dispatch, err := DecodeDispatch(fixture(t, "dispatch"), testWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	ack, err := DecodeAcknowledgment(fixture(t, "ack"), testWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	ack.LeaseEpoch--
	if err = MatchAcknowledgment(dispatch, ack, testRunner); !errors.Is(err, ErrAcknowledgmentMismatch) {
		t.Fatal("stale lease acknowledgment accepted")
	}
	ack.LeaseEpoch = dispatch.LeaseEpoch
	if err = MatchAcknowledgment(dispatch, ack, "rnr_00000000000000000000000000000002"); !errors.Is(err, ErrAcknowledgmentMismatch) {
		t.Fatal("wrong runner accepted")
	}
	ack.Status = "rejected"
	if _, err = DecodeAcknowledgment(encoded(t, ack), testWorkspace); err == nil {
		t.Fatal("rejection without reason accepted")
	}
	problem := NewProblem("unavailable", "Runner cannot accept new dispatches.")
	ack.Error = &problem
	if _, err = DecodeAcknowledgment(encoded(t, ack), testWorkspace); err != nil {
		t.Fatal(err)
	}
	ack.Error.Retry = Never
	if _, err = DecodeAcknowledgment(encoded(t, ack), testWorkspace); err == nil {
		t.Fatal("incorrect retry disposition accepted")
	}
	uncertain := NewProblem("uncertain_outcome", "Reconcile the existing execution.")
	ack.Error = &uncertain
	if _, err = DecodeAcknowledgment(encoded(t, ack), testWorkspace); err == nil {
		t.Fatal("uncertain execution described as an admission rejection")
	}
	badError := changed(t, fixture(t, "error"), func(value map[string]any) { value["error"].(map[string]any)["retry"] = "backoff" })
	if _, err = DecodeError(badError); err == nil {
		t.Fatal("uncertain outcome marked retryable")
	}
	hello, err := DecodeHello(fixture(t, "hello"))
	if err != nil {
		t.Fatal(err)
	}
	hello.ProtocolVersions = []string{"2"}
	if _, err = Negotiate(hello, testWorkspace, testRunner); !errors.Is(err, ErrProtocolMismatch) {
		t.Fatal("unsupported runner version negotiated")
	}
	hello.ProtocolVersions = []string{"1"}
	if _, err = Negotiate(hello, "wsp_00000000000000000000000000000002", testRunner); !errors.Is(err, ErrScopeMismatch) {
		t.Fatal("wrong enrollment workspace accepted")
	}
	forged := changed(t, fixture(t, "dispatch"), func(d map[string]any) { d["credentials"] = "synthetic-canary" })
	if _, err = DecodeDispatch(forged, testWorkspace); err == nil {
		t.Fatal("credential field accepted")
	}
	partial := changed(t, fixture(t, "dispatch"), func(d map[string]any) { d["workflow_run_id"] = "run_00000000000000000000000000000001" })
	if _, err = DecodeDispatch(partial, testWorkspace); err == nil {
		t.Fatal("unpinned workflow dispatch accepted")
	}
}
func TestCursorAndWriterErrors(t *testing.T) {
	for _, value := range []string{"", "0", "9007199254740993", "18446744073709551615"} {
		if _, err := ParseCursor(value); err != nil {
			t.Fatalf("%s: %v", value, err)
		}
	}
	for _, value := range []string{"01", "-1", "1e3", " 1", "1\n", "18446744073709551616"} {
		if _, err := ParseCursor(value); err == nil {
			t.Fatalf("bad cursor accepted: %q", value)
		}
	}
	event, err := DecodeEvent(fixture(t, "event"), testWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if err = WriteSSE(failingWriter{}, event); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write failure ignored: %v", err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func FuzzDecodeWorkflow(f *testing.F) {
	baseline, err := os.ReadFile("testdata/workflow.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(baseline)
	f.Add([]byte(`{"schema_version":"2"}`))
	f.Add([]byte(`{"schema_version":"1","schema_version":"1"}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		workflow, err := DecodeWorkflow(data)
		if err != nil {
			return
		}
		roundTrip, err := json.Marshal(workflow)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = DecodeWorkflow(roundTrip); err != nil {
			t.Fatalf("accepted graph cannot round-trip: %v", err)
		}
	})
}
