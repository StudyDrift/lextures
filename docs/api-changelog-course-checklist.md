# API changelog — Course Checklist (CC.2)

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
