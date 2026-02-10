#!/bin/sh
set -e
# Start Go Watcher in the foreground.
# It handles compilation (go -> binary) and restarts the server.
exec air -c .air.go.toml -- run 0.0.0.0 8080
