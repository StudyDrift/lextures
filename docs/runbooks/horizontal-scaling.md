# Runbook — Horizontal Scaling (Plan 17.2)

The API tier is stateless and scales horizontally behind a load balancer, with a
shared Redis instance providing the cross-node memory layer (JWT revocation
blocklist, rate-limit counters, caches). This runbook covers the operational
levers and the statelessness audit.

## Configuration

| Env var | Default | Purpose |
|---|---|---|
| `REDIS_URL` | _(unset)_ | Shared Redis connection. `rediss://` enables TLS (required by managed Redis). Unset ⇒ single-instance fallback. |
| `REDIS_POOL_MIN` | 5 | Minimum idle Redis connections per instance. |
| `REDIS_POOL_MAX` | 20 | Maximum Redis connections per instance. |
| `DB_POOL_MAX_CONNS` | _(pgx default)_ | Cap pgx pool per instance. Keep `instances × DB_POOL_MAX_CONNS` under Postgres `max_connections` (FR-7 / AC-5). |
| `DB_POOL_MIN_CONNS` | _(pgx default)_ | Warm pgx connections per instance. |
| `SHUTDOWN_TIMEOUT_SECS` | 30 | Graceful-shutdown drain window on SIGTERM (FR-8 / AC-4). |

### Sizing the database pool

Postgres `max_connections` is the hard ceiling. With the default managed
instance limit of 100 and a 10-connection reserve for migrations/admin:

```
DB_POOL_MAX_CONNS = floor((max_connections - 10) / max_instances)
```

For 3 instances and `max_connections = 100`: `DB_POOL_MAX_CONNS = 20` ⇒ 60 used,
40 headroom. If you need to scale past that, put PgBouncer (transaction pooling)
between the app and Postgres rather than raising the cap.

## Scaling the app tier

The instance count is managed by the platform (EKS deployment replicas / IaC in
`iac/production`). To change it, adjust the replica count and confirm the load
balancer health check (`GET /health/ready`) shows all instances healthy.
`/health/ready` verifies both Postgres and Redis connectivity, so the LB removes
an instance whose Redis or DB link is down.

## Rolling restart / zero-downtime deploy

1. The LB stops routing new traffic to an instance once `/health/ready` fails or
   the instance is marked draining.
2. On `SIGTERM` the process stops accepting new connections and drains in-flight
   requests for up to `SHUTDOWN_TIMEOUT_SECS` (default 30s) before exiting.
3. Set the orchestrator's termination grace period **≥ `SHUTDOWN_TIMEOUT_SECS`**
   (the scale compose uses `stop_grace_period: 35s`) so the drain completes
   before SIGKILL.

## Emergency single-instance restart

Restarting one instance is safe: the LB routes around it within ~15s of the
first failed health check, and the 30s drain prevents dropped requests. No
maintenance window is required.

## Local multi-instance testing

```bash
docker compose -f docker-compose.scale.yml up --build --scale api=2 -d
curl -fsS http://localhost:8088/health/ready
# roll one instance and watch failover:
docker compose -f docker-compose.scale.yml restart api
```

Caddy (`deploy/caddy/Caddyfile.scale`) round-robins across replicas using Docker
DNS A-records and gates each upstream on `/health/ready`.

## JWT revocation propagation

On logout the presented access token's `jti` is added to Redis under
`session:jti:{jti}` with a TTL equal to the token's remaining lifetime. Every
instance checks this blocklist on `Verify`, so a revoked token is rejected on any
node within the propagation window (AC-2). The DB-backed `jwt_session_version` /
token-invalidation checks remain the hard fallback, so a Redis outage fails
**open** for the blocklist (availability) without weakening session-version
revocation.

## Statelessness audit (FR-2)

Audit of package-level mutable state in `server/internal/httpserver`:

| Symbol | Kind | Verdict |
|---|---|---|
| `altTextRateByUser` | per-user rate counter | **Migrated** to Redis `rate:alttext:*` (in-process kept as Redis-down fallback). |
| `gradingAgentRunStatusCache` (`sync.Map`) | short-lived run-status cache | Acceptable: best-effort UI cache, re-derivable from DB; not correctness-bearing. |
| `*_ws.go` hubs (`collab_docs`, `course_file_manager`, `course_structure`, feed/notif) | per-instance WebSocket/SSE registries | Acceptable: each instance owns its own client sockets. Cross-instance fan-out for these channels is tracked under the realtime/job-queue plans, not 17.2. |
| `validCloudProviders`, `validOERProviders`, `allowedGradingScales`, `supportedLocales`, `tusMimeAllowlist`, `validCadences`, `allowedContentTypes`, `currentLegalVersions` | read-only lookup tables | Safe: immutable after init, no per-request mutation. |

**Policy:** new per-request mutable state (counters, caches, progress) must be
backed by Redis, not package-level vars. Enforced by code review and the
multi-instance integration test (`docker-compose.scale.yml`).
