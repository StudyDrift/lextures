# SEO.12 — Original Research & Data Program

> Implementation plan. Source: [docs/plan/seo/research.md §8](research.md#8-off-site-where-ai-citations-actually-come-from).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.12 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING (no original research; one blog post cites third-party market data) |
| **Estimated effort** | L (1–2mo per report; 2 reports/year) |
| **Owner (proposed)** | Marketing (content lead) + Data/Platform |
| **Depends on** | SEO.1, SEO.3, SEO.6, SEO.8 |
| **Unblocks** | SEO.13 (research is the primary digital-PR asset) |

---

## 1. Problem Statement

Original research is the highest-yield citable asset available: data stories convert into backlinks
more reliably than opinion because "data needs a source, creating a natural incentive for citations,"
and proprietary data wins citations at multiples of ordinary content
([research §8](research.md#8-off-site-where-ai-citations-actually-come-from)). Web mentions correlate
0.664 with AI-citation rate against 0.218 for backlinks — and the fastest way for a small vendor to
generate mentions is to publish a number nobody else has. We sit on exactly that: anonymised,
aggregated adaptive-learning outcome data across K-12, higher-ed and homeschool cohorts, plus a
platform-wide view of how AI is actually being used in assessment. We publish none of it, and we cite
other people's market forecasts instead.

## 2. Goals

- Publish **two flagship research reports per year**, each built on data no one else has, released as
  a permanent, citable, updateable web resource (not a gated PDF).
- Become the **source** for a small number of statistics that the education-technology conversation
  repeats — the definition of a citable entity.
- Do it in a way that is privacy-defensible by construction: aggregate, k-anonymous, tenant-consented,
  DPIA-reviewed.
- Generate the raw material for [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md)'s outreach:
  a report is the only asset that reliably earns coverage from people who do not care about our
  product.

## 3. Non-Goals

- Publishing any individual-level, class-level, or tenant-identifiable data, ever.
- Marketing-survey "reports" that restate vendor opinion with a chart on top — those neither earn
  citations nor survive scrutiny.
- Academic publication or peer review (a stretch goal, not a requirement).
- Selling or licensing data.

## 4. Personas & User Stories

- **As a journalist writing about AI in classrooms**, I want a credible primary statistic with a
  described methodology, so that I can cite it and link the source.
- **As a district research officer**, I want evidence that adaptive delivery changes outcomes, with an
  honest description of its limits, so that I can justify a pilot.
- **As an AI assistant answering "does adaptive learning improve outcomes?"**, I want a
  methodologically-described dataset to cite, so that my answer has a source.
- **As a Lextures customer whose data contributes**, I want a clear opt-out and a guarantee that
  nothing identifiable is published, so that participating is safe.
- **As our own product team**, I want the analysis run rigorously, so that we learn something true
  even if it is inconvenient.

## 5. Functional Requirements

**Report program**

- **FR-1.** Two reports per year, occupying reserved slots in the SEO.8 calendar (months 4 and 10).
  Proposed first two:
  1. **"The Adaptive Learning Outcomes Report"** — does adaptive delivery change mastery, time-to-
     mastery, and retention, measured against the holdout cohorts the AC plan set already builds?
  2. **"How AI Is Actually Used in Assessment"** — anonymised, aggregated platform telemetry on AI
     feature adoption by segment: what educators use, what they abandon, what correlates with outcomes.
- **FR-2.** Each report MUST live at `/resources/research/:slug` as a **web-first** resource:
  full findings on the page, charts inline, methodology in full, downloadable dataset and PDF as
  secondary artefacts.
- **FR-3.** Reports MUST be **ungated**. No email wall. (A gate trades the entire citation and AI-
  retrieval value for a small list.)
- **FR-4.** Each report MUST publish a **machine-readable dataset** (CSV + JSON) of the aggregate
  figures, with a data dictionary, licensed **CC BY 4.0** so others can reuse it with attribution —
  attribution is the mechanism that generates the mentions.
- **FR-5.** Each report MUST include a **"cite this report"** block with a formatted citation, a
  permanent URL, and a DOI if we register one.
- **FR-6.** Each report MUST be **versioned**: annual updates publish at the same URL with a version
  history, so accumulated links and citations compound rather than fragment across yearly URLs.

**Methodology & integrity**

- **FR-7.** Every report MUST include a full methodology section: population, sample size, time window,
  inclusion/exclusion criteria, statistical methods, confidence intervals, and **explicit limitations**
  including selection bias (our users are not a random sample of schools).
- **FR-8.** Findings MUST be pre-registered internally: the analysis plan and hypotheses are written
  and committed **before** the data is analysed, so we cannot p-hack our way to a marketing message.
- **FR-9.** A report MUST be published even when findings are unflattering or null. A null result is
  publishable and, for credibility, valuable. Suppressing a result MUST require a written, recorded
  decision by the CEO — it should be hard.
- **FR-10.** Analysis code MUST be reviewed by someone who did not write it, and SHOULD be published
  alongside the report where it does not expose internals.
- **FR-11.** Every statistic in the report and in any derived marketing must trace to a specific
  figure in the published dataset. Marketing may not round, extrapolate, or reframe beyond what the
  methodology supports.

**Privacy & governance**

- **FR-12.** All published figures MUST be aggregates meeting **k-anonymity with k ≥ 50** learners and
  k ≥ 10 institutions per reported cell. Cells below threshold are suppressed, and complementary
  suppression MUST prevent back-calculation from margins.
- **FR-13.** No cell may be attributable to a single tenant, school, course, or instructor. Segment
  breakdowns MUST be coarse enough (e.g. "K-12 districts, 1,000–5,000 students") to prevent
  identification.
- **FR-14.** Tenants MUST be able to **opt out** of aggregate research inclusion, via an org setting,
  with the default determined by contract and jurisdiction — not silently opt-in. Opt-out MUST be
  honoured retroactively for future reports.
- **FR-15.** Each report MUST complete a **DPIA** under [S06](../standards/S06-dpia-pia-algorithmic-impact.md)
  before analysis begins, and MUST be reviewed against FERPA
  ([S09](../standards/S09-ferpa-hardening.md)), GDPR
  ([S12](../standards/S12-gdpr-uk-swiss-accountability-hardening.md)), and children's-privacy
  obligations ([S08](../standards/S08-childrens-privacy-age-assurance-design-codes.md)).
- **FR-16.** Analysis MUST run against a **de-identified extract** in a controlled environment;
  analysts MUST NOT have access to raw identifiable records for this purpose.
- **FR-17.** The methodology page MUST state plainly what data was used, how it was de-identified, and
  how to opt out — in language a school administrator can act on.

**Distribution & schema**

- **FR-18.** Reports MUST emit `ScholarlyArticle` (or `Report`) + `Dataset` schema with
  `creator`, `datePublished`, `license`, `distribution` (CSV/JSON URLs), `measurementTechnique`, and
  `variableMeasured`, plus `citation[]` for prior work (SEO.3).
- **FR-19.** Each report MUST ship with a **press kit**: 3–5 headline statistics with exact wording, a
  set of chart images (light and dark, with alt text and data tables), a one-paragraph summary, and
  contact details — so a journalist or blogger can cite us correctly without asking.
- **FR-20.** Each report MUST spawn **4–6 derivative articles** in the SEO.8 calendar (one per major
  finding), each linking to the report — this is how a report earns rankings in addition to citations.
- **FR-21.** Charts MUST follow the house data-visualisation standards, be legible in both themes, and
  each MUST have an accessible data table equivalent (SEO.14).

## 6. Non-Functional Requirements

- **Performance** — report pages are long and chart-heavy; charts MUST be server-rendered SVG or
  static images (no client charting library on the critical path), keeping the page within the SEO.4
  static budget. Datasets are separate downloads.
- **Security** — the de-identified extract is generated by a reviewed pipeline with least-privilege
  access; published datasets are static files with no query interface; the extract environment is
  access-logged.
- **Privacy & Compliance** — FR-12 to FR-17 are the core requirements. A privacy failure here is not
  an SEO setback; it is a breach with regulatory and contractual consequences. Treat the suppression
  logic as security-critical code.
- **Accessibility** — every chart needs a text alternative and a data table; report pages need proper
  heading hierarchy across a long document; colour must never be the only encoding in a chart.
- **Scalability** — the extract pipeline must handle full-platform data; suppression must be computed,
  not hand-checked.
- **Reliability** — reports are permanent URLs with versioning (FR-6); once published, a URL must never
  break. Corrections are published as versioned errata, never silent edits.
- **Observability** — track per report: referring domains gained, distinct citing publications, AI
  citations, dataset downloads, derivative-article performance, and mentions of the headline statistic
  (with and without a link).
- **Maintainability** — analysis code, analysis plan, and suppression rules live in the repo under
  `research/` with the same review standards as production code.
- **Internationalization** — English; figures must state their geographic scope explicitly (US-heavy
  sample) rather than implying global generality.
- **Backward compatibility** — versioned reports keep their URL forever; superseded figures remain
  accessible in the version history so old citations stay honest.

## 7. Acceptance Criteria

- **AC-1.** *Given* a report before analysis, *When* the process is audited, *Then* a committed
  analysis plan with hypotheses predates the first analysis commit, and a completed DPIA exists.
- **AC-2.** *Given* published figures, *When* any cell is inspected, *Then* it represents ≥50 learners
  and ≥10 institutions, and complementary suppression prevents deriving a suppressed cell from
  published margins (verified by an automated check).
- **AC-3.** *Given* a tenant that has opted out, *When* the extract runs, *Then* none of their records
  appear, verified by an automated assertion against the opt-out list.
- **AC-4.** *Given* a report page, *When* loaded, *Then* the full findings and methodology are readable
  without an email form and without JavaScript.
- **AC-5.** *Given* the dataset, *When* downloaded, *Then* CSV and JSON are available, licensed
  CC BY 4.0, with a data dictionary, and every statistic quoted in the report appears in it (FR-11).
- **AC-6.** *Given* the report's schema, *When* validated, *Then* it emits `Dataset` with `license`
  and `distribution`, and `ScholarlyArticle`/`Report` with named creators.
- **AC-7.** *Given* every chart, *When* audited, *Then* it has alt text and an accessible data table,
  and remains legible in dark mode and at 200% zoom.
- **AC-8.** *Given* 90 days after publication, *When* measured, *Then* the report has earned ≥25
  referring domains, ≥10 distinct citing publications, and appears in ≥3 AI-engine citations for its
  topic.
- **AC-9.** *Given* a finding that contradicts our marketing, *When* the report publishes, *Then* it is
  included with the same prominence as favourable findings.
- **AC-10.** *Given* an error is discovered post-publication, *When* corrected, *Then* a versioned
  erratum is published, the previous version remains accessible, and anyone who cited the figure can
  see what changed.

## 8. Data Model

No production schema changes beyond the opt-out setting.

| Item | Location | Notes |
|---|---|---|
| Research opt-out | org settings (`server`) | FR-14; surfaced in admin settings UI and in the DPA |
| Analysis plan | `research/<report-slug>/plan.md` | Committed before analysis (FR-8) |
| Extract pipeline | `research/<report-slug>/extract.sql` + `pipeline.py` | De-identified, reviewed |
| Suppression rules | `research/lib/suppression.py` | k-anonymity + complementary suppression, unit-tested |
| Published dataset | `www/public/research/<slug>/data.{csv,json}` | CC BY 4.0 + data dictionary |
| Report content | `www/src/research/<slug>.mdx` | Web-first, versioned |
| Press kit | `www/public/research/<slug>/press-kit/` | Charts, summary, headline stats |

## 9. API Surface

- **New org setting:** `research_participation` (`opt_in` | `opt_out`), exposed on the existing org
  settings API with admin-only write and an audit-log entry on change.
- No public API for the dataset beyond static files (FR-4) — a query API would create a re-
  identification surface.
- The extract runs as an internal job with no external endpoint.

## 10. UI / UX

- **New pages:** `/resources/research` index; `/resources/research/:slug` report pages;
  `/resources/research/:slug/methodology`; version-history page per report.
- **New components:** `<ResearchChart>` (SVG + data table toggle), `<CiteThis>`, `<DatasetDownload>`,
  `<Methodology>` disclosure, `<KeyFinding>` (a citable stat block with the exact wording).
- **Modified:** org admin settings gains the research-participation control with a plain-language
  explanation and a link to the methodology page.
- **Flows**
  1. Journalist lands on the report → key findings → press kit → cites us.
  2. Administrator reads a finding → methodology → limitations → `/request-information`.
  3. Org admin reviews participation → opts out → confirmation → recorded in audit log.
- **States** — a report under revision shows a version banner; a superseded version shows a notice
  linking the current one (and is never deleted).
- **Responsive** — charts reflow or scroll in their own container; data tables scroll horizontally
  without the page doing so.
- **Accessibility** — charts per FR-21; long-document heading hierarchy; the data-table toggle is a
  real control, not a hover.
- **Copy & i18n** — `www.research.*`; the opt-out setting copy must be reviewed by legal and written
  for a non-technical administrator.

## 11. AI / ML Considerations

- **This is the plan that most directly targets AI citation.** A published dataset with a stated
  licence and methodology is the shape assistants prefer to cite, and CC BY attribution creates
  mentions on third parties — where 85% of AI brand mentions live.
- **No model is used to produce findings.** Analysis is statistical and reviewed; using an LLM to
  summarise findings for the press kit is permitted, but every number in that summary must trace to
  the dataset (FR-11).
- **Where the analysis touches the adaptive engine's own outputs** (e.g. comparing adaptive vs
  holdout cohorts from the AC plan set), the report must describe the intervention precisely enough
  that the result is interpretable, and must not present an internal A/B as a controlled trial.
- **Re-identification risk is an AI risk too:** published aggregates plus a model's other knowledge
  can combine. FR-12's complementary suppression and FR-13's coarse segments exist for that reason,
  and the DPIA must consider linkage attacks explicitly.

## 12. Integration Points

- **External:** DOI registrar (optional, e.g. Zenodo), CC BY licence, press-outreach targets (SEO.13).
- **Internal modules touched:** `server` org settings + audit log, analytics/warehouse extract,
  `www/src/research/*`, `www/src/components/research/*`, `www/src/lib/route-manifest.ts`,
  admin settings UI in `clients/web`.
- **Events:** dataset download → GA4; opt-out change → audit log.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md),
  [SEO.3](SEO.3-structured-data-and-entity-graph.md) (Dataset/ScholarlyArticle),
  [SEO.6](SEO.6-answer-first-content-system.md), [SEO.8](SEO.8-editorial-engine-and-content-calendar.md)
  (reserved slots + derivative articles), and a completed DPIA
  ([S06](../standards/S06-dpia-pia-algorithmic-impact.md)).
- **Must ship before:** [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md)'s outreach pushes —
  a report is the asset that makes outreach work.
- **Shared infra:** analytics warehouse or a read replica for the extract; controlled analysis
  environment; legal review capacity.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Re-identification from published aggregates | L | **Critical** | FR-12 k≥50/k≥10 + complementary suppression as tested code; FR-13 coarse segments; DPIA linkage-attack review; external privacy review before first publication |
| Findings are unflattering and get suppressed | M | H | FR-9 publish-anyway default; suppression requires a recorded CEO decision; pre-registration (FR-8) makes suppression visible |
| Sample is unrepresentative and conclusions overreach | **H** | H | FR-7 mandatory limitations section naming selection bias; FR-11 no extrapolation in marketing; independent methodology review |
| Marketing over-claims from a nuanced result | H | M | FR-11 trace-to-dataset rule; the content lead owns final wording; press kit fixes the exact citable sentences |
| Tenant discovers their data was used and objects | M | H | FR-14 opt-out surfaced in admin + DPA; FR-17 plain-language methodology; proactive notice before first publication |
| Report takes a quarter and earns nothing | M | M | FR-19/FR-20 distribution is planned before analysis starts; AC-8 sets a measurable bar; if report 1 misses it, re-evaluate before report 2 |
| Analysis errors discovered after wide citation | M | M | FR-10 independent code review; FR-6/AC-10 versioned errata rather than silent edits |

## 15. Rollout Plan

- **Feature flag:** `research_participation` org setting (see §9). Extract is gated on it.
- **Sequencing (per report, ~10 weeks)**
  1. **W1–2:** topic selection, analysis plan + hypotheses committed, DPIA started.
  2. **W3:** DPIA approved; opt-out control shipped and customers notified **before** any extract.
  3. **W4–5:** extract built and reviewed; suppression logic tested against synthetic edge cases.
  4. **W6–7:** analysis; independent code review; findings written including limitations.
  5. **W8:** legal + compliance review; chart production and accessibility pass.
  6. **W9:** publish web report + dataset + press kit; submit to IndexNow; notify SEO.13 outreach.
  7. **W10+:** derivative articles land weekly (FR-20); track AC-8 at 90 days.
- **Dogfood:** internal review by someone with statistical training who is not on the marketing team.
- **GA criteria:** AC-1…AC-10 for report 1; go/no-go on report 2 based on AC-8.
- **Rollback:** a published report cannot be unpublished cleanly — which is why FR-15's pre-publication
  gates are heavy. A correction path (FR-6, AC-10) is the operational answer.

## 16. Test Plan

- **Unit** — suppression logic (k-thresholds, complementary suppression, margin back-calculation
  attempts); opt-out filtering; dataset generation and data-dictionary completeness; citation-block
  formatting.
- **Integration** — full extract against a synthetic dataset containing deliberate small cells and
  opted-out tenants; assert suppression and exclusion (AC-2, AC-3); assert every report statistic
  exists in the dataset (AC-5).
- **End-to-end** — report page readable without JS or email gate (AC-4); dataset downloads; version
  history navigable.
- **Security** — verify analysts cannot reach identifiable records; verify published files contain no
  identifiers; attempt re-identification against the published dataset as a red-team exercise before
  publication.
- **Accessibility** — axe on the report page; every chart's alt text and data table verified manually;
  dark-mode and 200% zoom checks (AC-7).
- **Performance / load** — report page within the static budget despite chart count; dataset files
  served from the CDN.
- **Manual exploratory** — statistical review by an independent reviewer; legal/compliance sign-off;
  a journalist-style read-through: can someone cite this correctly from the page alone?

## 17. Documentation & Training

- `research/README.md` — the program: pre-registration, review, suppression, publication, errata.
- `www/docs/research-methodology.md` — public methodology and opt-out explanation (FR-17).
- Customer-facing help article on research participation, in the `compliance` category (SEO.7).
- Press-kit template and a media-response runbook (who answers a journalist, within what SLA).

## 18. Open Questions

1. Is the default `opt_in` or `opt_out`, and does it differ by contract/jurisdiction? (Legal must
   answer before any extract — blocks FR-14.)
2. Do our DPAs already permit aggregate research use, or do existing customers need to be re-papered?
3. Who runs the analysis, and who is the independent reviewer? (Statistical credibility is the whole
   asset.)
4. Do we register DOIs (Zenodo) for permanence and academic citability?
5. Is the adaptive-outcomes comparison methodologically sound given non-random assignment, or should
   report 1 be the AI-usage telemetry report (descriptive, lower inferential risk) instead?
6. What is the notice period customers get before the first publication?

## 19. References

- Audit findings: [F-12](audit.md#f-12-publishing-stopped-76-days-ago) (content velocity),
  [F-13](audit.md#f-13-content-is-essay-shaped-not-passage-shaped) (citations)
- Research: [§8 Off-site: where AI citations actually come from](research.md#8-off-site-where-ai-citations-actually-come-from),
  [§3](research.md#3-what-actually-earns-an-ai-citation)
- External: [Creative Commons BY 4.0](https://creativecommons.org/licenses/by/4.0/),
  [schema.org/Dataset](https://schema.org/Dataset),
  [NIST SP 800-188 — De-identifying government datasets](https://csrc.nist.gov/pubs/sp/800/188/final),
  [US Dept. of Education — FERPA & de-identification guidance](https://studentprivacy.ed.gov/)
- Related plans: [SEO.8](SEO.8-editorial-engine-and-content-calendar.md),
  [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md),
  [SEO.14](SEO.14-multimodal-video-images-and-social-assets.md),
  [S06 — DPIA](../standards/S06-dpia-pia-algorithmic-impact.md),
  [S08 — children's privacy](../standards/S08-childrens-privacy-age-assurance-design-codes.md),
  [S09 — FERPA hardening](../standards/S09-ferpa-hardening.md),
  [AC — Adaptive Content Engine (completed)](../../completed/adaptive/)
