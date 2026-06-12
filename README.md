# Go Service Boilerplate

Production-oriented Go service boilerplate using feature-first Clean Architecture. It ships an HTTP API, Redis-backed queue worker, scheduler, SMTP mail delivery, MySQL migrations, container images, health checks, metrics, and CI.

## Architecture

```text
cmd/                         independently deployable processes and CLI tools
internal/identity/           identity domain, application ports/service, adapters, HTTP delivery
internal/platform/           config, database, HTTP, security, Redis, SMTP, and logging adapters
internal/queue/              backend-neutral job contracts
internal/scheduler/          schedule contracts and runner
internal/mail/               mail contracts and templates
```

Application code depends on feature-owned interfaces. Request contexts flow from HTTP delivery through application services and repositories. Redis/Asynq is the only supported queue backend.

## Local Development

Requirements: Docker, Docker Compose, and optionally Make.

```bash
cp .env.example .env
# Replace every placeholder in .env.
docker compose up -d --build
make migrate args=up
```

Services:

- API: `http://localhost:8080`
- Mailpit: `http://localhost:8025`
- MySQL: `localhost:3306`
- Redis: `localhost:6379`

Configuration is environment-first. `configs/config.yaml` contains non-secret development defaults; JWT secrets and database credentials must come from environment variables. Production startup rejects missing credentials, weak JWT secrets, and invalid limits.

If an existing development volume was created by `mysql:latest` 9.x, export or remove that development volume before starting the pinned MySQL 8.4 service; MySQL does not support that major-version downgrade in place.

## HTTP API

All application routes are versioned under `/api/v1`.

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET  /api/v1/me

GET  /health/live
GET  /health/ready
GET  /metrics
```

The default API intentionally contains no cross-user administration endpoints. Add those only with explicit project-specific authorization.

Registration example:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"example","email":"example@example.com","password":"a-long-local-password"}'
```

Refresh tokens are stored as hashes, atomically rotated on refresh, single-use, and revocable through logout. Welcome mail dispatch is best-effort and does not roll back registration.

## Commands

```bash
make check
make test-unit
make test-race
make test-integration
make lint
make vuln
make build
make prod-images

make migrate-create name=create_something
make migrate args=up
make migrate args='down 1'

make queue args=status
make queue args=failed
make queue args='retry all'
make schedule args=list
make schedule args=run
```

Integration tests require migrated MySQL and Redis:

```bash
export IDENTITY_TEST_MYSQL_DSN='app:password@tcp(127.0.0.1:3306)/app?parseTime=true'
export REDIS_ADDRESS='127.0.0.1:6379'
go test -tags=integration ./...
```

## Deployment

`Dockerfile.dev` contains pinned development tools. `Dockerfile` is a multi-stage production build that emits a minimal non-root image:

```bash
docker build --build-arg TARGET=server -t service-server .
docker build --build-arg TARGET=worker -t service-worker .
docker build --build-arg TARGET=scheduler -t service-scheduler .
```

Deploy server, worker, and scheduler independently. Configure container health probes against `/health/live` and `/health/ready`.

## Start A New Project

Run the initializer once in a fresh copy:

```bash
go run ./cmd/boilerplate-init \
  --module github.com/example/my-service \
  --service my-service \
  --database my_service \
  --dry-run
```

Remove `--dry-run` after reviewing the affected files. The command updates the Go module path, service name, and default database metadata while skipping `.git`, `.env`, and binary files.
