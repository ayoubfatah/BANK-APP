migrationup:

	migrate -path db/migration -database "postgresql://root:illchangeitlater@localhost:5433/bank-app?sslmode=disable" -verbose up

migrationdown:

	migrate -path db/migration -database "postgresql://root:illchangeitlater@localhost:5433/bank-app?sslmode=disable" -verbose down

postgres:

	docker run --name postgres12 -p 5433:5432 -e POSTGRES_PASSWORD=illchangeitlater -e POSTGRES_USER=root -v bank-app-postgres-data:/var/lib/postgresql/data  -d postgres:12-alpine

createdb:

	docker exec -it postgres12 createdb --username=root --owner=root bank-app

dropdb:

	docker exec -it postgres12 dropdb bank-app

sqlc:

	sqlc generate

test:
	go test -v ./...

clean:
	docker stop postgres12 || true
	docker rm postgres12 || true
	docker volume rm bank-app-postgres-data || true
	

.PHONY: createdb dropdb postgres migrationup migrationdown sqlc test clean