//go:build integration

package storage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	v1 "github.com/UXCentury/confignostic/internal/contracts/v1"
	"github.com/UXCentury/confignostic/internal/domain"
	"github.com/UXCentury/confignostic/internal/workflows"
	"github.com/jackc/pgx/v5"
)

func testDB(t *testing.T, migrate bool) (*Store, string, string) {
	t.Helper()
	base := os.Getenv("CONFIGNOSTIC_TEST_DATABASE_URL")
	if base == "" {
		t.Skip("set CONFIGNOSTIC_TEST_DATABASE_URL for real PostgreSQL integration tests")
	}
	admin, err := Open(t.Context(), base)
	if err != nil {
		t.Fatal(err)
	}
	id, err := domain.NewID[domain.RequestKind]()
	if err != nil {
		t.Fatal(err)
	}
	name := "confignostic_test_" + strings.TrimPrefix(id.String(), "req_")
	if _, err = admin.pool.Exec(t.Context(), "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal("create isolated test database:", dbError(err))
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "postgres" && u.Scheme != "postgresql" {
		t.Fatal("test database must use a PostgreSQL URL")
	}
	u.Path = "/" + name
	// A dbname query override must not redirect the isolated connection.
	query := u.Query()
	query.Del("dbname")
	u.RawQuery = query.Encode()
	dsn := u.String()
	s, err := Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := admin.pool.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
		admin.Close()
		if err != nil {
			t.Error("cleanup isolated database failed")
		}
	})
	if migrate {
		if err = s.Migrate(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	return s, dsn, name
}
func seed(t *testing.T, s *Store, n int) Provision {
	t.Helper()
	p := Provision{Workspace: domain.WorkspaceID(fmt.Sprintf("wsp_%032x", n)), Project: domain.ProjectID(fmt.Sprintf("prj_%032x", n)), Agent: domain.AgentID(fmt.Sprintf("agt_%032x", n)), Connection: domain.ConnectionID(fmt.Sprintf("con_%032x", n)), Conversation: domain.ConversationID(fmt.Sprintf("cnv_%032x", n)), Name: "Integration fixture"}
	if err := s.Provision(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	return p
}
func audit() Audit {
	return Audit{Actor: domain.Actor{Kind: domain.Human, ID: "act_00000000000000000000000000000001"}, Correlation: "req_00000000000000000000000000000001"}
}
func submission(p Provision, key string) Submission {
	return Submission{Workspace: p.Workspace, Conversation: p.Conversation, Connection: p.Connection, Key: key, Input: v1.Input{Message: "bounded fixture task"}, Audit: audit()}
}
func accept(t *testing.T, s *Store, p Provision, key string) Accepted {
	t.Helper()
	a, err := s.Submit(t.Context(), submission(p, key))
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func claim(t *testing.T, s *Store, p Provision, owner string) Lease {
	t.Helper()
	l, err := s.Claim(t.Context(), p.Workspace, owner, time.Minute, audit())
	if err != nil {
		t.Fatal(err)
	}
	return l
}
func execute(t *testing.T, s *Store, sql string, args ...any) {
	t.Helper()
	if _, err := s.pool.Exec(t.Context(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationsUpgradeAndDrift(t *testing.T) {
	s, dsn, _ := testDB(t, false)
	if !errors.Is(s.Ready(t.Context()), ErrSchema) {
		t.Fatal("unmigrated database reported ready")
	}
	if err := s.migrate(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	p := seed(t, s, 1)
	if !errors.Is(s.Ready(t.Context()), ErrSchema) {
		t.Fatal("old database reported ready")
	}
	var group sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		group.Add(1)
		go func() { defer group.Done(); errs <- s.Migrate(t.Context()) }()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal("concurrent migration:", err)
		}
	}
	if err := s.Ready(t.Context()); err != nil {
		t.Fatal(err)
	}
	accept(t, s, p, "after-upgrade")
	u, _ := url.Parse(dsn)
	query := u.Query()
	query.Set("default_transaction_read_only", "on")
	u.RawQuery = query.Encode()
	readOnly, err := Open(t.Context(), u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer readOnly.Close()
	if !errors.Is(readOnly.Ready(t.Context()), ErrUnavailable) {
		t.Fatal("read-only database reported ready")
	}
	execute(t, s, `UPDATE confignostic.schema_migrations SET checksum='tampered' WHERE version=1`)
	if !errors.Is(s.Migrate(t.Context()), ErrSchema) || !errors.Is(s.Ready(t.Context()), ErrSchema) {
		t.Fatal("checksum drift accepted")
	}
}
func TestSubmissionIdempotencyAndWorkspaceIsolation(t *testing.T) {
	s, dsn, _ := testDB(t, true)
	p := seed(t, s, 1)
	other := seed(t, s, 2)
	results := make(chan Accepted, 12)
	errs := make(chan error, 12)
	var group sync.WaitGroup
	for range 12 {
		group.Add(1)
		go func() {
			defer group.Done()
			a, err := s.Submit(t.Context(), submission(p, "same-key"))
			results <- a
			errs <- err
		}()
	}
	group.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first Accepted
	for a := range results {
		if first.Task == "" {
			first = a
		}
		if a != first {
			t.Fatal("duplicate submission created different identities")
		}
	}
	changed := submission(p, "same-key")
	changed.Input.Message = "changed"
	if _, err := s.Submit(t.Context(), changed); !errors.Is(err, ErrConflict) {
		t.Fatal("changed payload accepted:", err)
	}
	changed = submission(p, "same-key")
	changed.Audit.Correlation = "req_00000000000000000000000000000002"
	if a, err := s.Submit(t.Context(), changed); err != nil || a != first {
		t.Fatal("new correlation changed idempotency", err)
	}
	changed = submission(other, "cross-scope")
	changed.Connection = p.Connection
	if _, err := s.Submit(t.Context(), changed); !errors.Is(err, ErrNotFound) {
		t.Fatal("cross-workspace connection accepted:", err)
	}
	if _, err := s.Attempt(t.Context(), other.Workspace, first.Attempt); !errors.Is(err, ErrNotFound) {
		t.Fatal("cross-workspace read:", err)
	}
	a2 := accept(t, s, other, "same-key")
	if a2.Task == first.Task {
		t.Fatal("workspace idempotency collided")
	}
	reopened, err := Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if a, err := reopened.Submit(t.Context(), submission(p, "same-key")); err != nil || a != first {
		t.Fatal("accepted work did not survive reconnect:", err)
	}
	execute(t, s, `INSERT INTO confignostic.artifacts VALUES($1,$2,$3,$4,1,'text/plain')`, p.Workspace.String(), "art_00000000000000000000000000000001", first.Attempt.String(), strings.Repeat("a", 64))
	changed = submission(other, "cross-artifact")
	changed.Input.ArtifactRefs = []domain.ArtifactID{"art_00000000000000000000000000000001"}
	if _, err := s.Submit(t.Context(), changed); !errors.Is(err, ErrNotFound) {
		t.Fatal("cross-workspace artifact reference accepted:", err)
	}
	changed = submission(p, "artifact")
	changed.Input.ArtifactRefs = []domain.ArtifactID{"art_00000000000000000000000000000001"}
	if _, err := s.Submit(t.Context(), changed); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.pool.QueryRow(t.Context(), `SELECT count(*) FROM confignostic.tasks WHERE workspace_id=$1`, other.Workspace.String()).Scan(&count); err != nil || count != 1 {
		t.Fatal("failed submissions left partial tasks", count, err)
	}
}
func TestWorkflowCASPersistsImmutableVersions(t *testing.T) {
	s, _, _ := testDB(t, true)
	p := seed(t, s, 1)
	other := seed(t, s, 2)
	data, err := os.ReadFile("../contracts/v1/testdata/workflow.json")
	if err != nil {
		t.Fatal(err)
	}
	d, err := v1.DecodeWorkflow(data)
	if err != nil {
		t.Fatal(err)
	}
	d.WorkspaceID = p.Workspace
	if err = s.CreateWorkflow(t.Context(), d); err != nil {
		t.Fatal(err)
	}
	errs := make(chan error, 2)
	versions := make(chan workflows.Definition, 2)
	var group sync.WaitGroup
	for _, name := range []string{"one", "two"} {
		group.Add(1)
		go func(name string) {
			defer group.Done()
			body := d.Body
			body.Name = name
			version, err := s.ReviseWorkflow(t.Context(), p.Workspace, d.ID, 1, body)
			versions <- version
			errs <- err
		}(name)
	}
	group.Wait()
	close(errs)
	close(versions)
	success, conflict := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, workflows.ErrRevisionConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("lost edit:", success, conflict)
	}
	old, err := s.WorkflowVersion(t.Context(), p.Workspace, d.VersionID)
	if err != nil || old.Name != d.Name || old.Revision != 1 {
		t.Fatal("old version mutated:", err)
	}
	for version := range versions {
		if version.VersionID != "" {
			persisted, err := s.WorkflowVersion(t.Context(), p.Workspace, version.VersionID)
			if err != nil || persisted.Revision != 2 {
				t.Fatal(err)
			}
		}
	}
	if _, err := s.WorkflowVersion(t.Context(), other.Workspace, d.VersionID); !errors.Is(err, ErrNotFound) {
		t.Fatal("version crossed workspace:", err)
	}
}
func TestDispatchFencesInboxAndRetention(t *testing.T) {
	s, _, _ := testDB(t, true)
	p := seed(t, s, 1)
	accepted := accept(t, s, p, "dispatch")
	l := claim(t, s, p, "worker-one")
	if l.Mode != "send" || l.Epoch != 1 {
		t.Fatal(l)
	}
	if _, err := s.Claim(t.Context(), p.Workspace, "worker-two", time.Minute, audit()); !errors.Is(err, ErrNotFound) {
		t.Fatal("live lease stolen:", err)
	}
	if err := s.BeginDispatch(t.Context(), l, audit()); err != nil {
		t.Fatal(err)
	}
	if err := s.BeginDispatch(t.Context(), l, audit()); err == nil {
		t.Fatal("second send intent accepted")
	}
	execute(t, s, `UPDATE confignostic.dispatches SET lease_until=clock_timestamp()-interval '1 second',epoch=9007199254740993 WHERE workspace_id=$1`, p.Workspace.String())
	next := claim(t, s, p, "worker-two")
	if next.Mode != "reconcile" || next.Epoch != 9007199254740994 {
		t.Fatal("expired send was requeued or fence lost precision", next)
	}
	delivery := Delivery{Source: "provider-operation-1", ExternalID: "running", Change: domain.Transition{To: domain.Running, Evidence: domain.Reconciled}, Audit: audit()}
	if _, err := s.Ingest(t.Context(), l, delivery); !errors.Is(err, ErrLeaseLost) {
		t.Fatal("stale worker committed:", err)
	}
	if _, err := s.Renew(t.Context(), l, time.Minute); !errors.Is(err, ErrLeaseLost) {
		t.Fatal("expired worker renewed:", err)
	}
	if _, err := s.Renew(t.Context(), next, time.Minute); err != nil {
		t.Fatal(err)
	}
	// A malformed event must roll back its state change and allocated sequence.
	bad := delivery
	bad.Provider = &v1.ProviderReference{SessionID: "bad\nreference"}
	if _, err := s.Ingest(t.Context(), next, bad); !errors.Is(err, ErrInvalid) {
		t.Fatal("bad event accepted:", err)
	}
	a, err := s.Attempt(t.Context(), p.Workspace, accepted.Attempt)
	if err != nil || a.State != domain.Unknown {
		t.Fatal("invalid event partially committed", a, err)
	}
	sequence, err := s.Ingest(t.Context(), next, delivery)
	if err != nil || sequence != 3 {
		t.Fatal("ingestion sequence:", sequence, err)
	}
	if seq, err := s.Ingest(t.Context(), next, delivery); err != nil || seq != sequence {
		t.Fatal("replay duplicated event:", seq, err)
	}
	changed := delivery
	changed.Change.To = domain.Completed
	changed.Change.Evidence = domain.ProviderStopped
	if _, err := s.Ingest(t.Context(), next, changed); !errors.Is(err, ErrConflict) {
		t.Fatal("event identity reused with changed content:", err)
	}
	changed.ExternalID = "completed"
	if seq, err := s.Ingest(t.Context(), next, changed); err != nil || seq != 4 {
		t.Fatal(seq, err)
	}
	page, err := s.PendingEvents(t.Context(), p.Workspace, 0, 100)
	if err != nil || len(page.Events) != 4 {
		t.Fatal("outbox mismatch:", len(page.Events), err)
	}
	if err := s.Compact(t.Context(), p.Workspace, 1); !errors.Is(err, ErrConflict) {
		t.Fatal("unpublished events deleted:", err)
	}
	for _, event := range page.Events {
		if err := s.Published(t.Context(), p.Workspace, event.Sequence); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Compact(t.Context(), p.Workspace, 1); err != nil {
		t.Fatal(err)
	}
	page, err = s.Events(t.Context(), p.Workspace, 0, 100)
	if !errors.Is(err, ErrCursorExpired) || page.Snapshot == nil || page.Snapshot.Cursor != 4 || len(page.Snapshot.Attempts) != 1 || page.Snapshot.Attempts[0].State != domain.Completed {
		t.Fatal("expired cursor lost snapshot:", page, err)
	}
	page, err = s.Events(t.Context(), p.Workspace, 3, 100)
	if err != nil || len(page.Events) != 1 || page.Events[0].Sequence != 4 {
		t.Fatal("retained suffix:", page, err)
	}
	if seq, err := s.Ingest(t.Context(), next, delivery); err != nil || seq != 3 {
		t.Fatal("retention removed deduplication receipt:", seq, err)
	}
	a, err = s.Attempt(t.Context(), p.Workspace, accepted.Attempt)
	if err != nil || a.State != domain.Completed {
		t.Fatal("replay reopened a terminal attempt", err)
	}
	if _, err := s.Claim(t.Context(), p.Workspace, "worker-three", time.Minute, audit()); !errors.Is(err, ErrNotFound) {
		t.Fatal("terminal work dispatched:", err)
	}
}

func TestKilledProcessTransactionBoundaries(t *testing.T) {
	s, dsn, _ := testDB(t, true)
	p := seed(t, s, 1)
	for _, mode := range []string{"before", "after"} {
		t.Run(mode, func(t *testing.T) {
			in := submission(p, "crash-"+mode)
			encoded, _ := json.Marshal(in)
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestCrashHelper$")
			cmd.Env = append(os.Environ(), "CONFIGNOSTIC_CRASH_MODE="+mode, "CONFIGNOSTIC_CRASH_URL="+dsn, "CONFIGNOSTIC_CRASH_INPUT="+string(encoded))
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			scanner := bufio.NewScanner(stdout)
			if !scanner.Scan() {
				_ = cmd.Wait()
				t.Fatal("crash helper failed to reach transaction boundary")
			}
			var prior Accepted
			if err = json.Unmarshal(scanner.Bytes(), &prior); err != nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				t.Fatal(err)
			}
			if err = cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()
			actual, err := s.Submit(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "after" && actual != prior {
				t.Fatal("acknowledged work lost after process death")
			}
			if mode == "before" {
				if actual == prior {
					t.Fatal("uncommitted work survived")
				}
				if _, err = s.Attempt(ctx, p.Workspace, prior.Attempt); !errors.Is(err, ErrNotFound) {
					t.Fatal("partial attempt survived:", err)
				}
			}
		})
	}
}
func TestCrashHelper(t *testing.T) {
	mode := os.Getenv("CONFIGNOSTIC_CRASH_MODE")
	if mode == "" {
		return
	}
	ctx := context.Background()
	s, err := Open(ctx, os.Getenv("CONFIGNOSTIC_CRASH_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var in Submission
	if json.Unmarshal([]byte(os.Getenv("CONFIGNOSTIC_CRASH_INPUT")), &in) != nil {
		t.Fatal("bad fixture")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal("begin failed")
	}
	defer rollback(tx)
	accepted, err := s.submit(ctx, tx, in)
	if err != nil {
		t.Fatal(err)
	}
	if mode == "after" {
		if err = tx.Commit(ctx); err != nil {
			t.Fatal("commit failed")
		}
	}
	if err = json.NewEncoder(os.Stdout).Encode(accepted); err != nil {
		t.Fatal(err)
	}
	for {
		time.Sleep(time.Hour)
	}
}
func TestBackupRestore(t *testing.T) {
	tool := os.Getenv("CONFIGNOSTIC_TEST_PG_TOOLS")
	if tool == "" {
		t.Skip("set CONFIGNOSTIC_TEST_PG_TOOLS to the PostgreSQL tool wrapper")
	}
	tool, err := filepath.Abs(tool)
	if err != nil {
		t.Fatal(err)
	}
	s, _, name := testDB(t, true)
	p := seed(t, s, 1)
	a := accept(t, s, p, "backup")
	l := claim(t, s, p, "worker")
	if err = s.BeginDispatch(t.Context(), l, audit()); err != nil {
		t.Fatal(err)
	}
	restored, _, target := testDB(t, false)
	user := s.pool.Config().ConnConfig.User
	dump := exec.CommandContext(t.Context(), tool, "pg_dump", "-U", user, "-Fc", "-d", name)
	data, err := dump.Output()
	if err != nil {
		t.Fatal("pg_dump failed:", err)
	}
	restore := exec.CommandContext(t.Context(), tool, "pg_restore", "-U", user, "--no-owner", "--no-privileges", "-d", target)
	restore.Stdin = bytes.NewReader(data)
	if output, err := restore.CombinedOutput(); err != nil {
		t.Fatalf("pg_restore failed: %v: %s", err, output)
	}
	if err = restored.Ready(t.Context()); err != nil {
		t.Fatal(err)
	}
	if actual, err := restored.Submit(t.Context(), submission(p, "backup")); err != nil || actual != a {
		t.Fatal("restored idempotency:", err)
	}
	page, err := restored.PendingEvents(t.Context(), p.Workspace, 0, 100)
	if err != nil || len(page.Events) != 1 {
		t.Fatal("restored outbox:", err)
	}
	execute(t, restored, `UPDATE confignostic.dispatches SET lease_until=clock_timestamp()-interval '1 second'`)
	recovered := claim(t, restored, p, "restored-worker")
	if recovered.Mode != "reconcile" || recovered.Epoch <= l.Epoch {
		t.Fatal("restore allowed duplicate dispatch")
	}
}
