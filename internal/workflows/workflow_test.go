package workflows

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/UXCentury/confignostic/internal/domain"
)

func fixture(t *testing.T) Definition {
	t.Helper()
	data, err := os.ReadFile("../contracts/v1/testdata/workflow.json")
	if err != nil {
		t.Fatal(err)
	}
	var d Definition
	if err = json.Unmarshal(data, &d); err != nil {
		t.Fatal(err)
	}
	return d
}
func hasIssue(err error, code string) bool {
	var validation *ValidationError
	if !errors.As(err, &validation) {
		return false
	}
	for _, issue := range validation.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
func TestGraphValidation(t *testing.T) {
	if err := Validate(fixture(t)); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, code string
		edit       func(*Definition)
	}{
		{"cycle", "cycle", func(d *Definition) {
			d.Edges = append(d.Edges, Edge{From: PortRef{NodeID: "review"}, To: PortRef{NodeID: "implement"}})
		}},
		{"required input", "missing_required_input", func(d *Definition) { d.Edges = nil }},
		{"port types", "type_mismatch", func(d *Definition) { d.Nodes[1].Inputs["change"] = InputPort{Type: Text, Required: true} }},
		{"missing node", "missing_node", func(d *Definition) { d.Edges[1].To.NodeID = "missing" }},
		{"duplicate edge", "duplicate_edge", func(d *Definition) { d.Edges = append(d.Edges, d.Edges[0]) }},
		{"duplicate binding", "multiple_input_bindings", func(d *Definition) {
			d.Inputs["alternate"] = Text
			d.Edges = append(d.Edges, Edge{From: PortRef{"$input", "alternate"}, To: PortRef{"implement", "prompt"}})
		}},
		{"duplicate node", "duplicate_node", func(d *Definition) { d.Nodes[1].ID = d.Nodes[0].ID }},
		{"unreachable", "unreachable_node", func(d *Definition) { d.Nodes = append(d.Nodes, Node{ID: "isolated", Kind: Join}) }},
		{"bad entry", "entry_has_dependencies", func(d *Definition) { d.EntryNodes = []string{"review"} }},
		{"capability name", "invalid_capability", func(d *Definition) { d.Nodes[0].Requires = []domain.Capability{"imaginary"} }},
		{"kind", "unsupported_node", func(d *Definition) { d.Nodes[0].Kind = "execute_shell" }},
		{"too large", "graph_limit", func(d *Definition) {
			for len(d.Nodes) <= MaxNodes {
				d.Nodes = append(d.Nodes, d.Nodes[0])
			}
		}},
		{"wrong version", "unsupported_version", func(d *Definition) { d.SchemaVersion = "2" }},
		{"wrong ID resource", "invalid_id", func(d *Definition) { d.WorkspaceID = domain.WorkspaceID(d.ID.String()) }},
		{"bad port", "invalid_port", func(d *Definition) { d.Nodes[0].Outputs["bad port"] = Text }},
		{"agent fields", "agent_fields_on_non_agent", func(d *Definition) { d.Nodes[0].Kind = Join }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := fixture(t)
			tc.edit(&d)
			if err := Validate(d); !hasIssue(err, tc.code) {
				t.Fatalf("expected %s, got %#v", tc.code, err)
			}
		})
	}
}
func TestDiscoveredCapabilitiesCannotCrossWorkspaces(t *testing.T) {
	d := fixture(t)
	if !hasIssue(ValidateCapabilities(d, nil), "connection_unavailable") {
		t.Fatal("unconfigured agents accepted")
	}
	connections := map[domain.ConnectionID]ConnectionCapabilities{}
	for _, node := range d.Nodes {
		connections[node.ConnectionID] = ConnectionCapabilities{d.WorkspaceID, node.ConnectionID, []domain.Capability{domain.Invoke, domain.Artifacts, domain.StructuredOutput}}
	}
	if err := ValidateCapabilities(d, connections); err != nil {
		t.Fatal(err)
	}
	reviewerID := d.Nodes[1].ConnectionID
	reviewer := connections[reviewerID]
	reviewer.Capabilities = []domain.Capability{domain.Invoke, domain.StructuredOutput}
	connections[reviewerID] = reviewer
	if !hasIssue(ValidateCapabilities(d, connections), "capability_unsupported") {
		t.Fatal("artifact input capability ignored")
	}
	reviewer.Capabilities = append(reviewer.Capabilities, domain.Artifacts)
	connections[reviewerID] = reviewer
	id := d.Nodes[0].ConnectionID
	connection := connections[id]
	connection.Capabilities = nil
	connections[id] = connection
	if !hasIssue(ValidateCapabilities(d, connections), "capability_unsupported") {
		t.Fatal("invoke/artifact requirements ignored")
	}
	connection.WorkspaceID = "wsp_00000000000000000000000000000002"
	connections[id] = connection
	if !hasIssue(ValidateCapabilities(d, connections), "connection_unavailable") {
		t.Fatal("cross-workspace descriptor accepted")
	}
}

func TestConcurrentEditsCompareRevisionAndPreserveSnapshots(t *testing.T) {
	original := fixture(t)
	draft, err := NewDraft(original.WorkspaceID, original.ID, original.Body)
	if err != nil {
		t.Fatal(err)
	}
	base := draft.Snapshot()
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, name := range []string{"first editor", "second editor"} {
		group.Add(1)
		go func(name string) {
			defer group.Done()
			body := base.Body
			body.Name = name
			<-start
			_, err := draft.Revise(base.Revision, body)
			results <- err
		}(name)
	}
	close(start)
	group.Wait()
	close(results)
	success, conflicts := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrRevisionConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflicts=%d", success, conflicts)
	}
	current := draft.Snapshot()
	if current.Revision != 2 || current.VersionID == base.VersionID || base.Name != original.Name {
		t.Fatal("version immutability broken")
	}
	// Mutating submitted data or returned data must not mutate the aggregate.
	original.Nodes[0].Outputs["change"] = Text
	current.Nodes[0].Inputs["prompt"] = InputPort{Type: Boolean}
	current.EntryNodes[0] = "review"
	stable := draft.Snapshot()
	if stable.Nodes[0].Outputs["change"] != Artifact || stable.Nodes[0].Inputs["prompt"].Type != Text || stable.EntryNodes[0] != "implement" {
		t.Fatal("snapshot aliases shared mutable data")
	}
	if _, err := draft.Revise(1, base.Body); !errors.Is(err, ErrRevisionConflict) {
		t.Fatal("stale revision accepted")
	}
	bad := stable.Body
	bad.EntryNodes = []string{"missing"}
	if _, err := draft.Revise(2, bad); err == nil {
		t.Fatal("invalid replacement accepted")
	}
	if draft.Snapshot().Revision != 2 {
		t.Fatal("failed edit advanced revision")
	}
}
