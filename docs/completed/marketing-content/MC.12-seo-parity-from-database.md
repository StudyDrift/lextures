# MC.12 — SEO Parity from the Database

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Non-negotiable
> constraints; the SEO.1–SEO.4 foundation in [docs/completed/seo/](../../completed/seo/).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.12 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | BLOCKER (for the cutover) |
| **Markets** | K12 / HE / HS |
| **Status (today)** | COMPLETE — database-sourced content now drives structured data, sitemap/LLM artefacts, redirects, feeds, authors, canonicals, and social images with parity and CI guards |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform + SEO owner |
| **Depends on** | MC.3, MC.7 |
| **Unblocks** | MC.15 |

---

## 1. Problem Statement

SEO.1–SEO.4 turned a site where 28 of 31 sitemap URLs returned 404 into one where every page is real
HTML with a validated JSON-LD graph, honest `lastmod`, `llms.txt`, IndexNow submission and CI
assertions. All of it is fed by TypeScript modules that read markdown files. If content moves to the
database and the entity graph, canonicals, sitemaps, feeds and AI-crawler artefacts are not rebuilt
from the same data, the migration quietly undoes the most expensive work in the marketing programme.
This plan is the guarantee that it does not.

## 2. Goals

- Rebuild every SEO artefact from database-sourced content with **no** change in output: JSON-LD
  graphs, canonical/robots/OG/Twitter tags, sitemaps and `lastmod`, `llms.txt` / `llms-full.txt`,
  markdown siblings, `_redirects`, IndexNow submission.
- Move the author entity graph (SEO.3 `Person` nodes, `sameAs`, bylines) onto the DB author registry
  without losing the "retired author" behaviour.
- Add what DB content makes newly possible and cheap: an RSS/Atom feed, `dateModified` that reflects
  real edits, and redirects managed by content people rather than code.
- Strengthen CI so that a regression in any of the above fails the build rather than surfacing weeks
  later in Search Console.

## 3. Non-Goals

- No new SEO strategy, no new page types, no schema types beyond those already emitted
  (`Article`/`BlogPosting`, `HowTo`, `FAQPage`, `BreadcrumbList`, `Person`, `Organization`,
  `WebSite`).
- No change to URL structure, titles or descriptions of existing pages (parity is the point).
- No hreflang/international work — that is [MC.14](MC.14-localization-and-translations.md).
- No measurement/analytics work — SEO.15 owns Search Console and AI share-of-voice.
- No AI-crawler policy change; `crawler-policy.ts` stays authoritative for robots.

## 4. Personas & User Stories

- **As an SEO owner**, I want proof that structured data, canonicals and sitemaps are identical after
  the migration, so I do not have to re-audit 75 pages by hand.
- **As a content expert**, I want to move or rename a page and have the redirect happen, so I stop
  breaking inbound links.
- **As an AI crawler** (indirect), I want `llms-full.txt` to still contain the full text of our
  content, so we remain citable.
- **As a subscriber**, I want an RSS feed of the blog, because that is how practitioners follow
  writing.
- **As a release engineer**, I want CI to fail loudly if a DB-sourced page ships without a canonical
  or with a duplicate title.

## 5. Functional Requirements

- **FR-1.** `blogPostGraph()` and `docArticleGraph()` in `www/src/lib/schema/page-graphs.ts` MUST
  build from the content source (MC.7) rather than `content-meta.ts`'s file-derived arrays, producing
  byte-identical graphs for the same content.
- **FR-2.** `Person` nodes MUST come from the DB author registry, including `jobTitle`, `knowsAbout`
  and `sameAs` links; retired authors MUST keep plain-text bylines and MUST NOT emit a `Person` node
  or an author page (today's `isAuthorLinkable` policy).
- **FR-3.** `Article`/`BlogPosting` nodes MUST set `datePublished` from `published_at` and
  `dateModified` from `content_updated_at`, and MUST include `citation` entries from the article's
  `citations[]`.
- **FR-4.** `FAQPage` and `HowTo` nodes MUST continue to be derived from `:::faq` and `:::steps`
  directives in the body, unchanged.
- **FR-5.** JSON-LD validation (absolute `@id`, no dangling references, ≤ 12 KB) MUST continue to
  fail the build on error, now covering DB-sourced pages.
- **FR-6.** Canonical URLs MUST come from `canonical_override` when set, otherwise from the derived
  path; `noindex` articles MUST emit `robots: noindex, follow` and MUST be excluded from sitemaps,
  `llms.txt` and the RSS feed.
- **FR-7.** Sitemap `lastmod` MUST use `content_updated_at ?? published_at` for DB content, and the
  existing sitemap sectioning (`sitemapSectionForPath`) MUST place blog and docs URLs exactly as
  today.
- **FR-8.** `llms.txt` MUST keep its curated structure with DB-sourced titles/paths;
  `llms-full.txt` MUST contain the full plain text of every published, indexable article, generated
  from the fetched bodies.
- **FR-9.** Markdown siblings MUST be emitted for the same set of paths as today, with reconstructed
  front matter in a stable key order, and the `<link rel="alternate" type="text/markdown">` tag MUST
  continue to be emitted.
- **FR-10.** `dist/_redirects` MUST merge DB redirects (MC.3 `/redirects`) with the static list in
  `www/src/lib/redirects.ts`, static winning on conflict, and MUST reject cycles at build time.
- **FR-11.** An RSS 2.0 feed (`/blog/feed.xml`) and a JSON Feed (`/blog/feed.json`) MUST be generated
  from published blog posts (20 most recent, full description + link, author, pubDate), linked from
  `<head>` on `/blog` and each post.
- **FR-12.** OG images MUST use the article's hero media rendition (1200×630 preferred) when present,
  falling back to the existing default OG image; the image URL MUST be absolute and localised into
  `dist/`.
- **FR-13.** IndexNow submission MUST continue to work off the manifest diff, and a content-only
  publish MUST result in exactly the new/changed URLs being submitted.
- **FR-14.** CI MUST assert, for DB-sourced content: at least one blog and one docs page with `<h1>`,
  canonical, description and a valid JSON-LD graph; unique titles across the manifest; no page with a
  description > 160 chars; the markdown sibling exists; the feed validates.
- **FR-15.** A structured-data regression test MUST compare the emitted graph for a fixed fixture
  article against a checked-in golden JSON file, in addition to the schema validator.
- **FR-16.** `www/docs/structured-data.md` MUST be updated to state the data source for each node
  type.

## 6. Non-Functional Requirements

- **Performance** — Artefact generation adds < 5 s to the build; `llms-full.txt` generation is
  streaming, not held fully in memory beyond the current article.
- **Security** — Feeds and markdown siblings expose only published content. Feed output escapes
  content correctly (XML entities); no raw HTML from bodies enters the RSS `description` beyond the
  sanitized excerpt.
- **Privacy & Compliance** — Author `sameAs` links are consented profile links stored on the author
  row; retiring an author removes them from the graph.
- **Accessibility** — Not directly applicable; the OG/hero image path must preserve alt text for the
  on-page image even though OG has none.
- **Scalability** — Artefact size grows with content; sitemap sharding already exists
  (`buildSitemapArtifacts`) and must keep sharding correctly at 500+ URLs.
- **Reliability** — When content fetch degrades to previous-deploy HTML (MC.7 FR-5), the generator
  MUST NOT emit a truncated sitemap or a truncated `llms-full.txt`; it MUST reuse the previous
  artefacts instead, and record `fallbackUsed`.
- **Observability** — Manifest gains `contentSource`, `articleCount`, `feedItemCount`; the deploy
  workflow prints the sitemap URL count and fails if it drops by more than 10% versus the previous
  manifest (new guard).
- **Maintainability** — One content source feeds graphs, sitemaps, feeds and llms artefacts; no
  module reads markdown files directly after MC.15.
- **Internationalization** — All artefacts take a locale dimension; with one locale the output is
  unchanged. `hreflang` emission is explicitly MC.14.
- **Backward compatibility** — Output parity is the acceptance bar (MC.6's harness). Any intentional
  difference must be listed in the parity allowlist with a reason.

## 7. Acceptance Criteria

- **AC-1.** *Given* the imported corpus, *when* the site is built from the API, *then* every JSON-LD
  graph is byte-identical to the file-sourced build (parity harness).
- **AC-2.** *Given* a retired author, *when* their articles are built, *then* the byline is plain
  text, no `Person` node is emitted, and `/authors/{slug}` is absent from the manifest.
- **AC-3.** *Given* an article edited after publication, *when* built, *then* `dateModified` equals
  `content_updated_at` and the sitemap `lastmod` matches it.
- **AC-4.** *Given* an article with `noindex: true`, *when* built, *then* it has `noindex, follow`,
  is absent from every sitemap, `llms.txt`, `llms-full.txt` and the feed.
- **AC-5.** *Given* a DB redirect `/docs/old → /docs/new`, *when* built, *then* `_redirects`
  contains it, and a conflicting static redirect wins with a build log note.
- **AC-6.** *Given* a redirect cycle, *when* built, *then* the build fails with a clear error.
- **AC-7.** *Given* 5 published blog posts, *when* built, *then* `/blog/feed.xml` validates against
  RSS 2.0, contains 5 items with correct `pubDate`, `link`, `guid` and author, and `<head>` links to
  it.
- **AC-8.** *Given* an article with a hero image, *when* built, *then* `og:image` is an absolute URL
  to the localised 1200×630 rendition and `twitter:card` is `summary_large_image`.
- **AC-9.** *Given* `llms-full.txt`, *when* built, *then* it contains the full plain text of every
  indexable article (assert on a known phrase from the last article alphabetically).
- **AC-10.** *Given* a content-only publish, *when* the deploy completes, *then* IndexNow receives
  exactly the changed URL set (manifest diff test).
- **AC-11.** *Given* CI, *when* a DB-sourced build runs, *then* all existing crawlability assertions
  plus FR-14's new ones pass, and a deliberately broken graph fails the build.
- **AC-12.** *Given* the golden structured-data fixture, *when* the graph builder changes
  unintentionally, *then* the golden test fails.

## 8. Data Model

No new tables. Fields consumed: `canonical_override`, `noindex`, `citations[]`,
`content_updated_at`, `published_at`, `hero_media_id`, author registry (`knows_about`, `links`,
`status`), `content_redirects`.

One addition to the author row for `sameAs` completeness (if not already covered by `links JSONB`):
document the expected shape `{ "sameAs": ["https://…"], "website": "https://…" }` in
`docs/guides/marketing-content-dialect.md`.

Build artefacts gain: `dist/blog/feed.xml`, `dist/blog/feed.json`, manifest fields
(`contentSource`, `articleCount`, `feedItemCount`).

## 9. API Surface

No new server endpoints. Consumes MC.3's `/index`, `/articles/*`, `/authors`, `/redirects`,
`/media/*`.

Generator-side function contracts (all in `www`):

```ts
blogPostGraph(ctx: SchemaRenderContext): JsonLdNode[]      // now source-backed
docArticleGraph(ctx: SchemaRenderContext): JsonLdNode[]
buildFeeds(posts: BlogPostMeta[]): { rss: string; json: string }
buildLlmsFullTxt(articles: ArticleWithBody[]): string      // extended, same signature shape
mergeRedirects(dbRedirects, staticRedirects): RedirectRow[]  // static wins, cycle detection
```

## 10. UI / UX

No new UI. Two content-facing behaviours that must be explained in the workspace (MC.10 metadata
panel copy):

- Changing a published slug creates a 301 redirect automatically; the panel says so before saving.
- `noindex` removes the page from sitemaps, feeds and AI artefacts but keeps it reachable; the panel
  explains this in one sentence rather than exposing a bare checkbox.
- Hero image guidance: "Used as the social preview image. 1200×630 or larger works best."

i18n keys: `marketingContent.metadata.slug.redirectNotice`, `.noindex.help`, `.hero.help`.

## 11. AI / ML Considerations

AI-crawler-facing, not model-using. `llms.txt` and `llms-full.txt` are the artefacts AI retrieval
systems consume; their completeness after the migration is a first-class acceptance criterion (AC-9)
because a truncated `llms-full.txt` would silently reduce AI citation surface — the exact failure
mode SEO.2 was built to prevent.

## 12. Integration Points

- **`www` modules:** `src/lib/schema/page-graphs.ts`, `src/lib/schema/{article,person,faq,how-to}.ts`,
  `src/utils/content-meta.ts` (becomes source-backed), `src/lib/authors.ts` (becomes source-backed),
  `src/lib/redirects.ts`, `src/lib/crawler-policy.ts` (unchanged), `scripts/seo-artifacts.mjs`,
  `scripts/generate-site.mjs`, `scripts/submit-indexnow.mjs`.
- **Server:** MC.3 endpoints only.
- **CI:** `.github/workflows/pages-www.yml` assertion block extended.
- **Standards:** schema.org (`Article`, `BlogPosting`, `FAQPage`, `HowTo`, `Person`,
  `BreadcrumbList`), RSS 2.0, JSON Feed 1.1, IndexNow, sitemaps.org 0.9.

## 13. Dependencies & Sequencing

- Must ship after: MC.3, MC.7 (the content must be fetchable and renderable first).
- Must ship before: MC.15 (the cutover cannot happen until parity is proven at the artefact level).
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Silent structured-data regression | M | **H** | Golden graph fixtures (FR-15) + existing validator + parity harness; CI fails the build |
| Sitemap shrinks unnoticed (content fetch partially failed) | M | H | New guard: fail the deploy if URL count drops > 10% vs the previous manifest, unless `fallbackUsed` explains it |
| `llms-full.txt` truncated by a body fetch failure | M | H | Artefact reuse on fallback (NFR Reliability); AC-9 asserts content presence |
| Redirect conflicts create loops | L | M | Cycle detection at build time (AC-6) |
| Feed introduces duplicate-content signals | L | L | Feed items link to canonical URLs and carry `guid isPermaLink="true"` |
| Author `sameAs` data missing after migration | M | M | MC.6 imports the registry verbatim; a build warning lists authors with empty `sameAs` |

## 15. Rollout Plan

- **Feature flag:** none of its own; gated by `WWW_CONTENT_SOURCE` (MC.7).
- **Sequencing:** source-backed graphs → sitemaps/lastmod → llms artefacts → redirects merge → feeds
  → OG/hero → CI assertions → golden fixtures.
- **Dogfood:** run the parity harness after each step; keep a running diff report so regressions are
  attributed to a single change.
- **GA criteria:** parity harness green; Rich Results Test passes on one blog and one docs URL from
  the staging build; feed validates in an external validator; sitemap count guard active.
- **Rollback:** `WWW_CONTENT_SOURCE=files`.

## 16. Test Plan

- **Unit** — graph builders against fixtures (published, retired author, noindex, citations, FAQ,
  HowTo); redirect merge and cycle detection; feed generation and XML escaping; lastmod precedence;
  OG image resolution and fallback.
- **Integration** — full build against a stubbed content API; assert manifest, sitemaps, llms files,
  feeds, siblings and `_redirects` contents; simulate fallback and assert artefact reuse.
- **End-to-end** — CI build from staging API with the extended assertion block; external validation
  of one page's structured data and the RSS feed in the nightly job.
- **Security** — feed escaping with hostile content (script tags, control characters); markdown
  sibling emission cannot leak drafts; redirect targets restricted to same-origin paths.
- **Accessibility** — N/A directly; verify the hero image's on-page rendering keeps alt text.
- **Performance / load** — artefact generation timing; `llms-full.txt` memory profile at 500
  articles.
- **Manual exploratory** — Search Console URL inspection on two staged URLs; social preview check
  (Slack/X/LinkedIn unfurl) for a post with and without a hero image.

## 17. Documentation & Training

- `www/docs/structured-data.md` — per-node data source table (DB vs file).
- `www/docs/url-policy.md` — redirects are now content-managed; explain the automatic 301 on slug
  change.
- `www/docs/site-generation.md` — feeds and new manifest fields.
- Content team note: what `noindex` does, and why slug changes are not free.

## 18. Open Questions

1. Do we want per-category help feeds in addition to the blog feed? (Proposed: no — low demand, more
   artefacts to keep correct.)
2. Should `llms-full.txt` include help articles or only blog posts? (Today it includes content pages;
   keep the current behaviour exactly — verify in the parity harness rather than deciding anew.)
3. Do we generate a branded fallback OG image per article (title on a template) when no hero exists?
   (Proposed: yes, as a follow-on — it measurably improves social CTR, but it is not parity work.)
4. Should the sitemap-count guard threshold be 10% or absolute (any drop)? (Proposed: 10% to tolerate
   legitimate archiving.)

## 19. References

- Files this work touches: `www/src/lib/schema/*`, `www/src/utils/content-meta.ts`,
  `www/src/lib/authors.ts`, `www/src/lib/redirects.ts`, `www/scripts/seo-artifacts.mjs`,
  `www/scripts/generate-site.mjs`, `.github/workflows/pages-www.yml`.
- Related plans: [MC.3](MC.3-public-content-read-api.md),
  [MC.7](MC.7-www-build-time-content-integration.md),
  [MC.14](MC.14-localization-and-translations.md),
  [MC.15](MC.15-rollout-cutover-and-decommission.md); SEO.1–SEO.4 (foundation), SEO.16 (governance).
- Standards: schema.org, RSS 2.0, JSON Feed 1.1, sitemaps.org 0.9, IndexNow.
