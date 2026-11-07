# Version 1 domain and wire contracts

Status: stage C accepted in [PR #6](https://github.com/UXCentury/confignostic/pull/6). The stateless workflow validator is operational. Stage D adds [internal durable storage](STORAGE.md); authentication, operational agent connections, runner transport and execution remain planned.

## Implemented HTTP surface

[OpenAPI](../api/openapi.json) describes the routes currently exposed by the Go API. Existing health and empty-list endpoints remain available. The new endpoint is:

```http
POST /api/v1/workflows/validate
Content-Type: application/json
```

From the repository root, while the API is running:

```sh
curl --fail-with-body http://127.0.0.1:8080/api/v1/workflows/validate \
  -H 'Content-Type: application/json' \
  --data-binary @internal/contracts/v1/testdata/workflow.json
```

A valid graph returns schema_version "1", a request_id, valid true, capabilities_checked false and executable false. It does not save anything, look up connections, authorize actions or run agents. This endpoint is safe to use for local schema/graph development; it is not a substitute for the authenticated submission path.

The validator rejects unsupported schema versions, unknown fields, case aliases, duplicate JSON keys, nulls, invalid IDs, excessive sizes/depth, cycles, unreachable nodes, invalid references, incompatible ports and missing required bindings. It does not evaluate conditions/transforms or validate future executor-specific settings.

Errors use the [error schema](../internal/contracts/v1/schema/error.schema.json) and X-Request-ID correlation header. HTTP statuses: 400 for malformed/schema/version errors; 405 for a wrong method; 413 for size limits; 415 for media type; 422 for graph semantics; 500 for unavailable built-in validation. Errors contain stable codes and bounded path/code issues, never copied request bodies or provider credentials.

The developer API remains unauthenticated and bound to loopback by default. Workspace IDs submitted to this stateless endpoint are structural data, not authorization claims.

## Identity and ownership

[Domain identities](../internal/domain/id.go) use resource-specific Go types and a wire prefix followed by 32 lowercase hexadecimal digits. New IDs use 128 random bits. Resource types cannot be interchanged by normal Go assignment or JSON decoding.

| Resource | Prefix | Ownership / identity rule |
| --- | --- | --- |
| Workspace | wsp_ | Root boundary; no resource ID itself proves access. |
| Project | prj_ | Belongs to one workspace. |
| Agent registration | agt_ | Catalog/runtime definition, separate from a configured connection. |
| Connection | con_ | Workspace-scoped configuration and discovered capabilities. |
| Conversation / message | cnv_ / msg_ | Messages belong to a conversation and its workspace. |
| Task / attempt / dispatch | tsk_ / atm_ / dsp_ | Logical work, execution attempt and delivery identity are distinct. |
| Event | evt_ | Stable occurrence identity plus a persisted workspace sequence. |
| Approval / artifact | apr_ / art_ | Scoped permission decision / immutable output reference. |
| Workflow / version / run | wfl_ / wfv_ / run_ | Definition, immutable revision and execution instance. |
| Runner | rnr_ | Bound to authenticated enrollment; JSON alone cannot enroll it. |
| Actor / request | act_ / req_ | Principal provenance and cross-operation correlation. |

Provider session/event/operation IDs remain opaque external references, not Confignostic IDs. Do not coerce an external ID into a task/attempt identity.

Actor kinds are human, planner, agent and system. A planner label never grants human approval authority. Stage E must derive the real actor and workspace from authenticated state; stage F supplies action authorization.

Counters, revisions, sequence numbers, lease epochs and artifact byte counts use canonical decimal **strings**, including values above JavaScript's safe integer range. Leading zeros, negatives, fractions and uint64 overflow are rejected. Individual contracts reject zero where a positive value is required.

## Attempt lifecycle and evidence

[Attempt.Apply](../internal/domain/attempt.go) returns a new value and enforces both an allowed state pair and the required evidence. It does not mutate or reopen terminal attempts.

- Record a dispatch ID before queued becomes dispatched. A dispatched attempt cannot return to queued.
- A local approval denial can cancel a task that has never been dispatched.
- After dispatch, completed/failed/canceled requires authoritative confirmation that execution stopped.
- A cancellation request records cancel_requested. Ordinary progress cannot silently retract it; returning to running requires explicit cancellation rejection.
- A timeout or lost response with possible external effects becomes unknown.
- An unknown attempt can resume or terminate only through reconciliation evidence.
- A retry requires explicit authorization, a different attempt ID and a confirmed failed/canceled predecessor. It retains the logical task ID and increments the attempt number. Unknown and completed attempts cannot be retried through this operation.

Evidence values are inputs from trusted policy/connector/reconciliation code. They are not bearer permissions and must never be accepted directly from a client/model as proof. Stage D [storage](STORAGE.md) applies transitions and persists events atomically under concurrency control; these domain functions do not provide storage or exactly-once external effects.

The state's dispatch marker is sticky for one attempt. A separate provider session reference does not replace it.

## Shared workflow graph

[Workflow schema](../internal/contracts/v1/schema/workflow.schema.json) defines the immutable version metadata plus the body used by chat and the future canvas. [The example](../internal/contracts/v1/testdata/workflow.json) represents implementation followed by independent review.

Nodes have stable graph-local names, kinds, typed input/output ports, and agent connection/capability requirements where applicable. Current kinds are agent, approval, condition, transform and join. Port types are text, json, artifact and boolean. Executor behavior belongs to N; kind/port validity alone does not establish executable semantics.

A data edge binds one output port to one input port of exactly the same type. Each input has at most one binding. Required inputs must be bound. The reserved source $input refers to a declared workflow input. A control edge omits both ports and represents a dependency; it cannot originate at $input.

Entry nodes must exist and have no incoming node dependencies. Every node must be reachable from a declared entry. Graphs may have several explicit parallel entries. Cycles, self-edges, duplicate nodes/edges, mixed missing ports and unbound required inputs are rejected.

The registry capability check is separate from structural validation. It requires a descriptor for the same workspace and exact connection, always requires invoke for agent nodes, and checks declared requirements plus artifact capability for artifact inputs/outputs and structured-output capability for JSON outputs. Imported/client-supplied descriptors cannot certify themselves. No operational provider connections exist yet, so the HTTP endpoint always reports capabilities_checked false.

## Revision conflicts and immutability

The [Draft reference aggregate](../internal/workflows/draft.go) snapshots nested collections, serializes revision checks, generates a fresh version ID on accepted edits, and never mutates previously returned definitions. Two concurrent edits based on revision 1 yield exactly one accepted revision 2 and one ErrRevisionConflict. Invalid edits do not advance the revision.

This is an in-memory reference implementation of the contract, deliberately not API storage. Stage D performs the equivalent compare-and-swap in PostgreSQL and persists immutable version records in the same transaction; see [Storage](STORAGE.md). The later edit API will carry an expected_revision decimal string and return 409 revision_conflict for a stale edit. It must not reread and overwrite silently.

## Events and browser replay

[Event schema](../internal/contracts/v1/schema/event.schema.json) supports task.state_changed, task.output, approval.requested and artifact.created. Payloads are discriminated by type and reject unknown fields. Events carry workspace/task/attempt IDs, occurrence ID, positive sequence, UTC timestamp, actor, correlation ID and optional opaque provider reference.

State notifications cannot describe impossible state pairs or reopen a terminal attempt. A notification is not authority to perform the transition; the task aggregate must have already validated evidence. Approval requests bind action digest, policy version, scope and expiry. Artifact metadata carries an ID, SHA-256 digest, media type and bounded byte size. No file path or download URL is trusted from this event.

The SSE encoder emits id, event and one JSON data line; embedded newlines remain escaped. ParseCursor accepts an absent cursor or a canonical decimal Last-Event-ID. It does not establish retention or persistence.

The planned event endpoint must authorize its workspace, replay committed events after the cursor, deduplicate by occurrence identity, and provide a snapshot/reset path on 409 cursor_expired. Sequence allocation/publication must not skip a committed event because an earlier transaction is still pending. No live event endpoint is registered yet.

## Runner compatibility and dispatch

[Hello](../internal/contracts/v1/schema/hello.schema.json), [dispatch](../internal/contracts/v1/schema/dispatch.schema.json) and [acknowledgment](../internal/contracts/v1/schema/ack.schema.json) are typed contracts; no runner transport is implemented.

Negotiation requires the runner/workspace from an already authenticated enrollment, a common protocol version, and a bounded negotiated message size. Version 1 is currently the only selectable protocol. Frames declare their type explicitly: runner.hello, runner.welcome, task.dispatch, task.ack, or a normalized event type. An unavailable version or wrong enrollment scope fails.

Dispatch carries separate task/attempt/dispatch IDs, connection ID, lease epoch, correlation, actor, idempotency key, policy version, approval references and bounded message/artifact references. Workflow runs must also pin their workflow version and node together. Credentials are not a field in this envelope.

An accepted acknowledgment means the runner has durably recorded the dispatch identity; it does not mean provider acceptance or task completion. A rejected acknowledgment is an admission failure and requires an error. An uncertain provider outcome belongs to an accepted attempt's event/reconciliation history. Acknowledgments must match runner, workspace, dispatch, attempt and lease epoch; a stale epoch is rejected.

Stage D implements database intake, replay and lease fencing. Authentication, runner-side fencing and action authorization remain E/F/G work. Structural envelope validation cannot supply those guarantees.

## Limits and compatibility policy

| Boundary | V1 limit |
| --- | --- |
| Workflow HTTP JSON | 256 KiB |
| Event/runner JSON envelope | 64 KiB; a negotiated runner limit may be smaller |
| Message/output text | 32 KiB UTF-8, also subject to envelope size |
| Artifact metadata size | 64 MiB; upload/download enforcement arrives in M |
| JSON nesting | 32 levels |
| Workflow | 100 nodes, 500 edges; 32 workflow inputs, and 32 inputs/32 outputs per node |
| Declared capabilities / approval/artifact references | 16 each |
| Public graph diagnostics | 20 path/code issues |

Both incoming JSON and its serialized representation must fit the envelope limit. Escaping can increase serialized size. Numeric JSON literals are bounded unsigned integer values; large counters use strings. Optional values are omitted rather than null. Unknown properties are rejected, including case variants that legacy JSON decoders might otherwise accept.

JSON Schema Draft 2020-12 describes structural constraints. Go additionally checks byte limits, uint64 bounds, graph semantics, scope and operation-specific rules. Schemas are embedded and compiled from local resources; the loader never fetches arbitrary URLs.

Version 1 is explicit on every envelope. Schema/runtime changes require compatibility review; breaking changes use a new version rather than silently accepting unknown fields. Pin tested schema/runtime combinations in connector records when real integrations exist.

Error retry dispositions distinguish never, backoff and reconcile. Unavailable/rate-limited responses may advise backoff; uncertain outcomes require reconciliation. This is not permission to blindly repeat a side effect. The later durable submission path must use the same scoped idempotency key, reject payload changes under a reused key, and reconcile accepted external work.

## Planned HTTP operations

These routes are contracts for later stages, **not registered endpoints**:

| Operation | Request/result contract | First implementation stage |
| --- | --- | --- |
| POST /api/v1/workflows; PUT /api/v1/workflows/{id} | Version 1 body; server-derived workspace/IDs; edit carries expected_revision; returns an immutable version or 409 conflict. | D/N |
| POST /api/v1/conversations/{id}/messages | Connection, bounded message and artifact refs; scoped idempotency key; returns stable message/task IDs after durable commit. | D/I/J |
| GET /api/v1/tasks/{id} | Workspace-authorized task, attempts and events; explicit unknown/cancel_requested outcomes. | D/J/K |
| POST /api/v1/tasks/{id}/cancel or /retry | Scoped action authorization; cancel is a request, retry creates a new authorized attempt. | F/K |
| GET /api/v1/workspaces/{id}/events | Authorized SSE with decimal replay cursor, retention/reset handling and bounded buffering. | D/J |
| POST /api/v1/runners/enroll; runner connection route finalized in G | Authenticated scope, negotiated hello, dispatch/ack correlation and fencing. | G |

No write endpoint should be added before the required persistence and authorization exist. The API/fixture validation delivered here must not be presented as a working chat or provider integration.

## Evidence

Local checks cover resource-kind separation, precision above 2^53, lifecycle guards, forbidden retries, schema round trips, duplicate/unknown/deep JSON, size limits, incompatible capabilities, stale acknowledgments, SSE injection, concurrent edits and stateless HTTP behavior. The parser also has a fuzz target whose successful decodes must round-trip.

Run make check-go and make check-reference. A bounded additional parser check is:

```sh
go test ./internal/contracts/v1 -run '^$' -fuzz '^FuzzDecodeWorkflow$' -fuzztime=5s -parallel=2
```

GitHub checks and their exact source commit are recorded in the stage C PR. All 21 agent integrations remain Planned.
