# SEO.9 — Comparison, Alternatives & Integration Pages

> Implementation plan. Source: [docs/plan/seo/audit.md](audit.md) §S2 (F-15).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.9 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING (zero comparison, alternatives, or integration pages) |
| **Estimated effort** | L (1–2mo to launch, then 3 pages/month for 12 months) |
| **Owner (proposed)** | Marketing (content lead) |
| **Depends on** | SEO.1, SEO.3, SEO.5, SEO.6, SEO.7 |
| **Unblocks** | — |

---

## 1. Problem Statement

We have no `/compare/*`, no `/alternatives/*`, and no `/integrations/*` pages, while competitors own
those queries outright — D2L ranks for *Moodle alternatives*, Jotform and Teachfloor for *Canvas LMS
alternatives* (audit F-15). This is the one segment that survived the AI-era traffic decline:
bottom-of-funnel comparison and alternatives queries convert at **10–20%**, and the recommended
cadence is one comparison + one alternatives + one integration page per month for twelve months
([research §7](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)). It is
also the query class assistants answer most literally — "what are alternatives to Canvas for a small
district?" produces a list, and we are not on it. A **May 2026 breach affecting ~9,000 institutions
and 275M records** measurably raised interest in Canvas alternatives, and our shipped trust surfaces
(VPAT, security page, public privacy history, self-hosting) are a genuinely differentiated answer to
exactly that concern.

## 2. Goals

- Publish **36 bottom-of-funnel pages** over 12 months: 12 comparisons, 12 alternatives, 12
  integrations — the highest-converting page class we can build.
- Be **accurate and fair enough to be citable**. An assistant will not cite a comparison it can tell
  is marketing; a comparison that documents where the competitor wins is the one that gets quoted.
- Convert: every page ends in a segment-appropriate next step, and conversion is measured per page.
- Keep every competitor claim **sourced, dated, and re-verified** so we are never wrong in public.
- Turn the integration surface (LTI, SIS, Zapier, Make, webhooks, SSO) into discoverable landing pages
  that answer "does Lextures work with X?".

## 3. Non-Goals

- Attacking competitors. Pages state verifiable differences; disparagement is prohibited and is also
  a legal risk.
- Fabricated or scraped comparison matrices at scale. Each page is hand-verified; this is explicitly
  *not* programmatic (SEO.10 covers the programmatic-with-utility class).
- Review/rating schema on comparison pages (self-serving; excluded in SEO.3 non-goals).
- Competitor bidding or ad strategy.

## 4. Personas & User Stories

- **As a district administrator whose LMS contract is up for renewal**, I want an honest
  Lextures-vs-Canvas comparison including where Canvas is stronger, so that I can trust the parts
  that favour Lextures.
- **As a higher-ed evaluator after a vendor breach**, I want to know how Lextures handles security,
  data residency and self-hosting versus incumbents, so that I can brief my CISO.
- **As an IT lead**, I want a page that says exactly which LTI version, which SIS formats and which
  SSO protocols we support, so that I can rule us in or out in five minutes.
- **As someone asking an assistant "best Google Classroom alternatives for a small school"**, I want
  a source that actually compares them on the criteria that matter, so that the answer is useful.
- **As a Lextures sales engineer**, I want a URL I can send instead of writing the same comparison
  email, so that evaluations move faster.

## 5. Functional Requirements

**Page inventory**

- **FR-1.** Comparison pages MUST live at `/compare/lextures-vs-:competitor` and alternatives pages at
  `/alternatives/:competitor`, with index hubs at `/compare` and `/alternatives` (SEO.5 FR-9).
- **FR-2.** The 12-month comparison set MUST be, in priority order:

  | # | Competitor | Segment | Why this order |
  |---|---|---|---|
  | 1 | Canvas (Instructure) | HE · K12 | Largest incumbent; highest "alternatives" demand, amplified post-breach |
  | 2 | Google Classroom | K12 | Highest volume; clearest capability gap (workflow tool, not an LMS) |
  | 3 | Moodle | HE · K12 | Self-hosting overlap — our self-host story competes directly |
  | 4 | Schoology (PowerSchool) | K12 | District incumbent |
  | 5 | D2L Brightspace | HE · K12 | Direct adaptive/analytics competitor |
  | 6 | Blackboard (Anthology) | HE | Renewal-cycle displacement |
  | 7 | Teachable | HS · creator | Marketplace/creator overlap |
  | 8 | Thinkific | HS · creator | Marketplace/creator overlap |
  | 9 | Edmentum / Edgenuity | K12 | Adaptive courseware comparison |
  | 10 | Khan Academy | HS · K12 | Free-alternative objection handling |
  | 11 | Open edX | HE | Open-source/self-host comparison |
  | 12 | LearnWorlds | HS · creator | Creator marketplace |

- **FR-3.** Alternatives pages MUST be **list-shaped** ("8 alternatives to X in 2027"), evaluating
  5–8 real options against stated criteria, with Lextures presented on the same criteria as everyone
  else — not automatically first.
- **FR-4.** Integration pages MUST live at `/integrations/:slug` with a hub at `/integrations`. The
  12-month set: LTI 1.3 / LTI Advantage, Clever, ClassLink, PowerSchool SIS, Infinite Campus,
  Skyward, Google Workspace, Microsoft Entra ID (SSO), Canvas (import/migration), Zapier, Make,
  Webhooks & API. Each MUST state supported versions, exact capabilities, setup effort, and link to
  the corresponding help article (SEO.7).

**Content requirements**

- **FR-5.** Every comparison page MUST contain, in this order:
  1. `<AnswerBox>` — a 40–60 word direct answer to "should I choose Lextures or X?" that names the
     condition under which each wins.
  2. `<ComparisonTable>` — 12–20 criteria, sourced, with an explicit "as of <date>" note.
  3. **Where X is stronger** — a required section with at least two genuine items.
  4. **Where Lextures is stronger** — with evidence links to `/platform/*` and `/docs/*`.
  5. Migration path — what moving costs, what carries over, what does not.
  6. Pricing comparison — only where the competitor publishes pricing; otherwise state that it is not
     public rather than estimating.
  7. `<FAQ>` — 4–6 real questions.
  8. Publisher disclosure (FR-9).
- **FR-6.** Every competitor claim MUST cite the competitor's **own public documentation** with an
  access date (SEO.6 FR-21). Third-party review sites MAY be cited for satisfaction/adoption data but
  MUST NOT be the source for capability claims.
- **FR-7.** Claims MUST be **capability-level and falsifiable** ("supports LTI 1.3 Deep Linking, per
  their docs, accessed 2026-11-02"), never characterisations ("clunky", "outdated").
- **FR-8.** Alternatives pages MUST list the named competitor as an option with a fair description —
  the reader arrived searching for it — placed by the stated criteria, not by our preference.
- **FR-9.** Every page MUST display a visible publisher disclosure above the comparison table:
  *"This comparison is published by Lextures. We've documented where each product is stronger and
  cited every claim to the vendor's own documentation. Last verified <date>."*
- **FR-10.** Pages MUST NOT use the competitor's logo, wordmark, or trade dress; competitor names
  appear as plain text with a nominative-use footnote.

**Freshness & accuracy**

- **FR-11.** Every page MUST carry `verifiedAt`, and MUST be re-verified **quarterly**. A page more
  than 120 days unverified MUST surface in the SEO.16 lifecycle report and MUST display a visible
  "last verified" date so readers can judge staleness themselves.
- **FR-12.** A documented **correction process** MUST exist: a public contact for competitors to
  report inaccuracies, a 5-business-day correction SLA, and a visible changelog on each page.
- **FR-13.** Legal MUST review the first comparison page template before launch and any page naming a
  competitor's security incident.

**Conversion & schema**

- **FR-14.** Each page MUST end with a segment-appropriate CTA (K-12 → `/request-information`;
  HE → `/request-information`; HS/creator → `/get-started`), and MUST link to `/pricing`,
  the relevant segment hub, and ≥2 help articles.
- **FR-15.** Pages MUST emit `Article` + `BreadcrumbList` + `FAQPage` schema (SEO.3). They MUST NOT
  emit `Review`, `AggregateRating`, or `Product` schema about a competitor.
- **FR-16.** Integration pages MUST additionally emit `SoftwareApplication` with
  `isRelatedTo`/`supportingData` naming the integrated system, and `HowTo` for the setup steps.

## 6. Non-Functional Requirements

- **Performance** — static content pages under the SEO.4 static budget. Comparison tables must not
  cause layout shift; they scroll horizontally in their own container.
- **Security** — no third-party embeds; competitor documentation is cited by link, never proxied or
  mirrored.
- **Privacy & Compliance** — no customer named without written consent (S04); any reference to a
  publicly reported security incident must cite primary reporting and must be factual and dated.
  Trademark use is nominative only (FR-10).
- **Accessibility** — comparison tables are the accessibility risk: real `<table>` with `<caption>`,
  `<th scope="col|row">`, no colour-only "yes/no" indicators (use text + icon), and a text summary
  above the table conveying the headline conclusion for screen-reader users.
- **Scalability** — 36 pages is hand-verified volume; the template makes each cheaper without making
  any of them automatic.
- **Reliability** — the correction process (FR-12) is the reliability mechanism for a page class where
  being wrong in public is the main risk.
- **Observability** — per-page: impressions, position, clicks, scroll depth to the table, CTA clicks,
  assisted conversions, AI citations. Comparison pages are the highest-intent pages we will have, so
  conversion tracking is required, not optional.
- **Maintainability** — one MDX file per page plus a shared `criteria.ts` defining the comparison
  dimensions so tables stay consistent across pages.
- **Internationalization** — English only; competitor availability differs by region, which must be
  noted rather than assumed.
- **Backward compatibility** — new URLs; nothing to migrate.

## 7. Acceptance Criteria

- **AC-1.** *Given* 12 months, *When* counted, *Then* 12 comparison, 12 alternatives, and 12
  integration pages are published, at a rate of ≥3/month for 12 consecutive months.
- **AC-2.** *Given* any comparison page, *When* reviewed, *Then* it contains a "Where X is stronger"
  section with ≥2 substantive items, and every capability claim links to competitor documentation
  with an access date.
- **AC-3.** *Given* any comparison page, *When* rendered, *Then* the publisher disclosure is visible
  above the fold of the comparison table and the `verifiedAt` date is displayed.
- **AC-4.** *Given* a page unverified for >120 days, *When* the lifecycle report runs, *Then* it is
  flagged, and the page displays its stale verification date to readers.
- **AC-5.** *Given* a comparison table, *When* tested with a screen reader, *Then* every cell's
  meaning is announced with its row and column headers, and no cell conveys meaning by colour alone.
- **AC-6.** *Given* any comparison or alternatives page, *When* its schema is validated, *Then* it
  emits `Article`, `BreadcrumbList` and `FAQPage`, and emits no `Review` or `AggregateRating`.
- **AC-7.** *Given* an integration page, *When* read, *Then* it states the exact supported versions
  and protocols and links to a working help article for setup.
- **AC-8.** *Given* 6 months post-launch, *When* measured, *Then* comparison + alternatives pages
  account for ≥25% of organic-sourced MQLs, and ≥6 of them appear in AI citations.
- **AC-9.** *Given* a correction request from a competitor, *When* received, *Then* it is
  acknowledged within 2 business days and resolved or answered within 5, with the changelog updated.

## 8. Data Model

No database changes.

```
www/src/compare/lextures-vs-canvas.mdx
www/src/alternatives/canvas.mdx
www/src/integrations/lti-1-3.mdx
www/src/lib/comparison-criteria.ts     # shared dimension definitions + display order
www/src/lib/competitors.ts             # name, docsUrl, segments, pricingPublic: bool, lastVerified
```

Front-matter additions (on top of SEO.6):

```yaml
competitor: canvas
competitorDisplayName: "Canvas (Instructure)"
verifiedAt: 2026-11-02
sourceUrls:                            # competitor docs cited, with access dates
  - url: https://…/lti
    accessed: 2026-11-02
changelog:
  - date: 2027-02-01
    note: "Updated LTI Advantage support after Canvas 2027.1 release notes."
```

Comparison criteria (`comparison-criteria.ts`) — the fixed dimension set so every page is comparable:
adaptive delivery, item response theory, question bank, quiz types, spaced review, rubrics, peer
review, gradebook features, standards/outcomes, SIS roster sync, LTI version, SSO protocols,
accessibility conformance (VPAT), self-hosting, data export, privacy posture, mobile apps, parent
access, marketplace/monetisation, pricing transparency, API/webhooks.

## 9. API Surface

None. Pricing figures on our side come from `www/src/lib/institution-pricing.ts` (SEO.3 FR-14) so our
own numbers cannot drift.

## 10. UI / UX

- **New pages:** `/compare` hub, 12 comparison pages, `/alternatives` hub, 12 alternatives pages,
  `/integrations` hub, 12 integration pages.
- **New components:** `<ComparisonMatrix>` (extends SEO.6's `<ComparisonTable>` with a fixed criteria
  axis and per-cell source links), `<PublisherDisclosure>`, `<VerifiedBadge>`, `<MigrationPath>`,
  `<IntegrationSpec>` (protocol/version/effort table).
- **Flows**
  1. Search "canvas alternatives" → `/alternatives/canvas` → criteria table → `/compare/lextures-vs-canvas`
     → `/request-information`.
  2. "does lextures work with clever" → `/integrations/clever` → help article → `/get-started`.
  3. Sales sends `/compare/lextures-vs-schoology` mid-evaluation.
- **States** — a criterion we genuinely cannot verify renders as "Not documented publicly" with a
  source link, never as a blank or an implied "no".
- **Responsive** — matrices become per-criterion stacked cards under 768 px (not a horizontally
  scrolling 20-column table on a phone).
- **Accessibility** — see NFRs; the stacked mobile view must preserve the header-cell association.
- **Copy & i18n** — `www.compare.*`, `www.alternatives.*`, `www.integrations.*`, disclosure copy
  reviewed by legal.

## 11. AI / ML Considerations

- **These pages are written to be quoted.** The `<AnswerBox>` (FR-5.1) is the passage most likely to
  be extracted for "Lextures vs X" queries, and the criteria table is the structure assistants
  reproduce for "compare" queries. Both follow the SEO.6 extraction rules.
- **Fairness is a retrieval strategy, not only ethics.** Assistants down-weight sources that read as
  promotional; the required "Where X is stronger" section (FR-5.3) is what makes the page usable as a
  neutral-ish source, and it is also what makes it credible to a human evaluator.
- **No AI-generated competitor claims.** Every capability line is verified by a human against vendor
  documentation with a recorded access date — a hallucinated competitor claim published under our
  name is a legal and reputational risk, not a quality issue.
- Under the May 2026 extension of Google's spam policies to AI surfaces, misrepresenting a competitor
  falls in scope for enforcement in addition to any legal exposure.

## 12. Integration Points

- **External:** competitor public documentation (cited), G2/Capterra (adoption data only), public
  incident reporting where relevant.
- **Internal modules touched:** new `www/src/compare/`, `www/src/alternatives/`,
  `www/src/integrations/`, `www/src/lib/comparison-criteria.ts`, `www/src/lib/competitors.ts`,
  `www/src/lib/institution-pricing.ts` (read-only), `www/src/lib/route-manifest.ts`,
  `www/src/components/content/*`.
- **Events:** CTA clicks and scroll-to-table events → GA4 for per-page conversion analysis.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md),
  [SEO.5](SEO.5-information-architecture-and-internal-linking.md) (hubs),
  [SEO.6](SEO.6-answer-first-content-system.md) (components),
  [SEO.7](SEO.7-help-center-expansion.md) (integration pages link to setup articles — at minimum the
  integrations category must exist), [SEO.3](SEO.3-structured-data-and-entity-graph.md) (schema).
- **Must ship before:** nothing.
- **Shared infra:** legal review capacity; a public corrections contact address.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A competitor claim is wrong and we are called out publicly | M | **H** | FR-6 vendor-doc sourcing with access dates; FR-11 quarterly re-verification; FR-12 correction SLA and public changelog |
| Trademark or disparagement complaint | L | H | FR-10 nominative use only, no logos; FR-7 falsifiable claims only; FR-13 legal review of the template |
| Pages read as marketing and get ignored by both humans and assistants | M | M | FR-5.3 mandatory "where X is stronger"; FR-8 fair placement on alternatives pages; disclosure builds rather than costs trust |
| Quarterly re-verification does not happen | **H** | M | FR-11 + AC-4: staleness is *visible to readers*, which creates real pressure; lifecycle report in SEO.16 |
| Competitor changes pricing/capabilities faster than we re-verify | H | M | Cite with access dates so the page is honest about its own currency; prioritise re-verification by traffic |
| Cannibalisation between `/compare/x` and `/alternatives/x` | M | L | Distinct intents and distinct page shapes (head-to-head vs. list); cross-linked, each canonical to itself |
| Sales wants claims stronger than evidence supports | M | M | Claims policy is FR-7; content lead owns the final wording |

## 15. Rollout Plan

- **Feature flag:** none.
- **Sequencing**
  1. Build `comparison-criteria.ts` + the page template + components; legal review (FR-13).
  2. Publish the pilot: `/compare/lextures-vs-canvas` and `/alternatives/canvas`. Measure for 4 weeks.
  3. Publish `/integrations/lti-1-3`, `/integrations/clever`, `/integrations/classlink` — highest
     technical-qualification value.
  4. Steady state: 1 comparison + 1 alternatives + 1 integration per month for 12 months.
  5. Quarter 2: first re-verification sweep; publish changelogs.
  6. Quarter 3: review conversion data and re-prioritise the remaining competitor list.
- **Dogfood:** sales engineers use the pilot pages in three live evaluations and report gaps before
  page 3 is written.
- **GA criteria:** AC-1…AC-9; the pilot pages convert at ≥5% to a CTA click within 8 weeks.
- **Rollback:** individual pages can be unpublished (410 + removal from sitemap) if a claim cannot be
  substantiated — the correction path is preferred.

## 16. Test Plan

- **Unit** — criteria-table rendering; source-link + access-date validation (fails if a capability
  claim has no source); `verifiedAt` staleness computation; competitor registry validation.
- **Integration** — build asserts every comparison page has the required sections (FR-5) and a
  publisher disclosure; asserts no `Review`/`AggregateRating` schema is emitted (AC-6).
- **End-to-end (Playwright)** — matrix renders server-side with JS disabled; mobile stacked view
  preserves header associations; CTA and scroll events fire.
- **Security** — no external embeds; assert competitor doc links are `rel="noopener"` and use HTTPS.
- **Accessibility** — axe plus manual NVDA/VoiceOver on the matrix in both desktop table and mobile
  stacked forms (AC-5); verify "yes/no" cells are not colour-only; 320 px and 200% zoom.
- **Performance / load** — matrices are the heaviest markup on the site; assert the static budget and
  zero CLS from the table.
- **Manual exploratory** — quarterly: re-open every cited competitor doc link, confirm it resolves and
  still supports the claim; record the new access date.

## 17. Documentation & Training

- `www/docs/comparison-page-policy.md` — sourcing rules, prohibited language, nominative trademark
  use, the correction process, and the re-verification cadence.
- `www/docs/competitor-research.md` — how to verify a capability claim and record it.
- Sales enablement: one-pager on which comparison URL to send for which objection.
- Public: a "Report an inaccuracy" link on every comparison page, routed to a monitored address.

## 18. Open Questions

1. Who signs off on competitor claims — content lead, legal, or both? What is the turnaround?
2. Do we name the May 2026 breach on the Canvas comparison? (Factual and relevant, but it is the
   single highest-risk sentence in the whole plan set — requires legal sign-off.)
3. Do we publish a pricing comparison where the competitor's pricing is quote-only, or state
   "not public"? (Recommendation: state "not public"; never estimate.)
4. Should alternatives pages be dated in the title ("in 2027") for freshness signalling, accepting the
   annual retitle cost?
5. Which integrations are genuinely shipped and supported today versus roadmap? FR-4's list must be
   verified against the codebase before any page is written.

## 19. References

- Existing files: `www/src/lib/institution-pricing.ts`, `www/src/docs/connecting-lextures-to-zapier.md`,
  `www/src/docs/using-lextures-with-make.md`, `www/src/docs/self-hosting.md`,
  `www/src/pages/vpat-page.tsx`, `www/src/pages/security-page.tsx`
- Audit findings: [F-15](audit.md#f-15-zero-bottom-of-funnel-pages)
- Research: [§7](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages),
  [§10 Category context](research.md#10-category-context-the-lmsedtech-search-landscape)
- External: [GenesysGrowth — designing competitive comparison pages](https://genesysgrowth.com/blog/designing-competitive-comparison-pages),
  [Optiseon — B2B SaaS SEO playbook 2026](https://optiseon.com/blog/b2b-saas-seo-playbook-2026/),
  [1EdTech — LTI Advantage](https://www.1edtech.org/standards/lti)
- Related plans: [SEO.5](SEO.5-information-architecture-and-internal-linking.md),
  [SEO.6](SEO.6-answer-first-content-system.md), [SEO.7](SEO.7-help-center-expansion.md),
  [SEO.16](SEO.16-seo-governance-and-ci-guardrails.md),
  [16 — Integrations & extensibility (completed)](../../completed/16-integrations-extensibility/)
