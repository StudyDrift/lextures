# MC.3 — Public Content Read API & Caching

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Plans.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.3 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING — `www` reads markdown from its own repo; no content endpoint exists |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Server platform |
| **Depends on** | MC.1 (MC.2 for preview tokens) |
| **Unblocks** | MC.7, MC.12, MC.13, MC.14 |

---

## 1. Problem Statement

The marketing site build has no way to learn what content exists. `generate-site.mjs` already knows
how to fetch a catalog from the API and survive its absence — it does exactly that for `/courses/*`
against `/api/v1/public/marketplace/*` — but there is no equivalent for articles. Without an
anonymous, cacheable, crawler-safe read surface that enumerates published content and returns its
markdown and metadata, `www` cannot be cut over from files, and neither the docs search index nor the
in-app help widget can share one source of truth.

## 2. Goals

- Publish an anonymous read API that mirrors the marketplace public-API conventions (shape, error
  handling, caching, rate limiting) so there is one pattern to learn.
- Return content in a form the existing `www` renderer consumes unchanged: raw markdown plus typed
  metadata — not pre-rendered HTML.
- Give the build a cheap enumeration endpoint (`/index`) that supplies path, lastmod and a content
  hash for every published item, so incremental builds and sitemap `lastmod` are honest.
- Support draft preview through short-lived signed tokens without ever making drafts crawlable.
- Be fast and boring: served from the object cache, ETag-validated, safe to hit from CI on every
  build.

## 3. Non-Goals

- No write operations (MC.2 owns those).
- No HTML rendering for `www` (the site renders markdown itself; MC.4's Go renderer serves preview
  and search excerpts only).
- No RSS/Atom or `llms.txt` generation — those are build artefacts produced by `www`
  ([MC.12](MC.12-seo-parity-from-database.md)).
- No CDN or edge configuration change; `self.lextures.com` serves these routes directly.
- No authenticated per-user personalization; every response is identical for every caller.

## 4. Personas & User Stories

- **As the `www` build**, I want one request that lists every published path with a lastmod, so I can
  enumerate routes and set sitemap dates without fetching every body.
- **As the `www` build**, I want a body endpoint that returns markdown and metadata in one payload,
  so page generation is a single round trip per article.
- **As a content expert**, I want to preview an unpublished article at a real URL shape, so I can
  check it before it goes live.
- **As an SRE**, I want these endpoints to be cheap and cacheable, so a CI build storm cannot degrade
  the app for signed-in users.
- **As a security reviewer**, I want unpublished content to be unreachable without a token that
  expires, so a leaked link does not leak our roadmap.

## 5. Functional Requirements

- **FR-1.** Routes MUST be registered under `/api/v1/public/content/` and MUST be anonymous
  (no session, no CSRF), consistent with `registerPublicMarketplaceRoutes`.
- **FR-2.** When `ff_marketing_content` is off, every route MUST return `404` with the standard
  problem body.
- **FR-3.** `GET /index` MUST return every `published` article as
  `{ path, kind, slug, locale, categorySlug, title, description, updatedAt, publishedAt,
  contentHash, noindex }`, plus `categories[]`, `authors[]` and `generatedAt`. It MUST NOT include
  bodies.
- **FR-4.** `contentHash` MUST be a stable hash (SHA-256, hex, first 16 chars) over body + metadata
  that affect rendering, so a build can skip unchanged pages.
- **FR-5.** `GET /articles` MUST support `kind`, `locale`, `category`, `author`, `tag`, `q`,
  `limit` (default 50, max 200) and `cursor`, returning metadata pages sorted by `publishedAt DESC`.
- **FR-6.** `GET /articles/blog/{slug}` and `GET /articles/docs/{category}/{slug}` MUST return the
  full article: metadata plus `bodyMd`. A `?render=html` parameter MAY return sanitized HTML from the
  MC.4 renderer for non-`www` consumers (the in-app help widget).
- **FR-7.** `GET /categories` MUST return help categories ordered by `sort_order`, each with a
  published article count.
- **FR-8.** `GET /authors` and `GET /authors/{slug}` MUST return public author profiles; retired
  authors MUST be excluded from the list and MUST return `404` on detail (matching today's
  `isAuthorLinkable` policy).
- **FR-9.** `GET /redirects` MUST return all redirect rows so the build can emit `dist/_redirects`.
- **FR-10.** `GET /search?q=` MUST run Postgres FTS over published content, returning ranked
  `{ path, title, description, snippet, kind }`, `limit` default 10 / max 50.
- **FR-11.** Preview: any detail route MUST accept `?preview_token=…`; with a valid, unexpired token
  bound to that article it MUST return the draft (current or a specific revision), and MUST set
  `Cache-Control: no-store` and `X-Robots-Tag: noindex, nofollow`. An invalid/expired token MUST
  return `403`; a missing token on an unpublished article MUST return `404` (no existence oracle).
- **FR-12.** Published responses MUST set `Cache-Control: public, max-age=60,
  stale-while-revalidate=300` and a strong `ETag`; the server MUST honour `If-None-Match` with `304`.
- **FR-13.** Responses MUST be served through `internal/objectcache` when Redis is configured, keyed
  by route + query + content version, and MUST be invalidated on publish/unpublish (MC.8).
- **FR-14.** Routes MUST be rate-limited per IP (default 600 req/min, burst 60) via
  `internal/ratelimit`, with a documented allowance for the build's `User-Agent`.
- **FR-15.** All routes MUST appear in `server/internal/publicapi/openapi.json` and the route
  inventory.
- **FR-16.** Responses MUST NOT include `draft`, `in_review`, `changes_requested`, `scheduled`
  (before its time), `archived` or soft-deleted content under any parameter combination.
- **FR-17.** CORS MUST allow `GET` from any origin (the marketing site and its previews are separate
  origins), matching the marketplace routes.

## 6. Non-Functional Requirements

- **Performance** — `/index` p95 < 120 ms cold, < 20 ms cached, payload < 250 KB at 500 articles
  (gzip). Detail p95 < 60 ms cached. `/search` p95 < 120 ms.
- **Security** — Anonymous but read-only; no parameter can widen visibility beyond `published`.
  Preview tokens are HMAC-verified, article-bound and expiring (MC.2 FR-15). No user input is
  interpolated into SQL (parameterized `plainto_tsquery`). Response headers include
  `X-Content-Type-Options: nosniff`.
- **Privacy & Compliance** — Only intentionally public content is served. Author profiles are public
  by design; a retired author's detail page 404s. No cookies, no tracking, so no consent surface.
- **Accessibility** — N/A (API). The `?render=html` output must satisfy MC.4's sanitizer, which
  preserves heading ids and figure/figcaption semantics used by the help widget.
- **Scalability** — Cacheable and stateless; a CI build performs ~1 + N requests where N is the
  number of changed articles when the build uses `contentHash` skipping.
- **Reliability** — Availability target 99.9% (same as the app). Degradation is tolerable because
  `www` falls back to previous-deploy HTML (MC.7); no user-facing page depends on this API at runtime
  except the in-app help widget, which already has an empty-state.
- **Observability** — Counters `marketing_content_public_requests_total{route,status}`,
  `…_cache_hits_total`, `…_preview_requests_total`; histogram of payload size for `/index`; alert if
  `/index` p95 > 500 ms for 10 min.
- **Maintainability** — `internal/httpserver/public_marketing_content_http.go`, modelled 1:1 on
  `public_marketplace_http.go`; DTO structs live beside the handler; no SQL in handlers.
- **Internationalization** — `locale` filter on every list route; `Content-Language` header set on
  detail responses; `/index` groups translations by `translationGroupId` for MC.14 hreflang.
- **Backward compatibility** — New namespace. `/api/v1/public/marketplace/*` is untouched. Response
  fields are additive-only once shipped; removals require a version bump of the whole namespace.

## 7. Acceptance Criteria

- **AC-1.** *Given* `ff_marketing_content` is off, *when* `GET /api/v1/public/content/index` is
  called, *then* the response is `404` (test mirrors `public_marketplace_nodb_test.go`).
- **AC-2.** *Given* 3 published and 4 unpublished articles, *when* `/index` is called, *then* exactly
  the 3 published paths are returned and no draft slug appears anywhere in the payload.
- **AC-3.** *Given* an article whose body changes, *when* `/index` is called again, *then* its
  `contentHash` changes; *given* only its `updated_at` touch, *then* the hash is unchanged.
- **AC-4.** *Given* a published doc at `/docs/courses/finding-your-course`, *when*
  `GET /articles/docs/courses/finding-your-course` is called, *then* `bodyMd` equals the stored
  markdown byte-for-byte and metadata matches the DB row.
- **AC-5.** *Given* the same request repeated with `If-None-Match` set to the returned ETag, *when*
  called, *then* the response is `304` with an empty body.
- **AC-6.** *Given* a draft article, *when* its detail path is requested without a token, *then* the
  response is `404`; *with* a valid preview token, *then* `200` with `Cache-Control: no-store` and
  `X-Robots-Tag: noindex, nofollow`.
- **AC-7.** *Given* a preview token that expired 1 second ago, *when* used, *then* the response is
  `403` with code `preview_token_expired`.
- **AC-8.** *Given* `?q=rubric`, *when* `/search` is called, *then* results are ranked by
  `ts_rank_cd`, contain a highlighted snippet, and exclude unpublished content.
- **AC-9.** *Given* a retired author, *when* `/authors` is listed, *then* they are absent; *when*
  `/authors/{slug}` is fetched, *then* `404`.
- **AC-10.** *Given* 700 requests in one minute from one IP, *when* the limit is 600, *then* the
  excess receives `429` with `Retry-After`.
- **AC-11.** *Given* an article is published via MC.2, *when* the next `/index` request arrives,
  *then* the cached payload has been invalidated and the new path appears within 5 seconds.
- **AC-12.** *Given* the OpenAPI spec, *when* `make openapi-check` runs, *then* every route in this
  plan is present with request/response schemas.

## 8. Data Model

No schema changes. Read-only projections over MC.1 tables:

- `PublishedArticleIndexRow` — computed by a single query joining articles + categories + authors,
  filtered `status='published' AND deleted_at IS NULL AND (noindex = FALSE OR include_noindex)`.
- `contentHash` — computed in SQL (`encode(sha256(convert_to(body_md || metadata_digest, 'utf8')),
  'hex')`) or in Go; Go is preferred so the digest definition lives with the DTO and is unit-tested.
- One covering index may be added if profiling shows a sort spill:
  `idx_mc_articles_public_index ON marketing.content_articles (locale, kind, published_at DESC)
  INCLUDE (slug, path, title, description, updated_at) WHERE status='published' AND deleted_at IS NULL`
  — migration `478_marketing_content_public_index.sql` (indicative number).

## 9. API Surface

Base `/api/v1/public/content`. All `GET`, all anonymous.

| Path | Purpose |
|---|---|
| `/index` | full published manifest (paths, lastmod, hashes, categories, authors) |
| `/articles?kind=&locale=&category=&author=&tag=&q=&limit=&cursor=` | metadata list |
| `/articles/blog/{slug}` | blog detail (`?render=html`, `?preview_token=`) |
| `/articles/docs/{category}/{slug}` | help article detail |
| `/categories?locale=` | help categories + counts |
| `/authors` · `/authors/{slug}` | public author profiles |
| `/redirects` | redirect table for `dist/_redirects` |
| `/search?q=&kind=&limit=` | FTS |

```ts
type PublicArticle = {
  path: string; kind: 'blog' | 'doc'; slug: string; locale: string
  translationGroupId: string
  categorySlug: string | null; categoryTitle: string | null
  title: string; description: string
  bodyMd: string                 // detail only
  html?: string                  // only with ?render=html
  author: { slug: string; name: string; jobTitle: string } | null
  reviewer: { slug: string; name: string } | null
  publishedAt: string; updatedAt: string; contentUpdatedAt: string | null
  reviewedAt: string | null
  primaryQuestion: string; cluster: string; pillar: string
  keywords: string[]; relatedTo: string[]; roles: string[]; segments: string[]
  citations: string[]; tags: string[]
  heroImageUrl: string | null
  noindex: boolean; canonicalOverride: string | null
}

type ContentIndex = {
  generatedAt: string
  articles: Array<Omit<PublicArticle, 'bodyMd' | 'html'> & { contentHash: string }>
  categories: Array<{ slug: string; title: string; description: string; sortOrder: number;
                      platformPath: string; articleCount: number }>
  authors: Array<{ slug: string; name: string; jobTitle: string; bio: string; knowsAbout: string[] }>
  redirects: Array<{ from: string; to: string; statusCode: number }>
}
```

- **Rate limits:** 600 req/min/IP; `429` includes `Retry-After`.
- **Caching:** `ETag` (strong, over the serialized payload), `Cache-Control: public, max-age=60,
  stale-while-revalidate=300`, `Vary: Accept-Encoding`.
- **WebSocket:** none.
- **OpenAPI:** added to `server/internal/publicapi/openapi.json`.

## 10. UI / UX

No UI. Consumer-facing behaviour this API must make possible:

- `www` build: enumerate → fetch changed → render (MC.7).
- In-app help widget (`clients/web/src/components/layout/help-widget.tsx`): search + article read
  with `?render=html` (MC.13), including loading skeleton, empty result copy and an offline error
  state that keeps the widget usable.
- Preview from the editor: opens `https://lextures.com/blog/{slug}?preview_token=…` in a new tab only
  after MC.7 teaches `www` to honour preview tokens client-side; before that, preview renders in-app
  (MC.10).

## 11. AI / ML Considerations

Not model-driven. One AI-adjacent consideration: this API is the surface AI crawlers will *not* use
(they read the static HTML), but it is the surface our own `llms-full.txt` generation reads. Payload
shape must therefore preserve full body text without truncation so MC.12 can build that artefact
without a second source.

## 12. Integration Points

- **Internal modules:** `internal/httpserver/public_marketing_content_http.go` (new),
  `internal/service/marketingcontent` (read methods), `internal/repos/marketingcontent`,
  `internal/objectcache`, `internal/ratelimit`, `internal/publicapi`, `internal/telemetry`.
- **Consumers:** `www/scripts/generate-site.mjs` (MC.7), `www/scripts/generate-docs-search.mjs`
  (MC.13), `clients/web` help widget (MC.13).
- **Events consumed:** cache invalidation on `marketingcontent.published/unpublished` from MC.2/MC.8.
- **External services:** none.

## 13. Dependencies & Sequencing

- Must ship after: MC.1. Preview (FR-11) additionally requires MC.2's token minting; ship the routes
  first and enable preview when MC.2 lands.
- Must ship before: MC.7 (build integration), MC.12 (SEO artefacts), MC.13 (search/help widget),
  MC.14 (locale routing).
- Shared infra: Redis (optional — degrades to direct DB reads).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| CI build storms hammer the API | M | M | `/index` + `contentHash` skipping means one large request and few detail requests; object cache; per-IP limit with a documented build allowance; build sets a distinct `User-Agent` |
| Draft leakage through a filter parameter | L | H | Status filter is applied in the repo layer, not the handler; a test enumerates every query-parameter combination and asserts no non-published row is ever returned |
| ETag churn from `updated_at`-only touches | M | L | ETag derives from `contentHash` + schema version, not `updated_at` (AC-3) |
| Payload growth as the help center reaches 60+ articles (SEO.7) | M | M | `/index` excludes bodies; gzip; cursor paging on `/articles`; alert on payload size histogram |
| Public API becomes an unintended content-scraping surface | M | L | It only exposes what is already published as HTML; rate limits and `robots` policy unchanged |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content` (OFF in production until MC.15).
- **Sequencing:** handlers + DTOs → object cache wiring → OpenAPI + route inventory → staging enable
  → verify against staging content imported by MC.6.
- **Dogfood:** point a staging `www` build (`SITE_ORIGIN` staging, `ROBOTS_DISALLOW_ALL=1`) at the
  staging API and confirm generated HTML matches the file-sourced build.
- **GA criteria:** ACs green; p95 targets met under the k6 profile; zero draft leakage in the
  enumeration test.
- **Rollback:** flag off → routes 404 → `www` continues to build from files (`WWW_CONTENT_SOURCE`
  default remains `files` until MC.15).

## 16. Test Plan

- **Unit** — DTO mapping; `contentHash` stability and sensitivity; ETag computation; preview-token
  verification (valid/expired/wrong-article/malformed); query-parameter validation and clamping.
- **Integration (DB)** — visibility matrix across all six statuses × soft-deleted × noindex; category
  counts; FTS ranking and snippet generation; `304` on `If-None-Match`; cache invalidation after a
  publish transition.
- **End-to-end** — a build-shaped script (`www` unit test with a stubbed fetch) proves the payload
  contract; full e2e lands in MC.7.
- **Security** — anonymous access confirmed for published only; parameter fuzzing for status/visibility
  escapes; SQL injection attempts through `q`, `category`, `cursor`; `429` behaviour; header
  assertions (`nosniff`, `X-Robots-Tag` on preview).
- **Accessibility** — `?render=html` output passes the sanitizer allowlist test (heading ids, alt
  text preserved) — full a11y coverage in MC.13.
- **Performance / load** — k6: 100 rps mixed for 5 min against 500 seeded articles; assert p95 and
  cache hit ratio > 90%.
- **Manual exploratory** — fetch every route by hand on staging; confirm a draft URL is a hard 404
  and the preview link works exactly once past expiry.

## 17. Documentation & Training

- `www/docs/site-generation.md` — new "Content API" section listing endpoints and the failure policy.
- Public API reference (`/api/openapi.json`) entries with examples.
- Internal runbook: "Content API is slow/down — what breaks" (answer: nothing user-facing until the
  next build; builds degrade to previous-deploy HTML).

## 18. Open Questions

1. Should `/index` support `?since=` so incremental builds transfer less? (Proposed: not yet —
   `contentHash` already avoids body transfers, and full manifests keep builds deterministic.)
2. Do we expose `bodyMd` for `noindex` articles in `/index`? (Proposed: include them in `/articles`
   and detail but mark `noindex: true`; `www` decides whether to emit them, per MC.12.)
3. Should the search endpoint be public at all, or only used by the in-app widget? (Proposed: public,
   since `www` may add client-side docs search fallback; rate limits cover abuse.)
4. Is a `Last-Modified` header needed in addition to `ETag` for build tooling simplicity?

## 19. References

- Files this work touches: `server/internal/httpserver/public_marketing_content_http.go`,
  `server/internal/publicapi/openapi.json`, `server/internal/service/marketingcontent/*`,
  `server/internal/repos/marketingcontent/*`.
- Precedents: `server/internal/httpserver/public_marketplace_http.go`,
  `server/internal/httpserver/public_marketplace_nodb_test.go`,
  `server/internal/objectcache`, `www/src/lib/marketplace-api.ts` (client shape to mirror).
- Related plans: [MC.7](MC.7-www-build-time-content-integration.md),
  [MC.12](MC.12-seo-parity-from-database.md), [MC.13](MC.13-docs-search-and-in-app-help.md),
  [MC.14](MC.14-localization-and-translations.md).
