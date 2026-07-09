# MKT2 — Course Marketplace Listing & Pricing Settings (Web)

> Implementation plan. Source: [docs/plan/marketplace/README.md](README.md). Part of the Marketplace epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKT2 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | MKT1 |
| **Unblocks** | MKT3, MKT4 |

---

## 1. Problem Statement

Instructors and creators have no way to put a course into the marketplace or set its price. Course settings already host a catalog-listing section (`GET/PUT /api/v1/courses/{code}/catalog-listing`, `course-catalog-listing.go`), but it only controls the unauthenticated public catalog (15.1) and has no marketplace concept. This story adds a **Marketplace** control to course settings: a toggle to list the course in the in-app storefront and a fee editor that defaults to **Free**. Without it, MKT3's storefront would always be empty.

## 2. Goals

- Add a course-settings **Marketplace** section (web) with: list-in-marketplace toggle, fee amount (currency-aware) defaulting to **Free ($0)**, and a live preview of how the storefront card will read.
- Persist `marketplace_listed`, `price_cents`, `price_currency` through a settings API extending the existing catalog-listing endpoint.
- Enforce publish/permission rules: only course editors may change it, and only published courses may be listed.
- Give clear guardrails around changing a fee after learners have already purchased.

## 3. Non-Goals

- The storefront/browse UI (MKT3) and the buy/claim flow (MKT4).
- Mobile course settings (MKT6).
- Payout configuration, coupons, or tax settings (existing 15.8 / 15.13 surfaces).
- Bulk listing of many courses at once (future admin tooling).

## 4. Personas & User Stories

- **As an instructor**, I want to flip my course into the marketplace and set a price so that learners can enroll (free or paid).
- **As a creator (self-learner market)**, I want the default to be Free so that listing is one click and monetization is opt-in.
- **As a course admin**, I want to be warned before changing the price of a course people already bought so that I don't create billing confusion.
- **As a platform admin**, I want the whole section hidden when the marketplace flag is off so that the UI stays coherent.

## 5. Functional Requirements

- **FR-1.** The system MUST add a "Marketplace" section to the course settings page, visible only when `ffCourseMarketplace` is true and the viewer holds `course:{code}:item:create`.
- **FR-2.** The section MUST provide a toggle "List in marketplace" bound to `marketplace_listed`.
- **FR-3.** The section MUST provide a fee editor: a currency selector (default from course `price_currency` or tenant default `usd`) and an amount input in major units, stored as `price_cents`. The default and the value shown when amount is 0 MUST render as **"Free"**.
- **FR-4.** The client MUST reject negative amounts and non-numeric input, and SHOULD cap at a sane maximum (e.g. 99,999.99) before submit; the server MUST re-validate (`price_cents >= 0`).
- **FR-5.** The server MUST reject listing a course whose workflow state is `draft`/unpublished with `422`/`CodeInvalidInput` ("Publish the course before listing it in the marketplace.").
- **FR-6.** Saving MUST persist via `PUT /api/v1/courses/{code}/catalog-listing` extended with `marketplaceListed` and (existing) `priceCents`/`priceCurrency`, returning the updated listing.
- **FR-7.** When a fee is changed on a course that already has ≥1 active `course_purchase` entitlement, the UI MUST show a confirmation ("Existing purchasers keep their access; the new price applies to future enrollments.") and the change MUST NOT retroactively alter existing entitlements.
- **FR-8.** Unlisting a course (`marketplace_listed = false`) MUST set `marketplace_listed_at = NULL` and MUST NOT revoke existing enrollments or entitlements.
- **FR-9.** The section MUST show a read-only "Storefront preview" reflecting title, price/Free badge, category, and level as the learner will see them (MKT3 card).
- **FR-10.** All writes MUST invalidate the catalog cache (reuse `invalidateCatalogCache`) so storefront listings update promptly.

## 6. Non-Functional Requirements

- **Performance** — Settings load reuses the existing listing GET (single query); save is one `UPDATE`. No N+1.
- **Security** — Server re-checks `course:{code}:item:create` on every write (already enforced in `handlePutCourseCatalogListing`); client-side gating is convenience only. Price is authoritative server-side to prevent tampering.
- **Privacy & Compliance** — No new PII. Price is public course metadata.
- **Accessibility** — Toggle and inputs are labelled; the "Free"/price state is announced; confirmation dialog is focus-trapped and keyboard-dismissible; currency + amount have associated `<label>`s and inline error text via `aria-describedby`. WCAG 2.1 AA.
- **Scalability** — No new load; per-course write.
- **Reliability** — Save is idempotent; optimistic UI reverts on error with a toast.
- **Observability** — Emit `marketplace_listing_saved{listed, free}` and log via admin-audit (course settings change).
- **Maintainability** — Extend the existing `course-catalog-settings` / `catalog-listing` module rather than forking a new one.
- **Internationalization** — All labels, the "Free" badge, and currency formatting use the i18n layer (`Intl.NumberFormat` for currency). RTL-safe.
- **Backward compatibility** — Additive fields on an existing endpoint; older clients that don't send `marketplaceListed` leave it unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* the marketplace flag is on and I can edit a published course, *When* I open course settings, *Then* I see a Marketplace section with the toggle off and fee "Free" by default.
- **AC-2.** *Given* I toggle "List in marketplace" and save, *When* the request completes, *Then* `marketplace_listed = true`, `marketplace_listed_at` is set, and the storefront (MKT3) shows the course.
- **AC-3.** *Given* I enter a fee of 19.99 USD and save, *When* I reload, *Then* the amount persists as `price_cents = 1999`, `price_currency = 'usd'`.
- **AC-4.** *Given* a course in draft, *When* I try to list it, *Then* the server returns `422` and the UI shows "Publish the course before listing it in the marketplace."
- **AC-5.** *Given* a course with existing purchasers, *When* I change its price, *Then* I must confirm, and existing entitlements are unchanged after save.
- **AC-6.** *Given* the marketplace flag is off, *When* I open course settings, *Then* the Marketplace section is not rendered.
- **AC-7.** *Given* I enter a negative amount, *When* I try to save, *Then* the client blocks it and, if bypassed, the server returns `400`.

## 8. Data Model

No new tables. Uses MKT1 columns: `course.courses.marketplace_listed`, `marketplace_listed_at`, `price_cents`, `price_currency`. This story extends the repo helpers:

- `course.CatalogListing` struct gains `MarketplaceListed bool` and `PriceCurrency string`.
- `course.SetCatalogListing` writes `marketplace_listed` / `marketplace_listed_at` (set to `NOW()` when true, `NULL` when false) and `price_currency`.
- `course.GetCatalogListing` returns the new fields plus `PublishState` and `ActivePurchaseCount` (for the FR-7 confirmation).

## 9. API Surface

Extend the existing endpoints (no new routes):

- `GET /api/v1/courses/{course_code}/catalog-listing` → response `listing` adds:
  ```ts
  { marketplaceListed: boolean, priceCents: number, priceCurrency: string,
    publishState: 'draft' | 'published' | ..., activePurchaseCount: number }
  ```
- `PUT /api/v1/courses/{course_code}/catalog-listing` → request body (`catalogListingBody`) adds `marketplaceListed: boolean` and `priceCurrency: string` alongside existing `priceCents`. Auth scope unchanged (`course:{code}:item:create`). Validation: `priceCents >= 0`; reject listing when `publishState != published` (FR-5).
- Rate-limit: reuse existing settings write limits.
- OpenAPI: update the catalog-listing schema.

## 10. UI / UX

- **New component** — `course-marketplace-settings-section.tsx` under `clients/web/src/pages/lms/`, embedded in `course-settings.tsx` next to the existing catalog/public-listing section.
- **Flow**:
  1. Open course settings → Marketplace section.
  2. Toggle "List in marketplace".
  3. Choose "Free" (default) or enter a price + currency.
  4. Save → toast confirmation; storefront preview updates.
- **States** — loading (skeleton reusing settings pattern); empty/default (toggle off, Free); error (inline + toast, optimistic revert); disabled (draft course: toggle disabled with helper "Publish first"); confirmation dialog for price change with existing purchasers.
- **Responsive** — stacks on mobile web; inputs full-width < 640px.
- **Accessibility** — label associations, `aria-describedby` for errors/help, focus-trapped dialog, currency announced.
- **Copy & i18n keys** — `course.settings.marketplace.title`, `.listToggle`, `.fee`, `.free`, `.currency`, `.publishFirst`, `.priceChangeWarning`, `.preview`.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Internal** — `clients/web/src/pages/lms/course-settings.tsx`, new `course-marketplace-settings-section.tsx`, `clients/web/src/lib/course-catalog-settings-api.ts` (extend), `server/internal/httpserver/course_catalog_listing.go`, `server/internal/repos/course` (`CatalogListing`, `SetCatalogListing`, `GetCatalogListing`).
- **Feature context** — `usePlatformFeatures().ffCourseMarketplace` gates rendering.
- **Cache** — `invalidateCatalogCache` on write.

## 13. Dependencies & Sequencing

- **After** — MKT1 (columns, flag).
- **Before** — MKT3 (storefront needs listed courses), MKT4 (buy flow).
- **Shared infra** — none new.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Instructors conflate "public catalog" and "marketplace" toggles | M | M | Distinct labels + help text; group under one "Discoverability" heading with clear subtitles |
| Price changed after purchases causes support confusion | M | M | FR-7 confirmation + "existing purchasers keep access" copy; never mutate existing entitlements |
| Currency mismatch with Stripe account currency | M | M | Validate against tenant's supported currencies (from 15.3 config) on save |
| Listing a half-built course | M | L | FR-5 publish gate |

## 15. Rollout Plan

- **Flag** — gated by `ffCourseMarketplace` (MKT1); no separate flag.
- **Sequencing** — ship after MKT1; section appears immediately for editors when flag on.
- **Dogfood** — internal creators list a free and a paid course.
- **GA criteria** — save/reload round-trips, publish gate enforced, confirmation on price change.
- **Rollback** — hide the section (flag off) or revert component; data columns remain.

## 16. Test Plan

- **Unit** — currency/amount → `price_cents` conversion; "Free" rendering at 0; client validation of negatives/max.
- **Integration** — PUT persists `marketplace_listed` + price; draft course rejected with 422; non-editor gets 403; cache invalidated.
- **End-to-end (Playwright)** — list a course, set price, reload persists; toggle off keeps enrollments; price-change confirmation for a course with a seeded purchase.
- **Security** — authz matrix: editor vs. non-editor; server rejects tampered negative/oversized price.
- **Accessibility** — axe on the section + dialog; keyboard-only save; screen-reader announces Free/price and errors.
- **Performance** — single query load/save.
- **Manual** — currency formatting across locales/RTL.

## 17. Documentation & Training

- **Instructor docs** — "List your course in the marketplace and set a price."
- **API reference** — updated catalog-listing schema.
- **Help center** — note on changing prices after purchases.

## 18. Open Questions

1. Should "public catalog" (`is_public`) and "marketplace" (`marketplace_listed`) be presented as one combined "Discoverability" control or two toggles? (Default: two toggles, grouped.)
2. Is per-course currency selection desired, or lock to tenant currency? (Default: allow currency field but validate against Stripe-supported set.)
3. Minimum paid price (Stripe minimum charge ~ $0.50)? (Default: enforce Stripe minimum for non-zero prices.)
4. Should unpublishing a listed course auto-unlist it? (Default: yes — MKT1/MKT3 storefront query already excludes unpublished; consider explicit unlist on unpublish.)

## 19. References

- Existing files: `server/internal/httpserver/course_catalog_listing.go`, `clients/web/src/lib/course-catalog-settings-api.ts`, `clients/web/src/pages/lms/course-settings.tsx`, `server/internal/repos/course` catalog listing helpers.
- Related plans: [MKT1](../../completed/marketplace/MKT1-marketplace-platform-foundation.md), [MKT3](MKT3-marketplace-discovery-web.md), `docs/completed/15-self-learner-specific/15.1-public-course-catalog.md`.
