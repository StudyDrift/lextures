# MKTC.2 — Creator Coupon Management API & Share Links

> Implementation plan. Source: [docs/plan/marketplace/README.md](../../plan/marketplace/README.md). Part of the MKTC Course Coupon Codes epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKTC.2 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | HS (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Commerce / Growth squad (backend) |
| **Depends on** | MKTC.1 |
| **Unblocks** | MKTC.3, MKTC.4, MKTC.7 |

---

## 1. Problem Statement

MKTC.1 gives the platform tables and a discount engine, but nothing can reach them. A course creator
needs to create, list, edit, pause and retire codes on **their own** course, see how many seats each
code has burned, and copy a shareable link that pre-applies the code. Today the only course-commerce
write path is `PUT /api/v1/courses/{course_code}/catalog-listing`
(`server/internal/httpserver/course_catalog_listing.go:64`), which sets a single price and is
authorized by `course:{code}:item:create`. Coupons need their own resource with the same authority
model, plus a redemption read endpoint so the creator can see *who* used a code.

## 2. Goals

- A complete authenticated CRUD surface for course coupons, authorized by the same capability that
  already gates marketplace listing settings.
- Server-generated **share links** so every client renders an identical, correct URL rather than
  string-building it three times (web, iOS, Android).
- A redemption listing per coupon (paginated) that shows learner, amount charged, discount, and
  state.
- Validation that fails loudly and specifically: bad code shape, overlapping duplicate, percent out
  of range, fixed amount in the wrong currency, end before start, cap below current usage.
- Introduce the `ffCourseCoupons` platform flag and gate every route behind it.

## 3. Non-Goals

- The learner-facing apply/preview/checkout endpoints (MKTC.3).
- Any UI (MKTC.4).
- Bulk import/export of codes, auto-generated code batches, or unique one-time codes per recipient.
- Platform-admin cross-course coupon administration (a platform admin acting on a course goes
  through the same course-scoped routes via their existing elevated grants).
- Coupon analytics dashboards beyond raw counts (MKTC.7).

## 4. Personas & User Stories

- **As a course creator**, I want to add a code with an amount, a window and a seat cap so that I
  can run a launch promotion without asking support.
- **As a course creator**, I want a one-click share link containing the code so that I can paste it
  into a newsletter and the recipient never has to type anything.
- **As a co-teacher with edit rights**, I want the same access as the owner so that coupon
  management is not bottlenecked on one person.
- **As a TA or student**, I want to be unable to see or create coupons so that codes are not leaked
  from inside the course.
- **As a course creator**, I want to pause a code instantly when it leaks, so that damage stops
  without me deleting my reporting history.
- **As a platform admin**, I want coupon writes to be audit-logged so that a disputed discount can
  be traced.

## 5. Functional Requirements

- **FR-1.** The system MUST expose `GET /api/v1/courses/{course_code}/coupons` returning the
  course's coupons with live seat counts, sorted by `created_at DESC`, with
  `?includeArchived=true` opting archived rows in (default: excluded).
- **FR-2.** The system MUST expose `POST /api/v1/courses/{course_code}/coupons` creating one coupon
  and returning `201` with the created row plus its `shareUrl`.
- **FR-3.** The system MUST expose `PATCH /api/v1/courses/{course_code}/coupons/{coupon_id}`
  supporting partial updates of `note`, `startsAt`, `endsAt`, `maxRedemptions`,
  `maxRedemptionsPerUser`, and `status`.
- **FR-4.** The system MUST NOT permit changing `code`, `discountType`, `percentOff`,
  `amountOffCents`, or `currency` after creation. Changing the value of a circulating code is a
  support incident waiting to happen; the creator archives and re-creates instead. A request
  attempting it returns `422` with `code: UNPROCESSABLE_ENTITY` and a field-specific message.
- **FR-5.** The system MUST expose `DELETE /api/v1/courses/{course_code}/coupons/{coupon_id}`
  performing a **soft delete** (`status='archived'`). Rows with redemptions MUST never be hard
  deleted, so financial history survives.
- **FR-6.** The system MUST expose
  `GET /api/v1/courses/{course_code}/coupons/{coupon_id}/redemptions` returning a cursor-paginated
  list (`limit` default 25, max 100) of `{userId, userName, userEmail, status, listPriceCents,
  discountCents, chargedCents, currency, reservedAt, redeemedAt}`.
- **FR-7.** Every route MUST require an authenticated session and the capability
  `course:{course_code}:item:create`, checked with
  `courseroles.UserHasPermission` exactly as `handlePutCourseCatalogListing` does. Absent
  capability → `403 FORBIDDEN`. Unknown course → `404`.
- **FR-8.** Every route MUST return `404 NOT_FOUND` when `ffCourseCoupons` is off, mirroring
  `courseMarketplaceOff`. A new `couponsFeatureOff(w)` helper MUST require **both**
  `FFCourseMarketplace` and `FFCourseCoupons`.
- **FR-9.** On create, the server MUST normalize the submitted code with `coupons.NormalizeCode`
  and validate it with `coupons.ValidateCode`. Rejections return `400 INVALID_INPUT` naming the
  rule violated (length, allowed characters, leading separator).
- **FR-10.** On create, a code that collides with a non-archived coupon on the same course MUST
  return `409 CONFLICT` with a message naming the existing code — never a 500 from the unique
  index.
- **FR-11.** The server MUST reject a `fixed` coupon whose `currency` differs from the course's
  `price_currency` with `422`, and MUST reject any coupon on a course whose `price_cents = 0` with
  `422` ("This course is free; coupons apply to paid courses only").
- **FR-12.** The server MUST reject `maxRedemptions` lower than the coupon's current consumed seat
  count with `422`, naming the current count. Raising the cap MUST always be allowed.
- **FR-13.** The server MUST warn — not reject — when a percent coupon would clamp the course's
  current price to free (MKTC.1 FR-12) by returning `warnings: ["clamps_to_free"]` on the create
  and update responses, so MKTC.4 can surface it.
- **FR-14.** The server MUST compute `shareUrl` as
  `{PublicWebOrigin}/marketplace/{slug}?coupon={CODE}` where `slug` falls back to `courseCode` when
  the listing has no slug, matching `marketplaceCheckoutHint`'s fallback in
  `marketplace_purchase_http.go:244`. When the course is also `is_public`, the response MUST
  additionally carry `publicShareUrl` = `{MarketingSiteOrigin}/courses/{slug}?coupon={CODE}`;
  otherwise `publicShareUrl` is null.
- **FR-15.** All timestamps in requests and responses MUST be RFC 3339 UTC. The API MUST reject a
  `startsAt`/`endsAt` pair where `endsAt <= startsAt` with `422`.
- **FR-16.** Coupon writes (create, update, status change, archive) MUST emit an audit event through
  the existing platform audit path used by other course-settings writes, carrying actor, course,
  coupon id and the changed fields — never a learner identity.
- **FR-17.** Write routes MUST be rate limited per user (reuse `checkBillingCheckoutRateLimit`'s
  bucket shape at a coupon-specific limit of 30/min) to bound accidental scripting.
- **FR-18.** All routes MUST be documented in `server/internal/openapi/openapi.json` and pass
  `make openapi-check`; the route inventory (`make route-inventory-update`) MUST be refreshed.
- **FR-19.** The platform settings surface MUST expose `ffCourseCoupons` as an admin-togglable flag
  in `GET /api/v1/platform/features` and the Settings → Global platform panel, defaulting **OFF**
  until MKTC.7 flips the default.

## 6. Non-Functional Requirements

- **Performance** — List p95 < 120 ms for 100 coupons including seat counts (single grouped count
  query, FR from MKTC.1). Create/update p95 < 100 ms. Redemption list p95 < 150 ms at limit 100.
- **Security** — Authorization is per-course capability, never "is staff anywhere". IDOR is
  prevented by always scoping the coupon lookup by `course_id` resolved from `{course_code}`, so a
  coupon id from another course 404s. Codes are returned only to authorized staff on these routes.
  Audit trail per FR-16. No secret material involved.
- **Privacy & Compliance** — The redemption list exposes learner name and email to course staff who
  already see the roster, so no new disclosure boundary is crossed; it MUST NOT expose payment
  instrument data or Stripe customer ids. FERPA: purchase-linked identity is shared with the
  instructor of that course only. GDPR: the endpoint is read-only and covered by the existing DSAR
  export.
- **Accessibility** — No UI in this story; error messages are written as complete sentences so
  MKTC.4 can surface them verbatim in an `role="alert"` region if a translation is missing.
- **Scalability** — Coupons per course are expected in the tens; pagination exists on redemptions
  only. Queries are index-covered.
- **Reliability** — Create is not idempotent by design (a second identical create 409s on the
  unique code), which is the desired behaviour for a double-submitted form. Update and archive are
  naturally idempotent.
- **Observability** — `coupon_admin_request_total{route,result}`,
  `coupon_created_total{discount_type}`, `coupon_status_changed_total{to}`. Logs carry
  `course_code`, `coupon_id`, actor id.
- **Maintainability** — New file `server/internal/httpserver/course_coupons_http.go` (routes +
  handlers) plus `course_coupons_json.go` if the DTO mapping pushes the file past 600 LOC. Routes
  registered from `courses_routes.go` next to the catalog-listing pair. Handlers contain no SQL
  (layering rule §2.1).
- **Internationalization** — Server messages are English fallbacks; clients render i18n keys keyed
  off the machine `code` field. Dates are UTC; MKTC.4 owns local-time conversion.
- **Backward compatibility** — Purely additive routes behind a default-off flag. No existing
  response shape changes.

## 7. Acceptance Criteria

- **AC-1.** *Given* I am the course owner of a paid marketplace course, *When* I `POST` a coupon
  `{code:"launch25", discountType:"percent", percentOff:25}`, *Then* I get `201` with
  `code:"LAUNCH25"` and `shareUrl` ending `/marketplace/{slug}?coupon=LAUNCH25`.
- **AC-2.** *Given* `LAUNCH25` already exists and is active, *When* I `POST` `LAUNCH25` again,
  *Then* I get `409 CONFLICT` and exactly one coupon exists.
- **AC-3.** *Given* `LAUNCH25` was archived, *When* I `POST` `LAUNCH25` again, *Then* I get `201`
  and the archived row is untouched.
- **AC-4.** *Given* I am a student enrolled in the course, *When* I `GET` the coupons route, *Then*
  I get `403` and the response body contains no coupon data.
- **AC-5.** *Given* `ffCourseCoupons` is off, *When* any coupon route is called by an authorized
  creator, *Then* the response is `404`.
- **AC-6.** *Given* a coupon with 12 consumed seats, *When* I `PATCH` `maxRedemptions: 10`, *Then*
  I get `422` naming 12; *When* I `PATCH` `maxRedemptions: 50`, *Then* I get `200`.
- **AC-7.** *Given* an existing coupon, *When* I `PATCH` `percentOff: 90`, *Then* I get `422` and
  the stored discount is unchanged.
- **AC-8.** *Given* a course priced in `eur`, *When* I `POST` a `fixed` coupon with
  `currency: "usd"`, *Then* I get `422`.
- **AC-9.** *Given* a free course (`price_cents = 0`), *When* I `POST` any coupon, *Then* I get
  `422`.
- **AC-10.** *Given* a coupon id belonging to course B, *When* I `PATCH` it via course A's path,
  *Then* I get `404`.
- **AC-11.** *Given* a coupon with three redemptions, *When* I `GET` its redemptions with
  `limit=2`, *Then* I get two rows and a `nextCursor` that returns the third and an empty cursor.
- **AC-12.** *Given* I `DELETE` a coupon that has redemptions, *When* the request completes, *Then*
  it returns `200`, `status` is `archived`, and the redemption rows still exist.
- **AC-13.** *Given* the course is `is_public`, *When* I create a coupon, *Then* `publicShareUrl`
  points at the marketing-site course URL with the same `?coupon=` parameter; *given* it is not
  public, *Then* `publicShareUrl` is `null`.
- **AC-14.** *Given* the OpenAPI document, *When* `make openapi-check` runs, *Then* all five coupon
  routes are present with request/response schemas and the check passes.

## 8. Data Model

No schema changes beyond MKTC.1, with one exception:

```sql
-- MKTC.2 — platform flag for the coupon feature.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_course_coupons BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_course_coupons IS
    'Enables creator-managed course coupon codes (plan MKTC). Default OFF until GA (plan MKTC.7).';
```

Wired through `repos/platformconfig`: `Row.FFCourseCoupons *bool`, `Merge` →
`out.FFCourseCoupons = mergeBool(db.FFCourseCoupons, false)` (note: **false** default, unlike
`FFCourseMarketplace`), `patch.go` → `addBool("ff_course_coupons", w.FFCourseCoupons)`, and
`config.Config.FFCourseCoupons bool`.

Reads/writes use only the MKTC.1 repo functions. No new indexes.

## 9. API Surface

All routes are authenticated, `ffCourseMarketplace && ffCourseCoupons` gated, and require
`course:{course_code}:item:create`.

| Verb | Path | Success | Notes |
|---|---|---|---|
| GET | `/api/v1/courses/{course_code}/coupons` | 200 | `?includeArchived=true` |
| POST | `/api/v1/courses/{course_code}/coupons` | 201 | 409 on duplicate code |
| PATCH | `/api/v1/courses/{course_code}/coupons/{coupon_id}` | 200 | immutable fields → 422 |
| DELETE | `/api/v1/courses/{course_code}/coupons/{coupon_id}` | 200 | soft delete |
| GET | `/api/v1/courses/{course_code}/coupons/{coupon_id}/redemptions` | 200 | cursor paginated |

```ts
type CourseCoupon = {
  id: string
  courseId: string
  code: string                       // always A–Z0–9_-
  discountType: 'percent' | 'fixed'
  percentOff: number | null          // 0 < n <= 100
  amountOffCents: number | null
  currency: string | null            // set for fixed
  startsAt: string | null            // RFC3339 UTC
  endsAt: string | null
  maxRedemptions: number | null      // null = unlimited
  maxRedemptionsPerUser: number      // >= 1
  seats: { consumed: number; reserved: number; redeemed: number; remaining: number | null }
  status: 'active' | 'disabled' | 'archived'
  note: string | null
  shareUrl: string
  publicShareUrl: string | null
  createdBy: string | null
  createdAt: string
  updatedAt: string
  warnings?: Array<'clamps_to_free'>
}

// POST body
type CreateCouponBody = {
  code: string
  discountType: 'percent' | 'fixed'
  percentOff?: number
  amountOffCents?: number
  currency?: string
  startsAt?: string | null
  endsAt?: string | null
  maxRedemptions?: number | null
  maxRedemptionsPerUser?: number
  note?: string | null
}

// PATCH body — all optional; any immutable key present → 422
type UpdateCouponBody = Pick<CreateCouponBody,
  'startsAt' | 'endsAt' | 'maxRedemptions' | 'maxRedemptionsPerUser' | 'note'> & {
  status?: 'active' | 'disabled'
}

type CouponRedemptionRow = {
  id: string
  userId: string
  userName: string | null
  userEmail: string | null
  status: 'reserved' | 'redeemed' | 'released'
  listPriceCents: number
  discountCents: number
  chargedCents: number
  currency: string
  reservedAt: string
  redeemedAt: string | null
}
```

Error bodies use the standard `apierr` envelope: `INVALID_INPUT` (400), `FORBIDDEN` (403),
`NOT_FOUND` (404), `CONFLICT` (409), `UNPROCESSABLE_ENTITY` (422), `RATE_LIMITED` (429).

No WebSocket events. Rate limit: 30 writes/min/user; reads share the standard API limiter.
OpenAPI: all five operations documented with schemas and examples, added to
`server/internal/openapi/openapi.json`; regenerate web types with `npm run openapi:types:file`.

## 10. UI / UX

No UI in this story. Two contracts are fixed here for MKTC.4:

- `shareUrl` / `publicShareUrl` are **server-rendered**; the client copies them verbatim and must
  not build coupon URLs itself.
- `seats.remaining` is `null` for uncapped coupons so the UI can render "Unlimited" without
  inventing a sentinel.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **External** — none.
- **Internal** —
  `server/internal/httpserver/course_coupons_http.go` (new),
  `server/internal/httpserver/courses_routes.go` (register, next to the catalog-listing routes at
  ~L191), `server/internal/httpserver/course_marketplace.go` (add `couponsFeatureOff`),
  `server/internal/repos/billing/coupons.go` (MKTC.1), `server/internal/courseroles` (permission
  check), `server/internal/repos/platformconfig/{platformconfig,features,patch}.go` (new flag),
  `server/internal/config/config.go` (`FFCourseCoupons`),
  `server/internal/openapi/openapi.json`, `clients/web/src/components/settings/platform-feature-definitions.ts`
  (admin toggle), `docs/route-inventory` golden.
- **Events** — audit event `course.coupon.created|updated|archived` on the existing audit channel.

## 13. Dependencies & Sequencing

- **Must ship after** — MKTC.1.
- **Must ship before** — MKTC.4 (the UI consumes this surface); MKTC.3 depends on MKTC.1 only and
  can be developed in parallel, but MKTC.2 must land first because it introduces the shared
  `ffCourseCoupons` flag and `couponsFeatureOff` helper.
- **Shared infra** — none beyond Postgres and the existing audit channel.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| IDOR — coupon id from another course | M | H | Always resolve `course_id` from `{course_code}` and scope every query by it; AC-10 test |
| Creator edits a circulating code's value, breaking promises made in print | M | M | Value fields immutable after create (FR-4); archive-and-recreate is the documented path |
| Duplicate code returns a 500 from the unique index | M | L | Explicit pre-check plus `pgerrcode.UniqueViolation` mapping to 409 |
| Flag defaults on by copy-paste from `FFCourseMarketplace` | M | H | Explicit `mergeBool(db.FFCourseCoupons, false)` with a unit test asserting default OFF |
| Redemption list leaks learner email to a TA | L | M | Capability is `item:create` (teacher/designer/owner), which TAs do not hold; authz matrix test |
| `httpserver` package file-count budget | L | L | One new file; package is already allowlisted and shrink-only under TD.6 |

## 15. Rollout Plan

- **Feature flag** — `ffCourseCoupons`, default **OFF**, admin-togglable in Settings → Global
  platform. All five routes 404 while off.
- **Sequencing** — migration (flag column) → config/platformconfig wiring → routes → OpenAPI +
  route inventory → merge. No data migration.
- **Dogfood** — enable the flag on staging for the internal demo course; create a percent coupon, a
  fixed coupon, a capped coupon and an expired coupon; verify the list, the counts and both share
  URLs.
- **GA criteria** — authz matrix green; OpenAPI check green; `make route-inventory` clean; no route
  reachable with the flag off.
- **Rollback** — turn the flag off (instant, no deploy). Code rollback is a revert; the flag column
  and MKTC.1 tables stay, inert.

## 16. Test Plan

- **Unit** (`*_nodb_test.go`, mirroring `marketplace_courses_nodb_test.go`) — body decoding and
  validation for every 400/422 branch; immutable-field rejection; share-URL construction including
  the slug fallback and URL-encoding of the code; flag-off 404; DTO mapping.
- **Integration** (DB) — full CRUD lifecycle; duplicate → 409; archive then re-create; cap lowered
  below usage → 422; redemptions pagination; cascade on course delete; audit event emitted.
- **End-to-end** — deferred to MKTC.4 (`e2e/tests/course-coupons.spec.ts` is created there); this
  story adds an API-level e2e smoke in the existing `course-marketplace-listing.spec.ts` harness
  that creates and lists a coupon over HTTP.
- **Security** — authz matrix across owner, teacher, designer, TA, student, observer, parent, org
  admin, platform admin, anonymous, and a staff member of a *different* course; IDOR probes with
  foreign coupon ids; rate-limit trip test; verify no SQL error text reaches the client.
- **Accessibility** — n/a (no UI).
- **Performance / load** — list 100 coupons with counts, assert single count query and p95 < 120 ms.
- **Manual exploratory** — create a coupon while the course price is being edited concurrently;
  archive a coupon mid-checkout (behaviour is defined in MKTC.3: an existing reservation still
  honours the price).

## 17. Documentation & Training

- **End-user docs** — none (learners never see these routes).
- **Admin / instructor docs** — draft the "Create a coupon code" help-centre page skeleton under
  `docs/help/`, completed with screenshots in MKTC.4.
- **API reference** — the five operations in `openapi.json`; add a changelog entry
  `docs/api-changelog-course-coupons.md` following the existing
  `api-changelog-course-checklist.md` format.
- **Internal runbook** — extend `docs/runbooks/coupons.md` with "how to archive a leaked code" and
  "how to check who can manage coupons on a course".

## 18. Open Questions

1. Should a platform admin be able to manage coupons on *any* course from an admin surface, or only
   through their existing course grants? (Proposed: existing grants only; revisit if support asks.)
2. Should `note` be visible in any learner-facing response? (Proposed: never — it is an internal
   label; MKTC.3 must not echo it.)
3. Do we need a "generate a random code" server endpoint, or is client-side generation with server
   validation enough? (Proposed: client-side in MKTC.4, since the server owns validation anyway.)
4. Should archiving a coupon release outstanding reservations? (Proposed: no — a learner mid-payment
   keeps the price they were quoted; MKTC.3 AC-9 pins this.)
5. Is 30 writes/min the right rate limit, or should coupon writes share the billing bucket?

## 19. References

- Existing files: `server/internal/httpserver/course_catalog_listing.go` (authz + validation
  pattern), `server/internal/httpserver/courses_routes.go:191` (registration site),
  `server/internal/httpserver/course_marketplace.go` (`courseMarketplaceOff` pattern),
  `server/internal/httpserver/billing_http.go:23-48` (rate-limit bucket),
  `server/internal/repos/platformconfig/features.go:158` (flag merge precedent),
  `server/internal/openapi/openapi.json`.
- Related plans: [MKTC.1](../../completed/marketplace/MKTC.1-coupon-data-model-and-discount-engine.md),
  [MKTC.3](MKTC.3-coupon-aware-checkout-and-redemption.md),
  [MKTC.4](MKTC.4-web-creator-coupon-manager.md),
  [MKT2](../../completed/marketplace/MKT2-course-marketplace-listing-settings.md),
  [TD.3 OpenAPI contract](../../plan/tech_debt/).
