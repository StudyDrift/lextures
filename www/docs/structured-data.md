# Structured data & entity graph (SEO.3)

The marketing site emits a single JSON-LD `@graph` on every page. Nodes share stable absolute `@id`s so Organization, WebSite, Article, Person, and Course reference each other instead of duplicating data.

## Rules

1. **Never assert what is not visible and true on the page.** Schema that disagrees with on-page copy is a spam policy risk and fails AI-surface trust.
2. **All `@id`s are absolute URLs** and are spelled only in `www/src/lib/schema/ids.ts`.
3. **No schema string literals outside `www/src/lib/schema/`.**
4. **`sameAs` is consented author data** from the content author registry; organization profiles remain the fixed allowlist in `entity.ts`.
5. **Build validates** missing `@id`, non-absolute `@id`, dangling refs, and a **12 KB** payload budget (`generate-site.mjs`).

## Envelope

```json
{
  "@context": "https://schema.org",
  "@graph": [ /* nodes */ ]
}
```

Emitted as one `<script type="application/ld+json" id="site-json-ld">`. Serialization escapes `<`, `>`, `&` so untrusted course titles cannot break out of the script element.

## Site-wide nodes (every page)

| Type | `@id` | Notes |
|---|---|---|
| `ImageObject` | `{origin}/#logo` | Logo |
| `Organization` | `{origin}/#organization` | Brand entity |
| `Person` (founder stub) | `{origin}/about#founder` | Founder |
| `WebSite` | `{origin}/#website` | No `SearchAction` until site search exists |
| `SoftwareApplication` | `{origin}/#software` | Offers from `institution-pricing.ts` + homeschool $20/mo |

## Page-type nodes

| Page | Types |
|---|---|
| `/about` | + active `Person` authors |
| `/authors`, `/authors/:slug` | `Person` |
| `/blog/:slug` | `Article`, `Person`, `BreadcrumbList` |
| `/docs/:slug` | `TechArticle`, optional `HowTo`, `Person` |
| `/pricing` | `Product`/`Offer`, `FAQPage` |
| `/courses` | `ItemList` (≥3 courses when API available) |
| `/courses/:slug` | `Course` (server payload merged; `@context` stripped) |
| `/accessibility`, `/vpat` | `WebPage` + VPAT `CreativeWork` |
| `/security` | `WebPage` (no unearned certifications) |
| `/privacy`, `/terms` | `DigitalDocument` |

Every path except `/` also gets `BreadcrumbList` (visible UI breadcrumbs land with SEO.5).

Translated content pages emit reciprocal `<link rel="alternate" hreflang>` tags (including `x-default` on the English URL) only when two or more locales in the translation group are published. English-only builds emit no hreflang. Sitemap urlsets for non-English locales include matching `xhtml:link` alternates.


## Node data sources

| Node | Source |
|---|---|
| `Organization`, `WebSite`, `SoftwareApplication` | Versioned `www` configuration |
| `Person` | Public database author registry for API builds; file registry for rollback builds |
| `Article`, `TechArticle` | Published content API article, including publication/update dates and citations |
| `FAQPage`, `HowTo` | Directives parsed from the published article body |
| `Course`, course `ItemList` | Public marketplace API |
| `BreadcrumbList` | Route manifest and the article's database path |

Retired authors keep a plain-text byline and embedded author name on the article node, but do not get a standalone `Person` node or author route.

## Adding a node type

1. Add a pure builder under `www/src/lib/schema/<type>.ts`.
2. Export from `index.ts` / wire via `page-graphs.ts`.
3. Attach `jsonLd` on the route in `route-manifest.tsx` (or compose in `entry-server.tsx` for courses).
4. Add unit coverage in `schema/schema.test.mjs`.
5. Build and confirm `.seo-manifest.json` `schemaTypes[]` for the URL.

## Course JSON-LD from the API

`GET /api/v1/public/marketplace/courses/{slug}` returns `jsonLd` **without** `@context`, **with** `@id`. The www graph builder merges it into the page graph and points `provider` at Organization `@id`.

## Validation

- Local: `cd www && npm test` (includes hostile-title escape + graph validation).
- Build: `npm run build` fails on dangling refs / oversize graph.
- Manual: [Google Rich Results Test](https://search.google.com/test/rich-results) and [Schema Markup Validator](https://validator.schema.org/) on homepage, `/pricing`, a blog post, and a course page.

## Related

- [authoring-bylines.md](./authoring-bylines.md) — author registry & consent
- [site-generation.md](./site-generation.md) — SSG pipeline
