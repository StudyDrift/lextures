# Marketplace SEO prerender (plan MKT10)

The www marketing site is a static SPA on GitHub Pages. Course pages are made crawlable by a **build-time prerender** step that runs after `vite build`.

## How it works

`npm run build` runs:

```
tsc -b && vite build && node scripts/prerender-courses.mjs
```

The prerender script:

1. Fetches all listed+published courses from `GET {API_BASE}/api/v1/public/marketplace/courses` (paginated).
2. For each course, fetches detail (including server-built `jsonLd`) and writes `dist/courses/<slug>/index.html` with per-page `<title>`, description, canonical, OG/Twitter tags, and Course JSON-LD.
3. Writes `dist/courses/index.html` for the storefront.
4. Writes `dist/sitemap.xml` (static routes + every course URL) and `dist/robots.txt`.

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `API_BASE` / `VITE_API_BASE_URL` | `https://self.lextures.com` | Public marketplace API origin |
| `SITE_ORIGIN` | `https://lextures.com` | Canonical / sitemap origin |
| `SKIP_COURSE_PRERENDER=1` | unset | Escape hatch: write `/courses` + sitemap shell only; **do not** fail the build when the API is unreachable |

## Failure mode

If the API is unreachable and `SKIP_COURSE_PRERENDER` is unset, the build **fails loudly** so we never deploy empty/stale course pages silently.

## Freshness

New listings appear in the sitemap/prerender after the next www deploy. Prefer a **daily scheduled CI rebuild** (plus manual trigger) so Search Console stays current. Rebuild-on-publish webhook is a fast-follow.

## Search Console

After deploy, submit `https://lextures.com/sitemap.xml` in Google Search Console and monitor coverage for `/courses/*`.

## Runtime head updates

`CoursesPage` and `CourseDetailPage` call `useDocumentHead` so client-side navigation keeps `document.title`, meta, canonical, and JSON-LD in sync with the prerendered values (idempotent on hydrate).
