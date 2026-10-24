# Architecture

Prefer a **modular Go monolith** and independent runners over an early microservice fleet.

```text
Svelte UI: chat / workflow canvas / execution timeline
             |
             v
Go API: identity, policy, agent registry, task coordination,
        workflow DAG validation, audit, artifact references
             |
             v
PostgreSQL: durable chat, workflows, tasks, events
             |
             v
Authenticated runner connection
     |-- Customer-hosted runners
     |-- Cloud runners
             |-- A2A adapters
             |-- Provider APIs
             |-- Custom HTTP agents
```

## Contracts
The **control plane** owns task state and approvals. **Runners** execute approved actions in isolation. **Connectors** expose provider-specific capabilities truthfully. **Chat and canvas** manipulate the same versioned workflow graph.

## Reliability and safety
- Stable task IDs, idempotency keys, typed inputs and outputs.
- Persist task transitions and event history to survive restart and reconnection.
- Represent an uncertain external side effect as `unknown`, not `completed`.
- Prevent secret values in logs, transcripts or workflow exports.
- Planners must not approve their own privileged actions.
- Cancellation and resume can only be offered when the connector actually supports them.
- Treat agent output as untrusted data unless validated and authorized.

## Order of implementation
Real single agent → durable tasks and chat → second provider and typed handoff → visual editor → managed cloud.

Redis is optional until a measured queue or caching requirement appears.
