package workflows

import (
	"errors"
	"maps"
	"math"
	"slices"
	"sync"

	"github.com/UXCentury/confignostic/internal/domain"
)

var ErrRevisionConflict = errors.New("workflow revision conflict")

// Draft is a concurrency-safe reference aggregate, not a durable repository.
// Stage D must enforce the same compare-and-swap in a database transaction.
// It is deliberately not used as API storage.
type Draft struct {
	mu      sync.Mutex
	current Definition
}

func NewDraft(workspaceID domain.WorkspaceID, id domain.WorkflowID, body Body) (*Draft, error) {
	versionID, err := domain.NewID[domain.WorkflowVersionKind]()
	if err != nil {
		return nil, err
	}
	d := Definition{SchemaVersion: SchemaVersion, ID: id, WorkspaceID: workspaceID, VersionID: versionID, Revision: 1, Body: body}
	if err := Validate(d); err != nil {
		return nil, err
	}
	return &Draft{current: clone(d)}, nil
}

func (d *Draft) Snapshot() Definition {
	d.mu.Lock()
	defer d.mu.Unlock()
	return clone(d.current)
}

func (d *Draft) Revise(expected domain.Counter, body Body) (Definition, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if expected != d.current.Revision || expected == math.MaxUint64 {
		return Definition{}, ErrRevisionConflict
	}
	next := d.current
	versionID, err := domain.NewID[domain.WorkflowVersionKind]()
	if err != nil {
		return Definition{}, err
	}
	next.VersionID = versionID
	next.Revision++
	next.Body = body
	if err := Validate(next); err != nil {
		return Definition{}, err
	}
	d.current = clone(next)
	return clone(d.current), nil
}

func clone(d Definition) Definition {
	d.Inputs = maps.Clone(d.Inputs)
	d.EntryNodes = slices.Clone(d.EntryNodes)
	d.Edges = slices.Clone(d.Edges)
	d.Nodes = slices.Clone(d.Nodes)
	for i := range d.Nodes {
		d.Nodes[i].Inputs = maps.Clone(d.Nodes[i].Inputs)
		d.Nodes[i].Outputs = maps.Clone(d.Nodes[i].Outputs)
		d.Nodes[i].Requires = slices.Clone(d.Nodes[i].Requires)
	}
	return d
}
