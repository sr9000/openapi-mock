#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/local/observability/docker-compose.yaml"
COMPOSE_CMD=(docker compose)
COMPOSE_ENV_FILE="$ROOT_DIR/deploy/local/.env"

if [[ ! -f "$COMPOSE_ENV_FILE" && -f "$ROOT_DIR/.env" ]]; then
  COMPOSE_ENV_FILE="$ROOT_DIR/.env"
fi

if [[ -f "$COMPOSE_ENV_FILE" ]]; then
  COMPOSE_CMD+=(--env-file "$COMPOSE_ENV_FILE")
fi

COMPOSE_CMD+=(-f "$COMPOSE_FILE")

cleanup() {
  "${COMPOSE_CMD[@]}" down >/dev/null 2>&1 || true
}
trap cleanup EXIT

wait_for_http() {
  local name="$1"
  local url="$2"
  local retries="${3:-60}"
  local i
  for ((i = 1; i <= retries; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "Timed out waiting for $name at $url" >&2
  return 1
}

wait_for_grafana_api_match() {
  local name="$1"
  local url="$2"
  local pattern="$3"
  local retries="${4:-30}"
  local i
  for ((i = 1; i <= retries; i++)); do
    if curl -fsS -u admin:admin "$url" | grep -q "$pattern"; then
      return 0
    fi
    sleep 2
  done
  echo "Timed out waiting for $name at $url" >&2
  return 1
}

echo "[1/6] Building app image"
"${COMPOSE_CMD[@]}" --progress plain build openapi-mock

echo "[2/6] Starting observability stack"
"${COMPOSE_CMD[@]}" up -d

echo "[3/6] Waiting for core endpoints"
wait_for_http "openapi-mock" "http://127.0.0.1:8080/status"
wait_for_http "metrics" "http://127.0.0.1:9100/metrics"
wait_for_http "prometheus" "http://127.0.0.1:9090/-/healthy"
wait_for_http "loki" "http://127.0.0.1:3100/ready"
wait_for_http "tempo" "http://127.0.0.1:3200/ready"
wait_for_http "collector" "http://127.0.0.1:13133/"
wait_for_http "collector-metrics" "http://127.0.0.1:8888/metrics"
wait_for_http "grafana" "http://127.0.0.1:3000/api/health"

echo "[4/6] Sending plain and traced requests"
resp_headers="$(mktemp)"
curl -fsS -D "$resp_headers" -o /dev/null -X POST "http://127.0.0.1:8080/echo" \
  -H 'Content-Type: application/json' \
  -d '{"message":"plain"}'
request_id="$(grep -i '^X-Request-ID:' "$resp_headers" | awk '{print $2}' | tr -d '\r')"
rm -f "$resp_headers"
if [[ -z "$request_id" ]]; then
  echo "Request ID header was not returned" >&2
  exit 1
fi

curl -fsS -o /dev/null -X POST "http://127.0.0.1:8080/echo" \
  -H 'Content-Type: application/json' \
  -H 'traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01' \
  -d '{"message":"traced"}'

echo "[5/6] Verifying metrics, Prometheus, Grafana, traces, and logs"
if ! curl -fsS "http://127.0.0.1:9100/metrics" | grep -q 'http_requests_total'; then
  echo "Metrics endpoint does not expose http_requests_total" >&2
  exit 1
fi

if ! curl -fsSG --data-urlencode 'query=up{job="openapi-mock"}' "http://127.0.0.1:9090/api/v1/query" | grep -q '"1"'; then
  echo "Prometheus up{job=\"openapi-mock\"} is not 1" >&2
  exit 1
fi

if ! curl -fsS "http://127.0.0.1:8888/metrics" | grep -E 'otelcol_receiver_accepted_spans.* [1-9][0-9]*' >/dev/null; then
  echo "Collector metrics do not show accepted spans" >&2
  exit 1
fi

if ! curl -fsSG --data-urlencode 'query={job="openapi-mock"}' "http://127.0.0.1:3100/loki/api/v1/query" | grep -q '"result"'; then
  echo "Loki query did not return a valid result payload" >&2
  exit 1
fi

for datasource in "HTTP Mock Metrics" "HTTP Mock Traces" "HTTP Mock Logs"; do
  encoded_name="${datasource// /%20}"
  if ! wait_for_grafana_api_match "Grafana datasource $datasource" "http://127.0.0.1:3000/api/datasources/name/$encoded_name" "\"name\":\"$datasource\""; then
    echo "Grafana datasource $datasource was not provisioned" >&2
    exit 1
  fi
done

for dashboard_uid in \
  openapi-mock-endpoints-overview \
  openapi-mock-endpoint-details \
  openapi-mock-resources-overview
do
  if ! wait_for_grafana_api_match "Grafana dashboard $dashboard_uid" "http://127.0.0.1:3000/api/dashboards/uid/$dashboard_uid" "\"uid\":\"$dashboard_uid\""; then
    echo "Grafana dashboard $dashboard_uid was not provisioned" >&2
    exit 1
  fi
done

echo "[6/6] Smoke validation passed"
