# Confignostic

**One chat to command your AI agents. One canvas to orchestrate them.**

Confignostic is an open-source control center for connecting and coordinating **existing AI agent systems**. Talk to multiple agents through a unified chat, connect their work through visual workflows, and inspect execution across providers.

> **Early-stage project:** This repository contains an initial Go API, a Svelte UI scaffold, architecture documentation, and automated tests. Chat execution, provider integrations, visual workflow editing, orchestration, cloud accounts and billing are **not implemented yet**.

## Core product

- **Unified chat** — interact with an agent directly or ask an orchestrator to delegate.
- **Bring your own agents** — integrate independent agents via ACP, A2A, provider APIs, or custom adapters according to their actual capabilities.
- **Visual workflow editor** — design handoffs, dependencies, parallel steps, approvals, and error handling.
- **Execution transparency** — see real task states, outputs, artifacts, and failures.
- **Self-host or cloud** — open-source local installation, with a future managed Confignostic Cloud service and optional private runners.

## Agent support targets

The [supported-agent catalog](docs/SUPPORTED_AGENTS.md) tracks all 21 planned agent targets, their integration routes and implementation status.

- **Coding agents:** Cursor, OpenAI Codex, Claude Code / Claude Agent SDK, GitHub Copilot, OpenCode, OpenHands and Grok Build.
- **Additional ACP agents:** Gemini CLI, Goose, Cline, Factory Droid, JetBrains Junie, Kimi CLI, Kiro CLI, Mistral Vibe and Qwen Code.
- **Stateful and custom agent systems:** Letta, LangGraph, Google ADK, Microsoft Agent Framework and Grok API-powered agents.
- **Connection routes and tools:** ACP, A2A, custom HTTP/SDK adapters, MCP tools and X platform integration.

**All entries are planned.** The current scaffold has no operational connectors. Provider capabilities and tested versions will be recorded as each integration is implemented.

## Run the development scaffold

Requires Go 1.23+ and Node.js 20+.

```sh
go run ./cmd/confignostic-api
# in another terminal:
cd web && npm install && npm run dev
```

The Go API binds to 127.0.0.1:8080 by default and provides `/healthz`, `/api/v1/agents`, and `/api/v1/workflows`. The latter two return empty lists: no agent integration exists yet.

## Project layout

- `cmd/confignostic-api/` — Go API entry point
- `internal/httpapi/` — HTTP routes and tests
- `web/` — Svelte/Vite interface scaffold
- `docs/` — architecture, product scope, connectors and roadmap

## Principles

1. **Connect existing agents.** Don't reinvent their runtimes.
2. **One execution model.** Chat and visual canvas should describe and monitor the same workflows.
3. **Honest capabilities.** Don't promise cancellation, session control or telemetry that a provider does not expose.
4. **Safe by default.** Require explicit permission for high-impact operations and isolate runners.
5. **Durable results.** Persist task state and tolerate retries, disconnects, and restarts.
6. **Focused scope.** Avoid the legacy project's infrastructure management, IDE migration, and model-hosting ambitions.

## Open source

Apache License 2.0. The planned cloud product sells managed hosting, collaboration, reliability and support; the core self-hosted chat and orchestration should remain available without a cloud subscription.

See [Architecture](docs/ARCHITECTURE.md), [Agent connectors](docs/CONNECTORS.md), [Supported-agent catalog](docs/SUPPORTED_AGENTS.md), [Product scope](docs/PRODUCT.md), [Roadmap](docs/ROADMAP.md), [Contributing](CONTRIBUTING.md), and [Security](SECURITY.md).

## Previous project

The original Confignostic engineering control-plane project is preserved with its Git history in the private, archived repository [UXCentury/confignostic-legacy](https://github.com/UXCentury/confignostic-legacy). This is a new repository with an independent history and product scope.
