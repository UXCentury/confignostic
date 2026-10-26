# Product contract and acceptance brief

Scope baseline: the approved A–Z plan, 2026-10-24. Foundation implementation: 2026-10-26. Product owner: Davit Gabrielyan. Technical owner for each change: its implementing PR author.

**One chat to command your AI agents. One canvas to orchestrate them.**

Confignostic connects existing independent agent systems through one self-hosted workspace. The first intended users are solo developers and small development teams who already use multiple coding agents. It coordinates their actual runtimes and preserves their capabilities and limitations.

## Primary journey

1. Install the self-hosted core, complete protected owner setup and create a workspace/project.
2. Enroll a private runner, configure an agent connection and verify its capabilities and authentication.
3. Select a connected agent or use an unambiguous @mention, then send a bounded task from chat.
4. Observe real progress and approve or deny actions with the exact scope shown.
5. Inspect persistent outputs, artifacts, errors and usage where the provider exposes it.
6. Ask for coordinated work, review the proposed graph, and hand a versioned result to another independent agent.
7. Open that same graph on the canvas, edit a new revision, and inspect the existing run without changing its recorded definition.

The repository currently implements none of this end-to-end journey. The initial API and UI remain scaffolds while these stages are built.

## Reference implementation-and-review task

The [reference fixture](../testdata/handoff/README.md) supplies a deliberately incomplete alias-normalization module, original tests, an external acceptance checker, and positive/negative checker tests.

Codex must make a bounded change so aliases are trimmed and lowercase while preserving type/empty-input validation. Cursor must independently review the exact resulting artifact against the same base and requirements. No remote publication or account-changing action is needed.

A successful full-system run must demonstrate:

- One persistent conversation containing the implementation and review tasks, with traceable internal and provider identities.
- A reviewed workflow and applicable action approvals before dispatch.
- Isolated execution workspaces and a handoff referencing the exact base and output digest.
- Passing original and external acceptance checks, with truthful reviewer findings.
- Recovery after API restart/runner disconnection without blindly repeating accepted external work.
- Agreement between the chat timeline, persisted run state and canvas.
- Denial of cross-workspace access to the conversation, artifacts and credentials.

The fixture checker is only an output oracle. A passing local check is not evidence of an agent connection or orchestration.

## Failure acceptance

| Trigger | Required outcome | Planned stages |
| --- | --- | --- |
| Missing/invalid provider access | Connection unavailable with a useful error; no simulated success. | H–I, L |
| Human denies an action | That action does not execute; record the decision and permitted next step. | F, I–K |
| API restarts or acknowledgment is lost | Reconcile the existing attempt; preserve unknown when the external result cannot be confirmed. | D, K |
| Output is missing, invalid or fails acceptance | Preserve evidence, block dependent review/execution where appropriate, and expose the failure. | M–N |
| Review receives a different base/output | Reject the mismatched handoff. | M |
| Concurrent agents edit the same files | Keep isolated outputs; require an explicit merge decision. | M |
| Unauthorized user guesses a resource ID | Deny reads, downloads, event streams and mutations. | E, W |
| Graph changes while a run is active | Create a new workflow version; preserve the running version. | C, N–P |

## Complete scope and delivery mapping

| Capability | Stages |
| --- | --- |
| Durable chat, identity, permissions, scoped credentials and isolated runners | C–K |
| Codex followed by Cursor and a typed independent-provider handoff | I–M |
| Deterministic orchestration, chat planning and a shared drag-and-drop canvas | N–P |
| All 21 named agent targets | I, L, Q–T; per-provider list in the catalog |
| ACP, A2A and custom HTTP/SDK adapters | H, L, Q–S |
| MCP tools, X integration and configured Grok API agents | T |
| Templates, scheduling, webhooks and team collaboration | U |
| Installation, upgrades, export/deletion, backup/restore and observability | V–W |
| Optional managed hosting, tenant operations, billing and support | X–Z |

The [catalog](SUPPORTED_AGENTS.md) is authoritative for the 21 planned targets. Protocol support does not automatically qualify every provider.

Full chat, connections, orchestration, canvas, approvals, artifacts, tools, automation and collaboration remain available in the open-source self-hosted core. Cloud sells managed hosting, operations, reliability and support. Provider accounts and usage charges remain separate. No cloud account or hosted license is required to use the core.

## Assumptions and owners

| Assumption | Owner | Validation checkpoint |
| --- | --- | --- |
| Initial supported development environment is Linux x86-64 with Docker for PostgreSQL and future isolated runners. | Technical implementer | B; broader OS/runtime support remains unverified. |
| A single primary implementer delivers small changes in dependency order. | Product owner | Review actual throughput at R1 before assigning dates. |
| Codex and Cursor accounts/runtime access will be available for real tests. | Product owner supplies access; implementer verifies it | H→I and L. Access is not verified by this foundation change. |
| Developers value a persistent cross-provider handoff enough to return. | Product owner | User feedback alongside I–M. Unvalidated commercial hypothesis. |
| Managed hosting offers enough convenience to justify payment. | Product owner | X–Y paid pilots; no revenue assumption in engineering gates. |

Initial feedback questions: Which two agents do you actually use together? What do you manually copy between them? Which permissions would you require before execution? Can you repeat the reference journey without assistance? What would you pay to have hosted, and why?

Recruit a small initial tester group during implementation. No user interviews, paid demand or provider access are claimed as completed by these documents.

## Exclusions

Model hosting, legacy IDE configuration migration, generic remote-device administration, and recreating providers' model/runtime infrastructure remain outside this product. New scope requires a recorded product decision and updated acceptance criteria.

The complete release requirements are in [Quality gates](plans/QUALITY_GATES.md). Early releases can advertise a verified subset; the complete 21-target milestone requires real evidence for all 21.
