#!/bin/bash
set -e

find "specs" -name "openapi.yaml" -o -name "openapi.json" | while read spec; do
    dir=$(dirname "$spec")
    rel="${dir#specs/}"
    pkg=$(basename "$rel" | tr '-' '_')
    out="internal/generated/$rel"

    mkdir -p "$out"
    oapi-codegen -package "$pkg" -generate types -o "$out/types.gen.go" "$spec"
    # Use Echo server generator (the plain `server` generator targets chi/mux).
    oapi-codegen -package "$pkg" -generate echo-server -o "$out/server.gen.go" "$spec"
    oapi-codegen -package "$pkg" -generate spec -o "$out/spec.gen.go" "$spec"

    echo "Generated code for $spec -> $out"
done
