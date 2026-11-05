# CLAUDE.md

This file provides contributor guidance for Claude Code in this repository.

## Project

This is a Go 1.26.5 backend Template Source and Runnable Default. One `cmd/app`
binary provides independently deployable `serve`, `worker`, and `scheduler`
roles, a combined `all` role, and `queue` and `schedule` operator commands. The
Runnable Default uses Fiber v3, GORM/MySQL, and the database-backed queue; Redis
with Asynq is an explicit alternative.

Read the authoritative guidance before changing shared behavior:

- [CONTEXT.md](CONTEXT.md) defines project and identity terminology.
- [AGENTS.md](AGENTS.md) defines Go and Clean Architecture operating rules.
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) defines boundaries, lifecycle,
  consistency, queue behavior, and extension seams.
- [docs/adr](docs/adr) records accepted architectural trade-offs.
- [docs/PATTERNS.md](docs/PATTERNS.md) covers implementation patterns.
- [docs/STYLE_GUIDE.md](docs/STYLE_GUIDE.md) covers formatting and the
  comment-free Go policy.
- [docs/LINTING.md](docs/LINTING.md) describes the quality gates.
- [README.md](README.md) covers setup, operations, and the HTTP API.

## Commands

```bash
make build
make run-server
make run-worker
make run-scheduler
make run-all
make fmt
make fmt-check
make vet
make lint
make test
make test-race
make test-integration
make vuln
make check
make template-pack
```

`make check` is the local CI gate. Integration tests require
`QUEUE_TEST_MYSQL_DSN` and `USER_TEST_MYSQL_DSN`. Run focused tests with `go
test ./path/to/package -run TestName`.

Migrations and operator commands use:

```bash
make migrate-create name=create_something
make migrate args=up
make migrate args='down 1'
make queue args=status
make schedule args=list
```

The local stack is started with `docker compose up -d --build`; Mailpit is
available at `http://localhost:8025`.

## Commit workflow

Use concise, imperative Conventional Commit subjects in the form
`type(scope): summary`, for example `docs(agents): require incremental commits`.
Keep each commit focused on one logical change. Use a commit body when it
explains the reason or important implementation details.

Before implementation, split each task into small, independently reviewable
commit checkpoints. Record checkpoints in an existing task plan when one
applies. A simple change with one checkpoint needs only one commit.

For each checkpoint:

1. Implement one logical change with its required tests and documentation.
2. Run the relevant checks and fix any failures.
3. Review and stage only files or hunks for that checkpoint. Exclude
   pre-existing and unrelated worktree changes.
4. Commit immediately after the checks pass, before starting the next
   checkpoint. Do not wait until the parent task, feature, or phase is complete.

Update the matching task checkbox or status only in the commit that completes
that task.

## Contributor Constraints

- Treat this repository as both the Template Source and a complete Runnable
  Default. Shared changes must remain meaningful in Generated Projects.
- Keep core packages independent of adapters, bootstrap wiring, configuration,
  and external SDKs. `depguard` enforces the import direction.
- Keep Go files free of human-written comments. Repository tests allow only Go
  directives, line directives, and the standard generated-code marker.
- When `queue.driver` is absent, it resolves to `database`. Redis must be
  selected explicitly with `QUEUE_DRIVER=redis`.
- Add an optional search integration at the `cmd/app/search.go` readiness seam;
  do not introduce a search SDK into core packages.
- Commands own resource creation, cancellation, graceful shutdown, and closure.
- Preserve registration rollback on verification-dispatch failure and
  best-effort welcome delivery after First Verification; see ADR 0002.
- MySQL queue and identity behavior require their integration suites in addition
  to ordinary unit and race tests.
- Template packaging reads `.craft/manifest.json` by default, archives only
  Git-tracked files, and must remain deterministic.
