# MC.7 — www Build-Time Content Integration (SSG from the API)

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Architecture
> decision 1; [www/docs/site-generation.md](../../../www/docs/site-generation.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.7 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | BLOCKER (for the program) |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING — `www` loads content via `import.meta.glob` from its own `src/blog` and `src/docs` |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform |
| **Depends on** | MC.3, MC.6 |
| **Unblocks** | MC.8, MC.12, MC.13, MC.15 |

---

## 1. Problem Statement

`www` is a statically generated site whose content is compiled into the bundle: `blog.ts` and
`docs.ts` eagerly glob markdown at build time, the route manifest enumerates `/blog/*` and `/docs/*`
from those globs, and `generate-site.mjs` renders each path to HTML. If content moves to the database
and this pipeline does not change, nothing published in the workspace ever reaches the site. The
change must be surgical: the same rendering, the same route manifest contract, the same SEO
artefacts, the same failure tolerance the course prerenderer already has — with the source of the
content swapped behind an interface.

## 2. Goals

- Introduce a single content-source abstraction with two implementations (`files`, `api`) selected by
  `WWW_CONTENT_SOURCE`, so both paths can run side by side during MC.6's parity phase.
- Fetch published content from `/api/v1/public/content/*` at build time, with retries, a bounded
  concurrency pool and a previous-deploy fallback — exactly the policy `generate-site.mjs` already
  applies to `/courses/*`.
- Keep the route manifest as the single source of truth for routes; `enumerate()` for blog and docs
  reads the content index rather than a glob.
- Localise content images into `dist/` so a published page never fetches from the app origin.
- Preserve every SEO artefact: markdown siblings, sitemaps with honest `lastmod`, JSON-LD graphs,
  `llms.txt`/`llms-full.txt`, `_redirects` and the manifest CI assertions.
- Make a content-only publish able to rebuild the site without a code change (the trigger itself is
  [MC.8](MC.8-publish-pipeline-and-scheduling.md)).

## 3. Non-Goals

- No runtime/client-side fetching of article content on the public site. Pages stay static.
- No change to the visual design, page components or URL structure.
- No removal of `src/blog` / `src/docs` — that is [MC.15](MC.15-rollout-cutover-and-decommission.md).
- No change to file-based pages (legal, VPAT, glossary, comparisons, templates, standards) which stay
  in TypeScript modules.
- No incremental/partial deploys; GitHub Pages publishes a whole `dist/`.

## 4. Personas & User Stories

- **As a content expert**, I want an article I publish to appear on the public site without an
  engineer, so publishing is a content decision.
- **As a crawler**, I want the same fully rendered HTML at the same URL with the same metadata, so
  nothing regresses in indexing.
- **As a release engineer**, I want a build to succeed even when the content API is unreachable, so
  the marketing site is never blocked by an app outage.
- **As a web developer**, I want to run `npm run dev` offline without a database, so local work does
  not require the whole stack.

## 5. Functional Requirements

- **FR-1.** A content source interface MUST be introduced in `www/src/lib/content-source.ts`:
  `listArticles()`, `getArticle(path)`, `listCategories()`, `listAuthors()`, `listRedirects()`,
  returning the same shapes `BlogPostMeta` / `DocArticleMeta` / `BlogPost` / `DocArticle` expose
  today, so page components are unchanged.
- **FR-2.** `WWW_CONTENT_SOURCE ∈ {files, api}` MUST select the implementation, defaulting to `files`
  until MC.15 flips it. `CONTENT_API_BASE` MUST default to `API_BASE`
  (`https://self.lextures.com`).
- **FR-3.** The `api` implementation MUST fetch `/api/v1/public/content/index` once per build and
  fetch bodies only for articles whose `contentHash` is absent from the local cache.
- **FR-4.** Fetches MUST send `User-Agent: lextures-www-prerender/<version>`, retry up to 3 times
  with backoff, and use the existing bounded pool (`GENERATE_CONCURRENCY`, default 8).
- **FR-5.** On content API failure the build MUST log a `WARN`, reuse the previous deploy's HTML for
  every content path discoverable from the live sitemaps, and exit `0` — never fail the build
  (matching the course policy in `generate-site.mjs`).
- **FR-6.** A build cache directory (`www/.content-cache/`, gitignored, restored via `actions/cache`)
  MUST store fetched bodies keyed by `contentHash`, so a rebuild triggered by one publish transfers
  one article, not 75.
- **FR-7.** The route manifest's `/blog/:slug`, `/blog` (with pagination if present), `/docs`,
  `/docs/:category` and `/docs/:category/:slug` descriptors MUST enumerate from the content source,
  preserving today's titles, descriptions and `lastmod` semantics.
- **FR-8.** SSR data MUST carry the article payload (`SsrData.article`, `SsrData.articleIndex`) so
  `entry-server.tsx` renders content without a second lookup, and the hydrated client render matches.
- **FR-9.** `lastmod` MUST come from `contentUpdatedAt ?? publishedAt` for DB-sourced content, and
  MUST continue to use git for file-sourced pages; `resolveLastmod()` in `seo-artifacts.mjs` MUST be
  extended, not replaced.
- **FR-10.** Markdown siblings (`dist/blog/x.md`, `dist/docs/cat/x.md`) MUST be emitted from the
  fetched body, byte-identical to what the file build emits today (front matter reconstructed from
  metadata in a stable key order).
- **FR-11.** Content images MUST be downloaded to `dist/assets/content/{checksum}/{rendition}.{ext}`
  and URLs in rendered HTML rewritten to those paths; a missing AVIF/WebP rendition MUST be generated
  locally with `sharp` (already a dependency).
- **FR-12.** `dist/_redirects` MUST include rows from `/api/v1/public/content/redirects` merged with
  the existing static redirect list, with static entries winning on conflict.
- **FR-13.** `noindex` articles MUST be generated as pages with `robots: noindex` and excluded from
  sitemaps and `llms.txt`.
- **FR-14.** After a successful deploy the workflow MUST POST the generated route list to
  `/api/v1/admin/marketing/known-paths` (MC.4 FR-9) using a scoped service token, and MUST not fail
  the deploy if that call fails.
- **FR-15.** `npm run dev` MUST work with `WWW_CONTENT_SOURCE=files` and with `api` against a local
  server; with `api` and no server reachable it MUST fall back to the cache and then to an empty
  content set with a visible console warning, never a crash.
- **FR-16.** All existing CI assertions in `.github/workflows/pages-www.yml` MUST keep passing, and
  new ones MUST be added: at least one DB-sourced blog and one DB-sourced doc page exist with `<h1>`,
  canonical and description; the markdown sibling exists; the content-source mode is printed in the
  build log.
- **FR-17.** The generator MUST print a content summary (`source`, `articles fetched`, `cache hits`,
  `fallback used`) so a build log answers "where did this content come from".

## 6. Non-Functional Requirements

- **Performance** — Content fetch adds < 20 s to a cold build and < 5 s to a warm (cached) build.
  Total `www` build stays under the current CI budget; the Lighthouse and perf-budget jobs are
  unchanged and must still pass.
- **Security** — The build reads a public API anonymously. The `known-paths` POST uses a scoped
  service token stored as a GitHub secret; it grants only `…:admin` on that route. No secrets are
  embedded in `dist/`. Fetched markdown is rendered by the same sanitizing renderer used today
  (`html: false`), so a malicious body cannot inject script into the static site.
- **Privacy & Compliance** — Only published content is fetched; drafts never enter `dist/`. Build
  logs must not include preview tokens.
- **Accessibility** — Generated pages keep the current heading, landmark and image semantics.
  Localised images preserve `alt`, `width` and `height` (MC.5 FR-11).
- **Scalability** — Linear in changed articles thanks to `contentHash` caching; the index request is
  a single ~250 KB payload at 500 articles.
- **Reliability** — Three-tier degradation: API → cache → previous deploy. Any tier produces a
  complete site; the build never emits a partial content set silently (the summary line and a
  non-zero `fallbackUsed` count make it visible, and MC.8 alerts on it).
- **Observability** — Build summary in logs; `dist/.seo-manifest.json` gains
  `contentSource`, `contentGeneratedAt` and `fallbackUsed` fields; the deploy workflow uploads the
  content summary as an artefact.
- **Maintainability** — One abstraction, two implementations, no conditional logic scattered through
  page components. `src/utils/blog.ts` and `src/utils/docs.ts` become thin adapters over the source.
- **Internationalization** — The source interface takes a `locale` parameter from day one; the `api`
  implementation passes it through, the `files` implementation ignores it (MC.14 uses it).
- **Backward compatibility** — With `WWW_CONTENT_SOURCE=files` the build is byte-identical to today
  (proved by MC.6's parity harness in both directions).

## 7. Acceptance Criteria

- **AC-1.** *Given* `WWW_CONTENT_SOURCE=files`, *when* `npm run build` runs, *then* the output is
  byte-identical to the current build (parity harness, MC.6 AC-5).
- **AC-2.** *Given* `WWW_CONTENT_SOURCE=api` and a reachable API with imported content, *when* the
  build runs, *then* every blog and docs URL present in the file build exists with the same title,
  description, canonical and `lastmod`.
- **AC-3.** *Given* the content API returns `503`, *when* the build runs, *then* it logs a WARN,
  reuses previous-deploy HTML for content paths, exits `0`, and `.seo-manifest.json` records
  `fallbackUsed: true`.
- **AC-4.** *Given* a warm `.content-cache`, *when* one article changes, *then* the build fetches
  exactly one body and the summary reports 1 fetch / N-1 cache hits.
- **AC-5.** *Given* an article with `noindex: true`, *when* the build runs, *then* its page exists
  with a `noindex` robots meta and it is absent from every sitemap and from `llms.txt`.
- **AC-6.** *Given* a DB-sourced doc page, *when* generated, *then* `dist/docs/{cat}/{slug}.md` exists
  and its front matter round-trips to the same values the API returned.
- **AC-7.** *Given* an article referencing a media asset, *when* the build runs, *then* the image
  exists under `dist/assets/content/{checksum}/`, the HTML references the local path, and AVIF, WebP
  and original renditions are present with `width`/`height` on the `<img>`.
- **AC-8.** *Given* a redirect row `from=/docs/old` `to=/docs/new`, *when* the build runs, *then*
  `dist/_redirects` contains that row with status 301.
- **AC-9.** *Given* a successful deploy, *when* the workflow completes, *then*
  `/api/v1/admin/marketing/known-paths` has been called with the generated route list; *and given*
  that call fails, *then* the deploy still succeeds.
- **AC-10.** *Given* `npm run dev` with `WWW_CONTENT_SOURCE=api` and no server, *when* the dev server
  starts, *then* it serves the site with an empty content set and a console warning, without
  crashing.
- **AC-11.** *Given* the CI workflow, *when* it runs on a DB-sourced build, *then* all existing
  crawlability assertions pass plus the new DB-sourced page assertions (FR-16).
- **AC-12.** *Given* a build, *when* the log is read, *then* one line states source, fetched count,
  cache hits and fallback status.

## 8. Data Model

No database changes. Build-side artefacts:

- `www/.content-cache/{contentHash}.json` — cached article payloads (gitignored).
- `dist/.seo-manifest.json` — new fields `contentSource`, `contentGeneratedAt`, `fallbackUsed`,
  `contentArticleCount`.
- `dist/assets/content/{checksum}/{rendition}.{ext}` — localised media.
- `www/scripts/parity-allowlist.json` — shared with MC.6.

## 9. API Surface

Consumes MC.3 only:

```
GET {CONTENT_API_BASE}/api/v1/public/content/index
GET {CONTENT_API_BASE}/api/v1/public/content/articles/blog/{slug}
GET {CONTENT_API_BASE}/api/v1/public/content/articles/docs/{category}/{slug}
GET {CONTENT_API_BASE}/api/v1/public/content/categories
GET {CONTENT_API_BASE}/api/v1/public/content/authors
GET {CONTENT_API_BASE}/api/v1/public/content/redirects
GET {CONTENT_API_BASE}/api/v1/public/content/media/{id}/{rendition}.{ext}
```

Produces one write:

```
POST {API_BASE}/api/v1/admin/marketing/known-paths   # deploy job, service token, best-effort
```

New environment variables (documented in `www/docs/site-generation.md`):

| Variable | Default | Purpose |
|---|---|---|
| `WWW_CONTENT_SOURCE` | `files` (→ `api` at MC.15) | content source implementation |
| `CONTENT_API_BASE` | `API_BASE` | content API origin |
| `CONTENT_CACHE_DIR` | `.content-cache` | build cache location |
| `CONTENT_KNOWN_PATHS_TOKEN` | — | service token for the post-deploy path sync |

## 10. UI / UX

No visual change is permitted. The user-facing surfaces this plan must leave identical:

- `/blog` index (cards, pillar filters, pagination), `/blog/{slug}` (byline, dates, citations, FAQ),
  `/docs` index, `/docs/{category}`, `/docs/{category}/{slug}` (breadcrumbs, related links,
  freshness stamp), author pages.
- Empty/loading/error states: SSG has none at runtime; the *dev* experience gains a console warning
  and an empty-index banner in the local dev overlay only.
- Responsive and accessibility behaviour unchanged; the parity harness protects both by diffing HTML.

## 11. AI / ML Considerations

Not AI-touching. Indirectly relevant: `llms.txt` and `llms-full.txt` are generated from the content
set, so the source swap must keep those artefacts complete — asserted by the parity harness and by
MC.12.

## 12. Integration Points

- **`www` modules:** `src/lib/content-source.ts` (new), `src/utils/blog.ts`, `src/utils/docs.ts`,
  `src/lib/route-manifest.tsx`, `src/lib/ssr-data.ts`, `src/entry-server.tsx`,
  `scripts/generate-site.mjs`, `scripts/seo-artifacts.mjs`, `scripts/generate-docs-search.mjs`
  (MC.13), `scripts/generate-help-center.mjs`, `scripts/check-help-freshness.mjs`.
- **CI:** `.github/workflows/pages-www.yml` — env, cache step, new assertions, post-deploy
  known-paths call.
- **Server:** MC.3 read API; MC.4 known-paths write.
- **External:** GitHub Pages (unchanged), `sharp` (rendition fill-in).

## 13. Dependencies & Sequencing

- Must ship after: MC.3 (API), MC.6 (content to fetch — at least on staging).
- Must ship before: MC.8 (a rebuild trigger is pointless without this), MC.12, MC.13, MC.15.
- Shared infra: GitHub Actions cache; a staging API reachable from CI.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Build becomes dependent on API availability | M | **H** | Three-tier degradation (FR-5/FR-6) inherited from the course prerenderer, asserted by AC-3 |
| Subtle HTML differences slip through | M | H | MC.6 parity harness diffs byte-for-byte in both directions before the flip |
| `lastmod` semantics diverge between sources | M | H | `resolveLastmod` extended with an explicit source precedence and unit tests per source (AC-2) |
| Cache poisoning by a stale `contentHash` | L | M | Hash covers body + rendering-relevant metadata (MC.3 FR-4); cache key includes the renderer version, so a renderer change invalidates everything |
| Route manifest enumeration becomes async | **H** | M | `enumerate()` is currently synchronous; the generator pre-loads the content index before rendering and passes it through module-level state, keeping the manifest API synchronous (documented in `adding-a-page.md`) |
| Image localisation bloats `dist/` | M | L | Only referenced renditions are downloaded; checksum paths dedupe across articles |
| Dev experience degrades for engineers | M | M | `files` remains the default for local dev; `api` is opt-in |

## 15. Rollout Plan

- **Feature flag:** build-time `WWW_CONTENT_SOURCE` (not a DB flag). Default `files`; staging runs
  `api`; production flips in MC.15.
- **Sequencing:** abstraction + files implementation (no behaviour change) → api implementation →
  cache → media localisation → CI env/assertions → staging build on `api` → parity green.
- **Dogfood:** staging site (`ROBOTS_DISALLOW_ALL=1`) serves DB-sourced content for at least one
  week; content team publishes two real articles through it.
- **GA criteria:** parity harness green; CI assertions pass on an `api` build; fallback path
  exercised deliberately (kill the API, build, verify exit 0 and reused HTML).
- **Rollback:** set `WWW_CONTENT_SOURCE=files` in the workflow and redeploy — one line, no code
  revert, and the files are still in the repo until MC.15.

## 16. Test Plan

- **Unit** — content-source interface conformance suite run against both implementations; front
  matter reconstruction for markdown siblings; `lastmod` precedence; cache key derivation; redirect
  merge precedence; `noindex` exclusion from sitemaps.
- **Integration** — `generate-site.test.mjs` extended: build with a stubbed API server (fixtures),
  assert generated HTML, manifest fields, siblings, `_redirects`, localised media; simulate 503 and
  assert fallback; simulate partial failure (index ok, one body 500).
- **End-to-end** — CI builds the site from the staging API and runs the existing crawlability
  assertion block plus the new DB-sourced assertions; Lighthouse and perf-budget jobs must pass
  unchanged.
- **Security** — assert no token appears in `dist/` or logs; assert rendered content cannot contain
  raw HTML (renderer config test); verify the known-paths call uses a scoped token.
- **Accessibility** — axe run over one DB-sourced blog page and one docs page in the built output.
- **Performance / load** — measure cold vs warm build time; enforce a CI budget (+20 s cold max);
  `check-perf-budget.mjs` unchanged and still green.
- **Manual exploratory** — publish an article in staging, trigger a build, confirm it appears with
  correct metadata, image, and sitemap entry.

## 17. Documentation & Training

- `www/docs/site-generation.md` — new "Content source" section: env vars, failure policy, cache,
  known-paths sync.
- `www/docs/adding-a-page.md` — clarify that blog/docs pages are no longer added as files once
  `api` is the source; the manifest entry stays for the *route family*.
- `www/docs/contributor-guide.md` — how to run the site against local content.
- Runbook: "Marketing site build used fallback content" — how to detect (manifest field, alert from
  MC.8) and what to do.

## 18. Open Questions

1. Should the build fail (rather than warn) when fallback is used on a *scheduled content publish*
   build, since the whole point of that build was to ship an article? (Proposed: warn + alert via
   MC.8; failing gains nothing because the previous site is still correct.)
2. Do we keep serving markdown siblings for every article, or only for docs? (Proposed: keep exactly
   today's behaviour — `shouldEmitMarkdownSibling()` decides.)
3. Should `.content-cache` be committed for reproducible builds? (Proposed: no; use Actions cache.)
4. Does the dev server need hot-reload on content changes when running against a local API?
   (Proposed: no; a manual refresh is acceptable.)

## 19. References

- Files this work touches: `www/src/lib/content-source.ts`, `www/src/utils/{blog,docs}.ts`,
  `www/src/lib/route-manifest.tsx`, `www/src/entry-server.tsx`, `www/src/lib/ssr-data.ts`,
  `www/scripts/generate-site.mjs`, `www/scripts/seo-artifacts.mjs`,
  `.github/workflows/pages-www.yml`.
- Precedent: the course prerender path in `generate-site.mjs` (fetch, retry, fallback, concurrency).
- Related plans: [MC.3](MC.3-public-content-read-api.md),
  [MC.6](MC.6-markdown-to-database-migration.md),
  [MC.8](MC.8-publish-pipeline-and-scheduling.md), [MC.12](MC.12-seo-parity-from-database.md),
  [MC.15](MC.15-rollout-cutover-and-decommission.md).
