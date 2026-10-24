# Agent connectors

## Status and scope

The [supported-agent catalog](SUPPORTED_AGENTS.md) records targets and implementation status. All entries are currently **Planned**. A catalog entry must not be returned by the API as an installed or connected agent.

The [A–Z plan](plans/DEVELOPMENT_PLAN.md) introduces the shared contract in C/H, the first real connectors in I/L, and expanded coverage in Q/R/S/T. This document describes the target contract; it is not a claim that the API currently implements it.

## Integration routes

| Route | Responsibility |
| --- | --- |
| ACP | A runner-side client connects compatible coding agents; validate provider authentication, version, permission/input requests and session behavior. |
| A2A | Discover and invoke remote agents using supported protocol versions and endpoint capabilities. |
| Native SDK or provider API | Use the provider's execution interface; supervise Node.js/Python SDK processes in runners where needed. |
| Custom HTTP adapter | Implement the same task, policy and event semantics for user-provided agents. |
| MCP / X platform | Tool access for an agent/runtime, tracked separately from agent lifecycle connections. |

Connecting a CLI, SDK or server does not automatically grant access to every existing desktop/GUI session. Record the exact runtime, account and workspace a connection controls.

## Capability descriptor

A connection reports its tested adapter/version, execution location, authentication status, health and optional operations:

- invoke and streaming events;
- list/create/load sessions where supported;
- cancellation, resume and external-state reconciliation;
- artifacts, structured output and usage/cost;
- approval requests and user-input requests;
- idempotent invocation or an operation lookup mechanism, with its actual scope and limitations.

The final field names/schema are defined in stage C. A capability must describe the combination of adapter, provider version and account configuration. Report Unsupported, Unauthorized and Unavailable separately. Missing usage is unknown, not zero.

## Task and event boundary

The canonical planned task-state table is in [Architecture](ARCHITECTURE.md#planned-task-lifecycle). Preserve distinct logical task, attempt, dispatch and provider session/operation IDs.

Conceptual envelope to formalize in C:

```json
{
  "schema_version": "1",
  "workspace_id": "uuid",
  "workflow_run_id": "uuid",
  "task_id": "uuid",
  "attempt_id": "uuid",
  "dispatch_id": "uuid",
  "connection_id": "uuid",
  "input": { "message": "string", "artifact_refs": [] },
  "idempotency_key": "string",
  "policy_version": "string",
  "approval_refs": [],
  "trace_id": "string"
}
```

Workspace identity and approval references are supplied and validated by the control plane, not trusted from a provider. Credentials travel through scoped secret resolution, not this envelope.

Normalized events carry type/schema version, internal identifiers, provider event/reference when available, sequence/replay identity, timestamp and a bounded payload. Preserve original source IDs for diagnostics while omitting credentials.

The adapter must not acknowledge durable results before persistence, turn a lost connection into success, or retry uncertain external side effects blindly. Store enough identity to reconcile. A cancel request is not a confirmed canceled result.

## Authorization and isolation

Run only within the registered workspace, policy and credential scope. The planner cannot approve its own privileged actions. Forward provider approval/input requests with their real granularity; explicitly handle unsupported extensions instead of leaving a hidden wait.

Validate discovery URLs, redirects and artifact downloads through the endpoint/egress policy. Private services require explicit scoped access. Execute local runtime tools within the supported runner sandbox profile.

## Qualification and delivery

Use the [connector qualification record](plans/QUALITY_GATES.md#connector-qualification-record) before changing an entry to Supported.

Each integration needs meaningful tests for discovery, actual task execution, failure, denial, secret redaction, replay/idempotency and reconnect. Test optional cancel/resume/artifact/usage behavior when claimed. Deterministic fakes support CI; a real agent task is also required.

Delivery stages:

- I: Codex.
- L: reusable ACP client and Cursor.
- Q: Claude, Copilot, OpenCode, OpenHands and Grok Build.
- R: A2A/custom services, ADK, Microsoft Agent Framework, LangGraph and Letta.
- S: the nine additional ACP catalog targets.
- T: configured Grok API agents, MCP tools and X integration.

ACP, A2A, native SDKs and MCP have distinct roles. Protocol compatibility does not eliminate provider-specific qualification or imply full remote session control.
