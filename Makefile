export MYSQL_URL='mysql://root:rootpassword@tcp(db:3306)/db_name'

migrate-create:
	docker compose exec web migrate create -ext sql -dir migrations -seq $(name)

migrate:
	docker compose exec web migrate -database $(MYSQL_URL) -path migrations $(args)