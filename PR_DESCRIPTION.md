# Commit Message

```text
feat!: harden boilerplate for production use

- reorganize application into feature-first identity and platform packages
- add secure JWT rotation, logout, validation, and versioned auth routes
- standardize queues on Redis/Asynq and remove the database queue
- add health checks, metrics, structured logging, CI, and production images
- add integration, race, lint, vulnerability, and migration verification

BREAKING CHANGE: replace the existing /api routes with /api/v1 auth and me
routes, remove unrestricted user CRUD endpoints, require environment-provided
credentials and JWT secrets, and remove the database queue driver.
```

# Pull Request Title

```text
feat!: harden boilerplate for production use
```

# Pull Request Description

## Summary

This PR upgrades the project from a learning-oriented starter into a production-oriented Go service boilerplate.

It introduces feature-first Clean Architecture, strengthens authentication and configuration safety, standardizes background jobs on Redis/Asynq, adds container-ready operational endpoints, and expands automated verification.

## Key Changes

### Architecture

- Reorganize identity behavior into `internal/identity`, including domain types, application ports, services, HTTP delivery, mail dispatch, and MySQL adapters.
- Consolidate configuration and infrastructure adapters under `internal/platform`.
- Propagate `context.Context` through HTTP requests, application services, repositories, queue operations, and database calls.
- Replace concrete infrastructure dependencies in application code with feature-owned interfaces.

### Authentication And HTTP

- Replace the custom JWT implementation with `golang-jwt/jwt/v5`.
- Validate signing algorithm, issuer, audience, token type, timestamps, and secret strength.
- Use immutable user IDs as JWT subjects.
- Add atomic, single-use refresh-token rotation and `POST /api/v1/auth/logout`.
- Version application routes under `/api/v1`.
- Remove unrestricted cross-user CRUD routes.
- Add strict JSON decoding, input normalization, validation, body limits, request timeouts, request IDs, CORS, security headers, panic recovery, and auth rate limiting.

### Platform And Operations

- Add context-aware database startup retries, connection pinging, pool configuration, and safe error handling without logging credentials.
- Make configuration environment-first and require production credentials and strong JWT secrets.
- Replace file-based global logging with structured `slog` output.
- Add `/health/live`, `/health/ready`, and `/metrics`.
- Standardize background jobs on Redis/Asynq and remove the custom database queue implementation and queue-table migration.
- Add pinned development tooling and a multi-stage non-root production image.
- Pin MySQL, Redis, Mailpit, Go, Air, and migration-tool versions.
- Add `cmd/boilerplate-init` for changing the module path, service name, and database metadata in new projects.

### Quality And Automation

- Add GitHub Actions checks for formatting, vet, unit tests, race tests, integration tests, linting, vulnerability scanning, builds, production images, and migration rollback/reapply.
- Add HTTP, configuration, JWT, authentication rotation/logout, Redis integration, and MySQL repository integration tests.
- Add expanded Make targets and a pinned GolangCI-Lint configuration.
- Upgrade to Go `1.26.4` and patched dependencies.

## Breaking Changes

- Existing `/api/*` routes are replaced by `/api/v1/*`.
- Default user-administration CRUD endpoints are removed.
- The database queue driver and queue-table migration are removed.
- Redis is now required for queues and scheduled-job dispatch.
- JWT secrets and database credentials must be provided through environment variables.
- Database migrations moved to `internal/platform/database/migrations`.
- Existing MySQL 9.x development volumes cannot be reused with the pinned MySQL 8.4 image without export/recreation.

## Verification

- [x] `go test ./...`
- [x] `go test -race ./...`
- [x] `go vet ./...`
- [x] `go build ./cmd/...`
- [x] GolangCI-Lint: `0 issues`
- [x] Govulncheck: `0 vulnerabilities found`
- [x] Redis integration test
- [x] MySQL refresh-token rotation integration test
- [x] Migration rollback and reapply
- [x] Docker Compose configuration validation
- [x] Production image build
- [x] Production image verified to run as non-root user `app`
- [x] Boilerplate initializer dry run

## Deployment Notes

1. Create `.env` from `.env.example` and replace every placeholder.
2. Recreate or migrate any local MySQL volume created by MySQL 9.x before starting the pinned MySQL 8.4 service.
3. Deploy `server`, `worker`, and `scheduler` as independent processes.
4. Configure container probes against `/health/live` and `/health/ready`.
