# API changelog — Course Checklist (CC.2 / CC.10)

## 2026-08 — Assisted fixes & action slot (CC.10)

Checklist items may include an optional `action` object (`kind`, `labelKey`, `label`, `endpoint`, `requiresAi`).
Unknown `kind` values must be ignored by clients.

| Verb | Path | Notes |
|---|---|---|
| POST | `/api/v1/courses/{course_code}/outcomes/suggest-links` | Read-only AI proposals for unmapped assessments; **no writes** |
| POST | `/api/v1/courses/{course_code}/feed/draft-welcome` | Read-only welcome announcement draft; **never auto-posts** |

Accepted outcome-mapping proposals still use the existing  
`POST /outcomes/{outcome_id}/links` endpoint (one write per accepted proposal).

AI assists honour opt-out / gateway policy. See [CC.10](completed/checklist/CC.10-analytics-guidance-and-rollout.md).

## 2026-08 — Checklist state API & dismissals

New staff-only routes under `/api/v1/courses/{course_code}/checklist…`:

| Verb | Path | Notes |
|---|---|---|
| GET | `/checklist` | Full checklist; `?includeNotApplicable=1` includes N/A items |
| GET | `/checklist/summary` | Badge counters; served from warm snapshot when possible |
| POST | `/checklist/refresh` | Force recomputation (6/min/course) |
| GET | `/checklist/history` | Recent dismiss/restore audit events (≤ 100) |
| POST | `/checklist/items/{item_id}/dismiss` | Body `{ reason?, note? }` |
| POST | `/checklist/items/{item_id}/restore` | Undo dismissal |
| POST | `/checklist/items/{item_id}/recheck` | Re-evaluate one item into the snapshot |

Authorisation: course `item:create` (owner/instructor/teacher/designer) or org/platform admin.
Students, TAs, observers, and other non-staff roles receive `403`. Unknown/inaccessible courses receive `404` with the same body.

See OpenAPI (`server/internal/openapi/openapi.json`) and [CC.2 plan](completed/checklist/CC.2-checklist-state-api-and-dismissals.md).
