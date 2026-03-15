export MYSQL_URL='mysql://root:greygoose@tcp(db:3306)/db_name'

migrate-create:
	docker compose exec backend migrate create -ext sql -dir internal/infrastructure/database/migrations -seq $(name)

migrate:
	docker compose exec backend migrate -database $(MYSQL_URL) -path internal/infrastructure/database/migrations $(args)