# Step-by-Step Plan: OpenAPI Mock Server Tool

A comprehensive guide to building an OpenAPI mock server similar to [grpc-mock](.), adapted for REST/HTTP APIs.

---

## 📋 Overview

### Key Design Principles

1. **Parallel Architecture**: Run both gRPC and HTTP mocks from the same codebase
2. **Shared Infrastructure**: Reuse `pkg/` components (recorder, metrics, mgmt, ctxkeys)
3. **Consistent Patterns**: Mirror the grpc-mock patterns for familiarity
4. **Incremental Addition**: Add OpenAPI support without breaking existing gRPC functionality

### Component Mapping

| Component | gRPC Mock | OpenAPI Mock |
|-----------|-----------|--------------|
| Schema | `.proto` files in `protos/` | `openapi.yaml/json` in `specs/` |
| Server | gRPC (`google.golang.org/grpc`) | HTTP (chi router) |
| Code Gen | `protoc` + `protoc-gen-go` | `oapi-codegen` |
| Generated Output | `internal/genproto/` | `internal/generated/` |
| Stubs | `internal/stubs/` (per service) | `internal/stubs/` (per tag) |
| Stub Generator | `cmd/upd-stubs/` (existing) | `cmd/upd-stubs/` (extended) |
| Default Port | 50051 | 8080 |
| Interceptor | `grpc.UnaryInterceptor` | chi middleware |
| Metrics Prefix | `grpc_*` | `http_*` |

---

## 🛠️ Implementation Roadmap

### Milestone 1: Minimal Viable OpenAPI Mock (Week 1-2)

**Goal**: Single OpenAPI spec → working HTTP server with basic stubs

| Task | Effort | Dependencies |
|------|--------|--------------|
| 1.1 Add dependencies (`chi`, `oapi-codegen`, `kin-openapi`) | 1h | None |
| 1.2 Create `scripts/gen-openapi.sh` | 2h | None |
| 1.3 Add sample `specs/petstore/openapi.yaml` | 1h | None |
| 1.4 Manual stub creation for petstore | 4h | 1.2, 1.3 |
| 1.5 Create `cmd/openapi-mock/main.go` | 4h | 1.4 |
| 1.6 Create `pkg/middleware/recording.go` | 3h | None |
| 1.7 Basic wire.go for HTTP (manual) | 2h | 1.4, 1.5 |
| 1.8 Test end-to-end | 2h | All above |

**Deliverable**: `./bin/openapi-mock run` serves petstore API with hardcoded responses

### Milestone 2: Automated Stub Generation (Week 3-4)

**Goal**: Extend `cmd/upd-stubs/` to generate OpenAPI stubs automatically

| Task | Effort | Dependencies |
|------|--------|--------------|
| 2.1 Refactor `main.go` into modular files | 4h | None |
| 2.2 Create `openapi_discovery.go` | 3h | 2.1 |
| 2.3 Create `openapi_stubs.go` (tag-based generation) | 8h | 2.2 |
| 2.4 Create `openapi_provider.go` (composite handler) | 4h | 2.3 |
| 2.5 Create `openapi_wire.go` (wire.go generation) | 4h | 2.4 |
| 2.6 Implement AST-based body preservation | 6h | 2.3 |
| 2.7 Integration testing | 4h | All above |

**Deliverable**: `make stub` generates both gRPC and OpenAPI stubs

### Milestone 3: Full Feature Parity (Week 5-6)

**Goal**: Metrics, Grafana dashboards, Docker support

| Task | Effort | Dependencies |
|------|--------|--------------|
| 3.1 Extend `pkg/metrics/` for HTTP | 3h | M2 |
| 3.2 Create HTTP Grafana dashboards | 4h | 3.1 |
| 3.3 Update Dockerfile for dual binaries | 2h | M2 |
| 3.4 Update Makefile with all targets | 2h | M2 |
| 3.5 Update docker-compose files | 2h | 3.3 |
| 3.6 Documentation | 4h | All above |

**Deliverable**: Production-ready OpenAPI mock with full observability

---

## Phase 1: Project Restructuring

### Step 1.1: Updated Directory Structure

Extend the existing structure to support both protocols:

```
openapi-mock/
├── cmd/
│   ├── grpc-mock/                     # KEEP - existing gRPC server
│   │   └── main.go
│   ├── openapi-mock/                  # NEW - HTTP mock server
│   │   └── main.go
│   └── upd-stubs/                     # EXTEND - support both protocols
│       ├── main.go                    # Entry point, orchestration
│       ├── grpc_stubs.go              # Existing gRPC logic (refactored)
│       ├── grpc_wire.go               # gRPC wire generation
│       ├── openapi_discovery.go       # NEW: Find OpenAPI specs
│       ├── openapi_stubs.go           # NEW: Generate tag-based stubs
│       ├── openapi_provider.go        # NEW: Generate composite handlers
│       ├── openapi_wire.go            # NEW: HTTP wire generation
│       ├── ast_utils.go               # Shared AST utilities
│       └── type_utils.go              # Shared type formatting
├── internal/
│   ├── app/
│   │   ├── wire.go                    # Combined gRPC + HTTP providers
│   │   └── wire_gen.go
│   ├── genproto/                      # KEEP - gRPC generated code
│   ├── generated/                     # NEW - OpenAPI generated code
│   │   └── petstore/
│   │       ├── types.gen.go
│   │       ├── server.gen.go
│   │       └── spec.gen.go
│   └── stubs/
│       ├── echo/                      # KEEP - gRPC stubs
│       ├── complex/service/           # KEEP - gRPC stubs
│       └── petstore/                  # NEW - OpenAPI stubs
│           ├── pets.go                # "pets" tag handlers (editable)
│           ├── users.go               # "users" tag handlers (editable)
│           └── provider.go            # Auto-generated composite
├── pkg/
│   ├── ctxkeys/                       # KEEP - shared
│   ├── metrics/                       # EXTEND - add HTTP metrics
│   ├── mgmt/                          # KEEP - shared
│   ├── ptrtools/                      # KEEP - shared
│   ├── recorder/                      # KEEP - shared
│   └── middleware/                    # NEW - HTTP middleware
│       └── recording.go
├── specs/                             # NEW - OpenAPI schemas
│   └── petstore/
│       └── openapi.yaml
└── scripts/
    ├── gen-protos.sh                  # KEEP
    └── gen-openapi.sh                 # NEW
```

### Step 1.2: Stub Generator Refactoring Plan

The current `cmd/upd-stubs/main.go` (1158 lines) should be split:

```
cmd/upd-stubs/
├── main.go                 (~100 lines)  - Entry point, CLI
├── grpc_discovery.go       (~100 lines)  - discoverGRPCServices()
├── grpc_stubs.go           (~400 lines)  - generateStubFile(), updateStubFile()
├── grpc_wire.go            (~150 lines)  - generateWireFile()
├── openapi_discovery.go    (~80 lines)   - discoverOpenAPISpecs()
├── openapi_stubs.go        (~350 lines)  - generateOpenAPIStubs(), tag-based
├── openapi_provider.go     (~150 lines)  - generateProviderFile()
├── openapi_wire.go         (~150 lines)  - generateHTTPWireSection()
├── ast_utils.go            (~200 lines)  - updateStubFile shared logic
└── type_utils.go           (~150 lines)  - formatType(), zeroValue(), etc.
```

**Refactoring approach**:
1. Extract shared utilities first (`type_utils.go`, `ast_utils.go`)
2. Move gRPC-specific code to `grpc_*.go` files
3. Add OpenAPI-specific code to `openapi_*.go` files
4. Update `main.go` to orchestrate both

---

## Phase 2: Tooling & Dependencies

### Step 2.1: Add New Dependencies

```bash
# Run these commands:
go get github.com/go-chi/chi/v5@latest
go get github.com/getkin/kin-openapi/openapi3@latest
go get github.com/oapi-codegen/runtime@latest

# Install code generator globally (for scripts)
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

### Step 2.2: Updated go.mod

Add to existing `go.mod`:

```go
require (
    // ...existing dependencies...
    
    // NEW - OpenAPI dependencies
    github.com/go-chi/chi/v5 v5.1.0
    github.com/getkin/kin-openapi v0.128.0
    github.com/oapi-codegen/runtime v1.1.1
)
```

### Step 2.3: Update Dockerfile

Add `oapi-codegen` installation in the tools stage:

```dockerfile
# In Stage 1 (tools), add to the RUN command:
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
    go install github.com/google/wire/cmd/wire@latest && \
    go install github.com/air-verse/air@latest && \
    go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

---

## Phase 3: OpenAPI Code Generation

### Step 3.1: Create `scripts/gen-openapi.sh`

```bash
#!/bin/bash
set -e

SPECS_DIR="specs"
OUTPUT_DIR="internal/generated"

echo "Generating OpenAPI code..."

# Find all openapi.yaml or openapi.json files
find "$SPECS_DIR" -type f \( -name "openapi.yaml" -o -name "openapi.json" \) | while read spec; do
    # Skip hidden directories
    [[ "$spec" == *"/."* ]] && continue
    
    # Calculate paths
    spec_dir=$(dirname "$spec")
    rel_path="${spec_dir#$SPECS_DIR/}"
    
    # Package name: last component, sanitized (dashes to underscores)
    pkg_name=$(basename "$rel_path" | tr '-' '_' | tr '.' '_')
    
    output_dir="$OUTPUT_DIR/$rel_path"
    mkdir -p "$output_dir"
    
    echo "  $spec -> $output_dir (package: $pkg_name)"
    
    # Generate types (models)
    oapi-codegen -package "$pkg_name" \
        -generate types \
        -o "$output_dir/types.gen.go" \
        "$spec"
    
    # Generate chi-server interface
    oapi-codegen -package "$pkg_name" \
        -generate chi-server \
        -o "$output_dir/server.gen.go" \
        "$spec"
    
    # Generate embedded spec (for Swagger UI)
    oapi-codegen -package "$pkg_name" \
        -generate spec \
        -o "$output_dir/spec.gen.go" \
        "$spec"
done

echo "OpenAPI generation complete!"
```

### Step 3.2: Sample OpenAPI Spec

Create `specs/petstore/openapi.yaml`:

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
      summary: List all pets
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
            default: 20
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
    post:
      tags: [pets]
      operationId: CreatePet
      summary: Create a pet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewPet'
      responses:
        '201':
          description: Pet created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
  /pets/{petId}:
    get:
      tags: [pets]
      operationId: GetPetById
      summary: Get a pet by ID
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
            format: int64
      responses:
        '200':
          description: A pet
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
        '404':
          description: Pet not found
    delete:
      tags: [pets]
      operationId: DeletePet
      summary: Delete a pet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
            format: int64
      responses:
        '204':
          description: Pet deleted
  /health:
    get:
      operationId: HealthCheck
      summary: Health check endpoint
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string

components:
  schemas:
    Pet:
      type: object
      required: [id, name]
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
        tag:
          type: string
    NewPet:
      type: object
      required: [name]
      properties:
        name:
          type: string
        tag:
          type: string
```

### Step 3.3: Generated Code Structure

After running `./scripts/gen-openapi.sh`:

```
internal/generated/petstore/
├── types.gen.go      # Pet, NewPet structs
├── server.gen.go     # ServerInterface + HandlerFromMux()
└── spec.gen.go       # GetSwagger() - embedded spec
```

**Generated `ServerInterface`** (excerpt from `server.gen.go`):

```go
// ServerInterface represents all server handlers.
type ServerInterface interface {
    // (GET /health)
    HealthCheck(w http.ResponseWriter, r *http.Request)
    // (GET /pets)
    ListPets(w http.ResponseWriter, r *http.Request, params ListPetsParams)
    // (POST /pets)
    CreatePet(w http.ResponseWriter, r *http.Request)
    // (DELETE /pets/{petId})
    DeletePet(w http.ResponseWriter, r *http.Request, petId int64)
    // (GET /pets/{petId})
    GetPetById(w http.ResponseWriter, r *http.Request, petId int64)
}

// ListPetsParams defines parameters for ListPets.
type ListPetsParams struct {
    Limit *int `form:"limit,omitempty" json:"limit,omitempty"`
}
```

---

## Phase 4: Stub Generator Extension

### Step 4.1: Core Data Structures

Add to `cmd/upd-stubs/openapi_discovery.go`:

```go
package main

import (
    "os"
    "path/filepath"
    "strings"
    
    "github.com/getkin/kin-openapi/openapi3"
)

const (
    generatedPath = "grpc-mock/internal/generated"
    specsDir      = "specs"
)

type openAPISpecInfo struct {
    SpecPath   string            // e.g., "specs/petstore/openapi.yaml"
    GenPkgPath string            // e.g., "grpc-mock/internal/generated/petstore"
    PkgName    string            // e.g., "petstore"
    StubOutDir string            // e.g., "internal/stubs/petstore"
    Doc        *openapi3.T       // Parsed OpenAPI document
    Tags       map[string]*tagInfo
}

type tagInfo struct {
    Name       string       // e.g., "pets", "default"
    TypeName   string       // e.g., "PetsHandlers", "DefaultHandlers"
    Operations []operation
}

type operation struct {
    OperationID string      // e.g., "ListPets"
    Method      string      // GET, POST, etc.
    Path        string      // /pets/{petId}
    Summary     string
    Tags        []string
}

func discoverOpenAPISpecs() ([]openAPISpecInfo, error) {
    var specs []openAPISpecInfo
    
    err := filepath.Walk(specsDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            return nil
        }
        
        base := filepath.Base(path)
        if base != "openapi.yaml" && base != "openapi.json" {
            return nil
        }
        
        // Load and parse spec
        loader := openapi3.NewLoader()
        doc, err := loader.LoadFromFile(path)
        if err != nil {
            return err
        }
        
        // Calculate paths
        rel := strings.TrimPrefix(filepath.Dir(path), specsDir+string(os.PathSeparator))
        rel = filepath.ToSlash(rel) // Normalize to forward slashes
        
        pkgName := sanitizePkgName(filepath.Base(rel))
        
        spec := openAPISpecInfo{
            SpecPath:   path,
            GenPkgPath: generatedPath + "/" + rel,
            PkgName:    pkgName,
            StubOutDir: filepath.Join(stubsOutDir, rel),
            Doc:        doc,
            Tags:       extractTags(doc),
        }
        
        specs = append(specs, spec)
        return nil
    })
    
    return specs, err
}

func extractTags(doc *openapi3.T) map[string]*tagInfo {
    tags := make(map[string]*tagInfo)
    
    for path, pathItem := range doc.Paths.Map() {
        for method, op := range pathItem.Operations() {
            if op == nil || op.OperationID == "" {
                continue
            }
            
            opInfo := operation{
                OperationID: op.OperationID,
                Method:      strings.ToUpper(method),
                Path:        path,
                Summary:     op.Summary,
                Tags:        op.Tags,
            }
            
            // Assign to tags (or "default" if no tags)
            tagNames := op.Tags
            if len(tagNames) == 0 {
                tagNames = []string{"default"}
            }
            
            for _, tagName := range tagNames {
                if tags[tagName] == nil {
                    tags[tagName] = &tagInfo{
                        Name:     tagName,
                        TypeName: toPascalCase(tagName) + "Handlers",
                    }
                }
                tags[tagName].Operations = append(tags[tagName].Operations, opInfo)
            }
        }
    }
    
    return tags
}

func sanitizePkgName(name string) string {
    name = strings.ReplaceAll(name, "-", "_")
    name = strings.ReplaceAll(name, ".", "_")
    return strings.ToLower(name)
}
```

### Step 4.2: Tag-Based Stub Generation

Add to `cmd/upd-stubs/openapi_stubs.go`:

```go
package main

import (
    "bytes"
    "fmt"
    "go/ast"
    "go/format"
    "go/parser"
    "go/token"
    "go/types"
    "os"
    "path/filepath"
    "sort"
    "strings"
    
    "golang.org/x/tools/go/packages"
)

func generateOpenAPIStubs(spec openAPISpecInfo) error {
    // Load generated ServerInterface
    cfg := &packages.Config{
        Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo,
        Dir:  ".",
    }
    pkgs, err := packages.Load(cfg, spec.GenPkgPath)
    if err != nil {
        return fmt.Errorf("failed to load generated package: %w", err)
    }
    if len(pkgs) == 0 {
        return fmt.Errorf("no packages found at %s", spec.GenPkgPath)
    }
    
    pkg := pkgs[0]
    serverInterface := findInterface(pkg, "ServerInterface")
    if serverInterface == nil {
        return fmt.Errorf("ServerInterface not found in %s", spec.GenPkgPath)
    }
    
    // Create output directory
    if err := os.MkdirAll(spec.StubOutDir, 0o755); err != nil {
        return err
    }
    
    // Generate stub file for each tag
    for tagName, tagInfo := range spec.Tags {
        if err := generateTagStubFile(spec, tagInfo, serverInterface); err != nil {
            return fmt.Errorf("failed to generate %s stubs: %w", tagName, err)
        }
    }
    
    // Generate provider.go
    if err := generateProviderFile(spec, serverInterface); err != nil {
        return fmt.Errorf("failed to generate provider: %w", err)
    }
    
    return nil
}

func findInterface(pkg *packages.Package, name string) *types.Interface {
    obj := pkg.Types.Scope().Lookup(name)
    if obj == nil {
        return nil
    }
    iface, ok := obj.Type().Underlying().(*types.Interface)
    if !ok {
        return nil
    }
    return iface
}

func generateTagStubFile(spec openAPISpecInfo, tag *tagInfo, serverInterface *types.Interface) error {
    fileName := toSnakeCase(tag.Name) + ".go"
    outPath := filepath.Join(spec.StubOutDir, fileName)
    
    // Check if file exists - if so, update it
    if _, err := os.Stat(outPath); err == nil {
        return updateOpenAPIStubFile(outPath, spec, tag, serverInterface)
    }
    
    // Generate new file
    var buf bytes.Buffer
    
    genAlias := spec.PkgName + "gen"
    
    imports := map[string]string{
        "encoding/json":         "",
        "log":                   "",
        "net/http":              "",
        spec.GenPkgPath:         genAlias,
        "grpc-mock/pkg/ctxkeys": "",
    }
    
    // Package declaration
    fmt.Fprintf(&buf, "package %s\n\n", spec.PkgName)
    
    // Imports
    fmt.Fprintf(&buf, "import (\n")
    sortedImports := sortedKeys(imports)
    for _, imp := range sortedImports {
        alias := imports[imp]
        if alias != "" {
            fmt.Fprintf(&buf, "\t%s %q\n", alias, imp)
        } else {
            fmt.Fprintf(&buf, "\t%q\n", imp)
        }
    }
    fmt.Fprintf(&buf, ")\n\n")
    
    // Handler struct
    fmt.Fprintf(&buf, "// %s handles \"%s\" endpoints\n", tag.TypeName, tag.Name)
    fmt.Fprintf(&buf, "type %s struct {\n", tag.TypeName)
    fmt.Fprintf(&buf, "\tEnableLogging bool\n")
    fmt.Fprintf(&buf, "}\n\n")
    
    // Constructor
    fmt.Fprintf(&buf, "func New%s(enableLogging bool) *%s {\n", tag.TypeName, tag.TypeName)
    fmt.Fprintf(&buf, "\treturn &%s{EnableLogging: enableLogging}\n", tag.TypeName)
    fmt.Fprintf(&buf, "}\n")
    
    // Generate methods
    for _, op := range tag.Operations {
        method := findMethod(serverInterface, op.OperationID)
        if method == nil {
            continue
        }
        
        methodCode := generateOpenAPIMethod(tag.TypeName, op, method.Type().(*types.Signature), imports)
        fmt.Fprintf(&buf, "\n%s\n", methodCode)
    }
    
    // Format
    src, err := format.Source(buf.Bytes())
    if err != nil {
        return fmt.Errorf("failed to format %s: %w\nSource:\n%s", fileName, err, buf.String())
    }
    
    return os.WriteFile(outPath, src, 0o644)
}

func generateOpenAPIMethod(typeName string, op operation, sig *types.Signature, imports map[string]string) string {
    var buf bytes.Buffer
    
    // Comment with HTTP method and path
    fmt.Fprintf(&buf, "// %s - %s\n", op.OperationID, op.Summary)
    fmt.Fprintf(&buf, "// %s %s\n", op.Method, op.Path)
    
    // Method signature
    fmt.Fprintf(&buf, "func (h *%s) %s(", typeName, op.OperationID)
    
    params := sig.Params()
    var paramList []string
    var logParams []string
    
    for i := 0; i < params.Len(); i++ {
        p := params.At(i)
        pName := p.Name()
        if pName == "" {
            switch i {
            case 0:
                pName = "w"
            case 1:
                pName = "r"
            default:
                pName = fmt.Sprintf("arg%d", i)
            }
        }
        
        pType := formatType(p.Type(), imports)
        paramList = append(paramList, fmt.Sprintf("%s %s", pName, pType))
        
        // Collect loggable params (skip w, r)
        if i >= 2 {
            logParams = append(logParams, pName)
        }
    }
    fmt.Fprintf(&buf, "%s)", strings.Join(paramList, ", "))
    
    // No return type for HTTP handlers
    fmt.Fprintf(&buf, " {\n")
    
    // Logging
    fmt.Fprintf(&buf, "\tif h.EnableLogging {\n")
    fmt.Fprintf(&buf, "\t\treqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)\n")
    
    if len(logParams) > 0 {
        formatParts := make([]string, len(logParams))
        for i := range formatParts {
            formatParts[i] = "%+v"
        }
        fmt.Fprintf(&buf, "\t\tlog.Printf(\"[req_id=%%s] [%s] %s called: %%s %%s params=%s\", reqID, r.Method, r.URL.Path, %s)\n",
            typeName, op.OperationID, strings.Join(formatParts, ", "), strings.Join(logParams, ", "))
    } else {
        fmt.Fprintf(&buf, "\t\tlog.Printf(\"[req_id=%%s] [%s] %s called: %%s %%s\", reqID, r.Method, r.URL.Path)\n",
            typeName, op.OperationID)
    }
    fmt.Fprintf(&buf, "\t}\n\n")
    
    // Default implementation based on HTTP method
    switch op.Method {
    case "GET":
        fmt.Fprintf(&buf, "\t// TODO: Implement mock response\n")
        fmt.Fprintf(&buf, "\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
        fmt.Fprintf(&buf, "\tjson.NewEncoder(w).Encode(map[string]string{\"status\": \"ok\"})\n")
    case "POST", "PUT":
        fmt.Fprintf(&buf, "\t// TODO: Implement mock response\n")
        fmt.Fprintf(&buf, "\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
        fmt.Fprintf(&buf, "\tw.WriteHeader(http.StatusCreated)\n")
        fmt.Fprintf(&buf, "\tjson.NewEncoder(w).Encode(map[string]string{\"status\": \"created\"})\n")
    case "DELETE":
        fmt.Fprintf(&buf, "\tw.WriteHeader(http.StatusNoContent)\n")
    default:
        fmt.Fprintf(&buf, "\tw.WriteHeader(http.StatusOK)\n")
    }
    
    fmt.Fprintf(&buf, "}")
    
    return buf.String()
}

func findMethod(iface *types.Interface, name string) *types.Func {
    for i := 0; i < iface.NumMethods(); i++ {
        m := iface.Method(i)
        if m.Name() == name {
            return m
        }
    }
    return nil
}

func sortedKeys(m map[string]string) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    return keys
}
```

### Step 4.3: Provider Generation

Add to `cmd/upd-stubs/openapi_provider.go`:

```go
package main

import (
    "bytes"
    "fmt"
    "sort"
    "strings"
)

type httpWireInfo struct {
    Specs []openAPISpecInfo
}

func generateHTTPWireSection(buf *bytes.Buffer, specs []openAPISpecInfo) {
    if len(specs) == 0 {
        return
    }
    
    fmt.Fprintf(buf, "\n// ============ HTTP/OpenAPI ============\n\n")
    
    // Imports for HTTP stubs
    fmt.Fprintf(buf, "// HTTP stub imports\n")
    for _, spec := range specs {
        alias := spec.PkgName + "stubs"
        importPath := "grpc-mock/" + spec.StubOutDir
        fmt.Fprintf(buf, "// %s %q\n", alias, importPath)
        
        genAlias := spec.PkgName + "gen"
        fmt.Fprintf(buf, "// %s %q\n", genAlias, spec.GenPkgPath)
    }
    fmt.Fprintf(buf, "\n")
    
    // HTTPApp struct
    fmt.Fprintf(buf, "type HTTPApp struct {\n")
    fmt.Fprintf(buf, "\tRouter http.Handler\n")
    fmt.Fprintf(buf, "}\n\n")
    
    // HTTP ProviderSet
    fmt.Fprintf(buf, "var HTTPProviderSet = wire.NewSet(\n")
    for _, spec := range specs {
        alias := spec.PkgName + "stubs"
        
        // Sort tags
        var tags []*tagInfo
        for _, tag := range spec.Tags {
            tags = append(tags, tag)
        }
        sort.Slice(tags, func(i, j int) bool {
            return tags[i].Name < tags[j].Name
        })
        
        for _, tag := range tags {
            fmt.Fprintf(buf, "\t%s.New%s,\n", alias, tag.TypeName)
        }
        fmt.Fprintf(buf, "\t%s.NewCompositeHandlers,\n", alias)
    }
    fmt.Fprintf(buf, "\tNewHTTPRouter,\n")
    fmt.Fprintf(buf, "\twire.Struct(new(HTTPApp), \"*\"),\n")
    fmt.Fprintf(buf, ")\n\n")
    
    // InitializeHTTPRouter function
    fmt.Fprintf(buf, "func InitializeHTTPRouter(rec *recorder.Recorder, m *metrics.Metrics, enableLogging bool) http.Handler {\n")
    fmt.Fprintf(buf, "\twire.Build(HTTPProviderSet)\n")
    fmt.Fprintf(buf, "\treturn nil\n")
    fmt.Fprintf(buf, "}\n\n")
    
    // NewHTTPRouter function
    fmt.Fprintf(buf, "func NewHTTPRouter(\n")
    for _, spec := range specs {
        genAlias := spec.PkgName + "gen"
        fmt.Fprintf(buf, "\t%sHandlers %s.ServerInterface,\n", spec.PkgName, genAlias)
    }
    fmt.Fprintf(buf, "\trec *recorder.Recorder,\n")
    fmt.Fprintf(buf, "\tm *metrics.Metrics,\n")
    fmt.Fprintf(buf, "\tenableLogging bool,\n")
    fmt.Fprintf(buf, ") http.Handler {\n")
    fmt.Fprintf(buf, "\tr := chi.NewRouter()\n")
    fmt.Fprintf(buf, "\tr.Use(middleware.Recording(rec, m, enableLogging))\n")
    for _, spec := range specs {
        genAlias := spec.PkgName + "gen"
        fmt.Fprintf(buf, "\t%s.HandlerFromMux(%sHandlers, r)\n", genAlias, spec.PkgName)
    }
    fmt.Fprintf(buf, "\treturn r\n")
    fmt.Fprintf(buf, "}\n")
}
```

### Step 4.6: Updated Main Entry Point

Update `cmd/upd-stubs/main.go`:

```go
package main

import (
    "log"
)

func main() {
    // 1. Process gRPC services
    log.Println("Discovering gRPC services...")
    grpcPkgMap, err := discoverAndGenerateGRPCStubs()
    if err != nil {
        log.Fatalf("gRPC stub generation failed: %v", err)
    }
    
    // 2. Process OpenAPI specs
    log.Println("Discovering OpenAPI specs...")
    openapiSpecs, err := discoverOpenAPISpecs()
    if err != nil {
        log.Fatalf("OpenAPI discovery failed: %v", err)
    }
    
    for _, spec := range openapiSpecs {
        log.Printf("Generating stubs for %s...", spec.PkgName)
        if err := generateOpenAPIStubs(spec); err != nil {
            log.Fatalf("OpenAPI stub generation failed for %s: %v", spec.PkgName, err)
        }
    }
    
    // 3. Generate unified wire.go
    log.Println("Generating wire.go...")
    if err := generateUnifiedWireFile(grpcPkgMap, openapiSpecs); err != nil {
        log.Fatalf("wire.go generation failed: %v", err)
    }
    
    log.Println("Stubs generated successfully!")
}

func discoverAndGenerateGRPCStubs() (map[string]*stubPkg, error) {
    // Existing gRPC discovery and generation logic
    // (moved from current main() function)
    // Returns pkgMap for wire generation
}

func generateUnifiedWireFile(grpcPkgMap map[string]*stubPkg, openapiSpecs []openAPISpecInfo) error {
    // Generate combined wire.go with both gRPC and HTTP providers
}
```

---

## Phase 5: HTTP Server Implementation

### Step 5.1: Create `cmd/openapi-mock/main.go`

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
    "github.com/spf13/cobra"
    
    "grpc-mock/internal/app"
    "grpc-mock/pkg/metrics"
    "grpc-mock/pkg/mgmt"
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
    rootCmd := &cobra.Command{
        Use:   "openapi-mock",
        Short: "OpenAPI/HTTP mock server",
    }

    runCmd := &cobra.Command{
        Use:   "run [host] [port]",
        Short: "Run the HTTP mock server",
        Args:  cobra.MaximumNArgs(2),
        RunE:  runServer,
    }

    // Same flag pattern as grpc-mock
    runCmd.Flags().StringP("host", "", "", "Host interface (overrides HOST env)")
    runCmd.Flags().StringP("port", "p", "", "Port (overrides PORT env)")
    runCmd.Flags().StringP("mgmt-port", "m", "", "Management port")
    runCmd.Flags().StringP("metrics-port", "", "", "Metrics port")
    runCmd.Flags().Bool("no-mgmt", false, "Disable management server")
    runCmd.Flags().Bool("no-metrics", false, "Disable metrics server")
    runCmd.Flags().Bool("no-logs", false, "Disable request logging")

    versionCmd := &cobra.Command{
        Use:   "version",
        Short: "Print version",
        Run:   func(cmd *cobra.Command, args []string) { fmt.Println(version) },
    }

    rootCmd.AddCommand(runCmd, versionCmd)

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func runServer(cmd *cobra.Command, args []string) error {
    cfg := loadConfig(cmd, args)
    
    addr := net.JoinHostPort(cfg.Host, cfg.Port)
    log.Printf("Config: host=%s port=%s mgmt=%t metrics=%t logging=%t",
        cfg.Host, cfg.Port, cfg.EnableMgmt, cfg.EnableMetrics, cfg.EnableLogging)

    // Shared components
    rec := recorder.New()
    
    var metricsServer *metrics.Metrics
    if cfg.EnableMetrics {
        metricsServer = metrics.NewHTTP(cfg.MetricsPort)
    }

    // Initialize router via Wire
    router := app.InitializeHTTPRouter(rec, metricsServer, cfg.EnableLogging)

    // Start management server
    var mgmtServer *mgmt.Server
    if cfg.EnableMgmt {
        mgmtServer = mgmt.New(rec, cfg.MgmtPort)
        mgmtServer.Start()
    }

    // Start metrics server
    if metricsServer != nil {
        metricsServer.Start()
    }

    // HTTP server
    server := &http.Server{
        Addr:         addr,
        Handler:      router,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
    }

    // Graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    go func() {
        log.Printf("HTTP mock server listening on %s", addr)
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    <-ctx.Done()
    log.Println("Shutting down...")
    
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if mgmtServer != nil {
        mgmtServer.Stop(shutdownCtx)
    }
    if metricsServer != nil {
        metricsServer.Stop(shutdownCtx)
    }
    return server.Shutdown(shutdownCtx)
}

func loadConfig(cmd *cobra.Command, args []string) Config {
    var cfg Config
    env.Parse(&cfg)
    
    // Override with flags (same pattern as grpc-mock)
    if v, _ := cmd.Flags().GetString("host"); v != "" {
        cfg.Host = v
    }
    if v, _ := cmd.Flags().GetString("port"); v != "" {
        cfg.Port = v
    }
    if v, _ := cmd.Flags().GetString("mgmt-port"); v != "" {
        cfg.MgmtPort = v
    }
    if v, _ := cmd.Flags().GetString("metrics-port"); v != "" {
        cfg.MetricsPort = v
    }
    if cmd.Flags().Changed("no-mgmt") {
        v, _ := cmd.Flags().GetBool("no-mgmt")
        cfg.EnableMgmt = !v
    }
    if cmd.Flags().Changed("no-metrics") {
        v, _ := cmd.Flags().GetBool("no-metrics")
        cfg.EnableMetrics = !v
    }
    if cmd.Flags().Changed("no-logs") {
        v, _ := cmd.Flags().GetBool("no-logs")
        cfg.EnableLogging = !v
    }

    if len(args) >= 1 {
        cfg.Host = args[0]
    }
    if len(args) >= 2 {
        cfg.Port = args[1]
    }
    
    return cfg
}
```

### Step 5.2: Create `pkg/middleware/recording.go`

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

func generateRequestID() string {
    b := make([]byte, 8)
    rand.Read(b)
    return fmt.Sprintf("%x", b)
}

func Recording(rec *recorder.Recorder, m *metrics.Metrics, enableLogging bool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            reqID := generateRequestID()
            start := time.Now()
            
            // Capture request body
            var bodyBytes []byte
            if r.Body != nil {
                bodyBytes, _ = io.ReadAll(r.Body)
                r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
            }
            
            // Wrap response writer
            rw := &responseWriter{ResponseWriter: w, statusCode: 200}
            
            // Add request ID to context
            ctx := context.WithValue(r.Context(), ctxkeys.RequestID{}, reqID)
            r = r.WithContext(ctx)
            
            // Handle panics
            defer func() {
                if err := recover(); err != nil {
                    duration := time.Since(start)
                    method := r.Method + " " + r.URL.Path
                    
                    rec.Record(recorder.CallRecord{
                        RequestID:  reqID,
                        Method:     method,
                        Timestamp:  start,
                        Request:    string(bodyBytes),
                        Panic:      fmt.Sprintf("%v", err),
                        DurationMs: duration.Milliseconds(),
                    })
                    
                    if m != nil {
                        m.RecordHTTPRequest(r.Method, r.URL.Path, duration.Milliseconds(), "panic")
                    }
                    
                    if enableLogging {
                        log.Printf("[req_id=%s] PANIC: %v", reqID, err)
                    }
                    
                    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                }
            }()
            
            if enableLogging {
                log.Printf("[req_id=%s] --> %s %s", reqID, r.Method, r.URL.Path)
            }
            
            next.ServeHTTP(rw, r)
            
            duration := time.Since(start)
            method := r.Method + " " + r.URL.Path
            status := statusCategory(rw.statusCode)
            
            rec.Record(recorder.CallRecord{
                RequestID:  reqID,
                Method:     method,
                Timestamp:  start,
                Request:    string(bodyBytes),
                Response:   rw.body.String(),
                DurationMs: duration.Milliseconds(),
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

func statusCategory(code int) string {
    switch {
    case code >= 200 && code < 300:
        return "success"
    case code >= 400 && code < 500:
        return "client_error"
    case code >= 500:
        return "server_error"
    default:
        return "other"
    }
}
```

---

## Phase 6: Metrics Extension

### Step 6.1: Extend `pkg/metrics/metrics.go`

Add HTTP metrics to the existing file:

```go
// Add these fields to the Metrics struct:
type Metrics struct {
    // ...existing gRPC fields...
    
    // HTTP metrics
    HTTPRequestsTotal   *prometheus.CounterVec
    HTTPRequestDuration *prometheus.HistogramVec
    
    protocol string // "grpc", "http", or "both"
}

// Add NewHTTP constructor:
func NewHTTP(port string) *Metrics {
    m := &Metrics{
        protocol: "http",
        port:     port,
        registry: prometheus.NewRegistry(),
    }
    
    m.HTTPRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"method", "path", "status"},
    )
    
    m.HTTPRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request latency",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path", "status"},
    )
    
    m.registry.MustRegister(m.HTTPRequestsTotal, m.HTTPRequestDuration)
    // ...register resource metrics...
    
    return m
}

// Add HTTP recording method:
func (m *Metrics) RecordHTTPRequest(method, path string, durationMs int64, status string) {
    if m.HTTPRequestsTotal != nil {
        m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
        m.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(float64(durationMs) / 1000.0)
    }
}
```

---

## Phase 7: Makefile Extension

### Step 7.1: Add New Targets

Add to existing `Makefile`:

```makefile
# Add 'openapi' to the 'all' target
all: proto openapi stub wire build

# OpenAPI code generation
openapi:
	@echo "===================="
	@echo "(Re)Generating OpenAPI code..."
	@chmod +x ./scripts/gen-openapi.sh
	./scripts/gen-openapi.sh

# Build targets
build: build-grpc build-openapi

build-grpc:
	@echo "===================="
	@echo "Building gRPC server..."
	go build -o bin/grpc-mock ./cmd/grpc-mock

build-openapi:
	@echo "===================="
	@echo "Building OpenAPI server..."
	go build -o bin/openapi-mock ./cmd/openapi-mock

# Run targets
run-grpc:
	go run ./cmd/grpc-mock run

run-openapi:
	go run ./cmd/openapi-mock run

run: run-grpc  # Default for backward compatibility

# Update help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Code Generation:"
	@echo "  proto        - Generate gRPC code from .proto files"
	@echo "  openapi      - Generate HTTP code from OpenAPI specs"
	@echo "  stub         - Update stubs for both protocols"
	@echo "  wire         - Generate Wire dependency injection"
	@echo ""
	@echo "Build:"
	@echo "  build        - Build both servers"
	@echo "  build-grpc   - Build gRPC server only"
	@echo "  build-openapi- Build OpenAPI server only"
	@echo ""
	@echo "Run:"
	@echo "  run-grpc     - Run gRPC server"
	@echo "  run-openapi  - Run OpenAPI server"
	@echo ""
	@echo "Pipeline: make all = proto + openapi + stub + wire + build"
```

---

## 📊 Implementation Checklist

### Milestone 1: Minimal Viable OpenAPI Mock
- [ ] Add dependencies to `go.mod`
- [ ] Create `scripts/gen-openapi.sh`
- [ ] Add `specs/petstore/openapi.yaml`
- [ ] Run `./scripts/gen-openapi.sh` successfully
- [ ] Create manual stubs in `internal/stubs/petstore/`
- [ ] Create `cmd/openapi-mock/main.go`
- [ ] Create `pkg/middleware/recording.go`
- [ ] Manual `internal/app/wire.go` for HTTP
- [ ] Run `make wire` successfully
- [ ] Test end-to-end: `curl http://localhost:8080/pets`
- [ ] Test management API: `curl http://localhost:9000/logs`

### Milestone 2: Automated Stub Generation
- [ ] Refactor `cmd/upd-stubs/` into modular files
- [ ] Create `openapi_discovery.go`
- [ ] Create `openapi_stubs.go`
- [ ] Create `openapi_provider.go`
- [ ] Create `openapi_wire.go`
- [ ] Test: `make stub` generates OpenAPI stubs
- [ ] Test: Editing stub preserves changes on re-run
- [ ] Test: Adding new endpoint adds new method

### Milestone 3: Full Feature Parity
- [ ] Extend `pkg/metrics/` for HTTP
- [ ] Create HTTP Grafana dashboards
- [ ] Update Dockerfile
- [ ] Update Makefile
- [ ] Update docker-compose files
- [ ] Documentation
- [ ] Integration tests

---

## 🚀 Quick Start (After Implementation)

```bash
# 1. Add your OpenAPI spec
cp your-api.yaml specs/myapi/openapi.yaml

# 2. Generate everything
make all

# 3. Edit stubs
vim internal/stubs/myapi/pets.go     # Your custom logic here

# 4. Rebuild
make wire build

# 5. Run
./bin/openapi-mock run

# 6. Test
curl http://localhost:8080/pets
curl http://localhost:9000/logs
```
