# Release quality gates

Prepared: 2026-10-24  
Status: proposed acceptance criteria; no gate is claimed to have passed.

## Meaning of S+

For this project, S+ means every applicable mandatory gate below has reproducible evidence tied to the release commit. It is an internal engineering target, not a certification or a substitute for independent review.

A feature can be Planned, Experimental or Supported independently of another feature. A release must name its scope. The complete product milestone requires every promised feature and all 21 named targets to qualify. An earlier alpha/beta must publish the smaller verified subset.

Tests and reviews start in their implementation stages. Stage W assembles and rechecks release evidence; it does not defer security, reliability or usability until the end.

## Mandatory self-hosted gates

| Gate | Pass condition | Minimum evidence |
| --- | --- | --- |
| G01 — reproducible build | Clean checkout produces the API, frontend and runner using documented pinned toolchains and locked dependencies. No undocumented local files are required. | Clean CI logs and artifact manifest. |
| G02 — task correctness | A submitted task has one stable internal identity, real provider execution, explicit state transitions and traceable output. No simulated execution is presented as real. | Actual Codex/independent-provider fixture runs and transition tests. |
| G03 — durable recovery | Accepted messages/tasks/events survive restart. Lost acknowledgments and replayed events do not cause blind duplicate side effects. Unknown outcomes stop dependent work until reconciled. | Boundary-by-boundary failure-injection histories. |
| G04 — approvals | A denial prevents the scoped action. Approvals expire, cannot be replayed for altered inputs, and cannot be granted by a planner acting as the human reviewer. | Policy tests plus real runtime denial/input-request checks. |
| G05 — access control | Every HTTP route, event subscription, artifact read and background operation enforces workspace permissions. Role revocation blocks subsequent operations. | Cross-workspace and role-matrix tests, including guessed IDs and stale sessions. |
| G06 — secret protection | Stored synthetic secrets are absent from returned UI/API payloads, application logs, event history and exports; credential rotation/revocation works. Secret-loss and backup recovery are documented. | Canary scan results and key/credential lifecycle tests. |
| G07 — runner isolation | Tasks cannot access another workspace, control-plane credentials or unauthorized mounts/networks. Revoked runners receive no new tasks; stale attempts cannot finalize newer work. | Sandbox, enrollment, fencing and process-cleanup tests. |
| G08 — workflow semantics | Chat and canvas use the same versioned schema/executor. Validation rejects invalid cycles, missing inputs and unsupported capabilities. Branches/joins and bounded concurrency behave correctly under replay. | Schema fixtures, scheduler tests and chat/canvas round-trip runs. |
| G09 — artifact integrity | A handoff references the verified output/base version; downloads are scoped, sizes/types bounded and previews sanitized. Parallel repository edits remain isolated. | Digest/provenance checks, unauthorized download tests and conflict fixture. |
| G10 — connector honesty | Every Supported connector has real execution evidence on recorded upstream versions, with optional capabilities tested or declared unavailable. | Per-provider compatibility record for each advertised connector. |
| G11 — usable interface | Critical chat, approval, connection and canvas journeys work with keyboard navigation, responsive layouts and explicit error/offline states. Accessibility target is WCAG 2.2 AA for shipped surfaces. | Automated checks plus manual keyboard/screen-reader and responsive review. |
| G12 — observable operation | Operators can follow a task through API, dispatch, runner, provider and artifact IDs without needing secret values. Stuck/unknown tasks and provider outages are diagnosable. | Trace examples, metrics and runbook walkthrough. |
| G13 — installation and recovery | A fresh operator installs, runs a real task, upgrades, exports, backs up and restores the deployment using published docs. Data/key prerequisites are explicit. | Clean-host exercise and verified restored workflow/artifacts. |
| G14 — supply and upgrade integrity | Release dependencies and artifacts are inventoried; images/binaries have verifiable provenance/digests; migrations and compatible rollback/recovery paths are exercised. | Build provenance, dependency inventory and upgrade/restore report. |
| G15 — automation and collaboration | Schedule occurrences and webhook events deduplicate correctly; timezone/DST behavior is documented; shared resources and approvals enforce roles. | Replay/DST tests and multi-user workflow evidence. |

## Reference performance budgets

These are initial engineering targets to validate and revise transparently, not measured results or commercial promises. Publish hardware, versions, dataset sizes and load-generator configuration with every result.

Reference profile: Linux x86-64, 4 vCPU, 8 GiB RAM, SSD storage; API and PostgreSQL on the reference host, with the load generator on a separate host. Use a documented deterministic provider simulator for control-plane load. Run real-provider correctness tests separately; simulated throughput does not establish provider rate limits or cost.

| Measurement | Initial target | Measurement boundary |
| --- | --- | --- |
| Ordinary API reads/writes | p95 ≤ 300 ms and unexpected 5xx < 0.1% over a 30-minute steady run | 20 HTTP requests/second, 50 active simulated tasks, 200 event-stream clients; exclude artifact uploads and external provider/model latency. |
| Eligible dispatch delay | p95 ≤ 2 seconds | From committed runnable state to dispatch on an available runner; exclude approval waits, scheduled delays and declared quota backpressure. |
| Event display latency | p95 ≤ 1 second | From event ingestion to browser render on the reference network; exclude time the provider spent producing the event. |
| Browser reconnect | Resume within 5 seconds after connectivity returns | No missing persisted events; duplicate delivery must not duplicate visible logical events. |
| Initial usable chat UI | ≤ 3 seconds in the documented desktop lab profile | Controlled network/cache conditions, excluding provider authentication. Publish the profile with results. |
| Large history and graph | 10,000 stored messages with paginated/virtualized access; 100-node editable graph | No unbounded fetch/render, broken keyboard navigation or loss of edits. |
| Resource stability | No unbounded memory, event-buffer or queue growth under the reference run | Record memory/queue series during load and after drain; document explicit queue limits and admission errors. |
| Recovery integrity | Zero loss of acknowledged task transitions/events in tested API/runner crash scenarios | Database availability remains intact; test database restore separately against the documented backup policy. |

For external-provider runs, measure time to first event, end-to-end duration, failure rate and cost only where available. Display provider latency separately from Confignostic overhead. Pre-dispatch spending limits do not guarantee an exact final invoice for an already-running external agent.

## Required failure scenarios

| Scenario | Required behavior |
| --- | --- |
| API dies before accepting a request | Client can retry with its idempotency key and obtain one logical task. |
| API dies after commit but before dispatch | The outbox eventually dispatches the stored task. |
| Provider accepts work but the acknowledgment is lost | Reconcile using a known external ID/idempotency mechanism; otherwise mark unknown and require resolution. |
| Runner disappears during execution | Preserve last known state, attempt/session identity and partial artifacts; do not report success or confirmed cancellation. |
| Stale runner reconnects after lease reassignment | Fencing rejects stale authoritative updates and duplicate dispatch acceptance. |
| Browser reconnects with an expired event cursor | Return an explicit snapshot/reset path and resume from a valid cursor. |
| Credential expires mid-run | Surface the error without exposing the token or switching to another user's credentials. |
| User denies or lets an approval expire | The action is not executed; the run shows the decision and next permitted path. |
| Prompt or imported graph attempts to bypass policy | Model-generated text cannot change authorization or secret scope. |
| Agent output contains executable HTML or a hostile download URL | Preview/download policy preserves origin and workspace boundaries. |
| A2A discovery/redirect points at cloud metadata or an unapproved private service | Egress policy rejects the request; an explicit scoped private-runner allowlist is required for legitimate private services. |
| Concurrent agents modify the same file | Isolated workspaces preserve both outputs; merge/conflict handling is explicit. |
| User repeats cancel, retry or approve | The action is idempotent at the control plane and respects the provider's actual semantics. |
| Provider reports completion after a timeout | Reconcile the existing attempt with provenance; do not silently create a second success or discard a known external result. |
| Storage becomes unavailable/full | Stop acknowledging non-durable mutations, apply bounded backpressure and surface readiness/operation errors. |
| Webhook or billing event is replayed | A stored event identity prevents duplicate logical execution or ledger entries. |
| Schema migration fails | Deployment stops safely and follows the documented compatible rollback or restore path. |

## Connector qualification record

Create one record per adapter/version combination when implementation exists. This template does not certify any current connector.

- Connector ID, agent/product name, upstream version, protocol/SDK version and last verification date.
- Execution location: local runner, customer remote service or provider cloud.
- Authentication requirements and setup steps, with secret values omitted.
- Supported operations: invoke, streaming, sessions, approvals/input, cancellation, resume, artifacts and usage/cost.
- Capability limitations, known defects and rate/concurrency constraints.
- Real task ID, sanitized trace, output artifact hash and actual provider session/reference.
- Denial/error and restart/reconnect results; optional operations tested or explicitly unsupported.
- Reviewer, source commit and scope of the resulting Experimental/Supported status.

Discovery alone, a model response from a different runtime, a mock server and a provider's marketing claim are not sufficient evidence of an implemented agent integration.

## Additional cloud gates

| Gate | Pass condition | Evidence |
| --- | --- | --- |
| C01 — tenant separation | Tenant identity is enforced in API/storage/events/artifacts/runner enrollment/secret access, including operator tools. | Multi-tenant adversarial tests and workload-isolation review. |
| C02 — metering and billing | Signed events, deduplicated ledger entries, reconciliation and subscription-state transitions work under replay/out-of-order delivery. Hosting charges and provider usage are distinguishable. | Sandbox billing tests and ledger reconciliation report. |
| C03 — availability | Proposed service target: 99.9% monthly control-plane availability, measured separately from provider availability. Commercial commitments require operating evidence and an explicit policy. | Synthetic checks, incident accounting and a pilot observation window; no claim of an SLA from unit tests. |
| C04 — disaster recovery | Proposed initial targets: RPO ≤ 15 minutes and RTO ≤ 4 hours for the documented cloud topology, including artifacts and key material. | Timed restore drill and recovered-data verification. |
| C05 — bounded operations | Tenant queues, storage and execution have enforced limits; overload is visible and one tenant cannot exhaust all capacity. | Noisy-neighbor/load tests and quota runbooks. |
| C06 — accountable support | Operator access is scoped and audited; export/deletion work; support coverage matches actual staffing. | Access audit, deletion/export exercise and published operating procedures. |
| C07 — product value | Pilot evidence records repeated useful workflows, onboarding friction and concrete willingness to pay. | Pilot decision report; technical readiness alone does not establish demand. |

## Evidence and release decision

Use a qualification report for each release candidate. Record the commit, scope, environment, check results, artifacts, failures, owner and follow-up. Store sanitized evidence in CI artifacts or a controlled evidence location; retain hashes/links in the report. Never commit live credentials, customer prompts or unredacted private repositories.

A stage may be Blocked when credentials, infrastructure or reviewer availability prevent verification. Record the reason and the next action; do not mark it Done by substituting screenshots or fake outputs.

Release blockers include known unauthorized access/action, exposed credentials, reproducible loss of acknowledged state, incorrect completion reporting, broken clean installation, or a promised critical journey that fails. Other defects require an explicit severity and release decision with a documented limitation. Close or fix release-blocking findings before claiming the corresponding gate passes.

The complete self-hosted release uses G01–G15 and the stated scope of the performance/failure tests. Managed cloud additionally uses C01–C07. Self-hosted stable can ship before cloud stable.

## Primary references

Provider-specific evidence starts from the [supported-agent catalog](../SUPPORTED_AGENTS.md). Protocol and review references:

- [ACP transport specification](https://agentclientprotocol.com/protocol/v1/transports).
- [A2A specification](https://a2a-protocol.org/latest/specification/).
- [WCAG 2.2](https://www.w3.org/TR/WCAG22/).
- [OWASP SSRF prevention guidance](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html).

Record the exact protocol/version used in an implementation; a moving documentation URL is not a runtime compatibility pin.
