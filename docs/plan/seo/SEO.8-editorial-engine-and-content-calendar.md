# SEO.8 — Editorial Engine: Pillars, Clusters & 12-Month Calendar

> Implementation plan. Source: [docs/plan/seo/audit.md](audit.md) §S2 (F-12).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.8 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN (5 posts, all published in a 3-week burst in May 2026; nothing in 76 days; no calendar, no cluster map, no briefs) |
| **Estimated effort** | XL (>2mo — a continuous program, not a project) |
| **Owner (proposed)** | Marketing (content lead) |
| **Depends on** | SEO.1, SEO.5, SEO.6, SEO.15 (baseline) |
| **Unblocks** | SEO.12 (research slots in the calendar), SEO.13 (content is the PR asset) |

---

## 1. Problem Statement

We published five genuinely good articles in three weeks in May 2026 and have published nothing since
(audit F-12). There is no calendar, no cluster map, no brief template, no keyword research, and no
owner. Topical authority in 2026 is earned by *concentration*: "25 well-connected articles on a topic
rank better than 250 scattered ones," and a pillar-plus-cluster architecture correlates with ~55%
organic lift within six months ([research §7](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)).
Five orphaned essays across four unrelated subjects is the scattered pattern, not the concentrated
one. We need an engine — pillars, clusters, cadence, briefs, review, refresh — that produces
compounding coverage in a small number of subjects we can credibly own.

## 2. Goals

- Establish **6 pillars** we intend to own, each with a comprehensive pillar page and a linked
  cluster of 12–20 articles.
- Sustain a **cadence of 5 articles per month** (≈60/year) plus 6 pillar pages, published on a
  calendar rather than in bursts.
- Reach **top-10 rankings for ≥150 non-brand keywords** and ≥60 AI-cited URLs within 12 months.
- Make every article traceable: brief → target question → cluster → internal links → measured outcome.
- Institute a **refresh loop** so the corpus improves rather than only growing.

## 3. Non-Goals

- Bottom-of-funnel comparison/alternatives/integration pages (SEO.9) and utility pages (SEO.10) —
  they have their own cadence and are counted separately.
- Original research reports (SEO.12) — they occupy reserved slots in this calendar but are planned
  separately because of their data and privacy requirements.
- Paid promotion or ads.
- Content in languages other than English (SEO.17).

## 4. Personas & User Stories

- **As a higher-ed instructional designer**, I want a comprehensive, current guide to assessment
  design under generative AI, so that I stop assembling it from twelve half-answers.
- **As a K-12 curriculum director**, I want practical, standards-aware writing about mastery and
  outcomes, so that I can bring a defensible approach to my board.
- **As a homeschool parent**, I want plain-language explanations of spaced repetition and mastery
  learning, so that I can run a better school year.
- **As an AI assistant**, I want a densely-interlinked set of Lextures pages on adaptive learning, so
  that I treat the domain as authoritative on the topic and cite it.
- **As the content lead**, I want a brief template and a queue, so that writers are never blocked and
  quality is predictable.
- **As the CEO**, I want to see which articles produced pipeline, so that the program is funded on
  evidence.

## 5. Functional Requirements

**Pillars & clusters**

- **FR-1.** The program MUST commit to exactly **6 pillars**, each a `/resources/guides/:slug` page of
  2,500–4,000 words that comprehensively answers its topic and links to every article in its cluster:

  | # | Pillar | URL | Primary audience | Cluster size |
  |---|---|---|---|---|
  | P1 | Adaptive learning: how it actually works | `/resources/guides/adaptive-learning` | HE · K12 | 18 |
  | P2 | Assessment design in the age of generative AI | `/resources/guides/assessment-design-ai` | HE · K12 | 20 |
  | P3 | Grading, feedback and academic integrity | `/resources/guides/grading-and-integrity` | HE · K12 | 16 |
  | P4 | Standards, outcomes and mastery-based grading | `/resources/guides/mastery-and-standards` | K12 | 14 |
  | P5 | Choosing and running a learning platform | `/resources/guides/choosing-an-lms` | K12 · HE | 16 |
  | P6 | Teaching at home: curriculum, pacing and evidence | `/resources/guides/homeschool-teaching` | HS | 12 |

- **FR-2.** Every cluster article MUST link **up** to its pillar and the pillar MUST link **down** to
  every article in its cluster (SEO.5 FR-17). Cross-cluster links are encouraged where genuine.
- **FR-3.** Each pillar MUST be assigned a `cluster` id used by the manifest for Related-content
  computation, hub membership, and reporting.
- **FR-4.** A pillar page MUST NOT ship before ≥5 of its cluster articles exist, so it launches as a
  real hub rather than a stub.

**Keyword & question research**

- **FR-5.** Every article MUST originate from a **brief** in `docs/plan/seo/briefs/` containing:
  primary question (verbatim, as a user would ask it), 3–8 secondary questions, target keywords with
  volume/difficulty, current SERP and AI-answer analysis (who is cited today and why), the angle that
  differentiates us, required primary sources, target word count, internal links to include, the
  author, and the success metric.
- **FR-6.** Research MUST cover **both** surfaces: classical keyword data (volume, difficulty, SERP
  features) **and** an AI-answer probe — run the primary question against the six tracked assistants
  and record who is cited today. An article whose question already returns a well-cited comprehensive
  answer from a stronger domain MUST be re-angled, not written head-on.
- **FR-7.** The backlog MUST be prioritised by a documented score:
  `(intent_value × 3) + (ai_citation_gap × 3) + (search_volume_band × 2) + (our_credibility × 2) − difficulty`.
  `intent_value` reflects proximity to a buying decision; `ai_citation_gap` is high when assistants
  currently cite weak or no sources.

**Cadence & calendar**

- **FR-8.** The program MUST publish **5 articles per month**, on fixed days (e.g. Tue/Thu), with the
  next 8 weeks always fully briefed and assigned. Bursts are prohibited — steady cadence is both a
  freshness signal and the only sustainable staffing model.
- **FR-9.** The 12-month calendar MUST be maintained in `docs/plan/seo/calendar.md` with this shape:

  | Month | Pillar focus | Articles | Reserved slots |
  |---|---|---|---|
  | M1 | P2 (assessment) | 5 | — |
  | M2 | P2 + P1 | 5 | P2 pillar page |
  | M3 | P1 (adaptive) | 5 | P1 pillar page |
  | M4 | P1 + P5 | 5 | SEO.12 research report #1 |
  | M5 | P5 (choosing an LMS) | 5 | P5 pillar page |
  | M6 | P3 (grading) | 5 | Mid-year refresh sprint |
  | M7 | P3 + P4 | 5 | P3 pillar page |
  | M8 | P4 (standards) | 5 | back-to-school seasonal push |
  | M9 | P4 + P6 | 5 | P4 pillar page |
  | M10 | P6 (homeschool) | 5 | SEO.12 research report #2 |
  | M11 | P6 + P1 refresh | 5 | P6 pillar page |
  | M12 | Cross-cluster gaps | 5 | Annual refresh + next-year planning |

- **FR-10.** Seasonality MUST be planned: back-to-school (Jul–Aug), semester start (Jan), budget/RFP
  season (Feb–Apr for K-12 districts), and end-of-year grading (May, Dec). Seasonal articles MUST be
  published **6–8 weeks ahead** of the demand peak.
- **FR-11.** Each month MUST reserve one slot for a **rapid-response** piece (a standards change, a
  major AI release, an incident affecting the category) to be used or forfeited — not banked.

**Quality & review**

- **FR-12.** Every article MUST meet the [SEO.6](SEO.6-answer-first-content-system.md) contract with a
  score ≥8.0 and MUST be reviewed by a second person before publication.
- **FR-13.** Every article MUST have a **named human author** from the registry (SEO.3 FR-20), and
  ≥30% of articles SHOULD have an external expert contributor or quoted practitioner, since
  first-hand experience is the "E" in E-E-A-T that a vendor blog most often lacks.
- **FR-14.** Articles MUST contain ≥3 citations to primary sources (SEO.6 FR-5); the shared source
  library per pillar MUST be maintained so this is not per-article archaeology.
- **FR-15.** No article may be published solely to hit cadence. A missed slot MUST be recorded with a
  reason in the calendar rather than filled with a thin piece — the March 2026 enforcement precedent
  makes filler an active risk, not just waste.

**Refresh**

- **FR-16.** Every article MUST carry a `reviewDue` date (default: published + 12 months; 6 months for
  anything referencing AI capabilities, which move fast).
- **FR-17.** A **quarterly refresh sprint** MUST update the 10 highest-traffic and 10
  highest-potential-but-declining articles: refresh data, re-verify competitor/product claims, add new
  sections for questions that emerged, update `updated` and `dateModified`.
- **FR-18.** Underperforming articles (no impressions after 6 months, no citations, no internal link
  value) MUST be consolidated into a stronger page with a 301 or improved — not left to rot. Pruning
  decisions MUST be recorded.
- **FR-19.** The five existing posts MUST be retrofitted to the contract (SEO.6) and re-slotted into
  P1/P2/P3 clusters in month 1, with `updated` dates reflecting the substantive revision.

## 6. Non-Functional Requirements

- **Performance** — articles are static content pages under the SEO.4 static budget (≤60 KB JS).
- **Security** — external contributor content goes through the same MDX allowlist and review as
  internal (SEO.6 NFR).
- **Privacy & Compliance** — quoting a customer or practitioner requires written permission recorded
  in the consent ledger ([S04](../standards/S04-unified-consent-preference-ledger.md)); no student
  data, no identifiable classroom examples without consent. Claims touching FERPA/GDPR/WCAG require
  compliance review (SEO.6 FR-6).
- **Accessibility** — all content components are WCAG 2.2 AA (SEO.6); any diagram must have a text
  equivalent and must be legible in both themes (SEO.14).
- **Scalability** — the process must sustain 5/month with 1 content lead + 2–3 rotating SME authors;
  briefs and the source library exist to make writing cheaper over time.
- **Reliability** — the calendar is the source of truth; the 8-week briefed buffer absorbs illness,
  launches, and holidays without breaking cadence.
- **Observability** — per-article dashboard: impressions, clicks, position, internal links in/out,
  AI citations, assisted conversions, time-to-first-ranking. Rolled up per pillar.
- **Maintainability** — briefs, calendar, and the source library live in the repo next to this plan
  so they are versioned and reviewable.
- **Internationalization** — English only; SEO.17 will select a subset of top performers to localise.
- **Backward compatibility** — the five existing URLs MUST NOT change; retrofits happen in place.

## 7. Acceptance Criteria

- **AC-1.** *Given* month 3 of the program, *When* the calendar is reviewed, *Then* 15 articles have
  been published on their scheduled dates, and the next 8 weeks are fully briefed and assigned.
- **AC-2.** *Given* any published article, *When* checked, *Then* it links up to its pillar, appears
  in that pillar's cluster list, and has ≥3 contextual internal links (SEO.5 FR-16).
- **AC-3.** *Given* a pillar page at launch, *When* checked, *Then* ≥5 cluster articles already exist
  and are linked from it (FR-4).
- **AC-4.** *Given* any published article, *When* scored, *Then* it is ≥8.0 (SEO.6) and has ≥3
  primary-source citations and a named registry author.
- **AC-5.** *Given* the brief archive, *When* audited, *Then* every published article has a brief
  containing an AI-answer probe recorded before writing began.
- **AC-6.** *Given* 6 months, *When* measured, *Then* ≥150 non-brand keywords rank in the top 10 and
  ≥60 URLs have appeared in an AI citation.
- **AC-7.** *Given* the quarterly refresh sprint, *When* it completes, *Then* 20 articles have updated
  `updated`/`dateModified` dates with substantive changes recorded in the commit.
- **AC-8.** *Given* a month where a slot was missed, *When* the calendar is reviewed, *Then* the slot
  is recorded as missed with a reason — and no article published that month scored below 8.0.
- **AC-9.** *Given* 12 months, *When* pillar performance is reviewed, *Then* each pillar has ≥12
  cluster articles and its pillar page ranks in the top 10 for its head term or is cited by ≥2 AI
  engines for its primary question.

## 8. Data Model

No database changes. Program artefacts, versioned in the repo:

```
docs/plan/seo/
  calendar.md                    # 12-month calendar, FR-9, updated monthly
  briefs/
    2026-11-rubric-design.md     # one brief per article, FR-5
  sources/
    p1-adaptive.md               # vetted primary sources per pillar, FR-14
    p2-assessment.md  …
  performance.md                 # monthly rollup, generated from SEO.15 data
```

Article front-matter additions (on top of SEO.6 FR-12):

```yaml
pillar: p2                       # p1..p6
briefRef: 2026-11-rubric-design
reviewDue: 2027-05-06
contributor: "Dr. … , instructional designer, <institution>"   # optional, consented
```

## 9. API Surface

No HTTP surface. Tooling commands:

| Command | Purpose |
|---|---|
| `npm run content:calendar` | Render the calendar with status (briefed / drafted / in review / published / missed) |
| `npm run content:gaps` | List cluster articles briefed but unpublished, and pillars below the 12-article floor |
| `npm run content:refresh-due` | List articles past `reviewDue`, sorted by traffic |

## 10. UI / UX

- **New pages:** 6 pillar guides at `/resources/guides/:slug`; `/resources/guides` index.
- **Modified:** `/blog` index gains pillar/cluster filtering, author facet, and a "Start here"
  module linking the six pillars; blog post pages gain the pillar breadcrumb (`Resources › Guides ›
  Assessment design`) and a "Part of the … guide" module.
- **Flows**
  1. Reader arrives on a cluster article → "Part of: Assessment design in the age of generative AI" →
     pillar → related cluster articles → `/platform/assessment` → `/get-started`.
  2. Reader arrives on a pillar from a head-term search → jumps to the sub-question via the TOC.
- **States** — pillar with fewer than 12 cluster articles still renders fully (it is comprehensive on
  its own); no "coming soon" placeholders.
- **Responsive** — pillar TOC becomes a sticky disclosure under 900 px.
- **Accessibility** — long pillar pages need a skip-to-content-sections mechanism, correct heading
  hierarchy (no skipped levels across 2,500–4,000 words), and a TOC that moves focus on activation.
- **Copy & i18n** — `www.guides.*`, `www.blog.filters.*`, `www.blog.partOf`.

## 11. AI / ML Considerations

- **AI-answer probing (FR-6) is the differentiating research step.** Traditional keyword tools
  describe a SERP that increasingly does not get clicked. Recording *who is cited today and why*
  identifies the gap we can actually fill — the `ai_citation_gap` term in the FR-7 priority score.
- **Drafting with AI is allowed under SEO.6's rules**: human byline, human verification of every
  statistic against a primary source, second-person review, and a named reviewer for
  compliance-adjacent claims. What is prohibited is publishing volume without that oversight — the
  precise behaviour the March 2026 core update penalised at 50–80% traffic loss.
- **Cadence is deliberately modest (5/month).** A higher rate is achievable with generation but moves
  us into the pattern enforcement targets. Concentration beats volume is not only strategy here; it is
  risk management.
- **`knowsAbout` must match the pillars** (SEO.3 FR-6): the entity graph's topical claims and the
  published corpus have to agree.

## 12. Integration Points

- **External:** a keyword/SERP data source (Ahrefs, Semrush, or similar), the SEO.15 AI-prompt harness,
  contributor agreements.
- **Internal modules touched:** `www/src/blog/*` (new articles), new `www/src/guides/*`,
  `www/src/pages/blog-index.tsx`, new `www/src/pages/guide-page.tsx`,
  `www/src/lib/route-manifest.ts`, `www/src/components/content/*` (SEO.6).
- **Events:** publication triggers IndexNow submission (SEO.2 FR-17) and a social/newsletter
  distribution step (SEO.13).

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md) (pages must be crawlable),
  [SEO.5](SEO.5-information-architecture-and-internal-linking.md) (clusters and linking),
  [SEO.6](SEO.6-answer-first-content-system.md) (the contract),
  [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md) **baseline** (or the program
  cannot be attributed).
- **Must ship before:** nothing hard-blocks on it, but
  [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md) needs a steady supply of content to
  promote, and [SEO.12](SEO.12-original-research-and-data-program.md) occupies its reserved slots.
- **Shared infra:** keyword-data subscription; contributor payment process; consent records for quotes.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Cadence collapses again after the first burst | **H** | **H** | Named owner + 8-week briefed buffer + calendar in the repo + missed slots recorded publicly (AC-8). This is the risk the plan exists to address |
| Writing capacity does not exist internally | H | H | 5/month is deliberately modest; SME rotation with the content lead editing; budget for 2 external contributors/month |
| We pick pillars we cannot credibly own | M | H | Pillars map to shipped product differentiators (adaptive, assessment, standards) where we have genuine engineering depth; P5/P6 map to segments we sell to |
| Articles rank but never convert | M | M | Every cluster links to a `/platform/*` page and a segment hub; SEO.15 tracks assisted conversions per pillar; underperformers pruned (FR-18) |
| Filler published to hit the number | M | H | FR-15 forbids it; AC-8 makes a missed slot the acceptable outcome |
| Competitors out-publish us 10:1 | H | M | Concentration strategy is the explicit answer — 25 connected beats 250 scattered; plus we hold assets they cannot copy (SEO.12 proprietary data) |
| Refresh never happens because new content is more exciting | H | M | FR-17 quarterly sprint is a calendar slot, not an intention; `content:refresh-due` makes debt visible |

## 15. Rollout Plan

- **Feature flag:** none.
- **Sequencing**
  1. **Weeks 1–2:** pillar selection sign-off; keyword + AI-probe research for P2 and P1; build the
     first 10 briefs; retrofit the five existing posts (FR-19).
  2. **Week 3:** publish first article; establish the Tue/Thu rhythm; confirm SEO.15 baseline exists.
  3. **Months 1–3:** P2 and P1 clusters; ship P2 pillar in M2, P1 in M3.
  4. **Month 4:** first SEO.12 research report lands in its reserved slot.
  5. **Month 6:** first refresh sprint; program review against AC-6.
  6. **Months 7–12:** P3, P4, P6 clusters and pillars; second research report in M10.
  7. **Month 12:** annual review — prune, consolidate, re-select pillars for year 2.
- **Dogfood:** the first three articles are written before the SEO.6 gate is enforced, to surface
  friction in the contract.
- **GA criteria:** AC-1…AC-9. Program is "healthy" when 3 consecutive months hit cadence with zero
  articles below 8.0.
- **Rollback:** the program can be paused; published content stays. Pausing MUST be a recorded
  decision with a restart date, because silent lapse is the failure mode we already experienced.

## 16. Test Plan

This plan is mostly process, but the artefacts are testable:

- **Unit** — brief front-matter validation; calendar parsing and status computation;
  `content:gaps` and `content:refresh-due` logic; priority-score computation (FR-7).
- **Integration** — every published article resolves to a brief (AC-5); every article's `pillar`
  exists; pillar pages link to every cluster member and vice versa (AC-2, AC-3).
- **End-to-end** — Playwright walks pillar → cluster → platform → CTA for each pillar; asserts the
  "Part of" module and breadcrumbs render server-side.
- **Security** — contributor content passes the MDX allowlist; no external scripts or embeds outside
  the approved set (SEO.14 governs video embeds).
- **Accessibility** — axe on each pillar page (long-document heading hierarchy is the common failure);
  screen-reader pass on the pillar TOC.
- **Performance / load** — pillar pages are the longest documents on the site; assert they meet the
  static budget and that LCP stays < 2.0 s with images.
- **Manual exploratory** — quarterly editorial review: read 5 random articles end-to-end and check
  they still describe the product and the world accurately.

## 17. Documentation & Training

- `docs/plan/seo/calendar.md` — the living calendar (FR-9).
- `docs/plan/seo/briefs/_TEMPLATE.md` — the brief template (FR-5).
- `www/docs/editorial-process.md` — brief → draft → review → publish → promote → refresh, with owners
  and SLAs at each step.
- `www/docs/contributor-guide.md` — for external SMEs: scope, tone, citation rules, consent, payment.
- Monthly program review deck template pulling from SEO.15.

## 18. Open Questions

1. Who is the named content lead, and is this their primary responsibility? (Highest delivery risk —
   AC-1 is unattainable without a clear answer.)
2. What is the budget for external contributors and for a keyword-data subscription?
3. Do we pursue guest contributions from customers, and does that require a customer-reference program?
4. Should P6 (homeschool) be a pillar now, or after the homeschool segment's revenue justifies it?
5. Newsletter: do we start one? It is the cheapest distribution channel for each new article and a
   direct input to SEO.13's mention program.

## 19. References

- Existing files: `www/src/blog/*.md`, `www/src/pages/blog-index.tsx`, `www/src/utils/blog.ts`
- Audit findings: [F-12](audit.md#f-12-publishing-stopped-76-days-ago),
  [F-13](audit.md#f-13-content-is-essay-shaped-not-passage-shaped)
- Research: [§7 Content strategy](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages),
  [§3](research.md#3-what-actually-earns-an-ai-citation)
- External: [Google — Creating helpful content](https://developers.google.com/search/docs/fundamentals/creating-helpful-content),
  [Google — Spam policies](https://developers.google.com/search/docs/essentials/spam-policies)
- Related plans: [SEO.6](SEO.6-answer-first-content-system.md),
  [SEO.9](SEO.9-comparison-alternatives-and-integration-pages.md),
  [SEO.12](SEO.12-original-research-and-data-program.md),
  [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md),
  [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md)
