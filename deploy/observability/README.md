# Lextures Observability Stack (plan 17.7)

Prometheus metrics, OpenTelemetry traces, Sentry error reporting, Grafana
dashboards, and Prometheus alerting for the Lextures API.

## Application configuration

The API reads these environment variables (see `server/internal/config`):

| Variable | Default | Purpose |
|---|---|---|
| `METRICS_ENABLED` | `true` | Gates the internal `/metrics` server. |
| `METRICS_ADDR` | `:9090` | **Internal** metrics port. Firewall to the VPC; never route via the public LB (FR-1, AC-6). |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | _(empty)_ | OTLP/HTTP collector `host:port`. Empty disables tracing. |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` | Plaintext OTLP to an in-VPC collector. |
| `OTEL_TRACES_SAMPLE_RATIO` | `0.1` | Head-based trace sampling (10% prod, 100% staging). |
| `OTEL_SERVICE_NAME` | `lextures-api` | Service name on metrics/traces. |
| `SENTRY_DSN` | _(empty)_ | Sentry project DSN (separate per environment — FR-4). Empty disables Sentry. |
| `SENTRY_TRACES_SAMPLE_RATE` | `0.1` | Sentry performance-transaction sampling. |
| `APP_VERSION` | _(empty)_ | Build/release id for `build_info` and Sentry release. |

`/metrics` is served on a **separate** HTTP server (`METRICS_ADDR`), not on the
public API port, so it can be firewalled independently and a scrape still
succeeds while the API is saturated.

## Local stack

```bash
docker compose -f docker-compose.yml -f docker-compose.observability.yml up
```

- Grafana: <http://localhost:3000> (admin/admin) — dashboards auto-provisioned.
- Prometheus: <http://localhost:9091>
- Tempo (traces) is wired as a Grafana datasource.

Run the API with `METRICS_ADDR=:9090` and
`OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4318` so Prometheus can scrape it and
spans flow to Tempo.

## Layout

```
prometheus/prometheus.yml   Scrape config (pull model; in-VPC targets)
prometheus/alerts.yml       Alert rules (FR-6): error rate, p95, dead letters, DB pool
otel-collector/config.yml   OTLP receiver → batch → Tempo (PII-scrubbing processor)
tempo/tempo.yml             Local single-binary trace store
grafana/provisioning/       Datasources + dashboard providers
grafana/dashboards/         HTTP Overview, Database & Cache, Job Queue, AI Provider, Business Metrics (FR-5)
```

## Metrics reference

All application metrics are prefixed `lextures_`. Key series:

- `lextures_http_requests_total{method,route,status}` — `route` is the chi route
  pattern and `status` is a 2xx/3xx/4xx/5xx class (bounded cardinality).
- `lextures_http_request_duration_seconds_bucket{method,route}` — latency histogram.
- `lextures_db_pool_*`, `lextures_db_pool_utilization_ratio` — pgx pool.
- `lextures_redis_pool_*` — Redis pool.
- `lextures_job_queue_depth`, `lextures_job_queue_jobs{status}`,
  `lextures_job_queue_depth_by_type{job_type}`, `lextures_job_queue_dead_letters`.
- `lextures_ai_provider_*`, `lextures_ai_estimated_cost_dollars_total{provider,model}`.
- `lextures_business_events_total{event}`.

**No PII** appears in any label or trace attribute (NFR Privacy / FERPA). Adding
a metric is one field + one registration in `server/internal/telemetry/metrics.go`.

## Production

In production (`iac/production/`) Prometheus, Grafana, Alertmanager, and the OTel
Collector run inside the private VPC. Sentry is SaaS; its DSN lives in the
secrets manager (17.17). See `iac/production/observability.tf`.
