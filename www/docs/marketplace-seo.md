# Marketplace SEO prerender

> **Superseded by [site-generation.md](./site-generation.md)** (SEO.1).
>
> Course prerender is now part of the full-site static generator
> (`scripts/generate-site.mjs`). That script renders every manifest route
> (including `/courses` and `/courses/:slug`) with `renderToString`, writes
> `.seo-manifest.json`, and degrades gracefully when the marketplace API is
> down.

See:

- [site-generation.md](./site-generation.md) — architecture, env vars, failure modes
- [adding-a-page.md](./adding-a-page.md) — how to add routes

## Catalog policy

Production builds fetch every listed, published marketplace course with bounded concurrency (default
8). Course pages use a distinctive title containing the course level and subject, creator-authored
summary metadata, a canonical URL, and a `Course` graph. Rating markup is emitted only after five
reviews. Creator markdown is escaped and external links receive `nofollow ugc noopener`.

The generator applies the SEO.11 quality floor in `scripts/marketplace-seo.mjs`. A listing must have
300 characters of original description, sufficiently distinct creator copy, 3 modules or 5 content
items, an image, complete subject/level/language/price metadata, a verified creator, and clear
moderation status. Failing pages remain usable but receive `noindex,follow` and are excluded from
sitemaps. `dist/.catalog-quality.json` records every check and its threshold.

Subject and level hubs are emitted only when at least three matching courses exist. They contain
orienting copy, crawlable course links, and `ItemList` schema. Multi-dimensional filtering stays on
the `/courses` client view and does not create indexable facet URLs.

## Freshness and recovery

The Pages workflow runs hourly as well as on www changes and manual dispatch. IndexNow receives new
and materially changed manifest URLs after deployment. `FORCE_COURSE_REBUILD=1` (or the manual
workflow checkbox) forces a full catalog rebuild. `.course-cache.json` stores course update versions
for incremental discovery. If the API is unavailable, generation attempts to reuse course HTML from
the previous deployment rather than replacing the catalog with an empty result.
