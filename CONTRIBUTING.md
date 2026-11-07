# Contributing

Confignostic is an early-stage project. Follow the [execution guide](docs/plans/EXECUTION_GUIDE.md), keep changes scoped to stable task IDs, and preserve truthful capability/status reporting. No real agent connection exists yet.

## Reproducible toolchain

| Tool | Pinned source | Current development baseline |
| --- | --- | --- |
| Go | go.mod, also read by CI | 1.27.3 |
| Node.js | .nvmrc, also read by CI | 24.22.1 |
| npm | web/package.json packageManager and engines | 11.9.0 |
| Frontend packages | Exact direct versions and web/package-lock.json | Install with npm ci |
| Development PostgreSQL | compose.dev.yaml image digest | PostgreSQL 18, Bookworm |

Use Linux x86-64 for the initial validated development/CI environment. Go race checks need a C compiler; the Makefile needs GNU Make and a POSIX shell. Other platforms are not yet qualified. Docker Engine and Compose v2 with `up --wait` are required only for the optional development database.

Install Go from [go.dev](https://go.dev/dl/). With nvm installed:

```sh
nvm install
nvm use
npm install --global npm@11.9.0
npm --prefix web ci
```

Do not use force or legacy-peer-deps to hide dependency conflicts. Update exact direct versions and regenerate the lockfile in a focused dependency change. Update .nvmrc, engines and CI together if their compatibility changes. Review pinned action commits and image digests when updating them.

## Run locally

From the repository root, in separate terminals:

```sh
make dev-api
make dev-web
```

The API defaults to 127.0.0.1:8080; Vite defaults to 127.0.0.1:5173 and proxies API requests. The UI is still a disabled scaffold. The two listing endpoints return empty arrays. A stateless POST `/api/v1/workflows/validate` endpoint checks graph structure; see [Contracts](docs/CONTRACTS.md). The development API exposes listings and stateless validation without authentication; keep the loopback default until identity and workspace authorization are implemented.

## Checks and builds

```sh
make check
make build
```

`make check` runs Go formatting checks, module verification, vet, race-enabled tests, an API build, a clean frontend install, Svelte/TypeScript checks, a production frontend build, and the reference-fixture self-checks. Individual targets: `make check-go`, `make check-web`, `make check-reference`. Fix Go formatting with `gofmt -w cmd internal`.

Build outputs are bin/confignostic-api and web/dist/. Run `npm --prefix web ci` before `make build` on a fresh checkout. No global Vite/Svelte installation is needed. The database migration binary is bin/confignostic-db. See [Storage](docs/STORAGE.md) for migration/readiness setup and `make check-storage` against a disposable PostgreSQL server. The API does not serve the built frontend yet; unified packaging belongs to stage V.

The reference checks deliberately test a failing candidate and a passing bounded change. They do not invoke Codex, Cursor or any external provider. See [the fixture walkthrough](testdata/handoff/README.md).

CI runs the same Go/frontend/fixture commands and a separate real-PostgreSQL durability suite, including migration, fencing, process interruption and backup/restore checks. It uses read-only repository permissions, does not retain checkout credentials, and uses no provider credentials. Do not add production secrets or execute untrusted PR code with privileged pull_request_target workflows.

## Optional development database

The database backs the internal stage D repositories. Export CONFIGNOSTIC_DATABASE_URL and run `make db-migrate` to enable API storage readiness; [Storage](docs/STORAGE.md) documents the full setup and durability tests.

```sh
cp .env.example .env
# Edit .env and choose a disposable local-only database password.
make db-up
docker compose -f compose.dev.yaml exec -T postgres psql -U confignostic -d confignostic_dev -c 'SELECT 1;'
make db-down
```

Compose binds PostgreSQL to 127.0.0.1:5432. Set CONFIGNOSTIC_DEV_DB_PORT in .env if that port is already used. The database/user are confignostic_dev/confignostic. The named volume preserves development data across stop/start. PostgreSQL's initialization password applies when the volume is first created; editing .env does not rotate an existing database password.

To deliberately delete only this Compose project's disposable database data:

```sh
docker compose -f compose.dev.yaml down --volumes
```

The Compose database is a development convenience, not production installation or backup tooling. Never use production data or credentials in it. Changing project names/directories can create a separate Compose volume.

## API configuration and shutdown

Only the process environment configures the Go API. It does not automatically read .env; that file is for Compose.

| Setting | Default | Validation |
| --- | --- | --- |
| CONFIGNOSTIC_LISTEN_ADDR | 127.0.0.1:8080 | host:port; numeric port 0–65535. Port 0 selects a free local port, reported in startup logs. |
| CONFIGNOSTIC_SHUTDOWN_TIMEOUT | 10s | Positive Go duration such as 250ms or 15s. |
| CONFIGNOSTIC_DATABASE_URL | Unset; stateless scaffold | PostgreSQL URL with a database name; when set, connectivity and compatible migrations are required at startup. |

Explicit empty or invalid values fail before binding. Configuration errors name the setting without echoing its raw value. Read headers have a five-second deadline, idle keep-alive connections a sixty-second deadline, and headers a 32 KiB limit.

SIGINT/SIGTERM stops admission and waits for active HTTP requests. If the drain deadline expires, the server closes remaining connections and exits unsuccessfully. It does not report a clean shutdown after truncating requests. These semantics currently apply to HTTP requests; future durable workers and long-lived streams need their own coordinated shutdown behavior.

Logs are JSON on stderr with time, level and msg plus bounded operational fields. Never log credentials, authorization headers, request bodies or provider prompts. Add workspace/task/attempt correlation identifiers when their domain types exist. Use the stage C versioned public error contract. Database startup/migration logs are deliberately generic; never log a raw driver error that could contain a credential or SQL value.

GET /healthz reports process liveness only. GET /readyz checks configured database connectivity, write mode and migration compatibility. It returns 503 when storage is absent, unavailable, read-only or incompatible; it does not claim that authentication or agent execution is operational.

## Review and evidence

Include the task IDs, user-visible behavior, meaningful validation and relevant limitations in each PR. Runtime tests should prove behavior and failure boundaries. Documentation-only changes need content/link review.

Record architecture choices in docs/decisions/ when needed, provider compatibility evidence in docs/compatibility/ when real integrations exist, and release qualification in docs/releases/. Keep live credentials and private customer data out of both repository and CI artifacts.
