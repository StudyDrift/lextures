# MKT8 — Courses Tab & Public Storefront (lextures.com / www)

> Implementation plan. Source: [docs/plan/marketplace/README.md](README.md). Part of the Marketplace epic — **www storefront sub-epic (MKT7–MKT10).**

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKT8 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web / marketing-site team |
| **Depends on** | MKT7 (public marketplace API) |
| **Unblocks** | MKT9 (detail), MKT10 (SEO) |

---

## 1. Problem Statement

The marketing site `lextures.com` (the `www/` Vite app) has no way to browse courses. Prospective learners land on audience pages and pricing but must sign up at `self.lextures.com` before they can see a single course — a conversion killer compared to Coursera, Udemy, or Fiverr, where a public, browsable course catalog **is** the front door. This story adds a **Courses** tab to the site header and a modern, responsive public **storefront** page (`/courses`) that fetches marketplace-listed courses from the MKT7 public API and presents them in a searchable, filterable, category-organized grid — building trust and driving sign-ups before login.

## 2. Goals

- Add a **Courses** item to the www header nav (desktop + mobile) linking to `/courses`.
- Build a modern storefront page: hero + search, category rails/chips, filters (level, price, language), sort, and a responsive course-card grid.
- Fetch data from the MKT7 public API (`/api/v1/public/marketplace/courses` + `/categories`) via `API_BASE` (already `self.lextures.com`), with a shared, typed client.
- Match the visual quality of top course marketplaces (clean cards, hero image, price/Free badge, rating, enrollment count) while staying on-brand with the existing www design system (CSS-variable theming, Tailwind v4).
- Handle loading, empty, and error states gracefully; be fully responsive and WCAG 2.1 AA accessible.

## 3. Non-Goals

- The public course **detail** page and Enroll handoff (MKT9) — cards route to `/courses/:slug`, which MKT9 implements.
- SEO/prerendering/JSON-LD/sitemap (MKT10).
- The server API itself (MKT7).
- Any authenticated action, purchase, or enrollment on www (all commerce stays on `self.lextures.com`).
- Personalized recommendations / "continue learning" (no viewer identity on www).

## 4. Personas & User Stories

- **As an anonymous visitor**, I want to browse available courses on the marketing site and filter by topic/level/price, so that I can evaluate Lextures before creating an account.
- **As a self-learner**, I want free courses clearly badged "Free" and paid ones showing a price, so that I know what I'm getting into.
- **As a mobile visitor**, I want the catalog to be fast and thumb-friendly, so that I can browse on my phone.
- **As an instructor**, I want my listed course to show up on the public site with an accurate, attractive card, so that strangers can discover it.

## 5. Functional Requirements

- **FR-1.** The system MUST add `{ label: 'Courses', href: '/courses' }` to `NAV_LINKS` in `www/src/components/header.tsx` (rendered in both the desktop nav and the mobile menu).
- **FR-2.** The system MUST add a `/courses` route branch in `www/src/app.tsx` (`resolveRoute` / `App`) rendering a new `CoursesPage`, wrapped in `MarketingPageShell`.
- **FR-3.** `CoursesPage` MUST fetch `GET {API_BASE}/api/v1/public/marketplace/courses` (params: `q, category, level, language, price_max, free_only, sort, cursor, limit`) through a new typed client `www/src/lib/marketplace-api.ts`, using `API_BASE` from `lib/api-base.ts`.
- **FR-4.** Each course card MUST show: hero image (with a branded placeholder when `heroImageUrl` is null), title, category + level chips, instructor name, enrollment count, star rating + `ratingCount` (when `averageRating` present), and a price badge — **"Free"** when `priceCents === 0`, else a localized price (with `listPriceCents` strikethrough when present).
- **FR-5.** The page MUST provide: a search input (`q`, debounced), category chips/rail (from `/categories`), and filter controls for level, language, and price (Free-only toggle + optional max price), plus a sort control (popular / newest / price). Changing any control MUST re-query and MUST reflect state in the URL (query string) so results are shareable/back-button-safe.
- **FR-6.** Each card MUST be a single focusable link routing to `/courses/{slug}` (MKT9). Cards MUST fall back to `courseCode` if `slug` is missing (defensive; slug should always exist for listed courses).
- **FR-7.** The grid MUST paginate using the API `nextCursor` (either "Load more" or infinite scroll; **default:** "Load more" button for simplicity + a11y), and MUST show a result count from `total`.
- **FR-8.** The page MUST render distinct **loading** (skeleton grid), **empty** ("No courses available yet"), and **error** (message + Retry) states; a fetch failure MUST NOT blank the page.
- **FR-9.** When the marketplace is disabled server-side, the API returns `404`; the page MUST treat that as a graceful "Courses are not available right now" empty state (and, per MKT10, the nav item MAY be hidden — but www is static, so default is: keep the tab, show the empty state).
- **FR-10.** All prices MUST be formatted with `Intl.NumberFormat` using `priceCurrency`; the client MUST NOT invent or compute prices.

## 6. Non-Functional Requirements

- **Performance** — First meaningful paint of the grid < 1.5 s on a warm CDN; API responses are cached server-side (MKT7). Lazy-load hero images (`loading="lazy"`), constrain image dimensions, and debounce search (~250 ms). Bundle impact minimized — no new heavy deps; reuse `lucide-react` for icons.
- **Security** — Read-only anonymous fetches; no secrets; no tokens. Escape/normalize any course text before rendering (React handles this; avoid `dangerouslySetInnerHTML`).
- **Privacy & Compliance** — Only non-PII aggregate fields shown (enrollment count, rating, instructor display name), consistent with MKT7.
- **Accessibility** — WCAG 2.1 AA: grid is a semantic list; each card is one link with an accessible name including title + price/Free; filters are labelled controls with visible focus; category chips are a labelled group; rating conveyed with text (not color/stars alone); the search has an associated `<label>`; skip-link and heading order preserved.
- **Scalability** — Client paginates; server + CDN absorb load. No client-side full-catalog fetch.
- **Reliability** — Fetch has timeout + retry affordance; partial/failed category fetch degrades to "All" only.
- **Observability** — Fire lightweight client analytics events (reusing whatever the www site already uses, if any) for `courses_view`, `courses_search`, `courses_filter`, `course_card_click` — feeding the MKT9 detail funnel. (If www has no analytics yet, note as Open Question.)
- **Maintainability** — Follow existing www conventions: functional components, CSS variables (`--paper`, `--panel`, `--ink-nav`, `--line-card`, `--coral`, etc.), Tailwind utility classes, no CSS-in-JS libraries. New components under `www/src/components/courses/`.
- **Internationalization** — Prices via `Intl.NumberFormat`; language filter uses `language` codes; copy centralized so it can be localized later (www is currently English-only — keep strings in one module).
- **Backward compatibility** — Additive route + nav item; no change to existing www pages.

## 7. Acceptance Criteria

- **AC-1.** *Given* the site loads, *When* I look at the header (desktop and mobile), *Then* a "Courses" link appears and routes to `/courses`.
- **AC-2.** *Given* the API returns a Free course and a $20 course, *When* the storefront loads, *Then* both cards render with correct "Free" and "$20.00" badges, hero/placeholder, rating, and enrollment count.
- **AC-3.** *Given* I type in search, *When* I pause, *Then* the grid re-queries with `q` and the URL updates; a shared URL reproduces the same filtered view.
- **AC-4.** *Given* I toggle "Free only" and pick category X, *When* results load, *Then* only free, category-X courses appear and the active filters are visibly indicated.
- **AC-5.** *Given* more results exist, *When* I click "Load more", *Then* the next page appends using `nextCursor` without a full reload.
- **AC-6.** *Given* the network fails, *When* the grid loads, *Then* an error state with a working Retry appears (no blank page); *Given* zero results, *Then* the empty state appears.
- **AC-7.** *Given* I click a card, *When* it activates, *Then* I navigate to `/courses/{slug}` (MKT9).
- **AC-8.** *Given* a screen reader / keyboard, *When* I traverse the page, *Then* filters are labelled, cards are reachable links with price in the accessible name, and focus order is search → filters → grid.
- **AC-9.** *Given* a viewport at 375px / 768px / 1200px, *When* the grid renders, *Then* it shows 1 / 2 / 3(+) columns respectively with no horizontal overflow.

## 8. Data Model

No data model. Consumes MKT7 JSON only. Defines TypeScript types in `www/src/lib/marketplace-api.ts` mirroring `PublicMarketplaceCourse` and the categories response.

## 9. API Surface

Consumes (no new server surface):

- `GET {API_BASE}/api/v1/public/marketplace/courses?q&category&level&language&price_max&free_only&sort&cursor&limit` → `{ courses, total, nextCursor }`.
- `GET {API_BASE}/api/v1/public/marketplace/categories` → `{ categories }`.

`API_BASE` resolves to `https://self.lextures.com` by default (`www/src/lib/api-base.ts`), overridable via `VITE_API_BASE_URL` for dev/staging. CORS is already `*` (MKT7 FR-9).

## 10. UI / UX

- **New nav** — "Courses" tab in `header.tsx` (desktop `NAV_LINKS` + mobile menu list).
- **New page** — `www/src/pages/courses-page.tsx`, routed in `app.tsx`, wrapped in `MarketingPageShell`.
- **New components** (under `www/src/components/courses/`):
  - `CourseCard` — hero/placeholder, title, category+level chips, instructor, rating, enrollment, price/Free badge; whole-card link.
  - `CourseFilters` — search, category rail/chips, level/language/price controls, sort; collapses into a filter sheet on mobile.
  - `CourseGrid` — responsive list with skeleton, empty, and error states + "Load more".
  - `PriceBadge`, `RatingStars` — small presentational primitives.
- **Storefront layout** — hero band (headline + search) → category chips → active-filter summary → results grid → load-more. Modeled on Coursera/Udemy density but using the www palette (`--panel` cards, `--line-card` borders, coral/teal accents, `font-display` headings) — reuse the card elevation/hover pattern already in `marketing-page-shell.tsx` (`CardGrid`).
- **Flows** — (1) Header → Courses → grid. (2) Search/filter/sort → grid updates + URL. (3) Load more → append. (4) Click card → `/courses/:slug` (MKT9).
- **States** — skeleton grid (loading); "No courses available yet" (empty); error + Retry; per-card placeholder image when `heroImageUrl` null.
- **Responsive** — 1/2/3-column grid by breakpoint; filters become a bottom sheet / disclosure on small screens; sticky search on scroll (optional).
- **Accessibility** — semantic list, labelled controls, discernible link names with price, visible focus, rating as text, respects `prefers-reduced-motion` for hover/animation.
- **Copy** — centralize in a `courses` copy module: title, search placeholder, "Free", "Load more", filter labels, empty/error text.

## 11. AI / ML Considerations

Not AI-touching. (Sorting reuses server popularity/rating/price.)

## 12. Integration Points

- **www internal (new)** — `pages/courses-page.tsx`, `components/courses/*`, `lib/marketplace-api.ts`; edits to `components/header.tsx` (`NAV_LINKS`) and `app.tsx` (route).
- **www internal (reused)** — `lib/api-base.ts` (`API_BASE`), `lib/site-links.ts` (`APP_ORIGIN` for MKT9 handoff), `components/marketing-page-shell.tsx`, existing CSS variables + `btn-*` classes.
- **Server** — MKT7 public marketplace endpoints (cross-origin, CORS `*`).
- **Handoff** — cards → MKT9 detail route.

## 13. Dependencies & Sequencing

- **After** — MKT7 (the API it fetches).
- **Before** — MKT9 (detail — the card destination), MKT10 (SEO layered on these pages).
- **Shared infra** — none new on www; server CDN/object cache from MKT7.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Cross-origin fetch blocked | L | H | MKT7 verifies `corsAll` covers the routes; www uses plain `fetch`, no credentials |
| Empty catalog at launch looks broken | M | M | Purposeful empty state; seed first-party free courses before GA |
| Missing hero images look bare | M | L | Branded gradient/initial placeholder component |
| Deep links to `/courses` / `/courses/:slug` don't resolve on the static host | L | M | www is on **GitHub Pages** using the `public/404.html` → `/?/…` SPA-redirect trick (already used for `/docs`, `/blog`); course routes inherit it. MKT10 additionally prerenders real files for listed slugs so crawlers/direct hits skip the 404 bounce |
| Design drift from marketing brand | M | L | Reuse existing tokens/components; design review against `marketing-page-shell` |
| API shape changes vs in-app marketplace | L | M | Single shared type mirroring MKT7 contract; contract noted in MKT7 §16 |

## 15. Rollout Plan

- **Flag** — no www build flag; server `FFCourseMarketplace` governs data (404 → empty state). Optionally hide the nav tab behind a build-time env until GA.
- **Sequencing** — after MKT7; ship the grid pointing at staging via `VITE_API_BASE_URL`, then production.
- **Dogfood** — internal browse of seeded listings across breakpoints.
- **GA criteria** — browse/search/filter/sort/pagination work; states correct; a11y pass; responsive at 3 breakpoints; nav tab live.
- **Rollback** — remove the nav item + route (revert), or point `VITE_API_BASE_URL` away; server data untouched.

## 16. Test Plan

- **Unit** — price/Free formatting; filter→query-param mapping; URL state round-trip; card accessible-name composition; rating-stars text.
- **Component** — `CourseGrid` loading/empty/error rendering; "Load more" appends; `CourseFilters` updates state.
- **End-to-end (whatever www uses; else Playwright)** — header→/courses→search→filter→load-more→card click→/courses/:slug; error/empty states via mocked API; back-button restores filtered view.
- **Accessibility** — axe on `/courses`; keyboard-only traversal; screen-reader reads price/Free and filters; contrast in the site's theme.
- **Responsive** — snapshot/visual at 375/768/1200; no horizontal scroll.
- **Cross-origin** — integration test against a running MKT7 endpoint (or mock) confirming fetch + CORS.

## 17. Documentation & Training

- **Marketing/site docs** — note the new Courses tab and how listings are sourced (`marketplace_listed` on `self.lextures.com`).
- **Instructor docs** — cross-link from MKT2 ("your listed course also appears publicly on lextures.com/courses").
- **Runbook** — how to point www at a different API origin (`VITE_API_BASE_URL`).

## 18. Open Questions

1. ~~Does the www static host rewrite arbitrary paths to `index.html`?~~ **Resolved:** www is on GitHub Pages using the `public/404.html` → `/?/…` redirect trick (`www/index.html` has the restore script; `vite.config.ts` notes root-relative assets for `/docs`/`/blog` deep links). `/courses*` inherits this; MKT10 prerenders listed slugs on top.
2. ~~Does www have client analytics?~~ **Resolved:** Google Analytics `gtag` (`G-JX182Q6KKX`) is already loaded in `www/index.html`. Emit funnel events via `window.gtag('event', …)`. (Scope which events for v1.)
3. "Load more" vs infinite scroll? (**Default:** "Load more" for a11y + simplicity.)
4. Should the storefront surface curated "Featured"/"Popular" rails (Coursera-style), or a single filterable grid for v1? (**Default:** single grid + category chips in v1; rails as a fast-follow.)
5. Which lucide icon / eyebrow for the Courses hero? (Proposed: `GraduationCap` / `BookOpen`.)

## 19. References

- Existing files: `www/src/components/header.tsx` (`NAV_LINKS`), `www/src/app.tsx` (`resolveRoute`, `App`), `www/src/lib/api-base.ts`, `www/src/lib/site-links.ts`, `www/src/components/marketing-page-shell.tsx` (shell + `CardGrid` pattern), `www/src/pages/self-learner-page.tsx` (page pattern). For visual reference of the in-app equivalent: `clients/web/src/pages/explore-catalog-page.tsx`.
- Related plans: [MKT7](MKT7-public-marketplace-api.md), [MKT9](MKT9-www-course-detail-enroll.md), [MKT10](MKT10-www-marketplace-seo.md), [MKT3](../../completed/marketplace/MKT3-marketplace-discovery-web.md).
