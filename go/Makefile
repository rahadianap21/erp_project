DB_URL=postgres://postgres:postgres@localhost:5432/central_db?sslmode=disable

migrateup:
	migrate -path db/migrations -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path db/migrations -database "$(DB_URL)" -verbose down

migrateforceup:
	migrate -path db/migrations -database "$(DB_URL)" force 1

seed:
	go run ./cmd/seeder

test:
	go build ./... && go vet ./... && go test -count=1 ./...

test-integration:
	go vet -tags=integration ./... && go test -tags=integration -count=1 ./...

lint:
	golangci-lint run

fmt:
	gofmt -s -w . && golangci-lint fmt ./...

build:
	go build ./...

sqlc:
	sqlc generate

run:
	go run ./cmd/api