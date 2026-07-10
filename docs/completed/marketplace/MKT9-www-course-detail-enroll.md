# MKT9 — Public Course Detail Page & Enroll Handoff (www)

> Implementation plan. Source: [docs/plan/marketplace/README.md](README.md). Part of the Marketplace epic — **www storefront sub-epic (MKT7–MKT10).**

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKT9 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web / marketing-site team |
| **Depends on** | MKT7 (public API), MKT8 (storefront + nav + client) |
| **Unblocks** | MKT10 (per-course SEO/JSON-LD) |

---

## 1. Problem Statement

Storefront cards (MKT8) need somewhere to go. A grid alone doesn't sell a course — Coursera/Udemy/Fiverr conversions happen on a rich **course detail page**: syllabus/what's-included, instructor, ratings and reviews, price, and a single prominent call-to-action. This story adds a branded public detail page at `lextures.com/courses/:slug` that renders the full course from the MKT7 public API, shows real reviews, and drives a clear **Enroll** action that hands off to `self.lextures.com` (where authentication + the shipped purchase/enroll flow live). It is the conversion surface that turns anonymous browsing into sign-ups.

## 2. Goals

- Build a public, branded course detail page at `/courses/:slug` in the www app.
- Render full course content from `GET /api/v1/public/marketplace/courses/{slug}` (description, what's-included, price, instructor, rating) and reviews from the public reviews endpoint.
- Provide a prominent, unambiguous primary CTA — **"Enroll — Free"** / **"Enroll — $X"** — that deep-links into `self.lextures.com` (`/explore/:slug`) to complete auth + enrollment/purchase.
- Handle not-found, loading, and error states cleanly; be fully responsive and WCAG 2.1 AA accessible.
- Keep the page share-ready (stable URL, hero, price) as the foundation MKT10 layers SEO/JSON-LD onto.

## 3. Non-Goals

- The purchase/checkout/enrollment transaction itself — it happens on `self.lextures.com` via the shipped MKT4 flow. www only routes there.
- The server API (MKT7) and storefront grid/nav (MKT8).
- SEO meta tags / JSON-LD / prerendering / sitemap (MKT10) — this story produces the page MKT10 optimizes.
- Writing reviews/ratings (read-only display; authoring lives in the app, plan 15.7).
- Any authenticated or personalized content on www.

## 4. Personas & User Stories

- **As an anonymous visitor**, I want a detailed page for a course — what I'll learn, who teaches it, what others rated it, and the price — so that I can decide before signing up.
- **As a ready-to-buy visitor**, I want one obvious Enroll button that takes me straight into the app to start, so that I don't get lost.
- **As a skeptical visitor**, I want to read real reviews and see the rating, so that I trust the course.
- **As someone who shares a link**, I want the URL to open directly to that course's page, so that friends land where I intended.

## 5. Functional Requirements

- **FR-1.** The system MUST add a `/courses/:slug` route branch in `www/src/app.tsx` rendering a new `CourseDetailPage` (wrapped in `MarketingPageShell`), parsing `slug` from the path.
- **FR-2.** `CourseDetailPage` MUST fetch `GET {API_BASE}/api/v1/public/marketplace/courses/{slug}` via the MKT8 client (`lib/marketplace-api.ts`, extended with `getCourse(slug)`), and MUST render a **404 / "Course not found"** state (with a link back to `/courses`) when the API returns `404`.
- **FR-3.** The page MUST display: hero (image or branded placeholder), title, category + level, language, instructor name, enrollment count, star rating + rating count, full description, and **what's-included** (module/item counts, estimated duration from `whatsIncluded`).
- **FR-4.** The page MUST display a **price block**: "Free" when `priceCents === 0`, else localized price via `Intl.NumberFormat(priceCurrency)`, with `listPriceCents` strikethrough when present.
- **FR-5.** The page MUST render a primary CTA whose label reflects price — **"Enroll — Free"** or **"Enroll — $X"** — linking to `{APP_ORIGIN}/explore/{slug}` (from `lib/site-links.ts`, `APP_ORIGIN = https://self.lextures.com`). The link MUST carry a UTM/source param (e.g. `?ref=www-courses`) for attribution.
- **FR-6.** The page MUST fetch and render **reviews** from `GET {API_BASE}/api/v1/public/catalog/courses/{slug}/reviews` (or the MKT7 marketplace sibling if added): list of rating + text + author display + date, with a graceful "No reviews yet" state and a bounded initial count (e.g. show 5, "Show more").
- **FR-7.** A secondary CTA MUST let a visitor go to the full catalog (`/courses`) and MAY surface a "View on Lextures" link to `{APP_ORIGIN}/explore/{slug}`.
- **FR-8.** The page MUST render distinct **loading** (skeleton), **error** (message + Retry), and **not-found** states. A reviews-fetch failure MUST degrade to hiding the reviews section, not blocking the page.
- **FR-9.** The Enroll CTA MUST be reachable/sticky on mobile (e.g. a sticky bottom bar with price + Enroll) so it's always actionable.
- **FR-10.** The page MUST be resilient to missing optional fields (null hero, null rating, empty description) without layout breakage.

## 6. Non-Functional Requirements

- **Performance** — Detail + reviews are two cached GETs; render skeleton immediately, hydrate on arrival. Lazy-load hero + review avatars. Target LCP < 2 s warm.
- **Security** — Read-only anonymous fetches. Course description/review text rendered as text (no raw HTML injection); if descriptions contain markdown, render via the existing `react-markdown` dep already in www with a safe config (no raw HTML).
- **Privacy & Compliance** — Reviews show only already-public author display names + aggregate rating; no PII beyond what the app already exposes publicly (15.1/15.7).
- **Accessibility** — WCAG 2.1 AA: one `<h1>` (course title), logical heading order for sections (About, What's included, Reviews), CTA is a discernible link with price in its accessible name, rating expressed as text, sticky CTA does not trap focus, review list is a semantic list, images have appropriate alt.
- **Scalability** — Static page + two cached API calls; CDN-friendly.
- **Reliability** — Independent fetches: detail failure → error state; reviews failure → section hidden; missing slug → 404 state.
- **Observability** — Emit `course_detail_view{slug}`, `enroll_cta_click{slug, price}` client events (if www analytics exists — see MKT8 OQ), feeding the sign-up funnel.
- **Maintainability** — Reuse MKT8 primitives (`PriceBadge`, `RatingStars`, placeholder) and the www design tokens; new components under `www/src/components/courses/`.
- **Internationalization** — Price via `Intl.NumberFormat`; dates localized; copy centralized in the `courses` copy module.
- **Backward compatibility** — Additive route/page; no change to existing www pages or the app's `/explore/:slug`.

## 7. Acceptance Criteria

- **AC-1.** *Given* a listed+published course with slug `s`, *When* I open `/courses/s`, *Then* I see its hero, title, description, what's-included, instructor, rating, and price.
- **AC-2.** *Given* a Free course, *When* I view its page, *Then* the CTA reads "Enroll — Free"; *Given* a $20 course, *Then* it reads "Enroll — $20.00".
- **AC-3.** *Given* I click Enroll, *When* it activates, *Then* I navigate to `https://self.lextures.com/explore/s?ref=www-courses`.
- **AC-4.** *Given* the course has reviews, *When* the page loads, *Then* the rating summary and a bounded review list render; *Given* none, *Then* "No reviews yet".
- **AC-5.** *Given* an unknown/unlisted slug, *When* I open `/courses/that-slug`, *Then* I see a "Course not found" state with a link back to `/courses`.
- **AC-6.** *Given* the detail fetch fails, *When* the page loads, *Then* an error + Retry appears; *Given* only the reviews fetch fails, *Then* the rest of the page renders and the reviews section is hidden.
- **AC-7.** *Given* a mobile viewport, *When* I scroll, *Then* a sticky price + Enroll bar remains actionable and keyboard-reachable.
- **AC-8.** *Given* a screen reader, *When* I traverse the page, *Then* headings are ordered, the Enroll link announces the price, and the rating is read as text.

## 8. Data Model

No data model. Consumes MKT7 detail JSON (`{ course, whatsIncluded }`) and the public reviews JSON. Extends the MKT8 TS client with `getCourse(slug)` and `getReviews(slug)` and a `Review` type.

## 9. API Surface

Consumes (no new server surface):

- `GET {API_BASE}/api/v1/public/marketplace/courses/{slug}` → `{ course: PublicMarketplaceCourse, whatsIncluded, jsonLd? }` (MKT7). `404` when not listed/published.
- `GET {API_BASE}/api/v1/public/catalog/courses/{slug}/reviews` → `{ reviews: Review[] }` (existing; MKT7 confirms marketplace-slug resolution / adds a sibling).
- Enroll handoff (navigation, not fetch): `{APP_ORIGIN}/explore/{slug}?ref=www-courses`.

## 10. UI / UX

- **New page** — `www/src/pages/course-detail-page.tsx`, routed in `app.tsx`, wrapped in `MarketingPageShell`.
- **Layout** (Udemy/Coursera-style two-column on desktop, stacked on mobile):
  - **Header band** — breadcrumb (`Courses / {title}`), title, category/level chips, rating summary, enrollment count, instructor.
  - **Main column** — hero image, About (description), What's included (module/item counts + duration), Reviews (summary + list + "Show more").
  - **Sticky purchase panel** (desktop right rail / mobile bottom bar) — price/Free, list-price strikethrough, primary **Enroll** CTA, secondary "View on Lextures".
- **New components** (under `www/src/components/courses/`): `CourseHero`, `WhatsIncluded`, `ReviewList` + `ReviewCard`, `EnrollPanel` (sticky), reusing MKT8 `PriceBadge` / `RatingStars` / placeholder.
- **Flows** — (1) Storefront card → detail. (2) Read description/reviews. (3) Enroll → app `/explore/:slug`. (4) Back to `/courses`.
- **States** — skeleton (loading); "Course not found" (404, with back link); error + Retry (detail); hidden reviews section (reviews error); placeholder hero (null image); "No reviews yet".
- **Responsive** — two-column ≥ lg; single column with sticky bottom Enroll bar on mobile.
- **Accessibility** — single h1, ordered sections, CTA accessible name with price, semantic review list, alt text, reduced-motion respected.
- **Copy** — extend the `courses` copy module: "Enroll — Free/{price}", "What's included", "Reviews", "No reviews yet", "Course not found", "View on Lextures".

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **www internal (new)** — `pages/course-detail-page.tsx`, `components/courses/{CourseHero,WhatsIncluded,ReviewList,EnrollPanel}.tsx`; route in `app.tsx`; `lib/marketplace-api.ts` extended (`getCourse`, `getReviews`).
- **www internal (reused)** — `lib/site-links.ts` (`APP_ORIGIN`), `lib/api-base.ts` (`API_BASE`), `lib/format-date.ts`, `react-markdown` (safe config), MKT8 primitives + design tokens.
- **Server** — MKT7 detail endpoint; existing public reviews endpoint.
- **Handoff** — Enroll → `self.lextures.com/explore/:slug` (`ExploreCoursePage` in `clients/web`, which owns the shipped enroll/purchase flow).

## 13. Dependencies & Sequencing

- **After** — MKT7 (detail API), MKT8 (client + nav + card destination).
- **Before** — MKT10 (SEO/JSON-LD attach to this page).
- **Shared infra** — none new on www.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Deep link `/courses/:slug` 404s on static host | L | M | GitHub Pages `public/404.html` SPA-redirect trick handles unknown paths (as for `/blog/:slug` today); MKT10 prerenders listed slugs so direct hits/crawlers skip the bounce; test before GA |
| Public reviews endpoint doesn't resolve marketplace-only slugs | M | M | MKT7 FR-10 verifies/patches; www degrades to hidden reviews if empty |
| Description contains unsafe HTML | L | M | Render as text or `react-markdown` with raw HTML disabled |
| Enroll handoff lands on a course the visitor can't access post-login | L | M | `/explore/:slug` already handles auth + enroll; verify the target course code maps by slug |
| CTA not prominent enough on mobile | M | M | Sticky bottom price+Enroll bar (FR-9) |
| Slug mismatch between www and app | L | M | Both use `catalog_slug`; fallback link to `/explore` by course code if needed |

## 15. Rollout Plan

- **Flag** — server `FFCourseMarketplace` governs data (detail 404 → not-found state). No separate www flag beyond MKT8's optional nav gate.
- **Sequencing** — after MKT8; ship against staging (`VITE_API_BASE_URL`) then production.
- **Dogfood** — open detail pages for seeded free/paid courses; click through Enroll to the app end-to-end.
- **GA criteria** — detail + reviews render, price + Enroll correct, handoff lands on `/explore/:slug`, 404/error/empty states work, a11y + responsive pass.
- **Rollback** — remove route/page (revert); storefront cards fall back to linking `{APP_ORIGIN}/explore/:slug` directly.

## 16. Test Plan

- **Unit** — CTA label/price composition; handoff URL builder (`ref` param); markdown/text rendering safety; review date formatting.
- **Component** — loading/error/not-found/empty-reviews states; sticky EnrollPanel; missing-field resilience.
- **End-to-end** — card → detail → read reviews → Enroll → assert navigation to `self.lextures.com/explore/:slug?ref=www-courses`; unknown slug → not-found; reviews-only failure → page still renders.
- **Accessibility** — axe on detail; keyboard to Enroll; screen-reader reads price/rating; heading order; sticky bar focus.
- **Responsive** — two-column ≥ lg, stacked + sticky bar on mobile; no overflow.
- **Security** — no raw HTML execution from description/reviews.

## 17. Documentation & Training

- **Site docs** — the detail page and its handoff to the app.
- **Instructor docs** — cross-link: "your public course page on lextures.com/courses/:slug".
- **Runbook** — attribution `ref=www-courses` and where it surfaces in analytics.

## 18. Open Questions

1. Should Enroll deep-link to `/explore/:slug` (course landing) or straight to `/get-started` sign-up with a return-to param? (**Chosen:** `/explore/:slug`, per planning decision; it already carries the enroll/purchase flow. Confirm it accepts anonymous visitors and prompts auth.)
2. Is the course `description` plain text or markdown? (Drives text vs `react-markdown`. **Default:** treat as markdown with raw HTML disabled — safe for both.)
3. Does the public detail response include `whatsIncluded` counts + duration, or must www compute/omit? (MKT7 §9 includes `whatsIncluded`; **confirm** duration is populated.)
4. Do we want a "More courses like this" rail on the detail page for v1? (**Default:** no; fast-follow using category filter.)
5. Which analytics events on www? **Tool resolved:** Google Analytics `gtag` (`G-JX182Q6KKX`) is already present — emit `course_detail_view` / `enroll_cta_click` via `window.gtag('event', …)`. (Scope the exact event set with MKT8.)

## 19. References

- Existing files: `www/src/app.tsx` (routing incl. `route.startsWith('/blog/')` param pattern), `www/src/lib/site-links.ts` (`APP_ORIGIN`, `SITE_LINKS`), `www/src/lib/api-base.ts`, `www/src/lib/format-date.ts`, `www/src/pages/blog-post.tsx` (slug-page pattern), `www/src/components/marketing-page-shell.tsx`. App target: `clients/web/src/pages/explore-course-page.tsx` (`/explore/:slug`, owns enroll flow).
- Related plans: [MKT7](MKT7-public-marketplace-api.md), [MKT8](MKT8-www-courses-storefront.md), [MKT10](MKT10-www-marketplace-seo.md), [MKT4](../../completed/marketplace/MKT4-course-purchase-entitlement-flow.md), `docs/completed/15-self-learner-specific/15.1-public-course-catalog.md`.
