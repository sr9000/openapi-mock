#!/bin/bash
set -e

find "specs" -name "openapi.yaml" -o -name "openapi.json" | while read spec; do
    dir=$(dirname "$spec")
    rel="${dir#specs/}"
    pkg=$(basename "$rel" | tr '-' '_')
    out="internal/generated/$rel"

    mkdir -p "$out"

    oapi-codegen -package "$pkg" -generate types -o "$out/types.gen.go" "$spec"
    # Strict server interface + chi routing.
    oapi-codegen -package "$pkg" -generate chi-server,strict-server -o "$out/server.gen.go" "$spec"
    oapi-codegen -package "$pkg" -generate spec -o "$out/spec.gen.go" "$spec"

    echo "Generated code for $spec -> $out"
done
