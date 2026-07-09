# MKT7 — Public Marketplace API (self.lextures.com)

> Implementation plan. Source: [docs/plan/marketplace/README.md](README.md). Part of the Marketplace epic — **www storefront sub-epic (MKT7–MKT10).**

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKT7 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Server platform team |
| **Depends on** | MKT1 (flag + `marketplace_listed` column), MKT3 (`repos/course/marketplace_storefront.go`) |
| **Unblocks** | MKT8, MKT9, MKT10 |

---

## 1. Problem Statement

The public marketing site (`lextures.com`, the `www/` app) needs a **Courses** tab that lists marketplace courses, but every existing marketplace listing endpoint (`GET /api/v1/marketplace/courses`, MKT3) **requires an authenticated session** and computes per-user ownership. There is no unauthenticated way to read the marketplace. The pre-existing public catalog API (`/api/v1/public/catalog/*`, plan 15.1) is unauthenticated but filters on `is_public`, which is the SEO catalog — a *different* course set than what instructors opt into the in-app marketplace via `marketplace_listed`. This story adds a thin **unauthenticated public marketplace API** that mirrors the in-app storefront (gated on `marketplace_listed = TRUE AND published = TRUE`) so the www site can render exactly the marketplace catalog without a login.

## 2. Goals

- Expose unauthenticated read endpoints for marketplace-listed courses under `/api/v1/public/marketplace/*`.
- Reuse the existing `repos/course/marketplace_storefront.go` query surface (`ListMarketplaceCourses`, `GetMarketplaceCourseBySlug`, `ListMarketplaceCategories`, `GetMarketplaceWhatsIncluded`) — do **not** fork the storefront query.
- Serve CDN-friendly, cacheable responses (no per-user data) suitable for a cross-origin static site.
- Gate on `FFCourseMarketplace` so the endpoints disappear when the marketplace is disabled.
- Document the endpoints in OpenAPI.

## 3. Non-Goals

- Any web/www UI (MKT8 storefront, MKT9 detail, MKT10 SEO).
- Purchase / checkout / entitlement flows (MKT4, already shipped, authenticated only).
- Per-user `owned` resolution — the public API is anonymous by definition; ownership is never computed here.
- Changing the authenticated in-app endpoints (`/api/v1/marketplace/*`) or the public **catalog** endpoints (`/api/v1/public/catalog/*`); both remain as-is.
- Reviews **authoring** — the public reviews read endpoint already exists (see §9) and is reused.

## 4. Personas & User Stories

- **As an anonymous visitor** on lextures.com, I want the site to fetch the list of marketplace courses without me logging in, so that I can browse before signing up.
- **As a search engine crawler**, I want a stable, cacheable JSON (and, via MKT10, JSON-LD) representation of each marketplace course.
- **As an instructor** who listed a course in the marketplace, I want it to appear on the public site automatically once published + listed.
- **As a platform admin**, I want to hide the entire public marketplace by toggling `FFCourseMarketplace` off.

## 5. Functional Requirements

- **FR-1.** The system MUST register unauthenticated routes:
  `GET /api/v1/public/marketplace/courses`, `GET /api/v1/public/marketplace/categories`, `GET /api/v1/public/marketplace/courses/{slug}` in a new `server/internal/httpserver/public_marketplace_http.go` (mirroring `registerPublicCatalogRoutes`).
- **FR-2.** All three endpoints MUST return `404` (`{"code":"not_found"}`) when `FFCourseMarketplace` is disabled, via a public analogue of `courseMarketplaceOff` (e.g. `publicMarketplaceOff`).
- **FR-3.** The list endpoint MUST return only courses where `marketplace_listed = TRUE AND published = TRUE`, by reusing `repoCourse.ListMarketplaceCourses` with a `MarketplaceFilter`.
- **FR-4.** The list endpoint MUST accept and validate the same query params as the in-app storefront: `q`, `category`, `level`, `language`, `price_max`, `free_only`, `sort`, `cursor`, `limit` — reusing `ValidDifficultyLevel`, `ValidCatalogSort`, and `DecodeCatalogCursor`. Invalid params MUST return `400` (`invalid_input`).
- **FR-5.** The list response MUST **omit** any `owned` field (there is no viewer). `applyMarketplaceOwnership` MUST NOT be called on this path.
- **FR-6.** The detail endpoint MUST return `{ course, whatsIncluded }` for a listed+published course by `catalog_slug`, and `404` for any slug that is not listed+published (even if it exists as a private or catalog-only course).
- **FR-7.** The categories endpoint MUST return category facets computed over listed+published courses (`repoCourse.ListMarketplaceCategories`).
- **FR-8.** All responses MUST set public, CDN-friendly cache headers and `ETag` (reuse `publicCatalogCacheHeaders` / `writeJSONWithETag`), and MAY reuse the object cache keyed by the marketplace filter.
- **FR-9.** CORS MUST allow the www origin. The global `corsAll` middleware already sets `Access-Control-Allow-Origin: *`; this story MUST verify (via test) that the public marketplace routes are covered by it.
- **FR-10.** For course reviews, the www site MUST reuse the **existing** `GET /api/v1/public/catalog/courses/{slug}/reviews` (already unauthenticated, by slug). This story MUST confirm that endpoint resolves reviews for marketplace-listed courses by `catalog_slug`; if it is scoped to `is_public` only, add a sibling `GET /api/v1/public/marketplace/courses/{slug}/reviews` that reuses the same review repo.

## 6. Non-Functional Requirements

- **Performance** — List p95 < 300 ms warm; reuse the catalog object cache (the response is viewer-independent, so it caches cleanly — unlike the in-app path which had to resolve per-user ownership post-cache). Pagination bounds payload.
- **Security** — Read-only, anonymous, no PII (only aggregate `enrollmentCount`, `averageRating`, `ratingCount`, and instructor display name). No price or listing state is writable here. Rate-limit under the standard public read tier; endpoints are safe to sit behind a CDN.
- **Privacy & Compliance** — Same non-PII surface as the public catalog (15.1). Instructor name is an already-public display field.
- **Accessibility** — N/A (API). Consumed by accessible UIs in MKT8/MKT9.
- **Scalability** — Partial index `idx_courses_marketplace` (MKT1) keeps the `marketplace_listed` scan cheap; CDN + object cache absorb read load.
- **Reliability** — Cache miss falls back to DB; endpoint degrades to `500 internal` on query failure (never partial/incorrect listings). Idempotent GETs.
- **Observability** — Emit a `public_marketplace_list_total` counter and reuse the catalog search metric; log flag-off `404`s at debug. Feed MKT8 funnel.
- **Maintainability** — Net-new file `public_marketplace_http.go`; **zero** duplication of the storefront SQL (call the shared repo funcs). Keep the handler a thin adapter of the authenticated one minus auth/ownership.
- **Internationalization** — `language` filter reuses `catalog_language`; currency/price returned as integer cents + currency code (formatting is the client's job).
- **Backward compatibility** — Purely additive routes. No change to existing catalog, marketplace, or billing endpoints.

## 7. Acceptance Criteria

- **AC-1.** *Given* `FFCourseMarketplace` is on and two published courses are marketplace-listed (one Free, one $20), *When* an **anonymous** client calls `GET /api/v1/public/marketplace/courses`, *Then* both are returned with `priceCents` `0` and `2000`, and no `owned` field is present.
- **AC-2.** *Given* a listed course is **unpublished** (draft), *When* the list is fetched, *Then* it is absent.
- **AC-3.** *Given* a course is `is_public = TRUE` but **not** `marketplace_listed`, *When* the list is fetched, *Then* it is absent (proving this is the marketplace set, not the catalog set).
- **AC-4.** *Given* `free_only=true` and `category=X`, *When* the list is fetched, *Then* only free, listed, published, category-X courses are returned.
- **AC-5.** *Given* a listed+published course with slug `s`, *When* `GET /api/v1/public/marketplace/courses/s` is called, *Then* `{ course, whatsIncluded }` is returned; *When* the slug is a non-listed course, *Then* `404`.
- **AC-6.** *Given* `FFCourseMarketplace` is off, *When* any `/api/v1/public/marketplace/*` route is called, *Then* `404 not_found`.
- **AC-7.** *Given* a cross-origin request from `https://lextures.com`, *When* any public marketplace route is called, *Then* the response carries `Access-Control-Allow-Origin: *` and succeeds.
- **AC-8.** *Given* an invalid `sort`/`level`/`cursor`/`price_max`, *When* the list is fetched, *Then* `400 invalid_input`.

## 8. Data Model

No schema changes. Reads `course.courses` (`marketplace_listed`, `published`, and catalog/pricing columns from MKT1/MKT3). Reuses existing repo functions in `server/internal/repos/course/marketplace_storefront.go`:

- `ListMarketplaceCourses(ctx, pool, filter)` → `[]MarketplaceCourse, total, nextCursor`
- `GetMarketplaceCourseBySlug(ctx, pool, slug)` → `*MarketplaceCourse`
- `ListMarketplaceCategories(ctx, pool)` → `[]CatalogCategory`
- `GetMarketplaceWhatsIncluded(ctx, pool, courseID)` → `MarketplaceWhatsIncluded`
- `MarketplaceFilter.ToPublicCatalogFilter()` for param mapping.

No new indexes (reuse `idx_courses_marketplace` from MKT1). No backfill.

## 9. API Surface

New unauthenticated routes (gated by `publicMarketplaceOff` → `FFCourseMarketplace`):

- `GET /api/v1/public/marketplace/courses` — params `q, category, level, language, price_max, free_only, sort, cursor, limit`. Response: `{ courses: PublicMarketplaceCourse[], total, nextCursor }`.
- `GET /api/v1/public/marketplace/categories` — `{ categories: CatalogCategory[] }`.
- `GET /api/v1/public/marketplace/courses/{slug}` — `{ course: PublicMarketplaceCourse, whatsIncluded }`; `404` if not listed/published.
- **Reviews (reused):** `GET /api/v1/public/catalog/courses/{slug}/reviews` — confirm it resolves for marketplace-listed slugs; else add `/api/v1/public/marketplace/courses/{slug}/reviews`.

`PublicMarketplaceCourse` is the existing `MarketplaceCourse` JSON **minus** `owned`:

```ts
type PublicMarketplaceCourse = {
  id: string; slug: string; courseCode: string; title: string; description: string
  heroImageUrl: string | null; category: string | null; difficultyLevel: string | null
  language: string; priceCents: number; priceCurrency: string
  listPriceCents: number | null
  enrollmentCount: number; averageRating: number | null; ratingCount: number
  instructorName: string | null; createdAt: string
}
```

- Rate-limit: standard public read tier; safe behind CDN.
- OpenAPI: document all three (+ reviews) in `server/internal/openapi/openapi.go` under a `public-marketplace` tag.

## 10. UI / UX

None — this is an API story. The consuming UIs are MKT8 (storefront) and MKT9 (detail). The response shape here is the contract MKT8/MKT9 build against; keep field names identical to the in-app `MarketplaceCourse` (sans `owned`) so a single shared TS type can serve both.

## 11. AI / ML Considerations

Not AI-touching. (Ranking reuses `catalogsearch` popularity/rating/price sort.)

## 12. Integration Points

- **Internal (new)** — `server/internal/httpserver/public_marketplace_http.go`, wired in `server.go`'s route registration next to `registerPublicCatalogRoutes`.
- **Internal (reused)** — `repos/course/marketplace_storefront.go`, `httpserver/course_marketplace.go` (`FFCourseMarketplace` gate pattern), `httpserver/public_catalog_http.go` (`publicCatalogCacheHeaders`, `writeJSONWithETag`, cursor/param validation), `httpserver/cors.go` (`corsAll`), `internal/objectcache`.
- **Consumers** — www `marketplace-api.ts` client (MKT8/MKT9).

## 13. Dependencies & Sequencing

- **After** — MKT1 (flag + column + index), MKT3 (storefront repo query it reuses).
- **Before** — MKT8 (storefront UI), MKT9 (detail UI), MKT10 (SEO/JSON-LD).
- **Shared infra** — object cache, CDN cache headers, existing catalog search.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Public reviews endpoint scoped to `is_public` only, so marketplace-only slugs 404 | M | M | Verify in test; add `/public/marketplace/.../reviews` sibling if needed (FR-10) |
| Confusion between `is_public` catalog set and `marketplace_listed` set | M | L | Doc + tests asserting the sets differ (AC-3); name routes `/public/marketplace/*` distinctly |
| Anonymous endpoint scraped at scale | M | L | CDN + object cache + standard public rate limit; no PII exposed |
| Divergence from in-app `MarketplaceCourse` shape over time | L | M | Share the Go struct; contract test comparing fields |

## 15. Rollout Plan

- **Flag** — `FFCourseMarketplace` (MKT1), default ON. Routes appear/disappear with it.
- **Sequencing** — ship server-only first; www (MKT8) can develop against it immediately in a dev/staging origin via `VITE_API_BASE_URL`.
- **Dogfood** — hit the endpoints from a staging www build; verify CORS + cache headers.
- **GA criteria** — list/detail/categories correct, flag-off 404, CORS verified, OpenAPI published.
- **Rollback** — flag off (hides routes) or revert the additive handler; no data migration to unwind.

## 16. Test Plan

- **Unit** — param validation mapping (`MarketplaceFilter.ToPublicCatalogFilter`); flag-off `publicMarketplaceOff` returns 404.
- **Integration (nodb where possible, db where needed)** — listed+published only; draft excluded; `is_public`-only excluded (AC-3); `free_only`/category filters; detail 404 for non-listed slug; categories over listed set; cache headers + ETag present; **no `owned` field** in JSON.
- **CORS** — assert `Access-Control-Allow-Origin: *` on a `/public/marketplace/courses` response (mirror any existing CORS test).
- **Contract** — snapshot that `PublicMarketplaceCourse` fields == `MarketplaceCourse` minus `owned`.
- **Security** — endpoint reachable without a session cookie/token; no set-cookie; no cross-user data.
- **Performance** — warm-cache p95; index used (EXPLAIN) on `marketplace_listed`.

## 17. Documentation & Training

- **API reference** — new `public-marketplace` section in OpenAPI + published docs.
- **Internal runbook** — note the `is_public` (catalog) vs `marketplace_listed` (marketplace) distinction and which flag gates which.
- **Cross-link** — MKT8/MKT9 reference this contract.

## 18. Open Questions

1. Does the existing `/api/v1/public/catalog/courses/{slug}/reviews` resolve by `catalog_slug` regardless of `is_public`, or is it catalog-scoped? (Determines whether FR-10 needs a new sibling route. **Default:** verify first; add sibling only if needed.)
2. Should the public detail response also embed `jsonLd` (like the public catalog detail does) here, or should MKT10 build JSON-LD client-side? (**Default:** embed `jsonLd` here by reusing `catalogsearch.BuildCourseJSONLD`, so MKT10 just injects it — cheaper and consistent.)
3. Should the public marketplace also be gated on `FFPublicCatalog` in addition to `FFCourseMarketplace` (belt-and-suspenders for "any public browsing")? (**Default:** gate on `FFCourseMarketplace` only, matching the in-app marketplace.)
4. Include `listPriceCents` (strikethrough) in the public shape now or later? (**Default:** include it now — it already exists on `MarketplaceCourse`.)

## 19. References

- Existing files: `server/internal/httpserver/marketplace_courses_http.go` (the authenticated analogue to mirror), `httpserver/public_catalog_http.go` (public-endpoint pattern + cache/ETag helpers), `httpserver/course_marketplace.go` (`courseMarketplaceOff`), `httpserver/cors.go`, `repos/course/marketplace_storefront.go`, `service/catalogsearch/*`, `internal/openapi/openapi.go`.
- Related plans: [MKT1](../../completed/marketplace/MKT1-marketplace-platform-foundation.md), [MKT3](../../completed/marketplace/MKT3-marketplace-discovery-web.md), [MKT8](MKT8-www-courses-storefront.md), [MKT9](MKT9-www-course-detail-enroll.md), [MKT10](MKT10-www-marketplace-seo.md), `docs/completed/15-self-learner-specific/15.1-public-course-catalog.md`.
