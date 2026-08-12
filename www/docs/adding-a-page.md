# Adding a marketing page (checklist)

Every public www URL must be in the **route manifest**. If it is not, the static generator will not emit HTML and crawlers will see a 404.

## Steps

1. **Create the page component** under `www/src/pages/` (use `MarketingPageShell` for standard layout).
2. **Add a `RouteDescriptor`** to `www/src/lib/route-manifest.tsx`:
   - `path` — no trailing slash except `/`
   - **Add a page loader** in `www/src/lib/route-pages.ts` (`PAGE_LOADERS[path]`)
   - `title` — unique, prefer ≤ 60 characters
   - `description` — unique, ≤ 160 characters (enforced at build)
   - `sitemap: true` (or `false` for thank-you / history / utility pages)
   - `robots` — default `index,follow`; use `noindex,follow` when appropriate
   - **`interactive`** — `false` if the page is prerendered content only (no forms, marketplace, or client state); `true` when it needs React hydration. See [performance-budget.md](./performance-budget.md).
   - `changefreq` / `priority` — optional sitemap hints
   - `jsonLd` — optional; prefer composing via `www/src/lib/schema/page-graphs.ts` (site-wide Organization/WebSite/SoftwareApplication is automatic when you use those helpers)
   - `locale` is assigned as `en` to root routes. For a localized entry, use an allowlisted BCP 47 locale plus `translationOf`, `translationStatus`, and `sourceUpdatedAt`; follow [internationalization.md](internationalization.md).
3. **Do not** add a new `if (route === ...)` branch in `app.tsx` — the router reads the manifest only.
4. **Link to the page** with a real `<a href="...">` (header, footer, or body). Avoid JS-only navigation for primary entry points.
5. **Build locally** and confirm output:
   ```bash
   cd www && npm run build
   # dist/<your-path>/index.html should exist and include your <h1>
   grep -n '<h1' dist/<your-path>/index.html
   grep -n 'canonical' dist/<your-path>/index.html
   ```
6. **If the page should appear in `llms.txt`**, add a curated entry with a
   question-style description to `www/src/lib/llms-catalog.ts` (≤200 links total).
7. Add at least one original diagram, automated product screenshot, or chart to substantive pages. Use `Figure`/`Diagram`, provide a complete text equivalent for complex visuals, and verify the generated raster social card in `dist/og/`. See [social cards](social-cards.md) and [diagram authoring](diagram-authoring.md).

## Dynamic families (`/blog/:slug`, `/docs/:slug`)

- Put markdown under `www/src/blog/` or `www/src/docs/`.
- Frontmatter: `title`, `date`, `description`, `author` (registry **slug**, not free text), optional `updated`, `reviewedBy`, `citations` (blog).
- Unknown author slugs fail the build — see [authoring-bylines.md](./authoring-bylines.md).
- `enumerate()` in the manifest already expands these files — no extra step if you only add a markdown file.
- The build emits a `.md` sibling (`/docs/foo.md`) and `<link rel="alternate" type="text/markdown">` on the HTML page.
- Blog/docs bodies are included in `llms-full.txt` automatically.

## Course pages

Course paths come from the public marketplace API at build time. Publish/list the course on the API; the next www deploy prerenders `/courses/<slug>`.

## Redirects

Path moves go in `www/src/lib/redirects.ts` (`REDIRECTS`). The generator emits `_redirects` and GitHub Pages-compatible meta-refresh stubs.

## CI

`pages-www.yml` asserts that high-value routes exist with `<h1>`, canonical, and description; that `.seo-manifest.json` has unique titles; that `robots.txt` / `llms.txt` / sitemap index exist; and that IndexNow submission runs after deploy (warnings only on failure).

See [crawler-policy.md](./crawler-policy.md), [site-generation.md](./site-generation.md), and [structured-data.md](./structured-data.md).
# IA and internal links

Set `parent` on every route except `/`. Set `hub: true` on cluster indexes, `cluster` for related-content fallback, `relatedTo` for explicit recommendations, and `navGroup` when the route belongs in global navigation. Content routes must include at least three descriptive contextual internal links and a Related module. Never use “click here,” “read more,” or a bare URL as anchor text. See [the information architecture guide](information-architecture.md) and [URL policy](url-policy.md).
