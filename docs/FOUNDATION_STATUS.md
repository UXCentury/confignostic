# Foundation implementation: A and B

Prepared: 2026-10-26. Status: Done; merged in [PR #5](https://github.com/UXCentury/confignostic/pull/5). No operational agent integration or release qualification is claimed.

## Delivered scope

| Tasks | Result |
| --- | --- |
| A01–A04 | Product acceptance brief, stage mapping, explicit self-hosted/cloud boundary, owned assumptions and initial feedback questions. A deterministic Codex-to-Cursor reference fixture defines the bounded implementation/review task. |
| B01 | Go/Node/npm pins, exact frontend dependencies and lockfile, documented clean setup, and a digest-pinned development PostgreSQL Compose service. |
| B02 | Go formatting/vet/race tests/build, frontend Svelte/TypeScript checks/build, reference-fixture checks, and a database smoke job in CI. |
| B03 | Validated environment configuration, JSON operational logs and documented error/evidence conventions. Readiness is explicitly deferred until required dependencies exist. |
| B04 | API main waits for request draining. Stalled requests are force-closed after the configured deadline and cause an unsuccessful exit. Tests cover real HTTP requests, listener failure and early cancellation. |

## Local evidence

Environment: Linux x86-64; Go 1.27.3; Node.js 24.22.1; npm 11.9.0.

- `make check-go`: formatting, vet, race-enabled tests and API build passed.
- `make check-web`: clean lockfile install, Svelte/TypeScript checks (zero errors/warnings), and production build passed.
- `make check-reference`: unchanged candidate rejected; the bounded reference change accepted while retaining the original tests.
- Negative frontend probe: a temporary TypeScript file assigning a string to a number made the same frontend check fail. The probe was then removed.
- Built-binary smoke check: all three existing HTTP routes returned their expected responses, SIGTERM exited 0 after JSON drain/completion logs, and invalid configuration exited 1 without echoing the synthetic supplied value.

All three [GitHub CI jobs](https://github.com/UXCentury/confignostic/actions/runs/40342775285) passed for foundation commit fbb6b41e2f2e23d5f4cf267ffd1ee1ef31c4e7aa. The database job started the pinned Compose service and ran SELECT 1. PR #5 was merged as df8713d03dfa5705ff73dba2f6078a038166ceac. The implementation environment itself has no Docker daemon.

## Scope limits and next step

All 21 agent targets remain Planned. The fixture self-check is local test code, not a real provider run. No provider access, user interviews, paid demand, authentication, persistence or canvas behavior is implied by this change. User feedback collection remains ongoing work.

Stages A/B are accepted and their task checkboxes are complete. The next change implements [stage C contracts](CONTRACTS.md), including stateless graph validation. PostgreSQL migrations follow in D; the current API does not use the Compose database.

See [Contributing](../CONTRIBUTING.md), [Product acceptance](PRODUCT.md), [Reference fixture](../testdata/handoff/README.md), and [A–Z plan](plans/DEVELOPMENT_PLAN.md).
