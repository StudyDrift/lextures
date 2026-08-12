# MC — Marketing Content Platform (database-backed blog & help center)

> 15 plans that move `www` blog posts and help articles out of the git repository and into the
> Lextures database, expose them through a public read API in the same shape the marketplace
> catalog already uses, and give content experts a first-class **Marketing Content** workspace
> inside the app — behind a platform feature flag and RBAC.

---

## The one-paragraph version

Today every marketing page is a markdown file. `www/src/utils/blog.ts` and `www/src/utils/docs.ts`
load them with `import.meta.glob('../blog/*.md')` at build time, so **publishing an article requires
a git commit, a code review, and a full CI deploy** — 5 blog posts and 70 help articles in, that is
already the binding constraint on content velocity, and [SEO.7](../../completed/seo/SEO.7-help-center-expansion.md)
targets 60+ new help articles while [SEO.8](../../completed/seo/SEO.8-editorial-engine-and-content-calendar.md)
targets a sustained publishing cadence. Content experts — the people who should be writing this —
cannot use GitHub. This program moves the content into Postgres on the same database that serves
`self.lextures.com`, publishes it through `GET /api/v1/public/content/*` (the same
build-time-fetch + previous-deploy-fallback pattern `generate-site.mjs` already uses for
`/courses/*`), and adds an authoring workspace at `/admin/marketing-content` gated by
`ff_marketing_content` and `global:app:marketing-content:*` permissions. **The public site stays
statically generated** — no runtime dependency on the API for a crawler, no regression of the
SEO.1–SEO.4 foundation that made these pages exist at all.

---

## Architecture in five decisions

1. **Postgres is the source of truth; `www` stays static.** A new `marketing` schema holds articles,
   revisions, categories, authors, media and redirects. `generate-site.mjs` fetches published
   content at build time and writes the same `dist/**/index.html` it writes today. Publishing
   triggers a rebuild ([MC.8](MC.8-publish-pipeline-and-scheduling.md)); it does not make crawlers
   depend on the app being up.
2. **One content table for both surfaces.** Blog posts and help articles differ by a `kind`
   discriminator and a few optional columns, not by schema. Everything the current front matter
   carries (`pillar`, `briefRef`, `reviewDue`, `citations`, `roles`, `segments`, `verifiedAgainst`,
   `relatedTo`, `primaryQuestion`, `cluster`, `keywords`) becomes a typed column or array
   ([MC.1](MC.1-content-data-model-and-migrations.md)).
3. **Markdown stays markdown.** The API stores and serves the same markdown + `:::directive` syntax
   authors write today, so `www/src/lib/markdown.ts` remains the renderer of record and the
   answer-first content contract survives the move. A Go renderer exists only for preview, search
   excerpts and lint scoring, and is pinned to the JS renderer by a shared golden corpus
   ([MC.4](MC.4-content-rendering-and-validation-service.md)).
4. **The content contract becomes a publish gate, not a CI gate.** `scripts/content-lint` rules move
   server-side: an article cannot reach `published` with a quality score below the floor, a missing
   citation for a numeric claim, an unknown directive, or a broken internal link
   ([MC.4](MC.4-content-rendering-and-validation-service.md), [MC.11](MC.11-editorial-workflow-and-governance.md)).
5. **Cutover is reversible until the last plan.** Files remain the source of truth while
   `WWW_CONTENT_SOURCE=files`; `api` is proved by a byte-diff parity harness over the generated
   `dist/` before the markdown directories are deleted ([MC.6](MC.6-markdown-to-database-migration.md),
   [MC.15](MC.15-rollout-cutover-and-decommission.md)).

---

## Plans

| ID | Plan | Effort | Depends on |
|---|---|---|---|
| MC.1 | [Content data model, feature flag & RBAC](MC.1-content-data-model-and-migrations.md) | S | — |
| MC.2 | [Authoring API, revisions & workflow states](MC.2-authoring-api-and-revisions.md) | M | MC.1 |
| MC.3 | [Public content read API & caching](MC.3-public-content-read-api.md) | S | MC.1 |
| MC.4 | [Rendering, sanitization & content-contract validation](MC.4-content-rendering-and-validation-service.md) | M | MC.1 |
| MC.5 | [Marketing media library & image pipeline](MC.5-marketing-media-library.md) | S | MC.1 |
| MC.6 | [Markdown → database migration & parity harness](MC.6-markdown-to-database-migration.md) | M | MC.1–MC.5 |
| MC.7 | [www build-time content integration (SSG from API)](MC.7-www-build-time-content-integration.md) | M | MC.3, MC.6 |
| MC.8 | [Publish pipeline, scheduling & rebuild dispatch](MC.8-publish-pipeline-and-scheduling.md) | S | MC.2, MC.7 |
| MC.9 | [Marketing Content workspace: nav, gating & content list](MC.9-marketing-content-workspace-shell.md) | S | MC.2 |
| MC.10 | [Article editor: authoring, metadata, preview & revisions](MC.10-article-editor.md) | L | MC.4, MC.5, MC.9 |
| MC.11 | [Editorial workflow, review & governance](MC.11-editorial-workflow-and-governance.md) | M | MC.2, MC.10 |
| MC.12 | [SEO parity from the database](MC.12-seo-parity-from-database.md) | M | MC.3, MC.7 |
| MC.13 | [Docs search & in-app help integration](MC.13-docs-search-and-in-app-help.md) | S | MC.3, MC.7 |
| MC.14 | [Localization & translated content](MC.14-localization-and-translations.md) | M | MC.1, MC.3, MC.10 |
| MC.15 | [Rollout, cutover & decommission of file-based content](MC.15-rollout-cutover-and-decommission.md) | S | all |

### Suggested delivery order

```
MC.1 ─┬─ MC.2 ─┬─ MC.9 ── MC.10 ── MC.11 ─┐
      │        └─ MC.8 ────────────┐      │
      ├─ MC.3 ─┬─ MC.7 ─┬─ MC.12 ──┼──────┼── MC.15
      │        │        └─ MC.13 ──┘      │
      ├─ MC.4 ─┤                          │
      └─ MC.5 ─┴─ MC.6 ──────────────── MC.14
```

Weeks 1–3 are backend only and ship dark. The first user-visible change is MC.9's nav link, which
appears only when `ff_marketing_content` is on **and** the viewer holds
`global:app:marketing-content:view`. `www` keeps building from files until MC.15 flips
`WWW_CONTENT_SOURCE=api` on a green parity run.

---

## Non-negotiable constraints

- **No crawlable regression.** Every URL that returns 200 with rendered HTML, a canonical, a unique
  title and a JSON-LD graph today must do the same after cutover. The parity harness in
  [MC.6](MC.6-markdown-to-database-migration.md) diffs generated HTML, `dist/.seo-manifest.json` and
  the sitemap set; the [CI assertions](../../../.github/workflows/pages-www.yml) stay in place and
  gain DB-sourced cases.
- **A build must never fail because the API is down.** Content fetch inherits the marketplace
  failure policy: WARN, reuse previous-deploy HTML, exit 0 ([MC.7](MC.7-www-build-time-content-integration.md)).
- **Drafts are never crawlable.** Unpublished content is reachable only through short-lived signed
  preview tokens with `no-store` + `X-Robots-Tag: noindex`, and is never emitted into `dist/`.
- **Editorial quality gates survive the migration.** The score floor, front-matter validation,
  citation policy, author registry and help-article freshness checks become API-enforced rather than
  disappearing with the files they used to lint.
- **Accessibility is not optional.** Every new admin surface is WCAG 2.1 AA, uses
  `clients/web/src/components/ui/*` primitives and semantic design tokens
  ([AGENTS.md](../../../AGENTS.md)).

---

## What this program deliberately does not do

- It does not turn `www` into a server-rendered app or move it off GitHub Pages.
- It does not put course content, syllabi or in-course pages under this model — that is
  [CT](../../completed/content_tools/) and [AC](../../completed/adaptive/) territory. This is
  *marketing* content only.
- It does not add AI content generation. The editor may later reuse
  `server/internal/service/contentpagegeneration`, but nothing in MC.1–MC.15 depends on it, and
  [MC.11](MC.11-editorial-workflow-and-governance.md) requires human accountability for every
  published byte.
- It does not change the public URL shape: `/blog/{slug}` and `/docs/{category}/{slug}` stay exactly
  as they are ([www/docs/url-policy.md](../../../www/docs/url-policy.md)).

---

## Key references

- Current loaders: [`www/src/utils/blog.ts`](../../../www/src/utils/blog.ts),
  [`www/src/utils/docs.ts`](../../../www/src/utils/docs.ts)
- Renderer: [`www/src/lib/markdown.ts`](../../../www/src/lib/markdown.ts)
- SSG: [`www/scripts/generate-site.mjs`](../../../www/scripts/generate-site.mjs) ·
  [site-generation.md](../../../www/docs/site-generation.md)
- Route manifest: [`www/src/lib/route-manifest.tsx`](../../../www/src/lib/route-manifest.tsx)
- Content contract: [`www/docs/content-contract.md`](../../../www/docs/content-contract.md) ·
  [`www/scripts/content-lint/core.mjs`](../../../www/scripts/content-lint/core.mjs)
- Public API precedent: [`server/internal/httpserver/public_marketplace_http.go`](../../../server/internal/httpserver/public_marketplace_http.go)
- Flags: [`server/internal/repos/platformconfig/features.go`](../../../server/internal/repos/platformconfig/features.go)
- Nav: [`clients/web/src/components/layout/side-nav-admin-links.tsx`](../../../clients/web/src/components/layout/side-nav-admin-links.tsx)
