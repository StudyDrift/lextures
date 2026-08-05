# Course Checklist — operations runbook

Plans: **CC.1–CC.10**. Ops path when a rule nags incorrectly, analytics fire, or assists misbehave.
There is **no feature flag** — levers are structural (see [ADR 0004](../adr/0004-course-checklist-no-feature-flag.md)).

## Purpose

Stop a single checklist item (or assist) from affecting instructors without disabling the whole checklist.

## Control surface

| Mechanism | Location | Effect |
|---|---|---|
| `RETIRED_ITEM_IDS` | `server/internal/service/coursechecklist/registry.go` | `ResolveItemID` returns false; responses omit the item |
| Tier demotion | Rule `Tier: recommended` | Removes item from nav badge (`OutstandingEssential`) after snapshot refresh |
| `EngineVersion` bump | `EngineVersionConst` | Invalidates all cached snapshots |
| `CHECKLIST_SNAPSHOT_TTL` | env | Warm snapshot lifetime (default `15m`) |
| Link-check kill switch | CC.6 env (see [checklist-linkhealth](../dev/checklist-linkhealth.md)) | Disables outbound URL checks |
| AI opt-out | user `ai_processing_opt_out` | Hides all assisted-fix actions client-side |

## Procedure — retire a misbehaving rule (fast)

1. Identify the `item_id` from logs/metrics:
   - counter `lextures_coursechecklist_rule_errors_total{item_id,kind}`
   - disagree rate queries in [checklist-reporting.md](../completed/checklist/checklist-reporting.md)
2. Add the ID to `RETIRED_ITEM_IDS` (and remove the descriptor from the builtin registry if it should no longer evaluate).
3. Ship a **server-only** release. No client deploy required.
4. Confirm `/metrics` no longer increments errors for that ID and checklist responses omit it.
5. Optionally bump `EngineVersionConst` if cached snapshots must drop immediately.

## Procedure — demote without retiring

Change the descriptor `Tier` from `essential` to `recommended` and deploy. Badge counts fall on the next snapshot TTL; dismissals remain valid. Exercise this path in staging at least once per promotion cycle (FR-22).

## Procedure — “an instructor says the checklist is wrong”

1. Collect: course code, `item_id`, screenshot, dismissal reason if any.
2. Check recent dismiss rates for that rule (`disagree` / `done_elsewhere`).
3. If the heuristic is wrong: demote or retire (above); file a fix against the evaluator.
4. If the course is a special case: coach the instructor to **dismiss** with a reason so co-teachers see context.
5. Do **not** promise a feature-flag kill — use structural levers.

## Force-refresh a course

`POST /api/v1/courses/{course_code}/checklist/refresh` (staff) bypasses TTL. Rate-limited to 6/min/course.

## Assisted fixes

| Action | Endpoint | Writes? |
|---|---|---|
| Suggest outcome mappings | `POST .../outcomes/suggest-links` | No — proposals only; apply via existing link create |
| Build rubric with AI | Existing `.../assignments/{id}/generate-rubric` | Existing flow; human saves |
| Draft welcome | `POST .../feed/draft-welcome` | No — draft only |
| Suggest alt text | Existing `.../alt-text/suggest` | Existing per-image approval |

On assist failure: UI degrades to the manual target link. No partial writes without confirm.

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

## Alerts

See [checklist-alerts.md](../completed/checklist/checklist-alerts.md):

- Target resolution failure > 1% / 24 h
- Rule `disagree` rate > 20% / 7 d
- Snapshot miss ratio > 40%

## Related

- Dev guide: [course-checklist-engine.md](../dev/course-checklist-engine.md)
- ADR registry: [0003](../adr/0003-course-checklist-code-registry.md)
- ADR no-flag: [0004](../adr/0004-course-checklist-no-feature-flag.md)
- Promotion programme: [checklist-promotion-programme.md](../completed/checklist/checklist-promotion-programme.md)
- Event dictionary: [checklist-event-dictionary.md](../completed/checklist/checklist-event-dictionary.md)
- Help hub: [docs/help/course-checklist/](../help/course-checklist/)
