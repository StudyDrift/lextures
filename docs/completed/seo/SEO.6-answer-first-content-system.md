# SEO.6 — Answer-First Content System & Extractability Primitives

> Completed 2026-08-11. Implemented with strict, allowlisted Markdown directives (the safe directive option permitted by FR-11) and deterministic build-time quality reporting.

> Implementation plan. Source: [docs/plan/seo/audit.md](audit.md) §S2 (F-13).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.6 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING (essay-shaped content, no TL;DR blocks, no direct-answer paragraphs, outbound citations in 1 of 5 posts) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Docs / Content |
| **Depends on** | SEO.1, SEO.5 |
| **Unblocks** | SEO.7, SEO.8, SEO.9, SEO.10, SEO.12 |

---

## 1. Problem Statement

Our existing posts are well-argued essays with narrative section headers ("The Traditional Ascent")
rather than the questions searchers and assistants actually ask, no TL;DR or direct-answer blocks, no
definition boxes or comparison tables, no "last reviewed" dates, and outbound citations in exactly
one of five posts (audit F-13). That format loses on every measured 2026 citation factor at once:
content that fully answers a query in **self-contained 134–167-word units** is **4.2× more likely to
be cited**, and authoritative outbound citations produce the single largest visibility gain measured
(**+132%**) ([research §3](research.md#3-what-actually-earns-an-ai-citation)). Before we publish 200
more pages, we need the format, the components, and the checks that make every one of them
extractable — otherwise we scale the wrong shape.

## 2. Goals

- Define one **content contract** every editorial page satisfies: answer first, self-contained
  passages, explicit questions as headings, cited claims, dated review.
- Ship the **MDX component set** that makes the contract easy to follow (`<KeyTakeaways>`,
  `<AnswerBox>`, `<Definition>`, `<ComparisonTable>`, `<Steps>`, `<FAQ>`, `<Callout>`, `<Stat>`,
  `<Sources>`).
- Ship an **extractability score** computed at build time, reported per page, with a minimum
  threshold enforced for new content.
- Retrofit the five existing posts and six help articles to the contract without diluting their
  argument.
- Make citations structural: a claim that carries a number must carry a source, in the prose and in
  `Article.citation[]` schema.

## 3. Non-Goals

- Producing content (SEO.7, SEO.8, SEO.9, SEO.10, SEO.12 do that). This plan is the system they use.
- Keyword research and topic selection — SEO.8.
- AI-generated content pipelines. Nothing here generates prose; the checks are deterministic and
  content is human-written and human-reviewed (see §11 for why that is a deliberate constraint).
- Rich-result chasing: `FAQPage` markup is emitted for machine comprehension only (SEO.3 FR-13).

## 4. Personas & User Stories

- **As a teacher asking an assistant "what's the difference between formative and summative
  assessment?"**, I want the model to find a Lextures passage that answers it completely in one
  paragraph, so that we get cited.
- **As a skim-reading administrator**, I want the takeaways at the top, so that I get the answer in
  15 seconds and read on only if I need depth.
- **As a writer**, I want components and a checklist rather than a style essay, so that following the
  contract is the path of least resistance.
- **As an editor**, I want a build-time score and a diff of what failed, so that review is about
  substance rather than format policing.
- **As a reader evaluating a claim**, I want a visible source next to every statistic, so that I can
  verify it.

## 5. Functional Requirements

**The content contract**

- **FR-1.** Every editorial page MUST open with a `<KeyTakeaways>` block: 3–5 bullets, each a complete
  sentence stating a conclusion (not a topic), placed above the first `<h2>` and after the byline.
- **FR-2.** Every page MUST contain at least one `<AnswerBox>`: a **40–60 word** direct answer to the
  page's primary question, phrased to stand alone with no anaphora ("this", "it", "the above") and no
  dependence on the surrounding page.
- **FR-3.** `<h2>`/`<h3>` headings for informational content MUST be phrased as the question a person
  would ask, or as a noun phrase that names the answer — never as a rhetorical or narrative label.
  A lint rule MUST flag headings that begin with "The " and contain no question word or entity.
- **FR-4.** Body sections MUST be composed of **self-contained passages of 120–180 words** that can be
  quoted without the preceding paragraph. Each passage restates its subject rather than pronominalising
  across the heading boundary.
- **FR-5.** Any numeric, statistical, research, legal or standards claim MUST carry an inline citation
  to a **primary source** (peer-reviewed paper, standards body, government dataset, vendor
  documentation) — not a secondary blog. Citations render as a superscript link plus a `<Sources>`
  list at the foot, and populate `Article.citation[]` (SEO.3 FR-11).
- **FR-6.** Every page MUST declare `published`, `updated`, and (where it makes a pedagogical,
  accessibility, or compliance claim) `reviewedBy` + `reviewedAt` in front-matter, rendered visibly.
- **FR-7.** Every page MUST end with a `<FAQ>` block of 3–6 real questions with 40–80-word answers,
  matching the visible text exactly in the emitted `FAQPage` schema.
- **FR-8.** Comparisons MUST use `<ComparisonTable>` (a real `<table>` with `<caption>`, `<th scope>`,
  and a text summary above it) rather than prose — tables are disproportionately extractable and are
  also the accessible form.
- **FR-9.** Procedural content MUST use `<Steps>`, which renders an `<ol>` and emits `HowTo` schema
  with matching step text.
- **FR-10.** Term definitions MUST use `<Definition term="…">`, which renders a definition block and
  is harvested by [SEO.10](SEO.10-programmatic-utility-pages.md) to build the glossary — so a term is
  defined once and reused.

**Authoring pipeline**

- **FR-11.** Content MUST move from plain markdown to **MDX** (or a markdown directive syntax) so the
  components above are usable in `www/src/blog/*` and `www/src/docs/*`, compiled at build time with
  no runtime markdown renderer (SEO.4 FR-5).
- **FR-12.** Front-matter MUST be schema-validated at build time. Required: `title`, `description`,
  `published`, `updated`, `author`, `cluster`, `primaryQuestion`, `keywords[]`. Optional:
  `reviewedBy`, `reviewedAt`, `relatedTo[]`, `noindex`. Unknown keys MUST fail the build.
- **FR-13.** `description` MUST be 120–160 characters and MUST NOT duplicate the `title`.
- **FR-14.** `primaryQuestion` MUST be the exact question the `<AnswerBox>` answers; the build MUST
  fail if no `<AnswerBox>` is present when `primaryQuestion` is set.

**Extractability score**

- **FR-15.** The build MUST compute a per-page **extractability score (0–10)** from deterministic
  signals:

  | Signal | Weight | Pass condition |
  |---|---|---|
  | `<KeyTakeaways>` present, 3–5 items | 1.0 | required |
  | `<AnswerBox>` present, 40–60 words | 1.5 | required |
  | Question-form headings ≥ 60% of `<h2>` | 1.5 | — |
  | Mean passage length 120–180 words | 1.5 | — |
  | Citations ≥ 1 per 400 words, all primary-source | 2.0 | ≥1 required if any statistic present |
  | ≥1 table, list, or steps block | 1.0 | — |
  | `<FAQ>` present with 3–6 Q&A | 1.0 | required |
  | Internal links ≥ 3 with descriptive anchors | 0.5 | ≥3 required (SEO.5 FR-16) |

- **FR-16.** New pages MUST score **≥ 8.0** to build. Pages scoring 6.0–7.9 MUST warn. Existing pages
  are grandfathered until their scheduled refresh, tracked in a report.
- **FR-17.** The build MUST emit `dist/.content-quality.json` per page (score, per-signal breakdown,
  word count, passage-length histogram, citation count, reading level) for SEO.15/SEO.16.
- **FR-18.** A `npm run content:lint` command MUST run the same checks locally with human-readable
  output naming file, line, and fix.

**Editorial standards**

- **FR-19.** Reading level MUST target grade 9–11 (Flesch–Kincaid) for marketing/guide content and
  grade 8–10 for help content; the build reports it and warns outside range.
- **FR-20.** Every page MUST state its perspective explicitly where it is commercial: comparison and
  alternatives pages MUST carry a visible disclosure that Lextures is the publisher (see SEO.9).
- **FR-21.** Claims about competitors MUST cite the competitor's own public documentation with an
  access date, and MUST be re-verified on the SEO.16 refresh cycle.

## 6. Non-Functional Requirements

- **Performance** — MDX compiles at build time; zero markdown/JSX runtime in the client bundle
  (SEO.4 FR-5). Component CSS ships in the shared stylesheet, not per-page.
- **Security** — MDX allows JSX, which is a code-execution surface in content files. Only components
  from an explicit allowlist may be used; arbitrary JSX expressions, raw HTML, and imports inside
  content files MUST be rejected at build time.
- **Privacy & Compliance** — content making accessibility or privacy claims requires `reviewedBy`
  (FR-6); claims about FERPA/GDPR/WCAG conformance MUST match the shipped position in
  [S09](../standards/S09-ferpa-hardening.md), [S12](../standards/S12-gdpr-uk-swiss-accountability-hardening.md),
  and the VPAT. The reviewer for such pages MUST be the compliance owner, not a writer.
- **Accessibility** — every component is WCAG 2.2 AA: tables have captions and scoped headers,
  callouts are not colour-only, `<Steps>` renders a real `<ol>`, `<FAQ>` uses a disclosure pattern
  with proper `aria-expanded` (or renders open — preferred, because collapsed content is less
  extractable and less accessible).
- **Scalability** — content lint over 500 pages in < 60 s; incremental in watch mode.
- **Reliability** — score computation is deterministic; the same input always produces the same score
  (no model calls — see §11).
- **Observability** — `.content-quality.json` feeds a dashboard: score distribution, pages below
  threshold, staleness, citation counts.
- **Maintainability** — components live in `www/src/components/content/`, one file each, with a
  Storybook-style examples page at `/internal/content-kit` (noindex) for writers.
- **Internationalization** — component copy (e.g. "Key takeaways", "Sources") comes from i18n keys;
  reading-level checks are English-only and MUST be skipped for other locales.
- **Backward compatibility** — the five existing posts and six help articles must continue to build
  during migration; markdown and MDX coexist until the retrofit completes.

## 7. Acceptance Criteria

- **AC-1.** *Given* a new MDX page missing `<AnswerBox>`, *When* the build runs, *Then* it fails
  naming the file and the missing requirement.
- **AC-2.** *Given* a page whose `<AnswerBox>` is 95 words, *When* the build runs, *Then* it fails
  with the actual word count and the 40–60 target.
- **AC-3.** *Given* a page containing "73% of teachers" with no citation, *When* the build runs,
  *Then* it fails naming the uncited statistic and its line number.
- **AC-4.** *Given* a page scoring 7.2, *When* the build runs, *Then* it warns with a per-signal
  breakdown and does not fail; *Given* a page scoring 5.0, *Then* it fails.
- **AC-5.** *Given* content using a component not on the allowlist, or raw HTML, or an `import`,
  *When* the build runs, *Then* it fails with a security-policy message.
- **AC-6.** *Given* the five existing blog posts after retrofit, *When* scored, *Then* all five score
  ≥ 8.0 and their arguments are unchanged (verified by editorial review, not automation).
- **AC-7.** *Given* a `<FAQ>` block, *When* the page is generated, *Then* the emitted `FAQPage`
  schema answers are byte-identical to the visible answers.
- **AC-8.** *Given* `<Definition term="formative assessment">` on any page, *When* the glossary builds
  (SEO.10), *Then* that definition appears at `/glossary/formative-assessment` with a link back to
  the source page.
- **AC-9.** *Given* any content component rendered, *When* axe runs, *Then* zero violations; *And*
  the `<FAQ>` disclosure is keyboard-operable with correct `aria-expanded`.
- **AC-10.** *Given* `npm run content:lint`, *When* run on a file with three issues, *Then* output
  names file, line, rule, and a concrete fix for each.

## 8. Data Model

No database changes. Content-side schema:

```yaml
# Required front-matter (validated, unknown keys rejected)
title: "How do you write a rubric that AI can't game?"
description: "…120–160 chars, not a repeat of the title…"
published: 2026-10-06
updated: 2026-10-06
author: chase-willden           # must exist in the author registry (SEO.3 FR-20)
cluster: assessment             # drives Related + hub membership (SEO.5)
primaryQuestion: "How do you write a rubric that AI can't game?"
keywords: [rubric design, ai-resistant assessment, analytic rubric]
# Optional
reviewedBy: <author-slug>
reviewedAt: 2026-10-04
relatedTo: [/resources/guides/assessment-design, /platform/assessment]
noindex: false
```

| Artefact | Path | Purpose |
|---|---|---|
| Components | `www/src/components/content/*.tsx` | Allowlisted MDX component set |
| Lint rules | `www/scripts/content-lint/*.mjs` | Deterministic checks (FR-15) |
| Quality report | `dist/.content-quality.json` | Per-page scores + signals |
| Definitions index | `dist/.definitions.json` | Harvested `<Definition>` blocks for SEO.10 |
| Writer reference | `/internal/content-kit` (noindex) | Live component examples |

## 9. API Surface

No HTTP surface. New npm scripts:

| Script | Purpose |
|---|---|
| `npm run content:lint` | Run all checks with human output (FR-18) |
| `npm run content:lint -- --fix` | Auto-fix mechanical issues (heading case, source-list ordering) |
| `npm run content:score <path>` | Score one file with the signal breakdown |
| `npm run content:report` | Score distribution + pages below threshold + staleness |

## 10. UI / UX

- **New components** (all server-rendered, zero client JS except `<FAQ>` disclosure):
  - `<KeyTakeaways>` — bordered card, "Key takeaways" heading, 3–5 bullets
  - `<AnswerBox>` — visually distinct lead paragraph, subtle background, no icon-only meaning
  - `<Definition term>` — term + definition, links to `/glossary/:term`
  - `<ComparisonTable>` — responsive table, horizontal scroll container, caption + summary
  - `<Steps>` — numbered `<ol>` with per-step optional screenshot slot
  - `<FAQ>` — `<h2>Frequently asked questions</h2>` + Q/A pairs, **rendered expanded by default**
  - `<Callout type="note|warning|tip">` — icon + label text (never colour-only)
  - `<Stat value source>` — pull-quote number with inline source link
  - `<Sources>` — auto-collected numbered source list with access dates
- **Modified pages:** `blog-post.tsx`, `docs-post.tsx` gain byline, takeaways slot, sources footer,
  updated/reviewed line, and the Related module from SEO.5.
- **Flows**
  1. Reader lands mid-page from an AI citation → sees the takeaways card and the answer box → gets the
     answer → follows an internal link.
  2. Writer runs `content:lint` locally → fixes → commits → CI confirms.
- **States** — a page with no sources renders no `<Sources>` section (rather than an empty heading).
- **Responsive** — tables scroll horizontally inside their own container; the page body never scrolls
  horizontally. Takeaway cards go full-width under 640 px.
- **Accessibility** — see NFRs. Specifically: `<Callout>` conveys type via text label + icon + shape,
  never colour alone; `<FAQ>` open-by-default avoids hiding content from both readers and extractors.
- **Copy & i18n** — `www.content.keyTakeaways`, `.answer`, `.sources`, `.faq`, `.updated`,
  `.reviewedBy`, `.callout.*`.

## 11. AI / ML Considerations

This plan is *about* AI consumption but deliberately uses **no models in the pipeline**:

- **Why deterministic.** The March 2026 core update targeted scaled content abuse; sites publishing
  volume without editorial oversight lost 50–80% of traffic
  ([research §7](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)). A
  scoring system that is deterministic and inspectable keeps the quality bar auditable, and keeps
  "did a human decide this?" answerable.
- **AI as a drafting aid is permitted, with rules.** Writers may draft with AI; the byline is the
  human who verified it, every statistic must be traced to a primary source by a human (FR-5), and
  the reviewer for compliance-adjacent claims is a named owner (FR-6). Google's own guidance is that
  AI-generated content is not penalised — low-value content is, regardless of origin.
- **Passage design targets retrieval, not word count.** FR-4's 120–180-word window brackets the
  measured 134–167-word citation sweet spot; FR-2's 40–60-word answer box targets the snippet/direct
  answer length.
- **A future evaluation loop** (out of scope, noted): sample published passages, ask each of the six
  tracked assistants the `primaryQuestion`, and record whether our passage is cited — that is
  [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md)'s prompt harness, not this
  plan's.

## 12. Integration Points

- **External:** MDX toolchain (`@mdx-js/rollup`), `remark`/`rehype` plugins for lint rules,
  readability library for FR-19.
- **Internal modules touched:** `www/src/utils/blog.ts`, `www/src/utils/docs.ts` (front-matter parsing
  → schema validation), `www/src/pages/blog-post.tsx`, `www/src/pages/docs-post.tsx`,
  `www/vite.config.ts`, `www/src/blog/*.md` and `www/src/docs/*.md` (migrated to `.mdx`),
  `www/scripts/generate-site.mjs`, `www/package.json` (scripts + deps).
- **Events:** none.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md) (build-time rendering),
  [SEO.5](SEO.5-information-architecture-and-internal-linking.md) (`cluster`, `relatedTo`, internal
  link rule).
- **Must ship before:** [SEO.7](SEO.7-help-center-expansion.md),
  [SEO.8](SEO.8-editorial-engine-and-content-calendar.md),
  [SEO.9](SEO.9-comparison-alternatives-and-integration-pages.md),
  [SEO.10](SEO.10-programmatic-utility-pages.md),
  [SEO.12](SEO.12-original-research-and-data-program.md) — every content plan writes to this contract.
- **Coordinates with:** [SEO.3](SEO.3-structured-data-and-entity-graph.md) — `<FAQ>`, `<Steps>`,
  `<Definition>`, and `<Sources>` feed `FAQPage`, `HowTo`, `DefinedTerm` and `citation[]`.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Format rules flatten voice into template prose | M | H | The contract governs *structure*, not argument; existing posts keep their thesis through retrofit (AC-6 requires editorial sign-off, not a script) |
| Score becomes a target and gets gamed | M | M | Score is a floor, not a goal; editorial review remains mandatory; signals are structural (citations, tables) so gaming them produces genuinely better pages |
| MDX opens a code-execution surface in content | M | H | FR-allowlist + no raw HTML + no imports, enforced at build (AC-5); content files reviewed like code |
| Migration to MDX breaks existing post rendering | M | M | Golden-file HTML snapshots before/after; markdown and MDX coexist during migration |
| Citation requirement slows publishing | H | M | Real cost, accepted: it is the highest-impact factor measured (+132%). Mitigate with a shared source library of vetted primary sources per cluster |
| Reading-level check fights technical accuracy | M | L | Warn-only; explicitly overridable for technical help content |
| Grandfathered pages never get refreshed | M | M | SEO.16 lifecycle assigns every page a review date; the report lists offenders |

## 15. Rollout Plan

- **Feature flag:** none. Staged by capability.
- **Sequencing**
  1. Front-matter schema validation on existing markdown (no MDX yet) — surfaces gaps immediately.
  2. MDX toolchain + component library + `/internal/content-kit`.
  3. Lint rules in **warn** mode; publish the score for all existing pages.
  4. Retrofit the 5 blog posts + 6 help articles; editorial review of each.
  5. Flip to **fail** for new files only (grandfather existing).
  6. Wire `.content-quality.json` into the SEO.16 dashboard and the SEO.15 report.
- **Dogfood:** the first three SEO.8 articles are written against the contract before it is enforced,
  to find friction.
- **GA criteria:** AC-1…AC-10 pass; all new content scores ≥8.0 for four consecutive weeks; writers
  report the checklist as usable (short survey).
- **Rollback:** lint mode flips warn/fail with one config value; components are additive.

## 16. Test Plan

- **Unit** — each lint rule against fixture content (pass + fail cases); score computation;
  front-matter validation incl. unknown-key rejection; word-count and passage-segmentation logic;
  citation detection (numbers, percentages, "studies show" patterns).
- **Integration** — full content build; `.content-quality.json` shape; `<FAQ>` → `FAQPage` byte
  equality (AC-7); `<Definition>` harvest → `.definitions.json` (AC-8).
- **End-to-end** — Playwright renders one page containing every component, asserts server-rendered
  presence with JS disabled and correct `<FAQ>` behaviour with JS on.
- **Security** — attempt raw HTML, an `import`, and a non-allowlisted component in a fixture; assert
  build failure (AC-5). Attempt a `javascript:` URL in a source link; assert rejection.
- **Accessibility** — axe on the content-kit page covering every component in light and dark; NVDA +
  VoiceOver on `<FAQ>` and `<ComparisonTable>`; 320 px and 200% zoom checks on tables.
- **Performance / load** — assert no markdown/MDX runtime in the client bundle; content lint runtime
  over a 500-file fixture < 60 s.
- **Manual exploratory** — editorial review of the retrofitted posts; a writer unfamiliar with the
  system authors one article using only `www/docs/content-contract.md`.

## 17. Documentation & Training

- `www/docs/content-contract.md` — the contract, each rule, and *why* (with the research citation),
  plus a copy-paste article skeleton.
- `www/docs/content-components.md` — component reference with examples; links to
  `/internal/content-kit`.
- `www/docs/sources-policy.md` — what counts as a primary source, how to cite, access dates,
  re-verification cadence.
- Writer onboarding: 45-minute walkthrough + the checklist in the PR template.
- Update the PR template with the content checklist for any change under `src/blog` or `src/docs`.

## 18. Open Questions

1. MDX vs. markdown directives (`:::answer`)? MDX is more flexible but is a code surface; directives
   are safer but less expressive. (Recommendation: MDX with a strict allowlist.)
2. Should `<FAQ>` render expanded by default (better for extraction and accessibility) or collapsed
   (denser page)? Recommendation is expanded — confirm with design.
3. Who is the named reviewer for accessibility/privacy claims, and what is their SLA?
4. Do we maintain a shared, vetted source library per cluster, and who curates it?
5. What is the minimum score for *help* content — the same 8.0, or lower given its different shape?

## 19. References

- Existing files: `www/src/blog/*.md`, `www/src/docs/*.md`, `www/src/utils/blog.ts`,
  `www/src/utils/docs.ts`, `www/src/pages/blog-post.tsx`, `www/src/pages/docs-post.tsx`
- Audit findings: [F-13](audit.md#f-13-content-is-essay-shaped-not-passage-shaped),
  [F-11](audit.md#f-11-zero-named-authors)
- Research: [§3 What actually earns an AI citation](research.md#3-what-actually-earns-an-ai-citation),
  [§7](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)
- External: [Google — Creating helpful, reliable, people-first content](https://developers.google.com/search/docs/fundamentals/creating-helpful-content),
  [Google — Guidance on gen-AI content](https://developers.google.com/search/docs/fundamentals/using-gen-ai-content),
  [W3C WAI — Tables tutorial](https://www.w3.org/WAI/tutorials/tables/)
- Related plans: [SEO.3](SEO.3-structured-data-and-entity-graph.md),
  [SEO.7](SEO.7-help-center-expansion.md), [SEO.8](SEO.8-editorial-engine-and-content-calendar.md),
  [SEO.10](SEO.10-programmatic-utility-pages.md),
  [SEO.16](SEO.16-seo-governance-and-ci-guardrails.md)
