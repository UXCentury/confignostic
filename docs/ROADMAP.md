# Product roadmap

This is the checkpoint summary for the [A–Z development plan](plans/DEVELOPMENT_PLAN.md): **26 stages and 104 implementation tasks**.

Stages A–C are Done; see [Foundation status](FOUNDATION_STATUS.md). Stage D PostgreSQL storage and dispatch foundations are implemented and In review; see [Storage](STORAGE.md). Stages E–Z remain Planned. The repository still has no working agent connector, chat execution or orchestration engine.

## Product destination

An open-source, self-hosted workspace with one chat for existing agents, typed handoffs, durable workflows, a shared visual canvas, approvals, tools, scheduling and collaboration. The catalog contains [21 named agent targets](SUPPORTED_AGENTS.md), plus generic ACP/A2A/custom adapters and MCP/X tool integrations.

The optional managed cloud service adds hosting, tenant operations and support. The complete self-hosted core does not require a cloud subscription.

## Checkpoints

| Checkpoint | Stages | Exit demonstration |
| --- | --- | --- |
| R0 — specification and foundation | A–C | Approved scope, reproducible builds and versioned contracts. |
| R1 — real one-agent private alpha | D–K after R0 | Codex runs from chat; identity, approvals, durable history and restart recovery work. |
| R2 — independent-agent handoff | L–O | Cursor joins through ACP; typed outputs pass between providers through a durable, chat-proposed workflow. |
| R3 — shared visual workflow beta | P and initial V packaging | Chat/canvas round-trip and a clean external installation work. |
| R4 — complete self-hosted release candidate | Q–V | Core connectors, additional ACP agents, remote services, tools, automation and collaboration are qualified or explicitly blocked. |
| R5 — self-hosted stable | W and Z-OSS | All advertised v1 capabilities pass the mandatory quality gates. |
| R6 — managed cloud pilot | X–Y | Tenant isolation, private runners, metering/billing and operational recovery are verified. |
| R7 — cloud stable | Z-CLOUD after Y | Pilot demand and service-readiness evidence support launch. |

Core implementation follows the stage prerequisites. Connector-expansion stages can be scheduled independently once those prerequisites pass. For one primary developer, keep runtime work sequential and use measured throughput to estimate each next checkpoint.

The complete 21-target milestone requires real evidence for all 21; earlier releases may advertise their verified subset. Missing provider access or protocol incompatibility remains a visible blocker.

## Start here

1. [A–Z plan](plans/DEVELOPMENT_PLAN.md): outcomes, dependencies, task checklists, deliverables and acceptance criteria.
2. [Execution guide](plans/EXECUTION_GUIDE.md): how to turn one task into a reviewable implementation change.
3. [Quality gates](plans/QUALITY_GATES.md): functional, security, recovery, usability, performance and operations evidence.
4. [Architecture](ARCHITECTURE.md): shared data model, runner boundaries and task lifecycle.
5. [Agent catalog](SUPPORTED_AGENTS.md): per-provider routes and status.

After stage D is accepted, the next implementation sequence is E01–E04 for authenticated bootstrap, sessions and workspace boundaries, followed by action/credential policy in F. The first two agents remain Codex and Cursor; broad connector coverage follows a proven handoff.

## Product validation and scope

Collect feedback from teams using multiple agent systems while the first reference workflow is built. Record useful repeat usage and willingness to pay before expanding managed-cloud commitments.

Preserve the full self-hosted scope. Keep legacy IDE migration, model hosting and generic device administration outside this plan. Add new scope through an explicit product/architecture decision and corresponding acceptance criteria.

Existing GitHub issues [#1](https://github.com/UXCentury/confignostic/issues/1) through [#4](https://github.com/UXCentury/confignostic/issues/4) remain early parent scopes; the master plan maps them to the expanded stages.
