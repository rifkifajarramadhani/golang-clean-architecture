# Go Architecture Playbook

The application uses domain-first Clean Architecture. Source dependencies point
from commands and adapters toward core capabilities. Shared terminology lives
in [CONTEXT.md](../CONTEXT.md).

## Template and Project Boundaries

This repository is both the Template Source and a complete Runnable Default.
The default selects Fiber, MySQL, and the database-backed queue so contributors
can run and validate a real application. A Generated Project contains its
Selected Stack and evolves independently. The distinction and its packaging
consequences are recorded in
[ADR 0001](adr/0001-runnable-default-as-template-source.md).

The source keeps optional integrations behind narrow composition seams. Search
readiness is the current example: `cmd/app/search.go` exposes the readiness
contract, while `openSearch` is the edge where a Selected Stack can open a
concrete search client and return its health check. Core packages do not depend
on a search SDK, and an unselected search adapter has no runtime cost.

## Package Map

- `cmd/app` owns process lifecycle and dependency wiring for `serve`, `worker`,
  `scheduler`, `all`, `queue`, and `schedule`.
- `internal/auth` owns authentication, verification, and refresh-token rules.
- `internal/user` owns the User entity, role rules, and CRUD use cases.
- `internal/mail`, `internal/queue`, and `internal/scheduler` own
  framework-neutral capabilities.
- `internal/adapter/http` translates Fiber requests and responses.
- `internal/adapter/<driver>` owns a database connection, migrations, and shared
  driver helpers.
- `internal/adapter/<driver>/<domain>` implements core persistence ports with
  per-entity models and repositories.
- `internal/adapter/jwt`, `password`, `smtp`, `queue`, and `cron` implement other
  core ports.
- `internal/adapter/jobs` translates durable jobs into core service calls.
- `internal/bootstrap` assembles adapter groups without owning process resources.
- `internal/config` loads and normalizes application configuration.

## Dependency Rules

- Core capabilities may depend on the standard library and other core
  capabilities.
- Core capabilities must not import `internal/adapter`, `internal/bootstrap`, or
  `internal/config`.
- Adapters may import core capabilities and third-party frameworks.
- Commands create and close loggers, database pools, queue clients, servers, and
  workers.
- Interfaces are defined by the package that consumes them and remain small.
- Contexts enter at commands, HTTP handlers, CLI commands, or job handlers and
  propagate through all I/O.

The `depguard` configuration in `.golangci.yml` enforces the core import rules.

## Data Flow

1. A command creates configuration, logging, and infrastructure resources.
2. An inbound adapter validates and translates an external request.
3. A core service applies application rules through core-owned ports.
4. Outbound adapters implement persistence, tokens, mail, queues, or scheduling.
5. The inbound adapter maps the result to the external representation.

## Process Lifecycle

Each long-running command derives a context from termination signals and passes
it through every owned component. The server converts cancellation into bounded
graceful shutdown. Workers stop reservations, cancel active handler contexts,
and wait up to the configured shutdown period. The scheduler stops its timer on
cancellation. The `all` command shares one context, database pool, and
dispatcher; when any component returns, it cancels the others, waits for all of
them, then reports the first component error. Resource closure remains in the
command that opened the resource.

## Queue Behavior

The database-backed queue is selected when `queue.driver` is omitted. Redis can
be selected explicitly. Switching drivers does not transfer in-flight jobs.

Database workers traverse the configured weighted queue sequence. They sleep
only after every weighted slot is empty, so an empty queue does not delay ready
work in another queue by a full polling interval. Reservation errors always
back off to prevent a hot loop. Active jobs extend their reservation lease while
their handler runs and use the process context plus any job timeout.

Cleanup is throttled across worker goroutines rather than executed on every
reservation attempt. It removes expired locks, then deletes bounded batches of
expired completed jobs and their non-expiring task locks. This keeps polling
cost predictable while preserving retry, uniqueness, retention, and task-ID
semantics.

## Identity Consistency and Delivery

Core services perform fast business-rule checks. Persistence is authoritative
for rules exposed to concurrency: User role changes and deletion re-check actor
authority, verification state, and last-admin protection while holding database
row locks. Email verification chooses any Bootstrap Admin role while its
consistency locks are held.

Login always performs a password-hash comparison, including when no User is
found, which reduces response-timing differences between known and unknown
emails. Logout treats an empty, unknown, or already-revoked refresh token as
success, keeping the operation idempotent without revealing token validity.

Registration, verification, and delivery boundaries are recorded in
[ADR 0002](adr/0002-registration-verification-delivery.md). Registration rolls
back User creation when verification dispatch fails, while acknowledging that
an external dispatch cannot be rolled back if the later database commit fails.
After verification commits, welcome delivery is best-effort and occurs only for
First Verification.

## Persistence Adapters

Database adapters use a two-level layout:

- **Driver root** (`internal/adapter/<driver>/`): connection pool, migrations,
  and shared helpers; no domain business logic.
- **Domain subpackage** (`internal/adapter/<driver>/<domain>/`): repository
  implementations and per-entity persistence models.

Keep the domain entity, persistence model, and transport DTO distinct. Swapping
drivers means adding a new adapter subpackage that implements the same core
ports; core packages do not change.

## Adding a Domain

1. Add the entity, service, and repository ports in `internal/<domain>`.
2. Add `internal/adapter/<driver>/<domain>` with persistence models, mappers,
   and a repository.
3. Add migrations under the driver root.
4. Register the repository constructor in `internal/bootstrap`.

## Testing

- Core packages use deterministic fakes and table-driven tests.
- Adapters test translation, error mapping, cancellation, and resource handling.
- MySQL queue and identity behavior use opt-in integration tests in their driver
  packages.
- Commands remain thin and are validated through builds and smoke tests.
- Repository policy tests reject human-written Go comments and verify template
  archive contents, determinism, manifest versioning, and checksums.
