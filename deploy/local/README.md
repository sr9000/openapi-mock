# Local Docker Compose assets

This directory is the canonical home for local deployment assets.

## Layout

- `dev/docker-compose.yaml` — hot-reload development stack.
- `observability/docker-compose.yaml` — full local stack with Prometheus, Grafana, Loki, Tempo, OTel Collector, and Promtail.
- `.env.example` — sample environment overrides for both local compose entrypoints.
- `observability/prometheus/` — Prometheus scrape configuration.
- `observability/grafana/` — Grafana image + provisioning (datasources and dashboards).
- `observability/otel/` — OpenTelemetry Collector config.
- `observability/promtail/` — Promtail config.
- `observability/tempo/` — Tempo config.

## Recommended entrypoints

From the repository root:

```bash
make docker-dev
make compose-up
make compose-smoke
make compose-down
```

If you prefer raw Docker Compose commands:

```bash
docker compose -f deploy/local/dev/docker-compose.yaml up --build
docker compose -f deploy/local/observability/docker-compose.yaml --progress plain build
docker compose -f deploy/local/observability/docker-compose.yaml up -d
```

## Environment overrides

The preferred override file is `deploy/local/.env`.
The `make` targets pass it to Docker Compose automatically when it exists.
For backward compatibility, a repo-root `.env` is also accepted if `deploy/local/.env` is absent.

```bash
cp deploy/local/.env.example deploy/local/.env
```
