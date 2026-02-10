# OpenAPI Mock Server Implementation Plan

## Quick Reference

| Item | Value |
|------|-------|
| HTTP Port | 8080 |
| Management Port | 9000 |
| Metrics Port | 9100 |
| OpenAPI Specs | `specs/<name>/openapi.yaml` |
| Generated Code | `internal/generated/<name>/` |
| Stubs | `internal/stubs/<name>/` |

---

## Implementation Milestones

### M1: Minimal Viable Mock

```
specs/petstore/openapi.yaml → oapi-codegen → internal/generated/petstore/
                                                      ↓
                                         manual stubs in internal/stubs/petstore/
                                                      ↓
                                              cmd/openapi-mock/main.go
```

**Tasks:**
1. Add dependencies: `chi`, `oapi-codegen`, `kin-openapi`
2. Create `scripts/gen-openapi.sh`
3. Add sample spec `specs/petstore/openapi.yaml`
4. Create manual stubs
5. Create `cmd/openapi-mock/main.go`
6. Create `pkg/middleware/recording.go`
7. Manual wire.go for HTTP

### M2: Automated Stub Generation

Extend `cmd/upd-stubs/` to handle OpenAPI specs alongside gRPC.

**New files in `cmd/upd-stubs/`:**
- `openapi_discovery.go` - Find specs in `specs/`
- `openapi_stubs.go` - Generate tag-based handlers
- `openapi_provider.go` - Generate composite handler
- `openapi_wire.go` - Generate HTTP wire section

### M3: Production Ready

- HTTP metrics (`http_requests_total`, `http_request_duration_seconds`)
- Grafana dashboards
- Dockerfile updates
- Documentation

---

## File Templates

### 1. `scripts/gen-openapi.sh`

```bash
#!/bin/bash
set -e

find "specs" -name "openapi.yaml" -o -name "openapi.json" | while read spec; do
    dir=$(dirname "$spec")
    rel="${dir#specs/}"
    pkg=$(basename "$rel" | tr '-' '_')
    out="internal/generated/$rel"
    
    mkdir -p "$out"
    oapi-codegen -package "$pkg" -generate types -o "$out/types.gen.go" "$spec"
    oapi-codegen -package "$pkg" -generate chi-server -o "$out/server.gen.go" "$spec"
    oapi-codegen -package "$pkg" -generate spec -o "$out/spec.gen.go" "$spec"
done
```

### 2. `specs/petstore/openapi.yaml`

```yaml
openapi: "3.0.0"
info:
  title: Petstore API
  version: "1.0.0"
paths:
  /pets:
    get:
      tags: [pets]
      operationId: ListPets
      parameters:
        - name: limit
          in: query
          schema: { type: integer }
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: array
                items: { $ref: '#/components/schemas/Pet' }
    post:
      tags: [pets]
      operationId: CreatePet
      requestBody:
        content:
          application/json:
            schema: { $ref: '#/components/schemas/NewPet' }
      responses:
        '201':
          description: Created
  /pets/{petId}:
    get:
      tags: [pets]
      operationId: GetPetById
      parameters:
        - name: petId
          in: path
          required: true
          schema: { type: integer, format: int64 }
      responses:
        '200':
          description: OK
    delete:
      tags: [pets]
      operationId: DeletePet
      parameters:
        - name: petId
          in: path
          required: true
          schema: { type: integer, format: int64 }
      responses:
        '204':
          description: Deleted
  /health:
    get:
      operationId: HealthCheck
      responses:
        '200':
          description: OK
components:
  schemas:
    Pet:
      type: object
      required: [id, name]
      properties:
        id: { type: integer, format: int64 }
        name: { type: string }
        tag: { type: string }
    NewPet:
      type: object
      required: [name]
      properties:
        name: { type: string }
        tag: { type: string }
```

### 3. `internal/stubs/petstore/pets.go`

```go
package petstore

import (
    "encoding/json"
    "log"
    "net/http"

    gen "grpc-mock/internal/generated/petstore"
    "grpc-mock/pkg/ctxkeys"
)

type PetsHandlers struct {
    EnableLogging bool
}

func NewPetsHandlers(enableLogging bool) *PetsHandlers {
    return &PetsHandlers{EnableLogging: enableLogging}
}

func (h *PetsHandlers) ListPets(w http.ResponseWriter, r *http.Request, params gen.ListPetsParams) {
    if h.EnableLogging {
        reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
        log.Printf("[req_id=%s] [PetsHandlers] ListPets", reqID)
    }
    
    pets := []gen.Pet{{Id: 1, Name: "Fluffy"}, {Id: 2, Name: "Buddy"}}
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(pets)
}

func (h *PetsHandlers) CreatePet(w http.ResponseWriter, r *http.Request) {
    if h.EnableLogging {
        reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
        log.Printf("[req_id=%s] [PetsHandlers] CreatePet", reqID)
    }
    
    var pet gen.NewPet
    json.NewDecoder(r.Body).Decode(&pet)
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(gen.Pet{Id: 123, Name: pet.Name, Tag: pet.Tag})
}

func (h *PetsHandlers) GetPetById(w http.ResponseWriter, r *http.Request, petId int64) {
    if h.EnableLogging {
        reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
        log.Printf("[req_id=%s] [PetsHandlers] GetPetById petId=%d", reqID, petId)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(gen.Pet{Id: petId, Name: "Mock Pet"})
}

func (h *PetsHandlers) DeletePet(w http.ResponseWriter, r *http.Request, petId int64) {
    if h.EnableLogging {
        reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
        log.Printf("[req_id=%s] [PetsHandlers] DeletePet petId=%d", reqID, petId)
    }
    
    w.WriteHeader(http.StatusNoContent)
}
```

### 4. `internal/stubs/petstore/default.go`

```go
package petstore

import (
    "encoding/json"
    "log"
    "net/http"

    "grpc-mock/pkg/ctxkeys"
)

type DefaultHandlers struct {
    EnableLogging bool
}

func NewDefaultHandlers(enableLogging bool) *DefaultHandlers {
    return &DefaultHandlers{EnableLogging: enableLogging}
}

func (h *DefaultHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
    if h.EnableLogging {
        reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
        log.Printf("[req_id=%s] [DefaultHandlers] HealthCheck", reqID)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

### 5. `internal/stubs/petstore/provider.go`

```go
// Code generated by upd-stubs. DO NOT EDIT.
package petstore

import (
    "net/http"

    gen "grpc-mock/internal/generated/petstore"
)

type CompositeHandlers struct {
    pets    *PetsHandlers
    default_ *DefaultHandlers
}

func NewCompositeHandlers(pets *PetsHandlers, default_ *DefaultHandlers) gen.ServerInterface {
    return &CompositeHandlers{pets: pets, default_: default_}
}

var _ gen.ServerInterface = (*CompositeHandlers)(nil)

func (c *CompositeHandlers) ListPets(w http.ResponseWriter, r *http.Request, params gen.ListPetsParams) {
    c.pets.ListPets(w, r, params)
}

func (c *CompositeHandlers) CreatePet(w http.ResponseWriter, r *http.Request) {
    c.pets.CreatePet(w, r)
}

func (c *CompositeHandlers) GetPetById(w http.ResponseWriter, r *http.Request, petId int64) {
    c.pets.GetPetById(w, r, petId)
}

func (c *CompositeHandlers) DeletePet(w http.ResponseWriter, r *http.Request, petId int64) {
    c.pets.DeletePet(w, r, petId)
}

func (c *CompositeHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
    c.default_.HealthCheck(w, r)
}
```

### 6. `pkg/middleware/recording.go`

```go
package middleware

import (
    "bytes"
    "context"
    "crypto/rand"
    "fmt"
    "io"
    "log"
    "net/http"
    "time"

    "grpc-mock/pkg/ctxkeys"
    "grpc-mock/pkg/metrics"
    "grpc-mock/pkg/recorder"
)

type responseWriter struct {
    http.ResponseWriter
    statusCode int
    body       bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
    rw.body.Write(b)
    return rw.ResponseWriter.Write(b)
}

func Recording(rec *recorder.Recorder, m *metrics.Metrics, enableLogging bool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            b := make([]byte, 8)
            rand.Read(b)
            reqID := fmt.Sprintf("%x", b)
            start := time.Now()

            var bodyBytes []byte
            if r.Body != nil {
                bodyBytes, _ = io.ReadAll(r.Body)
                r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
            }

            rw := &responseWriter{ResponseWriter: w, statusCode: 200}
            ctx := context.WithValue(r.Context(), ctxkeys.RequestID{}, reqID)
            r = r.WithContext(ctx)

            defer func() {
                if err := recover(); err != nil {
                    rec.Record(recorder.CallRecord{
                        RequestID: reqID, Method: r.Method + " " + r.URL.Path,
                        Timestamp: start, Request: string(bodyBytes),
                        Panic: fmt.Sprintf("%v", err), DurationMs: time.Since(start).Milliseconds(),
                    })
                    if m != nil {
                        m.RecordHTTPRequest(r.Method, r.URL.Path, time.Since(start).Milliseconds(), "panic")
                    }
                    http.Error(w, "Internal Server Error", 500)
                }
            }()

            if enableLogging {
                log.Printf("[req_id=%s] --> %s %s", reqID, r.Method, r.URL.Path)
            }

            next.ServeHTTP(rw, r)

            duration := time.Since(start)
            status := "success"
            if rw.statusCode >= 400 && rw.statusCode < 500 {
                status = "client_error"
            } else if rw.statusCode >= 500 {
                status = "server_error"
            }

            rec.Record(recorder.CallRecord{
                RequestID: reqID, Method: r.Method + " " + r.URL.Path,
                Timestamp: start, Request: string(bodyBytes),
                Response: rw.body.String(), DurationMs: duration.Milliseconds(),
            })

            if m != nil {
                m.RecordHTTPRequest(r.Method, r.URL.Path, duration.Milliseconds(), status)
            }

            if enableLogging {
                log.Printf("[req_id=%s] <-- %d (%dms)", reqID, rw.statusCode, duration.Milliseconds())
            }
        })
    }
}
```

### 7. `cmd/openapi-mock/main.go`

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/caarlos0/env/v11"
    "github.com/go-chi/chi/v5"
    "github.com/spf13/cobra"

    gen "grpc-mock/internal/generated/petstore"
    stubs "grpc-mock/internal/stubs/petstore"
    "grpc-mock/pkg/metrics"
    "grpc-mock/pkg/mgmt"
    "grpc-mock/pkg/middleware"
    "grpc-mock/pkg/recorder"
)

type Config struct {
    Host          string `env:"HOST" envDefault:"0.0.0.0"`
    Port          string `env:"PORT" envDefault:"8080"`
    MgmtPort      string `env:"MGMT_PORT" envDefault:"9000"`
    MetricsPort   string `env:"METRICS_PORT" envDefault:"9100"`
    EnableMgmt    bool   `env:"MGMT_ENABLED" envDefault:"true"`
    EnableMetrics bool   `env:"METRICS_ENABLED" envDefault:"true"`
    EnableLogging bool   `env:"HTTP_LOGGING" envDefault:"true"`
}

var version = "dev"

func main() {
    root := &cobra.Command{Use: "openapi-mock", Short: "OpenAPI mock server"}

    run := &cobra.Command{
        Use: "run", Short: "Run server", Args: cobra.MaximumNArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            var cfg Config
            env.Parse(&cfg)

            if v, _ := cmd.Flags().GetString("port"); v != "" {
                cfg.Port = v
            }
            if len(args) >= 1 {
                cfg.Host = args[0]
            }
            if len(args) >= 2 {
                cfg.Port = args[1]
            }

            return runServer(cfg)
        },
    }
    run.Flags().StringP("port", "p", "", "Port")

    root.AddCommand(run)
    root.AddCommand(&cobra.Command{
        Use: "version", Run: func(c *cobra.Command, a []string) { fmt.Println(version) },
    })

    if err := root.Execute(); err != nil {
        os.Exit(1)
    }
}

func runServer(cfg Config) error {
    addr := net.JoinHostPort(cfg.Host, cfg.Port)
    log.Printf("Starting HTTP server on %s", addr)

    rec := recorder.New()

    var m *metrics.Metrics
    if cfg.EnableMetrics {
        m = metrics.NewHTTP(cfg.MetricsPort)
        m.Start()
    }

    if cfg.EnableMgmt {
        mgmt.New(rec, cfg.MgmtPort).Start()
    }

    // Build handlers
    pets := stubs.NewPetsHandlers(cfg.EnableLogging)
    def := stubs.NewDefaultHandlers(cfg.EnableLogging)
    handlers := stubs.NewCompositeHandlers(pets, def)

    // Build router
    r := chi.NewRouter()
    r.Use(middleware.Recording(rec, m, cfg.EnableLogging))
    gen.HandlerFromMux(handlers, r)

    server := &http.Server{Addr: addr, Handler: r}

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    go func() {
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    <-ctx.Done()
    log.Println("Shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return server.Shutdown(ctx)
}
```

### 8. Metrics Extension (`pkg/metrics/metrics.go` additions)

```go
// Add to Metrics struct:
HTTPRequestsTotal   *prometheus.CounterVec
HTTPRequestDuration *prometheus.HistogramVec

// Add constructor:
func NewHTTP(port string) *Metrics {
    m := &Metrics{port: port, registry: prometheus.NewRegistry()}

    m.HTTPRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests"},
        []string{"method", "path", "status"},
    )
    m.HTTPRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP latency", Buckets: prometheus.DefBuckets},
        []string{"method", "path", "status"},
    )

    m.registry.MustRegister(m.HTTPRequestsTotal, m.HTTPRequestDuration)
    // ... register resource metrics
    return m
}

// Add method:
func (m *Metrics) RecordHTTPRequest(method, path string, durationMs int64, status string) {
    if m.HTTPRequestsTotal != nil {
        m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
        m.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(float64(durationMs) / 1000.0)
    }
}
```

### 9. Makefile Additions

```makefile
all: proto openapi stub wire build

openapi:
	@echo "Generating OpenAPI code..."
	@chmod +x ./scripts/gen-openapi.sh && ./scripts/gen-openapi.sh

build: build-grpc build-openapi

build-openapi:
	go build -o bin/openapi-mock ./cmd/openapi-mock

run-openapi:
	go run ./cmd/openapi-mock run
```

---

## Checklist

### M1: Minimal Viable
- [x] `go get github.com/go-chi/chi/v5 github.com/getkin/kin-openapi/openapi3 github.com/oapi-codegen/runtime`
- [x] `go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`
- [x] Create `scripts/gen-openapi.sh`
- [x] Create `specs/petstore/openapi.yaml`
- [x] Run `./scripts/gen-openapi.sh`
- [x] Create `internal/stubs/petstore/{pets,default,provider}.go`
- [x] Create `pkg/middleware/recording.go`
- [x] Create `cmd/openapi-mock/main.go`
- [x] Add `NewHTTP()` and `RecordHTTPRequest()` to `pkg/metrics/metrics.go`
- [x] Test: `go run ./cmd/openapi-mock run`
- [x] Test: `curl http://localhost:8080/pets`
- [x] Test: `curl http://localhost:9000/logs`

### M2: Auto Generation
- [ ] Refactor `cmd/upd-stubs/main.go` into separate files
- [ ] Add `openapi_discovery.go`
- [ ] Add `openapi_stubs.go`
- [ ] Add `openapi_provider.go`
- [ ] Add `openapi_wire.go`
- [ ] Test: `make stub` generates both gRPC and OpenAPI stubs
- [ ] Test: Custom code in stubs preserved after re-run

### M3: Production
- [ ] HTTP Grafana dashboard
- [ ] Update Dockerfile with `oapi-codegen`
- [ ] Update docker-compose
- [ ] README updates

---

## Usage After Implementation

```bash
# Add spec
cp api.yaml specs/myapi/openapi.yaml

# Generate
make all

# Edit stubs (preserved on regeneration)
vim internal/stubs/myapi/pets.go

# Run
./bin/openapi-mock run

# Test
curl localhost:8080/pets
curl localhost:9000/logs
```
