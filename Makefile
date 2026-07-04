-include .env
export

DATABASE_USER ?= root
DATABASE_NAME ?= db_name
export MYSQL_URL ?= mysql://$(DATABASE_USER):$(DATABASE_PASSWORD)@tcp(db:3306)/$(DATABASE_NAME)

MIGRATIONS := internal/adapter/mysql/migrations

.PHONY: build run-server run-worker run-scheduler queue schedule migrate-create migrate \
	fmt fmt-check vet lint test test-race test-integration vuln check

build:
	go build ./cmd/server ./cmd/worker ./cmd/scheduler ./cmd/queue ./cmd/schedule

run-server:
	go run ./cmd/server

run-worker:
	go run ./cmd/worker

run-scheduler:
	go run ./cmd/scheduler

queue:
	go run ./cmd/queue $(args)

schedule:
	go run ./cmd/schedule $(args)

migrate-create:
	docker compose exec server migrate create -ext sql -dir $(MIGRATIONS) -seq $(name)

migrate:
	docker compose exec server migrate -database '$(MYSQL_URL)' -path $(MIGRATIONS) $(args)

fmt:
	goimports -w .
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l .)"
	test -z "$$(goimports -l .)"

vet:
	go vet ./...

lint:
	golangci-lint run

test:
	go test ./...

test-race:
	go test -race ./...

test-integration:
	QUEUE_TEST_MYSQL_DSN="$${QUEUE_TEST_MYSQL_DSN}" go test ./internal/adapter/queue -run DatabaseQueue
	USER_TEST_MYSQL_DSN="$${USER_TEST_MYSQL_DSN}" go test ./internal/adapter/mysql -run UserSecurity

vuln:
	govulncheck ./...

check: fmt-check vet lint test test-race
