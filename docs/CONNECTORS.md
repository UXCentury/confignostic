# Agent connectors

Connecting an agent does not automatically give access to its existing GUI sessions. Each connector must publish its **actually implemented** capability flags:

`invoke`, `stream_events`, `list_sessions`, `cancel`, `resume`, `artifacts`, `cost`.

An adapter must never fake support for unsupported actions.

## Task lifecycle

```text
queued -> awaiting_approval? -> dispatched -> running
                                         |-> completed
                                         |-> failed
                                         |-> canceled
                                         |-> unknown
```

Unknown means remote execution may have succeeded without confirmation.

## Conceptual task envelope

```json
{
  "task_id": "uuid",
  "workflow_run_id": "uuid",
  "adapter_id": "string",
  "input": {"message": "string"},
  "idempotency_key": "string",
  "status": "queued",
  "trace_id": "string"
}
```

Every connector needs conformance tests for discovery, execution result, failure, idempotent retry, authorization and token redaction. A2A, provider SDKs, HTTP and MCP are not interchangeable transports for complete agent control.
