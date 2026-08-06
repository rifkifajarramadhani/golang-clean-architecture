-include .env
export

DATABASE_USER ?= root
DATABASE_NAME ?= db_name
DIST_DIR ?= dist
export MYSQL_URL ?= mysql://$(DATABASE_USER):$(DATABASE_PASSWORD)@tcp(db:3306)/$(DATABASE_NAME)

MIGRATIONS := internal/adapter/mysql/migrations

.PHONY: build run-server run-worker run-scheduler run-all queue schedule migrate-create migrate \
	fmt fmt-check vet lint test test-race test-integration vuln check template-pack

build:
	go build ./cmd/app

run-server:
	go run ./cmd/app serve

run-worker:
	go run ./cmd/app worker

run-scheduler:
	go run ./cmd/app scheduler

run-all:
	go run ./cmd/app all

queue:
	go run ./cmd/app queue $(args)

schedule:
	go run ./cmd/app schedule $(args)

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
	USER_TEST_MYSQL_DSN="$${USER_TEST_MYSQL_DSN}" go test ./internal/adapter/mysql/identity -run UserSecurity

vuln:
	govulncheck ./...

check: fmt-check vet lint test test-race

template-pack:
	go run ./internal/templatepack -output-dir "$(DIST_DIR)" $(if $(TEMPLATE_VERSION),-version "$(TEMPLATE_VERSION)")
