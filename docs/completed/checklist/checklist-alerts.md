# Course checklist — alerting (CC.10 FR-19)

Wire these into Alertmanager / Grafana when the metrics backend is available.
Example receiver config lives under `docs/monitoring/`.

## 1. Target resolution failures

**Condition:** `checklist_target_navigated{resolved=false}` rate > **1%** over **24 h**.

```promql
# Prefer warehouse when client events are exported; until then use a recording rule
# from product analytics export.
```

**Action:** Inspect recent page refactors vs `checklist-targets` / web routes fixture;
fix broken anchors before promoting more rules to `essential`.

## 2. High disagree dismissal rate

**Condition:** for any `item_id`, `disagree` dismissals / all dismissals > **20%** over **7 days**,
with at least 20 dismissals.

```sql
-- See checklist-reporting.md §2
```

**Action:** Manual review of the rule heuristic; demote or retire if wrong (runbook).

## 3. Snapshot miss ratio

**Condition:**

```promql
sum(rate(lextures_coursechecklist_snapshot_hits_total{result="miss"}[1h]))
/
clamp_min(sum(rate(lextures_coursechecklist_snapshot_hits_total[1h])), 1e-9) > 0.40
```

**Action:** Check DB load, evaluation latency, `CHECKLIST_SNAPSHOT_TTL`, singleflight waiters.

## Severity

| Alert | Severity | Page? |
|---|---|---|
| Target resolution > 1% | warning | no |
| Disagree rate > 20% | warning | no |
| Snapshot miss > 40% | warning → critical if sustained 6 h | on-call if critical |
