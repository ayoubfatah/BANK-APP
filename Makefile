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
	docker exec postgres12 dropdb --if-exists --username=root bank-app
	docker exec postgres12 createdb --username=root --owner=root bank-app
	migrate -path db/migration -database "$(DB_URL)" -verbose up

sqlc:
	sqlc generate

test:
	go test -v ./...

# Reset the database, then run all tests
testfresh: resetdb test

# Nuclear option: removes the container AND the volume
clean:
	docker stop postgres12 || true
	docker rm postgres12 || true
	docker volume rm bank-app-postgres-data || true

.PHONY: postgres createdb dropdb migrationup migrationdown resetdb sqlc test testfresh clean