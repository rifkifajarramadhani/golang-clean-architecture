-include .env
export

MYSQL_URL ?= mysql://$(DATABASE_USER):$(DATABASE_PASSWORD)@tcp(db:3306)/$(DATABASE_NAME)
MIGRATIONS := internal/platform/database/migrations

.PHONY: build check fmt lint migrate migrate-create prod-images queue run-scheduler run-server run-worker schedule test test-integration test-race test-unit vet vuln

build:
	go build ./cmd/...

fmt:
	test -z "$$(gofmt -l .)"

vet:
	go vet ./...

test: test-unit

test-unit:
	go test ./...

test-race:
	go test -race ./...

test-integration:
	go test -tags=integration ./...

lint:
	golangci-lint run

vuln:
	govulncheck ./...

check: fmt vet test-unit build

migrate-create:
	docker compose exec server migrate create -ext sql -dir $(MIGRATIONS) -seq $(name)

migrate:
	docker compose exec server migrate -database '$(MYSQL_URL)' -path $(MIGRATIONS) $(args)

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

prod-images:
	docker build --build-arg TARGET=server -t $(APP_NAME)-server:local .
	docker build --build-arg TARGET=worker -t $(APP_NAME)-worker:local .
	docker build --build-arg TARGET=scheduler -t $(APP_NAME)-scheduler:local .
