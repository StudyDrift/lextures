# FB0 — Feedback Data Model, Submission & Admin API

> Implementation plan. Source: Product request — in-app "Share Feedback" mechanism (2026-07-10). Follows [../_TEMPLATE.md](../_TEMPLATE.md). Foundation for [FB1](./FB1-web-share-feedback-button.md), [FB2](./FB2-web-feedback-admin.md), [FB3](./FB3-mobile-share-feedback.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | FB0 |
| **Section** | Feedback — In-App Feedback & Admin Review |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Platform |
| **Depends on** | — |
| **Unblocks** | FB1, FB2, FB3 |
| **Permission** | Submit: any authenticated user. Admin read/update: `global:app:rbac:manage` |

---

## 1. Problem Statement

There is no way for a signed-in user to send product feedback from inside Lextures, and no place for that feedback to land. Support email is the only channel today — out of band, un-queryable, and impossible to triage or trend. Without a store and a contract, none of the client surfaces (web/iOS/Android) or the admin review page can be built. This story creates the durable feedback record and the two API contracts (submit + admin) that the rest of the epic consumes.

## 2. Goals

- Persist a feedback submission as a first-class, queryable database row with the context needed to triage it.
- Expose a single authenticated **submit** endpoint that all three clients share.
- Expose **admin** list + detail + status-update endpoints, gated by `global:app:rbac:manage`.
- Capture source platform, app version, and originating route automatically — no extra user effort.
- Be abuse-resistant (rate-limited, length-capped, sanitized) and privacy-aware (retention, DSAR, deletion).

## 3. Non-Goals

- No unauthenticated / marketing-site feedback path (schema leaves room; not shipped here).
- No feedback UI — buttons/forms are FB1/FB3, the admin page is FB2.
- No email/notification fan-out to admins on new feedback (SHOULD-level, deferred — §18).
- No public roadmap, voting, or duplicate-merge workflow (future).
- No AI auto-categorization or sentiment (future — §11).

## 4. Personas & User Stories

- **As any signed-in user (student, instructor, admin, self-learner)**, I want to submit feedback so my voice reaches the team.
- **As a platform admin**, I want every submission stored with who/where/when context so I can triage it.
- **As a platform admin**, I want to change a submission's status (new → triaged → resolved) so the queue reflects reality.
- **As a privacy officer**, I want feedback tied to a user so it can be exported/erased on a DSAR.

## 5. Functional Requirements

- **FR-1.** The system MUST provide `POST /api/v1/feedback` accepting `{ message, category?, context? }` from any authenticated user, creating one row and returning its id + created timestamp.
- **FR-2.** The system MUST require a non-empty `message` (after trim) and enforce a max length (default 5,000 chars); it MUST reject empty/oversized with `400`.
- **FR-3.** The system MUST persist `user_id` and `org_id` from the authenticated session (never trust client-supplied identity).
- **FR-4.** The system MUST record `source` (`web`|`ios`|`android`, validated against the client-declared value + `User-Agent`), optional `app_version`, and optional `context` (originating route/URL, viewport, locale) as structured metadata.
- **FR-5.** `category` MUST be one of a fixed enum (`bug`, `idea`, `question`, `praise`, `other`); absent/invalid defaults to `other`.
- **FR-6.** Every new row MUST start in status `new`.
- **FR-7.** The system MUST rate-limit submissions per user (default: 10 / 10 min) and return `429` when exceeded.
- **FR-8.** The system MUST provide `GET /api/v1/admin/feedback` — a paginated, filterable (status, category, source, date range, free-text search on message) list, gated by `global:app:rbac:manage`.
- **FR-9.** The system MUST provide `GET /api/v1/admin/feedback/{id}` returning the full record plus submitter display context (name/email, resolved via existing user lookups).
- **FR-10.** The system MUST provide `PATCH /api/v1/admin/feedback/{id}` to update `status` and an internal `admin_note`, recording `resolved_by` + `resolved_at` when moved to a terminal status.
- **FR-11.** All admin endpoints MUST audit access/mutations via the existing admin-audit mechanism.
- **FR-12.** On account deletion / DSAR erasure, the user's feedback rows MUST be deleted or anonymized per platform policy; on DSAR export they MUST be included.

## 6. Non-Functional Requirements

- **Performance** — submit p95 < 300 ms; admin list p95 < 500 ms at 100k rows (indexed, keyset/offset paginated).
- **Security** — server-derived identity only; input sanitized (strip control chars, no HTML execution — stored as plain text, rendered escaped); admin endpoints permission-checked server-side; rate-limited; no SSRF via `context` URLs (stored, never fetched).
- **Privacy & Compliance** — feedback body is user content that may contain PII/FERPA-covered data. Include in DSAR export; delete/anonymize on erasure; apply platform data-retention policy (configurable retention window — §18). Redact obvious secrets on export where feasible.
- **Accessibility** — API only; UI conformance is FB1/FB2/FB3.
- **Scalability** — single append-heavy table; indexes on `(org_id, status, created_at)` and `(created_at)`; category/source as filterable columns.
- **Reliability** — submit is idempotent-friendly via optional client `idempotency_key` (dedupe within a short window); a failed write never blocks the user's app flow (client treats as fire-and-forget with retry).
- **Observability** — metrics `feedback_submitted_total{source,category}`, `feedback_submit_errors_total`, `feedback_admin_list_latency`; trace spans on submit + admin queries; log fields `feedback_id`, `user_id`, `org_id`, `source`. Wire through `internal/telemetry`.
- **Maintainability** — `productfeedback` repo/model packages mirror existing repo conventions; enums centralized.
- **Internationalization** — server stores raw UTF-8; no server-side copy. Error codes map to client-localized strings.
- **Backward compatibility** — additive schema; new endpoints; no changes to existing tables beyond a new schema.

## 7. Acceptance Criteria

- **AC-1.** *Given* an authenticated user, *When* they POST a valid `{message}`, *Then* a `feedback.submissions` row exists with their `user_id`/`org_id`, `status='new'`, correct `source`, and the API returns `201` with `{id, created_at}`.
- **AC-2.** *Given* an empty or >5,000-char message, *When* POSTed, *Then* the API returns `400` and no row is written.
- **AC-3.** *Given* 11 submissions in 10 minutes by one user, *When* the 11th is POSTed, *Then* the API returns `429`.
- **AC-4.** *Given* a non-admin, *When* they call any `/api/v1/admin/feedback*`, *Then* the API returns `403`.
- **AC-5.** *Given* an admin with 3 submissions across 2 statuses, *When* they GET the list filtered by `status=new`, *Then* only matching rows return, newest first, paginated.
- **AC-6.** *Given* an admin, *When* they PATCH a row's status to `resolved`, *Then* `resolved_by`/`resolved_at` are set and the change is audited.
- **AC-7.** *Given* a user is erased via DSAR, *When* erasure runs, *Then* their feedback rows are deleted/anonymized and appear in the export bundle.

## 8. Data Model

New schema `feedback`, table `feedback.submissions`:

| Column | Type | Notes |
|---|---|---|
| `id` | `uuid` PK | `gen_random_uuid()` |
| `user_id` | `uuid` NULL | FK → users; nullable for future anonymous |
| `org_id` | `uuid` NULL | tenant scope; from session |
| `message` | `text` NOT NULL | trimmed, length-checked in code + `CHECK (char_length(message) BETWEEN 1 AND 5000)` |
| `category` | `text` NOT NULL DEFAULT `'other'` | `CHECK (category IN ('bug','idea','question','praise','other'))` |
| `source` | `text` NOT NULL | `CHECK (source IN ('web','ios','android'))` |
| `app_version` | `text` NULL | client build/version |
| `context` | `jsonb` NOT NULL DEFAULT `'{}'` | route/url, locale, viewport, user-agent |
| `status` | `text` NOT NULL DEFAULT `'new'` | `CHECK (status IN ('new','triaged','in_progress','resolved','wont_fix','archived'))` |
| `admin_note` | `text` NULL | internal triage note |
| `resolved_by` | `uuid` NULL | admin user id |
| `resolved_at` | `timestamptz` NULL | set on terminal status |
| `created_at` | `timestamptz` NOT NULL DEFAULT `now()` | |
| `updated_at` | `timestamptz` NOT NULL DEFAULT `now()` | trigger or code-managed |

- Indexes: `(org_id, status, created_at DESC)`, `(created_at DESC)`, `(user_id)`; optional GIN/trigram on `message` for free-text search (or Postgres FTS `tsvector` — §18).
- Migration files: `server/migrations/370_feedback_submissions.sql` + `370_feedback_submissions.down.sql` (next free number is 370; verify at implementation time).
- Backfill: none (new table).
- New packages: `server/internal/models/productfeedback` (structs + enums), `server/internal/repos/productfeedback` (insert, list-with-filters, get-by-id, update-status, delete-by-user).

## 9. API Surface

**Submit (any authenticated user)**

- `POST /api/v1/feedback`
  - Request: `{ "message": string, "category"?: "bug"|"idea"|"question"|"praise"|"other", "source": "web"|"ios"|"android", "app_version"?: string, "context"?: { "route"?: string, "locale"?: string, "viewport"?: string }, "idempotency_key"?: string }`
  - Response `201`: `{ "id": uuid, "created_at": string }`
  - Errors: `400` (validation), `401` (unauth), `429` (rate-limited).

**Admin (requires `global:app:rbac:manage`)**

- `GET /api/v1/admin/feedback?status=&category=&source=&q=&from=&to=&limit=&cursor=`
  - Response: `{ "items": FeedbackListItem[], "next_cursor"?: string, "total"?: number }` where `FeedbackListItem = { id, message_preview, category, source, status, submitter: { name, email }, created_at }`.
- `GET /api/v1/admin/feedback/{id}` → full `FeedbackDetail` (all columns + resolved submitter/resolver display info + parsed `context`).
- `PATCH /api/v1/admin/feedback/{id}` — body `{ "status"?, "admin_note"? }` → updated `FeedbackDetail`.

- Rate limits: submit per-user quota (§FR-7) via `internal/ratelimit`. Admin list default `limit=25`, max `100`.
- OpenAPI: add all four operations to the spec (`internal/openapi`), matching repo convention.

## 10. UI / UX

- None in this story (contracts only). FB1/FB3 build the submit UI; FB2 builds the admin UI. This story ships example requests/responses in the OpenAPI spec so client teams can integrate against a stable shape.

## 11. AI / ML Considerations

- Out of scope for MVP. Future: auto-categorize `category`, cluster duplicates, and summarize themes via the platform's existing AI provider layer with PII redaction and a cost budget. Note as §18 Open Question; schema (`category`, `context`) is forward-compatible.

## 12. Integration Points

- **Internal:** `server/internal/httpserver` (new `feedback_http.go` handlers + route registration alongside `admin.go`), `server/internal/repos/productfeedback`, `server/internal/models/productfeedback`, `server/internal/repos/rbac` (permission check), `server/internal/ratelimit`, `server/internal/telemetry`, `server/internal/openapi`.
- **DSAR/erasure:** hook into the existing GDPR/DSAR export + account-deletion paths (`server/internal/repos/gdpr` / `ferpa` / account deletion) to include/erase feedback rows.
- **Admin audit:** emit via the existing `admin_audit_log` mechanism on admin reads/mutations.
- **External:** none.

## 13. Dependencies & Sequencing

- Must ship **before** FB1, FB2, FB3.
- Shared infra: Postgres (new schema), Redis (rate-limit counters, if that's the ratelimit backend), telemetry pipeline.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Spam / abuse floods the table | M | M | Per-user rate limit, length cap, category enum, admin bulk-archive (FB2) |
| Stored XSS via message rendered in admin UI | M | H | Store as plain text; FB2 renders escaped, never `dangerouslySetInnerHTML` |
| PII in feedback violates retention/DSAR | M | H | Include in DSAR export; delete/anonymize on erasure; configurable retention window |
| `feedbackmedia` naming confusion | L | L | Use `productfeedback` package + `feedback` schema; document in README |
| Free-text search slow at scale | L | M | Start with trigram/FTS index; cap `q` scans; keyset pagination |

## 15. Rollout Plan

- Feature flag: `ff_feedback` (platform flag, default **ON**) gating the submit endpoint + clients; admin page independently visible to RBAC managers.
- Sequencing: migration (370) → repo/model → handlers + flag (default off in prod initially) → FB1 wired → flip flag ON after smoke test.
- Dogfood: enable for internal org first; watch `feedback_submitted_total` and error rate.
- GA criteria: submit success rate > 99%, admin list p95 < 500 ms, zero authz escapes in test.
- Rollback: flip `ff_feedback` off (hides clients + 404s submit); table/rows retained.

## 16. Test Plan

- **Unit** — validation (empty/oversized/enum coercion), identity derived from session not body, source/user-agent reconciliation, status-transition rules, `resolved_at` set on terminal status.
- **Integration (DB)** — insert → row shape; list filters (status/category/source/q/date); pagination; get-by-id; PATCH; delete-by-user (DSAR).
- **API** — `201/400/401/429` on submit; `403` for non-admin on all admin routes; `200` for admin.
- **Security** — authz matrix (non-admin blocked), rate-limit trip, injection/XSS payload stored inertly, no identity spoofing via body, `context` URL never fetched.
- **Privacy** — DSAR export includes rows; erasure removes/anonymizes them.
- **Performance** — seed 100k rows; assert list p95 and index usage (`EXPLAIN`).

## 17. Documentation & Training

- API reference: OpenAPI entries for all four operations.
- Admin runbook: "Triage the feedback queue" (statuses, notes) — expanded in FB2.
- Privacy runbook: note feedback in the DSAR/erasure data map (RoPA).
- Internal: schema note in `server/migrations/README.md` conventions.

## 18. Open Questions

1. **Retention window** — auto-purge resolved/archived feedback after N days, or keep indefinitely? (Default: keep; add retention job later.)
2. **Org scoping of admin view** — MVP shows all orgs to a `global:app:rbac:manage` holder. Do we need org-admin-scoped visibility, and a dedicated `feedback:manage` permission? (See FB2 §18.)
3. **Free-text search** — trigram index vs. Postgres FTS `tsvector`? Pick based on expected volume.
4. **Admin notification** — email/notify admins on new (or high-signal) feedback? Deferred SHOULD.
5. **Anonymous / marketing-site feedback** — `user_id` is nullable to allow it later; confirm we won't need an unauthenticated abuse story sooner.
6. **AI categorization/sentiment** — worth it once volume justifies; forward-compatible schema.

## 19. References

- Existing files this work touches: `server/internal/httpserver/admin.go` (RBAC guard + route pattern), `server/migrations/368_course_marketplace.sql` (migration style), `server/internal/repos/*` + `server/internal/models/*` (package conventions), `server/internal/telemetry` (observability), `server/internal/ratelimit`.
- Distinct from: `server/internal/repos/feedbackmedia` (grading annotation media — **not** product feedback).
- Related plans: [FB1](../../plan/feedback/FB1-web-share-feedback-button.md), [FB2](../../plan/feedback/FB2-web-feedback-admin.md), [FB3](../../plan/feedback/FB3-mobile-share-feedback.md).
