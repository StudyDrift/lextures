# SEO.5 — Information Architecture, URL Policy & Internal Linking

> Implementation plan. Source: [docs/plan/seo/audit.md](audit.md) §S3 (F-19, F-20).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.5 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN (3 header links, no breadcrumbs, no hubs, no related-content modules, no URL policy) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform + Marketing |
| **Depends on** | SEO.1 |
| **Unblocks** | SEO.3 (BreadcrumbList), SEO.7, SEO.8, SEO.9, SEO.10 |

---

## 1. Problem Statement

`header.tsx` emits exactly three links — `/`, `/#institutions`, `/get-started` — and there are no
breadcrumbs, no hub pages, no related-content modules, and no cross-links between segment pages and
the content that supports them (audit F-19). Crawl budget and internal authority have nowhere to
flow, and a reader who lands on a blog post from an AI citation has no path to the product. URL
policy has also drifted: `/k-12` uses a hyphen against the universal `k12` convention, `/parents` and
`/homeschool` overlap without a declared canonical relationship, `/self-learner` "redirects" via
meta-refresh rather than a 301, and no trailing-slash rule is enforced (F-20). Topical authority in
2026 comes from *interconnected* clusters — "25 well-connected articles outrank 250 scattered ones"
([research §7](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)) — and
we are about to add 200+ pages. The link graph has to exist before the content does.

## 2. Goals

- Define and enforce a **URL policy** (casing, separators, trailing slash, depth, stability) and a
  **redirect map** so no URL change ever leaks equity again.
- Establish a **hub-and-spoke IA**: five product/segment hubs and five topic hubs, each with a real
  page that links down to its cluster and up to the product.
- Ship the internal-linking primitives every content plan depends on: breadcrumbs, related content,
  contextual in-body links, hub indexes, and a footer sitemap.
- Give every page a defined position — no orphans, maximum depth 3 clicks from the homepage.
- Make the primary navigation actually navigate: expose segments, solutions, resources and pricing.

## 3. Non-Goals

- Writing the content that fills the clusters (SEO.7, SEO.8, SEO.9, SEO.10).
- Redesigning the visual identity of the header/footer beyond what new links require.
- The in-product navigation IA — that is [UX.7](../ui-ux/UX.7-navigation-information-architecture.md)
  (reverted in #606 and pending re-plan) and applies to `clients/web`, not `www`.
- Site search (noted as a follow-on; `SearchAction` schema stays omitted until it exists — SEO.3 FR-8).

## 4. Personas & User Stories

- **As a curriculum director arriving from an AI citation on a rubrics article**, I want an obvious
  path to "what does Lextures do for higher ed", so that the article converts into evaluation.
- **As a homeschool parent**, I want to see all homeschool-relevant resources in one place, so that
  I do not have to guess at URLs.
- **As a search engine**, I want a shallow, densely-linked graph with breadcrumbs, so that I can
  crawl the whole site and understand which pages are the authoritative hubs.
- **As a writer publishing article #40**, I want the related-links module to place it automatically,
  so that internal linking does not depend on my memory.
- **As an SRE**, I want URL changes to route through a redirect map, so that a rename cannot silently
  404.

## 5. Functional Requirements

**URL policy**

- **FR-1.** URLs MUST be lowercase, hyphen-separated, without trailing slash (except `/`), without
  file extensions, and MUST NOT exceed **3 path segments** for content
  (`/resources/assessment/rubric-design` is the deepest permitted shape).
- **FR-2.** URL slugs MUST be stable. Renaming a published URL requires a redirect-map entry in the
  same commit; CI MUST fail if a URL disappears from `.seo-manifest.json` between builds without a
  matching redirect entry.
- **FR-3.** `/k-12` MUST become `/k12`, with `/k-12` → `/k12` in the redirect map. (Hyphenated form
  matches no competitor URL and no query pattern.)
- **FR-4.** The `/parents` ↔ `/homeschool` overlap MUST be resolved explicitly: `/homeschool` is the
  product/segment page (buyer = parent-as-purchaser); `/parents` becomes the **parent portal**
  audience page (parent-as-observer of a school-enrolled child) with distinct copy, or is merged into
  `/homeschool` with a 301. The decision MUST be recorded in the redirect map either way; ambiguity
  is the failure mode.
- **FR-5.** `/self-learner` → `/homeschool` MUST become a real 301 once SEO.1 FR-12 hosting lands,
  replacing the meta-refresh stub.
- **FR-6.** Query parameters that do not change content (`?coupon=`, `?utm_*`, `?ref=`) MUST NOT
  produce a distinct canonical; the canonical is always the clean path. Marketing parameters MUST be
  disallowed in `robots.txt` (SEO.2 FR-3).

**Redirect map**

- **FR-7.** `www/src/lib/redirects.ts` MUST hold `{ from, to, status: 301|308, addedAt, reason }[]`,
  rendered to the host's `_redirects` file and to canonical stubs while on GitHub Pages.
- **FR-8.** Redirect chains MUST be flattened at build time (A→B→C becomes A→C and B→C); CI MUST fail
  on a cycle or a redirect whose target 404s.

**Information architecture**

- **FR-9.** The site MUST adopt this structure. Every node is a real page with real content:

  ```
  /                                  Home
  ├── /k12                           Segment hub — K-12 districts & schools
  ├── /higher-ed                     Segment hub — colleges & universities
  ├── /homeschool                    Segment hub — homeschool families
  ├── /parents                       Audience — parents of enrolled students   (FR-4)
  ├── /platform/                     Product hub                                (new)
  │   ├── /platform/adaptive-learning
  │   ├── /platform/assessment
  │   ├── /platform/grading
  │   ├── /platform/analytics
  │   ├── /platform/accessibility
  │   └── /platform/ai
  ├── /pricing  ·  /pricing/calculator
  ├── /courses  ·  /courses/:slug  ·  /courses/subject/:subject            (SEO.11)
  ├── /compare/  ·  /compare/:competitor  ·  /alternatives/:competitor     (SEO.9)
  ├── /integrations/  ·  /integrations/:slug                               (SEO.9)
  ├── /resources/                    Content hub                             (new)
  │   ├── /blog  ·  /blog/:slug
  │   ├── /guides/:slug              Pillar pages                             (SEO.8)
  │   ├── /glossary  ·  /glossary/:term                                      (SEO.10)
  │   ├── /templates/:slug                                                   (SEO.10)
  │   └── /research/:slug            Original research reports                (SEO.12)
  ├── /docs  ·  /docs/:category  ·  /docs/:category/:slug                    (SEO.7)
  ├── /about  ·  /authors  ·  /authors/:slug                                 (SEO.3)
  ├── /trust/                        Trust hub                                (new)
  │   ├── /security  ·  /accessibility  ·  /accessibility/vpat
  │   ├── /privacy  ·  /privacy/history  ·  /terms  ·  /terms/history
  │   └── /privacy-rights/california
  └── /get-started  ·  /request-information
  ```

- **FR-10.** Every page MUST be reachable within **3 clicks** of the homepage. CI MUST compute the
  internal link graph from the built HTML and fail on any orphan (0 inbound internal links) or any
  page at depth > 3.
- **FR-11.** Existing trust-page URLs (`/security`, `/accessibility`, `/privacy`, `/terms`) MUST NOT
  move; `/trust/` is a hub that links to them, not a path prefix. (These are the pages most likely to
  be linked externally by procurement teams.)

**Navigation & linking primitives**

- **FR-12.** The header MUST expose a real navigation: **Platform** (mega-menu of the six platform
  pages), **Solutions** (K-12, Higher Ed, Homeschool, Parents), **Pricing**, **Resources** (Blog,
  Guides, Glossary, Research, Templates), **Docs**, plus **Get started** as the CTA. All links MUST
  be real `<a href>` elements present in the server-rendered HTML.
- **FR-13.** The footer MUST become a **sitemap footer** with columns for Platform, Solutions,
  Resources, Docs, Compare, Company (About, Authors, Contact) and Trust — every hub linked, plus
  `/sitemap.xml` and `/llms.txt`.
- **FR-14.** Every page except `/` MUST render a visible `<nav aria-label="Breadcrumb">` trail
  matching the IA, backed by `BreadcrumbList` schema (SEO.3 FR-10).
- **FR-15.** Every article (blog, guide, help, glossary, research) MUST render a **Related** module of
  3–6 links, computed from an explicit `relatedTo` front-matter list first and a shared-tag fallback
  second. Deterministic ordering is required so the module is stable across builds.
- **FR-16.** Every article MUST carry 3–10 **contextual in-body internal links** with descriptive
  anchor text. CI MUST warn below 3 and fail at 0 for any page in `/resources/` or `/docs/`.
- **FR-17.** Hub pages MUST link to **every** page in their cluster (no pagination-hidden children at
  the hub level; if a cluster exceeds 50 items, the hub links to sub-hubs).
- **FR-18.** Each segment hub (`/k12`, `/higher-ed`, `/homeschool`) MUST link to: its platform pages,
  its top 5 cluster articles, its comparison pages, pricing, and the relevant help-center category.
- **FR-19.** Anchor text MUST be descriptive; "click here", "read more", and bare URLs are prohibited
  in body content, enforced by a lint rule.
- **FR-20.** Cross-links MUST be bidirectional where meaningful: a comparison page links to the
  segment hub, and the segment hub links back to the comparison page.

## 6. Non-Functional Requirements

- **Performance** — the header mega-menu and footer sitemap add markup to every page. Combined
  header+footer HTML MUST stay ≤ 12 KB uncompressed, and the mega-menu MUST be CSS-driven with no
  JS required to reveal links (crawlers and no-JS users must see them).
- **Security** — external links in content MUST carry `rel="noopener"`; user-generated or
  partner-supplied links (course descriptions) MUST additionally carry `rel="nofollow ugc"`.
- **Privacy & Compliance** — trust-page URLs are cited in signed DPAs and procurement documents;
  FR-11 exists to protect that. Any change to a trust URL requires legal sign-off.
- **Accessibility** — the mega-menu MUST be keyboard-operable (arrow-key roving tabindex per the UX
  plan set's `role="menu"` conventions), with visible focus, `aria-expanded`, Escape to close, and no
  hover-only reveal. Breadcrumbs use `<nav aria-label="Breadcrumb">` with `aria-current="page"` on the
  last item. WCAG 2.2 AA, matching UX.5/UX.6 standards.
- **Scalability** — the link-graph checker must run over 1,000+ pages in < 30 s.
- **Reliability** — redirect map validation (FR-8) prevents chains and cycles reaching production.
- **Observability** — build emits `dist/.link-graph.json` (nodes, edges, depth, inbound counts) for
  SEO.16 assertions and SEO.15 reporting.
- **Maintainability** — IA is expressed in the route manifest (`parent` field), so breadcrumbs, hub
  membership, and depth all derive from one declaration.
- **Internationalization** — nav and breadcrumb labels are i18n keys, not hard-coded strings, ready
  for SEO.17.
- **Backward compatibility** — FR-3/FR-4/FR-5 are the only URL changes; each has a redirect entry.

## 7. Acceptance Criteria

- **AC-1.** *Given* the built site, *When* the link-graph checker runs, *Then* zero pages have 0
  inbound internal links and zero pages are at depth > 3 from `/`.
- **AC-2.** *Given* a request to `/k-12`, *When* it is served, *Then* the response is a 301 to `/k12`
  (or, pre-migration, a canonical stub pointing at `/k12`).
- **AC-3.** *Given* the redirect map, *When* CI validates it, *Then* no chains, no cycles, and every
  target returns 200.
- **AC-4.** *Given* a PR that renames a published URL without a redirect entry, *When* CI runs,
  *Then* it fails naming the removed URL.
- **AC-5.** *Given* any page other than `/`, *When* rendered with JS disabled, *Then* a breadcrumb
  trail is visible and matches the manifest `parent` chain, and `BreadcrumbList` schema is present
  with identical items.
- **AC-6.** *Given* the header with JS disabled, *When* rendered, *Then* all Platform / Solutions /
  Resources links are present in the HTML as `<a href>` elements.
- **AC-7.** *Given* keyboard-only operation, *When* I tab into the Platform menu, *Then* it opens on
  Enter/Space, arrow keys move between items, Escape closes and returns focus to the trigger, and
  focus is visible at every step.
- **AC-8.** *Given* any `/resources/*` or `/docs/*` page, *When* checked, *Then* it has ≥3 contextual
  in-body internal links and a Related module with 3–6 entries.
- **AC-9.** *Given* two consecutive builds with unchanged content, *When* I diff the Related modules,
  *Then* they are identical (deterministic ordering).
- **AC-10.** *Given* body content containing the anchor text "click here", *When* CI runs, *Then* the
  lint rule fails naming the file and line.

## 8. Data Model

No database changes. Manifest extensions and build artefacts:

```ts
// route-manifest.ts additions
type RouteDescriptor = {
  // …SEO.1 fields
  parent?: string          // '/resources' — drives breadcrumbs, depth, hub membership
  hub?: boolean            // true for pages that must link to all children
  cluster?: string         // 'assessment' | 'adaptive' | … — drives Related fallback
  relatedTo?: string[]     // explicit related paths, highest priority
  navGroup?: 'platform'|'solutions'|'resources'|'trust'|'company'
}
```

| Artefact | Path | Purpose |
|---|---|---|
| Redirect map | `www/src/lib/redirects.ts` | FR-7 |
| Link graph | `dist/.link-graph.json` | `{nodes:[{path,depth,inbound,outbound}],edges:[{from,to,anchor}]}` |
| Host redirects | `dist/_redirects` | Rendered from the map |

## 9. API Surface

No new HTTP routes. Course subject hubs (`/courses/subject/:subject`) require the marketplace API to
expose subject facets — specified in [SEO.11](SEO.11-marketplace-catalog-seo.md), not here.

## 10. UI / UX

- **New pages:** `/platform` + six children, `/resources` (content hub), `/trust` (trust hub),
  `/compare` and `/integrations` indexes (populated by SEO.9).
- **Modified components:** `header.tsx` (real navigation + mega-menu), `site-footer.tsx` (sitemap
  footer), new `breadcrumbs.tsx`, new `related-content.tsx`, new `hub-index.tsx`.
- **Key flows**
  1. Home → Platform menu → `/platform/adaptive-learning` → "See it for K-12" → `/k12` → Pricing.
  2. AI citation → `/resources/guides/rubric-design` → Related → `/blog/effective-rubrics…` →
     in-body link → `/platform/assessment` → `/get-started`.
  3. Procurement → `/trust` → `/accessibility/vpat` → download.
- **States** — hub with an empty cluster renders an honest "coming soon" only if it must exist for IA
  reasons; preferred behaviour is that a hub does not ship until it has ≥3 children (prevents thin
  pages, per the SEO.10 quality floor).
- **Responsive** — mega-menu collapses to an accordion drawer under 900 px; breadcrumbs truncate to
  `… / Parent / Current` with the full trail available to screen readers.
- **Accessibility** — see NFRs; breadcrumb truncation must not hide items from assistive tech.
- **Copy & i18n** — nav labels, breadcrumb labels, "Related", hub intro copy — all keyed under
  `www.nav.*`, `www.breadcrumb.*`, `www.related.*`, `www.hub.*`.

## 11. AI / ML Considerations

Not model-touching. Two design notes for AI consumption: assistants use breadcrumb and hub structure
to decide which page is the *canonical* answer for a topic (favouring hubs for broad queries and
spokes for specific ones), and descriptive anchor text (FR-19) is a direct signal of what a linked
page is about — one of the few remaining places where our own words describe a page other than itself.

## 12. Integration Points

- **External:** host redirect support (Cloudflare `_redirects`), per SEO.1 FR-12.
- **Internal modules touched:** `www/src/components/header.tsx`,
  `www/src/components/site-footer.tsx`, `www/src/lib/site-links.ts`,
  `www/src/lib/route-manifest.ts`, `www/src/app.tsx`, new `www/src/components/breadcrumbs.tsx`,
  `www/src/components/related-content.tsx`, `www/scripts/generate-site.mjs` (link-graph emission),
  `www/src/pages/*` (breadcrumb slot).
- **Events:** none.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md) (manifest, prerendering).
- **Must ship before:** [SEO.3](SEO.3-structured-data-and-entity-graph.md) FR-10 (BreadcrumbList
  mirrors the visible trail), [SEO.7](SEO.7-help-center-expansion.md),
  [SEO.8](SEO.8-editorial-engine-and-content-calendar.md),
  [SEO.9](SEO.9-comparison-alternatives-and-integration-pages.md),
  [SEO.10](SEO.10-programmatic-utility-pages.md) — all of which add pages that need a home.
- **Shared infra:** none beyond hosting.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `/k-12` → `/k12` loses whatever equity the old URL had | L | L | It currently 404s (audit F-1), so there is nothing to lose; 301 preserves any future equity |
| Empty hub pages read as thin content | M | M | FR "hub does not ship until ≥3 children"; hubs carry real intro copy, not just link lists |
| Mega-menu hurts mobile UX or adds JS weight | M | M | CSS-only reveal, accordion on mobile, ≤12 KB header+footer budget (also enforced by SEO.4) |
| `/parents` vs `/homeschool` decision stalls | M | M | FR-4 forces a recorded decision; default if unresolved by rollout step 2 is to merge `/parents` into `/homeschool` with a 301 |
| Link-graph check becomes noisy and gets disabled | M | M | Orphan/depth checks fail; anchor-text and link-count checks warn first, promoted to fail after one release |
| Trust URL change breaks a signed procurement document | L | H | FR-11 freezes those URLs; any change requires legal sign-off |
| Footer sitemap becomes a link-farm-looking block | L | L | Grouped, labelled columns; ≤ 60 links; genuinely useful hubs only |

## 15. Rollout Plan

- **Feature flag:** `www` has no runtime flags; rollout is staged by PR.
- **Sequencing**
  1. Manifest `parent`/`hub`/`cluster` fields + link-graph emitter (no visible change).
  2. Redirect map + validation; `/k-12` → `/k12`; `/self-learner` 301 if hosting allows.
  3. Breadcrumbs component on all pages (feeds SEO.3 FR-10).
  4. Header navigation + footer sitemap.
  5. Hub pages: `/platform` + children, `/resources`, `/trust`.
  6. Related-content module + in-body link lint (warn → fail).
  7. Enable orphan/depth CI gates.
- **Dogfood:** Marketing walks the three key flows on staging; procurement-facing trust paths verified
  by whoever owns RFP responses.
- **GA criteria:** AC-1…AC-10 pass; GSC shows no increase in "Discovered – currently not indexed";
  average crawl depth in GSC settles at ≤ 3.
- **Rollback:** per-step revert. The redirect map is additive and safe to keep even if nav changes
  revert.

## 16. Test Plan

- **Unit** — breadcrumb derivation from `parent` chain; related-content selection and deterministic
  ordering; redirect-map flattening, cycle detection, chain collapse; anchor-text lint rule.
- **Integration** — build emits `.link-graph.json`; orphan and depth assertions run against a fixture
  site with a deliberately orphaned page.
- **End-to-end (Playwright)** — JS-disabled nav link presence (AC-6); keyboard mega-menu operation
  (AC-7); breadcrumb rendering and `aria-current` (AC-5); the three key flows click-through.
- **Security** — assert `rel` attributes on external and UGC links; assert redirects cannot be
  open-redirected to an external origin.
- **Accessibility** — axe on header/footer/breadcrumbs; NVDA + VoiceOver scripts for the mega-menu
  and truncated breadcrumbs; 200% zoom and 320 px viewport checks.
- **Performance / load** — header+footer HTML size assertion; link-graph checker runtime assertion;
  Lighthouse re-run on the ten benchmark URLs after nav changes.
- **Manual exploratory** — crawl staging with Screaming Frog (or `wget --spider`), review depth
  distribution, orphan list, redirect chains, and anchor-text diversity.

## 17. Documentation & Training

- `www/docs/url-policy.md` — the rules, how to rename a URL, how the redirect map works.
- `www/docs/information-architecture.md` — the IA tree, what a hub owes its cluster, where new pages
  go.
- Update the add-a-page checklist with `parent`, `cluster`, `relatedTo` and the ≥3-internal-links rule.
- Runbook: diagnosing an orphan-page CI failure; adding a redirect safely.

## 18. Open Questions

1. `/parents` — distinct audience page or merge into `/homeschool`? (FR-4; blocks step 2 of rollout.)
2. Does `/platform/*` conflict with any planned product-marketing naming, or should it be
   `/product/*`? (Pick one and never move it.)
3. Do we want site search now (it would let SEO.3 FR-8 emit `SearchAction`), or defer?
4. Should `/courses` live under `/marketplace` for clarity, given it is a distinct business? (Would
   be a URL change with a redirect — decide before SEO.11 scales the catalog.)
5. Who owns nav copy and the hub intro copy — Marketing or Docs?

## 19. References

- Existing files: `www/src/components/header.tsx`, `www/src/components/site-footer.tsx`,
  `www/src/lib/site-links.ts`, `www/src/app.tsx`, `www/src/pages/*`
- Audit findings: [F-19, F-20](audit.md#s3--performance--ia)
- Research: [§7 Content strategy: concentration beats volume](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)
- External: [Google — URL structure best practices](https://developers.google.com/search/docs/crawling-indexing/url-structure),
  [Google — redirects and Search](https://developers.google.com/search/docs/crawling-indexing/301-redirects),
  [W3C WAI — breadcrumb pattern](https://www.w3.org/WAI/ARIA/apg/patterns/breadcrumb/)
- Related plans: [SEO.1](SEO.1-static-rendering-and-crawlability.md),
  [SEO.3](SEO.3-structured-data-and-entity-graph.md), [SEO.7](SEO.7-help-center-expansion.md),
  [SEO.9](SEO.9-comparison-alternatives-and-integration-pages.md),
  [UX.7 — navigation IA](../ui-ux/UX.7-navigation-information-architecture.md)
