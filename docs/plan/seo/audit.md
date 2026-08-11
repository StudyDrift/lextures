# www SEO Audit — Current State (2026-08-10)

> Measured audit of `www/` against the [late-2026 landscape](research.md). Every finding below was
> verified against the repository at commit `0e68a462`. Findings are ordered by severity.
> Each maps to the plan that fixes it.

---

## Executive summary

The marketing site is a **single-bundle client-rendered SPA on GitHub Pages**. Of the **31 URLs we
advertise in `sitemap.xml`, only 3 exist as HTML files**. The other 28 are served by GitHub Pages'
`404.html` — meaning they return **HTTP 404** and rely on a JavaScript redirect to render.

This is not a tuning problem. Today, an AI crawler that does not execute JavaScript — which is all of
them ([research §2](research.md#2-ai-crawlers-do-not-run-javascript)) — asking for
`https://lextures.com/pricing` receives a 404 with an **empty `<body>`**. There is no pricing page,
no K-12 page, no higher-ed page, no blog post, and no help article in the retrievable web as far as
ChatGPT, Claude, Perplexity, or Copilot are concerned.

**Headline numbers**

| Metric | Today | Target |
|---|---|---|
| URLs in sitemap | 31 | 300+ |
| URLs that return `200 OK` with rendered HTML | **3** (`/`, `/courses`, `/self-learner`) | 100% of sitemap |
| Pages with a unique `<title>` / `description` / `canonical` in the served HTML | **1** (`/`) | 100% |
| Pages calling `useDocumentHead` at runtime | **3 of 21** (14%) | 100% |
| Structured-data types emitted | 1 (`Course`, and only when prerender runs — it does not in CI) | 8+ |
| `llms.txt` | absent | present, ≤200 curated URLs |
| Named-author bylines | 0 (all posts are "Lextures Team") | 100% of editorial |
| Blog posts published | 5, all May 2026 | 4–6/month |
| Help-center articles | 6 | 60+ |
| Comparison / alternatives pages | 0 | 24+ |
| JS shipped to render any page | **583 KB** single chunk | <150 KB critical path |

---

## S0 — Critical: the site is not crawlable

### F-1. 28 of 31 sitemap URLs return HTTP 404

`www/dist/` contains exactly four HTML entry points:

```
dist/index.html                 → /
dist/courses/index.html         → /courses
dist/self-learner/index.html    → /self-learner  (meta-refresh stub)
dist/404.html                   → everything else
```

Every other route — `/pricing`, `/pricing/calculator`, `/k-12`, `/higher-ed`, `/homeschool`,
`/parents`, `/get-started`, `/request-information`, `/blog`, `/blog/*` (5), `/docs`, `/docs/*` (6),
`/privacy`, `/privacy/history`, `/terms`, `/terms/history`, `/security`, `/accessibility`,
`/accessibility/vpat`, `/privacy-rights/california` — has **no file on disk**. GitHub Pages therefore
serves `404.html` with a **404 status code**, whose entire body is a redirect script
(`www/public/404.html:6-31`) that bounces the browser to `/?/pricing`, which `www/index.html:35-49`
then rewrites back with `history.replaceState`.

Consequences:
- **Googlebot** treats a 404 as a removal signal. Pages cannot hold a stable index entry, and any
  that do get indexed via the redirect are indexed at the *wrong URL* (`/?/pricing`).
- **AI crawlers** get `<body></body>`. Zero tokens of Lextures content enter their corpus.
- **Social/link unfurls** for every blog post show the generic homepage OG card.
- We are actively telling Google, via `sitemap.xml`, to go fetch 28 URLs that 404 — a trust signal
  in the wrong direction.

→ **[SEO.1](SEO.1-static-rendering-and-crawlability.md)**

### F-2. The course prerender that exists is disabled in production

`scripts/prerender-courses.mjs` (423 lines) correctly generates per-course HTML with title,
description, canonical, OG tags and `Course` JSON-LD. But
`.github/workflows/pages-www.yml:38-40` sets:

```yaml
env:
  SKIP_COURSE_PRERENDER: "1"
```

So on every production deploy the script writes only the `/courses` shell and the sitemap. **No
course detail page has ever been prerendered in production.** The one piece of real SEO
infrastructure we built is switched off.

→ **[SEO.1](SEO.1-static-rendering-and-crawlability.md)**, **[SEO.11](SEO.11-marketplace-catalog-seo.md)**

### F-3. 18 of 21 pages inherit the homepage's metadata

`useDocumentHead` is called by exactly three pages: `courses-page.tsx`, `course-detail-page.tsx`,
`homeschool-page.tsx`. Every other page — including `/pricing`, `/k-12`, `/higher-ed`, `/security`,
every blog post and every help article — renders under the homepage `<title>` and meta description
hard-coded in `www/index.html:8-19`:

> "Lextures — The learning environment that adapts"

Even in the (rare) case where Google does render the JS, it sees 18 pages with identical titles and
descriptions — a textbook duplicate-metadata pattern that suppresses all of them.

→ **[SEO.1](SEO.1-static-rendering-and-crawlability.md)**

---

## S1 — Major: AI-search readiness is absent

### F-4. `robots.txt` says nothing about AI crawlers

`www/public/robots.txt` is 6 lines:

```
User-agent: *
Allow: /

Allow: /courses
Allow: /courses/

Sitemap: https://lextures.com/sitemap.xml
```

The two `Allow: /courses` lines are redundant under `Allow: /`. More importantly there is no explicit
posture for `GPTBot`, `OAI-SearchBot`, `ChatGPT-User`, `ClaudeBot`, `Claude-SearchBot`, `Claude-User`,
`PerplexityBot`, `Google-Extended`, `CCBot`, or `Bingbot`. A permissive wildcard *works* today, but it
is undocumented intent — one defensive edit by anyone unfamiliar with the
[three crawler jobs](research.md#2-ai-crawlers-do-not-run-javascript) silently costs 18–34% of
citations per blocked engine.

→ **[SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md)**

### F-5. No `llms.txt`, no plain-text content mirror

Nothing at `/llms.txt` or `/llms-full.txt`. Given F-1, even a crawler that *wanted* to understand the
site has no map and no HTML to fall back on.

→ **[SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md)**

### F-6. `lastmod` is the build date on every URL

`buildSitemap` stamps every static route with the same build timestamp — all 31 URLs currently claim
`2026-08-11`. A sitemap where everything changed on the same day, every deploy, carries no freshness
information; search engines learn to ignore the field. Real per-document `lastmod` (git mtime for
markdown, `updatedAt` for courses) is required before `lastmod`-driven recrawl works.

→ **[SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md)**

### F-7. No Bing / IndexNow path

There is no Bing Webmaster Tools verification file, no IndexNow key, and no post-deploy submission
step. Bing's index is the retrieval layer for ChatGPT Search and Copilot — together the largest
non-Google answer surface. We are not in it.

→ **[SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md)**

### F-8. Structured data is one type, on pages that do not ship

`document-head.ts` supports exactly one JSON-LD node, keyed to the hard-coded element id
`course-json-ld` (`www/src/lib/document-head.ts:12`). There is no `Organization`, no `WebSite`, no
`BreadcrumbList`, no `Article`, no `SoftwareApplication`, no `Offer`, no `ItemList` carousel, and no
`Person` author entity. The single-id design means a page **cannot emit two schema nodes**.

→ **[SEO.3](SEO.3-structured-data-and-entity-graph.md)**

### F-9. No brand entity anywhere

No `/about` entity-home page. No `sameAs` array. No Wikidata item. No `founder`,
`foundingDate`, `parentOrganization`, or `knowsAbout`. Given that branded mentions correlate 0.664
with AI citation versus 0.218 for backlinks, this is the cheapest unclaimed lever we have.

→ **[SEO.3](SEO.3-structured-data-and-entity-graph.md)**

### F-10. `og:image` is an SVG

`DEFAULT_OG_IMAGE = 'https://lextures.com/assets/lextures-mark.svg'`
(`www/src/lib/document-head.ts:11`). **LinkedIn, X, Facebook, Slack and iMessage do not render SVG
OG images.** Every share of every Lextures URL currently produces a blank or broken card, on the one
surface (social) where a mention is cheapest to earn.

→ **[SEO.14](SEO.14-multimodal-video-images-and-social-assets.md)**

---

## S2 — Major: content depth and E-E-A-T

### F-11. Zero named authors

All five blog posts are bylined `"Lextures Team"` (`www/src/blog/*.md` frontmatter). There is no
author page, no credentials, no `Person` schema, no `sameAs` to a LinkedIn or ORCID profile. E-E-A-T
in an education/YMYL-adjacent category has no anchor.

→ **[SEO.3](SEO.3-structured-data-and-entity-graph.md)**, **[SEO.8](SEO.8-editorial-engine-and-content-calendar.md)**

### F-12. Publishing stopped 76 days ago

| Post | Date |
|---|---|
| the-synthetic-renaissance | 2026-05-06 |
| adaptive-ai-and-education | 2026-05-06 |
| rethinking-assessment-in-the-ai-era | 2026-05-15 |
| effective-rubrics-in-the-age-of-ai | 2026-05-18 |
| blooms-taxonomy-in-the-age-of-ai | 2026-05-26 |

Five posts, all in a three-week burst, nothing since. The existing posts are genuinely good —
opinionated, technical, correctly positioned for the AI-era assessment conversation — which makes the
stall more costly. There is no calendar, no cluster map, and no brief template.

→ **[SEO.8](SEO.8-editorial-engine-and-content-calendar.md)**

### F-13. Content is essay-shaped, not passage-shaped

The posts open with narrative section headers ("The Problem With 'Personalized'", "The Traditional
Ascent") rather than the question a searcher typed. There are no TL;DR blocks, no direct-answer
paragraphs, no definition boxes, no comparison tables, no "last reviewed" dates. Outbound citations
exist in `the-synthetic-renaissance.md` (UNESCO, Grand View Research) and nowhere else — despite
authoritative outbound citation being the **single highest-impact** AI-visibility factor at +132%.

→ **[SEO.6](SEO.6-answer-first-content-system.md)**

### F-14. The help center is 6 articles against a platform with hundreds of features

`www/src/docs/` contains: creating-a-new-course, finding-your-course,
navigating-the-course-interface, self-hosting, connecting-lextures-to-zapier, using-lextures-with-make.

Meanwhile the product ships adaptive content, IRT-based quizzing, spaced review, outcomes/standards
alignment, rubrics, peer review, gradebook curving, what-if grades, SIS roster sync, LTI, SSO, parent
portal, accommodations engine, marketplace + coupons, interactive quizzes, collaboration boards, and
an AI stack. **None of it is documented on a crawlable public URL.** Help content is the highest-value
AI-citable asset a SaaS owns — it answers literal questions with literal answers.

→ **[SEO.7](SEO.7-help-center-expansion.md)**

### F-15. Zero bottom-of-funnel pages

No `/compare/*`, no `/alternatives/*`, no `/integrations/*`, no `/glossary/*`. Competitors own these
queries outright — D2L ranks for *Moodle alternatives*, Jotform and Teachfloor for *Canvas LMS
alternatives*. This is the segment that survived the AI-era traffic decline and converts at 10–20%.

→ **[SEO.9](SEO.9-comparison-alternatives-and-integration-pages.md)**, **[SEO.10](SEO.10-programmatic-utility-pages.md)**

---

## S3 — Performance & IA

### F-16. 583 KB of JavaScript in one chunk

`dist/assets/index-Cygo_j5N.js` is **583.5 KB** uncompressed, plus **56.5 KB** CSS — a single bundle
with no route-level code splitting. Every visitor to `/privacy` downloads the pricing calculator, the
course marketplace client, the markdown renderer and the hero canvas. Against the March-2026
thresholds (**LCP < 2.0 s**, **INP < 200 ms**) this is the dominant risk, and INP is the most-failed
vital industry-wide.

→ **[SEO.4](SEO.4-core-web-vitals-and-page-experience.md)**

### F-17. Fonts are loaded twice, one of them render-blocking third-party

`www/index.html:20-21` preloads self-hosted `lextures-400.woff2` / `lextures-600.woff2`, then
`index.html:22-27` `preconnect`s to `fonts.googleapis.com` + `fonts.gstatic.com` and loads a
**render-blocking stylesheet** for IBM Plex Mono + Spectral. Two font systems, one of them a
third-party round-trip on the critical path — and a GDPR surface (Google Fonts CDN logs visitor IPs)
that sits oddly next to our own privacy positioning.

→ **[SEO.4](SEO.4-core-web-vitals-and-page-experience.md)**

### F-18. `hero-canvas.tsx` runs an animation loop above the fold

284 lines of canvas animation rendering in the LCP region. Needs measurement against INP and against
`prefers-reduced-motion` (which the AN plan set already established as a house rule).

→ **[SEO.4](SEO.4-core-web-vitals-and-page-experience.md)**

### F-19. The header exposes three links; there is no internal link graph

`www/src/components/header.tsx` emits exactly `/`, `/#institutions`, `/get-started`. There are no
breadcrumbs anywhere, no related-content modules on blog or docs, no hub pages, and no cross-links
between the segment pages (`/k-12`, `/higher-ed`, `/homeschool`, `/parents`) and the content that
supports them. PageRank and crawl budget have nowhere to flow.

→ **[SEO.5](SEO.5-information-architecture-and-internal-linking.md)**

### F-20. URL policy drift

- `/k-12` uses a hyphen where the market term and every competitor URL is `k12`.
- `/parents` and `/homeschool` overlap in audience with no canonical relationship declared.
- `/self-learner` → `/homeschool` is a **meta-refresh + client `location.replace`**, not a 301. On
  GitHub Pages we cannot issue a true 301, which argues for moving www to a host that can
  (see [SEO.1 §14](SEO.1-static-rendering-and-crawlability.md#14-risks--mitigations)).
- No trailing-slash policy is declared or enforced.

→ **[SEO.5](SEO.5-information-architecture-and-internal-linking.md)**

---

## S4 — Measurement & governance

### F-21. GA4 is installed and configured for the wrong era

`G-JX182Q6KKX` is wired in `index.html:29-37` with default config. There is no custom channel group
for AI assistants (GA4's native "AI Assistant" channel excludes Perplexity and referrer-less
sessions), no UTM convention, no CRM handoff field, and no server-side view of AI-bot crawl activity.
We cannot currently answer "did ChatGPT send us anyone?" — let alone "are we cited?".

→ **[SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md)**

### F-22. No SEO regression gate in CI

`.github/workflows/pages-www.yml` runs `oxlint` and `vite build`. Nothing checks that a new route has
a title, that titles are unique, that the sitemap matches the route table, that JSON-LD validates, or
that a page returns 200. `www/package.json`'s test script covers six unit files, none of them SEO
invariants. The `lighthouse.yml` workflow exists but does not gate the www deploy.

→ **[SEO.16](SEO.16-seo-governance-and-ci-guardrails.md)**

### F-23. No internationalisation signals

The product ships i18n including an RTL locale (per the UX plan set), but www is English-only with no
`hreflang`, no locale routing, and `<html lang="en">` hard-coded. Non-English institutional buyers
find nothing.

→ **[SEO.17](SEO.17-international-seo-and-hreflang.md)**

---

## What is already good (do not regress it)

- **`prerender-courses.mjs` is well-built** — pure helpers, unit-tested (`prerender-courses.test.mjs`),
  fails loudly when the API is unreachable, correct escaping. It is the right shape to generalise
  into a full SSG step rather than replace.
- **`document-head.ts` correctly shares logic** between the runtime hook and the prerenderer, and is
  unit-tested. It needs multi-node JSON-LD support, not a rewrite.
- **Trust surfaces already exist and are genuinely differentiated**: VPAT 2.5, accessibility
  conformance page, security/disclosure policy, privacy + terms with public version history,
  California privacy rights, self-hosting docs. In a post-breach market where "Canvas alternatives"
  interest spiked on security grounds, this is our strongest unexploited content asset — it is
  currently 404ing.
- **Existing blog posts are high quality** and correctly aimed at the AI-era assessment conversation.
  They need reformatting for extraction, not replacement.
- **Design system and accessibility work is ahead of most competitors** (WCAG 2.2 AA remediation
  landed in UX.5/UX.6), which supports both a page-experience story and a linkable-asset story.

---

## Severity → plan map

| Finding | Severity | Plan |
|---|---|---|
| F-1, F-2, F-3 | **BLOCKER** | [SEO.1](SEO.1-static-rendering-and-crawlability.md) |
| F-4, F-5, F-6, F-7 | BLOCKER | [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md) |
| F-8, F-9, F-11 | MAJOR | [SEO.3](SEO.3-structured-data-and-entity-graph.md) |
| F-16, F-17, F-18 | MAJOR | [SEO.4](SEO.4-core-web-vitals-and-page-experience.md) |
| F-19, F-20 | MAJOR | [SEO.5](SEO.5-information-architecture-and-internal-linking.md) |
| F-13 | MAJOR | [SEO.6](SEO.6-answer-first-content-system.md) |
| F-14 | MAJOR | [SEO.7](SEO.7-help-center-expansion.md) |
| F-12 | MAJOR | [SEO.8](SEO.8-editorial-engine-and-content-calendar.md) |
| F-15 | MAJOR | [SEO.9](SEO.9-comparison-alternatives-and-integration-pages.md) · [SEO.10](SEO.10-programmatic-utility-pages.md) |
| F-2 (catalog scale) | MAJOR | [SEO.11](SEO.11-marketplace-catalog-seo.md) |
| — (opportunity) | MAJOR | [SEO.12](SEO.12-original-research-and-data-program.md) · [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md) |
| F-10 | MAJOR | [SEO.14](SEO.14-multimodal-video-images-and-social-assets.md) |
| F-21 | MAJOR | [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md) |
| F-22 | MAJOR | [SEO.16](SEO.16-seo-governance-and-ci-guardrails.md) |
| F-23 | MINOR | [SEO.17](SEO.17-international-seo-and-hreflang.md) |
