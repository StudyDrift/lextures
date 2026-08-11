# SEO.1 — Static Rendering & Crawlability Foundation

> Implementation plan. Source: [docs/plan/seo/audit.md](audit.md) §S0 (F-1, F-2, F-3).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.1 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING (3 of 31 URLs return `200 OK` with content) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform |
| **Depends on** | — |
| **Unblocks** | SEO.2, SEO.3, SEO.5, SEO.6, SEO.7, SEO.8, SEO.9, SEO.10, SEO.11, SEO.14, SEO.16, SEO.17 |

---

## 1. Problem Statement

`www` is a client-rendered SPA on GitHub Pages with exactly four HTML files on disk. Every route
other than `/`, `/courses` and `/self-learner` is served by `404.html` — an **HTTP 404 with an empty
body** — which uses a JavaScript hack to bounce the browser to `/?/<path>` and let the SPA take over
(audit F-1). Search engines treat that as a removed page; AI crawlers, which are HTML-only parsers
([research §2](research.md#2-ai-crawlers-do-not-run-javascript)), receive nothing at all. The one
prerenderer we built is switched off in CI via `SKIP_COURSE_PRERENDER: "1"` (audit F-2), and 18 of 21
pages render under the homepage's `<title>` and meta description (audit F-3). Until every URL returns
real HTML with its own metadata, no other SEO investment can return anything.

## 2. Goals

- Every URL in the route manifest returns **HTTP 200** with fully-rendered, JavaScript-free HTML
  containing the page's primary content.
- Every page carries a unique `<title>`, meta description, and self-referential canonical **in the
  served bytes**, not applied by a `useEffect`.
- A single **route manifest** becomes the source of truth shared by the router, the static generator,
  the sitemap builder, and CI assertions — so a new page cannot ship without SEO metadata.
- Course prerendering is re-enabled in production and made resilient rather than skipped.
- Client-side navigation continues to work, with hydration that does not flash or duplicate content.

## 3. Non-Goals

- Server-side rendering **on request**. Content is publishable at build time; SSG is sufficient and
  strictly cheaper to operate.
- Rewriting the site in Next.js / Astro. The existing `prerender-courses.mjs` is well-factored and
  unit-tested; this plan generalises it.
- Changing visual design, copy, or page structure. Content changes live in SEO.6–SEO.10.
- Localised routing (SEO.17) and structured-data expansion beyond plumbing (SEO.3).

## 4. Personas & User Stories

- **As a prospective district administrator**, I want to ask an AI assistant "which LMS has a public
  VPAT and a self-hosting option" and see Lextures, so that we enter the shortlist.
- **As a homeschool parent**, I want the Lextures pricing page to appear in a Google result, so that
  I can evaluate cost without creating an account.
- **As an instructor sharing a blog post in Slack**, I want the unfurl to show that post's title and
  summary, so that colleagues click it.
- **As a marketing engineer**, I want adding a page to be one manifest entry, so that I cannot forget
  the sitemap, the canonical, or the title.
- **As an SRE**, I want the build to fail loudly when a route would ship without HTML, so that a
  regression never reaches production silently.

## 5. Functional Requirements

**Route manifest**

- **FR-1.** The system MUST define a single route manifest at `www/src/lib/route-manifest.ts`
  exporting an ordered list of `RouteDescriptor` records:
  ```ts
  export type RouteDescriptor = {
    path: string                     // '/pricing' — no trailing slash except '/'
    component: () => ReactElement    // rendered by both router and SSG
    title: string
    description: string              // ≤160 chars, enforced
    changefreq?: 'daily'|'weekly'|'monthly'|'yearly'
    sitemap: boolean                 // false for legal-history, thank-you pages
    robots?: 'index,follow'|'noindex,follow'
    ogImage?: string                 // absolute URL, PNG/JPEG (see SEO.14)
    jsonLd?: (ctx: RenderContext) => JsonLdNode[]  // multi-node (see SEO.3)
    lastmodSource?: 'git'|'content'|'build'
  }
  ```
- **FR-2.** `App` in `www/src/app.tsx` MUST resolve routes by looking them up in the manifest rather
  than the current 30-line `if` ladder, so router and generator can never diverge.
- **FR-3.** Dynamic route families (`/blog/:slug`, `/docs/:slug`, `/courses/:slug`) MUST expose an
  `enumerate()` function returning every concrete path at build time.

**Static generation**

- **FR-4.** `npm run build` MUST, after `vite build`, render **every** manifest path to
  `dist/<path>/index.html` using `react-dom/server`'s `renderToString`, producing HTML whose
  `<body>` contains the page's headings and body copy without executing JavaScript.
- **FR-5.** Each generated file MUST contain, in `<head>`: unique `<title>`, `<meta name="description">`,
  `<link rel="canonical">` pointing at the absolute non-trailing-slash URL, OG + Twitter tags, and any
  JSON-LD nodes returned by `jsonLd()`.
- **FR-6.** The generator MUST fail the build (non-zero exit) if any manifest route produces no
  output file, an empty `<body>`, a missing/duplicate title, or a description longer than 160
  characters.
- **FR-7.** `SKIP_COURSE_PRERENDER` MUST be removed from `.github/workflows/pages-www.yml`. Course
  prerendering MUST run in production. When the marketplace API is unreachable, the build MUST reuse
  the **previous deploy's** course HTML (fetched from the live site) and emit a warning, rather than
  either failing the whole site build or shipping zero course pages.
- **FR-8.** Generated pages MUST hydrate cleanly: `hydrateRoot` replaces `createRoot`, and rendering
  MUST be free of hydration mismatches (verified by a zero-console-error assertion in E2E).

**Metadata coverage**

- **FR-9.** Every page component MUST obtain its head values from the manifest; `useDocumentHead`
  MUST be called for all routes so client-side navigation stays in sync (audit F-3).
- **FR-10.** `www/index.html` MUST stop hard-coding homepage-specific `<title>`/`description`/OG tags;
  those move into the homepage's manifest entry. The template keeps only truly global tags.
- **FR-11.** The SPA-fallback script in `www/index.html` and `www/public/404.html` MUST be reduced to
  a genuine 404 experience — a styled "page not found" page with links to the top hubs and
  `<meta name="robots" content="noindex">` — because no legitimate route reaches it any more.

**Hosting & status codes**

- **FR-12.** The site SHOULD migrate from GitHub Pages to a host that can issue real
  `301`/`308` responses and set response headers (Cloudflare Pages is the recommended target, matching
  existing AWS/Cloudflare tooling in `iac/`). GitHub Pages cannot express a redirect or a
  `Cache-Control` header, which blocks SEO.5's redirect map and SEO.15's edge-log crawl analytics.
- **FR-13.** Until FR-12 lands, legacy path changes MUST use a `<link rel="canonical">` + meta-refresh
  stub (the shape already used by `dist/self-learner/index.html`), and the redirect debt MUST be
  recorded in SEO.5's redirect map for replay on migration.
- **FR-14.** The generator MUST emit both `dist/<path>/index.html` and, where the host supports it, a
  `_redirects`/`_headers` file so trailing-slash and legacy-path handling is declarative.

## 6. Non-Functional Requirements

- **Performance** — SSG must not regress runtime performance. Generated HTML ≤ 100 KB uncompressed
  per page before assets. Full build (including ~200 pages + all courses) ≤ 5 minutes in CI.
  LCP target inherited from [SEO.4](SEO.4-core-web-vitals-and-page-experience.md): < 2.0 s.
- **Security** — the generator runs in CI with no secrets beyond the public marketplace API base. All
  interpolated content passes through the existing `escapeHtml` helper
  (`www/src/lib/document-head.ts:16`). No user-supplied course text may reach an unescaped sink; a
  JSON-LD serialiser MUST escape `<`, `>`, `&` and `</script`.
- **Privacy & Compliance** — prerendered course pages expose only already-public marketplace fields.
  No learner data, no enrollment counts below the k-anonymity threshold used in SEO.12.
- **Accessibility** — generated HTML must preserve the WCAG 2.2 AA work from UX.5/UX.6: landmark
  order, skip link, heading hierarchy, and focus behaviour after client-side navigation (focus moves
  to the new `<h1>`; announced via a polite live region).
- **Scalability** — generation is O(routes); the course family may reach thousands. Course rendering
  MUST be concurrent (bounded pool, default 8) and MUST stream to disk rather than buffering all
  pages in memory.
- **Reliability** — build is deterministic: same inputs → byte-identical output (no timestamps in
  page HTML). Course-API failure degrades to previous-deploy reuse (FR-7), never to a broken deploy.
- **Observability** — generator prints a summary table (routes generated, bytes, longest title,
  slowest render) and writes `dist/.seo-manifest.json` listing every generated URL with its title,
  description, canonical, `lastmod` and schema types, for consumption by SEO.2 and SEO.16.
- **Maintainability** — pure helpers stay in `document-head.ts` and are unit-tested with
  `node --test`, matching the existing pattern. No new runtime dependency in the browser bundle.
- **Internationalization** — the manifest carries an optional `locale` field, unused in this plan,
  reserved for [SEO.17](SEO.17-international-seo-and-hreflang.md).
- **Backward compatibility** — all current URLs keep working. `/?/path` legacy URLs (which may exist
  in the wild from the 404 hack) MUST 301/canonical to the clean path.

## 7. Acceptance Criteria

- **AC-1.** *Given* the production build output, *When* I request any URL listed in `sitemap.xml`
  with `curl -sI`, *Then* the status is `200` and the body contains that page's `<h1>` text.
- **AC-2.** *Given* JavaScript is disabled, *When* I load `/pricing`, `/k-12`, `/higher-ed`,
  `/homeschool`, `/security`, `/accessibility/vpat`, `/blog/<any>` and `/docs/<any>`, *Then* the full
  page copy is visible and all navigation links are real `<a href>` elements.
- **AC-3.** *Given* the build output, *When* I extract every `<title>`, *Then* all titles are unique,
  non-empty, and ≤ 60 characters; and every description is unique, non-empty, ≤ 160 characters.
- **AC-4.** *Given* the build output, *When* I extract every `<link rel="canonical">`, *Then* each is
  an absolute `https://lextures.com` URL equal to the file's own path, with no trailing slash except
  for `/`.
- **AC-5.** *Given* the marketplace API returns 500 during a build, *When* the build runs, *Then* it
  completes, reuses the previously deployed course HTML, logs a `WARN`, and exits 0.
- **AC-6.** *Given* a developer adds a route to `app.tsx` without a manifest entry, *When* CI runs,
  *Then* the build fails with a message naming the missing route.
- **AC-7.** *Given* a generated page, *When* it hydrates in a browser, *Then* the console contains
  zero hydration-mismatch warnings (asserted in Playwright).
- **AC-8.** *Given* a client-side navigation from `/` to `/pricing`, *When* it completes, *Then*
  `document.title`, the meta description, and the canonical link all match the prerendered values for
  `/pricing`, and focus has moved to the new `<h1>`.
- **AC-9.** *Given* a request for a genuinely nonexistent path (`/nope`), *When* it is served, *Then*
  the response is a 404 page carrying `<meta name="robots" content="noindex">` and links to `/`,
  `/docs` and `/blog`.

## 8. Data Model

No database changes. Build-time artefacts only:

| Artefact | Path | Purpose |
|---|---|---|
| Route manifest | `www/src/lib/route-manifest.ts` | Source of truth for routes + head metadata |
| SEO manifest | `dist/.seo-manifest.json` | Machine-readable record of every generated URL (consumed by SEO.2 sitemap builder and SEO.16 CI assertions) |
| Redirect map | `www/src/lib/redirects.ts` | `{ from, to, status }[]`, rendered to `_redirects` (FR-14) and to canonical stubs (FR-13) |
| Course snapshot cache | `dist/.course-cache.json` | Last-known-good course payload for FR-7 degradation |

`.seo-manifest.json` shape:

```jsonc
{
  "generatedAt": "2026-09-01T00:00:00Z",
  "origin": "https://lextures.com",
  "urls": [
    {
      "path": "/pricing",
      "title": "Lextures pricing — per-student plans for K-12, higher ed & homeschool",
      "description": "…",
      "canonical": "https://lextures.com/pricing",
      "lastmod": "2026-08-22",
      "robots": "index,follow",
      "schemaTypes": ["WebPage", "BreadcrumbList", "Offer"],
      "bytes": 41233
    }
  ]
}
```

## 9. API Surface

No new HTTP routes on the Go server. The build consumes one existing public endpoint:

- `GET {API_BASE}/api/v1/public/marketplace/courses?page=&perPage=` — paginated listing (already used).
- `GET {API_BASE}/api/v1/public/marketplace/courses/{slug}` — detail incl. server-built `jsonLd`.

Requirements on those calls:
- Requests MUST send `User-Agent: lextures-www-prerender/<version>` so server-side rate limits can
  distinguish build traffic.
- The generator MUST respect `Retry-After` and back off exponentially (max 3 attempts per page).
- Rate limit: build traffic must stay under 20 req/s against the public API.
- No OpenAPI change required.

## 10. UI / UX

- **No visual change** to any existing page — this is a rendering-pipeline change.
- **New page:** a real `/404` experience replacing the redirect stub (heading, short explanation,
  links to Home, Docs, Blog, Pricing, Courses; `noindex`).
- **Flows**
  1. Cold load of any URL → server returns full HTML → content visible before JS → hydration attaches
     interactivity.
  2. In-app navigation → client router swaps view, `useDocumentHead` updates head, focus moves to
     `<h1>`, live region announces the new page title.
  3. Unknown URL → 404 page.
- **States** — because content is server-rendered, the blog/docs/legal pages have **no loading state**
  at all on first paint. The marketplace pages keep a loading state for the live enrollment panel
  only. Offline: previously visited pages are served from HTTP cache.
- **Responsive** — unchanged.
- **Accessibility** — SSG must preserve the skip link as the first focusable element; post-navigation
  focus management is new behaviour and must be verified with NVDA + VoiceOver.
- **Copy & i18n** — one new string set for the 404 page; keys registered under `www.notFound.*`.

## 11. AI / ML Considerations

Not AI-touching. It is, however, the enabling step for every AI-search outcome in this plan set: the
whole point is that model crawlers can only read what is in the HTML bytes.

## 12. Integration Points

- **External:** GitHub Pages (current host) → Cloudflare Pages (target, FR-12); public marketplace API.
- **Internal modules touched:**
  - `www/src/app.tsx` — router rewritten against the manifest
  - `www/src/main.tsx` — `createRoot` → `hydrateRoot`
  - `www/index.html` — strip page-specific meta + SPA-restore script
  - `www/public/404.html` — replaced by a real 404 page
  - `www/scripts/prerender-courses.mjs` → generalised to `www/scripts/generate-site.mjs`
  - `www/src/lib/document-head.ts` — multi-node JSON-LD, `robots` support
  - `www/src/lib/use-document-head.ts` — manifest-driven defaults
  - `.github/workflows/pages-www.yml` — remove `SKIP_COURSE_PRERENDER`, add assertions
- **Events:** none.

## 13. Dependencies & Sequencing

- **Must ship after:** nothing.
- **Must ship before:** SEO.2 (needs `.seo-manifest.json`), SEO.3 (needs multi-node JSON-LD), SEO.5,
  SEO.6, SEO.7, SEO.8, SEO.9, SEO.10, SEO.11, SEO.14, SEO.16, SEO.17.
- **Shared infra:** CI runner with Node 22 (present); optional Cloudflare Pages project + DNS change
  for FR-12.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Hydration mismatches cause visible flicker or lost interactivity | M | H | Render-time guards for `window`/`document` access; Playwright asserts zero hydration warnings on every route (AC-7); `hero-canvas` and other browser-only components render a static placeholder server-side |
| Build time balloons with thousands of course pages | M | M | Bounded concurrency pool, incremental generation keyed on course `updatedAt`, streamed writes; CI budget alarm at 5 min |
| Marketplace API outage breaks the deploy | M | H | FR-7 previous-deploy reuse + `.course-cache.json`; never fail the site build on course-data failure |
| GitHub Pages cannot 301, so URL changes leak equity | H | M | FR-12 migration to Cloudflare Pages; until then FR-13 canonical stubs and a recorded redirect map |
| Google briefly re-crawls old `/?/path` URLs and sees duplicates | M | M | Self-referential canonicals everywhere + explicit 301/canonical from `/?/*` to the clean path; monitor GSC "Duplicate, Google chose different canonical" |
| Ranking dip during re-indexing of URLs that were 404 | L | M | These URLs have effectively no current equity to lose (they 404 today); submit sitemap immediately after deploy (SEO.2) |
| Manifest becomes a merge-conflict hotspot | M | L | Split manifest into per-section files re-exported from one index |

## 15. Rollout Plan

- **Feature flag:** none at runtime — this is a build-output change. The rollout gate is the
  **staging deploy**, published to a `preview.lextures.com` Pages environment.
- **Sequencing**
  1. Land the route manifest + manifest-driven router; no output change yet (pure refactor, green CI).
  2. Land `generate-site.mjs` writing HTML for **static** routes only; verify on staging.
  3. Switch `main.tsx` to `hydrateRoot`; verify no hydration warnings.
  4. Extend generation to `/blog/*`, `/docs/*`; verify.
  5. Remove `SKIP_COURSE_PRERENDER`; verify course pages on staging against the real API.
  6. Replace `404.html` with the real 404 page and delete the SPA-restore script.
  7. Deploy to production; immediately resubmit sitemap (SEO.2) and request indexing for the 12
     highest-value URLs.
  8. FR-12 host migration as a separate, reversible change (DNS cutover with low TTL).
- **Dogfood:** internal review of staging by Marketing + Docs before step 7.
- **GA criteria:** AC-1 through AC-9 pass; GSC shows ≥ 25 newly indexed URLs within 14 days; zero
  `Soft 404` or `Not found (404)` entries for manifest URLs.
- **Rollback:** revert the deploy commit — output reverts to the previous `dist`. DNS rollback for
  FR-12 (TTL 300 s during cutover).

## 16. Test Plan

- **Unit** (`node --test`, extending `prerender-courses.test.mjs`)
  - manifest → head-tag rendering, escaping, description truncation
  - canonical construction (trailing slash, absolute origin)
  - multi-node JSON-LD serialisation and `</script` escaping
  - redirect-map → `_redirects` rendering
- **Integration**
  - Full `npm run build` against a mock marketplace API; assert file count, `.seo-manifest.json`
    contents, sitemap parity.
  - API-failure path: assert previous-deploy reuse and exit 0 (AC-5).
- **End-to-end (Playwright)**
  - For every manifest route: `page.setJavaScriptEnabled(false)` → assert `<h1>` and body copy present
    (AC-2).
  - Hydration: zero console errors/warnings per route (AC-7).
  - Client-side navigation head sync + focus management (AC-8).
  - Unknown path → 404 page with `noindex` (AC-9).
- **Security** — snapshot test that a course whose title contains `</script><img onerror>` is escaped
  in both the visible HTML and the JSON-LD block; run the site through the existing security-review
  checklist for the new generator script.
- **Accessibility** — axe on the server-rendered HTML (no JS) for every route; NVDA + VoiceOver script
  for post-navigation focus.
- **Performance / load** — Lighthouse CI on 10 representative routes with the SEO.4 budget applied;
  build-duration assertion in CI.
- **Manual exploratory** — QA checklist: view-source on 10 routes and confirm title/description/
  canonical/OG; share each in Slack and LinkedIn and confirm the unfurl; fetch each with
  `curl -A "GPTBot"` and confirm content is present.

## 17. Documentation & Training

- Rewrite `www/docs/marketplace-seo.md` → `www/docs/site-generation.md`: how the manifest works, how
  to add a page, how the course prerender degrades, what `.seo-manifest.json` is for.
- New `www/docs/adding-a-page.md` — a five-step checklist for Marketing/Docs contributors.
- Update `AGENTS.md` / `CLAUDE.md` www section with the "no route without a manifest entry" rule.
- Runbook: what to do when the course API is down during a deploy; how to force a full regeneration.

## 18. Open Questions

1. Cloudflare Pages vs. keeping GitHub Pages with canonical stubs — who owns the DNS cutover and what
   is the acceptable maintenance window? (Blocks FR-12; does not block FR-1–FR-11.)
2. Do we generate `/courses/*` on every www deploy, or on a scheduled daily job plus a
   publish-webhook rebuild? (Interacts with SEO.11 freshness targets.)
3. Should legal history pages (`/privacy/history`, `/terms/history`) be `index,follow` or
   `noindex,follow`? They are trust signals but also thin/duplicative.
4. Is `renderToString` sufficient, or do any pages require `renderToPipeableStream` for size? (Measure
   at step 2 of rollout.)
5. Who owns the 404-page copy and its i18n keys?

## 19. References

- Existing files: `www/src/app.tsx`, `www/src/main.tsx`, `www/index.html`, `www/public/404.html`,
  `www/scripts/prerender-courses.mjs`, `www/src/lib/document-head.ts`,
  `www/src/lib/use-document-head.ts`, `.github/workflows/pages-www.yml`, `www/docs/marketplace-seo.md`
- Audit findings: [F-1, F-2, F-3](audit.md#s0--critical-the-site-is-not-crawlable), F-20
- Research: [§2 AI crawlers do not run JavaScript](research.md#2-ai-crawlers-do-not-run-javascript)
- External: [Google — JavaScript SEO basics](https://developers.google.com/search/docs/crawling-indexing/javascript/javascript-seo-basics),
  [Google — HTTP status codes and Search](https://developers.google.com/search/docs/crawling-indexing/http-network-errors),
  [React — `hydrateRoot`](https://react.dev/reference/react-dom/client/hydrateRoot)
- Related plans: [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md),
  [SEO.3](SEO.3-structured-data-and-entity-graph.md),
  [SEO.4](SEO.4-core-web-vitals-and-page-experience.md),
  [SEO.16](SEO.16-seo-governance-and-ci-guardrails.md),
  [UX.17 — perceived performance & web-vitals budget](../ui-ux/UX.17-perceived-performance-and-web-vitals-budget.md)
