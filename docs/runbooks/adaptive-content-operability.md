# Runbook — Adaptive Content Operability (AC.9)

SRE dashboards: Grafana → **Lextures — Adaptive Content Engine (AC.9)** (`deploy/observability/grafana/dashboards/adaptive-content.json`).

Instructor report: Course settings → Adaptive content → **Report** tab.  
Admin org rollup: Settings → AI governance → Adaptive content oversight / org report API.

## Alerts

### ACEQueueDepthHigh

Pending `adaptive_content_jobs` > 100 for 10m.

1. Check worker is running (`go run ./cmd/server` / API process with job workers).
2. Inspect `lextures_adaptive_content_inflight` and generation pause (`GET /api/v1/admin/adaptive-content`).
3. Look for `ACEGenerationRetryStorm` or provider errors on the AI Provider dashboard.
4. Optionally pause generation: `PATCH /api/v1/admin/adaptive-content` `{ "generationPaused": true }`.

### ACECacheHitRateLow

Cache hit/(hit+miss) < 50% for 15m.

1. Check for mass content-version bumps (base edits) causing regenerations.
2. Confirm prewarm is running for active units.
3. Review holdout % — high holdout reduces adapted cache usefulness but should not tank hit rate alone.

### ACEGenerationRetryStorm

Job retries > 0.5/s for 10m.

1. Check AI provider error rate (`lextures_ai_provider_calls_total{outcome="error"}`).
2. Verify rate limiter / budget not thrashing.
3. Inspect dead-letter / cancelled jobs in `course.adaptive_content_jobs`.

### ACEBudgetExhausted

Budget-skip counter increased in the last 15m.

1. Identify course(s) via instructor budget meter / `ai_usage_log` (`feature=adaptive_content`).
2. Raise `monthlyTokenBudget` or wait for period rollover.
3. Pause generation if spend is unexpected.

### ACEFidelityRejectionSpike

Fidelity rejects / generations > 25% for 10m.

1. Sample rejected variants in the review queue.
2. Check prompt/key-term changes and model routing.
3. Quarantine a unit if quality is unsafe: `POST /api/v1/admin/adaptive-content/quarantine`.

### ACEUnitRegressing

A unit transitioned to `verdict=regressing` (AC.7).

1. Open the instructor report → Units to review (regressing first).
2. Review variants / pause the unit; consider quarantine.
3. Link methodology: AC.7 effectiveness docs.

### ACEDisparityFlag

Fairness audit raised a disparity flag (AC.8).

1. `GET /api/v1/admin/adaptive-content/fairness?course=…`
2. Follow [adaptive-content-governance.md](adaptive-content-governance.md).

### ACEKillSwitchEngaged

See [adaptive-content-kill-switch.md](adaptive-content-kill-switch.md).

## SLOs (GA)

| SLO | Target | Panel |
|---|---|---|
| Serve latency p95 | ≤ 50 ms resolution (page still dominated by content fetch) | Serve latency p95 |
| Adapted availability with base fallback | ≥ 99.9% of serve decisions return content | SLO — adapted availability |
| Generation success (non-retry) | ≥ 95% over 15m | SLO — generation success rate |
| Report read p95 | ≤ 300 ms from caches | API route group for `/adaptive-content/report` |

## Refresh-job health

- Effectiveness: `scheduled.adaptive_content_effectiveness` → also refreshes coverage + course report matview (AC.9).
- Fairness: `scheduled.adaptive_content_fairness`.
- Instructor on-demand: `POST …/effectiveness/refresh`.
- Report “data as of” timestamp comes from the latest coverage/matview refresh.
