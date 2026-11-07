// Package storage implements PostgreSQL durability for trusted control-plane code.
// Workspace-scoped methods are not a substitute for caller authentication.
package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/UXCentury/confignostic/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUnavailable   = errors.New("database unavailable")
	ErrSchema        = errors.New("database schema is incompatible or migrations are missing")
	ErrNotFound      = errors.New("workspace-scoped record not found")
	ErrConflict      = errors.New("idempotency key or revision conflict")
	ErrLeaseLost     = errors.New("dispatch lease expired or was superseded")
	ErrInvalid       = errors.New("invalid storage operation")
	ErrLimit         = errors.New("storage operation limit exceeded")
	ErrCursorExpired = errors.New("event cursor expired; load the returned snapshot")
	ErrCommitUnknown = errors.New("commit outcome unknown; reconcile using the same idempotency identity")
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct{ pool *pgxpool.Pool }

// Open never returns a driver error that could contain a credential or SQL value.
func Open(ctx context.Context, url string) (*Store, error) {
	if strings.TrimSpace(url) == "" {
		return nil, ErrInvalid
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, ErrInvalid
	}
	cfg.MaxConns = 8
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.ConnConfig.RuntimeParams["application_name"] = "confignostic"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, ErrUnavailable
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, ErrUnavailable
	}
	return &Store{pool: pool}, nil
}
func (s *Store) Close() { s.pool.Close() }

func dbError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrNotFound
		case "23514", "22001", "22003":
			return ErrInvalid
		}
	}
	return ErrUnavailable
}
func digest(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func bounded(v string, max int) bool {
	return utf8.ValidString(v) && strings.TrimSpace(v) != "" && len(v) <= max && !strings.ContainsRune(v, 0)
}
func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}
func (s *Store) transaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dbError(err)
	}
	defer rollback(tx)
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return ErrCommitUnknown
	}
	return nil
}
func lockWorkspace(ctx context.Context, tx pgx.Tx, ws domain.WorkspaceID) error {
	if ws.Validate() != nil {
		return ErrInvalid
	}
	var id string
	return dbError(tx.QueryRow(ctx, `SELECT id FROM confignostic.workspaces WHERE id=$1 FOR UPDATE`, ws.String()).Scan(&id))
}

// Migrate holds one transaction-scoped advisory lock for the entire upgrade.
// The ledger, checksums and schema changes commit together; no down migration runs.
func (s *Store) Migrate(ctx context.Context) error { return s.migrate(ctx, 2) }
func (s *Store) migrate(ctx context.Context, target int) error {
	return s.transaction(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(1129270854)`); err != nil {
			return dbError(err)
		}
		if _, err := tx.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS confignostic; CREATE TABLE IF NOT EXISTS confignostic.schema_migrations(version integer PRIMARY KEY, checksum text NOT NULL)`); err != nil {
			return dbError(err)
		}
		applied, err := migrationState(ctx, tx)
		if err != nil {
			return err
		}
		names := []string{"001_core.sql", "002_delivery.sql"}
		if len(applied) > target {
			return ErrSchema
		}
		for i, name := range names[:target] {
			sql, err := migrations.ReadFile("migrations/" + name)
			if err != nil {
				return ErrSchema
			}
			checksum := fmt.Sprintf("%x", sha256.Sum256(sql))
			if i < len(applied) {
				if applied[i] != checksum {
					return ErrSchema
				}
				continue
			}
			if _, err = tx.Exec(ctx, string(sql)); err != nil {
				return dbError(err)
			}
			if _, err = tx.Exec(ctx, `INSERT INTO confignostic.schema_migrations VALUES($1,$2)`, i+1, checksum); err != nil {
				return dbError(err)
			}
		}
		return nil
	})
}

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func migrationState(ctx context.Context, q querier) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT version,checksum FROM confignostic.schema_migrations ORDER BY version`)
	if err != nil {
		return nil, ErrSchema
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var version int
		var checksum string
		if rows.Scan(&version, &checksum) != nil || version != len(result)+1 {
			return nil, ErrSchema
		}
		result = append(result, checksum)
	}
	return result, dbError(rows.Err())
}
func (s *Store) Ready(ctx context.Context) error {
	if s.pool.Ping(ctx) != nil {
		return ErrUnavailable
	}
	var writable bool
	if err := s.pool.QueryRow(ctx, `SELECT current_setting('transaction_read_only')='off' AND NOT pg_is_in_recovery()`).Scan(&writable); err != nil || !writable {
		return ErrUnavailable
	}
	applied, err := migrationState(ctx, s.pool)
	if err != nil || len(applied) != 2 {
		return ErrSchema
	}
	for i, name := range []string{"001_core.sql", "002_delivery.sql"} {
		sql, _ := migrations.ReadFile("migrations/" + name)
		if applied[i] != fmt.Sprintf("%x", sha256.Sum256(sql)) {
			return ErrSchema
		}
	}
	return nil
}

// Provision establishes internal records atomically. No HTTP provisioning route
// exists yet; stage E must authorize bootstrap and membership before exposing it.
type Provision struct {
	Workspace    domain.WorkspaceID
	Project      domain.ProjectID
	Agent        domain.AgentID
	Connection   domain.ConnectionID
	Conversation domain.ConversationID
	Name         string
}

func (s *Store) Provision(ctx context.Context, p Provision) error {
	if p.Workspace.Validate() != nil || p.Project.Validate() != nil || p.Agent.Validate() != nil || p.Connection.Validate() != nil || p.Conversation.Validate() != nil || !bounded(p.Name, 120) {
		return ErrInvalid
	}
	return s.transaction(ctx, func(tx pgx.Tx) error {
		statements := []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO confignostic.workspaces(id,name) VALUES($1,$2)`, []any{p.Workspace.String(), p.Name}},
			{`INSERT INTO confignostic.projects VALUES($1,$2,$3)`, []any{p.Workspace.String(), p.Project.String(), p.Name}},
			{`INSERT INTO confignostic.agents VALUES($1,$2,$3)`, []any{p.Workspace.String(), p.Agent.String(), p.Name}},
			{`INSERT INTO confignostic.connections VALUES($1,$2,$3)`, []any{p.Workspace.String(), p.Connection.String(), p.Agent.String()}},
			{`INSERT INTO confignostic.conversations VALUES($1,$2,$3)`, []any{p.Workspace.String(), p.Conversation.String(), p.Project.String()}},
		}
		for _, stmt := range statements {
			if _, err := tx.Exec(ctx, stmt.sql, stmt.args...); err != nil {
				return dbError(err)
			}
		}
		return nil
	})
}

func compactJSON(data []byte) []byte {
	var b bytes.Buffer
	if json.Compact(&b, data) != nil {
		return data
	}
	return b.Bytes()
}
