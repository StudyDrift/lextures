# Information architecture

The route manifest is the source of truth. A route's `parent` drives its visible breadcrumb, structured breadcrumb, and crawl-depth calculation. `hub: true` declares an index responsible for its direct children; `cluster`, `relatedTo`, and `navGroup` support related links and navigation.

The homepage links to segment hubs (`/k12`, `/higher-ed`, `/homeschool`, `/parents`), product (`/platform`), content (`/resources`), trust (`/trust`), pricing, docs, courses, compare, integrations, and conversion pages. Platform, resource, trust, blog, docs, author, pricing, and course hubs link to their children. Trust documents keep their established top-level URLs.

## Adding a page

- Put the route in `src/lib/route-manifest.tsx` and its loader in `src/lib/route-pages.ts`.
- Choose an existing `parent`; do not exceed three clicks from `/`.
- Add it to its hub and use descriptive internal anchor text.
- Articles need three contextual internal links and a deterministic Related module of three to six links.
- Run the build. For an orphan or depth error, inspect `dist/.link-graph.json` and add a meaningful link from the declared hub.
