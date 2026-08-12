# SEO.4 — Core Web Vitals & Page-Experience Budget

> Completed August 11, 2026.

> Implementation plan. Source: [docs/plan/seo/audit.md](../../plan/seo/audit.md) §S3 (F-16, F-17, F-18).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.4 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL (583 KB single JS chunk, duplicated font loading with a render-blocking third-party stylesheet, unmeasured above-the-fold canvas animation) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform |
| **Depends on** | SEO.1 |
| **Unblocks** | — (quality gate for all content plans) |

---

## 1. Problem Statement

`www` ships **583.5 KB of JavaScript in a single chunk** plus 56.5 KB of CSS with no route-level code
splitting, so a visitor reading the privacy policy downloads the pricing calculator, the marketplace
client, the markdown renderer and a canvas animation (audit F-16). Fonts load **twice** — self-hosted
WOFF2 preloads *and* a render-blocking Google Fonts stylesheet on the critical path (F-17) — and a
284-line canvas animation runs in the LCP region unmeasured (F-18). Google lowered the "good" LCP
threshold to **2.0 s** in March 2026 and promoted **INP to an equal ranking signal**; sites failing
these dropped 0.8–4 positions on competitive queries, and INP is the most-failed vital industry-wide
([research §6](../../plan/seo/research.md#6-page-experience-the-bar-moved-up-in-march-2026)). Once SEO.1 makes every
page real, page experience becomes the differentiator on queries where many relevant pages exist —
which is every query we care about.

## 2. Goals

- Pass the March-2026 thresholds on **100% of indexable URLs** in field data: LCP < 2.0 s,
  INP < 200 ms, CLS < 0.1 (p75, mobile).
- Cut the critical-path JavaScript for a content page to **< 150 KB** compressed by splitting per
  route and shipping content pages with effectively no interactive JS.
- Eliminate third-party render-blocking resources entirely — which also removes a Google Fonts CDN
  privacy surface that sits badly next to our own privacy positioning.
- Make the budget **enforced in CI**, not aspirational: a PR that regresses it fails.

## 3. Non-Goals

- Redesign. Visual output must be pixel-identical except where a fix is inherently visual (e.g.
  reserving image space to remove layout shift).
- Server/app performance (`clients/web`) — that is [UX.17](../../plan/ui-ux/UX.17-perceived-performance-and-web-vitals-budget.md)
  and section 17 plans. This plan is scoped to `www`.
- Removing motion. The signature animation language from the AN plan set stays; it gets budgeted and
  gated on `prefers-reduced-motion` and device capability.

## 4. Personas & User Stories

- **As a parent on a 4G phone**, I want the homeschool page readable in under two seconds, so that I
  do not bounce before I learn what Lextures is.
- **As a district IT director on a locked-down network**, I want the site to render without calling
  third-party domains, so that it works behind our egress filter.
- **As a privacy-conscious buyer**, I want the marketing site not to leak my IP to a font CDN, so
  that the privacy claims on the site are consistent with its behaviour.
- **As a web engineer**, I want a failing CI check when I add 80 KB to the bundle, so that regression
  is caught in review rather than in Search Console 28 days later.

## 5. Functional Requirements

**Budgets**

- **FR-1.** The repo MUST define a performance budget in `www/perf-budget.json`, enforced in CI:

  | Metric | Content page | Interactive page (`/pricing/calculator`, `/courses/*`) |
  |---|---|---|
  | JS transferred (gzip) | ≤ 60 KB | ≤ 150 KB |
  | CSS transferred (gzip) | ≤ 25 KB | ≤ 25 KB |
  | Total transferred | ≤ 350 KB | ≤ 600 KB |
  | Requests | ≤ 20 | ≤ 30 |
  | Lab LCP (Moto G Power, 4G) | ≤ 1.8 s | ≤ 2.0 s |
  | Lab TBT | ≤ 150 ms | ≤ 250 ms |
  | CLS | ≤ 0.05 | ≤ 0.05 |

- **FR-2.** CI MUST fail when any budget is exceeded, reporting the offending route, the metric, and
  the delta.

**JavaScript**

- **FR-3.** The bundle MUST be split per route using dynamic `import()` at the route-manifest
  boundary (SEO.1 FR-2), so no page loads another page's code.
- **FR-4.** Purely-static pages (legal, security, accessibility, blog posts, help articles, `/about`,
  `/authors/*`, glossary) MUST ship **zero page-level JavaScript** beyond the shared header/footer
  behaviour — they are prerendered HTML and need no hydration. Implement via an `interactive: false`
  flag on the route descriptor that skips hydration for that route entirely (islands, not full-page
  hydration).
- **FR-5.** `react-markdown` + `remark-gfm` MUST NOT ship to the browser. Markdown is rendered to
  HTML at build time (SEO.1 already renders server-side); the runtime dependency is removed from the
  client graph.
- **FR-6.** `lucide-react` MUST be imported per-icon (or inlined as SVG at build time) so the icon set
  is not bundled wholesale.
- **FR-7.** Third-party scripts MUST be deferred and must not block rendering. The GA4 snippet moves
  behind `requestIdleCallback` (or a consent gate, see FR-16) and MUST NOT be in the critical path.

**Fonts**

- **FR-8.** The Google Fonts `<link>` and both `preconnect`s MUST be removed from `www/index.html`.
  IBM Plex Mono and Spectral MUST be self-hosted as subset WOFF2 alongside the existing
  `lextures-*.woff2` files.
- **FR-9.** All font faces MUST declare `font-display: swap` and a metric-compatible
  `size-adjust`/`ascent-override` fallback so the swap causes no layout shift.
- **FR-10.** Only fonts used above the fold MUST be `preload`ed (currently two weights are preloaded;
  audit which are actually needed for the H1 and body).
- **FR-11.** Fonts MUST be subset to the Latin range actually used, with the full range loaded lazily
  only if a page needs it.

**Images & media**

- **FR-12.** All raster images MUST be served as AVIF with WebP fallback, with explicit `width`/
  `height` (or `aspect-ratio`) to eliminate CLS, `loading="lazy"` below the fold, and
  `fetchpriority="high"` on the LCP image only.
- **FR-13.** The seven screenshots in `public/assets/screenshots/` and seven docs images MUST be
  compressed and converted at build time by a Vite image pipeline, not committed pre-optimised by
  hand.
- **FR-14.** `hero-canvas.tsx` and `wind-lines.tsx` MUST: not run before first paint; respect
  `prefers-reduced-motion: reduce` by rendering a static frame; pause when `document.hidden`; cap
  work with `requestAnimationFrame` throttling on low-end devices (`navigator.hardwareConcurrency ≤ 4`
  or `navigator.connection.saveData`); and MUST NOT be the LCP element.

**INP**

- **FR-15.** Every interactive control MUST keep main-thread work per interaction under 50 ms;
  long tasks in the calculator and course filters MUST be broken up with `scheduler.yield()` or
  chunked processing.
- **FR-16.** Analytics and any future consent banner MUST NOT run synchronous work in an event
  handler; measurement is deferred to idle time.

**Field measurement**

- **FR-17.** A lightweight `web-vitals` collector (≤ 2 KB) MUST report LCP, INP, CLS, TTFB and the
  attribution payload (LCP element, INP target selector) to GA4 as custom events, so we can debug
  *which element* is slow rather than only that a page is slow.
- **FR-18.** CrUX data for `lextures.com` MUST be pulled weekly into the SEO.15 dashboard.

## 6. Non-Functional Requirements

- **Performance** — the requirements above *are* the performance spec. Additionally: TTFB ≤ 200 ms at
  p75 (static files on a CDN edge; requires SEO.1 FR-12 hosting to set `Cache-Control` properly).
- **Security** — self-hosting fonts removes a third-party origin; a strict CSP becomes achievable and
  SHOULD be adopted (`default-src 'self'`, no `unsafe-inline` for scripts once the GA snippet is
  externalised with a nonce or hash). Track separately if CSP work grows.
- **Privacy & Compliance** — removing Google Fonts eliminates an unnecessary IP disclosure to a third
  party. If a consent banner is required in any jurisdiction, analytics must be consent-gated
  (coordinate with [S04](../../plan/standards/S04-unified-consent-preference-ledger.md)); the perf design
  must not assume analytics loads.
- **Accessibility** — `prefers-reduced-motion` handling (FR-14) is a WCAG 2.3.3 obligation, not only
  a perf one. Font-swap fallbacks must not reduce contrast or clip text.
- **Scalability** — budgets are per-route, so adding pages does not degrade existing ones.
- **Reliability** — no runtime dependency on a third-party CDN means no third-party outage can blank
  the site.
- **Observability** — FR-17 field data + weekly CrUX + per-PR Lighthouse CI report.
- **Maintainability** — budget lives in one JSON file; adding a route requires choosing
  `interactive: true|false`, which is reviewable.
- **Internationalization** — font subsetting must not drop glyphs needed by SEO.17 locales; the
  subsetting config is keyed by locale.
- **Backward compatibility** — no URL or markup contract changes. Removing the Google Fonts link
  changes rendered typography only if a family is missed — verified by visual regression.

## 7. Acceptance Criteria

- **AC-1.** *Given* a production build, *When* CI evaluates `perf-budget.json` against every route,
  *Then* no route exceeds any budget, and exceeding one fails the job with the route + metric + delta.
- **AC-2.** *Given* `/privacy` (a static content page), *When* loaded, *Then* transferred JavaScript
  is ≤ 60 KB gzip and the page is fully readable and navigable with JS disabled.
- **AC-3.** *Given* the deployed site, *When* I inspect network requests on any page, *Then* zero
  requests go to `fonts.googleapis.com` or `fonts.gstatic.com`.
- **AC-4.** *Given* a Moto G Power / 4G Lighthouse run on the homepage, `/pricing`, `/k12`,
  `/higher-ed`, `/homeschool`, `/blog/<post>`, `/docs/<article>`, `/courses`, `/courses/<slug>`,
  `/pricing/calculator`, *Then* LCP ≤ 2.0 s, TBT ≤ 250 ms, CLS ≤ 0.05 on all ten.
- **AC-5.** *Given* `prefers-reduced-motion: reduce`, *When* the homepage loads, *Then* the hero canvas
  renders a single static frame and no animation loop is scheduled.
- **AC-6.** *Given* a hidden tab, *When* checked after 5 s, *Then* no `requestAnimationFrame` callbacks
  from the canvas have run.
- **AC-7.** *Given* CrUX field data 28 days after launch, *When* reviewed, *Then* the origin passes
  all three Core Web Vitals at p75 on mobile and desktop.
- **AC-8.** *Given* the pricing calculator, *When* I change every input in sequence, *Then* no
  interaction records an INP above 200 ms in the field collector, and no long task exceeds 50 ms.
- **AC-9.** *Given* any page, *When* fonts swap from fallback to webfont, *Then* measured CLS
  contribution from the swap is 0.
- **AC-10.** *Given* a PR that adds a 100 KB dependency to a content route, *When* CI runs, *Then* the
  budget job fails.

## 8. Data Model

No database changes.

| Artefact | Path | Purpose |
|---|---|---|
| Budget | `www/perf-budget.json` | Per-route-class thresholds (FR-1) |
| Lighthouse CI config | `www/lighthouserc.json` | URL list, device profile, assertions |
| Field vitals events | GA4 custom events `web_vitals` | `{metric, value, rating, page_path, element_selector, navigation_type}` |
| Bundle report | CI artefact `dist/stats.html` | rollup-plugin-visualizer output attached to each PR |

## 9. API Surface

No new Lextures routes. Outbound:

- GA4 Measurement Protocol / gtag for `web_vitals` events (batched, `sendBeacon`, sampled at 100% —
  volume is low enough).
- CrUX API (`https://chromeuxreport.googleapis.com/v1/records:queryRecord`) called weekly by the
  SEO.15 collector with a keyed request; failures are non-fatal.

## 10. UI / UX

- **No intended visual change.** Two permitted exceptions:
  1. Images gain reserved space (removes visible reflow — an improvement).
  2. Reduced-motion users see a static hero (already the correct behaviour).
- **Flows** — unchanged.
- **States** — content pages lose their loading state entirely (they are prerendered). The calculator
  and course grid keep skeletons, which MUST match final layout dimensions to keep CLS at 0.
- **Responsive** — budgets are enforced on the mobile profile, which is the binding constraint.
- **Accessibility** — verify font fallback metrics do not clip descenders at 200% zoom; verify the
  static hero frame still meets contrast requirements.
- **Copy & i18n** — none.

## 11. AI / ML Considerations

Not AI-touching directly. Two indirect effects worth stating: AI crawlers time out on slow responses,
so TTFB and payload size affect whether we are ingested at all; and self-hosted fonts + no
third-party blocking resources mean a crawler fetching raw HTML gets a complete, self-contained
document.

## 12. Integration Points

- **External:** Lighthouse CI, CrUX API, GA4, CDN (Cloudflare Pages per SEO.1 FR-12).
- **Internal modules touched:** `www/index.html` (font + GA changes), `www/vite.config.ts` (splitting,
  image pipeline, visualizer), `www/src/main.tsx` (islands hydration), `www/src/hero-canvas.tsx`,
  `www/src/components/home/wind-lines.tsx`, `www/src/pages/blog-post.tsx` + `docs-post.tsx` (drop
  runtime markdown), `www/src/lib/route-manifest.ts` (`interactive` flag), `www/package.json`
  (dependency removals), `.github/workflows/pages-www.yml`, `.github/workflows/lighthouse.yml`.
- **Events:** `web_vitals` GA4 events.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md) — islands/no-hydration
  requires prerendered HTML to exist; per-route splitting requires the manifest.
- **Must ship before:** no plan hard-blocks on it, but it MUST be in place before
  [SEO.7](../../plan/seo/SEO.7-help-center-expansion.md) and [SEO.8](../../plan/seo/SEO.8-editorial-engine-and-content-calendar.md)
  scale page count, or the budget is retrofitted across hundreds of pages.
- **Shared infra:** CDN with proper cache headers; Lighthouse CI runner.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Islands/partial hydration breaks interactive components subtly | M | H | `interactive` flag is opt-in per route; Playwright coverage of every interactive control; staged rollout starting with legal pages |
| Self-hosted Spectral/IBM Plex Mono licensing | L | H | Both are OFL — verify and commit the licence alongside the files (an `OFL.txt` already ships for the existing family) |
| Font swap causes visible reflow | M | M | FR-9 metric-compatible fallbacks + AC-9 measurement |
| Budget too tight, blocks legitimate feature work | M | M | Budgets are per-route-class with an explicit, reviewed override mechanism recorded in `perf-budget.json` |
| Removing runtime markdown changes rendering of existing posts | M | M | Golden-file snapshot of rendered HTML for all existing posts before/after |
| CrUX has insufficient traffic for page-level data | H | L | Rely on origin-level CrUX plus our own field collector (FR-17), which works at any volume |
| Canvas removal requested for perf, losing brand character | L | M | Budget the canvas rather than delete it; measure first (FR-14), and it is explicitly not the LCP element |

## 15. Rollout Plan

- **Feature flag:** none at runtime. Rollout is staged by change class.
- **Sequencing**
  1. Instrument first: land FR-17 field collector + Lighthouse CI reporting **without** assertions,
     to capture a real baseline (also feeds SEO.15).
  2. Fonts (FR-8…FR-11) — highest ratio of impact to risk.
  3. Route splitting (FR-3) + drop runtime markdown (FR-5) + per-icon imports (FR-6).
  4. Islands / `interactive: false` (FR-4), starting with legal + security + accessibility pages.
  5. Images (FR-12, FR-13) and animation budgeting (FR-14).
  6. INP work on the calculator and course filters (FR-15).
  7. Turn budget assertions from warn → fail (FR-2).
- **Dogfood:** every step validated on staging with a Lighthouse run and a visual-regression diff.
- **GA criteria:** AC-1…AC-10 pass; CrUX origin-level pass on all three vitals at 28 days.
- **Rollback:** each step is an independent revert. Budget assertions can be flipped back to
  warn-only with a one-line change if they block an urgent fix.

## 16. Test Plan

- **Unit** — budget-file parsing and comparison logic; font-fallback metric calculation; image
  pipeline output format selection.
- **Integration** — build produces per-route chunks; assert no route's chunk graph includes another
  route's page module; assert `react-markdown` is absent from the client graph.
- **End-to-end (Playwright)** — JS-disabled readability of every `interactive: false` route;
  reduced-motion static hero (AC-5); hidden-tab rAF pause (AC-6); no `fonts.gstatic.com` request
  (AC-3); INP measurement over a scripted calculator interaction (AC-8).
- **Security** — verify no new third-party origins; confirm CSP compatibility of the deferred GA
  snippet.
- **Accessibility** — axe on all ten benchmark routes; 200% zoom check with fallback font active;
  reduced-motion audit.
- **Performance / load** — Lighthouse CI on the ten benchmark URLs, mobile profile, 3 runs median,
  asserting FR-1; bundle-size assertion via `rollup-plugin-visualizer` JSON output; weekly CrUX pull.
- **Manual exploratory** — throttled 3G walkthrough on a real low-end Android; verify on a network
  that blocks Google domains.

## 17. Documentation & Training

- `www/docs/performance-budget.md` — the budget, why each number, how to request an override, how to
  read the CI failure.
- Update `www/docs/site-generation.md` with the `interactive` flag semantics.
- Add "does this route need hydration?" to the add-a-page checklist (SEO.1 §17).
- Runbook: diagnosing a CrUX regression using the `web_vitals` attribution events.

## 18. Open Questions

1. Which font weights are genuinely needed above the fold? (Audit before setting FR-10.)
2. Is a consent banner required in any market we serve? It materially changes the analytics loading
   strategy — needs an answer from [S04](../../plan/standards/S04-unified-consent-preference-ledger.md) /
   [S11](../../plan/standards/S11-us-state-privacy-expansion.md).
3. Do we adopt a full islands framework, or hand-roll `interactive: false` in the existing Vite
   setup? (Recommendation: hand-roll first; the route count is small.)
4. Should GA4 be replaced with a lighter, cookieless analytics option given the privacy positioning?
   (Interacts with SEO.15's attribution needs.)
5. What is the acceptable visual delta in the regression suite — pixel-exact, or perceptual threshold?

## 19. References

- Existing files: `www/index.html` (:20-27 fonts, :29-37 GA), `www/vite.config.ts`,
  `www/src/hero-canvas.tsx`, `www/src/components/home/wind-lines.tsx`, `www/src/main.tsx`,
  `www/package.json`, `.github/workflows/lighthouse.yml`, `www/public/assets/screenshots/*`
- Audit findings: [F-16, F-17, F-18](../../plan/seo/audit.md#s3--performance--ia)
- Research: [§6 Page experience: the bar moved up in March 2026](../../plan/seo/research.md#6-page-experience-the-bar-moved-up-in-march-2026)
- External: [web.dev — Core Web Vitals](https://web.dev/articles/vitals),
  [web.dev — Optimize INP](https://web.dev/articles/optimize-inp),
  [Google — Page experience](https://developers.google.com/search/docs/appearance/page-experience),
  [CrUX API](https://developer.chrome.com/docs/crux/api)
- Related plans: [SEO.1](SEO.1-static-rendering-and-crawlability.md),
  [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md),
  [SEO.16](../../plan/seo/SEO.16-seo-governance-and-ci-guardrails.md),
  [UX.17 — perceived performance & web-vitals budget](../../plan/ui-ux/UX.17-perceived-performance-and-web-vitals-budget.md),
  [AN — motion & animation polish](../animations/)
