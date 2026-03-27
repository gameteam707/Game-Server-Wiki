.PHONY: run test lint clean

run:
	go run ./cmd/server/

test:
	go test ./... -v

lint:
	golangci-lint run ./...

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

infra-up:
	docker compose up -d

infra-down:
	docker compose down
