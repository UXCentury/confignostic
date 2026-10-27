# Confignostic: A–Z development plan

Plan version: 1.0  
Prepared: 2026-10-24  
Baseline: repository commit 4959695403a3d287366615b6f79fb114af81c6e5

## Product contract

Confignostic is an open-source, self-hosted chat workspace that connects and orchestrates existing independent agents. Users can address an agent directly, request a coordinated plan, edit the same plan on a visual canvas, approve actions, and inspect durable results across providers.

The full self-hosted core includes chat, connections, orchestration, canvas, artifacts, approvals, templates, scheduling and workspace collaboration. A later managed cloud service sells hosting, operations and support. Model/provider usage retains its own account and billing requirements.

The catalog covers 21 named agent targets plus generic ACP, A2A and custom adapters; MCP and X are tool integrations. See [Supported agents](../SUPPORTED_AGENTS.md) for source documentation and current implementation status.

## Starting point and interpretation

The baseline contains a Go HTTP scaffold, Svelte UI shell, documentation and backend CI. It has no operational agent integration, chat execution, persistence or workflow engine. Stages A and B are Done after [PR #5](https://github.com/UXCentury/confignostic/pull/5). Stage C is implemented and In review; see [Contracts](../CONTRACTS.md). Stages D–Z remain Planned. Existing scaffolding does not satisfy a stage's exit gate.

“S+” is our internal quality target: demonstrated correctness, reliability, security, usability and operability against [measurable gates](QUALITY_GATES.md). It is not a claim that the current repository is production ready or an external certification.

The architecture and file/module names in this plan describe the intended implementation. The [architecture](../ARCHITECTURE.md) distinguishes that target from current code.

## How to execute the plan

1. Read [Execution guide](EXECUTION_GUIDE.md), then choose the earliest unblocked task.
2. Implement a small reviewable change, including the relevant failure behavior.
3. Run checks appropriate to the change and attach evidence to the task/PR.
4. Advance a stage only when its exit gate passes. Provider access that cannot be verified remains a visible blocker.
5. Keep the catalog status independent for each connector. Protocol support alone does not certify every listed agent.

Letters provide navigation; **dependencies determine execution order**. Q, R, S and U can progress independently after their prerequisites when capacity allows. For a solo developer, finish one runtime feature at a time. Core tests and security controls are built alongside each stage; W is final qualification, not the start of testing or security work.

## Stage dependency map

| Stage | Outcome | Prerequisites | Status |
| --- | --- | --- | --- |
| A | [Lock the product contract](#stage-a) | None | Done |
| B | [Make development and CI reproducible](#stage-b) | A | Done |
| C | [Define versioned domain and wire contracts](#stage-c) | A, B | In review |
| D | [Build durable storage and dispatch foundations](#stage-d) | C | Planned |
| E | [Establish identity and workspace boundaries](#stage-e) | C, D | Planned |
| F | [Protect credentials and authorize actions](#stage-f) | E | Planned |
| G | [Create isolated local and private runners](#stage-g) | C, D, E, F | Planned |
| H | [Build the agent registry and adapter contract](#stage-h) | C, F, G | Planned |
| I | [Integrate the first real agent: Codex](#stage-i) | G, H | Planned |
| J | [Deliver functional unified chat](#stage-j) | D, E, F, I | Planned |
| K | [Prove recovery and execution correctness](#stage-k) | D, G, I, J | Planned |
| L | [Add ACP and the second provider: Cursor](#stage-l) | H, K | Planned |
| M | [Implement artifacts and typed agent handoffs](#stage-m) | K, L | Planned |
| N | [Build the deterministic workflow engine](#stage-n) | C, D, K, M | Planned |
| O | [Let chat propose and route workflows](#stage-o) | F, H, J, N | Planned |
| P | [Build the shared visual workflow canvas](#stage-p) | C, J, N, O | Planned |
| Q | [Expand the core coding-agent connectors](#stage-q) | H, K, M | Planned |
| R | [Connect remote and stateful agent services](#stage-r) | H, K, M, N | Planned |
| S | [Qualify the broader ACP ecosystem](#stage-s) | H, K, L | Planned |
| T | [Add tools, X integration and Grok research agents](#stage-t) | F, H, K, N, Q | Planned |
| U | [Add automation, collaboration and reusable workflows](#stage-u) | E, F, K, N, O | Planned |
| V | [Package and operate the self-hosted product](#stage-v) | E, F, G, K, P | Planned |
| W | [Qualify the complete self-hosted release](#stage-w) | P, Q, R, S, T, U, V | Planned |
| X | [Build the optional managed cloud service](#stage-x) | W | Planned |
| Y | [Validate cloud operations and paid pilots](#stage-y) | W, X | Planned |
| Z | [Release and maintain the product](#stage-z) | W | Planned |

## Release checkpoints

| Checkpoint | Required work | Demonstration |
| --- | --- | --- |
| R0 — executable specification | A–C | Product contract, reproducible development and reviewed schemas. |
| R1 — one-agent private alpha | A–K | Self-hosted chat completes a real Codex task with approval, history and restart recovery. |
| R2 — cross-agent execution | L–O after R1 | Cursor and Codex exchange a typed artifact through a durable workflow proposed in chat. |
| R3 — visual orchestration beta | P, plus an initial V package | Chat and canvas manipulate one workflow model; another person can install it. |
| R4 — complete self-hosted release candidate | Q–V, after R3 | All 21 targets have qualification evidence or explicit blockers; tools, automation and collaboration work. |
| R5 — self-hosted stable | W and Z-OSS | All promised v1 features and supported integrations meet mandatory release gates. |
| R6 — managed cloud pilot | X, then Y | Tenant isolation, usage/billing and operations verified with pilot users. |
| R7 — managed cloud stable | Y and Z-CLOUD | Paid-pilot and service-readiness evidence support general availability. |

R1/R2/R3 are incremental validation checkpoints for the full product. They do not remove any target from the plan. A narrower early release can advertise only its verified subset. Calling the complete 21-target scope finished requires evidence for all 21.

## Reference end-to-end workflow

On a clean self-hosted installation, an operator connects Codex and Cursor to an authorized disposable repository. In one chat, the operator asks for a bounded implementation and independent review. The system proposes a versioned graph, obtains required approval, runs the implementation in an isolated workspace, stores a typed artifact, and hands that exact version to the reviewer.

During the run, restart the API and interrupt the runner connection. Reconnect without blindly repeating accepted external work. The user sees progress, approvals, failure/unknown states, the final diff, review findings and usage when provided. Opening the canvas shows the same graph and persisted events. A different workspace cannot read the conversation, credentials or artifacts.

## Detailed implementation stages

<a id="stage-a"></a>

### A. Lock the product contract

**Prerequisites:** None.  
**Outcome:** Make the full product and the first useful workflow unambiguous.  
**Deliverables:** Product acceptance brief, reference project, scope decisions and initial user-feedback questions.

- [x] **A01** — Document the primary journey: self-host, create a workspace, connect an agent, send a task, inspect real progress, approve an action, and receive a persistent result.
- [x] **A02** — Define the reference workflow: Codex implements a bounded change in a test repository, Cursor reviews the resulting artifact, and the user receives both results in one conversation.
- [x] **A03** — Record all 21 catalog targets, the three protocol/adapter routes, MCP and X tools, and the open-source/cloud boundary. Keep local chat, connectors, orchestration, canvas and collaboration self-hostable.
- [x] **A04** — Record assumptions with a named owner: initial Linux/Docker support, access to provider accounts, private-runner execution, and the early developer-team audience. Collect feedback alongside implementation.

**Exit gate:** A reviewer can trace every requested product capability to a stage below. The reference workflow has a deterministic fixture and explicit success/failure outcomes; research assumptions are visible.

**Evidence:** [Product acceptance brief](../PRODUCT.md), [reference-fixture walkthrough](../../testdata/handoff/README.md), and [foundation validation](../FOUNDATION_STATUS.md). The full agent workflow remains planned.

<a id="stage-b"></a>

### B. Make development and CI reproducible

**Prerequisites:** A.  
**Outcome:** Make each subsequent change easy to build, verify and review.  
**Deliverables:** Pinned toolchain policy, frontend lockfile, developer commands, CI jobs and repository conventions.

- [x] **B01** — Pin compatible Go/Node/package-manager versions, commit the frontend dependency lockfile, and document clean-checkout setup. Add a development PostgreSQL service with disposable test data.
- [x] **B02** — Run Go formatting checks, vet, tests and build in CI; add frontend type/Svelte checks and production build. Add UI tests only for meaningful behaviors as those behaviors arrive.
- [x] **B03** — Define configuration parsing, structured error/log conventions and the documentation/evidence layout. Make health and readiness separate when dependencies are introduced.
- [x] **B04** — Fix the reviewed server shutdown issue so main waits for shutdown completion. Verify a controlled in-flight request drains; keep CI credentials away from untrusted pull-request code.

**Exit gate:** A clean checkout builds through documented commands and CI. A type error fails the frontend job; an in-flight request completes during graceful shutdown.

**Evidence:** [Development commands/toolchain policy](../../CONTRIBUTING.md), real-HTTP lifecycle regression tests, and [foundation validation](../FOUNDATION_STATUS.md).

<a id="stage-c"></a>

### C. Define versioned domain and wire contracts

**Prerequisites:** A, B.  
**Outcome:** Give chat, runners and the canvas one common representation of work.  
**Deliverables:** Domain types, API/schema specifications, event envelope and workflow schema v1.

- [ ] **C01** — Specify workspace, project, agent registration, connection, conversation, message, task, attempt, event, approval, artifact and workflow-version identities. Provider session IDs remain separate from internal IDs.
- [ ] **C02** — Define task transitions, actor identities, correlation IDs, sequence numbers and structured errors. Distinguish unavailable, unsupported, unauthorized, retryable and uncertain outcomes.
- [ ] **C03** — Specify versioned HTTP APIs and browser event streams; define the runner envelope, acknowledgments and compatibility handshake. Bound request, event and artifact sizes.
- [ ] **C04** — Define immutable workflow versions with typed node inputs/outputs, dependency edges, capability requirements and an explicit schema version. Reject unsupported versions rather than silently changing meaning.

**Exit gate:** Contract examples round-trip and invalid states/schemas are rejected. Two concurrent clients cannot overwrite a workflow revision silently.

**Evidence:** [Implemented contract specification](../CONTRACTS.md), embedded JSON Schemas, real HTTP validation tests, guarded lifecycle tests, concurrent revision-conflict tests and round-trip protocol fixtures. Stage C remains In review until its PR is accepted.

<a id="stage-d"></a>

### D. Build durable storage and dispatch foundations

**Prerequisites:** C.  
**Outcome:** Persist accepted work before acknowledging it or dispatching an external action.  
**Deliverables:** PostgreSQL migrations, repositories, transactional outbox/inbox and retention primitives.

- [ ] **D01** — Implement workspace-scoped storage and indexes for the core entities. Define uniqueness constraints for request idempotency, external event IDs, task attempt numbers and artifact references.
- [ ] **D02** — Persist task transitions and corresponding outbox events in one transaction. Acknowledge an event only after durable ingestion; deduplicate replayed runner/provider events.
- [ ] **D03** — Implement dispatch leases, lease expiry and fencing tokens. A stale worker must not commit an authoritative task transition after losing its lease.
- [ ] **D04** — Test fresh install, upgrade from the previous schema, backup/restore and interrupted transactions against real PostgreSQL. Define bounded event retention and snapshot behavior for old stream cursors.

**Exit gate:** An accepted task survives restart. Duplicate requests create one logical task; killing a process at the transaction boundary loses no acknowledged state.

**Evidence:** Migration logs and storage/idempotency failure tests.

<a id="stage-e"></a>

### E. Establish identity and workspace boundaries

**Prerequisites:** C, D.  
**Outcome:** Require a valid user and workspace authorization before shared access or execution.  
**Deliverables:** Self-hosted bootstrap, sessions, workspace membership and permission enforcement.

- [ ] **E01** — Implement one-time owner setup, sign-in/sign-out, secure session lifecycle and recovery procedures. Prevent public takeover of an uninitialized deployment.
- [ ] **E02** — Define owner, admin, operator and viewer roles with server-enforced permissions for connections, runs, approvals, credentials and artifacts.
- [ ] **E03** — Scope every API, event subscription, file download and background job to its authenticated workspace. Reject caller-supplied identity claims.
- [ ] **E04** — Build the basic onboarding/settings UI and document reverse-proxy/cookie/CSRF configuration. Test invitation expiry and revocation if invitations are included in this stage.

**Exit gate:** An unauthorized request and cross-workspace ID substitution both fail on reads, writes, streams and downloads. Revoked access cannot create new operations.

**Evidence:** Authorization matrix, negative tests and bootstrap walkthrough.

<a id="stage-f"></a>

### F. Protect credentials and authorize actions

**Prerequisites:** E.  
**Outcome:** Make permission and credential handling part of the first real execution path.  
**Deliverables:** Secret references, credential lifecycle, policy engine and durable approval records.

- [ ] **F01** — Store provider credentials encrypted or through an external secret-store interface. Define key rotation, backup and recovery; return metadata and secret references to the browser.
- [ ] **F02** — Scope secrets to workspace, connection and runner. Supply only the credentials needed by a task; redact structured logs, events, error messages and exports.
- [ ] **F03** — Implement approvals bound to actor, workspace, action/input digest, scope, expiry and policy version. Approval replay, stale approval and self-approval by a planner must fail.
- [ ] **F04** — Define low-risk permissions and explicit approval for privileged actions. Where a provider offers only whole-run approval, expose that granularity and restrict execution accordingly.

**Exit gate:** A denied action has no side effect. Changing an approved action invalidates its approval; stored synthetic secrets do not appear in returned browser/API payloads, logs or exports.

**Evidence:** Policy/approval tests and redaction evidence with synthetic credentials.

<a id="stage-g"></a>

### G. Create isolated local and private runners

**Prerequisites:** C, D, E, F.  
**Outcome:** Execute agent processes outside the control-plane process with scoped workspaces.  
**Deliverables:** Runner binary/service, authenticated outbound transport and execution sandbox profiles.

- [ ] **G01** — Implement one-time runner enrollment, revocation, capability/version handshake and an authenticated outbound connection. Reconnect with bounded local buffering and replay acknowledgments.
- [ ] **G02** — Define workspace mounts, resource/time limits, process-tree termination and cleanup. Run isolated tasks without control-plane credentials, host-wide mounts or an exposed Docker socket.
- [ ] **G03** — Enforce dispatch fencing and deduplication locally. Persist the dispatch identity and provider session reference before reporting acceptance when the provider permits it.
- [ ] **G04** — Support Go adapters and supervised Node.js/Python adapter processes through the same runner contract. Report offline, overloaded and incompatible runners explicitly.

**Exit gate:** A task cannot read another workspace or inherit unrelated credentials. Disconnect/reconnect does not launch a duplicate process; revoked runners cannot accept new work.

**Evidence:** Runner lifecycle, isolation and reconnect tests on the documented reference host.

<a id="stage-h"></a>

### H. Build the agent registry and adapter contract

**Prerequisites:** C, F, G.  
**Outcome:** Register real connections and discover their actual capabilities.  
**Deliverables:** Registry APIs/UI, connector interfaces, connection testing and conformance harness.

- [ ] **H01** — Separate catalog definitions, installed connector versions and configured agent connections. Collect an alias, adapter ID, workspace, execution location and secret reference.
- [ ] **H02** — Implement connection/discovery checks that distinguish missing configuration, bad credentials, unsupported protocol, unreachable service and available agent.
- [ ] **H03** — Define optional operations such as stream, cancel, resume, artifacts, usage and approvals. Report capabilities from the tested adapter/provider combination.
- [ ] **H04** — Create deterministic protocol fakes for CI and conformance cases for success, failure, denial, cancellation and reconnect. Keep test fakes out of the production connection catalog.

**Exit gate:** Registry responses describe configured connections accurately. Unsupported operations fail explicitly; test-only agents cannot masquerade as installed production providers.

**Evidence:** Registry acceptance tests and reusable connector conformance cases.

<a id="stage-i"></a>

### I. Integrate the first real agent: Codex

**Prerequisites:** G, H.  
**Outcome:** Complete an actual coding task through the runner and normalized event contract.  
**Deliverables:** Codex adapter, setup documentation, pinned compatibility record and a real fixture run.

- [ ] **I01** — Implement one documented Codex integration route, initially App Server for interactive sessions and approvals. Pin the tested runtime/schema; keep SDK alternatives behind the same contract.
- [ ] **I02** — Translate messages, progress, permission requests, terminal outcomes and provider thread IDs into Confignostic events without inventing missing fields.
- [ ] **I03** — Run a small change against a disposable reference repository and collect its result/diff, trace and provider usage when available.
- [ ] **I04** — Exercise invalid authentication, provider failure and unsupported operations. Document account/runtime setup and explicitly mark capabilities that are unavailable.

**Exit gate:** A real Codex task produces verifiable output in the assigned workspace. Denied actions do not run, and failure is visible with its original correlation identifiers.

**Evidence:** Sanitized real-run transcript, artifact digest, runtime version and conformance results.

<a id="stage-j"></a>

### J. Deliver functional unified chat

**Prerequisites:** D, E, F, I.  
**Outcome:** Give the user a complete one-agent experience through the Svelte UI.  
**Deliverables:** Conversation UI, durable messages, live timeline, approvals and artifact views.

- [ ] **J01** — Implement conversation creation/history, project context, connected-agent selection and unambiguous @mentions. Persist the outbound message and task together.
- [ ] **J02** — Stream real events through the API with reconnect cursors, bounded rendering and clear queued/running/waiting/error states. Maintain useful empty and offline states.
- [ ] **J03** — Render approval and user-input requests with the actual action scope. Show agent identity, task identity, outputs and sanitized artifact previews/downloads.
- [ ] **J04** — Add accessible keyboard navigation, mobile layouts and attachment limits. Accept attachments only through capabilities the selected connector supports.

**Exit gate:** From a clean browser session a user can connect, send, observe, approve or deny, reload, and find the same task/result. Retrying the same submission with its idempotency key, including across tabs, creates one logical task.

**Evidence:** Browser end-to-end recording and critical chat/approval regression checks.

<a id="stage-k"></a>

### K. Prove recovery and execution correctness

**Prerequisites:** D, G, I, J.  
**Outcome:** Handle process loss, network failure and ambiguous external execution honestly.  
**Deliverables:** Task reconciliation, retry policy, cancellation semantics and failure drills.

- [ ] **K01** — Reconcile persisted attempts with runner/provider session state after restart. Mark an unconfirmed external outcome unknown and stop dependent steps until reconciled.
- [ ] **K02** — Retry only according to the operation's documented idempotency/reconciliation support. Keep dispatch IDs, attempt IDs and logical task IDs distinct.
- [ ] **K03** — Implement cancel-requested and timeout behavior without claiming the provider has stopped until confirmation. Queue or reject concurrent prompts to a provider session according to its capabilities.
- [ ] **K04** — Inject failures before dispatch, after remote acceptance, during streaming and at completion. Verify lease expiry, event replay, graceful drain and partial-output recovery.

**Exit gate:** The reference task survives API restart and reconnect. Losing an acceptance response never causes a blind duplicate side effect; cancellation and uncertainty remain visible.

**Evidence:** Fault-injection report with task/event histories for every failure boundary.

<a id="stage-l"></a>

### L. Add ACP and the second provider: Cursor

**Prerequisites:** H, K.  
**Outcome:** Connect a second independent agent through a reusable protocol client.  
**Deliverables:** ACP runner client, Cursor adapter/configuration and protocol-version tests.

- [ ] **L01** — Implement ACP initialization, session setup, prompts, notifications, permission requests and cancellation over stdio. Bound messages and keep protocol stdout separate from diagnostic stderr.
- [ ] **L02** — Integrate Cursor CLI through its documented ACP entry point. Handle blocking user-question/plan requests and explicitly reject unsupported extension methods.
- [ ] **L03** — Pin the tested agent/protocol versions and map capabilities per connection. Do not equate access to a CLI session with access to all IDE sessions.
- [ ] **L04** — Run the same reference fixture and failure tests through Cursor; verify the same chat UI can select Codex or Cursor without losing provider-specific limitations.

**Exit gate:** Two independently configured providers complete real tasks through one chat. A permission/input request cannot hang silently or be approved by the planner.

**Evidence:** Real Codex and Cursor runs plus ACP interoperability and permission tests.

<a id="stage-m"></a>

### M. Implement artifacts and typed agent handoffs

**Prerequisites:** K, L.  
**Outcome:** Pass an agent's verified output to another agent with clear provenance.  
**Deliverables:** Artifact storage abstraction, typed handoff envelopes and isolated repository workspaces.

- [ ] **M01** — Store immutable artifact references with workspace ownership, content hashes, size/type metadata and provenance. Validate downloads and preview content safely.
- [ ] **M02** — Define typed handoff inputs and bounded context selection. Validate structured outputs and distinguish untrusted agent text from executable workflow instructions.
- [ ] **M03** — Use per-run/per-agent worktrees or equivalent isolated workspaces. Track base revisions, resulting commits/patches and conflicts; require explicit merge/publication policy.
- [ ] **M04** — Implement the reference handoff: Codex change to Cursor review, with the artifact/base revision visible to both user and reviewer. Surface invalid/missing outputs.

**Exit gate:** The reviewer receives the exact artifact version produced by the first agent. Concurrent edits stay isolated and invalid outputs block the handoff.

**Evidence:** Two-provider handoff run with artifact hashes, schema checks and a conflict case.

<a id="stage-n"></a>

### N. Build the deterministic workflow engine

**Prerequisites:** C, D, K, M.  
**Outcome:** Execute versioned workflows independently of how the user created them.  
**Deliverables:** Workflow validator, scheduler, node executors and run state aggregation.

- [ ] **N01** — Implement agent, approval, condition, bounded transform and join nodes, with typed edges. Validate cycles, unreachable nodes, missing inputs, limits and capability requirements before dispatch.
- [ ] **N02** — Support sequential and bounded parallel execution, explicit failure/skip branches, join rules, timeouts and per-node retry policies. Keep condition/transform evaluation constrained.
- [ ] **N03** — Persist a run's immutable workflow version and inputs. Use transaction/lease boundaries for node dispatch; resume runnable nodes correctly after restart.
- [ ] **N04** — Add run-wide pause/cancel controls and scoped node retries that create traceable attempts. Keep unknown upstream outcomes from silently releasing downstream work.

**Exit gate:** The same persisted graph yields the same scheduling decisions under replay. Branch/join, duplicate dispatch and restart tests show no unauthorized or premature execution.

**Evidence:** Engine state-transition tests and real sequential/parallel multi-provider workflows.

<a id="stage-o"></a>

### O. Let chat propose and route workflows

**Prerequisites:** F, H, J, N.  
**Outcome:** Translate user intent into reviewable workflows while preserving policy authority.  
**Deliverables:** Planner interface, routing preferences, graph review and context/budget controls.

- [ ] **O01** — Offer direct-agent mode and orchestrated mode. Use configured capabilities and user routing preferences to choose eligible agents.
- [ ] **O02** — Have the planner return the versioned workflow schema, then validate it through the deterministic engine. Show agent choices, intended outputs and required approvals before execution.
- [ ] **O03** — Keep authorization outside model output. Re-check policy at dispatch and when an action changes; treat retrieved text, agent outputs and imported plans as untrusted inputs.
- [ ] **O04** — Track usage/cost when reported, label estimates and unknown values, enforce pre-dispatch budgets and max steps, and disclose limits on stopping already-running remote work.

**Exit gate:** A natural-language request creates an editable valid graph and runs only after applicable review/policy checks. Malformed plans, injected instructions and unavailable agents fail safely.

**Evidence:** Planner evaluation fixtures and a real chat-created two-provider workflow.

<a id="stage-p"></a>

### P. Build the shared visual workflow canvas

**Prerequisites:** C, J, N, O.  
**Outcome:** Edit and inspect the exact workflow representation used by chat.  
**Deliverables:** Svelte node canvas, graph inspector, import/export and live run visualization.

- [ ] **P01** — Implement node creation, drag/drop, typed ports, edge editing, settings panels and validation messages. Provide a keyboard-operable alternative to pointer-only editing.
- [ ] **P02** — Round-trip chat-created graphs through the canvas and canvas-created graphs through chat. Use revision checks, undo/redo and versioned saves.
- [ ] **P03** — Render live node/edge states from persisted execution events. Keep the immutable historical run definition separate from editable future versions.
- [ ] **P04** — Implement schema-versioned import/export with credential values removed and connection bindings explicitly remapped. Handle large graphs, loading, empty, offline and error states.

**Exit gate:** One graph can be created in chat, edited on the canvas, exported/imported and executed without semantic drift. Concurrent editing cannot silently overwrite a newer revision.

**Evidence:** Round-trip fixtures, browser workflow tests and keyboard/screen-reader review.

<a id="stage-q"></a>

### Q. Expand the core coding-agent connectors

**Prerequisites:** H, K, M.  
**Outcome:** Add the remaining dedicated coding-runtime targets using the shared contract.  
**Deliverables:** Claude, Copilot, OpenCode, OpenHands and Grok Build connectors and compatibility records.

- [ ] **Q01** — Integrate Claude Agent SDK and GitHub Copilot SDK as separate reviewable changes. Select one supported language/runtime route per connector before adding alternatives.
- [ ] **Q02** — Integrate OpenCode through its server API/SDK and OpenHands through Agent Server/SDK; verify session ownership, workspace behavior and event translation.
- [ ] **Q03** — Integrate Grok Build through the validated ACP path or documented structured headless interface. Keep its agent runtime distinct from the Grok model API.
- [ ] **Q04** — Run conformance and real-task qualification for each provider. Publish tested versions, credential setup, execution location and capability gaps independently.

**Exit gate:** Each of the five additions has a real passing task and honest capability manifest. Any blocked provider remains Planned or Experimental without blocking unrelated connectors.

**Evidence:** Five provider-specific compatibility records and sanitized real-run evidence.

<a id="stage-r"></a>

### R. Connect remote and stateful agent services

**Prerequisites:** H, K, M, N.  
**Outcome:** Support customer-deployed agents beyond local coding CLIs.  
**Deliverables:** A2A/custom HTTP clients plus ADK, Microsoft Agent Framework, LangGraph and Letta integrations.

- [ ] **R01** — Implement A2A discovery, authentication, negotiated protocol bindings, messages/tasks and optional streaming/cancel/resume behavior. Validate cards and advertised capabilities.
- [ ] **R02** — Define endpoint and egress policies for discovery, redirects and artifact URLs. Block cloud metadata access and unapproved private destinations; allow explicit private-service access only through scoped policies/runners.
- [ ] **R03** — Qualify real Google ADK and Microsoft Agent Framework sample deployments through A2A. Add a versioned custom HTTP adapter example with the same task contract.
- [ ] **R04** — Connect LangGraph deployments and Letta agents through documented APIs/SDKs, preserving remote session identity, state ownership and supported continuation behavior.

**Exit gate:** All four agent families pass real execution tests. Endpoint redirection and cross-workspace credentials cannot bypass network or authorization policy.

**Evidence:** Remote-service fixtures, SSRF/authorization tests and four compatibility records.

<a id="stage-s"></a>

### S. Qualify the broader ACP ecosystem

**Prerequisites:** H, K, L.  
**Outcome:** Expand coverage through one ACP client with provider-specific validation.  
**Deliverables:** Nine additional ACP configurations/adapters and a versioned support matrix.

- [ ] **S01** — Qualify Gemini CLI, Goose and Cline individually; document executable acquisition, authentication and session behavior.
- [ ] **S02** — Qualify Factory Droid, JetBrains Junie and Kimi CLI individually; test permission/input handling and extension behavior.
- [ ] **S03** — Qualify Kiro CLI, Mistral Vibe and Qwen Code individually; verify capabilities and supported protocol versions.
- [ ] **S04** — Run the same conformance suite per agent and keep compatibility quirks in explicit adapter configuration. Record third-party adapter provenance where applicable.

**Exit gate:** Every agent promoted to Supported passes an actual task, denial/error and reconnect checks for its declared capabilities. An ACP directory listing alone cannot change its status.

**Evidence:** Nine per-provider compatibility records; blocked cases remain visible in the catalog.

<a id="stage-t"></a>

### T. Add tools, X integration and Grok research agents

**Prerequisites:** F, H, K, N, Q.  
**Outcome:** Extend agent work with governed tools and research without confusing tools with agents.  
**Deliverables:** MCP configuration, X tool connector, Grok API-powered agent adapter and action policies.

- [ ] **T01** — Connect MCP tools through supported agent runtimes, including discovery, authorization, timeouts and tool-level allowlists. Keep tool connections separate from agent registrations.
- [ ] **T02** — Implement X read/research operations and separately permissioned account-changing operations through supported APIs/SDKs or MCP interfaces.
- [ ] **T03** — Build the configured Grok API-powered agent route with an explicit agent loop/tool policy, web/X research capability and normalized output/citations/usage.
- [ ] **T04** — Enforce approval on externally visible mutations, validate tool results, and keep secrets out of prompts/exports. Record rate limits and unknown/partial provider results honestly.

**Exit gate:** A research result can feed another agent as a typed artifact. Denied X/account actions do not occur, and an agent cannot elevate its own tool permissions.

**Evidence:** Research-to-handoff demo, synthetic tool tests and authorized sandbox-account action tests.

<a id="stage-u"></a>

### U. Add automation, collaboration and reusable workflows

**Prerequisites:** E, F, K, N, O.  
**Outcome:** Make repeatable work usable by a self-hosted team.  
**Deliverables:** Templates, scheduled/webhook runs, in-app notifications and shared workspace workflows.

- [ ] **U01** — Provide reusable templates for implementation/review, issue triage and research/reporting. Version templates and require users to bind their own connections and secrets.
- [ ] **U02** — Implement scheduled runs with IANA timezones, DST behavior, missed-run policy, pause/resume and deduplication. Record the exact scheduled occurrence that created a run.
- [ ] **U03** — Implement signed webhooks with replay protection, event deduplication, bounded payloads and concurrency limits. Route notifications through explicit user configuration.
- [ ] **U04** — Support shared conversations, run ownership, human approvals, activity history and concurrent graph edits with server-enforced roles. Include these capabilities in self-hosted core.

**Exit gate:** Replayed webhooks and duplicated scheduler ticks create one logical occurrence. Unauthorized collaborators cannot approve, modify or read restricted work.

**Evidence:** Schedule/DST and webhook replay tests plus a multi-user browser walkthrough.

<a id="stage-v"></a>

### V. Package and operate the self-hosted product

**Prerequisites:** E, F, G, K, P.  
**Outcome:** Make installation, upgrades and recovery repeatable for someone outside the project.  
**Deliverables:** Versioned images/binaries, Compose deployment, operations guide and backup/restore commands.

- [ ] **V01** — Ship the built Svelte UI/API, PostgreSQL configuration and separately enrolled runners with persistent volumes, health checks, least-privilege defaults and documented TLS setup.
- [ ] **V02** — Document supported OS/architecture, runtime prerequisites, proxy/network requirements and resource sizing. Allow inspection/configuration without requiring a cloud subscription.
- [ ] **V03** — Implement export, retention, deletion and backups for database state, artifacts and required key material. Test restore to a fresh host and dependency-compatible application rollback.
- [ ] **V04** — Provide sanitized diagnostics, structured logs, metrics, tracing and runbooks for offline runners, stuck/unknown tasks, expired credentials, migration failure and provider outages.

**Exit gate:** A fresh operator completes install, one real task, upgrade, backup and restore using only published instructions. Core chat and workflows remain usable without a Confignostic cloud account.

**Evidence:** Clean-host deployment log, backup/restore report and release artifact manifest.

<a id="stage-w"></a>

### W. Qualify the complete self-hosted release

**Prerequisites:** P, Q, R, S, T, U, V.  
**Outcome:** Prove the quality target with evidence across the complete planned scope.  
**Deliverables:** Release qualification report, resolved blockers and an accurate support matrix.

- [ ] **W01** — Run every applicable gate in QUALITY_GATES.md against the candidate commit, including authorization, secret handling, real connectors, workflow recovery and artifact provenance.
- [ ] **W02** — Measure control-plane latency, streaming responsiveness and resource usage on the defined reference profile. Distinguish deterministic load tests from actual provider tests.
- [ ] **W03** — Review accessibility, responsive layouts, keyboard workflows and long-running/large-history behavior; fix failures in critical journeys.
- [ ] **W04** — Complete a targeted security/design review of runners, imports, network egress and authorization. Record findings, owners, fixes and retests; keep unverified capabilities out of release claims.

**Exit gate:** All mandatory self-hosted gates pass with evidence. No unresolved release-blocking correctness, data-loss, authorization or secret-exposure defects remain.

**Evidence:** Candidate qualification report linked to exact commit, runtime versions and test artifacts.

<a id="stage-x"></a>

### X. Build the optional managed cloud service

**Prerequisites:** W.  
**Outcome:** Operate the same core as a managed service with explicit tenant and billing boundaries.  
**Deliverables:** Tenant provisioning, managed runner isolation, usage ledger, billing integration and operator controls.

- [ ] **X01** — Provision isolated workspaces, storage, secrets and runner capacity with quotas and rate limits. Treat customer code as untrusted and use a suitable per-tenant workload boundary.
- [ ] **X02** — Support outbound customer-hosted runners with scoped enrollment/revocation. Define what data/artifacts are transferred and allow administrators to inspect that policy.
- [ ] **X03** — Implement plans, metering and billing webhooks with verified signatures, idempotent ledgers, reconciliation and grace/export behavior. Keep provider billing/BYOK costs distinct from Confignostic hosting charges.
- [ ] **X04** — Add audited operator access, tenant export/deletion, retention controls and environment separation. Preserve the documented self-hosted feature set without a hosted-license dependency.

**Exit gate:** Cross-tenant access fails across every resource, billing replays do not double-charge, and a self-hosted installation continues to work without cloud billing.

**Evidence:** Tenant isolation, metering/reconciliation and private-runner acceptance results.

<a id="stage-y"></a>

### Y. Validate cloud operations and paid pilots

**Prerequisites:** W, X.  
**Outcome:** Prove service operation and willingness to pay before general cloud availability.  
**Deliverables:** Pilot results, incident/recovery exercises, SLO dashboards and a support model.

- [ ] **Y01** — Onboard a small explicitly recruited pilot group and record setup friction, repeat usage, successful handoffs and concrete payment feedback.
- [ ] **Y02** — Exercise provider outages, tenant throttling, queue pressure, revoked runners, billing failures and operator-access audits in staging.
- [ ] **Y03** — Measure the cloud availability/recovery targets from QUALITY_GATES.md and perform a real restore exercise with database, artifacts and keys.
- [ ] **Y04** — Define support coverage that matches actual staffing, escalation/runbooks, incident communication and rollback decisions. Adjust pricing/limits from measured operating costs.

**Exit gate:** Paid-pilot evidence and service-readiness evidence are both recorded. Remaining blockers prevent general cloud availability rather than being hidden by a launch date.

**Evidence:** Pilot decision report, operating-cost model and cloud recovery/incident reports.

<a id="stage-z"></a>

### Z. Release and maintain the product

**Prerequisites:** W. Z-CLOUD also requires Y.  
**Outcome:** Ship the self-hosted product and optional cloud on separate, evidence-backed release tracks.  
**Deliverables:** Tagged releases, migration notes, documentation and a sustainable maintenance process.

- [ ] **Z01** — Z-OSS: after W passes, publish the self-hosted stable release with signed/provenanced artifacts, installation/upgrade guides, support matrix and known limitations. Cloud readiness is not an OSS release dependency.
- [ ] **Z02** — Create a regular connector-compatibility review and dependency-update process. Document deprecations, migration windows and security-reporting channels.
- [ ] **Z03** — Z-CLOUD: after Y passes, approve general availability separately with billing, recovery, support and tenant-isolation evidence.
- [ ] **Z04** — Maintain a feedback-driven backlog and rerun the relevant quality gates on changes. Keep statuses, compatibility versions and release claims aligned with tested behavior.

**Exit gate:** The OSS release satisfies W. The cloud release additionally satisfies Y. Release artifacts, documentation and the published capability matrix agree.

**Evidence:** Release tags/manifests, qualification reports and post-release issue/compatibility tracking.

## Decisions and risk handling

| Decision or risk | Default / resolution step | Owner and checkpoint |
| --- | --- | --- |
| First integration | Codex App Server; confirm account access and runtime compatibility before I. | Implementer, H→I |
| Second independent provider | Cursor via ACP; use a separate connection/runtime and verify actual provider execution. | Implementer, L |
| Provider unavailable or incompatible | Record a blocker and preserve its catalog status; choose another explicit target for continued engineering without claiming the blocked integration works. | Product owner, affected stage |
| SDK language mismatch | Keep the control plane in Go; supervise a Node/Python adapter in the runner where required. | Implementer, G/H |
| Uncertain external side effect | Reconcile by provider/session/operation ID; preserve unknown if confirmation is unavailable. | Implementer, K/N |
| Several agents editing one repository | Isolated worktrees/workspaces, artifact/base hashes and explicit merge policy. | Implementer, M |
| Provider cost unavailable | Display unknown or a labeled estimate with its basis; never turn missing usage into a zero-cost claim. | Implementer, O/X |
| Scope growth | Map a request to the product contract, document its tradeoff and update affected gates before implementation. | Product owner, ongoing |
| Capacity and delivery dates | Plan one primary implementer; measure actual throughput through I–K, then estimate the next checkpoint with a range and stated assumptions. | Product owner, R1 |
| Cloud operating burden | Prove self-hosted value first; commit support coverage and cloud limits only after measured pilot operation. | Product owner, Y |

## Relationship to the existing GitHub issues

| Existing issue | Expanded coverage in this plan |
| --- | --- |
| [#1 Agent registry](https://github.com/UXCentury/confignostic/issues/1) | C–I, especially the registry/conformance contract in H. |
| [#2 Functional unified chat](https://github.com/UXCentury/confignostic/issues/2) | E–K, with the user journey in J. |
| [#3 Durable state and second-provider handoff](https://github.com/UXCentury/confignostic/issues/3) | D, K–N. Persistence starts before handoff work. |
| [#4 Visual workflow canvas](https://github.com/UXCentury/confignostic/issues/4) | N–P, using one engine and schema. |

Keep these issue identities for continuity. Use task IDs such as K02 or P03 in focused implementation issues and PRs. No implementation issue is completed merely because this plan has been written.
