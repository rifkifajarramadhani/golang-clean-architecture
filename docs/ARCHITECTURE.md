# Go Architecture Playbook

The application uses domain-first Clean Architecture. Source dependencies point
from commands and adapters toward core capabilities.

## Package Map

- `cmd/app` is a single multi-command binary owning process lifecycle and wiring dependencies. Each role is a subcommand: `serve` (HTTP API), `worker` (queue), `scheduler` (schedule ticker), `all` (all three in one process, sharing config, DB pool, and dispatcher), plus the `queue` and `schedule` operator CLIs. Deploy one subcommand per service to scale independently, or run `all` for small single-process deployments.
- `internal/auth` owns authentication rules, refresh tokens, and required ports.
- `internal/user` owns the user entity, CRUD rules, and required ports.
- `internal/mail`, `internal/queue`, and `internal/scheduler` own framework-neutral capabilities.
- `internal/adapter/http` translates Fiber requests and responses.
- `internal/adapter/<driver>` (e.g. `mysql`) owns the database connection, migrations, and shared driver helpers.
- `internal/adapter/<driver>/<domain>` (e.g. `mysql/identity`) implements core persistence ports with per-entity models and repositories.
- `internal/adapter/jwt`, `password`, `smtp`, `queue`, and `cron` implement other core ports.
- `internal/adapter/jobs` translates durable jobs into core service calls.
- `internal/bootstrap` assembles reusable adapter groups without owning process resources.
- `internal/config` loads and normalizes application configuration.

## Dependency Rules

- Core capabilities may depend on the standard library and other core capabilities.
- Core capabilities must not import `internal/adapter`, `internal/bootstrap`, or `internal/config`.
- Adapters may import core capabilities and third-party frameworks.
- Commands create and close loggers, database pools, queue clients, servers, and workers.
- Interfaces are defined by the package that consumes them and remain small.
- Contexts enter at commands, HTTP handlers, CLI commands, or job handlers and propagate through all I/O.

These rules are enforced by the `depguard` configuration in `.golangci.yml`.

## Data Flow

1. A command creates configuration, logging, and infrastructure resources.
2. An inbound adapter validates and translates an external request.
3. A core service applies application rules through core-owned ports.
4. Outbound adapters implement persistence, tokens, mail, queues, or scheduling.
5. The inbound adapter maps the result to the external representation.

## Persistence Adapters

Database adapters follow a two-level layout:

- **Driver root** (`internal/adapter/<driver>/`): connection pool, migrations, shared helpers. No domain business logic.
- **Domain subpackage** (`internal/adapter/<driver>/<domain>/`): repository implementations and per-entity persistence models.

Keep three model layers distinct:

- **Domain entity** — core package (`internal/<domain>/`); no ORM tags or driver imports.
- **Persistence model** — adapter subpackage; ORM/schema tags; maps to and from the domain entity inside the repository.
- **Transport DTO** — HTTP/CLI/job boundary only.

Swapping drivers (e.g. `mysql` to `postgres`) means adding `internal/adapter/<new-driver>/<domain>/` packages that implement the same core ports; core packages do not change.

## Adding a Domain

1. Add entity, service, and repository ports in `internal/<domain>/`.
2. Add `internal/adapter/<driver>/<domain>/` with one persistence-model file per entity and a repository.
3. Add migrations under the driver root (`internal/adapter/<driver>/migrations/`).
4. Register the repository constructor in `internal/bootstrap`.

Example: users and auth persistence live in `internal/adapter/mysql/identity`; a future `order` domain would add `internal/order` and `internal/adapter/mysql/order`.

## Testing

- Core packages use deterministic fakes and table-driven tests.
- Adapters test translation, error mapping, cancellation, and resource handling.
- MySQL and queue behavior use integration tests under driver domain subpackages (e.g. `internal/adapter/mysql/identity`).
- Commands remain thin and are validated through builds and smoke tests.
