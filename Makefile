.PHONY: build test lint tidy run compose-up compose-down compose-logs

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

run:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL not set"; exit 1; fi
	go run cmd/server/main.go

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f zanguard postgres
