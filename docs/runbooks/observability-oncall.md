# Runbook: Observability & on-call response (plan 17.7)

How to use Grafana, Prometheus alerts, traces, and Sentry during on-call.

## Where things live

- **Grafana** (dashboards): HTTP Overview, Database & Cache, Job Queue, AI
  Provider Usage, Business Metrics. Folder: *Lextures*.
- **Prometheus** (metrics + alert rules): scrapes each API instance's internal
  `/metrics` port. Rules in `deploy/observability/prometheus/alerts.yml`.
- **Tempo** (traces): linked from Grafana. Search by the `X-Trace-Id` a user
  quotes from a failed request.
- **Sentry** (errors/panics): ERROR-level logs and panics, PII-scrubbed.

## Getting a trace from a user report

Every response carries an `X-Trace-Id` header. Ask the reporter for it (or read
it from the access log), then open Grafana → Explore → Tempo and search the
trace ID to see the full request span tree, including DB queries.

## Alert playbook

One section per alert in `alerts.yml`. Each alert carries a `runbook`
annotation linking back here.

### HighErrorRate
*5xx error rate > 1% for 5m.* Check the HTTP Overview "Error rate" panel and the
"Top routes" panel to find the failing endpoint. Pull a recent trace for that
route from Tempo and check Sentry for a matching exception. Common causes:
bad deploy (roll back), a downstream dependency (DB/Redis/AI provider) failing,
or a migration issue. Mitigate by rolling back the latest release.

### HighP95Latency
*p95 latency > 1s for 5m.* Check the latency panel. Correlate with the Database
& Cache dashboard — a rising `lextures_db_pool_utilization_ratio` or Redis miss
spike usually explains it. Look for slow DB spans in Tempo. Mitigate by scaling
out the app tier (17.2) or the database, or by shedding load.

### JobQueueDeadLetterBacklog
*Dead letters > 10.* Open the Job Queue dashboard "Backlog by job type" to find
the failing job type. Inspect dead letters via the admin jobs API, fix the root
cause, then redrive. A single poison message can drive this — cancel it if so.

### DBPoolNearExhaustion
*DB pool utilisation > 90% for 5m.* The app is about to start queueing for
connections. Check for a connection leak (acquired stays high with low traffic),
a slow-query storm, or simply insufficient `DB_POOL_MAX_CONNS` for current load.
Scale the database or raise the pool cap (mind Postgres `max_connections` vs.
instance count — 17.2 FR-7).

### APITargetDown
*Prometheus cannot scrape an instance for 2m.* The instance may be down or its
metrics port blocked. Check the instance health/readiness probe and the load
balancer pool. If the instance is serving traffic but not scrapeable, verify
`METRICS_ENABLED`/`METRICS_ADDR` and the VPC firewall rule.

### ReadinessProbeUnhealthy
*`GET /health/ready` returning 503 for 30s.* The load balancer may have removed
this instance from rotation. Check Postgres and Redis connectivity from the
instance (`curl /health/detailed` with a Global Admin JWT for per-component
latency). Common causes: RDS/Postgres outage, Redis unreachable, or the
dedicated health-check pool cannot connect while the main pool is exhausted.
Mitigate by restoring the dependency or restarting the instance after the DB
recovers.

### AIProviderElevatedErrors
*AI provider error rate > 5% for 10m (AP.9).* Open Grafana → **AI Provider**
dashboard; filter by `provider`. Correlate with a recent credential change or
upstream outage. If the spike followed a multi-provider deploy, consider
flag rollback (`AI_PROVIDER_ABSTRACTION_ENABLED=0`) per
[ai-provider-rollback.md](ai-provider-rollback.md). Disable the failing
provider credential if another peer provider can serve traffic.

## Sentry triage

1. Triage by Sentry environment (production vs staging) and severity.
2. Assign to the owning team; link the Sentry issue to a ticket.
3. Confirm the payload is PII-free — events are scrubbed by `before_send`
   (`server/internal/telemetry/sentry.go`); report any leak as a privacy
   incident (10.14 / FERPA).

## Adding a metric or span

- **Metric:** add one field + one registration in
  `server/internal/telemetry/metrics.go` and call its `Observe…`/`Inc…` helper.
  Never put user/course IDs in labels (cardinality + privacy).
- **Span:** `telemetry.Tracer("name").Start(ctx, "op")`; keep attributes PII-free.
