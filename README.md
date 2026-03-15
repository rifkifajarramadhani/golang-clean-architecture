# Golang Clean Architecture

Backend API project using Clean Architecture principles with Go, Fiber, MySQL, and GORM.

## Tech Stack

- **Language:** Go 1.26
- **HTTP Framework:** Fiber v3
- **Database:** MySQL 8 (Docker)
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
│   └── worker/                 # Worker entrypoint (reserved)
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

## Quick Start (Recommended: Docker)

### 1) Start services

```bash
docker compose up -d --build
```

This starts:
- `backend` (Go API with Air hot reload)
- `db` (MySQL)

### 2) Run migrations

```bash
make migrate args=up
```

Alternative without `make`:

```bash
docker compose exec backend migrate -database 'mysql://root:greygoose@tcp(db:3306)/db_name' -path internal/infrastructure/database/migrations up
```

### 3) Check logs

```bash
docker compose logs backend --tail=100
docker compose logs db --tail=100
```

### 4) Test API

```bash
curl http://localhost:8080/api/users
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

## API Endpoints

Base URL: `http://localhost:8080/api`

- `GET /users`
- `GET /users/:id`
- `POST /users`
- `PUT /users/:id`
- `DELETE /users/:id`

### Create User Example

```bash
curl -X POST http://localhost:8080/api/users \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "rifki",
    "email": "rifki@example.com",
    "password": "secret123"
  }'
```

## Development Notes

- API runs via Air using `.air.toml`.
- Build target for Air is `./cmd/server` and binary output is `tmp/main`.
- Password hashing is handled in `usecase` layer before repository insert.
- At the moment, `POST /users` is wired to usecase/repository flow; other endpoints may still return placeholder responses depending on handler implementation.

## Troubleshooting

### MySQL Error 1130 (host not allowed)

If you see:

`Host '172.x.x.x' is not allowed to connect to this MySQL server`

Run:

```bash
docker compose exec db mysql -uroot -pgreygoose -e "CREATE USER IF NOT EXISTS 'root'@'%' IDENTIFIED BY 'greygoose'; GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION; FLUSH PRIVILEGES;"
docker compose restart backend
```

If you can reset local DB data completely:

```bash
docker compose down -v
docker compose up -d --build
make migrate args=up
```

### Backend not building `tmp/main`

Ensure `.air.toml` has:

- `cmd = "go build -o ./tmp/main ./cmd/server"`

Then restart backend:

```bash
docker compose restart backend
```

## License

No license file is currently defined in this repository.
