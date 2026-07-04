# Golang Clean Architecture

Backend services boilerplate using Clean Architecture principles with independently deployable HTTP, queue worker, and scheduler processes.

## Tech Stack

- **Language:** Go 1.26
- **HTTP Framework:** Fiber v3
- **Database:** MySQL 8 (Docker)
- **Queue:** Configurable MySQL or Redis + Asynq
- **Scheduler:** Application-defined cron schedules
- **Mail:** Laravel-inspired mailables with SMTP and queued delivery
- **ORM:** GORM
- **Config:** Viper (`configs/config.yaml`)
- **Migrations:** golang-migrate
- **Hot Reload (Dev):** Air
- **Containerization:** Docker + Docker Compose

## Project Structure

```text
.
├── cmd/
│   ├── server/                 # API entrypoint
│   ├── worker/                 # Queue worker service
│   ├── scheduler/              # Long-running scheduler service
│   ├── queue/                  # Queue operations CLI
│   └── schedule/               # Schedule operations CLI
├── configs/
│   └── config.yaml             # App + DB config
├── internal/
│   ├── config/                 # Config loader
│   ├── auth/                   # Authentication core and ports
│   ├── user/                   # User core and ports
│   ├── mail, queue, scheduler/ # Framework-neutral capabilities
│   ├── adapter/                # HTTP, MySQL, JWT, SMTP, queue, cron, logging
│   ├── bootstrap/              # Reusable dependency assembly
│   └── config/                 # Config loader
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── .air.toml
```

## Prerequisites

- Docker
- Docker Compose
- Make (optional, for migration shortcuts)

## Configuration

Non-sensitive defaults live in `configs/config.yaml`. Credentials are supplied
through environment variables. For local development, create an ignored `.env`
file from the safe template:

```bash
cp .env.example .env
```

The application loads `.env` when present without overriding variables already
set by the shell or container platform.

Default values in this project:

- App port: `8080`
- DB host: `db`
- DB port: `3306`
- DB user: `root`
- DB password: configured with `DATABASE_PASSWORD`
- DB name: `db_name`
- JWT access secret: configured with `AUTH_JWT_ACCESS_SECRET`
- JWT refresh secret: configured with `AUTH_JWT_REFRESH_SECRET`
- Access token TTL: `15` minutes
- Refresh token TTL: `168` hours
- Email verification TTL: `24` hours
- Bootstrap admin email: unset; configure `AUTH_BOOTSTRAP_ADMIN_EMAIL`
- Queue driver: `redis`
- SMTP host: `mailpit:1025`
- Default sender: `Golang Clean Architecture <hello@example.com>`
- Database queue poll interval: `500` milliseconds
- Database queue reservation lease: `60` seconds

Redis is the default queue backend. The database queue remains available with
`QUEUE_DRIVER=database`.

## Quick Start (Recommended: Docker)

### 1) Start services

```bash
cp .env.example .env
docker compose up -d --build
```

This starts:
- `server` (Go API with Air hot reload)
- `worker` (configured queue worker with Air hot reload)
- `scheduler` (long-running application scheduler)
- `db` (MySQL)
- `redis` (default queue backend)
- `mailpit` (local SMTP server and message inspector)

Compose uses the Dockerfile's `development` target. A minimal, non-root
production image can be built with:

```bash
docker build --target production -t golang-clean-architecture .
```

MySQL remains the application database and is required by the API and queue job
handlers.

### 2) Run migrations

```bash
make migrate args=up
```

Alternative without `make`:

```bash
docker compose exec server sh -c 'migrate -database "mysql://root:${DATABASE_PASSWORD}@tcp(db:3306)/db_name" -path internal/adapter/mysql/migrations up'
```

### 3) Check logs

```bash
docker compose logs server --tail=100
docker compose logs worker --tail=100
docker compose logs scheduler --tail=100
docker compose logs db --tail=100
docker compose logs redis --tail=100
docker compose logs mailpit --tail=100
```

Mailpit is available at [http://localhost:8025](http://localhost:8025). Registering
a user queues an email-verification message that the worker delivers to Mailpit.

### 4) Test API

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "rifki",
    "email": "rifki@example.com",
    "password": "long-password"
  }'
```

## Migration Commands

Create a new migration:

```bash
make migrate-create name=create_something
```

Apply migrations:

```bash
make migrate args=up
```

Rollback one step:

```bash
make migrate args='down 1'
```

## Queue Commands

```bash
make queue args='dispatch-demo --message="Hello from the queue"'
make queue args=status
make queue args=failed
make queue args='retry <job-id> --queue=default'
make queue args='retry all'
make queue args='delete <job-id> --queue=default'
make queue args='delete all'
```

The worker, scheduler, schedule command, and queue command all use the configured
`queue.driver`. Redis is the default. Database mode requires migration
`000003_create_queue_tables`.

## Logging

Applications emit structured JSON logs to stdout and `logs/app.log`. Configure
the level and file path with `LOGGING_LEVEL` and `LOGGING_FILE`.

The server exposes `GET /health/live` for process health and
`GET /health/ready` for database readiness. Every HTTP response includes an
`X-Request-ID`, and access logs include that identifier.

## Quality Checks

```bash
make fmt
make check
make test-integration
make vuln
```

GitHub Actions runs formatting/lint checks, tests, race detection, vulnerability
scanning, MySQL queue integration tests, and the Docker build.

Database jobs support delayed processing, retries, handler timeouts, uniqueness,
retained completed jobs, failed-job inspection, retry, and deletion. Queue
weights and concurrency use the same configuration for both drivers.

### Queue Backend Configuration

Docker Compose starts Redis and selects it as the queue driver by default:

```bash
docker compose up -d --build
REDIS_ADDRESS=localhost:6379 make queue args=status
```

When running commands outside Compose, ensure `REDIS_ADDRESS` points to the
Redis instance. Switching queue drivers does not transfer jobs already stored
by the previous backend.

To use the database-backed queue instead:

```bash
QUEUE_DRIVER=database docker compose up -d --build
```

Redis still starts in this mode, but worker, scheduler, schedule, and queue
commands use MySQL for queued jobs.

## Scheduler Commands

Schedules are registered in application code and only enqueue durable jobs.

```bash
make schedule args=list
make schedule args=run
```

The default schedule queues refresh-token cleanup daily at midnight UTC. Deterministic task IDs prevent duplicate dispatches for the same schedule and minute.

## Service Processes

Build or run each process independently:

```bash
make build
make run-server
make run-worker
make run-scheduler
```

In production, supervise `cmd/server`, `cmd/worker`, and `cmd/scheduler` as separate services.

## Mail

Mailables define an envelope, rendered plain-text and HTML content, and optional
attachments. `mail.Mailer` supports both immediate SMTP delivery and queued
delivery:

```go
mailer.Send(ctx, mailable)
mailer.Queue(ctx, mailable, mail.QueueOptions{})
```

Queued mail is fully rendered before dispatch and sent by the worker as a
`mail:send` job. By default, it uses the `mail` queue, retries three times, and
has a 30-second handler timeout.

SMTP settings live under `mail` in `configs/config.yaml` and can be overridden
with environment variables such as `MAIL_HOST`, `MAIL_PORT`, `MAIL_USERNAME`,
`MAIL_PASSWORD`, `MAIL_ENCRYPTION`, `MAIL_FROM_ADDRESS`, and `MAIL_FROM_NAME`.
Supported encryption values are `none`, `starttls`, and `tls`.

When running the application outside Docker Compose, use `MAIL_HOST=localhost`
to connect to the local Mailpit container.

## Production Secrets

Set `APP_ENVIRONMENT=production` and configure the container platform to inject
`DATABASE_PASSWORD`, `AUTH_JWT_ACCESS_SECRET`, `AUTH_JWT_REFRESH_SECRET`,
`REDIS_PASSWORD`, and `MAIL_PASSWORD` from its managed secret store. Grant the
workload access through workload identity; do not place cloud credentials or a
secret-manager SDK in the application.

Generate independent JWT secrets with a cryptographically secure generator,
for example `openssl rand -base64 48`. Production startup rejects missing,
placeholder, identical, or short JWT secrets and a missing database password.

Rotate secrets in the managed secret store and restart or roll out the affected
workloads. Secrets must never be committed to Git, copied into images, written
to YAML, or included in logs and error messages. Values previously committed to
Git history must be rotated if they were used outside local development.

## API Endpoints

Base URL: `http://localhost:8080/api`

- Public auth routes:
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`
- `POST /auth/verify-email`
- `POST /auth/resend-verification`
- Protected routes (require `Authorization: Bearer <access_token>`):
- `GET /auth/me`
- `PATCH /users/me`
- `PUT /users/me/password`
- `DELETE /users/me`
- Admin-only routes:
- `GET /users?page=1&limit=20`
- `GET /users/:id`
- `POST /users`
- `PUT /users/:id`
- `PUT /users/:id/role`
- `DELETE /users/:id`

### Login Example

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "rifki@example.com",
    "password": "long-password"
  }'
```

### Access Protected Endpoint Example

```bash
curl http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer <access_token>"
```

## Development Notes

- API runs via Air using `.air.toml`.
- Worker runs via Air using `.air.worker.toml`.
- Build target for Air is `./cmd/server` and binary output is `tmp/main`.
- Configuration supports environment-variable overrides such as `DATABASE_HOST`, `DATABASE_PASSWORD`, `AUTH_JWT_ACCESS_SECRET`, `AUTH_JWT_REFRESH_SECRET`, `QUEUE_DRIVER`, `REDIS_ADDRESS`, and `MAIL_PASSWORD`.
- Password hashing and current-password checks are handled in the core service.
- Refresh tokens are persisted as SHA-256 hashes in `refresh_tokens` table.
- User-management routes require the `admin` role; self-service routes only operate on the authenticated account.
- Login requires a verified email. Existing users are unverified after migration `000004` and must request verification.
- The first verified account matching `AUTH_BOOTSTRAP_ADMIN_EMAIL` becomes admin only while no admin exists.
- Non-development/non-test environments require distinct non-placeholder JWT secrets of at least 32 bytes.

## Troubleshooting

### MySQL Error 1130 (host not allowed)

If you see:

`Host '172.x.x.x' is not allowed to connect to this MySQL server`

Confirm that `DATABASE_PASSWORD` is set in `.env`. If you can reset local DB
data completely, recreate the database volume so MySQL initializes with the
current local credential:

```bash
docker compose down -v
docker compose up -d --build
make migrate args=up
```

### Server not building `tmp/main`

Ensure `.air.toml` has:

- `cmd = "go build -o ./tmp/main ./cmd/server"`
- `entrypoint = ["./tmp/main"]`

Then restart the server:

```bash
docker compose restart server
```

## License

No license file is currently defined in this repository.
