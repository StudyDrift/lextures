# Site generation & crawlability (SEO.1 + SEO.2 + SEO.3 + SEO.4)

The www marketing site is a Vite + React SPA. For search engines and AI crawlers (which do not run JavaScript), **every public URL is statically generated at build time** into real HTML under `dist/`.

## How it works

`npm run build` runs:

```
tsc -b && node scripts/optimize-images.mjs && vite build && node scripts/generate-site.mjs
```

`generate-site.mjs`:

1. Starts a Vite SSR server and loads `src/entry-server.tsx` + `src/lib/route-manifest.tsx`.
2. Expands the **route manifest** into concrete paths (static pages, `/blog/*`, `/docs/*`, `/courses/*`, plus published translations at `/{locale}/blog/*` and `/{locale}/docs/*`).
3. For each path, calls `renderToString` so `#root` contains headings and body copy **without executing client JS**.
4. Injects per-page `<title>`, meta description, canonical, robots, OG/Twitter, JSON-LD `@graph`, and optional markdown alternate into `<head>`.
5. Validates JSON-LD graphs (absolute `@id`, no dangling refs, ≤12 KB) and fails the build on errors (SEO.3).
6. Writes HTML under `dist/`, plus SEO artefacts (manifest, sitemap index, robots, llms.txt, IndexNow key, markdown siblings).

## Route manifest

**Source of truth:** `www/src/lib/route-manifest.tsx`

Every page is a `RouteDescriptor` with path, component, title, description, sitemap flag, optional robots, and (for dynamic families) `enumerate()`.

The client router in `src/app.tsx` resolves routes **only** from this manifest. Adding a page means adding a manifest entry — not a new `if` branch.

See [adding-a-page.md](./adding-a-page.md).

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `API_BASE` / `VITE_API_BASE_URL` | `https://self.lextures.com` | Public marketplace API origin |
| `SITE_ORIGIN` | `https://lextures.com` | Canonical / sitemap origin |
| `COURSE_CACHE_URL` | same as `SITE_ORIGIN` | Origin used to reuse previous course HTML on API failure |
| `GENERATE_CONCURRENCY` | `8` | Bounded pool for course detail fetches / renders |
| `ROBOTS_DISALLOW_ALL` | auto | `1` forces staging robots; also auto when `SITE_ORIGIN` ≠ production |
| `CONTENT_API_BASE` | — | Public content API origin used by API-sourced builds and parity checks |

## Course pages & API failure

Course listing and detail are fetched from:

- `GET {API_BASE}/api/v1/public/marketplace/courses`
- `GET {API_BASE}/api/v1/public/marketplace/courses/{slug}`

Requests send `User-Agent: lextures-www-prerender/<version>` and retry with backoff (max 3).

If the marketplace API is unreachable:

1. The build logs a **WARN** and continues (exit 0).
2. The generator tries to reuse `/courses/<slug>/` HTML from the live site (`COURSE_CACHE_URL` / `SITE_ORIGIN`) via section/index sitemaps.
3. Static marketing pages still ship fully rendered.

`SKIP_COURSE_PRERENDER` is **removed** — do not reintroduce it in CI.

## Artefacts

| File | Purpose |
|---|---|
| `dist/.seo-manifest.json` | Machine-readable list of every generated URL + `schemaTypes[]` + sitemap stats |
| `dist/.course-cache.json` | Slugs from the last successful course fetch |
| `dist/sitemap.xml` | **Sitemap index** referencing section files |
| `dist/sitemaps/*.xml` | Section urlsets (`pages`, `blog`, `docs`, `courses`, …); sharded at 50k |
| `dist/robots.txt` | Generated from `src/lib/crawler-policy.ts` |
| `dist/llms.txt` | Curated agent map (`src/lib/llms-catalog.ts`) |
| `dist/llms-full.txt` | Concatenated help + blog markdown (≤5 MB) |
| `dist/**/*.md` | Plain-text siblings for `/docs/*` and `/blog/*` content |
| `dist/{indexnow-key}.txt` | Public IndexNow key file |
| `dist/_redirects` | Cloudflare Pages / Netlify redirects |
| `dist/_headers` | Optional edge headers (Content-Type / X-Robots-Tag for `.md`) |
| `dist/404.html` | Real 404 with `noindex` and hub links |

### lastmod rules (SEO.2)

Resolution order: front-matter `updated` → front-matter `date` → git commit date of source → course `updatedAt`/`createdAt` → **omit**.

Build date is never used. Two consecutive deploys with no content change must not rewrite `lastmod` values.

### Parity gate

The build fails if any sitemap URL is missing from the indexable set in `.seo-manifest.json`, or vice versa. `robots: noindex` URLs never appear in sitemaps.

To compare file- and API-sourced builds after importing content, run:

```bash
npm run content:parity -- --api-base https://staging.self.lextures.com
```

The command compares generated article HTML, `.seo-manifest.json`, section sitemaps,
LLM catalogues, and markdown siblings. Unexpected byte differences exit `1`; build failures
exit `2`. Intentional differences must be listed in `scripts/parity-allowlist.json` with a reason.

## Crawler policy & IndexNow

See [crawler-policy.md](./crawler-policy.md) for:

- Named AI/search agents and the three crawler jobs
- How to add/remove an agent
- GSC / Bing verification runbook
- Post-deploy IndexNow submission (`npm run submit-indexnow`)

## Runtime hydration & islands (SEO.4)

Each `RouteDescriptor` may set `interactive`:

| `interactive` | Client JS | Behaviour |
|---|---|---|
| `false` (default for legal, blog, docs, about, authors, static marketing) | `static-island-*.js` only (~nav + deferred analytics + web-vitals) | No React hydration; HTML is complete |
| `true` (home, calculator, courses, forms) | main entry + route chunk via `import()` | `hydrateRoot` when `#root` has server HTML |

`src/main.tsx` skips React when `data-interactive="false"` on `<html>`. Page modules load from `src/lib/route-pages.ts` (dynamic import per route).

`window.__LEXTURES_SSR__` carries build-time course payloads so the first client render matches the prerendered tree (interactive routes only).

Performance budgets: [performance-budget.md](./performance-budget.md).

`useDocumentHead` (via `App`) keeps title/meta/canonical (and markdown alternate) in sync on client navigation.

## Failure modes

| Situation | Behaviour |
|---|---|
| Empty body / missing title / desc > 160 / duplicate title | Build **fails** (non-zero exit) |
| JSON-LD missing `@id` / non-absolute `@id` / dangling ref / >12 KB | Build **fails** (SEO.3) |
| Unknown blog/docs author slug | Module load / build **fails** (SEO.3) |
| Sitemap ↔ manifest mismatch | Build **fails** naming the URL |
| Marketplace API down | WARN + previous-deploy course reuse; site still deploys |

## Public content API

Blog and docs content is always prerendered from the public API. `CONTENT_API_BASE` falls back to `API_BASE`, and `CONTENT_CACHE_DIR` defaults
to `.content-cache`. The API loader retries three times, honors `GENERATE_CONCURRENCY`, fetches only
hashes absent from the cache, and logs source/fetch/cache/fallback counts. On failure it uses the
cached index and bodies, or an empty content set when no cache exists. Media is localized under
`dist/assets/content`, and API redirects are merged with static redirects (static wins).

After deployment, `scripts/sync-known-paths.mjs` best-effort posts generated paths using
`CONTENT_KNOWN_PATHS_TOKEN`; missing credentials or request failures do not fail deployment.

Published, indexable blog articles also generate RSS 2.0 at `/blog/feed.xml` and JSON Feed 1.1 at `/blog/feed.json` (20 newest entries). Blog pages advertise both feeds. The SEO manifest records `contentSource`, `articleCount`, `feedItemCount`, and `fallbackUsed`; deployment fails when the indexable sitemap count drops by more than 10% without a recorded fallback.

Database-backed marketing content is available anonymously under
`/api/v1/public/content` when `ff_marketing_content` is enabled. Builds should fetch
`/index` first, compare each article's 16-character `contentHash`, and fetch only changed
blog or docs detail routes. `/index` also supplies sitemap dates, categories, active authors,
and redirects. Published responses support strong ETags and are cacheable for 60 seconds;
send `If-None-Match` on repeated builds.

If the content API is unavailable, keep the previous deployment's generated article HTML and
log a warning. Draft preview links are the exception: they use a short-lived
`preview_token`, are `no-store`, and must never be persisted into build caches or sitemaps.

Structured data details: [structured-data.md](./structured-data.md). Author bylines: [authoring-bylines.md](./authoring-bylines.md).
| IndexNow / Google ping failure | WARN only — deploy still succeeds |
| Unknown path in production | Host serves `404.html` (`noindex`) |

## Related

- [adding-a-page.md](./adding-a-page.md)
- [crawler-policy.md](./crawler-policy.md)
- Plans: [SEO.1](../../docs/completed/seo/SEO.1-static-rendering-and-crawlability.md), [SEO.2](../../docs/completed/seo/SEO.2-crawler-access-sitemaps-and-llms-txt.md)
