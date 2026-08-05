# Course checklist — reporting queries (CC.10 FR-17)

Refresh at least weekly. Sources: client event stream (warehouse when wired),
`course.course_checklist_events`, and Prometheus server metrics.
All queries are **bounded by time range**.

## 1. Pass rate per rule

From Prometheus (preferred for live ops):

```promql
sum by (item_id) (rate(lextures_coursechecklist_item_status_total{status="done"}[7d]))
/
clamp_min(
  sum by (item_id) (rate(lextures_coursechecklist_item_status_total{status=~"done|todo|in_progress"}[7d])),
  1e-9
)
```

Warehouse sketch from client snapshots (if you store evaluation summaries):

```sql
-- Adapt to your product_events / metrics warehouse schema
SELECT
  item_id,
  COUNT(*) FILTER (WHERE status = 'done')::float
    / NULLIF(COUNT(*) FILTER (WHERE status IN ('done','todo','in_progress')), 0) AS pass_rate,
  COUNT(*) AS samples
FROM checklist_item_status_samples  -- materialised from Prometheus remote-write or eval jobs
WHERE ts >= NOW() - INTERVAL '7 days'
GROUP BY item_id
ORDER BY pass_rate ASC NULLS FIRST;
```

## 2. Dismissal rate per rule by reason

```sql
SELECT
  item_id,
  reason,
  COUNT(*) AS dismissals
FROM course.course_checklist_events
WHERE action = 'dismiss'
  AND occurred_at >= NOW() - INTERVAL '7 days'
  AND item_id NOT IN ('accommodations.honored', 'accommodations.reviewed')
GROUP BY item_id, reason
ORDER BY dismissals DESC;
```

Disagree rate (FR-19 / FR-20 gate):

```sql
WITH d AS (
  SELECT item_id,
    COUNT(*) FILTER (WHERE reason = 'disagree') AS disagree_n,
    COUNT(*) AS total_n
  FROM course.course_checklist_events
  WHERE action = 'dismiss'
    AND occurred_at >= NOW() - INTERVAL '7 days'
  GROUP BY item_id
)
SELECT item_id,
  disagree_n::float / NULLIF(total_n, 0) AS disagree_rate,
  total_n
FROM d
WHERE total_n >= 20
ORDER BY disagree_rate DESC;
```

## 3. Time-to-completion per rule

Approximate: first `todo`/`in_progress` observation → first `done` for the same course+item
(requires warehouse of evaluation samples or client `checklist_item_rechecked` / dismiss timeline).

```sql
-- Using dismiss/restore history as a coarse proxy is not equivalent to "fixed";
-- prefer evaluation samples when available.
SELECT item_id,
  percentile_cont(0.5) WITHIN GROUP (ORDER BY hours_to_done) AS p50_hours,
  percentile_cont(0.9) WITHIN GROUP (ORDER BY hours_to_done) AS p90_hours
FROM checklist_item_time_to_done  -- derived table
WHERE completed_at >= NOW() - INTERVAL '30 days'
GROUP BY item_id;
```

## 4. Badge-count distribution

```promql
# outstandingEssential is returned on /summary; instrument a histogram at the API if needed.
# Until then, warehouse:
```

```sql
SELECT outstanding_essential, COUNT(*) AS courses
FROM checklist_summary_samples
WHERE ts >= NOW() - INTERVAL '1 day'
GROUP BY outstanding_essential
ORDER BY outstanding_essential;
```

## 5. Target-resolution failure rate

```sql
SELECT
  COUNT(*) FILTER (WHERE (props->>'resolved')::boolean = false)::float
    / NULLIF(COUNT(*), 0) AS unresolved_rate,
  COUNT(*) AS navigations
FROM product_events
WHERE event = 'checklist_target_navigated'
  AND ts >= NOW() - INTERVAL '24 hours';
```

Alert when unresolved_rate > 1% over 24 h (FR-19).

## 6. Assisted-fix acceptance rate

```sql
SELECT
  props->>'actionKind' AS action_kind,
  SUM((props->>'acceptedCount')::int)::float
    / NULLIF(SUM((props->>'proposedCount')::int), 0) AS acceptance_rate,
  COUNT(*) AS assist_sessions
FROM product_events
WHERE event = 'checklist_assist_accepted'
  AND ts >= NOW() - INTERVAL '7 days'
GROUP BY 1;
```

Target ≥ 60% for outcome mappings before promoting the assist (CC.10 §11).

## 7. Snapshot miss ratio (CC.2 / FR-19)

```promql
sum(rate(lextures_coursechecklist_snapshot_hits_total{result="miss"}[1h]))
/
clamp_min(sum(rate(lextures_coursechecklist_snapshot_hits_total[1h])), 1e-9)
```

Alert when > 40%.
