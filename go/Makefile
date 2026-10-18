DB_URL=postgres://postgres:postgres@localhost:5432/central_db?sslmode=disable

migrateup:
	migrate -path db/migrations -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path db/migrations -database "$(DB_URL)" -verbose down

migrateforceup:
	migrate -path db/migrations -database "$(DB_URL)" force 1

seed:
	go run ./cmd/seeder