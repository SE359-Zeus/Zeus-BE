.PHONY: run migrate migrate-up migrate-down seed test build fmt

# Database file for local sqlite migrations/seeding
DB ?= ./mrp.db
MANIFEST ?= ./contracts/scm-manifest.json

run:
	go run ./cmd/server

migrate:
	go run ./cmd/migrate -db $(DB)

# Alias for compatibility
migrate-up: migrate

# Keep migrate-down as a no-op to avoid accidental data loss
migrate-down:
	@echo "migrate-down is not implemented for mrp yet."

seed:
	go run ./cmd/seeder -db $(DB) -manifest $(MANIFEST)

test:
	go test ./...

build:
	go build -o bin/mrp ./cmd/server

fmt:
	go vet ./...
	go fmt ./...
