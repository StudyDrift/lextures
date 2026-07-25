# Pinned settings — reporting queries (PS.4 FR-14)

Refresh at least weekly. Source: client event stream (warehouse) + Prometheus server metrics.

## 1. Pins per user (distribution)

From successful `settings_pin_added` / `settings_pin_removed` with `pin_count`, or from `lextures_pinned_settings_pins_gauge` histogram:

```sql
-- Warehouse sketch (adapt to your event table)
SELECT
  pin_count,
  COUNT(DISTINCT user_id) AS users
FROM product_events
WHERE event = 'settings_pin_added'
  AND ts >= NOW() - INTERVAL '7 days'
GROUP BY pin_count
ORDER BY pin_count;
```

Prometheus:

```promql
histogram_quantile(0.5, sum(rate(lextures_pinned_settings_pins_gauge_bucket[7d])) by (le))
histogram_quantile(0.95, sum(rate(lextures_pinned_settings_pins_gauge_bucket[7d])) by (le))
```

## 2. Top pinned settings per surface

```sql
SELECT
  surface,
  setting_id,
  COUNT(*) AS pin_events
FROM product_events
WHERE event = 'settings_pin_added'
  AND ts >= NOW() - INTERVAL '7 days'
GROUP BY surface, setting_id
ORDER BY pin_events DESC
LIMIT 25;
```

Also compare against `settings_suggestion_accepted` to measure suggestion dominance vs organic pins.

## 3. Top searched-then-unmatched queries (hash buckets)

```sql
SELECT
  surface,
  query_hash,
  COUNT(*) AS zero_result_searches
FROM product_events
WHERE event = 'settings_search_zero_results'
  AND ts >= NOW() - INTERVAL '7 days'
GROUP BY surface, query_hash
ORDER BY zero_result_searches DESC
LIMIT 50;
```

Decode only via a **pre-registered allowlist** of setting-related terms hashed at build time (never reverse arbitrary hashes). Example allowlist seeds: `lockdown`, `due date`, `late policy`, `points`, `access code`, `rubric`.

## 4. Settings never pinned / never changed

Join registry IDs against distinct `setting_id` from `settings_pin_added` and `settings_control_changed` for the window. IDs present in the registry but absent from both streams are “hidden or unused” candidates for the FR-15 review.

## 5. Save health

```promql
sum(rate(lextures_pinned_settings_rejects_total[1d])) by (reason)
/
clamp_min(sum(rate(lextures_pinned_settings_writes_total[1d])), 1e-9)
```

Client: `settings_pin_save_failed` / (`settings_pin_added` + `settings_pin_removed` + `settings_pin_reordered`) should stay under 0.5 % of pin attempts (GA criterion).

## 6. Hidden-settings review (FR-15)

**Due:** 4 weeks after GA flag default flip.

**Owner:** Product (recorded on the rollout ticket).

**Output:** short write-up recommending concrete IA changes (section reorder, promote a control out of an accordion, merge sections) **or** explicitly “none needed,” with:

- pins-per-user distribution
- top pins vs suggestion accept rate
- top zero-result hash buckets (allowlist-decoded where possible)
- settings never pinned/changed
- opt-out share (if measurable) so conclusions use the right denominator
