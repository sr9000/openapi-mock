.PHONY: all build run wire clean help openapi stub
# Default target: show help when `make` is called without arguments
.DEFAULT_GOAL := help
all: openapi stub wire build
# Generate OpenAPI code
openapi:
	@echo
	@echo "===================="
	@echo "Generating OpenAPI code..."
	./scripts/gen-openapi.sh
stub:
	@echo
	@echo "===================="
	@echo "Updating stubs..."
	@if command -v upd-stubs >/dev/null 2>&1; then \
		upd-stubs; \
	else \
		go build -o bin/upd-stubs ./cmd/upd-stubs && ./bin/upd-stubs; \
	fi
# Generate wire dependency injection
wire:
	@echo
	@echo "===================="
	@echo "Updating wire..."
	go run github.com/google/wire/cmd/wire@latest gen ./internal/app
# Build the server
build:
	@echo
	@echo "===================="
	@echo "Building OpenAPI server..."
	go build -o bin/openapi-mock ./cmd/openapi-mock
# Run the server
run:
	@echo
	@echo "===================="
	@echo "Running OpenAPI server..."
	go run ./cmd/openapi-mock run
# Docker operations
docker-build:
	@echo
	@echo "===================="
	@echo "Building Docker image..."
	docker build -t openapi-mock:latest .
docker-run:
	@echo
	@echo "===================="
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 -p 9000:9000 -p 9100:9100 openapi-mock:latest
docker-dev:
	@echo
	@echo "===================="
	@echo "Starting development environment..."
	docker compose up --build
# Docker Compose (Full Stack with Grafana)
compose-up:
	@echo
	@echo "===================="
	@echo "Starting full stack (OpenAPI Mock + Prometheus + Grafana)..."
	docker compose -f docker-compose-grafana.yaml up --build -d
compose-logs:
	@echo
	@echo "===================="
	@echo "Following logs..."
	docker compose -f docker-compose-grafana.yaml logs -f
compose-down:
	@echo
	@echo "===================="
	@echo "Stopping full stack..."
	docker compose -f docker-compose-grafana.yaml down
# Show help
help:
	@echo "Available targets:"
	@echo "  openapi        - Generate OpenAPI code from specs"
	@echo "  stub           - Update OpenAPI stubs"
	@echo "  wire           - Update wire dependency injection"
	@echo "  build          - Build server binary"
	@echo "  run            - Run OpenAPI server"
	@echo "  docker-build   - Build production Docker image"
	@echo "  docker-run     - Run production Docker container"
	@echo "  docker-dev     - Start development environment (watch mode)"
	@echo "  compose-up     - Start full stack (Mock + Monitoring)"
	@echo "  compose-logs   - Follow logs of full stack"
	@echo "  compose-down   - Stop full stack"
