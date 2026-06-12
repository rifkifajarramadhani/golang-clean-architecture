export MYSQL_URL='mysql://root:greygoose@tcp(db:3306)/db_name'

migrate-create:
	docker compose exec server migrate create -ext sql -dir internal/infrastructure/database/migrations -seq $(name)

migrate:
	docker compose exec server migrate -database $(MYSQL_URL) -path internal/infrastructure/database/migrations $(args)

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

test:
	go test ./...

vet:
	go vet ./...
