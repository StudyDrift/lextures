# SEO.15 — Measurement: Search Console, GA4 & AI Share-of-Voice

> Implementation plan. Source: [docs/plan/seo/audit.md](audit.md) §S4 (F-21) and
> [research §9](research.md#9-measurement-in-2026).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.15 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN (GA4 installed with default config; no GSC/Bing property, no AI channel grouping, no rank tracking, no citation tracking, no CRM attribution) |
| **Estimated effort** | M (2–4w to build, then continuous) |
| **Owner (proposed)** | Growth / Analytics |
| **Depends on** | SEO.2 (GSC/Bing verification) |
| **Unblocks** | SEO.8, SEO.9, SEO.12, SEO.13 (all need attribution to justify continuation) |

---

## 1. Problem Statement

GA4 is installed with default configuration and nothing else (audit F-21): no custom channel group for
AI assistants, no UTM convention, no CRM handoff, no server-side view of AI-bot crawl activity, no
rank tracking, no citation tracking. GA4's native "AI Assistant" channel, shipped May 2026, **excludes
Perplexity and every session arriving without a referrer**, so the default view systematically
under-reports the channel. And only 12–18% of Perplexity citations produce a click — meaning
session-based measurement misses roughly 5× the actual visibility
([research §9](research.md#9-measurement-in-2026)). We are about to spend twelve months and real money
on SEO.7–SEO.14. Without a baseline captured **before** that work lands, none of it is attributable,
and the first budget question will end the program.

## 2. Goals

- Capture a **complete baseline in week 3** — before SEO.8's content program starts — across search,
  AI visibility, crawl, and conversion.
- Measure **AI Share of Voice** as the headline KPI: % of AI answers mentioning Lextures across a
  fixed prompt set on six engines.
- Attribute organic and AI-sourced traffic **end to end**, from session to MQL to closed-won.
- See what crawlers actually do on our site, including the AI bots that never send a referrer.
- Produce one weekly dashboard and one monthly review that the whole plan set is judged against.

## 3. Non-Goals

- Building an analytics product. We use GSC, Bing WMT, GA4, the CRM, edge logs, and one AI-visibility
  tool; the work is wiring and reporting, not platform engineering.
- Replacing GA4 (a possible follow-on given the privacy positioning — noted in SEO.4 open questions).
- Marketing attribution beyond organic and AI channels.
- Real-time alerting on ranking movements (weekly cadence is sufficient and less noisy).

## 4. Personas & User Stories

- **As the CEO**, I want to know whether the SEO program is producing pipeline, so that I can decide
  whether to keep funding it.
- **As the content lead**, I want per-article performance including AI citations, so that I can
  double down on what works and prune what does not.
- **As a web engineer**, I want to see which AI crawlers fetch which URLs and how often, so that I can
  tell whether SEO.1 and SEO.2 actually worked.
- **As the growth owner**, I want AI-sourced sessions to survive into CRM records, so that "ChatGPT
  sent us a customer" is a provable statement.
- **As anyone in the company**, I want one dashboard that answers "how is search going?", so that we
  are not reconstructing it from four tools each month.

## 5. Functional Requirements

**Baseline**

- **FR-1.** A **baseline snapshot** MUST be captured in week 3 and committed to
  `docs/plan/seo/baseline-2026-09.md`, covering: indexed pages (Google + Bing), organic
  clicks/impressions/position by query and page, referring domains and total backlinks, brand vs
  non-brand split, AI Share of Voice across the prompt set, third-party mention count, Core Web Vitals
  field data, and conversions by channel.
- **FR-2.** The baseline MUST be re-captured monthly with the same method, so the series is comparable.

**Search Console & Bing**

- **FR-3.** GSC (DNS-verified) and Bing Webmaster Tools MUST be connected (SEO.2 FR-16), with data
  exported daily via API into a warehouse table — GSC's UI retains only 16 months and samples heavily.
- **FR-4.** Reporting MUST segment **brand vs non-brand** queries using a maintained brand-term regex,
  and MUST segment by page family (`/docs`, `/blog`, `/resources`, `/compare`, `/courses`,
  `/glossary`, `/platform`).
- **FR-5.** Index-coverage state MUST be tracked per URL over time, so a page dropping out of the index
  is visible within a week rather than at the next audit.

**GA4 & attribution**

- **FR-6.** A **custom channel group** MUST be created that classifies AI-assistant traffic correctly,
  covering at minimum: `chatgpt.com`, `chat.openai.com`, `perplexity.ai`, `gemini.google.com`,
  `claude.ai`, `copilot.microsoft.com`, `bing.com/chat`, `you.com`, `grok.com`, `poe.com`. This
  supplements — not replaces — GA4's native AI Assistant channel, which excludes Perplexity and
  referrer-less sessions.
- **FR-7.** A **UTM convention** MUST be documented and applied to every link we place off-site
  (community answers, profiles, press, newsletters), so referrer-less arrivals are still classifiable.
- **FR-8.** A **hidden source field** MUST be added to `/request-information` and `/get-started` forms,
  populated from the first-touch channel stored in a first-party cookie, and MUST be passed into the
  CRM lead record so channel survives to closed-won.
- **FR-9.** Conversion events MUST be defined and instrumented: `request_information_submitted`,
  `get_started_started`, `pricing_calculator_completed`, `template_downloaded`, `dataset_downloaded`,
  `video_played`, `docs_helpful_vote`, `compare_cta_clicked`.
- **FR-10.** Web-vitals field events from SEO.4 FR-17 MUST land in the same warehouse for joint
  analysis (does a slow page rank worse for us?).

**AI visibility**

- **FR-11.** A **prompt set of 60 questions** MUST be defined and version-controlled, spanning:
  category queries ("best LMS for a small district"), capability queries ("which LMS has adaptive
  quizzing"), comparison queries ("Canvas alternatives"), brand queries ("what is Lextures"),
  problem queries ("how do I stop AI cheating on essays"), and segment queries ("homeschool
  curriculum platform with transcripts"). The set MUST NOT change without versioning, or the series
  breaks.
- **FR-12.** The prompt set MUST be run **weekly** against six engines — ChatGPT, Google AI
  Overviews/AI Mode, Gemini, Perplexity, Claude, Copilot — recording per prompt: mentioned (y/n),
  position in the answer, sentiment, which of our URLs was cited, and which competitors appeared.
- **FR-13.** **AI Share of Voice** MUST be computed as `mentions / total answers` per engine and
  overall, with the target trajectory 5% (3 mo) → 12% (6 mo) → ≥20% (12 mo).
- **FR-14.** **Entity Accuracy** MUST be tracked: for brand prompts, does the assistant state our
  founding, category, segments and capabilities correctly? Errors feed back into SEO.3 and SEO.13.
- **FR-15.** **Competitor share** MUST be tracked on the same prompt set, so our number has a
  denominator that means something.
- **FR-16.** Where a competitor or third party is cited instead of us, the **cited URL MUST be
  recorded** — that list is the highest-quality content backlog we will ever have.

**Crawl analytics**

- **FR-17.** Edge/CDN logs MUST be collected and analysed for bot traffic by user-agent, recording:
  requests per bot per day, URLs fetched, status codes returned, and bytes served. (Requires the
  SEO.1 FR-12 hosting migration; GitHub Pages exposes no logs.)
- **FR-18.** A weekly report MUST show: which AI bots visited, which sections they crawled, any
  non-200 responses they received, and first-crawl latency for newly published URLs.
- **FR-19.** Alerts MUST fire when: a major bot's crawl rate drops >50% week-over-week, any bot
  receives >5% non-200 responses, or a sitemap URL returns non-200.

**Reporting**

- **FR-20.** A **weekly dashboard** MUST cover: indexed pages, non-brand clicks, top movers, AI SoV,
  new citations, referring domains, crawl health, CWV pass rate, conversions.
- **FR-21.** A **monthly review document** MUST be produced from the same data with commentary,
  published to `docs/plan/seo/performance.md`, including what shipped, what moved, and what changes
  next month.
- **FR-22.** Every content plan's pages MUST be attributable to their plan (via the page-family
  segmentation in FR-4), so SEO.7/8/9/10/12 can each be evaluated on its own.

## 6. Non-Functional Requirements

- **Performance** — measurement must not slow the site. The web-vitals collector is ≤2 KB and idle-
  deferred (SEO.4 FR-17); GA4 is not on the critical path (SEO.4 FR-7).
- **Security** — API credentials for GSC/Bing/CRM live in secret storage; the warehouse is
  access-controlled; edge logs may contain IPs and MUST be access-restricted and retention-limited.
- **Privacy & Compliance** — IP anonymisation on; no PII in GA4 custom dimensions; the hidden source
  field (FR-8) carries a channel string, never a fingerprint; edge-log retention MUST have a defined
  limit and appear in the RoPA ([S05](../standards/S05-ropa-data-inventory-mapping.md)); if a consent
  banner is required in any market, analytics must be consent-gated and the dashboard must state the
  resulting coverage gap rather than silently under-reporting.
- **Accessibility** — internal dashboards should be usable by everyone on the team; if built as a web
  page, it follows the same WCAG standard as the site.
- **Scalability** — 60 prompts × 6 engines × weekly = 1,560 observations/month; storage and querying
  are trivial, but the collection must be automated or it will not happen.
- **Reliability** — collection failures MUST be visible (a missing week is worse than a wrong number,
  because it silently breaks the series). Each collector reports success/failure to the dashboard.
- **Observability** — the measurement system reports on itself: last successful collection per source.
- **Maintainability** — prompt set, brand regex, channel definitions and UTM convention live in the
  repo as versioned files.
- **Internationalization** — engine coverage and prompt sets are US-English at launch; SEO.17 would add
  locale-specific prompt sets.
- **Backward compatibility** — the existing GA4 property `G-JX182Q6KKX` is retained; historical data
  is preserved even if the measurement plan changes.

## 7. Acceptance Criteria

- **AC-1.** *Given* week 3, *When* the baseline is captured, *Then* `baseline-2026-09.md` exists with
  every FR-1 metric populated, and it predates the first SEO.8 article.
- **AC-2.** *Given* GSC and Bing, *When* checked, *Then* both are verified, sitemaps submitted, and
  daily API exports are landing in the warehouse with no gaps.
- **AC-3.** *Given* a session arriving from `perplexity.ai`, *When* reported, *Then* it appears in the
  custom AI-assistant channel (not "Referral" or "Direct").
- **AC-4.** *Given* a form submission from an AI-sourced session, *When* the CRM record is inspected,
  *Then* the first-touch channel is present on the lead and survives to the opportunity.
- **AC-5.** *Given* the weekly AI run, *When* it completes, *Then* 60 prompts × 6 engines are recorded
  with mention, position, sentiment, cited URL and competitors, and AI SoV is computed per engine.
- **AC-6.** *Given* a brand prompt, *When* the assistant's answer is evaluated, *Then* entity accuracy
  is scored and any factual error is logged with the engine and date.
- **AC-7.** *Given* edge logs, *When* the weekly crawl report runs, *Then* it shows per-bot request
  counts, crawled URLs, and status-code distribution; *And* an alert fires on the FR-19 conditions.
- **AC-8.** *Given* a newly published URL, *When* crawl latency is measured, *Then* time-to-first-crawl
  by Googlebot and by at least one AI bot is recorded.
- **AC-9.** *Given* any month, *When* the review document is produced, *Then* it reports per-plan
  performance (SEO.7 / 8 / 9 / 10 / 11 / 12 / 13) using the page-family segmentation.
- **AC-10.** *Given* a collector fails for a week, *When* the dashboard is viewed, *Then* the gap is
  visibly flagged rather than rendered as zero.

## 8. Data Model

Warehouse tables (or equivalent in a spreadsheet-plus-scripts setup at low volume):

| Table | Grain | Key fields |
|---|---|---|
| `gsc_daily` | date × query × page × device | clicks, impressions, ctr, position, brand_flag, page_family |
| `bing_daily` | date × query × page | clicks, impressions, position |
| `ai_visibility` | date × prompt_id × engine | mentioned, answer_position, sentiment, cited_urls[], competitors[], raw_answer_hash |
| `crawl_log_daily` | date × user_agent × page_family | requests, status_2xx/3xx/4xx/5xx, bytes |
| `mentions` | date × source_url | type (link/unlinked), sentiment, campaign (from SEO.13) |
| `web_vitals` | date × page × metric | p75_value, sample_count, element_selector |
| `conversions` | date × channel × page | event, count, crm_lead_id |

Repo artefacts:

```
docs/plan/seo/
  baseline-2026-09.md
  performance.md              # monthly review, FR-21
  measurement/
    prompts.yaml              # 60 prompts, versioned (FR-11)
    brand-terms.txt           # brand regex source (FR-4)
    channels.yaml             # AI referrer classification (FR-6)
    utm-convention.md         # FR-7
```

## 9. API Surface

Inbound to our warehouse (read-only, credentialed):

| Source | Endpoint | Cadence |
|---|---|---|
| Google Search Console | Search Analytics API | daily |
| Bing Webmaster Tools | API | daily |
| GA4 | Data API | daily |
| CrUX | `records:queryRecord` | weekly |
| CDN | log export | daily |
| AI-visibility tool **or** direct engine APIs | vendor API | weekly |
| CRM | lead/opportunity export | daily |

No new Lextures endpoints. The hidden-source form field (FR-8) uses the existing
`institution-inquiry-api` payload with one added string field.

## 10. UI / UX

- **Internal dashboard** (not public): weekly view with the FR-20 metrics, per-plan tabs, and a
  data-freshness banner per source.
- **Modified public surfaces:** hidden channel field on `/request-information` and `/get-started`
  (invisible to users, must not affect form accessibility or validation).
- **Flows**
  1. Monday: collectors run → dashboard updates → owner reviews alerts.
  2. Monthly: review doc generated → discussed → next month's priorities set.
  3. A page drops out of the index → coverage alert → investigated within the week.
- **States** — stale data source shows a "last updated" warning rather than a stale number presented
  as current (AC-10).
- **Responsive** — dashboard readable on a laptop; mobile not required.
- **Accessibility** — dashboard charts need text/table equivalents (same standard as SEO.14).
- **Copy & i18n** — internal only.

## 11. AI / ML Considerations

- **The prompt harness is the core instrument.** Automate it via vendor APIs where available; where an
  engine has no API (AI Overviews, some assistant surfaces), use an established AI-visibility tool
  rather than scraping — scraping is fragile, often prohibited, and would make the series unreliable.
- **Answers are non-deterministic.** Weekly sampling with a fixed prompt set gives a usable trend, but
  single-week movements are noise. Report 4-week rolling averages and never make a decision on one
  week's number.
- **Record the cited URL, always** (FR-16). "Perplexity cited a 2023 blog post from a competitor for
  our best capability query" is a content brief, not a statistic.
- **Entity accuracy (FR-14) is a feedback loop into SEO.3 and SEO.13**: if assistants consistently get
  our founding date or category wrong, the fix is entity data and third-party corroboration, not more
  content.
- Storing full answer text may raise vendor-terms questions; store a hash plus the extracted fields,
  and retain full text only where terms permit.

## 12. Integration Points

- **External:** Google Search Console, Bing Webmaster Tools, GA4, CrUX, CDN log export, CRM,
  AI-visibility tool, rank tracker.
- **Internal modules touched:** `www/index.html` (GA4 config per SEO.4 FR-7),
  `www/src/lib/institution-inquiry-api.ts` (+ hidden channel field),
  `www/src/pages/request-information-page.tsx`, `www/src/pages/get-started-page.tsx`,
  `www/src/lib/web-vitals.ts` (SEO.4), collectors under `scripts/seo-measurement/`.
- **Events:** all FR-9 conversion events.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md) (GSC/Bing verification).
  Crawl analytics (FR-17) additionally requires [SEO.1](SEO.1-static-rendering-and-crawlability.md)
  FR-12 hosting.
- **Must ship before:** [SEO.8](SEO.8-editorial-engine-and-content-calendar.md) — the baseline must
  predate the content program. Also before SEO.9, SEO.12, SEO.13, which are each judged on measured
  outcomes.
- **Shared infra:** warehouse (or a lightweight equivalent), CRM access, tool subscriptions.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Baseline is captured after content ships, destroying attribution | M | **H** | FR-1 fixes week 3; it is a hard dependency of SEO.8's start and appears in that plan's sequencing |
| AI-visibility data is noisy and gets over-interpreted | **H** | M | 4-week rolling averages; fixed versioned prompt set; competitor denominator (FR-15); never act on one week |
| Tool costs are not approved | M | M | Prompt harness can run against engine APIs directly at low cost for most of the six; AI Overviews is the one needing a tool |
| No edge logs because the hosting migration slips | M | M | FR-17 degrades to GSC crawl stats + server-side course-API logs until migration; note the gap in the dashboard |
| Collectors break silently and the series has holes | M | H | AC-10 freshness flags; per-source success reporting; a missing week is displayed, not zero-filled |
| Privacy exposure from edge-log retention | L | H | Retention limit + access control + RoPA entry (S05) |
| Dashboard becomes a monthly chore nobody reads | M | M | One page, ten numbers, one owner, monthly discussion slot; anything not acted on gets removed |

## 15. Rollout Plan

- **Feature flag:** none.
- **Sequencing**
  1. **Week 1:** verify GSC + Bing (with SEO.2); define brand terms, page families, prompt set, UTM
     convention; ship the FR-9 conversion events and the hidden channel field.
  2. **Week 2:** custom channel group; CRM field mapping; GA4 + GSC daily exports.
  3. **Week 3:** **capture the baseline** (FR-1) and commit it. This is the gate for SEO.8.
  4. **Week 4:** first weekly AI-visibility run; dashboard v1.
  5. **Weeks 5–6:** crawl analytics once hosting supports logs; alerts (FR-19).
  6. **Month 2 onward:** monthly review document; quarterly prompt-set review (add prompts, never
     silently change existing ones — version instead).
- **Dogfood:** run the monthly review twice internally before it becomes the official artefact.
- **GA criteria:** AC-1…AC-10; three consecutive monthly reviews produced without manual data
  reconstruction.
- **Rollback:** measurement is additive; the only user-facing change is a hidden form field.

## 16. Test Plan

- **Unit** — brand/non-brand classification; page-family assignment; AI SoV computation; channel
  classification against a fixture of referrer strings; UTM parsing.
- **Integration** — collectors write expected rows for a fixture day; a simulated collector failure
  surfaces as a freshness warning, not a zero (AC-10); form submission carries the channel through to
  a CRM sandbox record (AC-4).
- **End-to-end** — Playwright submits both forms and asserts the hidden channel value matches the
  first-touch cookie; asserts the field does not break validation or screen-reader flow.
- **Security** — no credentials in the repo; warehouse access review; edge-log access restricted;
  verify no PII lands in GA4 custom dimensions.
- **Accessibility** — hidden field must not be focusable or announced; dashboard charts have table
  equivalents.
- **Performance / load** — measurement adds no blocking requests; verify GA4 remains off the critical
  path after SEO.4's changes.
- **Manual exploratory** — quarterly: manually run 10 prompts and compare to the automated record, to
  catch harness drift.

## 17. Documentation & Training

- `docs/plan/seo/measurement/README.md` — every metric: definition, source, cadence, owner, and how to
  interpret it (including what a week-over-week move does *not* mean).
- `docs/plan/seo/measurement/utm-convention.md` — the tagging rules for SEO.13's off-site links.
- Runbook: collector failure recovery; re-verifying GSC after a host change; adding a prompt without
  breaking the series.
- Monthly review template with the standing questions.

## 18. Open Questions

1. Which AI-visibility tool, and is the budget approved? (AI Overviews coverage is the deciding
   factor.)
2. Which CRM, and who owns the field mapping for FR-8?
3. Do we have a warehouse, or do we start with scheduled scripts writing to BigQuery/Sheets?
4. What is the edge-log retention period, and who signs off on it for the RoPA?
5. Is a consent banner required in our markets? If so, the dashboard must state the coverage gap
   (interacts with SEO.4 open question 2).
6. Who owns the weekly dashboard review, and what is the escalation path when an alert fires?

## 19. References

- Existing files: `www/index.html` (:29-37 GA4), `www/src/lib/institution-inquiry-api.ts`,
  `www/src/pages/request-information-page.tsx`, `www/src/pages/get-started-page.tsx`
- Audit findings: [F-21](audit.md#f-21-ga4-is-installed-and-configured-for-the-wrong-era),
  [F-7](audit.md#f-7-no-bing--indexnow-path)
- Research: [§9 Measurement in 2026](research.md#9-measurement-in-2026),
  [§1](research.md#1-the-structural-shift-retrieval-replaced-ranking)
- External: [Google — Search Analytics API](https://developers.google.com/webmaster-tools/v1/searchanalytics/query),
  [Bing Webmaster Tools API](https://learn.microsoft.com/en-us/bingwebmaster/getting-access),
  [GA4 Data API](https://developers.google.com/analytics/devguides/reporting/data/v1),
  [CrUX API](https://developer.chrome.com/docs/crux/api)
- Related plans: [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md),
  [SEO.4](SEO.4-core-web-vitals-and-page-experience.md),
  [SEO.8](SEO.8-editorial-engine-and-content-calendar.md),
  [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md),
  [SEO.16](SEO.16-seo-governance-and-ci-guardrails.md),
  [S05 — RoPA / data mapping](../standards/S05-ropa-data-inventory-mapping.md)
