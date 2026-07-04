# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Backend services boilerplate built with Clean Architecture principles: independently deployable HTTP API, queue worker, and scheduler processes, plus `queue`/`schedule` CLIs. Go 1.26, Fiber v3, GORM/MySQL, Redis or MySQL-backed queue (Asynq), Viper config, golang-migrate, Air hot reload.

Detailed agent guidance already lives in these files — read them, don't duplicate them here:

- [AGENTS.md](AGENTS.md) — Go style rules pulled from Effective Go, Uber's style guide, Go Code Review Comments, and the Clean Architecture essay.
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — package map, dependency rules, data flow.
- [docs/PATTERNS.md](docs/PATTERNS.md) — common implementation patterns.
- [docs/STYLE_GUIDE.md](docs/STYLE_GUIDE.md) — idioms and formatting.
- [docs/LINTING.md](docs/LINTING.md) — tooling and lint checklist.
- [README.md](README.md) — full setup, API endpoints, troubleshooting.

## Commands

```bash
make build              # build all binaries (server, worker, scheduler, queue, schedule)
make run-server          # go run ./cmd/server
make run-worker          # go run ./cmd/worker
make run-scheduler       # go run ./cmd/scheduler
make fmt                 # goimports + gofmt -w
make fmt-check           # verify formatting without writing
make vet                 # go vet ./...
make lint                # golangci-lint run
make test                # go test ./...
make test-race           # go test -race ./...
make test-integration    # MySQL queue + user-security integration tests (needs QUEUE_TEST_MYSQL_DSN / USER_TEST_MYSQL_DSN)
make vuln                # govulncheck ./...
make check               # fmt-check + vet + lint + test + test-race (the CI gate — run before considering work done)
```

Run a single test:

```bash
go test ./internal/user/... -run TestServiceName
go test ./internal/user/... -run TestServiceName/subtest_name -v
```

Migrations (require the `server` container, via `docker compose exec`):

```bash
make migrate-create name=create_something
make migrate args=up
make migrate args='down 1'
```

Queue/schedule CLIs:

```bash
make queue args=status
make schedule args=list
```

Local dev stack (server + worker + scheduler + MySQL + Redis + Mailpit): `docker compose up -d --build`. Mailpit UI at http://localhost:8025.

## Architecture

Domain-first Clean Architecture; dependencies point from `cmd/` and adapters inward toward core packages, never the reverse. This is enforced by `depguard` in [.golangci.yml](.golangci.yml): `internal/auth`, `internal/user`, `internal/mail`, `internal/queue`, `internal/scheduler` (the "core" packages) must not import `internal/adapter`, `internal/bootstrap`, or `internal/config`.

- `cmd/<app>` (server, worker, scheduler, queue, schedule) — owns process lifecycle, creates config/logging/DB/queue resources, and wires dependencies. Kept thin.
- `internal/auth`, `internal/user` — core entities, use-case rules, and the ports (interfaces) adapters must implement. Define interfaces at the consumption site.
- `internal/clock` — wall-clock boundary (`Clock` interface, `Real`, `Func`) so core packages take time as a dependency instead of calling `time.Now()`.
- `internal/mail`, `internal/queue`, `internal/scheduler` — framework-neutral capabilities (mailables, job/dispatcher abstractions, cron registry) usable independently of any transport.
- `internal/adapter/http` — Fiber handlers, DTOs, middleware, router. Translates HTTP <-> core service calls.
- `internal/adapter/mysql`, `jwt`, `password`, `smtp`, `queue`, `cron`, `logging` — concrete implementations of core ports (GORM repositories, JWT signing, bcrypt, SMTP transport, Asynq/DB queue backends, cron parsing, slog setup).
- `internal/adapter/jobs` — translates durable queue jobs into calls against core services (e.g. `jobs.RegisterHandlers`, `jobs.CleanupRefreshTokens`).
- `internal/bootstrap` — reusable dependency-assembly functions (`WireHTTPServices`, `Dispatcher`, `Inspector`, `Worker`, `ScheduleRegistry`) shared across `cmd/*`; does not own process resources (no opening/closing DB pools itself).
- `internal/config` — Viper-backed config loading/normalization (`configs/config.yaml` + env var overrides).

Data flow: a `cmd/` entrypoint builds config/logging/infra -> an inbound adapter (HTTP handler, job handler, CLI command) validates/translates the external request -> a core service applies business rules via ports it defines -> outbound adapters (MySQL, JWT, SMTP, queue, cron) implement those ports -> the inbound adapter maps the result back out.

Queue backend is pluggable at runtime via `QUEUE_DRIVER` (`redis` default, or `database`, requiring migration `000003_create_queue_tables`); `internal/bootstrap` selects the concrete adapter based on `cfg.Queue.Driver`. Switching drivers does not migrate in-flight jobs between backends.

Auth details worth knowing before touching that code: refresh tokens are persisted as SHA-256 hashes; login requires a verified email; the first verified user matching `AUTH_BOOTSTRAP_ADMIN_EMAIL` becomes admin only while no admin exists yet; non-development/non-test environments require distinct JWT secrets of at least 32 bytes.

## Testing conventions

- Core packages (`internal/auth`, `internal/user`, etc.) use deterministic fakes and table-driven tests — no real DB/network.
- Adapters test translation, error mapping, cancellation, and resource handling.
- MySQL and queue behavior is covered by integration tests gated behind `*_TEST_MYSQL_DSN` env vars (see `make test-integration`); these aren't run by plain `make test`.
- `cmd/` packages stay thin and are validated by build + smoke testing rather than unit tests.
