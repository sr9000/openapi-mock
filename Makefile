.PHONY: all build run gen-proto gen-wire clean help openapi

# Default target: show help when `make` is called without arguments
.DEFAULT_GOAL := help

all: proto openapi stub wire build

# Update go_package in proto files
update-proto-pkg:
	@echo
	@echo "===================="
	@echo "Updating go_package in proto files..."
	python3 scripts/update_go_packages.py

# Generate protobuf files
proto: update-proto-pkg
	@echo
	@echo "===================="
	@echo "(Re)Generating protobufs..."
	./scripts/gen-protos.sh

# Generate OpenAPI code
openapi:
	@echo
	@echo "===================="
	@echo "Generating OpenAPI code..."
	@chmod +x ./scripts/gen-openapi.sh && ./scripts/gen-openapi.sh

stub:
	@echo
	@echo "===================="
	@echo "Updating stubs..."
	@# Check if upd-stubs exists in PATH (Docker environment)
	@# OR fallback for local run: build it first
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
build: build-grpc build-openapi

build-grpc:
	@echo
	@echo "===================="
	@echo "Building gRPC server..."
	go build -o bin/grpc-mock ./cmd/grpc-mock

build-openapi:
	@echo
	@echo "===================="
	@echo "Building OpenAPI server..."
	go build -o bin/openapi-mock ./cmd/openapi-mock

# Run the gRPC server
run:
	@echo
	@echo "===================="
	@echo "Running gRPC server..."
	go run ./cmd/grpc-mock

# Run the OpenAPI server
run-openapi:
	@echo
	@echo "===================="
	@echo "Running OpenAPI server..."
	go run ./cmd/openapi-mock run

# Docker operations
docker-build:
	@echo
	@echo "===================="
	@echo "Building Docker image..."
	docker build -t grpc-mock:latest .

docker-run:
	@echo
	@echo "===================="
	@echo "Running Docker container..."
	docker run --rm -p 50051:50051 -p 9000:9000 -p 9100:9100 grpc-mock:latest

docker-dev:
	@echo
	@echo "===================="
	@echo "Starting development environment..."
	docker compose up --build

# Docker Compose (Full Stack with Grafana)
compose-up:
	@echo
	@echo "===================="
	@echo "Starting full stack (gRPC Mock + Prometheus + Grafana)..."
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
	@echo "  proto          - (Re)Generate .pb.go files"
	@echo "  openapi        - Generate OpenAPI code from specs"
	@echo "  stub           - Update gRPC stubs"
	@echo "  wire           - Update wire dependency injection"
	@echo "  build          - Build all server binaries"
	@echo "  build-grpc     - Build gRPC server binary"
	@echo "  build-openapi  - Build OpenAPI server binary"
	@echo "  run            - Run gRPC server"
	@echo "  run-openapi    - Run OpenAPI server"
	@echo "  docker-build   - Build production Docker image"
	@echo "  docker-run     - Run production Docker container"
	@echo "  docker-dev     - Start development environment (watch mode)"
	@echo "  compose-up     - Start full stack (Mock + Monitoring)"
	@echo "  compose-logs   - Follow logs of full stack"
	@echo "  compose-down   - Stop full stack"
