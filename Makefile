DB_URL=postgresql://root:illchangeitlater@localhost:5433/bank-app?sslmode=disable

postgres:
	docker run --name postgres12 -p 5433:5432 -e POSTGRES_PASSWORD=illchangeitlater -e POSTGRES_USER=root -v bank-app-postgres-data:/var/lib/postgresql/data -d postgres:12-alpine

createdb:
	docker exec -it postgres12 createdb --username=root --owner=root bank-app

dropdb:
	docker exec -it postgres12 dropdb bank-app

migrationup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrationdown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down


resetdb:
	docker exec postgres12 psql -U root -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'bank-app' AND pid <> pg_backend_pid();"
	docker exec postgres12 dropdb --if-exists --username=root bank-app
	docker exec postgres12 createdb --username=root --owner=root bank-app
	migrate -path db/migration -database "$(DB_URL)" -verbose up

sqlc:
	sqlc generate

test:
	go test -v ./...
test-nocache:
	go test -count=1 ./...


testfresh: resetdb test

server:
	go run main.go

createmigration:
	migrate create -ext sql -dir db/migration -seq $(name)



clean:
	docker stop postgres12 || true
	docker rm postgres12 || true
	docker volume rm bank-app-postgres-data || true
f:
	gofmt -w .

mock: 
	mockgen -package mockdb  -destination db/mock/store.go  github.com/bank-app/db/store Store 

	
.PHONY: postgres createdb dropdb migrationup migrationdown resetdb sqlc test testfresh test-nocache clean server  createmigration f mock