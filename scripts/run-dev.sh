#!/bin/sh
set -e
# Build and run the server.
# After editing api-specs/** or stubs, run `make all` to regenerate, then restart.
make all && exec ./bin/openapi-mock run 0.0.0.0 8080
