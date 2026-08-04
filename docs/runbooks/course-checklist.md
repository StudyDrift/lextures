# Course Checklist — retire a misbehaving rule

Plans: **CC.1** (engine), **CC.2** (API / dismissals). Ops path when a rule nags incorrectly
or panics in production.

## Purpose

Stop a single checklist item from affecting instructors without disabling the whole checklist
(there is no feature flag).

## Control

| Mechanism | Location | Effect |
|---|---|---|
| `RETIRED_ITEM_IDS` | `server/internal/service/coursechecklist/registry.go` | `ResolveItemID` returns false; CC.2 filters retired IDs from responses |
| Tier demotion | Rule `Tier: recommended` | Removes item from nav badge (`OutstandingEssential`) after snapshot refresh |
| `EngineVersion` bump | `EngineVersionConst` | Invalidates all CC.2 cached snapshots |

## Procedure — retire fast

1. Identify the `item_id` from logs/metrics:
   - counter `lextures_coursechecklist_rule_errors_total{item_id,kind}`
   - logs include `item_id` and `catalog_version`
2. Add the ID to `RETIRED_ITEM_IDS` (and remove the descriptor from the builtin registry if it
   should no longer evaluate).
3. Ship a server release. No client deploy required.
4. Confirm `/metrics` no longer increments errors for that ID and CC.2 responses omit it.
5. Optionally bump `EngineVersionConst` if cached snapshots must drop immediately.

## Procedure — demote without retiring

Change the descriptor `Tier` from `essential` to `recommended` and deploy. Badge counts fall
on the next snapshot TTL; dismissals remain valid.

## Rollback

Revert the rules/`RETIRED_ITEM_IDS` change. If `EngineVersion` was bumped, leave it bumped
(monotonic) or accept a second bump on restore.

## Tune snapshot TTL

| Env | Default | Effect |
|---|---|---|
| `CHECKLIST_SNAPSHOT_TTL` | `15m` | How long a warm snapshot serves `/checklist` and `/summary` without re-evaluating |
| `CHECKLIST_EVIDENCE_MAX_ROWS` | `200` | Evidence row cap per finding (CC.1/CC.2) |

## Force-refresh a course

`POST /api/v1/courses/{course_code}/checklist/refresh` (staff) bypasses TTL. Rate-limited to 6/min/course.

## Inspect dismissals

```sql
SELECT item_id, dismiss_reason, dismiss_note, dismissed_at, dismissed_by_user_id
FROM course.course_checklist_item_state
WHERE course_id = $1 AND dismissed_at IS NOT NULL;

SELECT item_id, action, actor_user_id, reason, occurred_at
FROM course.course_checklist_events
WHERE course_id = $1
ORDER BY occurred_at DESC
LIMIT 100;
```

Nightly sweeper `scheduled.course_checklist_retention` deletes untouched snapshots (90d) and aged events (400d).

## Related

- Dev guide: [course-checklist-engine.md](../dev/course-checklist-engine.md)
- ADR: [0003-course-checklist-code-registry.md](../adr/0003-course-checklist-code-registry.md)
- Plan: [CC.2](../completed/checklist/CC.2-checklist-state-api-and-dismissals.md)
