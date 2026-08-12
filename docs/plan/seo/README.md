# SEO — Organic & AI-Search Ranking Program

> 17 plans that take `www.lextures.com` from **3 crawlable pages** to a compounding organic and
> AI-citation engine. Grounded in a [measured audit](audit.md) of the current site and a
> [sourced review](research.md) of the late-2026 search landscape.

---

## The one-paragraph version

Search stopped being a ranking problem and became a *retrieval* problem. ~70% of Google searches end
without a click, 47% of AI citations come from pages ranking below #5, and every major AI crawler is
an HTML-only parser. Our marketing site is a client-rendered SPA where **28 of the 31 URLs we publish
in our own sitemap return HTTP 404 with an empty body**. We are not ranking badly — for most of the
site, we do not exist. SEO.1–SEO.4 fix that in ~6 weeks. SEO.5–SEO.11 build the content and IA that
earns rankings and citations. SEO.12–SEO.14 manufacture the off-site mentions that 85% of AI brand
references actually come from. SEO.15–SEO.17 measure it and keep it from regressing.

---

## Strategy

Five bets, in dependency order. Each is defended in [research.md §11](research.md#11-what-this-means-for-lextures--the-five-bets).

1. **Retrievability before everything.** Real HTML at `200 OK` for every URL, per-page metadata,
   correct crawler directives. Nothing compounds until this is true.
2. **Become a resolvable entity.** `Organization` + `sameAs` + Wikidata + named human authors. Branded
   mentions correlate **0.664** with AI citation vs **0.218** for backlinks — entity work is the
   cheapest lever available and takes 90–120 days to show, so it starts early.
3. **Write for passage extraction.** Answer-first blocks, self-contained 134–167-word units, tables,
   and authoritative outbound citations (**+132% visibility**, the single highest measured factor).
4. **Own the bottom of the funnel and the utility layer.** Comparisons, alternatives, integrations,
   glossary, calculators, standards browser — the queries that convert at 10–20% and that assistants
   answer literally.
5. **Earn mentions honestly.** Original research from our own anonymised outcome data, disclosed
   community participation, review-site and directory presence. Google extended its spam policies —
   including **inauthentic mentions** — to AI Overviews and AI Mode on 2026-05-15, so the program is
   built to be defensible, not clever.

### Non-negotiable constraints

- **No scaled content abuse.** Every programmatic page must let a user *do* something. The March 2026
  core update cost offenders 50–80% of traffic. [SEO.10](SEO.10-programmatic-utility-pages.md)
  encodes a hard quality floor and a `noindex` rule for pages that fall below it.
- **No astroturfing.** [SEO.13](../../completed/seo/SEO.13-offsite-entity-mentions-and-digital-pr.md) requires disclosed
  affiliation on every community post, with a named accountable owner.
- **No claim without a source.** Editorial standard in [SEO.6](SEO.6-answer-first-content-system.md);
  enforced by the CI link/citation check in [SEO.16](../../completed/seo/SEO.16-seo-governance-and-ci-guardrails.md).
- **Privacy first in research.** [SEO.12](SEO.12-original-research-and-data-program.md) runs through
  the existing DPIA process ([S06](../standards/S06-dpia-pia-algorithmic-impact.md)) with k-anonymity
  thresholds and tenant opt-out before any aggregate is published.

---

## Plan index

### Phase 0 — Technical foundation (weeks 1–6) · *blocks everything else*

| ID | Plan | Effort | Severity |
|---|---|---|---|
| **SEO.1** | [Static rendering & crawlability foundation](../../completed/seo/SEO.1-static-rendering-and-crawlability.md) | M | **BLOCKER** |
| **SEO.2** | [Crawler access, sitemaps, llms.txt & index submission](../../completed/seo/SEO.2-crawler-access-sitemaps-and-llms-txt.md) | S | **BLOCKER** |
| **SEO.3** | [Structured data & brand entity graph](../../completed/seo/SEO.3-structured-data-and-entity-graph.md) | M | MAJOR |
| **SEO.4** | [Core Web Vitals & page-experience budget](../../completed/seo/SEO.4-core-web-vitals-and-page-experience.md) | M | MAJOR — completed |

### Phase 1 — Architecture & content system (weeks 5–12)

| ID | Plan | Effort | Severity |
|---|---|---|---|
| **SEO.5** | [Information architecture, URL policy & internal linking](../../completed/seo/SEO.5-information-architecture-and-internal-linking.md) | M | MAJOR |
| **SEO.6** | [Answer-first content system & extractability primitives](SEO.6-answer-first-content-system.md) | M | MAJOR |
| **SEO.7** | [Help center expansion (6 → 60+ articles)](SEO.7-help-center-expansion.md) | L | MAJOR |

### Phase 2 — Content programs (weeks 9–52) · *the ranking engine*

| ID | Plan | Effort | Severity |
|---|---|---|---|
| **SEO.8** | [Editorial engine: pillars, clusters & 12-month calendar](../../completed/seo/SEO.8-editorial-engine-and-content-calendar.md) | XL | MAJOR — completed |
| **SEO.9** | [Comparison, alternatives & integration pages](SEO.9-comparison-alternatives-and-integration-pages.md) | L | MAJOR |
| **SEO.10** | [Programmatic utility pages](SEO.10-programmatic-utility-pages.md) | L | MAJOR |
| **SEO.11** | [Marketplace catalog SEO at scale](../../completed/seo/SEO.11-marketplace-catalog-seo.md) | M | MAJOR — completed |

### Phase 3 — Authority & off-site (weeks 13–52)

| ID | Plan | Effort | Severity |
|---|---|---|---|
| **SEO.12** | [Original research & data program](SEO.12-original-research-and-data-program.md) | L | MAJOR |
| **SEO.13** | [Off-site entity, mentions & digital PR](../../completed/seo/SEO.13-offsite-entity-mentions-and-digital-pr.md) | L | MAJOR — completed |
| **SEO.14** | [Multimodal: video, images & social preview assets](SEO.14-multimodal-video-images-and-social-assets.md) | M | MAJOR |

### Phase 4 — Measurement & governance (weeks 3–ongoing)

| ID | Plan | Effort | Severity |
|---|---|---|---|
| **SEO.15** | [Measurement: Search Console, GA4 & AI share-of-voice](../../completed/seo/SEO.15-measurement-search-console-and-ai-share-of-voice.md) | M | MAJOR — completed |
| **SEO.16** | [SEO governance, CI guardrails & content lifecycle](../../completed/seo/SEO.16-seo-governance-and-ci-guardrails.md) | M | MAJOR |
| **SEO.17** | [International SEO & hreflang](../../completed/seo/SEO.17-international-seo-and-hreflang.md) | M | MINOR — completed |

---

## Sequencing

```
Week   1   2   3   4   5   6   7   8   9  10  11  12 ......  26 ......  52
     ├───SEO.1───┤
         ├─SEO.2─┤
             ├───SEO.3───┤
             ├─────SEO.4─────┤
     ├──SEO.15 (baseline)──┤·············· ongoing ··············▶
                 ├──SEO.5──┤
                     ├──SEO.6──┤
                     ├──────────SEO.7──────────┤
                         ├────────────────SEO.8 ───────────────────▶
                             ├──────────SEO.9 (1/mo × 12)─────────▶
                                 ├────────SEO.10────────┤
                                 ├──SEO.11──┤
                                     ├──────SEO.12 (2 reports/yr)─▶
                                     ├──────SEO.13 ───────────────▶
                                         ├────SEO.14────┤
                             ├──SEO.16──┤·· ongoing gate ·········▶
                                                     ├──SEO.17──┤
```

**Hard dependencies**

- SEO.1 blocks **everything**. No content plan ships value while pages 404.
- SEO.2 needs SEO.1's route manifest to build honest sitemaps.
- SEO.3 needs SEO.1's multi-node JSON-LD support.
- SEO.6 blocks SEO.7 / SEO.8 / SEO.9 (they consume its content primitives).
- SEO.15 baseline must land **before** SEO.8 starts, or we cannot attribute the content program.
- SEO.16 must gate the deploy before SEO.7/8/9/10 start producing pages at volume.

---

## Targets

Measured from a baseline captured in [SEO.15](../../completed/seo/SEO.15-measurement-search-console-and-ai-share-of-voice.md) at week 3.

| KPI | Baseline (Aug 2026) | 3 months | 6 months | 12 months |
|---|---|---|---|---|
| Indexed pages (GSC) | ~3 | 120 | 260 | 500 |
| Indexed pages (Bing) | ~0 | 120 | 260 | 500 |
| Non-brand organic clicks / mo | ~0 | 400 | 2,500 | 12,000 |
| Non-brand keywords in top 10 | ~0 | 25 | 150 | 600 |
| **AI Share of Voice** (60-prompt set, 6 engines) | measure at wk 3 | 5% | 12% | **≥20%** |
| Distinct AI-cited URLs / mo | 0 | 15 | 60 | 200 |
| AI-assistant referral sessions / mo | unknown | 150 | 900 | 4,000 |
| Referring domains | baseline at wk 3 | +25 | +90 | +250 |
| Third-party brand mentions / mo | baseline at wk 3 | +40 | +150 | +400 |
| CWV: % URLs passing LCP<2.0s / INP<200ms / CLS<0.1 | unknown | 90% | 100% | 100% |
| MQLs attributed to organic + AI | baseline at wk 3 | +20% | +75% | +200% |

**Leading indicators to watch weekly** (they move before the lagging ones): crawl requests by
AI user-agent in edge logs, `200 OK` ratio on sitemap URLs, average passage-extractability score on
new content, and net new third-party mentions.

---

## Owners

| Area | Proposed owner |
|---|---|
| SEO.1, SEO.2, SEO.4, SEO.16, SEO.17 | Web platform |
| SEO.3, SEO.5, SEO.11, SEO.14 | Web platform + Marketing |
| SEO.6, SEO.7 | Docs / Content |
| SEO.8, SEO.9, SEO.10, SEO.12, SEO.13 | Marketing (content lead) |
| SEO.15 | Growth / Analytics |

---

## Conventions

- File naming: `SEO.{n}-{kebab-slug}.md`, per [`docs/plan/README.md`](../README.md#conventions).
- Every plan fills every section of [`_TEMPLATE.md`](../_TEMPLATE.md).
- Claims about the search landscape cite [`research.md`](research.md); claims about our current
  state cite a finding ID in [`audit.md`](audit.md).
- Content plans express volume as **cadence** (pages/month), never as a one-time dump — that is the
  behavioural difference between a content program and scaled content abuse.
