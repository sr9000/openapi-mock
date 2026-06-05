#!/bin/bash
set -e

find "api-specs" \( -name "openapi.yaml" -o -name "openapi.yml" -o -name "openapi.json" \) | while read spec; do
    dir=$(dirname "$spec")
    rel="${dir#api-specs/}"
    pkg=$(basename "$rel" | tr '-' '_')
    out="internal/generated/$rel"

    mkdir -p "$out"

    go tool oapi-codegen -package "$pkg" -generate types -o "$out/types.gen.go" "$spec"
    # Strict server interface + chi routing.
    go tool oapi-codegen -package "$pkg" -generate chi-server,strict-server -o "$out/server.gen.go" "$spec"
    go tool oapi-codegen -package "$pkg" -generate spec -o "$out/spec.gen.go" "$spec"

    echo "Generated code for $spec -> $out"
done
