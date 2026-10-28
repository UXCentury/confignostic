# PostgreSQL storage foundation

Stage D implements internal PostgreSQL repositories, forward-only migrations and database readiness. Public task/workflow write routes, authentication, runner transport and agent execution remain planned. A registered database record does not certify a working agent connection.

## Run and migrate

Use the pinned PostgreSQL 18 Compose service described in [Contributing](../CONTRIBUTING.md). Set its disposable password in .env, then:

```sh
make db-up
export CONFIGNOSTIC_DATABASE_URL='postgres://confignostic:URL_ENCODED_LOCAL_PASSWORD@127.0.0.1:5432/confignostic_dev?sslmode=disable'
make db-migrate
make dev-api
```

Replace the password placeholder with the URL-encoded development password. Go reads process environment, not .env. Keep this local development service on loopback. The remote-production TLS and credential configuration belongs to the deployment qualification in V.

The migration command is also available as `bin/confignostic-db migrate` after `make check-go` or `make build`. Migrations take a transaction-scoped advisory lock and commit their SQL and checksum ledger together. Concurrent migration processes serialize. A missing, modified or newer migration history fails closed. There is no automatic migration at API startup and no destructive down migration.

With a database URL configured, the API checks connectivity, write mode and schema compatibility before opening its listener. `/healthz` remains process liveness. `/readyz` returns 200 only when storage is connected, writable and compatible, and 503 otherwise. With no database URL, the stateless development scaffold still starts but readiness reports `storage_not_configured`. Readiness responses and startup failures do not return connection strings or driver error details.

## Ownership and durable acceptance

Every repository operation requires a typed workspace identity. Composite foreign keys keep projects, agent registrations, connections, conversations, messages, tasks, attempts, workflows, approvals, runners and artifact references within that workspace. These are data boundaries; authenticated membership and action authorization are stage E/F work. There is no public provisioning or submission endpoint yet.

`Provision` atomically creates an initial workspace/project/agent/connection/conversation record set for trusted bootstrap code and tests. It does not discover a provider or authorize execution. `Submit` validates a bounded message and artifact references, then commits the message, logical task, first attempt, request idempotency receipt and dispatch intent together. Callers acknowledge acceptance only when it returns no error.

Submission keys are unique within a workspace. Repeating the same key and canonical operation returns the original identities; changing the payload under that key returns a conflict. A new correlation ID does not create new work. Artifact references must exist in the same workspace; duplicate references are rejected. Attempt numbers and message/task ownership also have database uniqueness constraints.

A network failure during COMMIT returns `ErrCommitUnknown`: the caller must reconcile by repeating the same idempotency identity or looking up the original record. It must not substitute a new key. PostgreSQL's normal durable commit settings remain enabled; the application does not disable fsync or synchronous commit.

## Events and dispatch leases

State transitions use the stage C evidence guards. Each accepted transition updates the attempt, allocates a workspace event sequence, stores the versioned event and inserts its outbox row in one transaction. Workspace row locking makes sequence allocation follow commit order. The initial queued submission is represented by its durable task and dispatch intent; it does not invent a state transition from an unspecified state.

`Claim` returns a dispatch identity, monotonically increasing uint64 fencing epoch, owner and database-clock deadline. Lease durations are bounded from one second to five minutes. A worker must commit `BeginDispatch` before sending any external request. That operation records the send intent and dispatched event together.

After lease expiry:

- An intent with no recorded send can be claimed for sending.
- A previously sent intent is claimed in `reconcile` mode. Its active attempt becomes unknown if necessary. It is never returned as fresh work to resend.
- Completed, failed or canceled attempts cannot be claimed.

Every authoritative ingestion validates the workspace, dispatch, attempt, owner, epoch and unexpired deadline inside the transaction, and checks the fence again before commit. Renewal cannot revive an expired/superseded lease. The workspace lock prevents a replacement lease from committing concurrently with the old owner's transition. Runner-side fencing/deduplication and provider reconciliation still belong to G/K; the database alone cannot guarantee exactly-once external effects.

`Ingest` currently ingests normalized **state-change** deliveries. Its inbox identity is workspace + connection + stable source session/operation/stream + external event ID. Callers must preserve that source scope across reconnects. Repeated content returns the committed sequence; changed content under an existing receipt conflicts. Inbox receipts outlive event compaction so replay cannot reopen a terminal attempt. Output, artifact upload and approval execution paths are added by their later feature stages.

`PendingEvents` reads committed outbox events. `Published` marks an event only after durable downstream acceptance. Publication is at-least-once; consumers deduplicate stable event IDs. The transport/publisher worker is not implemented here.

## Workflow versions and replay retention

Workflow creation requires revision 1 and full v1 validation. Revision updates compare the expected revision in a database transaction, insert a new immutable version and advance the current revision together. Concurrent editors cannot both overwrite the same revision. Old versions remain readable by version identity within their workspace.

Event reads use a committed workspace sequence cursor and bounded pages of at most 1,000 events. Future cursors are rejected. `Compact` keeps at least the requested 1–10,000 recent events, deletes at most 1,000 published events per call, and refuses to remove unpublished outbox entries. Compaction is explicit; there is no background retention scheduler yet.

The same transaction stores a consistent snapshot of the latest attempt state for each task and advances the retained floor. An expired cursor returns `ErrCursorExpired` with that snapshot; clients reset state and resume after its cursor. This is a task-state snapshot, not an archive of message/output content. Snapshot creation is limited to 1,000 tasks and 1 MiB and fails without pruning if either limit is exceeded. Larger-workspace pagination/archival is later operations work. Idempotency/inbox receipts are retained; an explicit receipt-expiry policy requires a documented replay horizon before deletion can be introduced.

Workspace writes currently serialize on the workspace row. This deliberately simple concurrency model establishes correctness; throughput qualification and more granular locking remain W work.

## Verification and recovery

The database integration suite requires a disposable PostgreSQL server whose test role can create databases. Each test creates and removes its own randomly named database; it does not truncate the configured base database.

```sh
export CONFIGNOSTIC_TEST_DATABASE_URL="$CONFIGNOSTIC_DATABASE_URL"
export CONFIGNOSTIC_TEST_PG_TOOLS="$PWD/scripts/postgres-tools"
make check-storage
```

The wrapper invokes pg_dump/pg_restore inside the pinned Compose container, keeping client/server versions aligned. The test target requires both variables, so a missing database or backup-tool configuration cannot silently count as a passed integration run. Ordinary `make check-go` covers compilation and non-database tests; it does not claim PostgreSQL durability coverage.

The suite covers fresh install, version 1→2 upgrade, concurrent migrations, checksum drift, concurrent submissions and workflow edits, workspace/reference isolation, rollback on invalid events, stale/expired leases, replay after compaction, real child-process kills before/after COMMIT, and pg_dump/pg_restore of accepted tasks, idempotency receipts and pending events. CI runs it against the pinned PostgreSQL 18 image. Results and the exact source commit are recorded in the stage D PR.

Operational restore must quiesce API/worker writers and disconnect old runners before restoring a database. A backup can roll back lease epochs and provider receipt history; restore is not a license to replay external work. Re-enroll/fence old runners and reconcile outstanding provider operations before execution resumes. The integration restore test verifies persisted state and reconciliation mode; full operator recovery drills are still part of V/W.

Next is stage E: authenticated bootstrap, sessions, workspace membership and authorization before exposing database write operations to users.
