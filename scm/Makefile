.PHONY: build run seed migrate test clean vet lint dev backfill help

APP_NAME   := zeus-scm
BUILD_DIR  := ./build
CMD_DIR    := ./cmd/server
GO_FLAGS   := -ldflags="-s -w"
DB_PATH    ?= scm.db

help:
	@echo "Usage:"
	@echo "  make build     - Build the binary"
	@echo "  make run       - Run the server"
	@echo "  make seed      - Run the SCM seeder"
	@echo "  make backfill  - Backfill parts for products with no parts"
	@echo "  make migrate   - Run SQL migrations"
	@echo "  make test      - Run all tests"
	@echo "  make vet       - Run go vet"
	@echo "  make lint      - Run staticcheck"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make dev       - Run with live reload (requires air)"

build:
	@echo "Building $(APP_NAME)..."
	go build $(GO_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(APP_NAME)"

run:
	go run $(CMD_DIR)

seed:
	rm -f $(DB_PATH) $(DB_PATH)-wal $(DB_PATH)-shm
	go run ./cmd/seeder -db $(DB_PATH) -migrations ./internal/migration -parts-data ../reference/seeder/parts.json

backfill:
	go run ./cmd/seeder -db $(DB_PATH) -backfill

migrate:
	go run ./cmd/migrate -db $(DB_PATH) -migrations ./internal/migration

test:
	go test ./... -v -count=1

vet:
	go vet ./...

lint:
	@command -v staticcheck > /dev/null 2>&1 || (go install honnef.co/go/tools/cmd/staticcheck@latest && echo "installed staticcheck")
	staticcheck ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f scm.db
	@echo "Cleaned"

dev:
	@command -v air > /dev/null 2>&1 || (echo "Installing air for live reload..." && go install github.com/air-verse/air@latest)
	air --tmp_dir "$(BUILD_DIR)/tmp"
