#!/bin/bash
# Test script for OpenAPI mock management server
# This script validates:
# - Management server endpoints (/doc, /openapi.json, /logs, /clear)
# - OpenAPI mock server with petstore API
# - Recording and clearing of HTTP calls
set -e
# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color
# Configuration
HTTP_HOST="localhost"
HTTP_PORT="8080"
MGMT_PORT="9000"
HTTP_URL="http://localhost:${HTTP_PORT}"
MGMT_URL="http://localhost:${MGMT_PORT}"
# Path to the openapi-mock binary
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="${PROJECT_DIR}/bin/openapi-mock"
# PID of the server process
SERVER_PID=""
# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    echo -e "${GREEN}Cleanup complete${NC}"
}
# Set trap for cleanup on exit
trap cleanup EXIT
# Helper function to print test results
print_result() {
    local test_name="$1"
    local result="$2"
    if [ "$result" -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name"
        return 1
    fi
}
# Helper function to wait for server to be ready
wait_for_server() {
    local url="$1"
    local max_attempts=30
    local attempt=1
    echo -e "${YELLOW}Waiting for server at ${url}...${NC}"
    while [ $attempt -le $max_attempts ]; do
        if curl -s -o /dev/null -w "%{http_code}" "$url" | grep -q "200"; then
            echo -e "${GREEN}Server is ready${NC}"
            return 0
        fi
        sleep 0.5
        attempt=$((attempt + 1))
    done
    echo -e "${RED}Server failed to start${NC}"
    return 1
}
# Build the binary if needed
build_binary() {
    echo -e "${YELLOW}Building openapi-mock binary...${NC}"
    cd "$PROJECT_DIR"
    go build -o bin/openapi-mock ./cmd/openapi-mock
    echo -e "${GREEN}Build complete${NC}"
}
# Start the OpenAPI mock server
start_server() {
    echo -e "${YELLOW}Starting OpenAPI mock server...${NC}"
    "$BINARY" run "$HTTP_HOST" "$HTTP_PORT" &
    SERVER_PID=$!
    # Wait for both servers to be ready
    wait_for_server "${MGMT_URL}/openapi.json"
}
# Test 1: Check OpenAPI spec endpoint
test_openapi_endpoint() {
    echo -e "\n${YELLOW}Test: OpenAPI endpoint (/openapi.json)${NC}"
    local response
    response=$(curl -s "${MGMT_URL}/openapi.json")
    if echo "$response" | jq -e '.openapi' > /dev/null 2>&1; then
        print_result "OpenAPI spec is valid JSON with openapi field" 0
    else
        print_result "OpenAPI spec validation" 1
        echo "Response: $response"
        return 1
    fi
}
# Test 2: Check Swagger UI endpoint
test_swagger_ui_endpoint() {
    echo -e "\n${YELLOW}Test: Swagger UI endpoint (/doc)${NC}"
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "${MGMT_URL}/doc")
    if [ "$http_code" -eq 200 ]; then
        print_result "Swagger UI returns 200" 0
    else
        print_result "Swagger UI HTTP code" 1
        echo "HTTP code: $http_code"
        return 1
    fi
}
# Test 3: Check logs endpoint initially empty
test_logs_empty_initially() {
    echo -e "\n${YELLOW}Test: Logs empty initially${NC}"
    # Clear logs first
    curl -s -X POST "${MGMT_URL}/clear" > /dev/null
    local response
    response=$(curl -s "${MGMT_URL}/logs")
    if [ "$response" = "[]" ] || [ "$response" = "null" ]; then
        print_result "Logs are empty initially" 0
    else
        print_result "Initial logs check" 1
        echo "Response: $response"
        return 1
    fi
}
# Test 4: Make an HTTP call and check it's recorded
test_http_call_recording() {
    echo -e "\n${YELLOW}Test: HTTP call recording${NC}"
    # Make an HTTP call to the petstore API
    local http_response
    http_response=$(curl -s "${HTTP_URL}/pets")
    echo "HTTP Response: $http_response"
    # Check if the call was recorded
    local logs
    logs=$(curl -s "${MGMT_URL}/logs")
    if echo "$logs" | jq -e '.[0].method' > /dev/null 2>&1; then
        local method
        method=$(echo "$logs" | jq -r '.[0].method')
        print_result "HTTP call was recorded (method: $method)" 0
    else
        print_result "HTTP call recording" 1
        echo "Logs: $logs"
        return 1
    fi
}
# Test 5: Test clear endpoint
test_clear_logs() {
    echo -e "\n${YELLOW}Test: Clear logs endpoint${NC}"
    # Clear logs
    local clear_response
    clear_response=$(curl -s -X POST "${MGMT_URL}/clear")
    # Check logs are empty
    local logs
    logs=$(curl -s "${MGMT_URL}/logs")
    if [ "$logs" = "[]" ] || [ "$logs" = "null" ]; then
        print_result "Logs cleared successfully" 0
    else
        print_result "Clear logs" 1
        echo "Logs after clear: $logs"
        return 1
    fi
}
# Main test runner
main() {
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}OpenAPI Mock Management Server Tests${NC}"
    echo -e "${YELLOW}========================================${NC}"
    local failed=0
    # Build and start server
    build_binary
    start_server
    # Run tests
    test_openapi_endpoint || failed=$((failed + 1))
    test_swagger_ui_endpoint || failed=$((failed + 1))
    test_logs_empty_initially || failed=$((failed + 1))
    test_http_call_recording || failed=$((failed + 1))
    test_clear_logs || failed=$((failed + 1))
    # Summary
    echo -e "\n${YELLOW}========================================${NC}"
    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}$failed test(s) failed${NC}"
        exit 1
    fi
}
# Run main function
main "$@"
