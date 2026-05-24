# ==============================================================================
# STAGE 1: Tools & Base Dependencies
# ==============================================================================
FROM golang:1.25-alpine AS tools

# 0. Setup fastet mirror for apk to speed up dependency installation
# Completely overwrite the file with LeaseWeb's fast mirror
RUN echo "https://ftp.halifax.rwth-aachen.de/alpine/latest-stable/main/" > /etc/apk/repositories
RUN echo "https://ftp.halifax.rwth-aachen.de/alpine/latest-stable/community/" >> /etc/apk/repositories
RUN apk update

# 1. Install System Dependencies
#    - make: to run the pipeline
#    - git: for go mod download
#    - build-base: for gcc (if needed by cgo)
RUN apk add --no-cache bash make git build-base

# 2. Install Go Global Tools
#    - wire for dependency injection
#    - air for hot-reloading (watcher)
#    - oapi-codegen for OpenAPI code generation
RUN go install github.com/google/wire/cmd/wire@latest && \
    go install github.com/air-verse/air@latest && \
    go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

WORKDIR /app

# 3. Pre-build upd-stubs (Optimization)
#    We copy only what is needed to build the CLI tool first.
#    This allows us to cache the tool compilation even if business logic changes.
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/upd-stubs ./cmd/upd-stubs

# Build and install upd-stubs to global path so it is available in PATH
RUN go build -o /usr/local/bin/upd-stubs ./cmd/upd-stubs


# ==============================================================================
# STAGE 2: Development (Watcher / Hot-Reload)
# ==============================================================================
FROM tools AS dev
WORKDIR /app
# --------------------------------------------------------
# FIX: Trust the mounted directory to allow Git operations
# --------------------------------------------------------
RUN git config --global --add safe.directory /app

# Copy repo to provide template structure
COPY . .

# Start the script
CMD ["./scripts/run-dev.sh"]


# ==============================================================================
# STAGE 3: Production Builder
# ==============================================================================
FROM tools AS builder
WORKDIR /app

# For production build, we copy all source code into the image
COPY . .

# Run the full generation and build pipeline
RUN make all


# ==============================================================================
# STAGE 4: Production Runner (Minimal Image)
# ==============================================================================
FROM alpine:latest AS production

# Install certificates for HTTPS and timezone data
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /home/app

# Create a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Copy compiled binary from the builder stage
COPY --from=builder /app/bin/openapi-mock .

# Default configuration
ENV HTTP_PORT=8080
ENV MGMT_PORT=9000
ENV METRICS_PORT=9100
EXPOSE 8080 9000 9100
ENTRYPOINT ["./openapi-mock"]
CMD ["run"]
