# CC.6 — Rule Pack D: Accessibility, Inclusive Design & Launch Readiness

> Implementation plan. Source: Course Checklist product request. Folder overview: [README](README.md).
> Rubric mapping: [course-design-research.md](course-design-research.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CC.6 |
| **Section** | Course Checklist |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Server / platform team + accessibility |
| **Depends on** | CC.1, CC.2 |
| **Unblocks** | CC.7, CC.10 |

---

## 1. Problem Statement

Accessibility is the one course-quality area with legal teeth — ADA Title II/III, Section 508, EN 301 549
and the EAA all attach to the content an instructor authors, and QM General Standard 8, OSCQR standards
15/17–28/34–36 and NSQ Standard E all test it. Lextures already has the enforcement primitives (alt-text
enforcement, caption requirements, a11y flags, the accessibility settings surface) but no course-level
"here is what in *your* course is still inaccessible" view. The same gap exists for go-live: nothing tells
an instructor that they have never previewed the course as a student, that three external links are dead, or
that no backup export exists. CC.6 ships the 20 rules that cover authored-content accessibility, UDL
breadth, and launch readiness — the last of the four rule packs.

## 2. Goals

- Ship the **20 accessibility and readiness rules** in §5, each citing a WCAG success criterion or rubric
  standard, with evidence naming the exact page/image/video to fix.
- Reuse the existing accessibility primitives (`alt_text_enforcement`, `require_captions`, `a11y_flags`)
  rather than building a second scanner.
- Introduce the **only lazy loader** in the checklist — external link health — with a strict budget and
  cached results, so it never slows the page.
- Give the checklist a credible **"ready to launch"** closing section: student-view preview, calendar sanity,
  no drafts after start, backup taken.

## 3. Non-Goals

- No changes to the accessibility enforcement engine, the alt-text policy or the captions pipeline.
- No automated remediation (no auto-generated alt text) — CC.10 links to the existing AI alt-text affordance
  where one exists.
- No full WCAG audit. The checklist covers what is **machine-checkable in authored content**; it explicitly
  does not claim conformance, and its copy says so.
- No accessibility scanning of uploaded PDFs' internal structure beyond a scanned-image heuristic.
- No VPAT/ACR generation — that lives in `docs/vpat/`.

## 4. Personas & User Stories

- **As an instructor**, I want a list of exactly which images lack alt text, so that I can fix them in ten
  minutes instead of never.
- **As a disabled student** (indirectly), I want captions and heading structure to be checked before I have
  to ask, so that I am not the accessibility QA process.
- **As an accessibility coordinator**, I want per-course visibility of authored-content gaps, so that
  remediation is targeted rather than institution-wide guesswork.
- **As an instructor the day before term**, I want to know that I have never viewed my own course as a
  student, so that I catch what they will see.
- **As an instructor**, I want dead links found before students hit them, so that week 3 is not a support
  ticket.
- **As a compliance lead**, I want the checklist to be honest that it is a helper, not a conformance
  guarantee, so that we do not create a false assurance.

## 5. Functional Requirements

### D1 — Authored-content accessibility

- **FR-1.** `a11y.image-alt-text` (**essential**; WCAG 1.1.1, QM 8.2, OSCQR 36) — DONE when every image in
  content pages, syllabus sections, assignment and quiz bodies has non-empty alt text or is explicitly marked
  decorative. Evidence columns `["Page", "Image", "Location"]`; each row targets that page with the image
  focused. Uses the same markdown/HTML parse the alt-text enforcement engine uses (`210_alt_text_enforcement`).
- **FR-2.** `a11y.video-captions` (**essential**; WCAG 1.2.2, QM 8.3, OSCQR 35) — DONE when every embedded
  video/audio has captions or a transcript, or the course sets `require_captions`. Evidence lists uncaptioned
  media with the page it appears on. `not_applicable` when the course embeds no time-based media.
- **FR-3.** `a11y.heading-structure` (**essential**; WCAG 1.3.1/2.4.6, OSCQR 21) — DONE when no content page
  skips a heading level, starts below H2, or uses bold-as-heading. Evidence lists pages with the first
  offending heading.
- **FR-4.** `a11y.link-text` (**recommended**; WCAG 2.4.4, OSCQR 37) — DONE when no link text is
  "click here", "read more", "link", or a bare URL longer than 40 characters. Evidence lists offenders.
- **FR-5.** `a11y.table-headers` (**recommended**; WCAG 1.3.1, OSCQR 25/26) — DONE when every data table in
  authored content has a header row or column. Evidence lists tables without.
- **FR-6.** `a11y.tables-for-layout` (**recommended**; OSCQR 24) — DONE when no table is used purely for
  layout (single-row/single-column tables with no headers).
- **FR-7.** `a11y.color-contrast` (**recommended**; WCAG 1.4.3, OSCQR 18) — applies only when
  `markdown_theme_custom` is set. DONE when the custom theme's foreground/background pairs meet 4.5:1
  (3:1 for large text). `Detail` names the failing pair and its ratio.
- **FR-8.** `a11y.text-formatting` (**recommended**; OSCQR 22/23) — DONE when authored content contains no
  blinking/marquee constructs, no all-caps blocks longer than 80 characters, and no font-size overrides
  below the platform minimum.
- **FR-9.** `a11y.document-accessibility` (**recommended**; WCAG 1.1.1, OSCQR 34) — DONE when no uploaded
  PDF referenced by a module is image-only (no extractable text layer). Evidence lists them with a
  "replace or add a text alternative" hint. Heuristic: text-layer presence, page count, file size.
- **FR-10.** `a11y.media-alternatives` (**recommended**; WCAG 1.2.x, UDL Representation) — DONE when every
  module whose primary content is video also offers a text alternative (transcript, notes page, or a text
  content page in the same module).
- **FR-11.** `a11y.enforcement-settings` (**recommended**) — DONE when the course's accessibility settings
  have been reviewed (alt-text enforcement / caption requirement explicitly chosen). Uses the CC.5 §8
  review-marker pattern with a new `a11y_reviewed_at` column.

### D2 — Inclusive design (UDL)

- **FR-12.** `udl.multiple-representations` (**recommended**; UDL Representation, QM 4.5, OSCQR 29) — DONE
  when ≥ 60% of modules offer content in ≥ 2 modalities (text + media, text + interactive, etc.). `Detail`
  reports the observed ratio. Complements CC.4 FR-8 (course-wide variety) at module granularity.
- **FR-13.** `udl.expression-choice` (**recommended**; UDL Action & Expression, QM 3.4) — DONE when ≥ 1
  assessment in the course allows a choice of submission type (multiple accepted submission types, or an
  explicit choice-of-format instruction). One is enough — this is a nudge, not a mandate.
- **FR-14.** `udl.engagement-relevance` (**recommended**; UDL Engagement, QM 4.x) — DONE when ≥ 1 authentic /
  applied activity exists (project, case study, portfolio, capstone, or an item flagged as authentic by the
  authoring UI). Detection is lexicon-based over item titles and instructions.
- **FR-15.** `a11y.plain-language` (**recommended**; QM 8.x usability) — DONE when no content page's
  readability estimate exceeds the course's grade band by more than 3 levels. Uses a deterministic
  Flesch–Kincaid computation (no AI). `not_applicable` when the course declares no grade band, and for
  locales without a validated readability formula.

### D3 — Link & reference health (lazy)

- **FR-16.** `links.external-health` (**recommended**; OSCQR 37) — DONE when every distinct external URL in
  authored content resolves (HTTP < 400) on the most recent check. This is the checklist's **only lazy
  loader** (CC.1 FR-8): it MUST run at most once per course per 24 h, MUST be bounded to 200 distinct URLs,
  MUST use a 5 s total budget with 2 s per request and 8-way concurrency, MUST respect `robots.txt` and send
  a descriptive User-Agent, and MUST report `unknown` rather than blocking when the budget is exceeded.
  Results are cached in `course.course_checklist_link_health`. Evidence lists dead links with status code
  and the page they appear on.

### D4 — Launch readiness

- **FR-17.** `launch.student-preview` (**essential**; OSCQR 16, QM 8.x) — DONE when a staff member has used
  "View as: Student" for this course since the last structural change. Uses the existing course-view-as
  mechanism plus a new `student_preview_at` marker column.
- **FR-18.** `launch.no-drafts-after-start` (**essential**; OSCQR 7) — DONE when, once `starts_at` has
  passed, no gradable item due within the next 14 days is unpublished. Evidence lists them. Distinct from
  CC.4 FR-5, which is about publication inside published modules; this is time-relative and urgent.
- **FR-19.** `launch.calendar-sanity` (**recommended**) — DONE when the course calendar has no due date on a
  configured non-instructional day (institutional holiday / blackout date), where the org defines any.
  `not_applicable` when the org publishes no academic calendar.
- **FR-20.** `launch.backup-export` (**recommended**) — DONE when a course export exists that is newer than
  the last structural change, or the course is younger than 7 days. Target: `/settings/import-export`.

### Cross-cutting

- **FR-21.** All 20 rules land at `Tier: recommended`; the five marked **essential** are promoted per §15.
- **FR-22.** Accessibility rules MUST parse authored content **once** per evaluation into a shared
  document model (`ContentDoc`) reused by FR-1 through FR-10 and FR-15 — no rule re-parses.
- **FR-23.** Every accessibility item's copy MUST state that the check is automated and partial, and link to
  the manual-testing guidance in `docs/accessibility/`.

## 6. Non-Functional Requirements

- **Performance** — The shared parse (FR-22) MUST be O(total authored bytes) with a 4 MB course cap; the
  whole pack MUST add < 120 ms p95 excluding the lazy loader. `links.external-health` runs outside the
  request budget: on a cache miss the item returns `unknown` with "checking…" and a background job populates
  it for the next read.
- **Security** — The link checker is an **outbound fetcher driven by user content**: it MUST block private
  and link-local address ranges (SSRF defence), MUST NOT follow redirects to blocked ranges, MUST cap
  response reads at 64 KB (HEAD preferred, GET fallback), MUST NOT send cookies or auth headers, and MUST
  NOT execute or store fetched content. It MUST be rate-limited per host.
- **Privacy & Compliance** — The link checker discloses to third-party hosts that a Lextures course links to
  them; the User-Agent MUST identify the platform and link to a documentation page explaining the crawl.
  No learner data leaves the platform. Accessibility findings are course content, not personal data.
  **Legal framing:** the checklist MUST NOT be described as producing WCAG conformance; §17 documents this.
- **Accessibility** — This pack is the checklist's own accessibility contribution; its evidence tables must
  themselves be accessible (CC.7), and its copy must avoid ableist framing ("fix these for your students",
  not "accessibility violations").
- **Scalability** — Distinct-URL extraction deduplicates across the course; link-health rows are per
  `(course_id, url_hash)` and swept after 30 days.
- **Reliability** — Link checker failures degrade to `unknown`, never to `todo` (never claim a link is dead
  because our network hiccuped). PDF heuristics that cannot open a file yield `unknown` for that row.
- **Observability** — `coursechecklist_linkcheck_duration_seconds`, `..._urls_total{result=ok|dead|error|
  skipped}`, `..._blocked_total{reason=private_range|robots|rate_limit}`. Alert on blocked-ratio spikes.
- **Maintainability** — `rules_a11y.go`, `rules_udl.go`, `rules_launch.go`, `linkhealth/` sub-package for the
  fetcher (isolated so its security properties are testable alone).
- **Internationalization** — FR-15 readability applies only to locales with a validated formula (en initially);
  FR-14's lexicon is locale-keyed; FR-4's link-text lexicon is locale-keyed.
- **Backward compatibility** — Two additive marker columns and one new table; no API or behaviour change to
  existing accessibility enforcement.

## 7. Acceptance Criteria

- **AC-1.** *Given* a content page with three images, one lacking alt text, *Then* `a11y.image-alt-text` is
  `in_progress` with `progress = {2, 3}` and one evidence row naming the page and image.
- **AC-2.** *Given* an image marked decorative, *Then* it does not appear as evidence.
- **AC-3.** *Given* a page whose headings go H2 → H4, *Then* `a11y.heading-structure` lists it with the
  offending heading.
- **AC-4.** *Given* a course embedding no video, *Then* `a11y.video-captions` is `not_applicable`.
- **AC-5.** *Given* a custom markdown theme with a 3.1:1 body contrast, *Then* `a11y.color-contrast` is
  `todo` and `Detail` states "3.1:1 (needs 4.5:1)".
- **AC-6.** *Given* a link to `http://169.254.169.254/latest/meta-data/`, *When* the link checker runs,
  *Then* the request is **not** issued, the URL is recorded as `skipped`, and
  `..._blocked_total{reason="private_range"}` increments.
- **AC-7.** *Given* a redirect chain ending at `127.0.0.1`, *Then* the fetch is aborted and recorded as
  blocked.
- **AC-8.** *Given* 300 distinct URLs, *Then* only 200 are checked and `Detail` states the cap.
- **AC-9.** *Given* a cold link-health cache, *Then* `links.external-health` returns `unknown` within the
  normal request budget and populates on the next read.
- **AC-10.** *Given* a course whose staff has never used "View as: Student", *Then* `launch.student-preview`
  is `todo`; *Given* a preview after the last structural change, *Then* `done`.
- **AC-11.** *Given* a started course with an unpublished quiz due in 5 days, *Then*
  `launch.no-drafts-after-start` is `todo` listing it.
- **AC-12.** *Given* an image-only PDF, *Then* `a11y.document-accessibility` lists it; *Given* a text-layer
  PDF, *Then* it does not.
- **AC-13.** *Given* the full pack, *Then* the shared `ContentDoc` parse runs exactly once per evaluation
  (asserted by a parse-counter test).
- **AC-14.** *Given* any accessibility item, *Then* its `why` copy contains the automated-and-partial
  disclaimer (asserted by a copy test).

## 8. Data Model

`server/migrations/464_course_checklist_link_health.sql`:

```sql
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS a11y_reviewed_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS student_preview_at TIMESTAMPTZ;

CREATE TABLE course.course_checklist_link_health (
    course_id    UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    url_hash     BYTEA NOT NULL,              -- sha256 of the normalized URL
    url          TEXT NOT NULL,
    status_code  INT,                          -- NULL when not fetched
    result       TEXT NOT NULL CHECK (result IN ('ok','dead','error','skipped')),
    reason       TEXT NOT NULL DEFAULT '',
    checked_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (course_id, url_hash)
);
CREATE INDEX idx_checklist_link_health_checked ON course.course_checklist_link_health (checked_at);

COMMENT ON TABLE course.course_checklist_link_health IS
    'Cached outbound link-health results for checklist item links.external-health (30-day retention).';
```

- **Backfill**: none. Absent rows ⇒ `unknown` until the first background check.
- **Writers**: `student_preview_at` stamped by the course-view-as toggle handler; `a11y_reviewed_at` by the
  accessibility settings save.
- **Retention**: nightly sweeper deletes link-health rows older than 30 days and rows whose URL no longer
  appears in the course.

## 9. API Surface

No new public routes. One internal job:

- Background worker `checklist-linkcheck` (in `server/internal/workers`), triggered when
  `links.external-health` is read with a stale/empty cache. Enqueues at most one job per course per 24 h via
  the existing queue infrastructure. Job payload: `{courseId}`. No public trigger endpoint — `POST
  /checklist/refresh` (CC.2) may enqueue it, subject to the same 24 h floor.

The course-view-as toggle handler and accessibility settings handler gain marker side effects (response
shapes unchanged).

## 10. UI / UX

No new pages. Two UI contracts CC.7 must honour:

1. **`unknown` needs a real state.** `links.external-health` will legitimately be `unknown` on first view;
   CC.7 renders "Checking links…" with a muted style and no action, not an error.
2. **Accessibility copy is help, not blame.** Titles are imperative and specific ("Add alt text to 4
   images"), `why` carries the automated-and-partial disclaimer, and every accessibility item links to
   `docs/accessibility/` guidance for the manual checks the platform cannot do.

## 11. AI / ML Considerations

No AI in the evaluators — readability (FR-15) is Flesch–Kincaid, contrast (FR-7) is a WCAG formula, PDF
detection (FR-9) is a text-layer probe. Where the platform **already** has an AI affordance for remediation
(alt-text suggestion, caption generation via `auto_captioning_enabled`), CC.10 wires the checklist item's
action to it with human review before anything is written. CC.6 introduces no model, prompt or token budget.

## 12. Integration Points

- Internal: `server/internal/service/coursechecklist` (+ `linkhealth/` sub-package),
  the alt-text enforcement code from `210_alt_text_enforcement`, `course.courses.require_captions` and
  `a11y_flags`, `server/internal/repos/coursefiles` (PDF probing), `server/internal/repos/coursesyllabus`,
  `server/internal/workers`, `server/internal/telemetry`, markdown parsing shared with the content pipeline.
- Nav targets: content page editors, `/syllabus`, `/settings/accessibility`, `/files`,
  `/settings/import-export`, `/calendar`, course-view-as toggle.
- External: outbound HTTP to arbitrary hosts (FR-16) — the only external dependency in the whole checklist,
  with the §6 security constraints.

## 13. Dependencies & Sequencing

- Must ship after: CC.1, CC.2. Best after CC.4 (shares the content-parse pass).
- Must ship before: CC.10, CC.7 GA.
- Migration `464` before the marker rules and link health evaluate.
- The `linkhealth` fetcher MUST pass a security review before the worker is enabled; the other 19 rules can
  ship without it.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Link checker is used as an SSRF pivot | M | **H** | Private/link-local range blocklist pre- and post-redirect, no cookies/auth, 64 KB read cap, per-host rate limit, isolated sub-package with dedicated security tests (AC-6, AC-7); security review gate before enabling |
| Outbound crawling gets Lextures IPs blocked or looks abusive | M | M | 24 h per-course floor, 200-URL cap, 8-way concurrency, robots.txt respect, identifying User-Agent with a docs link |
| Checklist implies WCAG conformance and creates legal exposure | M | **H** | FR-23 mandatory disclaimer in copy, asserted by test (AC-14); §17 doc; no "accessible ✓" language anywhere |
| Alt-text rule fires on hundreds of images in imported courses | **H** | M | `progress` framing, evidence capped at 200, `recommended` tier, links to the bulk alt-text affordance |
| Readability rule is wrong for technical content | M | M | 3-level tolerance, `recommended` only, `not_applicable` without a declared grade band |
| Content parsing doubles evaluation cost | M | M | FR-22 single shared parse, asserted (AC-13); 4 MB cap |

## 15. Rollout Plan

**No feature flag** for the checklist itself. The link checker is the one component with an operational
kill switch, because it makes outbound network calls:

1. Migration `464`; ship the 19 non-network rules `recommended`.
2. Ship `links.external-health` **disabled at the worker level** (env `CHECKLIST_LINKCHECK_ENABLED=false`,
   item reports `unknown`) until the security review passes; then enable in staging, then production.
3. Dogfood 2 weeks. Accessibility rules are hand-verified against the existing `docs/accessibility/` manual
   test scripts on at least five real courses.
4. Promotion gate for the five `essential` rules: `disagree` dismissal < 10%, accessibility-team sign-off on
   copy, and legal sign-off on the disclaimer language.
5. Rollback: `RETIRED_ITEM_IDS` / tier demotion for rules; `CHECKLIST_LINKCHECK_ENABLED=false` for the
   fetcher, which needs no deploy.

## 16. Test Plan

- **Unit** — Per-rule table tests. Contrast math against the WCAG reference vectors. Heading-sequence
  matrix. Link-text lexicon. Flesch–Kincaid against published reference passages. Decorative-image handling.
- **Integration** — Shared-parse counter (AC-13); PDF probe against fixture files (text-layer, image-only,
  encrypted, corrupt); marker columns written by their handlers.
- **End-to-end** — Playwright: upload an image with no alt text, assert the item and evidence row; add alt
  text, recheck, assert `done`. Toggle "View as: Student", assert `launch.student-preview` flips.
- **Security** — Dedicated `linkhealth` suite: private ranges (10/8, 172.16/12, 192.168/16, 127/8,
  169.254/16, ::1, fc00::/7), DNS-rebinding via redirect, oversized response, slowloris timeout,
  cookie/header leakage, robots.txt honouring, per-host rate limit. These are blocking tests.
- **Accessibility** — The pack's own copy reviewed by the accessibility owner; disclaimer test (AC-14);
  evidence tables verified in CC.7's axe run.
- **Performance / load** — Benchmark: < 120 ms p95 excluding the loader; parse cost on a 4 MB course;
  link-check job wall-clock on 200 URLs.
- **Manual exploratory** — Run against a course imported from Canvas (worst-case alt-text/heading debt) and
  a media-heavy course; hand-verify a sample of 50 findings.

## 17. Documentation & Training

- `docs/accessibility/course-checklist-scope.md` — **what the automated checks do and do not cover**, in
  plain language, with an explicit statement that passing the checklist is not WCAG conformance and does not
  substitute for manual testing or an ACR/VPAT. Linked from every accessibility item.
- `docs/dev/checklist-linkhealth.md` — fetcher design, SSRF defences, tuning env vars, how to disable.
- Help-centre: "Adding alt text", "Captions and transcripts", "Heading structure", "Previewing as a student".
- Runbook: disable the link checker; interpret `blocked_total` spikes; purge link-health cache for a course.

## 18. Open Questions

1. Should `links.external-health` ship at all in v1, given it is the only outbound-network component in the
   checklist? Proposed: yes, but last and behind the security gate. It is the highest-value OSCQR-37 check
   and instructors ask for it constantly.
2. Does the institutional academic calendar (FR-19) exist as data today, or does this rule need an org-level
   blackout-dates feature first? If the latter, FR-19 ships `not_applicable` everywhere and is a stub.
3. Should `a11y.plain-language` exist given the risk of discipline-inappropriate advice? Proposed: keep,
   `recommended`, generous tolerance, English-only initially.
4. Should the accessibility rules roll up into a single "Accessibility" score for reporting to a coordinator?
   Deferred to CC.10 / section 12 (Accessibility) plans.
5. Do we need per-org suppression of the accessibility rules where a central team owns remediation?
   Relates to CC.2 §18 Q3 (org policy table).

## 19. References

- Existing files this work touches: `server/internal/service/coursechecklist/rules_{a11y,udl,launch}.go` and
  `linkhealth/` (new), `server/migrations/210_alt_text_enforcement.sql`,
  `server/internal/repos/course/require_captions.go`, `server/internal/repos/course/markdown_theme.go`,
  `server/internal/workers/`, `docs/accessibility/`, `docs/vpat/`.
- External standards: WCAG 2.1 AA (1.1.1, 1.2.2, 1.3.1, 1.4.3, 2.4.4, 2.4.6, 3.1.1);
  [Quality Matters](https://www.qualitymatters.org/qa-resources/rubric-standards/higher-ed-rubric) GS8
  (Accessibility and Usability); [OSCQR](https://oscqr.suny.edu/) "Design and Layout" (16–28) and
  "Content and Activities" (34–37); [NSQ](https://nsqol.org/the-standards/quality-online-courses/)
  Standard E; [CAST UDL Guidelines 3.0](https://udlguidelines.cast.org/).
- Related plans: [CC.4](CC.4-rule-pack-structure-outcomes-alignment.md),
  [CC.5](CC.5-rule-pack-assessment-feedback-interaction.md), [CC.7](CC.7-web-checklist-page-and-nav-badge.md),
  [CC.10](CC.10-analytics-guidance-and-rollout.md); accessibility plans in
  [`docs/completed/12-accessibility/`](../../completed/12-accessibility/) and the accessibility-law work in
  [`docs/plan/standards/`](../standards/).
