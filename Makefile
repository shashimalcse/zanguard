.PHONY: build test lint migrate-up migrate-down tidy

build:
	go build ./...

test:
	go test ./...

test-integration:
	go test -tags integration ./...

lint:
	go vet ./...

tidy:
	go mod tidy

migrate-up:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL not set"; exit 1; fi
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL not set"; exit 1; fi
	migrate -path migrations -database "$(DATABASE_URL)" down 1

run:
	go run cmd/server/main.go
