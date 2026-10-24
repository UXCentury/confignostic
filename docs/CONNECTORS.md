# Agent connectors

The [supported-agent catalog](SUPPORTED_AGENTS.md) is the source of truth for planned targets and implementation status. All entries are currently **Planned**; a catalog entry must not be returned by the API as an installed or connected agent.

## Integration routes

- **ACP:** a shared client in a runner connects compatible coding agents. Validate each provider's authentication, protocol version, permissions and session behavior.
- **A2A:** connect remote agents using agent discovery and the capabilities advertised by the endpoint.
- **Native SDK or provider API:** use the provider's supported execution interface when it offers the required functionality. Keep Node.js/Python SDK processes behind runner boundaries where needed.
- **Custom HTTP adapter:** implement the same task and capability contract for user-provided agents.
- **MCP and X platform tools:** tool access for agents, tracked separately from agent lifecycle connectors.

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

Every connector needs conformance tests for discovery, execution result, failure, idempotent retry, authorization and token redaction. ACP, A2A, provider SDKs, HTTP and MCP are not interchangeable transports for complete agent control.
