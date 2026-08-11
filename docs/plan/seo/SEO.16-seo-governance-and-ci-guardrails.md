# SEO.16 — SEO Governance, CI Guardrails & Content Lifecycle

> Implementation plan. Source: [docs/plan/seo/audit.md](audit.md) §S4 (F-22).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.16 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING (CI runs `oxlint` and `vite build`; no SEO assertions, no schema validation, no link checking, no lifecycle process) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform |
| **Depends on** | SEO.1, SEO.2, SEO.3, SEO.5, SEO.6 |
| **Unblocks** | Safe scaling of SEO.7, SEO.8, SEO.9, SEO.10, SEO.11 |

---

## 1. Problem Statement

`.github/workflows/pages-www.yml` runs `oxlint` and `vite build`, and `www/package.json`'s test script
covers six unit files — none of them SEO invariants (audit F-22). Nothing checks that a new route has
a title, that titles are unique, that the sitemap matches the route table, that JSON-LD validates, or
that a page returns 200. That absence is exactly how we arrived at 28 of 31 sitemap URLs returning
404 while the build stayed green for months. We are about to grow from ~30 pages to 500+ across five
content plans; without automated invariants, the same class of failure will recur at ten times the
scale, and content will silently rot.

## 2. Goals

- Make every SEO invariant from SEO.1–SEO.14 an **automated check that fails the build**, so
  regressions are caught in review rather than in Search Console 28 days later.
- Run a **post-deploy verification** against the live site, because a passing build is not the same as
  a correct deploy.
- Institute a **content lifecycle**: every page has an owner, a review date, and a defined end state
  (refresh, consolidate, or retire).
- Give the team one place to see quality debt: stale pages, `noindex` pages, orphans, low-score
  content, broken links, missing schema.
- Prevent the specific failure this plan set was written to fix from ever recurring.

## 3. Non-Goals

- Replacing human editorial review. The gates are structural; judgement stays human.
- Blocking unrelated engineering work — checks scope to `www/` and its build output.
- A commercial SEO-audit tool (may complement, does not replace repo-level gates).
- Rank monitoring and reporting (SEO.15).

## 4. Personas & User Stories

- **As a web engineer**, I want CI to tell me a route is missing metadata before merge, so that I do
  not learn about it from a traffic drop.
- **As the content lead**, I want a dashboard of stale and underperforming content, so that refresh
  work is planned rather than reactive.
- **As a reviewer**, I want the SEO checklist automated, so that review is about substance.
- **As the site owner**, I want a post-deploy check that every sitemap URL returns 200, so that the
  audit F-1 failure cannot silently recur.
- **As anyone adding a page**, I want the rules enforced consistently, so that "did I forget
  something?" has a definite answer.

## 5. Functional Requirements

**Build-time gates** (fail the build)

- **FR-1.** Every route in the manifest MUST produce an output file with a non-empty `<body>`
  (SEO.1 FR-6).
- **FR-2.** Titles MUST be unique, non-empty, ≤60 characters. Descriptions MUST be unique, non-empty,
  120–160 characters.
- **FR-3.** Every page MUST have a self-referential absolute canonical (SEO.1 FR-5).
- **FR-4.** Sitemap ↔ manifest parity in **both directions**; no `noindex` page in a sitemap
  (SEO.2 FR-10, FR-11).
- **FR-5.** JSON-LD MUST parse, validate against the node schemas, contain no dangling `@id`, and
  every `@id` MUST be absolute (SEO.3 FR-5).
- **FR-6.** Zero orphan pages; maximum depth 3 from `/` (SEO.5 FR-10).
- **FR-7.** Redirect map MUST have no cycles, no chains, and no target that 404s (SEO.5 FR-8).
- **FR-8.** A published URL MUST NOT disappear without a redirect entry (SEO.5 FR-2) — computed by
  diffing `.seo-manifest.json` against the previous build.
- **FR-9.** Every internal link MUST resolve to a manifest route or a redirect entry. Zero broken
  internal links.
- **FR-10.** Every content page MUST meet the SEO.6 extractability threshold (≥8.0 for new pages).
- **FR-11.** Every page MUST have an OG image that is a raster format and resolves (SEO.14 FR-6).
- **FR-12.** Performance budgets MUST hold per route class (SEO.4 FR-2).
- **FR-13.** Every image MUST have `alt` or an explicit decorative marker; axe MUST report zero
  violations on a representative page per template.
- **FR-14.** Front-matter MUST validate, including author-slug existence (SEO.3 FR-20) and unknown-key
  rejection (SEO.6 FR-12).
- **FR-15.** Programmatic pages MUST pass the utility floor or be emitted `noindex` (SEO.10 FR-2), and
  the count of `noindex` pages MUST NOT exceed 5% of the family.

**Warn-level checks** (report, do not fail)

- **FR-16.** Pages with <3 internal links; anchor text on a denylist ("click here", "read more");
  content pages scoring 6.0–7.9; external links that 404 (checked weekly, not per-build, since they
  are outside our control).

**Post-deploy verification**

- **FR-17.** After every production deploy, a smoke job MUST fetch **every** sitemap URL and assert
  HTTP 200 with a non-empty body and a matching canonical. It MUST fail loudly and notify.
- **FR-18.** The smoke job MUST additionally fetch `/robots.txt`, `/sitemap.xml` and every child
  sitemap, `/llms.txt`, and the IndexNow key file, asserting 200 and content-type.
- **FR-19.** The smoke job MUST fetch 10 sampled URLs with `User-Agent: GPTBot`, `OAI-SearchBot`,
  `ClaudeBot` and `PerplexityBot` and assert the response body contains the page's `<h1>` — the direct
  test of the audit's central failure.
- **FR-20.** Weekly, a scheduled job MUST re-run FR-17–FR-19 plus external-link checking and
  structured-data validation against the live site.

**Content lifecycle**

- **FR-21.** Every content page MUST have an `owner` and a `reviewDue` date in front-matter. Missing
  either fails the build for new pages.
- **FR-22.** A **lifecycle report** MUST be generated weekly listing: pages past `reviewDue`, pages
  unverified >180 days (help, SEO.7 FR-13), comparison pages unverified >120 days (SEO.9 FR-11),
  pages with zero impressions after 180 days, pages with declining traffic over 90 days, `noindex`
  pages and why, and orphan/near-orphan pages.
- **FR-23.** The build MUST fail if **>10% of help articles** are stale (SEO.7 FR-13) or **>10% of
  comparison pages** are past re-verification. Staleness at scale is a quality signal, not a backlog.
- **FR-24.** A documented **retire/consolidate** path MUST exist: an underperforming page is improved,
  merged into a stronger page with a 301, or retired with a 410 and sitemap removal — never left
  unmaintained. Decisions are recorded in the lifecycle log.
- **FR-25.** A **quarterly SEO audit** MUST run against a documented checklist covering everything in
  this plan set, with findings tracked to closure.

**Process**

- **FR-26.** The PR template MUST include an SEO checklist for changes under `www/`: manifest entry,
  title/description, canonical, parent/cluster, internal links, OG image, schema, owner, reviewDue.
- **FR-27.** A `CODEOWNERS` entry MUST require web-platform review for `www/src/lib/route-manifest.ts`,
  `www/src/lib/schema/**`, `www/src/lib/redirects.ts`, `www/public/robots.txt` and the deploy workflow
  — the files where a single mistake is site-wide.
- **FR-28.** Any change to the crawler policy, the redirect map, or `noindex` behaviour MUST state its
  rationale in the PR description; the policy file requires a `rationale` field per entry
  (SEO.2 FR-1).

## 6. Non-Functional Requirements

- **Performance** — the full check suite MUST add ≤ 90 s to CI. Expensive checks (axe, Lighthouse,
  link crawl) run on a representative sample per template in PR CI and exhaustively on the weekly job.
- **Security** — the smoke job runs against the public site with no credentials; the notification
  channel must not leak internal URLs. Post-deploy checks must not be a DoS vector against our own
  CDN (rate-limited, sampled above 1,000 URLs).
- **Privacy & Compliance** — no user data touched.
- **Accessibility** — axe in CI (FR-13) is a floor, not a substitute for the manual testing each plan
  specifies; the lifecycle report should track accessibility debt alongside content debt.
- **Scalability** — checks must handle 10k+ pages: parity and link checks operate on
  `.seo-manifest.json` and `.link-graph.json` rather than re-parsing HTML; the smoke job samples above
  1,000 URLs with full coverage weekly.
- **Reliability** — a flaky check is worse than no check. Any check that fails intermittently MUST be
  demoted to warn within one week and fixed or removed.
- **Observability** — every check emits a machine-readable result; the dashboard shows pass/fail
  history per check so flakiness is visible.
- **Maintainability** — one `www/scripts/seo-check/` directory, one file per check, each independently
  runnable via `npm run seo:check -- --only=<check>`.
- **Internationalization** — checks are locale-aware from the start (uniqueness is per-locale, not
  global) so SEO.17 does not require rewriting them.
- **Backward compatibility** — checks are introduced in warn mode and promoted to fail after one clean
  week, so they never block an unrelated urgent fix on day one.

## 7. Acceptance Criteria

- **AC-1.** *Given* a PR adding a route without a manifest entry, *When* CI runs, *Then* it fails
  naming the route and the missing fields.
- **AC-2.** *Given* a PR introducing a duplicate title, *When* CI runs, *Then* it fails naming both
  pages.
- **AC-3.** *Given* a PR that removes a published URL without a redirect, *When* CI runs, *Then* it
  fails naming the URL.
- **AC-4.** *Given* a PR adding an internal link to a nonexistent page, *When* CI runs, *Then* it
  fails naming the source file, line, and target.
- **AC-5.** *Given* a production deploy, *When* the smoke job runs, *Then* every sitemap URL returns
  200 with a matching canonical; a single failure fails the job and notifies.
- **AC-6.** *Given* the smoke job's AI-crawler probe, *When* it fetches 10 URLs as `GPTBot`,
  `OAI-SearchBot`, `ClaudeBot` and `PerplexityBot`, *Then* every response body contains that page's
  `<h1>` text.
- **AC-7.** *Given* invalid JSON-LD (dangling `@id`), *When* CI runs, *Then* it fails naming the page
  and the unresolved reference.
- **AC-8.** *Given* the weekly lifecycle report, *When* generated, *Then* it lists every FR-22
  category with counts and page lists, and is committed/published where the team sees it.
- **AC-9.** *Given* >10% of help articles are stale, *When* CI runs, *Then* the build fails with the
  list.
- **AC-10.** *Given* the full check suite on a 500-page build, *When* timed, *Then* it adds ≤90 s.
- **AC-11.** *Given* a check that fails intermittently, *When* flakiness is detected over 3 runs,
  *Then* it is visible on the dashboard and demoted per the FR reliability rule.

## 8. Data Model

No database changes.

```
www/scripts/seo-check/
  index.mjs                 # runner: --only, --warn-only, --report
  checks/
    manifest-parity.mjs  titles.mjs  canonicals.mjs  sitemap-parity.mjs
    schema-validate.mjs  link-graph.mjs  broken-links.mjs  redirects.mjs
    og-images.mjs  content-score.mjs  utility-floor.mjs  lifecycle.mjs
    perf-budget.mjs  a11y-sample.mjs
www/scripts/seo-smoke/      # post-deploy verification (FR-17–FR-19)
docs/plan/seo/lifecycle-log.md   # retire/consolidate decisions (FR-24)
docs/plan/seo/quarterly-audit/<yyyy-qN>.md
```

Check result artefact `dist/.seo-check.json`:

```jsonc
{ "check": "titles", "status": "fail", "severity": "error",
  "findings": [{ "page": "/k12", "message": "Duplicate title with /higher-ed", "line": null }] }
```

## 9. API Surface

No HTTP surface. Commands:

| Command | Purpose |
|---|---|
| `npm run seo:check` | Run all checks (fail mode) |
| `npm run seo:check -- --warn-only` | Report without failing |
| `npm run seo:check -- --only=titles,canonicals` | Run selected checks |
| `npm run seo:smoke -- --origin=https://lextures.com` | Post-deploy verification |
| `npm run seo:lifecycle` | Generate the lifecycle report |

## 10. UI / UX

- **No public UI.** Internal surfaces:
  - CI output: grouped by check, each finding with page, message, and a link to the rule's docs.
  - PR comment summarising failures and warnings (concise; the full report is an artifact).
  - Lifecycle report published to the internal dashboard (SEO.15) and committed monthly.
- **Flows**
  1. Developer opens a PR → checks run → PR comment shows 2 failures with fixes → fixed → merged.
  2. Deploy completes → smoke job runs → all green → IndexNow submission proceeds.
  3. Monday → lifecycle report → content lead schedules refresh work.
- **States** — a check that cannot run (missing artefact) reports "skipped" with a reason, never a
  false pass.
- **Accessibility** — CI output should be readable in a terminal without colour as the sole signal.
- **Copy & i18n** — internal English only. Failure messages must state the fix, not only the problem.

## 11. AI / ML Considerations

- **FR-19 is the AI-specific check that matters most.** Fetching our own pages as `GPTBot` and
  `OAI-SearchBot` and asserting the `<h1>` is present is the direct, continuous test that the audit's
  central failure has not returned. No amount of internal green builds substitutes for it.
- The `llms.txt` freshness check (FR-18 + SEO.2) ensures the AI-facing corpus does not silently drift
  from the site.
- No models are used in the checks; determinism is the point. If AI-assisted review is added later
  (e.g. flagging vague passages), it must be advisory only and must never gate a build.

## 12. Integration Points

- **External:** GitHub Actions, axe-core, Lighthouse CI, a schema validator, the deployed site itself.
- **Internal modules touched:** `.github/workflows/pages-www.yml` (checks + smoke),
  `.github/workflows/lighthouse.yml` (wired into the www gate), `www/package.json` (scripts),
  `www/scripts/seo-check/*`, `www/scripts/seo-smoke/*`, `.github/PULL_REQUEST_TEMPLATE.md`,
  `CODEOWNERS`.
- **Events:** smoke-job failure → notification; lifecycle report → dashboard.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md) (manifest, `.seo-manifest.json`),
  [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md) (sitemaps), [SEO.3](SEO.3-structured-data-and-entity-graph.md)
  (schema to validate), [SEO.5](SEO.5-information-architecture-and-internal-linking.md)
  (`.link-graph.json`), [SEO.6](SEO.6-answer-first-content-system.md) (`.content-quality.json`).
- **Must ship before:** the *scaling* phase of SEO.7, SEO.8, SEO.9, SEO.10, SEO.11. Those plans can
  start, but must not reach volume without the gates.
- **Shared infra:** CI minutes; a notification channel for smoke failures.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Checks are too strict and block urgent work | **H** | M | Warn-mode introduction, one clean week before promotion, documented `--warn-only` escape hatch requiring a recorded reason |
| Flaky checks erode trust and get disabled wholesale | M | **H** | FR reliability rule: demote within a week; flakiness visible on the dashboard (AC-11) |
| Suite slows CI enough that people avoid it | M | M | ≤90 s budget (AC-10); sampling in PR CI, exhaustive weekly |
| Smoke job hammers our own CDN | L | M | Rate limiting + sampling above 1,000 URLs |
| Lifecycle report is generated and ignored | **H** | M | FR-23 makes staleness a build failure past a threshold, so it cannot be ignored indefinitely; monthly review slot |
| Checks encode today's assumptions and block a legitimate future change | M | L | Every rule documented with its rationale; changing a rule is a normal PR |
| False sense of safety — green CI, bad content | M | M | Explicit non-goal: gates are structural; editorial review remains mandatory in SEO.6/SEO.8 |

## 15. Rollout Plan

- **Feature flag:** `--warn-only` mode is the rollout mechanism.
- **Sequencing**
  1. Build the runner + the five highest-value checks (manifest parity, titles/descriptions,
     canonicals, sitemap parity, broken internal links) in **warn** mode.
  2. Fix everything they surface on `main`.
  3. Promote those five to **fail**.
  4. Add schema validation, link-graph/orphan, redirects, OG images; warn → fail on the same pattern.
  5. Ship the post-deploy smoke job including the AI-crawler probe (FR-19) — this one starts in fail
     mode, because it is the check that would have caught audit F-1.
  6. Add content-score, utility-floor, perf-budget and a11y checks as their source plans land.
  7. Ship the lifecycle report; enable FR-23 thresholds after one full refresh cycle.
  8. First quarterly audit at month 3.
- **Dogfood:** run the suite against the pre-SEO.1 site to confirm it reproduces every audit finding —
  a check suite that would not have caught the known failures is not finished.
- **GA criteria:** AC-1…AC-11; three consecutive weeks with zero flaky-check demotions.
- **Rollback:** any check can be demoted to warn with a one-line config change plus a recorded reason.

## 16. Test Plan

- **Unit** — each check against fixture sites with deliberately injected faults (duplicate title,
  missing canonical, orphan page, dangling `@id`, removed URL without redirect, broken internal link,
  SVG OG image, stale help article). Each must produce exactly one finding with the correct page and
  message.
- **Integration** — full suite on a golden fixture site; assert runtime ≤90 s at 500 pages (AC-10);
  assert `--warn-only` never exits non-zero.
- **End-to-end** — smoke job against a staging deploy including a deliberately broken URL; assert
  failure and notification. AI-crawler probe against staging (AC-6).
- **Security** — no credentials required by the smoke job; verify rate limiting; verify notifications
  contain no secrets.
- **Accessibility** — the axe sampling check itself is verified against a page with a known violation.
- **Performance / load** — CI duration tracked over time; smoke job request rate asserted.
- **Manual exploratory** — the "would it have caught it?" exercise: run the finished suite against the
  repository state at commit `0e68a462` and confirm it reports F-1, F-3, F-4, F-6, F-8, F-10 and F-19.

## 17. Documentation & Training

- `www/docs/seo-checks.md` — every check: what it asserts, why (linking the plan and the research),
  how to fix a failure, and how to request an exception.
- `www/docs/content-lifecycle.md` — owner, reviewDue, refresh, consolidate, retire; the decision tree
  and where decisions are recorded.
- `docs/plan/seo/quarterly-audit/_TEMPLATE.md` — the audit checklist.
- PR template update (FR-26) and CODEOWNERS (FR-27).
- Onboarding: new contributors run `npm run seo:check` locally once before their first `www` PR.

## 18. Open Questions

1. Where does the smoke-job failure notification go — Slack, email, GitHub issue? Who is on call for
   it?
2. Do we gate deploys on the smoke job (deploy → verify → roll back on failure), or notify only?
   (Recommendation: notify at first, gate once the job is proven stable.)
3. Who owns the quarterly audit and the lifecycle report review?
4. Should the content-score check block or warn for pages authored by external contributors?
5. Do we adopt a commercial crawler (Screaming Frog / Sitebulb) for the weekly job, or keep everything
   in-repo?

## 19. References

- Existing files: `.github/workflows/pages-www.yml`, `.github/workflows/lighthouse.yml`,
  `www/package.json` (test script), `www/scripts/prerender-courses.test.mjs`,
  `www/src/lib/document-head.test.mjs`
- Audit findings: [F-22](audit.md#f-22-no-seo-regression-gate-in-ci), and the full
  [severity → plan map](audit.md#severity--plan-map) (this plan's checks must reproduce it)
- Research: [§2](research.md#2-ai-crawlers-do-not-run-javascript),
  [§6](research.md#6-page-experience-the-bar-moved-up-in-march-2026)
- External: [Google — Search Essentials](https://developers.google.com/search/docs/essentials),
  [axe-core](https://github.com/dequelabs/axe-core),
  [Lighthouse CI](https://github.com/GoogleChrome/lighthouse-ci)
- Related plans: [SEO.1](SEO.1-static-rendering-and-crawlability.md),
  [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md),
  [SEO.5](SEO.5-information-architecture-and-internal-linking.md),
  [SEO.6](SEO.6-answer-first-content-system.md),
  [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md),
  [UX.18 — design-system governance](../ui-ux/UX.18-design-system-governance-and-measurement.md)
