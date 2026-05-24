.PHONY: build run seed test clean vet lint dev help infra exec-valkey

APP_NAME   := zeus-system
BUILD_DIR  := ./build
CMD_DIR    := ./cmd/server
GO_FLAGS   := -ldflags="-s -w"
COMPOSE    := podman compose

help:
	@echo "Usage:"
	@echo "  make build     - Build the binary"
	@echo "  make run       - Run the server"
	@echo "  make seed      - Seed the database with initial data"
	@echo "  make test      - Run all tests"
	@echo "  make vet       - Run go vet"
	@echo "  make lint      - Run staticcheck"
	@echo "  make clean     - Remove build artifacts and database"
	@echo "  make dev       - Run with live reload (requires air)"
	@echo "  make infra      - Start infra"
	@echo "  make exec-valkey - Open valkey-cli in container"
	
build:
	@echo "Building $(APP_NAME)..."
	go build $(GO_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(APP_NAME)"

run:
	go run $(CMD_DIR)

seed:
	go run ./cmd/seeder

test:
	go test ./... -v -count=1

vet:
	go vet ./...

lint:
	@command -v staticcheck > /dev/null 2>&1 || (go install honnef.co/go/tools/cmd/staticcheck@latest && echo "installed staticcheck")
	staticcheck ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f system.db
	@echo "Cleaned"

infra:
	$(COMPOSE) up -d

exec-valkey:
	$(COMPOSE) exec valkey valkey-cli

dev:
	@command -v air > /dev/null 2>&1 || (echo "Installing air for live reload..." && go install github.com/air-verse/air@latest)
	air --tmp_dir "$(BUILD_DIR)/tmp"
