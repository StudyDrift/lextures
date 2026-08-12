# MC.2 — Authoring API, Revisions & Workflow States

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Plans.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.2 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING — content is created by `git commit`; there is no write API |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Server platform |
| **Depends on** | MC.1 |
| **Unblocks** | MC.6, MC.8, MC.9, MC.10, MC.11 |

---

## 1. Problem Statement

With the schema in place there is still no way for anything but psql to create or change an article.
The admin workspace (MC.9/MC.10), the importer (MC.6) and the publish pipeline (MC.8) all need one
authenticated, permission-checked, audited write surface with revision history and safe concurrency —
and they need it to be the *only* such surface, so that "who changed this published help article and
when" always has an answer. Building it once here prevents three clients from each inventing their
own half-correct transaction boundaries.

## 2. Goals

- Provide complete CRUD for articles, categories, authors, tags and redirects under
  `/api/v1/admin/marketing/*`, gated by the MC.1 permissions and `ff_marketing_content`.
- Model the editorial lifecycle as explicit, server-validated transitions rather than a free-text
  status field the client can set to anything.
- Guarantee attribution and recoverability: every write produces a revision; every revision can be
  read and restored.
- Make concurrent editing safe and comprehensible — a conflicting save is rejected with enough
  information for the UI to show a diff, never silently merged or clobbered.
- Emit an audit-log entry for every state-changing operation, reusing
  [`internal/service/adminaudit`](../../../server/internal/service/adminaudit).

## 3. Non-Goals

- No public/anonymous read path — that is [MC.3](MC.3-public-content-read-api.md).
- No markdown rendering, sanitization or quality scoring beyond calling the validator — that is
  [MC.4](MC.4-content-rendering-and-validation-service.md).
- No file/image upload endpoints — [MC.5](MC.5-marketing-media-library.md).
- No scheduled-publish worker or site rebuild trigger — [MC.8](MC.8-publish-pipeline-and-scheduling.md)
  (this plan only records `scheduled` state and emits the event).
- No UI.
- No bulk import endpoint; the importer runs as a command with direct repo access (MC.6).

## 4. Personas & User Stories

- **As a content expert**, I want to save a draft as often as I like without publishing it, so that
  work in progress is never public.
- **As a content expert**, I want to be told when someone else changed the article since I opened it,
  so that I do not overwrite their work.
- **As an editor**, I want to request changes with a note, so the author knows what to fix.
- **As a publisher**, I want to publish or schedule an article and later unpublish it, so that a
  wrong or outdated page can be pulled without a deploy.
- **As a compliance/admin reviewer**, I want every content change in the admin audit log, so that
  public claims made on our site are traceable to a person.
- **As an engineer**, I want the same API the UI uses to be scriptable, so migrations and fixes do not
  need database access.

## 5. Functional Requirements

- **FR-1.** All routes MUST live under `/api/v1/admin/marketing/` and MUST return `404` (not `403`)
  when `ff_marketing_content` is off, matching the marketplace precedent in
  `public_marketplace_http.go`.
- **FR-2.** Every route MUST require an authenticated session and the permission below; a holder of
  a lower permission MUST receive `403` with a problem body:

  | Operation | Required permission |
  |---|---|
  | list/read articles, revisions, categories, authors | `global:app:marketing-content:view` |
  | create/update draft, upload-attach, restore revision | `…:author` |
  | `submit_review`, `approve`, `request_changes` | `…:review` |
  | `publish`, `schedule`, `unpublish`, `archive` | `…:publish` |
  | categories, authors, tags, redirects write | `…:admin` |

- **FR-3.** `POST /articles` MUST create a `draft`, derive `path` from `kind`+`category`+`slug`,
  reject a duplicate live path with `409`, and return the full article with `revisionNo = 1`.
- **FR-4.** `PATCH /articles/{id}` MUST accept a partial body, MUST require `expectedRevisionNo`,
  and MUST return `409 CONFLICT` with `{ currentRevisionNo, updatedBy, updatedAt }` when stale.
- **FR-5.** `PATCH` MUST NOT accept `status`; status changes happen **only** through
  `POST /articles/{id}/transition`.
- **FR-6.** `POST /articles/{id}/transition` MUST accept
  `{ action, scheduledFor?, note?, expectedRevisionNo }` where `action ∈ {submit_review, approve,
  request_changes, publish, schedule, unpublish, archive, restore_draft}`, and MUST enforce this
  state machine:

  ```
  draft ──submit_review──▶ in_review ──approve──▶ draft*(approved)
    ▲                          │                    │
    │                    request_changes      publish│schedule
    └── restore_draft ── changes_requested           ▼
                                              published ⇄ scheduled
                                                    │
                                              unpublish │ archive
                                                    ▼
                                                 archived ──restore_draft──▶ draft
  ```

  Any action not legal from the current status MUST return `422` naming the current status and the
  legal actions.
- **FR-7.** `publish` and `schedule` MUST refuse when the article fails the MC.4 validator at
  publish severity (score below floor, unknown directive, missing required front matter, unresolved
  internal link), returning `422` with the findings array. `author`-level saves MUST NOT be blocked
  by validation — they store the findings instead.
- **FR-8.** `publish` MUST set `published_at = now()` (and `first_published_at` if null),
  `schedule` MUST require `scheduledFor` in the future, and `unpublish` MUST clear
  `published_at` while keeping `first_published_at`.
- **FR-9.** Every successful mutation MUST insert a `content_revisions` row with the acting user,
  `status_after`, an optional `change_note`, and a full metadata snapshot.
- **FR-10.** `GET /articles/{id}/revisions` MUST return paginated revision metadata (no bodies);
  `GET /articles/{id}/revisions/{no}` MUST return the full snapshot;
  `POST /articles/{id}/revisions/{no}/restore` MUST create a *new* revision whose content equals the
  old one (never rewriting history) and MUST leave `status` unchanged.
- **FR-11.** `DELETE /articles/{id}` MUST soft-delete. Deleting an article that has ever been
  published MUST require `…:publish` and MUST create a `content_redirects` row when `redirectTo` is
  supplied, or return `422` when the article is published and no `redirectTo` is given.
- **FR-12.** Changing the slug or category of a **published** article MUST create a `301` redirect
  from the old path in the same transaction (MC.1 AC-3), and MUST be reported in the response as
  `createdRedirect`.
- **FR-13.** The API MUST emit an `adminaudit` entry for create, update, transition, restore, delete
  and every taxonomy write, recording actor, article id, path, action and revision numbers.
- **FR-14.** `GET /articles` MUST support `kind`, `status`, `locale`, `category`, `author`, `q`
  (title/description/body FTS), `reviewDueBefore`, cursor pagination and `sort ∈ {updated, published,
  title}`, and MUST NOT return `bodyMd`.
- **FR-15.** `POST /articles/{id}/preview-token` MUST mint a short-lived (default 30 min) HMAC token
  scoped to one article id + revision, for use by MC.3's preview read path.
- **FR-16.** Write endpoints MUST be rate-limited per user (default 120 writes/min) using
  `internal/ratelimit`, and request bodies MUST be capped (default 1 MB) to bound `body_md`.
- **FR-17.** All routes MUST be registered in the route inventory (`make route-inventory-update`) and
  documented in OpenAPI (`make openapi-check`).
- **FR-18.** Handlers MUST NOT contain SQL; they call `internal/service/marketingcontent`, which
  calls `internal/repos/marketingcontent`.

## 6. Non-Functional Requirements

- **Performance** — `GET /articles` p95 < 150 ms at 5,000 rows; `PATCH` p95 < 200 ms including
  validation; validation of a 40 KB body < 50 ms (MC.4 budget).
- **Security** — Session auth + permission check on every route; no permission implies another
  (`publish` does not grant `admin`). Bodies are stored raw and never rendered by this service;
  sanitization happens at render time (MC.4) so a stored-XSS payload cannot escape through the API.
  Preview tokens are HMAC-SHA256 over `(articleID, revisionNo, exp)` with the existing JWT secret,
  single-purpose, non-renewable.
- **Privacy & Compliance** — Audit entries contain actor user IDs; retention follows the existing
  admin audit policy. No learner data. AI disclosure not engaged.
- **Accessibility** — N/A (API), but error payloads MUST be human-readable strings the UI can render
  verbatim into an alert region.
- **Scalability** — Stateless handlers; revisions grow unbounded, so list endpoints are cursor-paged
  and revision bodies are never returned in list form.
- **Reliability** — Article write + revision insert + redirect insert are one transaction. Transition
  is idempotent for repeated identical `publish` calls within the same revision (returns 200 with the
  existing state rather than double-publishing).
- **Observability** — Counters `marketing_content_writes_total{action}`,
  `marketing_content_conflicts_total`, `marketing_content_validation_blocks_total`; span per handler
  via `internal/telemetry`; log fields `article_id`, `kind`, `path`, `action`, `actor_id`,
  `revision_no`.
- **Maintainability** — New package `internal/service/marketingcontent`; handlers in
  `internal/httpserver/marketing_content_admin_http.go` split by resource if the file approaches the
  400-line budget.
- **Internationalization** — `locale` is accepted on create and immutable thereafter (a different
  locale is a different article joined by `translation_group_id`); error strings are English-only
  server-side and mapped to i18n keys by the client.
- **Backward compatibility** — Entirely new routes; nothing existing changes. Route inventory and
  characterization goldens are additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* `ff_marketing_content` is off, *when* any `/api/v1/admin/marketing/*` route is
  called by a Global Admin, *then* the response is `404`.
- **AC-2.** *Given* a user with only `…:view`, *when* they `POST /articles`, *then* the response is
  `403` and no row is created.
- **AC-3.** *Given* an existing published article at `/docs/courses/finding-your-course`, *when* a
  new article is created with the same kind/locale/slug, *then* the response is `409` with code
  `duplicate_slug`.
- **AC-4.** *Given* two clients loaded revision 4, *when* both `PATCH` with
  `expectedRevisionNo = 4`, *then* the first succeeds with revision 5 and the second receives `409`
  containing `currentRevisionNo: 5` and the winning editor's name.
- **AC-5.** *Given* an article in `draft`, *when* `transition{action:"publish"}` is called by a
  `…:publish` holder and the article validates, *then* status is `published`, `published_at` is set,
  a revision with `status_after='published'` exists, and an audit entry is written.
- **AC-6.** *Given* an article whose quality score is below the floor, *when* `publish` is called,
  *then* the response is `422` with a `findings` array and the status is unchanged.
- **AC-7.** *Given* an article in `published`, *when* `transition{action:"submit_review"}` is called,
  *then* the response is `422` listing the legal actions (`unpublish`, `archive`, `schedule`).
- **AC-8.** *Given* a published article, *when* its slug is changed, *then* the response includes
  `createdRedirect: {from, to, statusCode: 301}` and `content_redirects` contains that row.
- **AC-9.** *Given* revision 2 of an article now at revision 7, *when* restore is called for 2,
  *then* the article body equals revision 2's body and the article is at revision 8 with revision 7
  still readable.
- **AC-10.** *Given* a `scheduled` transition with `scheduledFor` in the past, *when* called, *then*
  the response is `422` (`scheduled_in_past`).
- **AC-11.** *Given* a preview token minted for article A revision 5, *when* it is presented for
  article B, *then* MC.3's preview path rejects it with `403`.
- **AC-12.** *Given* any mutation, *when* it succeeds, *then* `make route-inventory` and
  `make openapi-check` remain green in CI (routes and schemas registered).

## 8. Data Model

No new tables. This plan consumes MC.1's schema and adds:

- Service-level enum `Action` mirroring FR-6 and a transition table implemented as a Go map
  `map[Status][]Action` with a single source of truth used by both the handler and the client
  contract (surfaced in OpenAPI as an enum).
- One index added if profiling requires it: `idx_mc_articles_review_due
  ON marketing.content_articles (review_due_on) WHERE status = 'published'` — migration
  `477_marketing_content_review_due_idx.sql` (indicative number).
- No backfill.

## 9. API Surface

Base: `/api/v1/admin/marketing`. All JSON; problem bodies via `internal/apierr`.

| Verb | Path | Permission | Notes |
|---|---|---|---|
| GET | `/articles` | view | filters + cursor; no body |
| POST | `/articles` | author | creates draft |
| GET | `/articles/{id}` | view | full article incl. `bodyMd`, `qualityReport` |
| PATCH | `/articles/{id}` | author | partial; `expectedRevisionNo` required |
| DELETE | `/articles/{id}` | author (publish if ever published) | soft delete; optional `redirectTo` |
| POST | `/articles/{id}/transition` | per action | state machine |
| GET | `/articles/{id}/revisions` | view | paginated metadata |
| GET | `/articles/{id}/revisions/{no}` | view | full snapshot |
| POST | `/articles/{id}/revisions/{no}/restore` | author | new revision |
| POST | `/articles/{id}/preview-token` | view | `{token, expiresAt, url}` |
| POST | `/lint` | author | ad-hoc validation of a body (MC.4) |
| GET/POST | `/categories` · PATCH/DELETE `/categories/{id}` | view / admin | |
| GET/POST | `/authors` · PATCH `/authors/{slug}` | view / admin | retire = status change |
| GET/POST | `/tags` · DELETE `/tags/{id}` | view / admin | |
| GET/POST | `/redirects` · DELETE `/redirects/{id}` | view / admin | |

```ts
type ArticleWrite = {
  kind: 'blog' | 'doc'
  slug: string
  locale?: string                 // create only, default 'en'
  categorySlug?: string           // required for kind 'doc'
  title: string
  description: string
  bodyMd: string
  authorSlug: string
  reviewerSlug?: string
  primaryQuestion?: string
  cluster?: string; pillar?: string; briefRef?: string; verifiedAgainst?: string
  keywords?: string[]; relatedTo?: string[]; roles?: string[]; segments?: string[]
  citations?: string[]; tags?: string[]
  heroMediaId?: string
  contentUpdatedAt?: string       // ISO date; sitemap lastmod
  reviewDueOn?: string
  noindex?: boolean
  canonicalOverride?: string
  expectedRevisionNo?: number     // required on PATCH
}

type Article = ArticleWrite & {
  id: string; path: string; status: Status; revisionNo: number
  publishedAt: string | null; firstPublishedAt: string | null; scheduledFor: string | null
  qualityScore: number | null
  qualityReport: { score: number; findings: Finding[] } | null
  createdBy: Actor; updatedBy: Actor; createdAt: string; updatedAt: string
  createdRedirect?: { from: string; to: string; statusCode: 301 }
}

type Finding = { rule: string; severity: 'error' | 'warn' | 'info'; message: string; line?: number }
```

- **Rate limits:** 120 writes/min/user, 600 reads/min/user; 1 MB max body.
- **WebSocket:** none. (Concurrent-edit awareness is optimistic-token based; a presence channel is an
  explicit non-goal — see MC.10 §18.)
- **OpenAPI:** every route added to `server/internal/openapi`; `make openapi-check` must pass.

## 10. UI / UX

No UI in this plan. The API's UX obligations to MC.9/MC.10:

- Conflict responses carry the winning editor's display name and timestamp so the editor can show
  "Jordan saved a newer version 2 minutes ago" with **Reload** / **Copy my changes** actions.
- Validation responses carry per-finding `line` numbers so the editor can anchor inline markers.
- Transition responses always return the full article so the client never needs a follow-up GET.
- Error copy is stable and mapped to i18n keys `marketingContent.error.<code>` on the client.

## 11. AI / ML Considerations

Not AI-touching. `POST /lint` is deterministic rule evaluation (MC.4), not a model call. If AI
assistance is added to the editor later it MUST go through `internal/service/aigateway` and record
disclosure via `internal/aidisclosure`; this API does not open a path around either.

## 12. Integration Points

- **Internal modules:** `internal/httpserver/marketing_content_admin_http.go` (new),
  `internal/service/marketingcontent/` (new), `internal/repos/marketingcontent/` (MC.1),
  `internal/service/marketingcontent/validate` (MC.4), `internal/service/adminaudit`,
  `internal/ratelimit`, `internal/apierr`, `internal/openapi`, `internal/telemetry`.
- **Events:** publishes `marketingcontent.published` / `.unpublished` / `.scheduled` on the internal
  event path consumed by MC.8; no external webhook here.
- **External services:** none.

## 13. Dependencies & Sequencing

- Must ship after: MC.1 (schema, flag, permissions). MC.4 may land in parallel; until it does, the
  validator is a no-op stub returning an empty findings array and FR-7 is inert.
- Must ship before: MC.6 (importer reuses the service layer for validation parity), MC.8, MC.9,
  MC.10, MC.11.
- Shared infra: none beyond Postgres. No queue, no object storage.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| State machine drifts between server and client | M | M | Single Go map is the source of truth and is emitted into OpenAPI; client imports generated types (`npm run openapi:types`) |
| Optimistic concurrency frustrates authors who edit for an hour | M | M | Editor autosaves drafts every 30 s (MC.10), so tokens stay fresh; conflict payload enables a diff view rather than a dead end |
| Publish blocked by an over-strict validator | M | H | Severity split: only `error` findings block; the floor is configurable per kind and the block is overridable by `…:publish` with a recorded justification (MC.11 FR-9) |
| Revision table growth | L | M | Metadata-only list endpoints, cursor paging, and a retention job in MC.11 that prunes non-published revisions older than 18 months |
| Handler package exceeds structure budgets | M | L | Split by resource file from the start (`…_articles.go`, `…_taxonomy.go`) |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content` (MC.1), still OFF. Routes 404 in production.
- **Sequencing:** service + handlers → route inventory update → OpenAPI → characterization goldens →
  merge with flag off.
- **Dogfood:** internal staging with the flag on for the Docs/Content owner and two engineers; create
  and publish one throwaway article end-to-end via `curl`.
- **GA criteria:** all ACs green; audit entries verified; no `5xx` in staging over 200 synthetic
  writes.
- **Rollback:** flip the flag off (routes 404 immediately); code rollback is a normal revert since no
  schema changes are introduced here beyond one optional index.

## 16. Test Plan

- **Unit** — transition legality matrix (every status × every action); path derivation; conflict
  detection; permission matrix per route; preview-token mint/verify incl. expiry and article binding;
  request validation (slug charset, locale, size caps).
- **Integration (DB)** — create → patch → submit → approve → publish → unpublish → archive →
  restore_draft round trip with revision assertions; concurrent PATCH race using two transactions;
  slug-change redirect creation; soft-delete of a published article without `redirectTo` rejected.
- **End-to-end** — deferred to MC.9/MC.10 (UI). A `no-DB` handler test asserts the 404-when-flag-off
  behaviour, mirroring `public_marketplace_nodb_test.go`.
- **Security** — authz matrix test asserting each route rejects every insufficient permission;
  IDOR test (article id from another kind/locale still requires permission); oversized body rejected;
  rate limiter trips at the configured threshold; preview token replay after expiry rejected.
- **Accessibility** — N/A.
- **Performance / load** — `go test -bench` on validation path; k6 script: 50 rps mixed read/write for
  5 min, p95 < 200 ms, zero conflicts under single-writer load.
- **Manual exploratory** — QA checklist: publish/unpublish/republish, restore an old revision, change
  a category on a published doc, verify the audit log rows in Admin → Audit.

## 17. Documentation & Training

- API reference: OpenAPI descriptions per route; a short `docs/guides/marketing-content-api.md`
  covering the state machine diagram and the conflict contract.
- Admin docs: deferred to MC.15 (help article "Publishing marketing content").
- Internal runbook: "Unpublishing a page in a hurry" — `curl` recipe using a service token, since it
  must work when the SPA is broken.

## 18. Open Questions

1. Should `approve` move the article to a distinct `approved` status rather than back to `draft`
   with an approval marker? (Proposed: keep the marker; a fifth status buys little and complicates
   the matrix.)
2. Do we need an explicit `POST /articles/{id}/duplicate` for "start from this article"? (Likely
   yes for the help center; deferred to MC.10 unless authors ask sooner.)
3. Should bulk transitions (publish 10 selected drafts) be a single endpoint or N calls? (Proposed:
   N calls from the client with a progress UI; revisit if the editorial calendar needs atomicity.)
4. What retention applies to preview tokens in logs — do we redact the token query parameter in
   access logs? (Proposed: yes, add to the existing redaction list.)

## 19. References

- Files this work touches: `server/internal/httpserver/marketing_content_admin_http.go`,
  `server/internal/service/marketingcontent/*`, `server/internal/repos/marketingcontent/*`,
  `server/internal/openapi/*`, `scripts/allowlists/*` (if any layering exception is needed — it
  should not be).
- Precedents: `server/internal/httpserver/banner_http.go` (admin CRUD shape),
  `server/internal/httpserver/public_marketplace_http.go` (flag-off 404),
  `server/internal/service/adminaudit` (audit entries).
- Related plans: [MC.1](MC.1-content-data-model-and-migrations.md),
  [MC.4](MC.4-content-rendering-and-validation-service.md),
  [MC.8](MC.8-publish-pipeline-and-scheduling.md), [MC.10](MC.10-article-editor.md),
  [MC.11](MC.11-editorial-workflow-and-governance.md).
