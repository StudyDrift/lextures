# MKT10 — www Marketplace SEO & Discoverability (prerender, meta, JSON-LD, sitemap)

> Implementation plan. Source: [docs/plan/marketplace/README.md](README.md). Part of the Marketplace epic — **www storefront sub-epic (MKT7–MKT10).**

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKT10 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web / marketing-site team |
| **Depends on** | MKT7 (public API + `jsonLd`), MKT8 (storefront), MKT9 (detail page) |
| **Unblocks** | — (growth/SEO enabler; closes the www sub-epic) |

---

## 1. Problem Statement

The entire value of a Coursera/Udemy/Fiverr-style course marketplace is **organic discoverability** — course pages ranking in search and rendering rich previews when shared. But `lextures.com` is a **client-rendered SPA hosted on GitHub Pages** with a **single static `<head>`** in `index.html` and no per-route metadata, structured data, or sitemap. Deep links to `/courses/:slug` currently rely on the `404.html` → `/?/…` redirect trick, which serves crawlers an empty shell before JS runs. As written, MKT8/MKT9 course pages would be effectively **invisible to search engines** and produce **generic link previews**. This story makes each course page crawlable and shareable: build-time **prerendering** of real HTML per course, per-page **meta/OG/Twitter** tags, **Course JSON-LD**, a generated **sitemap.xml**, and **robots.txt** — turning the storefront into a growth channel.

## 2. Goals

- Prerender static HTML for `/courses` and each `/courses/:slug` at build time, with correct per-page `<title>`, description, canonical, and Open Graph / Twitter tags — so crawlers and social scrapers see real content without executing JS.
- Emit **Course** JSON-LD (schema.org) on each course page, reusing the server's `BuildCourseJSONLD` (embedded in the MKT7 detail response).
- Generate `sitemap.xml` (storefront + all course URLs) and `robots.txt` at build, and reference the sitemap.
- Add runtime per-route `<head>` updates for client-side navigation (title/description/canonical/JSON-LD swap) so SPA transitions stay correct.
- Preserve GitHub Pages hosting; keep the build a single `npm run build` with a prerender step, no server runtime.

## 3. Non-Goals

- Server-side rendering at request time / migrating off GitHub Pages to an SSR host (prerender-at-build only; note as an Open Question if real-time freshness becomes a requirement).
- The API (MKT7), storefront UI (MKT8), or detail UI (MKT9) — this story adds a metadata/build layer over them.
- SEO for non-course www pages (home, pricing, docs) beyond what already exists — scope is the marketplace routes.
- Paid acquisition / ad landing pages.
- Internationalized (`hreflang`) course pages — www is English-only today.

## 4. Personas & User Stories

- **As a prospective learner searching Google**, I want a course's page to appear with an accurate title, description, and rating snippet, so that I click through from search.
- **As someone sharing a course link** in Slack/X/LinkedIn, I want a rich preview with the course hero, title, and price, so that the link is compelling.
- **As a search crawler**, I want a sitemap and pre-rendered HTML with structured data, so that I can index every course efficiently.
- **As the growth owner**, I want course pages indexed within days of listing, so that the marketplace compounds organic traffic.

## 5. Functional Requirements

- **FR-1.** The build MUST run a prerender step (after `vite build`) that, for `/courses` and every marketplace course from `GET {API_BASE}/api/v1/public/marketplace/courses` (paginating fully), writes a static `dist/courses/index.html` and `dist/courses/<slug>/index.html` derived from the built `index.html` shell.
- **FR-2.** Each prerendered course page MUST inject a per-course `<title>`, `<meta name="description">` (truncated course description), `<link rel="canonical" href="https://lextures.com/courses/<slug>">`, and Open Graph (`og:title`, `og:description`, `og:image`=`heroImageUrl` or a branded default, `og:type=website`, `og:url`) + Twitter (`summary_large_image`) tags — replacing the generic homepage defaults for that file only.
- **FR-3.** Each prerendered course page MUST embed a `<script type="application/ld+json">` **Course** document. It MUST reuse the server-provided `jsonLd` from the MKT7 detail response (`GET /api/v1/public/marketplace/courses/<slug>`) rather than re-implementing schema.org shaping.
- **FR-4.** The build MUST generate `dist/sitemap.xml` listing `https://lextures.com/courses` and every `https://lextures.com/courses/<slug>` (with `lastmod` where available), and MUST include the other primary static routes already present (home, pricing, docs, blog) so the sitemap is complete.
- **FR-5.** The build MUST generate/serve `robots.txt` allowing crawl of `/courses*` and referencing `Sitemap: https://lextures.com/sitemap.xml`.
- **FR-6.** At runtime, `CoursesPage` (MKT8) and `CourseDetailPage` (MKT9) MUST update `document.title`, the meta description, canonical, and (detail only) the JSON-LD `<script>` on client-side navigation, via a small shared `useDocumentHead` hook (no new heavy dependency; react-helmet optional).
- **FR-7.** The prerender step MUST be resilient: if the API is unreachable at build time, it MUST fail the build loudly (so we never deploy stale/empty course pages silently) OR fall back to prerendering only `/courses` — **default:** fail loudly, with a documented `SKIP_COURSE_PRERENDER=1` escape hatch.
- **FR-8.** Prerendered pages MUST hydrate cleanly: the injected `<head>` and JSON-LD MUST not conflict with the runtime `useDocumentHead` updates (idempotent — same values on hydrate).
- **FR-9.** The `404.html` SPA-redirect flow MUST remain intact for unknown paths, but known prerendered course URLs MUST resolve to their real static file (no 404 bounce for indexed courses).
- **FR-10.** The sitemap + prerender MUST reflect **only** listed+published courses (whatever MKT7 returns); unlisted/unpublished courses MUST NOT appear in the sitemap or get a prerendered page.

## 6. Non-Functional Requirements

- **Performance** — Prerendered HTML gives fast first paint + immediate crawlable content; runtime hydration unchanged. Prerender step bounded by course count (paginate; cap concurrency). Build-time cost scales linearly with catalog size.
- **Security** — Build-time fetch of a public endpoint only. Escape all injected text (title/description/OG) to prevent HTML injection via course fields. JSON-LD is server-generated + JSON-encoded.
- **Privacy & Compliance** — Only public, non-PII course fields in meta/JSON-LD (title, description, rating, price), consistent with MKT7/15.1.
- **Accessibility** — SEO layer doesn't alter the accessible DOM built in MKT8/MKT9; ensure JSON-LD/meta injection doesn't disturb landmark structure or focus.
- **Scalability** — For large catalogs, prerender concurrency-limited and incremental where feasible; sitemap may split (`sitemap-index`) beyond 50k URLs (note; unlikely near-term).
- **Reliability** — Build fails loudly on API error (FR-7) to avoid shipping empty course pages. Freshness bounded by deploy cadence (see Risks/OQ) — new courses appear after the next build.
- **Observability** — Post-build log: count of prerendered pages + sitemap URLs. Track Search Console coverage + `gtag` `page_view` per course route after launch.
- **Maintainability** — Prerender is a small, well-commented Node script wired into `package.json` `build` (`tsc -b && vite build && node scripts/prerender-courses.mjs`), reading `API_BASE`. Reuse server `jsonLd` to avoid schema drift. No SSR framework migration.
- **Internationalization** — English-only now; structure meta injection so locale could be added later.
- **Backward compatibility** — Additive build step + static files; existing pages, `404.html` trick, and `base:'/'` asset strategy unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a listed course with slug `s`, *When* I `curl https://lextures.com/courses/s` (no JS), *Then* the HTML contains the course's `<title>`, meta description, canonical, OG/Twitter tags, and a Course JSON-LD block.
- **AC-2.** *Given* the same course, *When* I run Google's Rich Results / a social preview scraper, *Then* it detects valid Course structured data and a rich preview with the hero image.
- **AC-3.** *Given* the build completes, *When* I open `dist/sitemap.xml`, *Then* it lists `/courses` and every listed course URL, and `robots.txt` references it.
- **AC-4.** *Given* an unlisted/unpublished course, *When* I inspect the build output + sitemap, *Then* it has no prerendered page and no sitemap entry.
- **AC-5.** *Given* I navigate client-side from `/courses` to a course and to another, *When* each renders, *Then* `document.title`, meta description, canonical, and JSON-LD reflect the current course.
- **AC-6.** *Given* the API is down at build time, *When* I run `npm run build`, *Then* the build fails with a clear error (unless `SKIP_COURSE_PRERENDER=1`).
- **AC-7.** *Given* a prerendered course URL, *When* a real browser loads it, *Then* the SPA hydrates without duplicate/flashing `<head>` tags or JSON-LD.
- **AC-8.** *Given* a course field with an HTML/script payload, *When* prerendered, *Then* it is escaped in title/description/OG (no injection).

## 8. Data Model

No data model. Build-time consumer of MKT7 JSON (list for enumeration + sitemap; detail for `jsonLd`/meta). Optionally cache the fetched list to a temp file during build for the sitemap + prerender passes.

## 9. API Surface

Consumes (build-time + runtime; no new server surface):

- `GET {API_BASE}/api/v1/public/marketplace/courses` (paginated) — enumerate slugs + `lastmod`/`createdAt` for sitemap + prerender list.
- `GET {API_BASE}/api/v1/public/marketplace/courses/{slug}` — per-course `jsonLd` + meta fields (MKT7 embeds `jsonLd`, its OQ #2 default).
- Emits static artifacts: `dist/courses/**/index.html`, `dist/sitemap.xml`, `dist/robots.txt` (or `public/robots.txt`).

## 10. UI / UX

- No new visible UI. Changes are in `<head>` and static build artifacts.
- **Runtime** — `useDocumentHead({ title, description, canonical, jsonLd? })` hook used by MKT8/MKT9 pages; swaps tags on route change; strips/replaces the previous route's JSON-LD.
- **Social preview** — verify OG/Twitter renders a card with hero + title + price via a scraper before GA.
- **Empty/error** — if a course is delisted between builds, its stale static page should 404 or redirect on next build; runtime detail fetch already shows "Course not found" (MKT9) if the API 404s.

## 11. AI / ML Considerations

Not AI-touching. (Meta description is a deterministic truncation of the course description; no generation.)

## 12. Integration Points

- **www internal (new)** — `www/scripts/prerender-courses.mjs` (prerender + sitemap + robots), `www/src/lib/use-document-head.ts`; `package.json` `build` script update.
- **www internal (reused)** — built `dist/index.html` shell, `www/public/404.html` (SPA redirect, unchanged), `www/index.html` (default meta as fallback template), `lib/api-base.ts` (`API_BASE`), MKT8/MKT9 pages (call `useDocumentHead`), existing `gtag` for post-launch measurement.
- **Server** — MKT7 list + detail (with `jsonLd`).
- **Ops** — GitHub Pages deploy; consider a scheduled rebuild to refresh the sitemap/prerender as new courses list (see OQ).

## 13. Dependencies & Sequencing

- **After** — MKT7 (API + `jsonLd`), MKT8 (storefront route), MKT9 (detail route). Prerender needs real pages to snapshot.
- **Before** — nothing in this epic; it is the closer. (Precedes any future paid-acquisition/landing work.)
- **Shared infra** — GitHub Pages build/deploy pipeline; optional scheduled CI rebuild.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Static prerender is stale until next deploy; new courses not indexed promptly | H | M | Scheduled CI rebuild (e.g. daily) or rebuild-on-publish webhook; `/courses` grid is always live client-side regardless |
| GitHub Pages `404.html` bounce serves crawlers empty shells for non-prerendered paths | M | H | Prerender real files for all listed slugs (FR-1/FR-9); sitemap only lists prerendered URLs |
| Large catalog blows up build time / URL count | L | M | Concurrency-limit prerender; split sitemap into an index beyond 50k URLs |
| HTML injection via course text in meta/OG | L | H | Escape all injected strings; JSON-encode JSON-LD (FR-8/AC-8) |
| Duplicate/flashing head tags on hydrate | M | L | Idempotent `useDocumentHead`; prerendered values equal runtime values |
| Delisted course leaves an orphan static page | M | L | Next build removes it (`emptyOutDir`); runtime shows not-found; sitemap drops it |
| API unreachable during CI build blocks deploys | M | M | Loud failure + `SKIP_COURSE_PRERENDER=1` escape hatch (FR-7) |

## 15. Rollout Plan

- **Flag** — none; ships with the www deploy. Server `FFCourseMarketplace` still governs whether the API returns courses (empty catalog → prerender only `/courses`).
- **Sequencing** — after MKT8/MKT9 are live; add prerender + sitemap + robots, deploy, then submit `sitemap.xml` to Google Search Console.
- **Dogfood** — validate a few course URLs via Rich Results Test + a social scraper on staging.
- **GA criteria** — course pages return crawlable HTML + valid Course JSON-LD; sitemap/robots live and submitted; runtime head updates correct; build fails loud on API error.
- **Rollback** — revert the `build` script to `tsc -b && vite build` (drops prerender/sitemap); pages still work client-side, just without SEO artifacts.

## 16. Test Plan

- **Unit** — meta/OG string builders (escaping, truncation); sitemap XML generation; `useDocumentHead` set/reset (incl. JSON-LD swap).
- **Build/integration** — run prerender against a mock/staging API: assert `dist/courses/<slug>/index.html` exists with expected tags + JSON-LD; sitemap contains exactly the listed slugs; robots references sitemap; unlisted course absent (AC-4); API-down fails build (AC-6).
- **Structured data** — validate JSON-LD against schema.org Course (Rich Results Test in CI or a schema validator lib).
- **End-to-end** — `curl` prerendered URL asserts no-JS content (AC-1); real-browser hydration has no duplicate head/JSON-LD (AC-7); client-side nav updates head (AC-5).
- **Security** — course field with `<script>`/HTML is escaped in output (AC-8).
- **Social** — manual OG/Twitter preview check pre-GA.

## 17. Documentation & Training

- **Runbook** — how prerender works, `SKIP_COURSE_PRERENDER`, freshness/rebuild cadence, submitting/monitoring the sitemap in Search Console.
- **Site docs** — note that course pages are SEO-optimized and how new listings become indexed (after the next build/scheduled rebuild).
- **Growth** — Search Console setup + coverage monitoring checklist.

## 18. Open Questions

1. Prerender freshness: daily scheduled CI rebuild, rebuild-on-course-publish webhook, or manual? (**Default:** daily scheduled rebuild + manual trigger; webhook as a fast-follow.)
2. Custom Node prerender script vs adopting a plugin (`vite-react-ssg` / `vite-plugin-ssr` / puppeteer prerender)? (**Default:** small custom script — matches the hand-rolled, dependency-light www stack and injects server `jsonLd` directly.)
3. react-helmet(-async) for runtime head vs a tiny custom `useDocumentHead` hook? (**Default:** custom hook — avoids a dependency for a small need.)
4. Should the sitemap also include the public **catalog** (`is_public`) pages on `self.lextures.com/explore/*`, or only the www `/courses/*` set? (**Default:** only www `/courses/*` here; the app owns its own SSR/JSON-LD for `/explore` — plan 15.1/17.5.)
5. Do we need `og:image` generation (branded card) for courses lacking a hero, or is a static default enough? (**Default:** static branded default for v1.)
6. Is real-time SEO (SSR host) ever required, or is build-time prerender sufficient? (**Default:** build-time is sufficient for a marketplace catalog; revisit only if freshness SLAs tighten.)

## 19. References

- Existing files: `www/index.html` (single static `<head>` + `gtag` `G-JX182Q6KKX` + `404.html` restore script), `www/public/404.html` (GitHub Pages SPA redirect), `www/vite.config.ts` (`base:'/'`, GitHub Pages asset note), `www/public/CNAME`/`.nojekyll`, `www/src/app.tsx` (routes to enumerate), `www/src/lib/api-base.ts`. Server JSON-LD: `server/internal/service/catalogsearch/jsonld.go` (`BuildCourseJSONLD`), `httpserver/public_catalog_http.go` (`jsonLd` embedding pattern), plan 17.5 (CDN cache headers).
- Related plans: [MKT7](MKT7-public-marketplace-api.md), [MKT8](MKT8-www-courses-storefront.md), [MKT9](MKT9-www-course-detail-enroll.md), `docs/completed/15-self-learner-specific/15.1-public-course-catalog.md`.
