.PHONY: help setup run test test-unit test-integration lint clean swagger

help:
	@echo "Available commands:"
	@echo "  make setup            - Install dependencies"
	@echo "  make run              - Run development server"
	@echo "  make test             - Run the full test suite (unit + integration + e2e; needs Docker)"
	@echo "  make test-unit        - Run only unit tests (utils/middleware/services; no Docker needed)"
	@echo "  make test-integration - Run only integration + e2e tests (repository + tests/e2e; needs Docker)"
	@echo "  make lint             - Run linter"
	@echo "  make swagger          - Regenerate Swagger API docs"
	@echo "  make clean            - Clean build files"

setup:
	go mod download
	go mod tidy

run:
	go run ./cmd/server/main.go

# Full suite: unit tests plus the Docker-backed integration/e2e tests
# (internal/repository, tests/e2e), which spin up a real MySQL container via
# testcontainers-go. Requires a running Docker daemon.
test:
	go test -v ./...

# Fast path: exercises internal/utils, internal/middleware, and
# internal/services against mocked dependencies only - no Docker required.
# internal/repository and tests/e2e skip themselves under -short.
test-unit:
	go test -short -v ./...

# internal/repository (repository layer against real MySQL) and tests/e2e
# (full HTTP stack against real MySQL) only. Requires a running Docker daemon.
test-integration:
	go test -v ./internal/repository/... ./tests/e2e/...

lint:
	golangci-lint run

swagger:
	swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

clean:
	rm -rf bin/
	go clean