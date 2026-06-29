# Production deploy — canary, promote, rollback (plan 17.9)

This runbook covers zero-downtime rolling restarts, blue/green swaps, and automated canary analysis for the Lextures API on EKS (production) and the local multi-instance stack used in CI.

## Strategies

| Strategy | When to use | Traffic shift | Automated analysis |
|----------|-------------|---------------|-------------------|
| `rolling` | Routine patch releases | In-place pod/container restart | No |
| `blue-green` | Schema-compatible releases | 0% → 100% via LB | Manual promote |
| `canary` | Risky changes | 5% → 25% → 100% (configurable) | Prometheus (auto promote/rollback) |

Trigger via GitHub Actions: **Deploy Production** workflow (`workflow_dispatch`).

## Prerequisites

- Migrations are backward-compatible (plan 17.10 expand/contract) when running blue/green or canary.
- `/health/ready` passes on new instances before traffic shifts (plan 17.8).
- Prometheus scrapes `deploy_color` labels on HTTP metrics (plan 17.9).
- GitHub Environment approval configured for `staging` / `production`.

## Canary deploy procedure

1. **Start deploy** — workflow builds images tagged with `${GITHUB_SHA}` and `${GITHUB_SHA}-${timestamp}`.
2. **Notify** — Slack/email on deploy start (`deploy/scripts/notify_deploy.sh start`).
3. **Annotate Grafana** — vertical deploy marker (`deploy/scripts/grafana_annotate.sh`).
4. **Provision green** — deploy green replicas with `DEPLOY_COLOR=green` and new image.
5. **Shift canary traffic** — `deploy/scripts/traffic_split.sh terraform <env> <percent>` (default 5%).
6. **Analyze** — `deploy/scripts/canary_analyze.py` for 10 minutes (configurable):
   - **Promote** when error rate &lt; 0.5% and p95 within 10% of blue baseline.
   - **Rollback** when green error rate &gt; 1% for 3 consecutive minutes.
7. **Promote or rollback** — set traffic to 100% green or 100% blue; tear down losing color.
8. **Notify** — Slack/email on promote, rollback, or failure.

## Rolling restart (zero downtime)

```bash
# EKS production
deploy/scripts/rolling_restart.sh kubectl lextures lextures-api

# Local / CI validation
docker compose -f docker-compose.scale.yml up --scale api=2 -d
deploy/scripts/rolling_restart.sh compose docker-compose.scale.yml api proxy
```

The load balancer (Caddy locally, ALB in production) removes instances whose `/health/ready` fails. `SHUTDOWN_TIMEOUT_SECS=30` drains in-flight requests before SIGKILL.

## Pipeline inputs

| Input | Default | Description |
|-------|---------|-------------|
| `environment` | `staging` | `staging` or `production` |
| `strategy` | `canary` | `rolling`, `blue-green`, or `canary` |
| `canary_percent` | `5` | Initial canary traffic share (0–100) |
| `canary_window_minutes` | `10` | Analysis window |
| `skip_canary_analysis` | `false` | Manual promote after traffic shift |

## Verification

- CI runs `deploy/scripts/test_rolling_restart_zero_downtime.sh` — zero HTTP 502/503 during rolling restart (AC-1).
- Unit tests: `python3 -m unittest deploy/scripts/test_canary_analyze.py`.

## References

- `deploy/scripts/` — pipeline scripts
- `deploy/k8s/` — Kubernetes manifests
- `iac/production/deploy-traffic.tf` — LB weight variables
- `docs/runbooks/emergency-rollback.md` — fast rollback to previous image tag
