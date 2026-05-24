.PHONY: run migrate migrate-up migrate-down seed test

DB ?= ./sales.db

run:
	go run ./cmd/server

migrate:
	go run ./cmd/migrate -db $(DB)

# Alias for compatibility
migrate-up: migrate

# Sales migrate command currently supports schema ensure only.
# Keep migrate-down as a no-op alias to avoid broken invocations.
migrate-down:
	@echo "migrate-down is not implemented for sales yet."

seed:
	go run ./cmd/seeder -db $(DB)

test:
	go test ./...

