# MKT1 — Marketplace Platform Foundation & Feature Flag

> Implementation plan. Source: [docs/plan/marketplace/README.md](../../plan/marketplace/README.md). Part of the Marketplace epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKT1 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE (continuing-ed) · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Commerce / Growth squad (backend platform) |
| **Depends on** | 15.1 public catalog, 15.3 Stripe billing |
| **Unblocks** | MKT2, MKT3, MKT4, MKT5, MKT6 |

---

## 1. Problem Statement

Lextures has all the commerce primitives for a course marketplace — course pricing (`course.courses.price_cents`), entitlements (`billing.user_entitlements`), a public catalog (15.1), and Stripe checkout (15.3) — but no first-class **in-app storefront** that ties them together, and no way for an operator to turn "the marketplace" on or off as a unit. The name `FFMarketplace` is already taken by the unrelated plugin/OAuth-app marketplace (16.9), so the course marketplace cannot simply reuse it. This story lays the platform foundation: a new toggleable feature flag (on by default), the data model for opting a course into the marketplace, and generalized entitlement plumbing that supports **free** course claims — everything the learner- and instructor-facing stories build on.

## 2. Goals

- Introduce a dedicated, platform-wide **`FFCourseMarketplace`** flag that gates every marketplace surface, **defaulting to ON**, toggleable by admins in Settings → Global platform.
- Add a course-level **`marketplace_listed`** flag (distinct from `is_public`) plus reuse of `price_cents`/`price_currency` as the course fee (default `0` = Free).
- Generalize `billing.user_entitlements` so a **Free** course grant is a first-class, idempotent entitlement (not dependent on a Stripe event id).
- Expose the flag through the existing config → platform-features pipeline to web and mobile clients.
- Provide repo/service helpers (`IsMarketplaceListed`, `MarketplaceAccess`) the other stories consume, with unit coverage.

## 3. Non-Goals

- The storefront UI, browse, or course detail pages (MKT3).
- Course-settings UI to toggle listing / set fee (MKT2).
- The purchase/checkout flow itself (MKT4).
- The purchased indicator on the Courses page (MKT5).
- Any mobile UI (MKT6).
- Payouts / revenue share (already covered by 15.8) and tax (15.13) — the marketplace reuses them unchanged.

## 4. Personas & User Stories

- **As a platform admin**, I want to turn the marketplace on or off for my whole tenant so that I can control whether course selling is available.
- **As an instructor/creator**, I want a course to be excludable from the marketplace independently of the public SEO catalog so that I can sell in-app without exposing an SEO landing page (and vice-versa).
- **As a self-learner**, I want free courses to be "claimable" so that claiming is a real, recorded acquisition just like a paid purchase.
- **As an engineer (MKT2–MKT6)**, I want one authoritative helper that answers "is this course sellable in the marketplace and does this user already have it?" so that every surface behaves consistently.

## 5. Functional Requirements

- **FR-1.** The system MUST expose a new boolean platform flag `FFCourseMarketplace`, persisted in `settings.platform_app_settings.ff_course_marketplace`, defaulting to **`true`** when the column is unset.
- **FR-2.** Every marketplace HTTP endpoint (defined in MKT2–MKT5) MUST return `404 Not Found` (`CodeNotFound`) with message "Marketplace is not enabled." when `FFCourseMarketplace` is false, mirroring `publicCatalogOff`.
- **FR-3.** The system MUST add `course.courses.marketplace_listed BOOLEAN NOT NULL DEFAULT FALSE` and `marketplace_listed_at TIMESTAMPTZ NULL`, set to `NOW()` when a course is listed and `NULL` when unlisted.
- **FR-4.** The course fee MUST reuse `price_cents` (INT, ≥0) and `price_currency` (ISO-4217, default `usd`). `price_cents = 0` MUST be interpreted everywhere as **Free**.
- **FR-5.** The system MUST allow a `course_purchase` entitlement to be created **without** a Stripe event id (for Free claims), while remaining idempotent per `(user_id, course_id)`.
- **FR-6.** The system MUST record the acquisition source on each `course_purchase` entitlement via a new `acquisition_source TEXT` column with values `stripe`, `free`, or `comp` (complimentary/admin grant), defaulting to `stripe` for existing rows.
- **FR-7.** The system MUST provide `billing.MarketplaceAccess(ctx, pool, userID, courseID) (owned bool, err error)` that returns true when an active `course_purchase` entitlement or subscription grants access — reusing `HasCourseAccess` semantics.
- **FR-8.** The flag MUST be emitted to clients as `ffCourseMarketplace` in `GET /api/v1/platform/features` and settable via the existing Settings → Global platform patch endpoint.
- **FR-9.** A course MUST NOT be marketplace-listable while in a `draft`/unpublished workflow state (enforced in MKT2's write path; the repo helper MUST expose publish state for that check).
- **FR-10.** Disabling the flag MUST NOT delete data: existing `marketplace_listed` rows and entitlements MUST be preserved and re-appear when the flag is re-enabled (soft off-switch).

## 6. Non-Functional Requirements

- **Performance** — The flag read is served from the already-cached `effectiveConfig()`; no new per-request DB call. `MarketplaceAccess` MUST be a single indexed query (p95 < 10 ms).
- **Security** — Only users holding the global platform-settings permission may toggle the flag (reuse existing Settings → Global platform authz). The course-listing column is written only via MKT2's permission-checked path (`course:{code}:item:create`).
- **Privacy & Compliance** — Entitlements are financial records; the `acquisition_source` and amount fields are covered by existing 15.13 tax/retention handling. Free claims store `amount_paid_cents = 0` and no PII beyond `user_id`.
- **Accessibility** — N/A (no UI in this story); the admin toggle inherits Settings → Global platform's existing WCAG-conformant control.
- **Scalability** — `marketplace_listed` is a low-cardinality boolean; a partial index supports storefront queries in MKT3.
- **Reliability** — Entitlement creation MUST be idempotent under retries and concurrent requests (unique constraint + `ON CONFLICT`).
- **Observability** — Emit a counter `marketplace_flag_state{enabled}` on config load and log flag flips through the existing admin-audit path.
- **Maintainability** — Follow the established flag pattern: DB column → `applyPlatformBools` → `config.Config` field → `platform_features.go` JSON → client contexts.
- **Internationalization** — Admin toggle label/help externalised via existing platform-settings i18n.
- **Backward compatibility** — New columns are additive with defaults; the down migration drops them. `acquisition_source` defaults preserve existing Stripe entitlements.

## 7. Acceptance Criteria

- **AC-1.** *Given* a fresh tenant with no `platform_app_settings` row, *When* the config loads, *Then* `FFCourseMarketplace` resolves to `true`.
- **AC-2.** *Given* the flag is toggled off by an admin, *When* any marketplace endpoint is called, *Then* it returns `404` and no marketplace nav item is emitted in `/platform/features` (`ffCourseMarketplace = false`).
- **AC-3.** *Given* a course with `price_cents = 0`, *When* a Free claim entitlement is created twice for the same user, *Then* exactly one row exists (idempotent) and `acquisition_source = 'free'`.
- **AC-4.** *Given* a user with an active `course_purchase` entitlement, *When* `MarketplaceAccess` is called for that course, *Then* it returns `true`; *When* called for an unrelated paid course, *Then* `false`.
- **AC-5.** *Given* the flag is disabled and re-enabled, *When* the storefront reloads, *Then* previously listed courses and entitlements are intact.
- **AC-6.** *Given* the down migration runs, *When* the schema is inspected, *Then* `marketplace_listed`, `marketplace_listed_at`, `acquisition_source`, and `ff_course_marketplace` are removed cleanly.

## 8. Data Model

New migration `server/migrations/NNN_course_marketplace.sql` (next free number; add matching `.down.sql`):

```sql
-- Course marketplace foundation (plan MKT1).
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS marketplace_listed    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS marketplace_listed_at TIMESTAMPTZ;

COMMENT ON COLUMN course.courses.marketplace_listed IS
    'When true, the course is offered in the in-app course marketplace (plan MKT1). Independent of is_public (SEO catalog).';

-- Storefront browse index: only listed rows.
CREATE INDEX IF NOT EXISTS idx_courses_marketplace
    ON course.courses (marketplace_listed, catalog_category, price_cents)
    WHERE marketplace_listed = TRUE;

-- Generalize entitlements for free claims (plan MKT1).
ALTER TABLE billing.user_entitlements
    ADD COLUMN IF NOT EXISTS acquisition_source TEXT NOT NULL DEFAULT 'stripe'
        CHECK (acquisition_source IN ('stripe', 'free', 'comp')),
    ALTER COLUMN stripe_event_id DROP NOT NULL;   -- free claims have no Stripe event

-- One active course_purchase per (user, course) — supports idempotent free + paid grants.
CREATE UNIQUE INDEX IF NOT EXISTS uq_entitlement_course_per_user
    ON billing.user_entitlements (user_id, course_id)
    WHERE entitlement_type = 'course_purchase' AND status = 'active';

-- Platform flag column (default handled in code = ON).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_course_marketplace BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_course_marketplace IS
    'Enables the in-app course marketplace/storefront (plan MKT1). Default ON.';
```

- **Backfill** — none required; existing courses default to unlisted, existing entitlements to `acquisition_source='stripe'`. `price_cents` already exists (default 0).
- **Constraints** — the partial unique index guarantees FR-5 idempotency; keep the existing `stripe_event_id` unique index for Stripe idempotency.
- **Naming** — follows `server/migrations/NNN_*.sql` per `docs/plan/README.md` / `server/migrations/README.md`.

## 9. API Surface

No new learner-facing routes in this story. Changes are:

- **Config/flag plumbing** (no new HTTP surface):
  - `settings.platform_app_settings.ff_course_marketplace` → `platformconfig.Row` field → `applyPlatformBools` sets `out.FFCourseMarketplace = mergeBool(db.FFCourseMarketplace, true)` (note **default `true`**).
  - `config.Config.FFCourseMarketplace bool` (documented as Settings-managed).
  - `httpserver/platform_features.go`: add `FFCourseMarketplace bool json:"ffCourseMarketplace"` to the features payload.
- **Settings → Global platform** (existing endpoint, `httpserver/settings_platform.go` + `repos/platformconfig/patch.go`): add `ffCourseMarketplace` to the read/patch allow-list so admins can flip it. Reuse existing auth scope; no new route.
- **Shared helper** in `httpserver`: `func (d Deps) courseMarketplaceOff(w http.ResponseWriter) bool` mirroring `publicCatalogOff`, consumed by MKT2–MKT5.
- **OpenAPI** — regenerate the `/platform/features` schema and Settings platform schema to include the new field (`server/internal/openapi`).

## 10. UI / UX

Only the admin toggle in **Settings → Global platform → Feature flags**:

- New toggle "Course marketplace" with help text: "Let learners discover and enroll in courses through an in-app storefront. Instructors opt individual courses in from course settings." Default **on**.
- Empty/loading/error: inherits the existing Global platform settings page states.
- Copy & i18n: add keys to the platform-settings i18n bundle (web) alongside existing flag toggles.

No learner-facing UI here.

## 11. AI / ML Considerations

Not AI-touching. (Personalized marketplace recommendations are a future enhancement — out of scope; noted in §18.)

## 12. Integration Points

- **Internal modules** — `server/internal/repos/platformconfig/features.go`, `config/config.go`, `httpserver/platform_features.go`, `httpserver/settings_platform.go`, `repos/platformconfig/patch.go`, `repos/billing/entitlements.go`, `server/migrations/`.
- **Downstream consumers** — MKT2 (listing write), MKT3 (storefront read gated by helper), MKT4 (entitlement create), MKT5 (purchased indicator), MKT6 (mobile feature model).
- **No external services** touched in this story.

## 13. Dependencies & Sequencing

- **Must ship after** — 15.1 (catalog columns/service) and 15.3 (entitlements table) already merged; verify at branch time.
- **Must ship before** — all other MKT stories.
- **Shared infra** — none new; reuses Postgres, config cache, admin-audit.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Confusion with existing `FFMarketplace` (plugin) flag | H | M | Distinct name `FFCourseMarketplace`, comments in `config.go`/`features.go`, README callout, PR checklist item |
| Default-ON surprises operators who don't want selling | M | M | Document in release notes; storefront is empty until instructors list courses, so ON is low-risk; admins can disable |
| Making `stripe_event_id` nullable weakens Stripe idempotency | L | H | Keep the existing unique index on `stripe_event_id`; add the separate `(user_id, course_id)` partial unique index for course grants |
| Free-claim entitlements pollute revenue reporting | M | L | Filter reports by `acquisition_source`/`amount_paid_cents > 0`; document in 15.8/analytics |

## 15. Rollout Plan

- **Feature flag** — `FFCourseMarketplace`, default **ON**. (Exception to the usual default-off convention; called out in review.)
- **Sequencing** — schema migration → config/flag plumbing → helpers → flip flag defaults on. Because dependent UI ships later, ON has no user-visible effect until MKT2/MKT3 land, so the flag can safely default on from day one.
- **Dogfood** — internal tenant lists a couple of free courses once MKT2/3 exist.
- **GA criteria** — flag resolves correctly, entitlement idempotency verified, no regression in existing billing/catalog tests.
- **Rollback** — flip flag off (soft, data-preserving) or run the down migration.

## 16. Test Plan

- **Unit** — `applyPlatformBools` default-ON; `MarketplaceAccess` truth table (free/paid/subscription/none); free-claim idempotency under duplicate insert.
- **Integration** — DB: partial unique index rejects a second active `course_purchase` for the same `(user,course)`; migration up/down round-trips; Settings patch flips `ff_course_marketplace` and is reflected in `/platform/features`.
- **End-to-end** — none in this story (no UI); `/platform/features` payload asserted via API test.
- **Security** — non-admin cannot patch the flag; marketplace endpoints (stubbed in later stories) 404 when off.
- **Accessibility** — N/A.
- **Performance** — verify no extra per-request DB round-trip for flag reads.
- **Manual** — toggle flag in Settings, confirm `ffCourseMarketplace` in the features response.

## 17. Documentation & Training

- **Admin docs** — "Enabling/disabling the course marketplace" in the Global platform settings help.
- **API reference** — regenerated OpenAPI for `/platform/features` + Settings platform.
- **Internal runbook** — add `FFCourseMarketplace` to the flag registry doc and note the `FFMarketplace` distinction.

## 18. Open Questions

1. Should marketplace listing be gated on `is_public` instead of a new `marketplace_listed` column? (Default decision: separate column, for independence of SEO catalog vs. in-app store. Revisit if product wants them unified.)
2. Should there be a tenant-level allow-list of who may *sell* (list courses) vs. anyone with course-edit rights? (Default: reuse `course:{code}:item:create`; MKT2 owns this.)
3. Do we need per-course currency, or a single tenant currency? (Default: reuse existing per-course `price_currency`; revisit for multi-currency payout in 15.8.)
4. Should `comp` (complimentary) grants have an admin UI now or later? (Default: schema supports it; UI deferred.)

## 19. References

- Existing files: `server/internal/repos/platformconfig/features.go`, `server/internal/config/config.go` (`FFMarketplace` at ~L535), `server/internal/httpserver/platform_features.go`, `server/internal/repos/billing/entitlements.go`, `server/migrations/276_public_course_catalog.sql`, `278_billing_stripe.sql`.
- Related plans: [MKT2](MKT2-course-marketplace-listing-settings.md), [MKT4](MKT4-course-purchase-entitlement-flow.md), `docs/completed/15-self-learner-specific/15.1-public-course-catalog.md`, `15.3-billing-stripe.md`.
