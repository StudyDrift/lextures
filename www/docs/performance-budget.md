# Performance budget (SEO.4)

Core Web Vitals and transfer budgets for the marketing site (`www/`). Enforced in CI so regressions fail the PR rather than Search Console 28 days later.

## Thresholds

Defined in [`www/perf-budget.json`](../perf-budget.json). Two route classes:

| Metric | Content (`interactive: false`) | Interactive |
|---|---|---|
| JS transferred (gzip) | ≤ 60 KB | ≤ 150 KB |
| CSS transferred (gzip) | ≤ 25 KB | ≤ 25 KB |
| Total transferred (HTML+CSS+JS gzip) | ≤ 350 KB | ≤ 600 KB |
| Lab LCP (Moto G Power / 4G) | ≤ 1.8 s | ≤ 2.0 s |
| Lab TBT | ≤ 150 ms | ≤ 250 ms |
| CLS | ≤ 0.05 | ≤ 0.05 |

Field targets (CrUX / `web_vitals` events): **LCP &lt; 2.0 s**, **INP &lt; 200 ms**, **CLS &lt; 0.1** (p75, mobile).

## How enforcement works

1. `npm run build` optimizes images, builds Vite (main + `static-island` entries), then SSG.
2. `npm run perf-budget` (`scripts/check-perf-budget.mjs`) measures gzip JS/CSS, total referenced assets, and requests on every indexable route in the generated SEO manifest.
3. `pages-www.yml` runs the budget check after build and fails the job with **route + metric + delta**.

Lighthouse CI config lives in [`www/lighthouserc.json`](../lighthouserc.json); its lab assertions fail CI when a benchmark regresses.

## `interactive` flag

On each `RouteDescriptor` in `route-manifest.tsx`:

- **`interactive: false`** (legal, blog, docs, about, authors, static marketing shells): generate-site strips the React module script and injects only `static-island-*.js` (nav + deferred GA + web-vitals). The page is fully readable with JS disabled.
- **`interactive: true`** (home, calculator, courses, request-information, get-started): full React hydration with per-route dynamic `import()`.

When adding a page, ask: does this route need client state, forms, or marketplace API? If not, set `interactive: false`.

## Override process

If a legitimate feature needs more budget:

1. Document the reason in a PR description.
2. Add an entry under `overrides` in `perf-budget.json` with `route`, `metric`, `value`, and `reason`.
3. Prefer splitting the feature into a lazy chunk over raising the global class budget.

## Diagnosing regressions

| Signal | Where |
|---|---|
| Bundle map | `dist/stats.html` (rollup-plugin-visualizer) after every build |
| Transfer budgets | CI log from `npm run perf-budget` |
| Field LCP/INP/CLS + element | GA4 custom event `web_vitals` (`metric_name`, `element_selector`, `page_path`) |
| Lab | Lighthouse against `lighthouserc.json` URLs on a preview deploy |

## Fonts & third parties

- All fonts are self-hosted under `public/fonts/` (Lextures, Spectral, IBM Plex Mono). **No** `fonts.googleapis.com` / `fonts.gstatic.com`.
- GA4 loads via `requestIdleCallback` (`src/lib/analytics.ts`); never on the critical path.
- Images: run `npm run optimize-images` (part of build) to emit AVIF/WebP next to PNG sources.

## Related docs

- [site-generation.md](./site-generation.md) — SSG + `interactive` semantics
- [adding-a-page.md](./adding-a-page.md) — checklist includes “does this route need hydration?”
# Content images

Database-backed articles receive image dimensions and rendition metadata from the public content
API. Render them as a `<picture>` with explicit `width` and `height`, lazy loading, async decoding,
and modern formats before the original. Do not infer dimensions by downloading the image at render
time; the legacy local-image dimensions map applies only to file-based content.
