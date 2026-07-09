# MKT3 — Marketplace Discovery & Storefront (Web)

> Implementation plan. Source: [docs/plan/marketplace/README.md](README.md). Part of the Marketplace epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKT3 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | MKT1, MKT2 |
| **Unblocks** | MKT4, MKT5 |

---

## 1. Problem Statement

There is no in-app place for a signed-in learner to browse courses they can enroll in. The unauthenticated public catalog (15.1, `explore-catalog-page.tsx`) is SEO-oriented and lives outside the app shell; it doesn't appear in the sidenav, doesn't know who the viewer is, and can't show "you already own this." This story adds a **Marketplace** destination to the web sidenav and an authenticated storefront — browse, filter, and a course detail page with a clear **Free / price** call-to-action — reusing the existing catalog search service so we don't rebuild search.

## 2. Goals

- Add a **Marketplace** item to the web sidenav, gated on `ffCourseMarketplace`.
- Build an authenticated storefront page: searchable, filterable grid of marketplace-listed courses with Free/price badges.
- Build a marketplace course detail page with description, what's-included, price, and a primary **Enroll (Free) / Buy** button (the action itself is implemented in MKT4).
- Reflect ownership: courses the viewer already purchased/claimed show "Owned / Go to course" instead of a buy CTA.
- Reuse the `catalogsearch` service and `PublicCatalogFilter` so ranking, categories, and pagination match the public catalog.

## 3. Non-Goals

- The actual purchase/claim transaction and entitlement creation (MKT4) — this story only renders the CTA and routes to it.
- The Courses-page "Purchased" indicator (MKT5).
- Mobile storefront (MKT6).
- Reviews/ratings authoring (existing 15.7) — the detail page may *display* existing ratings but doesn't add authoring.
- Personalized recommendations / "because you enrolled in…" (future).

## 4. Personas & User Stories

- **As a self-learner**, I want to browse available courses in the app and filter by topic/level/price so that I can find something to enroll in.
- **As a learner**, I want free courses clearly marked "Free" so that I know I can start immediately.
- **As a returning learner**, I want courses I already own to show "Go to course" so that I don't try to buy them twice.
- **As an instructor**, I want my listed course to appear in the storefront with an accurate card so that learners can find it.

## 5. Functional Requirements

- **FR-1.** The system MUST add a sidenav link "Marketplace" (`/marketplace`) in `side-nav-main-links.tsx`, rendered only when `ffCourseMarketplace` is true, in the "Learning" group near "Course catalog".
- **FR-2.** The storefront MUST list courses where `marketplace_listed = true` and the course is published, via a new authenticated endpoint `GET /api/v1/marketplace/courses` reusing `catalogsearch`.
- **FR-3.** Each card MUST show hero image, title, category, level, enrollment count, average rating (if any), and a price badge: **"Free"** when `price_cents = 0`, else formatted price (with `list_price_cents` strikethrough when present).
- **FR-4.** The storefront MUST support search (`q`), and filters for category, level, language, and price (free-only / max price), plus sort (popular, newest, price), reusing the existing filter/sort validation.
- **FR-5.** For each listed course, the response MUST include an `owned` boolean (from `billing.MarketplaceAccess`) so cards can render "Owned".
- **FR-6.** A course detail page (`/marketplace/:slug`) MUST show full description, "what's included" (module/item counts, estimated duration), price, instructor, and rating; and a primary CTA: **"Enroll — Free"** (price 0), **"Buy — $X"** (paid, unowned), or **"Go to course"** (owned).
- **FR-7.** The CTA MUST route into the MKT4 purchase/claim flow (free → claim; paid → checkout). If the marketplace flag is off, `/marketplace*` routes MUST redirect to the dashboard / show not-available.
- **FR-8.** The storefront MUST paginate (cursor-based, reusing `DecodeCatalogCursor`) and handle empty/loading/error states.
- **FR-9.** The detail endpoint `GET /api/v1/marketplace/courses/{slug}` MUST return `404` for non-listed/unpublished courses even if they exist in the public catalog.
- **FR-10.** Cards MUST link by `catalog_slug`; the page MUST be resilient to a missing slug (fallback to course code).

## 6. Non-Functional Requirements

- **Performance** — Storefront list p95 < 300 ms; reuse the catalog object cache (`objectcache`) keyed to include the marketplace filter + viewer-ownership computed post-cache (ownership is per-user, so cache the listing, resolve `owned` per request via one batched query). Lazy-load the route (`lazy-pages.ts`).
- **Security** — Authenticated endpoints require a session; `owned` is computed for the requesting user only. No price is trusted from the client.
- **Privacy & Compliance** — Only aggregate, non-PII course metadata on cards (enrollment_count, average_rating) — same as 15.1.
- **Accessibility** — Grid is a semantic list; cards are single focusable links with accessible names including price/Free; filters are labelled; CTA buttons have discernible text; WCAG 2.1 AA; keyboard and screen-reader navigable.
- **Scalability** — Partial index `idx_courses_marketplace` (MKT1) keeps listing queries cheap; pagination bounds payloads.
- **Reliability** — Cache miss falls back to DB; ownership query failure degrades to hiding the "Owned" badge (never blocks browsing).
- **Observability** — Emit `marketplace_storefront_view`, `marketplace_detail_view{owned}`, and search facet usage; funnel metric feeding MKT4.
- **Maintainability** — Reuse `catalogsearch`, `CatalogCourseHero`, and existing card/pill components; do not fork the public-catalog page.
- **Internationalization** — Price via `Intl.NumberFormat`, all copy externalised, RTL-safe; language filter reuses `catalog_language`.
- **Backward compatibility** — New routes/endpoints are additive; public catalog (15.1) unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* the flag is on, *When* I open the app, *Then* a "Marketplace" link appears in the sidenav and routes to `/marketplace`.
- **AC-2.** *Given* two published courses are marketplace-listed (one Free, one $20), *When* I open the storefront, *Then* both appear with correct "Free" and "$20.00" badges; a listed-but-draft course does not appear.
- **AC-3.** *Given* I already own a listed course, *When* I view the storefront and its detail page, *Then* it shows "Owned" / "Go to course" instead of a buy CTA.
- **AC-4.** *Given* I filter by "Free only" and category X, *When* results load, *Then* only free, category-X, listed, published courses appear.
- **AC-5.** *Given* the flag is off, *When* I navigate to `/marketplace`, *Then* I'm redirected/see "not available" and no sidenav link is shown.
- **AC-6.** *Given* a paid, unowned course, *When* I click "Buy", *Then* I enter the MKT4 checkout flow (verified by route/handoff).
- **AC-7.** *Given* the network fails, *When* the storefront loads, *Then* an error state with retry is shown (no blank page).

## 8. Data Model

No schema changes. Reads `course.courses` (marketplace + catalog columns from MKT1/15.1) and `billing.user_entitlements` (for `owned`). Adds a repo query `course.ListMarketplaceCourses(filter)` (thin wrapper over `catalogsearch` constrained to `marketplace_listed = true`) and a batched `billing.OwnedCourseIDs(userID, courseIDs)` for ownership resolution.

## 9. API Surface

New authenticated routes (gated by `courseMarketplaceOff`, MKT1):

- `GET /api/v1/marketplace/courses` — query params mirror public catalog (`q, category, level, language, price_max, sort, cursor, limit`) plus `free_only`. Response: `{ courses: MarketplaceCard[], nextCursor, categories? }` where each card includes `owned: boolean`.
- `GET /api/v1/marketplace/categories` — category facets for listed courses.
- `GET /api/v1/marketplace/courses/{slug}` — detail: `{ course, owned, priceCents, priceCurrency, listPriceCents?, whatsIncluded, rating }`; `404` if not listed/published.
- Rate-limit: standard authenticated read limits.
- OpenAPI: document the marketplace read endpoints.

```ts
type MarketplaceCard = {
  slug: string; courseCode: string; title: string; heroImageUrl: string | null
  category: string | null; level: string | null; language: string
  priceCents: number; priceCurrency: string; listPriceCents: number | null
  enrollmentCount: number; averageRating: number | null; owned: boolean
}
```

## 10. UI / UX

- **New pages** — `clients/web/src/pages/marketplace/marketplace-page.tsx` (storefront) and `marketplace-course-page.tsx` (detail); registered in `app.tsx` + `lazy-pages.ts`.
- **New sidenav** — add the link in `side-nav-main-links.tsx` under "Learning" (icon e.g. `Store` / `ShoppingBag` from lucide), gated on `ffCourseMarketplace`.
- **Reused components** — `CatalogCourseHero`, `CourseCatalogStatusPill` (or a new `MarketplacePriceBadge`), filter controls patterned on `explore-catalog-page.tsx`.
- **Flows** —
  1. Sidenav → Marketplace → grid.
  2. Search/filter/sort → results update.
  3. Click card → detail page.
  4. CTA (Free/Buy/Go to course) → MKT4 or course.
- **States** — loading skeleton grid; empty ("No courses available yet" for learners; instructor hint to list courses); error with retry; owned badge; price/Free badge.
- **Responsive** — 1/2/3-column grid by breakpoint; filters collapse into a sheet on mobile web.
- **Accessibility** — list semantics, focus order filters→results, CTA discernible text, price in accessible name.
- **Copy & i18n** — `marketplace.title`, `.searchPlaceholder`, `.free`, `.buy`, `.owned`, `.goToCourse`, `.empty`, `.filters.*`.

## 11. AI / ML Considerations

Not AI-touching in this story. (Recommendation ranking is a future enhancement; the current sort reuses popularity/rating/price from `catalogsearch`.)

## 12. Integration Points

- **Internal** — `clients/web/src/pages/marketplace/*` (new), `components/layout/side-nav-main-links.tsx`, `context/platform-features-context.tsx` (`ffCourseMarketplace`), `app.tsx`, `lazy-pages.ts`; server `httpserver/marketplace_courses_http.go` (new — distinct from plugin `marketplace_http.go`), `service/catalogsearch`, `repos/course`, `repos/billing`.
- **Handoff** — CTA → MKT4 endpoints/route.
- **Cache** — reuse `objectcache` catalog page caching for the listing portion.

## 13. Dependencies & Sequencing

- **After** — MKT1 (flag, columns, index), MKT2 (so there are listed courses).
- **Before** — MKT4 (buy), MKT5 (owned indicator on Courses page reuses the ownership query).
- **Shared infra** — object cache, catalog search.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Per-user `owned` breaks shared catalog cache | H | M | Cache the listing (no ownership) and resolve `owned` per request with one batched query over the page's course ids |
| Duplication with public-catalog page code | M | M | Extract shared card/filter components; reuse `catalogsearch`/`PublicCatalogFilter` |
| Empty storefront at launch looks broken | M | L | Purposeful empty state; seed a few first-party free courses |
| Slug collisions / missing slug | L | M | Reuse unique public-catalog slug; fallback to course code |

## 15. Rollout Plan

- **Flag** — `ffCourseMarketplace` (MKT1); no separate flag. Sidenav link + routes appear when on.
- **Sequencing** — after MKT1/MKT2; can ship the CTA in "route to MKT4" stub state and light up when MKT4 lands.
- **Dogfood** — internal tenant browses seeded free/paid listings.
- **GA criteria** — browse/filter/detail work, ownership badge correct, flag-off redirect works.
- **Rollback** — flag off hides nav + routes.

## 16. Test Plan

- **Unit** — price/Free badge formatting; filter param mapping; card accessible-name composition.
- **Integration** — listing endpoint returns only listed+published; detail 404 for non-listed; `owned` reflects entitlements; pagination cursor round-trip.
- **End-to-end (Playwright)** — sidenav→storefront→filter→detail→CTA; owned course shows "Go to course"; flag-off redirect.
- **Security** — unauthenticated request to `/api/v1/marketplace/courses` rejected; price authoritative server-side.
- **Accessibility** — axe on storefront + detail; keyboard-only browse; screen-reader reads price/Free/owned.
- **Performance** — cached listing p95; ownership batch query single round-trip.
- **Manual** — RTL + locale currency formatting; empty/error states.

## 17. Documentation & Training

- **Learner docs** — "Browsing the marketplace."
- **API reference** — marketplace read endpoints.
- **Instructor docs** — cross-link from MKT2 ("your listed course appears here").

## 18. Open Questions

1. Should the storefront also surface course **bundles/learning paths** (15.4) or courses only for v1? (Default: courses only; bundles later.)
2. Does the detail page reuse the public course landing page (`explore-course-page.tsx`) or a new authenticated page? (Default: new authenticated page to show `owned` + in-app CTA; share sub-components.)
3. Should unauthenticated users hitting `/marketplace` be sent to login or to the public catalog? (Default: login, then return to marketplace.)
4. Which lucide icon for the nav item? (Proposed: `Store`.)

## 19. References

- Existing files: `clients/web/src/pages/explore-catalog-page.tsx`, `explore-course-page.tsx`, `components/layout/side-nav-main-links.tsx`, `server/internal/service/catalogsearch`, `httpserver/public_catalog_http.go`.
- Related plans: [MKT1](MKT1-marketplace-platform-foundation.md), [MKT2](MKT2-course-marketplace-listing-settings.md), [MKT4](MKT4-course-purchase-entitlement-flow.md), `docs/completed/15-self-learner-specific/15.1-public-course-catalog.md`.
