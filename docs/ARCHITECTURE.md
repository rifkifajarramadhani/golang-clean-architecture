# Go Architecture Playbook

The application uses domain-first Clean Architecture. Source dependencies point
from commands and adapters toward core capabilities.

## Package Map

- `cmd/<app>` owns process lifecycle and wires dependencies.
- `internal/auth` owns authentication rules, refresh tokens, and required ports.
- `internal/user` owns the user entity, CRUD rules, and required ports.
- `internal/mail`, `internal/queue`, and `internal/scheduler` own framework-neutral capabilities.
- `internal/adapter/http` translates Fiber requests and responses.
- `internal/adapter/mysql`, `jwt`, `password`, `smtp`, `queue`, and `cron` implement core ports.
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

## Testing

- Core packages use deterministic fakes and table-driven tests.
- Adapters test translation, error mapping, cancellation, and resource handling.
- MySQL and queue behavior use integration tests.
- Commands remain thin and are validated through builds and smoke tests.
