# Marketplace — Course Storefront

> Epic: an in-app **course marketplace** (Coursera-style) where learners discover and "purchase" access to courses. Purchases may be **Free** (default) or paid. Instructors opt a course in from course settings and set a fee. The marketplace is a platform-wide, toggleable feature flag, **on by default**.

Each story follows [`../_TEMPLATE.md`](../_TEMPLATE.md). This folder is the working set; move stories to `docs/completed/marketplace/` when shipped.

## ⚠️ Naming: this is NOT the existing `FFMarketplace`

The codebase already has a flag named `FFMarketplace` — it gates the **plugin / OAuth-2.1 app marketplace** (plan 16.9, `server/internal/httpserver/marketplace_http.go`). That is unrelated to this epic.

To avoid a collision, the **course** marketplace introduces a **new** flag: **`FFCourseMarketplace`** (DB column `ff_course_marketplace`, JSON `ffCourseMarketplace`). Unlike almost every other flag, its default is **ON** (`true`). See [MKT1](../../completed/marketplace/MKT1-marketplace-platform-foundation.md) (shipped).

## What already exists (reuse, do not rebuild)

| Capability | Where |
|---|---|
| Course pricing columns `price_cents`, `price_currency`, `is_public`, `catalog_slug`, `catalog_category`, `difficulty_level`, `enrollment_count`, `average_rating` | `server/migrations/276_public_course_catalog.sql`, `278_billing_stripe.sql` |
| Entitlements (`billing.user_entitlements`, type `course_purchase`) + `HasCourseAccess()` | `server/internal/repos/billing/entitlements.go` |
| Public catalog search service + API | `server/internal/service/catalogsearch`, `httpserver/public_catalog_http.go` (plan 15.1) |
| Stripe checkout + webhook + `/me/entitlements` | `httpserver/billing_http.go`, `service/billing`, `background/payment_webhook_worker.go` (plan 15.3) |
| Self-paced enroll → `course.course_enrollments` + role grants | `httpserver/course_self_paced.go` (plan 15.2) |
| Catalog-listing settings `GET/PUT /api/v1/courses/{code}/catalog-listing` | `httpserver/course_catalog_listing.go` |
| Web sidenav + platform-features context | `clients/web/src/components/layout/side-nav-main-links.tsx`, `context/platform-features-context.tsx` |
| Mobile nav + feature model | Android `core/navigation/MobileDestinations.kt`, iOS `Core/Routing/MobileDestinations.swift` |

The marketplace is therefore mostly a **UX + workflow layer** over shipped commerce infrastructure, plus a small amount of new data (a `marketplace_listed` flag and generalized entitlement idempotency for free claims).

## Stories

| ID | Title | Effort | Depends on |
|---|---|---|---|
| [MKT1](../../completed/marketplace/MKT1-marketplace-platform-foundation.md) ✅ | Marketplace platform foundation & feature flag | M | 15.1, 15.3 |
| [MKT2](../../completed/marketplace/MKT2-course-marketplace-listing-settings.md) ✅ | Course marketplace listing & pricing settings (web) | M | MKT1 |
| [MKT3](../../completed/marketplace/MKT3-marketplace-discovery-web.md) ✅ | Marketplace discovery & storefront (web) | M | MKT1 |
| [MKT4](../../completed/marketplace/MKT4-course-purchase-entitlement-flow.md) ✅ | Course purchase & entitlement flow (free + paid) | L | MKT1, MKT3 |
| [MKT5](MKT5-purchased-indicator-courses.md) | "Purchased" indicator & My purchases | S | MKT4 |
| [MKT6](MKT6-marketplace-mobile.md) | Marketplace on mobile (iOS + Android) | L | MKT1–MKT5 |

## Sequencing

```
MKT1 ──┬─> MKT2 ─────────────┐
       ├─> MKT3 ──> MKT4 ──> MKT5 ──> MKT6
       └────────────────────────────────┘
```

Ship order: **MKT1 → MKT2/MKT3 (parallel) → MKT4 → MKT5 → MKT6.** MKT2 (instructors list courses) and MKT3 (learners browse) can be built in parallel once the flag and data model land; MKT4 wires the actual buy/claim; MKT5 closes the loop on the courses page; MKT6 brings the whole flow to mobile (with the App Store / Play in-app-purchase decision called out in that story).

## Cross-cutting decisions

1. **New dedicated flag** `FFCourseMarketplace`, default **ON**, admin-toggleable in Settings → Global platform.
2. **`marketplace_listed` is distinct from `is_public`.** `is_public` drives the unauthenticated SEO catalog (15.1); `marketplace_listed` drives the authenticated in-app storefront. A course can be in one, both, or neither. (Alternative — gating solely on `is_public` — is captured in MKT1 Open Questions.)
3. **Fee reuses `price_cents` / `price_currency`.** Default `0` = **Free**. No new pricing columns.
4. **"Purchase" always yields an entitlement + an enrollment**, whether Free or paid, so purchased courses appear on the Courses page and the learner has content access. Free claims are instant; paid claims complete via the existing Stripe webhook.
5. **Mobile paid purchases raise an App Store / Play Billing policy question** — see MKT6 §14/§18. Free claims are unaffected.
