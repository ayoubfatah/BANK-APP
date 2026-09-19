migrationup:
	migrate -path db/migration -database "postgresql://root:2002252AAEE@localhost:5433/bank-app?sslmode=disable" -verbose up

migrationdown:
	migrate -path db/migration -database "postgresql://root:2002252AAEE@localhost:5433/bank-app?sslmode=disable" -verbose down

postgres :
	docker run --name postgres12  -p 5433:5432 -e  POSTGRES_PASSWORD=2002252AAEE -e  POSTGRES_USER=root -d postgres:12-alpine


createdb : 
	docker exec -it postgres12 createdb --username=root --owner=root bank-app 

dropdb: 
	docker exec -it postgres12 dropdb  bank-app 

sqlc: 
	sqlc generate
 


.PHONY: createdb dropdb postgres  migrationup migrationup sqlc