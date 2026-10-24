# Supported agents and integration roadmap

This is the support target for Confignostic's open-source, self-hosted chat and orchestration core.

> **Implementation status — 2026-10-24:** All 21 named agent targets below are **Planned**. The current scaffold has no operational agent connectors. Listing an upstream SDK, API, or protocol establishes a possible integration route; it does not mean Confignostic has implemented or tested that integration.

## Coding agents and runtimes

| Agent | Planned integration route | Confignostic status |
| --- | --- | --- |
| Cursor | [Cursor CLI over ACP](https://cursor.com/docs/cli/acp), with a runner providing the workspace and connection. | Planned |
| OpenAI Codex | [App Server](https://developers.openai.com/codex/app-server) for interactive sessions and approvals; [TypeScript or Python SDK](https://developers.openai.com/codex/codex-sdk) for programmatic execution. | Planned |
| Claude Code / Claude Agent SDK | [Official Python or TypeScript Agent SDK](https://code.claude.com/docs/en/agent-sdk/overview) in a runner. | Planned |
| GitHub Copilot | [Copilot SDK](https://github.com/github/copilot-sdk), including a native Go client for the Copilot CLI runtime. | Planned |
| OpenCode | [JavaScript/TypeScript SDK](https://docs.opencode.ai/docs/sdk/) or [HTTP server API](https://docs.opencode.ai/docs/server/). | Planned |
| OpenHands | [Software Agent SDK and Agent Server](https://docs.openhands.dev/overview/introduction), using Python or REST/WebSocket interfaces. | Planned |
| Grok Build | [ACP or headless CLI](https://docs.x.ai/build/overview), with structured streaming output where available. | Planned |

Each connection must identify the actual runtime, account and workspace it controls. Access to an SDK or CLI does not automatically grant access to every session in that provider's desktop application.

## Additional agents through ACP

The [official ACP agent directory](https://agentclientprotocol.com/get-started/agents) lists these additional targets. We plan to reuse the ACP connector, while validating authentication, protocol version, permissions and session behavior separately for each agent.

| Agent | Planned integration route | Confignostic status |
| --- | --- | --- |
| Gemini CLI | ACP through a configured runner. | Planned |
| Goose | ACP through a configured runner. | Planned |
| Cline | ACP through a configured runner. | Planned |
| Factory Droid | ACP through a configured runner. | Planned |
| JetBrains Junie | ACP through a configured runner. | Planned |
| Kimi CLI | ACP through a configured runner. | Planned |
| Kiro CLI | ACP through a configured runner. | Planned |
| Mistral Vibe | ACP through a configured runner. | Planned |
| Qwen Code | ACP through a configured runner. | Planned |

An upstream directory listing is not a Confignostic compatibility test. Record the tested agent and protocol versions when each adapter becomes available.

## Stateful agents and customer-built agent services

These targets connect agents the user has configured or deployed. The model API entry requires a configured agent loop and tools; the SDK alone does not provide a coding workspace.

| Agent system | Planned integration route | Confignostic status |
| --- | --- | --- |
| Letta agents | [TypeScript Agent SDK](https://www.letta.com/agent-sdk/) with a [local, self-hosted or managed backend](https://docs.letta.com/agent-sdk/deployment). | Planned |
| LangGraph agents | [Python](https://reference.langchain.com/python/langgraph-sdk) or [TypeScript](https://github.com/langchain-ai/langgraphjs/blob/main/libs/sdk/docs/README.md) client SDK against the user's deployed agent server. | Planned |
| Google ADK agents | [A2A endpoints](https://adk.dev/a2a/) exposed by customer-deployed agents. | Planned |
| Microsoft Agent Framework agents | [A2A endpoints](https://learn.microsoft.com/en-us/agent-framework/hosting/self-hosting/a2a/server) exposed by customer-deployed agents. | Planned |
| Grok API-powered agents | [Official xAI Python SDK](https://github.com/xai-org/xai-sdk-python) or [HTTP API](https://docs.x.ai/overview), with a configured agent and [tools](https://docs.x.ai/developers/tools/overview) for tasks such as web and X research. | Planned |

Grok Build and Grok API-powered agents are separate targets: one connects the existing coding runtime; the other integrates a configured agent using the Grok API.

## Generic connection routes

| Route | Planned role | Confignostic status |
| --- | --- | --- |
| ACP — Agent Client Protocol | Connect compatible coding agents through runners, initially using the [stdio transport](https://agentclientprotocol.com/protocol/v1/transports). | Planned |
| A2A — Agent2Agent | Discover and invoke user-registered remote agents through their advertised agent cards and capabilities. | Planned |
| Custom HTTP / SDK adapter | Connect a user-provided agent that implements the [connector contract](CONNECTORS.md). | Planned |

ACP and A2A are protocols, not additional agent products. Protocol support must be versioned, and each connected agent must advertise only the operations it actually implements.

## Tools and platform integrations

| Integration | Planned role | Confignostic status |
| --- | --- | --- |
| MCP tools | Give a connected agent access to supported external tools and data through its runtime. | Planned |
| X platform | Use [X API SDKs or APIs](https://docs.x.com/tools-and-libraries) for authorized post and account operations; use [XMCP](https://docs.x.com/what-to-build) where appropriate. | Planned |

X platform access is a tool integration. It is separate from Grok Build and from Grok's X Search tool. Account-changing actions must follow Confignostic's approval policy.

## Status and capability policy

- **Planned:** a target is listed, but there is no verified Confignostic connector.
- **Experimental:** an implementation exists with documented limitations and tested upstream versions.
- **Supported:** the documented operations pass connector conformance checks and a real end-to-end task through Confignostic.

Before promoting an entry, record its connector ID, supported versions, authentication setup, execution location and limitations. Verify invocation, output/event handling, failure reporting, approvals and secret redaction. Test cancellation, resume, artifacts and cost reporting only when those capabilities are actually offered; expose unavailable capabilities explicitly.

Self-hosting covers Confignostic and the configured runners. Upstream agent services retain their own authentication, deployment and usage requirements.

Implement the catalog incrementally using the [roadmap](ROADMAP.md): complete one real chat-to-agent task, persist its state, prove a second-provider handoff, and then expand connector coverage and the shared canvas.
