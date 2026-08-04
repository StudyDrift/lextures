# CC.2 — Checklist State, API & Dismissals

> Implementation plan. Source: Course Checklist product request. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CC.2 |
| **Section** | Course Checklist |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Server / platform team |
| **Depends on** | CC.1 |
| **Unblocks** | CC.3, CC.4, CC.5, CC.6, CC.7, CC.9, CC.10 |

---

## 1. Problem Statement

CC.1 computes findings but forgets everything the moment the request ends. The checklist needs memory: an
instructor who deliberately does not want an outcomes-mapping nag on a pass/fail seminar must be able to
dismiss it once and never see it again, and the nav badge must be servable on every page load without
paying a 400 ms evaluation each time. CC.2 adds the persistence layer (dismissals + cached snapshots), the
authorisation rule that keeps the checklist staff-only, and the HTTP API that web (CC.7) and mobile (CC.9)
both consume.

## 2. Goals

- Persist **per-course, per-item dismissals** with actor, timestamp and optional reason, plus restore.
- Serve a **cheap badge count** (`GET .../checklist/summary`) that any page can call without evaluating.
- Cache evaluation results with correct invalidation on course mutations, engine and catalog version bumps.
- Enforce **teacher-and-higher** visibility with one shared guard used by every checklist route.
- Keep the schema **item-agnostic**: adding checklist item #81 must not require a migration.

## 3. Non-Goals

- No rule definitions (CC.3–CC.6) and no UI (CC.7 / CC.9).
- No per-user dismissals — dismissal is a property of the **course**, so a co-teacher does not re-dismiss
  what the lead teacher already dismissed (see §18 Q1).
- No snooze/remind-me-later in v1 (schema leaves room; see §18 Q2).
- No org-level policy forcing an item to be undismissable (see §18 Q3).
- No websocket push; CC.7 refreshes on navigation and on mutation (see §18 Q4).

## 4. Personas & User Stories

- **As a teacher**, I want to dismiss "Set up sections" on a course that will never have sections, so that
  my list reflects reality.
- **As a teacher**, I want to find something I dismissed by mistake and restore it, so that a mis-click is
  not permanent.
- **As a co-teacher**, I want to see that my colleague dismissed an item and why, so that we do not argue
  about a missing nag.
- **As a student**, I want to never see the checklist or its badge, so that instructor to-dos do not leak.
- **As an org admin**, I want the checklist API to answer for courses I administer, so that support can
  reproduce what a teacher sees.
- **As an SRE**, I want the badge endpoint to be O(1) against a cache, so that adding it to the shell does
  not multiply database load.

## 5. Functional Requirements

### Authorisation

- **FR-1.** Every checklist route MUST require an authenticated viewer **and** one of:
  (a) the `course:{course_code}:item:create` permission for the course (course `owner`, `instructor`/
  `teacher`, and `designer` hold this via `courseroles`), or (b) an org/platform admin role scoped to the
  course's org. Course roles `student`, `ta`, `observer`, `auditor`, `librarian` and parents MUST receive
  `403`.
- **FR-2.** The guard MUST be a single helper `requireCourseChecklistAccess(w, r) (courseCode, userID, ok)`
  in `server/internal/httpserver/course_checklist.go`, and MUST be used by every route in this plan.
- **FR-3.** Failure modes MUST use `apierr` codes: `401` unauthenticated, `403` forbidden, `404` unknown
  course code — and MUST NOT distinguish "course does not exist" from "you may not see this course".

### Persistence

- **FR-4.** The system MUST add `course.course_checklist_item_state` keyed by `(course_id, item_id)`
  recording dismissal metadata. `item_id` is `TEXT` — **no enum, no per-item column**.
- **FR-5.** Dismiss MUST be idempotent: dismissing an already-dismissed item returns `200` and does not
  change `dismissed_at`.
- **FR-6.** Restore MUST clear the dismissal and stamp `restored_at` / `restored_by_user_id`, retaining the
  previous dismissal in `course.course_checklist_events` for audit.
- **FR-7.** Dismissing an item whose ID is unknown or retired MUST return `404` with code `not_found`.
- **FR-8.** The system MUST add `course.course_checklist_snapshots` (one row per course) holding the
  serialized `Result`, `computed_at`, `engine_version`, `catalog_version`, and denormalised counters.
- **FR-9.** A snapshot MUST be considered stale when any of: `engine_version` ≠ current, `catalog_version` ≠
  current, `computed_at` older than `CHECKLIST_SNAPSHOT_TTL` (default 15 min), or `course.courses.updated_at`
  / max structure-item `updated_at` is newer than `computed_at`.
- **FR-10.** Reading the checklist MUST recompute when stale, write the snapshot, and return fresh data.
  Concurrent recomputation for the same course MUST be collapsed with a per-course single-flight so a
  thundering herd issues one evaluation.
- **FR-11.** Snapshot writes MUST be best-effort: a write failure MUST log and still return the freshly
  computed result.

### API behaviour

- **FR-12.** `GET /api/v1/courses/{course_code}/checklist` MUST return categories in registry order, items
  in registry order, each with resolved status, evidence, target, tier, sources and dismissal state; items
  with `not_applicable` MUST be omitted from the default response and included when `?includeNotApplicable=1`.
- **FR-13.** Dismissed items MUST be returned in a separate `dismissed[]` collection (the "dismissed pile"),
  never inline in their category.
- **FR-14.** `GET /api/v1/courses/{course_code}/checklist/summary` MUST return
  `{ outstandingEssential, outstandingTotal, done, total, dismissed, computedAt, stale }` and MUST be
  servable from a valid snapshot **without** evaluating.
- **FR-15.** `POST /api/v1/courses/{course_code}/checklist/items/{item_id}/dismiss` MUST accept
  `{ reason?: string, note?: string }` where `reason ∈ {not_applicable, done_elsewhere, disagree, later,
  other}` and `note` ≤ 500 chars, and MUST return the updated item.
- **FR-16.** `POST /api/v1/courses/{course_code}/checklist/items/{item_id}/restore` MUST undo a dismissal.
- **FR-17.** `POST /api/v1/courses/{course_code}/checklist/refresh` MUST force recomputation, bypassing TTL,
  rate-limited to 6 requests / course / minute.
- **FR-18.** `POST /api/v1/courses/{course_code}/checklist/items/{item_id}/recheck` MUST re-evaluate one item
  via `EvaluateOptions.Only` and patch it into the stored snapshot, so acting on an item updates it without a
  full pass.
- **FR-19.** Every response MUST carry `catalogVersion` and `engineVersion` so clients can invalidate their
  own caches.
- **FR-20.** All routes MUST be added to the OpenAPI document (`server/internal/openapi/openapi.json`) and
  to `server/internal/httpserver/testdata/route_inventory.golden`.
- **FR-21.** Course deletion / factory reset MUST cascade-delete checklist state; course **copy** and
  blueprint sync MUST NOT copy dismissals (a new course starts with a clean list).
- **FR-22.** Course export MUST NOT include checklist state; import MUST NOT create it.

## 6. Non-Functional Requirements

- **Performance** — `GET .../summary` p95 < 40 ms served from snapshot. `GET .../checklist` p95 < 60 ms warm,
  < 500 ms cold (one evaluation). Dismiss/restore p95 < 80 ms. Single-flight MUST cap concurrent evaluations
  per course at 1 and per process at 32.
- **Security** — FR-1 guard on every route; `item_id` path segment validated against `ResolveItemID` before
  any query (no unbounded text into the table). `note` sanitised and length-capped. Rate limits: dismiss 60/min
  per user, refresh 6/min per course, via the existing `ratelimit` middleware. Audit rows written for every
  mutation.
- **Privacy & Compliance** — FERPA: checklist evidence can name students; access is limited to course staff
  (FR-1) which matches the existing enrollments-read boundary. Dismissal notes are staff-authored free text —
  they are covered by the existing course-data retention policy and are deleted with the course. Notes MUST
  be included in the org DSAR export for the authoring staff member (`standards/` DSAR orchestration).
- **Accessibility** — N/A (API). Error copy MUST be plain language for client display.
- **Scalability** — One snapshot row per course; payload capped at 256 KB (enforced with a check constraint
  and a server-side guard that drops evidence before the payload if exceeded). At 1 M courses the table is
  ~50 GB worst case — payload is therefore stored `jsonb` compressed and the TTL sweeper (below) deletes
  snapshots for courses untouched for 90 days.
- **Reliability** — Snapshot writes best-effort (FR-11). Dismiss/restore are transactional and idempotent.
  A background sweeper (`server/internal/workers`) deletes stale snapshots nightly; its failure is
  non-fatal because staleness is also checked at read time.
- **Observability** — Metrics: `coursechecklist_api_requests_total{route,status}`,
  `coursechecklist_snapshot_hits_total{result=hit|stale|miss}`, `coursechecklist_dismissals_total{reason}`,
  `coursechecklist_singleflight_waiters`. Log fields: `course_id`, `item_id`, `actor_user_id`, `action`.
  Alert when `snapshot_hits_total{result="miss"}` ratio > 40% over 15 min (cache not working).
- **Maintainability** — Handlers in `course_checklist.go` / `course_checklist_state.go`; routes registered
  from a dedicated `registerCourseChecklistRoutes` invoked by `registerCourseRoutes` (keeps
  `courses_routes.go` from growing; consistent with TD.6 direction).
- **Internationalization** — The API returns i18n keys **and** English defaults from CC.1; clients pick.
  `Accept-Language` is honoured for server-rendered defaults where `l10n` has a translation.
- **Backward compatibility** — `catalogVersion` in every response. Unknown item IDs from an older client are
  ignored on read and `404` on write. Adding fields to the item object is additive only.

## 7. Acceptance Criteria

- **AC-1.** *Given* a student enrolled in a course, *When* they call `GET /checklist`, *Then* they receive
  `403` and no body fields leak item titles.
- **AC-2.** *Given* a `ta` enrollment, *When* they call `GET /checklist`, *Then* they receive `403`.
- **AC-3.** *Given* a teacher, *When* they call `GET /checklist` twice within the TTL, *Then* the second call
  is served from the snapshot and `coursechecklist_snapshot_hits_total{result="hit"}` increments.
- **AC-4.** *Given* a teacher dismisses `people.sections` with reason `not_applicable`, *When* they re-read
  the checklist, *Then* the item appears only under `dismissed[]`, the badge count drops by one, and an audit
  event exists.
- **AC-5.** *Given* a dismissed item, *When* the teacher restores it, *Then* it returns to its category with
  its live status and `restored_at` is set.
- **AC-6.** *Given* the course's `updated_at` changes, *When* the checklist is read, *Then* the snapshot is
  recomputed even though the TTL has not elapsed.
- **AC-7.** *Given* 20 concurrent cold reads for one course, *When* they are served, *Then* exactly one
  evaluation runs and all 20 receive identical `computedAt`.
- **AC-8.** *Given* `EngineVersion` is incremented, *When* any course checklist is read, *Then* it is
  recomputed and the stored `engine_version` updates.
- **AC-9.** *Given* an unknown `item_id`, *When* dismiss is called, *Then* the response is `404` with
  `code: "not_found"` and no row is created.
- **AC-10.** *Given* a course is factory-reset, *When* the checklist is read, *Then* no dismissals survive.
- **AC-11.** *Given* a course is copied, *When* the copy's checklist is read, *Then* it has zero dismissals.
- **AC-12.** *Given* the OpenAPI document, *When* the openapi test runs, *Then* all seven checklist routes
  are present and the route-inventory golden file matches.
- **AC-13.** *Given* an evaluation that produces a 400 KB payload, *When* the snapshot is written, *Then*
  evidence is dropped to fit 256 KB and `evidenceTruncated: true` is returned.

## 8. Data Model

`server/migrations/461_course_checklist.sql` (+ `.down.sql`):

```sql
CREATE TABLE course.course_checklist_item_state (
    course_id            UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    item_id              TEXT NOT NULL,
    dismissed_at         TIMESTAMPTZ,
    dismissed_by_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    dismiss_reason       TEXT NOT NULL DEFAULT '',
    dismiss_note         TEXT NOT NULL DEFAULT '',
    snoozed_until        TIMESTAMPTZ,           -- reserved; unused in v1 (§18 Q2)
    restored_at          TIMESTAMPTZ,
    restored_by_user_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (course_id, item_id),
    CONSTRAINT course_checklist_item_id_format CHECK (item_id ~ '^[a-z][a-z0-9]*(\.[a-z0-9-]+){1,3}$'),
    CONSTRAINT course_checklist_reason_check CHECK (
        dismiss_reason IN ('', 'not_applicable', 'done_elsewhere', 'disagree', 'later', 'other')),
    CONSTRAINT course_checklist_note_len CHECK (length(dismiss_note) <= 500)
);
CREATE INDEX idx_course_checklist_state_dismissed
    ON course.course_checklist_item_state (course_id) WHERE dismissed_at IS NOT NULL;

CREATE TABLE course.course_checklist_snapshots (
    course_id        UUID PRIMARY KEY REFERENCES course.courses (id) ON DELETE CASCADE,
    computed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    engine_version   INT NOT NULL,
    catalog_version  TEXT NOT NULL,
    payload          JSONB NOT NULL,
    total_count      INT NOT NULL DEFAULT 0,
    done_count       INT NOT NULL DEFAULT 0,
    outstanding_essential INT NOT NULL DEFAULT 0,
    outstanding_total     INT NOT NULL DEFAULT 0,
    dismissed_count  INT NOT NULL DEFAULT 0,
    CONSTRAINT course_checklist_payload_size CHECK (pg_column_size(payload) <= 262144)
);
CREATE INDEX idx_course_checklist_snapshots_computed ON course.course_checklist_snapshots (computed_at);

CREATE TABLE course.course_checklist_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id     UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    item_id       TEXT NOT NULL,
    action        TEXT NOT NULL CHECK (action IN ('dismiss', 'restore', 'complete', 'regress')),
    actor_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    reason        TEXT NOT NULL DEFAULT '',
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_course_checklist_events_course_time
    ON course.course_checklist_events (course_id, occurred_at DESC);
```

- **Backfill**: none — absence of a row means "not dismissed".
- **Cascades**: `ON DELETE CASCADE` from `course.courses` covers delete, factory reset and permanent delete.
  `server/internal/repos/course/factory_reset.go` and `.../coursecopy/copy.go` MUST be updated so copy does
  **not** carry state (FR-21) and reset clears the snapshot.
- **Retention**: nightly sweeper deletes snapshots whose course has no enrollment activity in 90 days and
  `course_checklist_events` rows older than 400 days.

## 9. API Surface

All routes are `/api/v1/courses/{course_code}/checklist…`, all require the FR-1 guard, all documented in
OpenAPI.

| Verb | Path | Purpose |
|---|---|---|
| GET | `/checklist` | Full checklist (categories, items, evidence, dismissed pile) |
| GET | `/checklist/summary` | Badge counts only, snapshot-served |
| POST | `/checklist/refresh` | Force recomputation (rate-limited) |
| POST | `/checklist/items/{item_id}/dismiss` | Dismiss with reason/note |
| POST | `/checklist/items/{item_id}/restore` | Restore a dismissed item |
| POST | `/checklist/items/{item_id}/recheck` | Re-evaluate one item |
| GET | `/checklist/history` | Recent checklist events (audit view, ≤ 100 rows) |

```ts
type ChecklistResponse = {
  courseCode: string
  engineVersion: number
  catalogVersion: string
  computedAt: string          // ISO-8601
  stale: boolean
  evidenceTruncated: boolean
  summary: ChecklistSummary
  categories: Array<{
    id: string                // "foundations"
    titleKey: string; title: string
    items: ChecklistItem[]
  }>
  dismissed: ChecklistItem[]
}

type ChecklistItem = {
  id: string                  // "outcomes.assessment-mapping"
  titleKey: string; title: string
  whyKey: string; why: string
  tier: 'essential' | 'recommended'
  status: 'done' | 'todo' | 'in_progress' | 'unknown' | 'not_applicable'
  detail: string | null
  progress: { done: number; total: number } | null
  sources: string[]           // ["QM 2.4", "OSCQR 45"]
  helpRef: string | null
  target: { route: string; anchor: string | null } | null
  evidence: {
    columns: string[]
    rows: Array<{ label: string; sublabel: string | null; status: string
                  target: { route: string; anchor: string | null } | null }>
    truncatedAt: number | null
  } | null
  dismissal: { dismissedAt: string; byUserId: string; byDisplayName: string
               reason: string; note: string } | null
}

type ChecklistSummary = {
  outstandingEssential: number; outstandingTotal: number
  done: number; total: number; dismissed: number
  computedAt: string; stale: boolean
}
```

Rate limits: `refresh` 6/min/course, `dismiss`+`restore` 60/min/user, `recheck` 30/min/course.

## 10. UI / UX

None in CC.2. The API shape above is the contract for CC.7 (web) and CC.9 (mobile). Two UI-driving decisions
are made here:

1. **Dismissed items are a separate collection**, not a status — so the "dismissed pile" is a first-class
   section rather than a filter the client has to invent.
2. **`summary` is a separate endpoint** so the shell can render the nav badge on every page without pulling
   the full payload.

## 11. AI / ML Considerations

None. No model calls, no prompts, no inference in CC.2.

## 12. Integration Points

- Internal: `server/internal/service/coursechecklist` (CC.1), `server/internal/httpserver` (new
  `course_checklist*.go`, `registerCourseChecklistRoutes` wired from `courses_routes.go`),
  `server/internal/apierr`, `server/internal/ratelimit`, `server/internal/authz` + `courseroles`,
  `server/internal/repos/course/factory_reset.go`, `server/internal/service/coursecopy/copy.go`,
  `server/internal/workers` (sweeper), `server/internal/openapi`.
- Route inventory golden: `server/internal/httpserver/testdata/route_inventory.golden`.
- No external services, no webhooks. Analytics event emission is CC.10.

## 13. Dependencies & Sequencing

- Must ship after: CC.1.
- Must ship before: CC.3–CC.6 (they need somewhere for their rules to surface), CC.7, CC.9, CC.10.
- Shared infra: Postgres migration slot `461`, existing rate limiter, existing worker scheduler.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Badge endpoint called on every page load overwhelms DB | M | H | Snapshot-served summary + client-side 60 s memo (CC.7); metric + alert on miss ratio |
| Snapshot payload grows past the size limit as the catalog grows | M | M | 256 KB check constraint + evidence-drop fallback (AC-13); evidence capped at 200 rows in CC.1 |
| Dismissals hide a genuinely broken course from an admin | L | M | `dismissed[]` always returned; `history` route; CC.10 reports dismissal rates per item |
| Course-scoped (not user-scoped) dismissal surprises a co-teacher | M | L | Dismissal shows who dismissed and why; restore is one click |
| Stale detection misses a mutation path (e.g. syllabus edit) | M | M | Staleness also uses a `GREATEST(updated_at)` probe across the snapshot's source tables, plus a 15 min TTL floor |
| Migration `461` collides with a concurrent branch | M | L | Confirm next free number at merge time; migration is additive and idempotent-safe |

## 15. Rollout Plan

**No feature flag.** The routes ship enabled for staff from the first release. Controls available instead:

- `CHECKLIST_SNAPSHOT_TTL` (env, default `15m`) to tune cache pressure without a deploy.
- `CHECKLIST_EVIDENCE_MAX_ROWS` (env, default `200`).
- Sequencing: migration `461` → server release (routes live but registry is near-empty from CC.1) → rule
  packs → clients. Because the catalog is nearly empty at first, the badge reads 0 and the page reads
  "nothing to check", so shipping the API ahead of the UI is safe.
- Dogfood: enable for Lextures-internal demo orgs first by shipping the rule packs there one week earlier
  (rule packs, not the API, are the staged unit).
- GA criteria: p95 summary < 40 ms, snapshot hit ratio > 70%, zero `403` regressions in the authz e2e matrix.
- Rollback: revert the server release; the migration stays (additive, unread by older code).

## 16. Test Plan

- **Unit** — Guard behaviour per role (owner/instructor/designer/ta/observer/auditor/librarian/student/
  parent/org-admin/platform-admin). Staleness predicate truth table. Idempotent dismiss. Unknown/retired
  item handling. Payload-size fallback. Summary derivation from snapshot counters.
- **Integration** — DB tests for the three tables: cascade on course delete, factory-reset clearing, copy not
  carrying state, constraint violations (bad item id, oversized note, oversized payload). Single-flight test
  with 20 goroutines (AC-7). Sweeper deletes only what it should.
- **End-to-end** — Playwright API-level specs in `e2e/tests`: teacher dismisses → badge decrements →
  restores → badge increments; student receives 403; org admin receives 200.
- **Security** — Authz matrix test asserting `403` for every non-staff role and every route; rate-limit
  tests; injection test passing `../../` and SQL metacharacters as `item_id`; assert error bodies are
  identical for "no such course" and "not authorised".
- **Accessibility** — N/A.
- **Performance / load** — k6 script hitting `/summary` at 200 rps for a warm course asserting p95 < 40 ms;
  cold-read burst test asserting single-flight collapse.
- **Manual exploratory** — QA checklist covering: dismiss with each reason, 500-char note boundary, restore
  after catalog version bump, behaviour when a dismissed item's rule is retired.

## 17. Documentation & Training

- API reference: OpenAPI entries with examples for every route; entry in `docs/api-changelog-*.md`.
- Instructor help-centre article: "Dismissing checklist items" (what dismissal means, that it is
  course-wide and visible to co-teachers).
- Admin doc: who can see the checklist and why TAs cannot.
- Runbook `docs/runbooks/course-checklist.md`: tune TTL, force-refresh a course, retire a rule, inspect
  `course_checklist_events`.

## 18. Open Questions

1. Course-scoped vs per-user dismissal — proposed course-scoped (co-teachers share one list). Revisit if
   CC.10 telemetry shows dismissal churn between co-teachers.
2. Snooze ("remind me in 2 weeks") — column reserved, behaviour deferred. Needs a decision on whether the
   badge counts snoozed items.
3. Should an org be able to mark an item **mandatory** (undismissable) for accreditation? Likely a section-18
   Admin Experience follow-up; the schema supports it via a future `org_checklist_policy` table.
4. Live updates: is polling `/summary` on route change enough, or should the checklist ride the existing
   course structure websocket (`/structure/ws`)? Proposed: polling in v1, websocket considered in CC.7.
5. Do blueprint children inherit parent dismissals? Proposed: no.
6. Retention of `course_checklist_events` at 400 days — confirm against the org retention policy engine in
   `docs/plan/standards/`.

## 19. References

- Existing files this work touches: `server/internal/httpserver/courses_routes.go`,
  `server/internal/httpserver/course_outcomes_report.go` (staff-guard pattern),
  `server/internal/repos/course/factory_reset.go`, `server/internal/service/coursecopy/copy.go`,
  `server/internal/openapi/openapi.json`, `server/migrations/`.
- Precedent in-repo: [PS.2 pinned-settings data model & API](../../completed/settings/PS.2-pinned-settings-data-model-and-api.md)
  (string-keyed per-user state, no per-setting migration).
- Related plans: [CC.1](../../completed/checklist/CC.1-checklist-registry-and-evaluation-engine.md),
  [CC.7](CC.7-web-checklist-page-and-nav-badge.md), [CC.9](CC.9-mobile-checklist-ios-and-android.md),
  [CC.10](CC.10-analytics-guidance-and-rollout.md).
