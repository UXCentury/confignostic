// Package workflows defines the versioned graph shared by chat and the canvas.
// Validation is not permission to execute; registry/policy checks remain separate.
package workflows

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/UXCentury/confignostic/internal/domain"
)

const (
	SchemaVersion = "1"
	MaxNodes      = 100
	MaxEdges      = 500
	MaxPorts      = 32
)

type ValueType string

const (
	Text     ValueType = "text"
	JSON     ValueType = "json"
	Artifact ValueType = "artifact"
	Boolean  ValueType = "boolean"
)

func (v ValueType) Valid() bool { return v == Text || v == JSON || v == Artifact || v == Boolean }

type NodeKind string

const (
	Agent     NodeKind = "agent"
	Approval  NodeKind = "approval"
	Condition NodeKind = "condition"
	Transform NodeKind = "transform"
	Join      NodeKind = "join"
)

func (k NodeKind) Valid() bool {
	return k == Agent || k == Approval || k == Condition || k == Transform || k == Join
}

type InputPort struct {
	Type     ValueType `json:"type"`
	Required bool      `json:"required"`
}
type Node struct {
	ID           string               `json:"id"`
	Kind         NodeKind             `json:"kind"`
	ConnectionID domain.ConnectionID  `json:"connection_id,omitempty"`
	Requires     []domain.Capability  `json:"requires,omitempty"`
	Inputs       map[string]InputPort `json:"inputs,omitempty"`
	Outputs      map[string]ValueType `json:"outputs,omitempty"`
}
type PortRef struct {
	NodeID string `json:"node_id"`
	Port   string `json:"port,omitempty"`
}
type Edge struct {
	From PortRef `json:"from"`
	To   PortRef `json:"to"`
}
type Body struct {
	Name       string               `json:"name"`
	Inputs     map[string]ValueType `json:"inputs,omitempty"`
	EntryNodes []string             `json:"entry_nodes"`
	Nodes      []Node               `json:"nodes"`
	Edges      []Edge               `json:"edges,omitempty"`
}
type Definition struct {
	SchemaVersion string                   `json:"schema_version"`
	ID            domain.WorkflowID        `json:"id"`
	WorkspaceID   domain.WorkspaceID       `json:"workspace_id"`
	VersionID     domain.WorkflowVersionID `json:"version_id"`
	Revision      domain.Counter           `json:"revision"`
	Body
}
type Issue struct {
	Path string `json:"path"`
	Code string `json:"code"`
}
type ValidationError struct{ Issues []Issue }

func (e *ValidationError) Error() string { return "workflow validation failed" }

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func Validate(d Definition) error {
	issues := []Issue{}
	add := func(path, code string) {
		if len(issues) < 20 {
			issues = append(issues, Issue{path, code})
		}
	}
	result := func() error {
		if len(issues) > 0 {
			return &ValidationError{issues}
		}
		return nil
	}
	if d.SchemaVersion != SchemaVersion {
		add("/schema_version", "unsupported_version")
	}
	if d.ID.Validate() != nil {
		add("/id", "invalid_id")
	}
	if d.WorkspaceID.Validate() != nil {
		add("/workspace_id", "invalid_id")
	}
	if d.VersionID.Validate() != nil {
		add("/version_id", "invalid_id")
	}
	if d.Revision == 0 {
		add("/revision", "invalid_revision")
	}
	if !utf8.ValidString(d.Name) || strings.TrimSpace(d.Name) == "" || len(d.Name) > 120 || strings.ContainsRune(d.Name, 0) {
		add("/name", "invalid_name")
	}
	if len(d.Nodes) < 1 || len(d.Nodes) > MaxNodes || len(d.Edges) > MaxEdges || len(d.Inputs) > MaxPorts || len(d.EntryNodes) < 1 || len(d.EntryNodes) > MaxNodes {
		add("/", "graph_limit")
		return result()
	}
	for _, key := range sortedKeys(d.Inputs) {
		if !namePattern.MatchString(key) || !d.Inputs[key].Valid() {
			add("/inputs", "invalid_port")
		}
	}
	nodes := map[string]Node{}
	adjacency := map[string]map[string]bool{}
	indegree := map[string]int{}
	bound := map[string]map[string]bool{}
	for i, node := range d.Nodes {
		path := fmt.Sprintf("/nodes/%d", i)
		if !namePattern.MatchString(node.ID) {
			add(path+"/id", "invalid_node_id")
		}
		if _, exists := nodes[node.ID]; exists {
			add(path+"/id", "duplicate_node")
		}
		if !node.Kind.Valid() {
			add(path+"/kind", "unsupported_node")
		}
		if node.Kind == Agent {
			if node.ConnectionID.Validate() != nil {
				add(path+"/connection_id", "invalid_connection")
			}
		} else if node.ConnectionID != "" || len(node.Requires) != 0 {
			add(path, "agent_fields_on_non_agent")
		}
		if domain.ValidateCapabilities(node.Requires) != nil {
			add(path+"/requires", "invalid_capability")
		}
		if len(node.Inputs) > MaxPorts || len(node.Outputs) > MaxPorts {
			add(path, "port_limit")
			continue
		}
		for _, key := range sortedKeys(node.Inputs) {
			if !namePattern.MatchString(key) || !node.Inputs[key].Type.Valid() {
				add(path+"/inputs", "invalid_port")
			}
		}
		for _, key := range sortedKeys(node.Outputs) {
			if !namePattern.MatchString(key) || !node.Outputs[key].Valid() {
				add(path+"/outputs", "invalid_port")
			}
		}
		nodes[node.ID] = node
		adjacency[node.ID] = map[string]bool{}
		bound[node.ID] = map[string]bool{}
	}
	if len(issues) > 0 {
		return result()
	}
	seenEdges := map[Edge]bool{}
	for i, edge := range d.Edges {
		path := fmt.Sprintf("/edges/%d", i)
		if seenEdges[edge] {
			add(path, "duplicate_edge")
			continue
		}
		seenEdges[edge] = true
		source, sourceExists := nodes[edge.From.NodeID]
		target, targetExists := nodes[edge.To.NodeID]
		inputEdge := edge.From.NodeID == "$input"
		if (!sourceExists && !inputEdge) || !targetExists {
			add(path, "missing_node")
			continue
		}
		if edge.From.NodeID == edge.To.NodeID {
			add(path, "cycle")
			continue
		}
		control := edge.From.Port == "" && edge.To.Port == ""
		if inputEdge && control {
			add(path, "input_requires_port")
			continue
		}
		if !control {
			if edge.From.Port == "" || edge.To.Port == "" {
				add(path, "missing_port")
				continue
			}
			var sourceType ValueType
			var found bool
			if inputEdge {
				sourceType, found = d.Inputs[edge.From.Port]
			} else {
				sourceType, found = source.Outputs[edge.From.Port]
			}
			targetPort, targetFound := target.Inputs[edge.To.Port]
			if !found || !targetFound {
				add(path, "missing_port")
				continue
			}
			if sourceType != targetPort.Type {
				add(path, "type_mismatch")
			}
			if bound[target.ID][edge.To.Port] {
				add(path, "multiple_input_bindings")
			}
			bound[target.ID][edge.To.Port] = true
		}
		if !inputEdge && !adjacency[source.ID][target.ID] {
			adjacency[source.ID][target.ID] = true
			indegree[target.ID]++
		}
	}
	for i, node := range d.Nodes {
		for _, key := range sortedKeys(node.Inputs) {
			if node.Inputs[key].Required && !bound[node.ID][key] {
				add(fmt.Sprintf("/nodes/%d/inputs/%s", i, key), "missing_required_input")
			}
		}
	}
	entries := map[string]bool{}
	queue := []string{}
	for i, id := range d.EntryNodes {
		if _, exists := nodes[id]; !exists {
			add(fmt.Sprintf("/entry_nodes/%d", i), "missing_node")
			continue
		}
		if entries[id] {
			add("/entry_nodes", "duplicate_entry")
		}
		if indegree[id] != 0 {
			add("/entry_nodes", "entry_has_dependencies")
		}
		entries[id] = true
		queue = append(queue, id)
	}
	reached := map[string]bool{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if reached[id] {
			continue
		}
		reached[id] = true
		for _, next := range sortedKeys(adjacency[id]) {
			queue = append(queue, next)
		}
	}
	for i, node := range d.Nodes {
		if !reached[node.ID] {
			add(fmt.Sprintf("/nodes/%d", i), "unreachable_node")
		}
	}
	queue = nil
	for _, node := range d.Nodes {
		if indegree[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, next := range sortedKeys(adjacency[id]) {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if visited != len(nodes) {
		add("/edges", "cycle")
	}
	return result()
}

type ConnectionCapabilities struct {
	WorkspaceID  domain.WorkspaceID
	ConnectionID domain.ConnectionID
	Capabilities []domain.Capability
}

// ValidateCapabilities accepts server-discovered descriptors only. A client or
// imported graph cannot certify its own connections or grant authorization.
func ValidateCapabilities(d Definition, connections map[domain.ConnectionID]ConnectionCapabilities) error {
	if err := Validate(d); err != nil {
		return err
	}
	issues := []Issue{}
	for i, node := range d.Nodes {
		if len(issues) >= 20 {
			break
		}
		if node.Kind != Agent {
			continue
		}
		path := fmt.Sprintf("/nodes/%d", i)
		connection, ok := connections[node.ConnectionID]
		if !ok || connection.ConnectionID != node.ConnectionID || connection.WorkspaceID != d.WorkspaceID {
			issues = append(issues, Issue{path + "/connection_id", "connection_unavailable"})
			continue
		}
		if err := domain.ValidateCapabilities(connection.Capabilities); err != nil {
			return errors.New("invalid discovered capability descriptor")
		}
		requirements := append([]domain.Capability{domain.Invoke}, node.Requires...)
		for _, port := range node.Inputs {
			if port.Type == Artifact {
				requirements = append(requirements, domain.Artifacts)
			}
		}
		for _, kind := range node.Outputs {
			if kind == Artifact {
				requirements = append(requirements, domain.Artifacts)
			}
			if kind == JSON {
				requirements = append(requirements, domain.StructuredOutput)
			}
		}
		for _, required := range requirements {
			if !slices.Contains(connection.Capabilities, required) {
				issues = append(issues, Issue{path + "/requires", "capability_unsupported"})
				break
			}
		}
		if len(issues) >= 20 {
			break
		}
	}
	if len(issues) > 0 {
		return &ValidationError{issues}
	}
	return nil
}
