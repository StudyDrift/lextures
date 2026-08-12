# SEO.10 — Programmatic Utility Pages

> Implementation plan. Source: [docs/plan/seo/audit.md](audit.md) §S2 (F-15) and
> [research §7](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.10 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING (no glossary, no templates, no standards browser; one calculator at `/pricing/calculator` that 404s to crawlers) |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Marketing + Web platform |
| **Depends on** | SEO.1, SEO.3, SEO.5, SEO.6 |
| **Unblocks** | — |

---

## 1. Problem Statement

We have no glossary, no template library, no standards browser, and our one genuine utility page —
the pricing calculator — is among the 28 URLs that return HTTP 404 to crawlers (audit F-1, F-15).
Programmatic SEO survived into 2026, but only in a specific form: the quality floor now requires that
**every page let the user do something** — "if a user can't perform an action or solve a specific
micro-problem, it's just digital litter"
([research §7](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)). The
March 2026 core update named scaled content abuse explicitly and cost offenders 50–80% of traffic. So
the opportunity is real and the failure mode is severe: we build ~350 pages that each solve a genuine
micro-problem, or we build none.

## 2. Goals

- Ship four utility page families that each pass a hard **utility test**: glossary (~200 terms),
  standards browser (~120 pages), template library (~24), and calculators/tools (~6).
- Capture the long tail of definitional and "how do I do X" queries that assistants answer literally —
  and be the source they answer from.
- Encode a **quality floor in code**: a page that fails the utility test cannot be indexed.
- Feed the rest of the site — glossary terms link into pillars, standards pages link into `/k12`,
  templates link into help articles.

## 3. Non-Goals

- Mass-generating location, "best X for Y", or vendor-directory pages. No page family ships without a
  data source we own or license and a genuine user action.
- Content that duplicates the help center (SEO.7) or the guides (SEO.8).
- A public API for these datasets (possible later; out of scope).
- Auto-translating any of it (SEO.17).

## 4. Personas & User Stories

- **As a new teacher**, I want a clear, sourced definition of "formative assessment" with an example
  and a way to try it, so that I understand it and can act on it.
- **As a K-12 curriculum lead**, I want to browse a standard and see what assessing it well looks
  like, so that I can plan a unit.
- **As a department chair**, I want a rubric template I can download and adapt, so that I do not start
  from a blank page.
- **As a district business manager**, I want to model our cost for 3,400 students across two schools,
  so that I can put a number in the budget.
- **As an AI assistant asked "what is item response theory?"**, I want a precise, cited definition
  page, so that I can answer accurately and attribute it.

## 5. Functional Requirements

**The utility floor (applies to every page in this plan)**

- **FR-1.** Every programmatic page MUST pass **all four** utility tests, checked at build time:
  1. **Action** — the page offers something to do: a calculation, a download, a copyable artefact, an
     interactive filter, or a decision the reader can make from the page alone.
  2. **Unique substance** — ≥150 words of page-specific prose not shared with any sibling page,
     verified by an n-gram similarity check (< 60% similarity to any sibling).
  3. **Sourced** — at least one primary-source citation (standards body, research, our own
     documentation).
  4. **Connected** — ≥3 inbound internal links and ≥3 outbound (SEO.5).
- **FR-2.** A page failing any test MUST be emitted with `noindex,follow` and excluded from sitemaps
  and `llms.txt`, and MUST appear in the SEO.16 quality report. Failing pages are never silently
  indexed.
- **FR-3.** A family MUST NOT launch until **≥80%** of its pages pass. Partial families ship the
  passing subset.

**Glossary**

- **FR-4.** `/glossary` MUST index ~200 terms across assessment, adaptive learning, standards,
  accessibility, and edtech operations, each at `/glossary/:term`.
- **FR-5.** Each term page MUST contain: a 40–60 word `<AnswerBox>` definition; "in one sentence";
  a worked example; "how it works in Lextures" with a link to the relevant `/platform/*` or `/docs/*`
  page; related terms (3–6); and sources. This satisfies FR-1 through *decision utility* — the reader
  leaves knowing whether the concept applies to them — plus a copyable definition block.
- **FR-6.** Definitions MUST be authored once via `<Definition>` in source content (SEO.6 FR-10) or in
  the glossary source file, never duplicated — the glossary harvests `dist/.definitions.json`.
- **FR-7.** Term pages MUST emit `DefinedTerm` within a `DefinedTermSet` for the glossary, plus
  `BreadcrumbList` (SEO.3).
- **FR-8.** Terms MUST be reviewed by a subject-matter expert; a term page without `reviewedBy` fails
  the build (these are the pages most likely to be quoted verbatim by an assistant).

**Standards browser**

- **FR-9.** `/standards` MUST let a user browse published academic standards frameworks
  (Common Core, NGSS, state frameworks where openly licensed) at
  `/standards/:framework/:grade/:code`, with hub pages per framework and per grade.
- **FR-10.** Each standard page MUST show: the standard's official text (only where the licence
  permits redistribution — otherwise a summary plus a link to the official source), what mastery of it
  looks like, 2–3 assessment approaches, common misconceptions, and a link to relevant marketplace
  courses (SEO.11) and to `/platform/…` outcomes features.
- **FR-11.** Licence compliance MUST be verified per framework **before** any page is generated, with
  the licence and attribution recorded in `standards-sources.ts`. A framework without a clear
  redistribution licence is summarised and linked, never reproduced.
- **FR-12.** Standard pages MUST NOT be generated for frameworks where we cannot add the
  mastery/assessment/misconception content — that content is the utility, and without it the page is a
  scraped copy of someone else's data.
- **FR-13.** The browser MUST offer real filtering (subject, grade, framework) that works server-side
  via distinct URLs, not only client-side.

**Templates**

- **FR-14.** `/templates` MUST offer ~24 downloadable, immediately usable artefacts: analytic and
  holistic rubrics, syllabus templates per segment, course-design checklists, accessibility review
  checklists, assessment blueprints, parent-communication templates, standards-mapping worksheets.
- **FR-15.** Each template page MUST offer download in ≥2 formats (PDF + editable DOCX/Google Docs
  copy link), a preview rendered in-page, guidance on how to adapt it, and a one-click "open in
  Lextures" path where the artefact maps to a product object (rubric, course blueprint).
- **FR-16.** Downloads MUST NOT be gated behind a form. (Gating trades a large SEO/AI-citation loss for
  a small lead gain; the CTA is contextual instead.)

**Calculators & tools**

- **FR-17.** Six tools, each on its own indexable URL with prerendered explanatory content:
  1. `/pricing/calculator` — existing, made crawlable and linkable with shareable URL state.
  2. `/tools/grade-calculator` — weighted grade / what-if calculator.
  3. `/tools/rubric-builder` — build and export a rubric (no account required).
  4. `/tools/reading-level` — text complexity analysis for assignment prompts.
  5. `/tools/assessment-blueprint` — map items to objectives and Bloom levels, export.
  6. `/tools/accessibility-checker` — check a course-page snippet against WCAG 2.2 AA basics.
- **FR-18.** Every tool MUST work **without an account** and MUST function server-rendered-then-hydrate:
  the explanatory content and instructions are in the HTML; only the interaction requires JS.
- **FR-19.** Tool state MUST be URL-encodable so a result can be shared and linked — this is what
  turns a tool into a linkable asset rather than a dead-end widget.
- **FR-20.** Tools MUST process input **client-side only** where the input could be sensitive (reading
  level, accessibility checker, rubric text). No user-entered text may be sent to our servers or to a
  third party without an explicit action and notice.

## 6. Non-Functional Requirements

- **Performance** — glossary, standards and template pages are static (`interactive: false`, SEO.4
  FR-4). Tools are interactive-budget pages (≤150 KB JS) and MUST lazy-load their logic after the
  explanatory content paints.
- **Security** — FR-20 client-side-only processing; no eval of user input; downloads are static files
  with correct `Content-Type` and `Content-Disposition`; template files are generated in CI, not
  uploaded ad hoc.
- **Privacy & Compliance** — FR-20 means no PII transits our servers from tools. Standards content
  licensing per FR-11 is a copyright obligation. Templates we publish must be our own work or
  appropriately licensed, with attribution.
- **Accessibility** — tools are the highest-risk surface: every input labelled, errors associated via
  `aria-describedby`, results announced in a live region, full keyboard operation, no colour-only
  status, and results readable at 200% zoom. WCAG 2.2 AA, consistent with UX.5/UX.6. The accessibility
  checker tool itself must be accessible — a failure here is a credibility event.
- **Scalability** — ~350 pages generated; build must add < 90 s. Similarity checking must be
  incremental.
- **Reliability** — generation is deterministic from committed data files; a data-source fetch failure
  reuses the last good snapshot.
- **Observability** — per-family: pages generated, pages failing the utility test (and which test),
  index coverage, organic entrances, tool completion rate, download counts.
- **Maintainability** — one generator per family under `www/scripts/generate/`, sharing the utility
  checker. Data lives in versioned files, not scraped at build time.
- **Internationalization** — English only; the standards browser is US-framework-only at launch, which
  MUST be stated on the page rather than implied.
- **Backward compatibility** — `/pricing/calculator` keeps its URL.

## 7. Acceptance Criteria

- **AC-1.** *Given* the build, *When* the utility checker runs, *Then* every generated page reports
  pass/fail per test, and every failing page is emitted `noindex` and excluded from sitemaps and
  `llms.txt`.
- **AC-2.** *Given* two sibling glossary pages, *When* their prose is compared, *Then* similarity is
  < 60% and each has ≥150 unique words.
- **AC-3.** *Given* a glossary term page, *When* rendered, *Then* it contains a 40–60 word definition,
  a worked example, a "in Lextures" link, ≥3 related terms, ≥1 source, and a `reviewedBy` byline; a
  page missing `reviewedBy` fails the build.
- **AC-4.** *Given* a standards framework without a verified redistribution licence, *When* generation
  runs, *Then* no page reproduces its text, and the summary page cites and links the official source.
- **AC-5.** *Given* any tool, *When* JavaScript is disabled, *Then* the page still explains what the
  tool does, how to use it, and why it matters, and is indexable.
- **AC-6.** *Given* a tool result, *When* I copy the URL and open it in a new browser, *Then* the same
  inputs and result are restored.
- **AC-7.** *Given* the reading-level and accessibility tools, *When* I submit text and inspect network
  traffic, *Then* zero requests carry that text off the client.
- **AC-8.** *Given* a template page, *When* I click download, *Then* the file downloads without a form,
  in both offered formats, and the in-page preview matched the file.
- **AC-9.** *Given* every tool, *When* audited with axe and a screen reader, *Then* zero violations and
  results are announced in a live region.
- **AC-10.** *Given* 6 months post-launch, *When* measured, *Then* ≥60% of generated pages are indexed
  and the family produces ≥1,500 organic sessions/month, with < 5% of pages in `noindex` state.

## 8. Data Model

No database changes.

```
www/src/data/
  glossary/<term>.mdx            # definition, example, sources, reviewedBy
  standards/<framework>.json     # committed, licence-verified, with attribution
  standards-sources.ts           # framework → licence, attribution, redistribution: bool
  templates/<slug>/              # source + generated pdf/docx + preview
www/src/tools/<tool>/            # component + explanatory MDX
www/scripts/generate/
  glossary.mjs  standards.mjs  templates.mjs  utility-check.mjs
```

Utility report artefact `dist/.utility-report.json`:

```jsonc
{ "family": "glossary", "path": "/glossary/item-response-theory",
  "tests": { "action": true, "unique": true, "sourced": true, "connected": false },
  "uniqueWords": 210, "maxSiblingSimilarity": 0.41, "inboundLinks": 2, "indexed": false }
```

## 9. API Surface

- No new server routes. The pricing calculator continues to compute from
  `www/src/lib/institution-pricing.ts` client-side.
- Standards data is committed, not fetched at build time (avoids a build-time dependency on a
  third-party endpoint and makes licence review a reviewable event).
- Marketplace course links on standard pages use the existing public marketplace API data already
  fetched by the SEO.1 generator.

## 10. UI / UX

- **New pages:** `/glossary` + ~200 terms; `/standards` + framework/grade/code pages (~120);
  `/templates` + ~24; `/tools` + 5 new tools.
- **New components:** `<GlossaryTerm>`, `<StandardCard>`, `<TemplatePreview>`, `<ToolShell>` (title,
  explanation, tool region, results region, share-URL control), `<CopyBlock>`.
- **Flows**
  1. Search "what is item response theory" → `/glossary/item-response-theory` → "how it works in
     Lextures" → `/platform/assessment` → `/get-started`.
  2. `/standards/ngss/ms-ps1-1` → assessment approaches → marketplace course → enrol.
  3. `/tools/rubric-builder` → build → export → "save this rubric in Lextures" → sign-up.
- **States** — tools: empty (worked example pre-filled), invalid input (inline error, associated), busy
  (spinner + live region), result (announced), error. Glossary: no "stub" pages — a term without full
  content is not published.
- **Responsive** — tools are usable at 320 px; standards filters become a bottom sheet on mobile;
  template previews scroll in their own container.
- **Accessibility** — the highest-risk area in this plan; see NFRs and AC-9. The accessibility checker
  must ship with its own conformance statement.
- **Copy & i18n** — `www.glossary.*`, `www.standards.*`, `www.templates.*`, `www.tools.*`.

## 11. AI / ML Considerations

- **Glossary pages are the archetypal AI-cited page**: a precise, sourced, self-contained definition
  is exactly the 40–60-word passage assistants extract. FR-5's structure is designed for that, and
  FR-8's expert review exists because a wrong definition quoted at scale is worse than no page.
- **No LLM generates these pages.** Definitions are human-authored and expert-reviewed; standards
  content is licensed data plus human-authored guidance. The reason is the same as SEO.6's: volume
  without oversight is the exact pattern that lost sites 50–80% of traffic in March 2026.
- The tools MAY use on-device heuristics (readability formulas, WCAG rule checks). If any tool later
  calls a model, it must (a) disclose it, (b) keep FR-20's no-transit guarantee or replace it with
  explicit consent, and (c) get a DPIA ([S06](../standards/S06-dpia-pia-algorithmic-impact.md)).

## 12. Integration Points

- **External:** standards frameworks (Common Core / NGSS / state, licence-checked), PDF and DOCX
  generation libraries (build-time only).
- **Internal modules touched:** `www/src/pages/pricing-calculator-page.tsx` (URL state, prerendered
  explanation), `www/src/lib/institution-pricing.ts`, `www/src/lib/route-manifest.ts`,
  `www/scripts/generate/*`, `www/src/components/content/*`, `dist/.definitions.json` from SEO.6.
- **Events:** tool completion, share-URL copy, template download → GA4.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md),
  [SEO.3](SEO.3-structured-data-and-entity-graph.md) (`DefinedTerm`),
  [SEO.5](SEO.5-information-architecture-and-internal-linking.md) (hubs, link requirements),
  [SEO.6](SEO.6-answer-first-content-system.md) (`<Definition>` harvest, content contract).
- **Must ship before:** nothing; standards pages link to [SEO.11](SEO.11-marketplace-catalog-seo.md)
  course pages, so sequencing them after SEO.11 improves the links but is not required.
- **Shared infra:** legal review for standards licensing; design time for tool UI.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Pages judged scaled content abuse | M | **H** | FR-1 four-test floor enforced in code, FR-2 `noindex` on failure, FR-3 80% family threshold; every family has genuine action utility |
| Standards text reproduced without licence | M | **H** | FR-11 per-framework licence verification recorded in code; FR-12 no page without our own added content; legal review before generation |
| Tools ship inaccessible and undermine our accessibility positioning | M | **H** | AC-9 + manual screen-reader testing per tool; the accessibility checker gets an independent review |
| 200 glossary terms is more expert review than we have | H | M | Launch with 60 high-value terms that pass fully; grow at 20/month; FR-3 ships the passing subset |
| A wrong definition gets quoted widely by assistants | M | H | FR-8 expert review + sources; correction process shared with SEO.9's; quarterly re-verification |
| Tools become a maintenance burden | M | M | Six tools, each small and client-side; no server dependency; usage reviewed annually and low performers retired |
| Similarity check produces false positives and blocks legitimate pages | M | L | Threshold tuned on the first family; per-page override with a recorded reason |

## 15. Rollout Plan

- **Feature flag:** none; families ship independently.
- **Sequencing**
  1. Build the utility checker + report first — the floor exists before any page does.
  2. Make `/pricing/calculator` crawlable with URL state and prerendered explanation (smallest change,
     immediate value).
  3. Glossary wave 1: 60 fully-reviewed terms.
  4. Templates: 12 templates.
  5. Tools: rubric builder, grade calculator (highest link-attraction potential).
  6. Standards browser: one framework end-to-end after licence review; expand only if it passes.
  7. Glossary waves 2–3 (+20/month), templates to 24, remaining tools.
- **Dogfood:** the rubric builder and blueprint tool are used by our own docs team to produce SEO.7
  screenshots and templates.
- **GA criteria:** AC-1…AC-10; < 5% of published pages in `noindex`; no manual action in GSC.
- **Rollback:** a family can be set to `noindex` wholesale with one flag if quality signals degrade;
  pages stay for users while the issue is fixed.

## 16. Test Plan

- **Unit** — each utility test (action/unique/sourced/connected); n-gram similarity; glossary
  front-matter incl. `reviewedBy`; standards licence gating; URL state encode/decode for tools;
  pricing calculation parity with `institution-pricing.ts`.
- **Integration** — generation produces the expected page counts; failing pages are `noindex` and
  absent from sitemap + `llms.txt` (AC-1); `.definitions.json` harvest round-trips (SEO.6 AC-8).
- **End-to-end (Playwright)** — JS-disabled tool pages remain informative and indexable (AC-5);
  URL-state restore (AC-6); template download in both formats (AC-8); network assertion that tool
  input never leaves the client (AC-7).
- **Security** — no user input reaches a server or third party; downloads served with safe headers;
  no `dangerouslySetInnerHTML` on tool input; fuzz the reading-level and rubric inputs.
- **Accessibility** — axe on every tool and a sample of 20 generated pages; NVDA + VoiceOver on each
  tool's full flow incl. error and result announcement; 200% zoom and 320 px.
- **Performance / load** — static families meet the static budget; tools meet the interactive budget
  with logic lazy-loaded; generation adds < 90 s to the build.
- **Manual exploratory** — SME spot-check of 20 glossary definitions and 10 standards pages; a teacher
  uses the rubric builder unassisted.

## 17. Documentation & Training

- `www/docs/utility-page-policy.md` — the four tests, why they exist (with the research citation), how
  to request an override, and what happens to a failing page.
- `www/docs/standards-licensing.md` — per-framework licence status, attribution requirements, and the
  review process before adding a framework.
- `www/docs/glossary-authoring.md` — term selection, definition style, review workflow.
- Runbook: retiring a tool; correcting a definition after publication.

## 18. Open Questions

1. Which standards frameworks have licences that permit redistribution, and who does that legal
   review? (Blocks FR-9 entirely.)
2. Do we have SME capacity to review 200 glossary terms, or should launch be 60 with growth?
   (Recommendation: 60 at launch.)
3. Should `/tools/*` require an account for export? (Recommendation: no — ungated is the whole
   strategy; convert on "save to Lextures".)
4. Does the rubric builder's export map cleanly onto the product's rubric model, so "open in Lextures"
   is real rather than aspirational?
5. Should the standards browser link to marketplace courses before SEO.11 ships subject facets?

## 19. References

- Existing files: `www/src/pages/pricing-calculator-page.tsx`, `www/src/lib/institution-pricing.ts`,
  `www/src/lib/vpat-data.ts`, `www/src/lib/conformance-ui.tsx`
- Audit findings: [F-15](audit.md#f-15-zero-bottom-of-funnel-pages), [F-1](audit.md#f-1-28-of-31-sitemap-urls-return-http-404)
- Research: [§7](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)
- External: [Google — Spam policies: scaled content abuse](https://developers.google.com/search/docs/essentials/spam-policies#scaled-content),
  [schema.org/DefinedTerm](https://schema.org/DefinedTerm),
  [W3C — WCAG 2.2](https://www.w3.org/TR/WCAG22/)
- Related plans: [SEO.6](SEO.6-answer-first-content-system.md),
  [SEO.11](SEO.11-marketplace-catalog-seo.md),
  [SEO.16](SEO.16-seo-governance-and-ci-guardrails.md),
  [S06 — DPIA](../standards/S06-dpia-pia-algorithmic-impact.md)
