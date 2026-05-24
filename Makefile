.PHONY: all build run wire clean help openapi stub
# Default target: show help when `make` is called without arguments
.DEFAULT_GOAL := help

DEV_COMPOSE_FILE := docker-compose.dev.yaml
OBSERVABILITY_COMPOSE_FILE := docker-compose.observability.yaml
COMPOSE_ENV_PATH := $(or $(wildcard .env),$(wildcard deploy/.env))
COMPOSE_ENV_FILE := $(if $(COMPOSE_ENV_PATH),--env-file $(COMPOSE_ENV_PATH),)

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
	@# Always use the in-repo generator to avoid picking up an outdated `upd-stubs` from PATH.
	go run ./cmd/upd-stubs
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
	docker compose $(COMPOSE_ENV_FILE) -f $(DEV_COMPOSE_FILE) up --build
# Docker Compose (Full Stack with Grafana)
compose-up:
	@echo
	@echo "===================="
	@echo "Building full stack images (plain progress)..."
	docker compose $(COMPOSE_ENV_FILE) -f $(OBSERVABILITY_COMPOSE_FILE) --progress plain build
	@echo "Starting full stack (OpenAPI Mock + Prometheus + Grafana)..."
	docker compose $(COMPOSE_ENV_FILE) -f $(OBSERVABILITY_COMPOSE_FILE) up -d
compose-logs:
	@echo
	@echo "===================="
	@echo "Following logs..."
	docker compose $(COMPOSE_ENV_FILE) -f $(OBSERVABILITY_COMPOSE_FILE) logs -f
compose-down:
	@echo
	@echo "===================="
	@echo "Stopping full stack..."
	docker compose $(COMPOSE_ENV_FILE) -f $(OBSERVABILITY_COMPOSE_FILE) down
compose-smoke:
	@echo
	@echo "===================="
	@echo "Running observability stack smoke test..."
	./scripts/validate-observability-stack.sh
# Show help
help:
	@echo "Available targets:"
	@echo "  openapi        - Generate OpenAPI code from api"
	@echo "  stub           - Update OpenAPI stubs"
	@echo "  wire           - Update wire dependency injection"
	@echo "  build          - Build server binary"
	@echo "  run            - Run OpenAPI server"
	@echo "  docker-build   - Build production Docker image"
	@echo "  docker-run     - Run production Docker container"
	@echo "  docker-dev     - Start development environment (watch mode via docker-compose.dev.yaml)"
	@echo "  compose-up     - Start full stack (Mock + Monitoring via docker-compose.observability.yaml)"
	@echo "  compose-logs   - Follow logs of full stack"
	@echo "  compose-down   - Stop full stack"
	@echo "  compose-smoke  - Run automated stack smoke validation"
