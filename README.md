# Golang Clean Architecture

Backend services boilerplate using Clean Architecture principles with independently deployable HTTP, queue worker, and scheduler processes.

## Tech Stack

- **Language:** Go 1.26
- **HTTP Framework:** Fiber v3
- **Database:** MySQL 8 (Docker)
- **Queue:** Configurable MySQL or Redis + Asynq
- **Scheduler:** Application-defined cron schedules
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
│   ├── delivery/http/          # HTTP handlers, routers, DTOs
│   ├── domain/                 # Domain entities
│   ├── infrastructure/         # DB connection, logger, migrations
│   ├── models/                 # Database models
│   ├── repository/             # Repository implementation
│   └── usecase/                # Business logic
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

Current configuration file:

- `configs/config.yaml`

Default values in this project:

- App port: `8080`
- DB host: `db`
- DB port: `3306`
- DB user: `root`
- DB password: `greygoose`
- DB name: `db_name`
- JWT access secret: `super-secret-access-key-change-me`
- JWT refresh secret: `super-secret-refresh-key-change-me`
- Access token TTL: `15` minutes
- Refresh token TTL: `168` hours
- Queue driver: `redis`
- Database queue poll interval: `500` milliseconds
- Database queue reservation lease: `60` seconds

Redis is the default queue backend. The database queue remains available with
`QUEUE_DRIVER=database`.

## Quick Start (Recommended: Docker)

### 1) Start services

```bash
docker compose up -d --build
```

This starts:
- `server` (Go API with Air hot reload)
- `worker` (configured queue worker with Air hot reload)
- `scheduler` (long-running application scheduler)
- `db` (MySQL)
- `redis` (default queue backend)

MySQL remains the application database and is required by the API and queue job
handlers.

### 2) Run migrations

```bash
make migrate args=up
```

Alternative without `make`:

```bash
docker compose exec server migrate -database 'mysql://root:greygoose@tcp(db:3306)/db_name' -path internal/infrastructure/database/migrations up
```

### 3) Check logs

```bash
docker compose logs server --tail=100
docker compose logs worker --tail=100
docker compose logs scheduler --tail=100
docker compose logs db --tail=100
docker compose logs redis --tail=100
```

### 4) Test API

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "rifki",
    "email": "rifki@example.com",
    "password": "secret123"
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

## API Endpoints

Base URL: `http://localhost:8080/api`

- Public auth routes:
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- Protected routes (require `Authorization: Bearer <access_token>`):
- `GET /auth/me`
- `GET /users`
- `GET /users/:id`
- `POST /users`
- `PUT /users/:id`
- `DELETE /users/:id`

### Login Example

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "rifki@example.com",
    "password": "secret123"
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
- Configuration supports environment-variable overrides such as `DATABASE_HOST`, `QUEUE_DRIVER`, `QUEUE_DATABASE_POLL_INTERVAL_MILLISECONDS`, `QUEUE_DATABASE_RESERVATION_SECONDS`, and `REDIS_ADDRESS`.
- Password hashing is handled in the `usecase` layer.
- Refresh tokens are persisted as SHA-256 hashes in `refresh_tokens` table.
- Existing `/users` routes are now JWT-protected.

## Troubleshooting

### MySQL Error 1130 (host not allowed)

If you see:

`Host '172.x.x.x' is not allowed to connect to this MySQL server`

Run:

```bash
docker compose exec db mysql -uroot -pgreygoose -e "CREATE USER IF NOT EXISTS 'root'@'%' IDENTIFIED BY 'greygoose'; GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION; FLUSH PRIVILEGES;"
docker compose restart server
```

If you can reset local DB data completely:

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
