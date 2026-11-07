package storage

import (
	"context"
	"encoding/json"

	v1 "github.com/UXCentury/confignostic/internal/contracts/v1"
	"github.com/UXCentury/confignostic/internal/domain"
	"github.com/UXCentury/confignostic/internal/workflows"
	"github.com/jackc/pgx/v5"
)

func workflowBytes(d workflows.Definition) ([]byte, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return nil, ErrInvalid
	}
	if _, err = v1.DecodeWorkflow(data); err != nil {
		return nil, err
	}
	return data, nil
}
func (s *Store) CreateWorkflow(ctx context.Context, d workflows.Definition) error {
	data, err := workflowBytes(d)
	if err != nil {
		return err
	}
	if d.Revision != 1 {
		return ErrInvalid
	}
	return s.transaction(ctx, func(tx pgx.Tx) error {
		if err := lockWorkspace(ctx, tx, d.WorkspaceID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO confignostic.workflows VALUES($1,$2,1)`, d.WorkspaceID.String(), d.ID.String()); err != nil {
			return dbError(err)
		}
		_, err := tx.Exec(ctx, `INSERT INTO confignostic.workflow_versions VALUES($1,$2,$3,1,$4)`, d.WorkspaceID.String(), d.ID.String(), d.VersionID.String(), data)
		return dbError(err)
	})
}
func (s *Store) WorkflowVersion(ctx context.Context, ws domain.WorkspaceID, id domain.WorkflowVersionID) (workflows.Definition, error) {
	if ws.Validate() != nil || id.Validate() != nil {
		return workflows.Definition{}, ErrInvalid
	}
	var data []byte
	if err := s.pool.QueryRow(ctx, `SELECT definition FROM confignostic.workflow_versions WHERE workspace_id=$1 AND id=$2`, ws.String(), id.String()).Scan(&data); err != nil {
		return workflows.Definition{}, dbError(err)
	}
	return v1.DecodeWorkflow(compactJSON(data))
}
func (s *Store) ReviseWorkflow(ctx context.Context, ws domain.WorkspaceID, id domain.WorkflowID, expected domain.Counter, body workflows.Body) (workflows.Definition, error) {
	var next workflows.Definition
	if id.Validate() != nil || expected == 0 || expected == ^domain.Counter(0) {
		return next, ErrInvalid
	}
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		if err := lockWorkspace(ctx, tx, ws); err != nil {
			return err
		}
		var current string
		if err := tx.QueryRow(ctx, `SELECT revision::text FROM confignostic.workflows WHERE workspace_id=$1 AND id=$2 FOR UPDATE`, ws.String(), id.String()).Scan(&current); err != nil {
			return dbError(err)
		}
		if current != counterArg(expected) {
			return workflows.ErrRevisionConflict
		}
		version, err := domain.NewID[domain.WorkflowVersionKind]()
		if err != nil {
			return ErrUnavailable
		}
		next = workflows.Definition{SchemaVersion: v1.Version, WorkspaceID: ws, ID: id, VersionID: version, Revision: expected + 1, Body: body}
		data, err := workflowBytes(next)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO confignostic.workflow_versions VALUES($1,$2,$3,$4,$5)`, ws.String(), id.String(), version.String(), counterArg(next.Revision), data); err != nil {
			return dbError(err)
		}
		_, err = tx.Exec(ctx, `UPDATE confignostic.workflows SET revision=$3 WHERE workspace_id=$1 AND id=$2`, ws.String(), id.String(), counterArg(next.Revision))
		return dbError(err)
	})
	return next, err
}
