# Development execution guide

This guide turns the [A–Z plan](DEVELOPMENT_PLAN.md) into reviewable implementation work. It does not grant credentials, authorize production actions or mark any feature complete.

## Working model

Plan for one primary implementer. Work on one substantial runtime change at a time, with documentation and review alongside it. Stages with satisfied prerequisites may be scheduled independently, but concurrent changes to shared schemas or execution state require explicit coordination.

Task IDs A01–Z04 are stable references. Use them in issues, branches, PRs and release evidence. A checkbox is complete only when its behavior and failure cases are verified. Update the task, stage and connector status separately.

| Status | Meaning |
| --- | --- |
| Planned | Defined, with implementation/evidence outstanding. |
| Ready | Prerequisites are verified and required access is available. |
| In progress | A scoped implementation change is being worked on. |
| Blocked | A concrete dependency, access or design problem prevents completion; owner and next step are recorded. |
| In review | Implementation and verification evidence are available for review. |
| Done | Acceptance criteria pass, evidence is linked, and documentation reflects actual behavior. |

## Per-task workflow

1. Inspect the current repository and its instructions. Read the selected stage, prerequisites, existing issue and relevant architecture/contracts. Preserve unrelated changes.
2. Write a short implementation note: user-visible result, affected contracts/modules, failure cases, verification and migration needs. Resolve consequential contract changes before coding.
3. Implement the smallest complete behavior. Include state/error handling and permissions on the first path that executes real work.
4. Run checks appropriate to the change. Use real PostgreSQL for persistence semantics and deterministic provider fakes for fault injection; also run a real provider task before claiming connector support.
5. Review the diff for hidden behavior, unsafe defaults, accidental secrets, unrelated changes and API/schema compatibility.
6. Submit the implementation for the repository's normal review path with the task ID and evidence. Update user/operator documentation and support status truthfully.
7. Mark Done after its acceptance gate passes. Re-estimate the next checkpoint from actual throughput and discovered integration costs.

Do not interpret the A–Z plan as a request to implement every stage in one unreviewable change. A table of unsupported capabilities is a valid outcome of discovery; a fabricated success is not.

## Task issue / PR brief

Copy this structure when starting an implementation task:

- **Task:** stage and ID, for example K02.
- **User result:** the concrete behavior the user will observe.
- **Prerequisites:** completed tasks, provider access and required environments.
- **Scope:** modules/contracts affected and the acceptance criteria being delivered.
- **Implementation:** the chosen approach and any material tradeoff.
- **Failure behavior:** denial, timeout, disconnect, duplicate request and unknown outcome where applicable.
- **Validation:** exact meaningful checks, environment and result; distinguish simulated from real-provider evidence.
- **Data/operations:** migration, backward compatibility, deployment and recovery implications.
- **Documentation:** setup, capabilities, limitations and status updates.
- **Outstanding:** explicit blockers or follow-up work; no unsupported completeness claim.

A documentation-only change needs a content/link/diff review, not unrelated runtime test execution. Runtime changes need focused tests for real behavior; avoid tests that merely repeat the implementation.

## Recommended first implementation sequence

| Order | Tasks | Result |
| --- | --- | --- |
| 1 | A01–A04 | Confirm the product contract and reference workflow. |
| 2 | B01–B04 | Reproducible toolchain, frontend CI and corrected shutdown behavior. |
| 3 | C01–C04 | Agree on identities, task/event contracts and workflow schema. |
| 4 | D01–D04 | Durable storage and dispatch foundations. |
| 5 | E01–F04, in stage order | User/workspace boundaries, credentials and action policy. |
| 6 | G01–H04, in stage order | Isolated runner and honest registry/connector interface. |
| 7 | I01–K04, in stage order | Real Codex task, functional chat, then restart/failure recovery. |
| 8 | L01–M04, in stage order | Real Cursor connection and typed independent-agent handoff. |
| 9 | N01–P04, in stage order | Workflow engine, chat planning and the shared canvas. |
| 10 | Q–V following prerequisites | Complete provider coverage, tools, collaboration and self-hosted operations. |
| 11 | W and Z-OSS | Qualify and publish the full self-hosted release. |
| 12 | X, Y and Z-CLOUD | Build, validate and release managed cloud separately. |

The first real provider milestone requires configured access. Record a missing account/credential as Blocked and continue other independent work; never substitute an unrelated model and call it the requested agent.

## Checks by change type

| Change | Relevant checks |
| --- | --- |
| Go domain/API behavior | Formatting, vet, focused unit/integration tests, build; race checks for concurrent dispatch/event code. |
| Database/state/retry behavior | Real PostgreSQL integration, migrations, concurrent claims and failure-boundary tests. |
| Svelte interaction | Type/Svelte checks, build, focused component tests and browser tests for affected critical journeys. |
| Provider adapter | Contract fixtures, conformance failure cases, pinned-version compatibility and a real authorized task. |
| Workflow/planner/canvas | Schema validation, scheduling replay, policy denial and chat/canvas round-trip tests. |
| Secrets/auth/permissions | Positive and negative authorization cases, secret canaries, revocation and approval replay tests. |
| Packaging/operations | Clean install, upgrade, backup/restore and sanitized diagnostics exercises. |
| Cloud metering/billing | Signed webhook verification, duplicate/out-of-order events and ledger reconciliation in a sandbox. |
| Documentation only | Factual consistency, valid relative links/anchors, task/status coherence and scoped diff review. |

As scripts are added in B, document exact runnable commands in CONTRIBUTING.md. Do not document a command as available before implementing it.

## Evidence organization

Proposed locations to introduce when implementation begins:

- docs/decisions/ for material architecture decisions, with consequences and revision history.
- docs/compatibility/ for per-provider version/capability qualification records.
- docs/releases/ for release-candidate scopes, gate results and operator notes.
- CI artifacts or an access-controlled evidence store for sanitized traces, browser recordings and benchmark data.

Each evidence record names the source commit, exact upstream versions, test environment and operator/reviewer. Keep expensive provider-dependent tests outside untrusted fork CI and make their credentials narrowly scoped.

## Scope and status updates

The [catalog](../SUPPORTED_AGENTS.md) records Planned, Experimental and Supported per connector. Change its status only when the associated evidence exists.

The [roadmap](../ROADMAP.md) summarizes checkpoints; the detailed A–Z plan owns task IDs and dependencies. Update both when a genuine dependency or product decision changes. Keep the self-hosted release path independent from managed cloud.

Use the existing GitHub issues #1–#4 as early parent scopes. Create narrowly scoped follow-up issues when implementation starts, rather than treating the entire roadmap as one task. A blocker should state what is missing, who can resolve it and what independent work remains available.
