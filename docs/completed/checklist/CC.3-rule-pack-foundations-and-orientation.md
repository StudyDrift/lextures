# CC.3 — Rule Pack A: Foundations, Orientation, Policies & People

> Implementation plan. Source: Course Checklist product request. Folder overview: [README](README.md).
> This is the first of four rule packs; the catalog rationale and rubric mapping live in
> [research.md](../../help/course-checklist/research.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CC.3 |
| **Section** | Course Checklist |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Server / platform team + instructional design |
| **Depends on** | CC.1, CC.2 |
| **Unblocks** | CC.7 (needs real items to render), CC.10 |

---

## 1. Problem Statement

The single biggest predictor of a bad first week is a course that opens with no dates, no welcome, no
syllabus and nobody enrolled. Every external quality rubric puts this first — Quality Matters General
Standard 1 ("Course Overview and Introduction"), OSCQR standards 1–10 ("Overview and Information") and NSQ
Standard A ("Course Overview and Support") all say the same thing: orient the learner before you teach
them. Lextures stores every one of these facts already (`course.courses` dates and timezone,
`course.course_syllabus`, `course.feed_channels`/`feed_messages`, `course.course_enrollments`) but never
tells the instructor which are still empty. CC.3 turns those stored facts into 33 concrete, evidence-backed
checklist rules covering course identity, learner orientation, published policies and people.

## 2. Goals

- Ship the **33 foundation rules** listed in §5 (11 identity · 10 orientation · 7 syllabus/policy · 5
  people), each with a rubric citation, a nav target and (where useful) an evidence table.
- Make the four rules the product request called out by name — **course dates**, **course syllabus**,
  **course features**, **welcome message**, **add enrollments** — land as first-class, well-worded items.
- Keep every rule **cheap**: all 33 evaluate purely against the CC.1 snapshot, no lazy loaders.
- Get the **false-positive rate near zero** on the three seeded demo courses before promoting any rule to
  `essential`.

## 3. Non-Goals

- No structure/outcome rules (CC.4), no assessment/interaction rules (CC.5), no accessibility or
  launch-readiness rules (CC.6).
- No UI. Rules are data + evaluators consumed by CC.7.
- No auto-fix. "Post a welcome message for me" is CC.10.
- No new authoring surfaces. If a rule has nowhere to link, it is deferred rather than shipped pointing at
  a page that does not exist.

## 4. Personas & User Stories

- **As a first-time online instructor**, I want to be told that a welcome message and a "Start Here" module
  matter, so that my students are not dropped into week 1 with no orientation.
- **As a teacher two days before term**, I want a fast read on whether dates, publication and enrollments
  are actually set, so that nothing silently blocks students on day one.
- **As an instructional designer**, I want each rule to cite the standard it comes from, so that the
  checklist is defensible in a course review.
- **As a homeschool parent**, I want rules about co-teachers and sections to disappear, so that my one-child
  course does not look 60% incomplete.
- **As a department chair**, I want "response-time expectations published" to be a checklist item, so that
  the commitment is explicit rather than folklore.

## 5. Functional Requirements

Each rule below MUST be registered per CC.1 FR-2 with the given ID, tier, sources, evaluator semantics and
nav target. `essential` items drive the nav badge; `recommended` items do not.

### A1 — Course identity & configuration

- **FR-1.** `course.title-and-description` (**essential**; QM 1.1, OSCQR 2) — DONE when `title` is non-empty
  and not a create-template placeholder, **and** `description` is ≥ 120 characters. Target:
  `/courses/{code}/settings/general#course.general.description`.
- **FR-2.** `course.dates` (**essential**; QM 1.2, OSCQR 7) — DONE when `starts_at` and `ends_at` are both
  set and `ends_at > starts_at`. `todo` with detail `"End date is before start date"` when inverted.
  `not_applicable` when `schedule_mode = 'relative'` (self-paced) — replaced by FR-5.
- **FR-3.** `course.timezone` (**essential**; OSCQR 7) — DONE when `course_timezone` is a valid IANA zone.
  Detail names the zone that due dates will be interpreted in.
- **FR-4.** `course.published` (**essential**; OSCQR 7) — DONE when `published = true`. `in_progress` with
  urgent detail when `published = false` **and** `starts_at` is within 7 days or already past.
- **FR-5.** `course.relative-schedule` (**essential**, applies only when `schedule_mode = 'relative'`;
  OSCQR 7) — DONE when a relative offset exists for every gradable item; evidence lists items with no
  offset. Uses `server/internal/relativeschedule`.
- **FR-6.** `course.visibility-window` (**recommended**; OSCQR 7) — DONE when `visible_from`/`hidden_at` are
  either both null or form a window containing `[starts_at, ends_at]`. Flags a window that hides the course
  during term.
- **FR-7.** `course.grading-scheme` (**essential**; QM 3.2, OSCQR 44) — DONE when `grading_scheme_id` is set
  or `grading_scale` is not the platform default-and-untouched value.
- **FR-8.** `course.hero-image` (**recommended**; QM 8.x usability) — DONE when `hero_image_url` is set.
  Detail nudges toward alt text (checked in CC.6 `a11y.image-alt-text`).
- **FR-9.** `course.home-landing` (**recommended**; OSCQR 16) — DONE when `course_home_landing` has been
  explicitly chosen (any value other than the untouched default) — i.e. the instructor decided what students
  see first.
- **FR-10.** `course.features-reviewed` (**essential**; product) — DONE when the course features page has
  been saved at least once for this course. Because no such marker exists today, CC.3 MUST add a
  `features_reviewed_at TIMESTAMPTZ` column to `course.courses` written by
  `handlePatchCourseFeatures`. `in_progress` shows which high-impact features are off (discussions, files,
  calendar) as evidence rows so the instructor can decide deliberately rather than by default.
- **FR-11.** `course.language` (**recommended**; WCAG 3.1.1, OSCQR 34) — DONE when the course's primary
  locale is set and matches the language of ≥ 80% of syllabus/content-page text (heuristic, no AI: script +
  stop-word detection via `internal/l10n`).

### A2 — Learner orientation

- **FR-12.** `orientation.welcome-message` (**essential**; QM 1.1, OSCQR 1, NSQ A) — DONE when the
  `#announcements` channel has ≥ 1 message authored by course staff with ≥ 200 characters posted on or after
  `created_at` of the current term. Target: `/courses/{code}/feed?channel=announcements`. Detail when
  `todo`: "Students see an empty announcements channel on day one."
- **FR-13.** `orientation.start-here` (**essential**; QM 1.1, OSCQR 1) — DONE when the first module (lowest
  `sort_order`) contains a content page whose title matches a start-here heuristic (`start here`,
  `orientation`, `welcome`, `getting started`, `how this course works`, plus locale variants) **or** the
  course home landing is a content page. Evidence lists module 1's items.
- **FR-14.** `orientation.instructor-contact` (**essential**; QM 1.7/1.8, OSCQR 10) — DONE when the
  syllabus contains a section whose heading or body matches contact patterns (email/phone/office/contact) and
  the body is ≥ 80 characters.
- **FR-15.** `orientation.response-time` (**essential**; QM 1.4, OSCQR 38) — DONE when syllabus text states a
  turnaround commitment (regex over "within N hours/days", "respond", "turnaround", "grade … within").
- **FR-16.** `orientation.participation-expectations` (**recommended**; QM 1.4, OSCQR 43) — DONE when the
  syllabus or a start-here page states participation/frequency expectations.
- **FR-17.** `orientation.netiquette` (**recommended**; QM 1.3, OSCQR 43) — DONE when a netiquette /
  community-agreement / code-of-conduct section exists; `not_applicable` when discussions, feed, groups and
  boards are all disabled.
- **FR-18.** `orientation.tech-requirements` (**recommended**; QM 1.5/1.6, OSCQR 11–13) — DONE when the
  syllabus states required technology and/or prerequisite technical skills.
- **FR-19.** `orientation.support-resources` (**recommended**; QM 7.1–7.4, OSCQR 6) — DONE when the syllabus
  contains ≥ 2 outbound support links (help desk, accessibility office, tutoring, library, counselling).
  Evidence lists detected links.
- **FR-20.** `orientation.instructor-introduction` (**recommended**; QM 1.8, OSCQR 40) — DONE when a
  staff-authored self-introduction exists (announcement or content page matching an introduction heuristic).
- **FR-21.** `orientation.learner-introductions` (**recommended**; QM 1.9, OSCQR 41) — DONE when an
  introductions discussion thread/forum or feed prompt exists. `not_applicable` when discussions and feed
  are both disabled or the course has < 2 students.

### A3 — Syllabus & published policies

- **FR-22.** `syllabus.exists` (**essential**; OSCQR 3, QM 1.2) — DONE when `course.course_syllabus.sections`
  contains ≥ 2 sections and ≥ 600 characters of combined markdown.
- **FR-23.** `syllabus.grading-policy` (**essential**; QM 3.2, OSCQR 44) — DONE when the syllabus states how
  grades are computed (weights/points/scale keywords) **and** the stated model is consistent with
  `course.assignment_groups` (flags "syllabus says weighted, groups sum to 0").
- **FR-24.** `syllabus.late-policy` (**essential**; QM 3.2, OSCQR 44) — DONE when the syllabus states a late
  policy **and** the per-item `late_submission_policy` values on `module_assignments` / `module_quizzes` are
  not left entirely at the untouched default across the course. Evidence lists items whose policy contradicts
  the syllabus keyword (e.g. syllabus says "no late work", items say `allow`).
- **FR-25.** `syllabus.academic-integrity` (**essential**; QM 1.3, OSCQR 5) — DONE when an academic-integrity
  section exists; detail nudges toward stating an **AI-use policy** and links to the AI-disclosure settings
  (`internal/aidisclosure`) when any AI feature is enabled on the course.
- **FR-26.** `syllabus.accessibility-statement` (**essential**; QM 8.1, OSCQR 5, ADA/§508) — DONE when an
  accessibility / accommodations / disability-services section exists.
- **FR-27.** `syllabus.acceptance-decision` (**recommended**; product) — DONE when
  `require_syllabus_acceptance` has been explicitly set (either value) rather than left untouched.
  Represented by the same `features_reviewed_at`-style marker: `course_syllabus.acceptance_decided_at`.
- **FR-28.** `syllabus.printable` (**recommended**; OSCQR 4) — DONE when the syllabus print/export path is
  reachable, i.e. sections contain no unsupported embeds that break the print view. Evidence lists offending
  blocks.

### A4 — People & enrollment

- **FR-29.** `people.students-enrolled` (**essential**; product, OSCQR 7) — DONE when ≥ 1 active `student`
  enrollment exists. `in_progress` when only pending invitations exist; evidence lists pending invites with
  age. Target: `/courses/{code}/enrollments`.
- **FR-30.** `people.staff-roles` (**recommended**; QM 1.8) — DONE when the course has ≥ 1 staff enrollment
  beyond the creator **or** the item is dismissed. `not_applicable` for homeschool-mode courses
  (`course_type`/`course_mode` indicating single-instructor).
- **FR-31.** `people.sections` (**recommended**, applies only when `sections_enabled`) — DONE when ≥ 1
  section exists and every student enrollment belongs to a section. Evidence lists unsectioned students.
- **FR-32.** `people.stale-invitations` (**recommended**) — DONE when no invitation is older than 14 days
  unaccepted. Evidence lists them with a resend target.
- **FR-33.** `people.guardian-links` (**recommended**, applies only when the org is K-12 and the parent
  portal is enabled) — DONE when every student has ≥ 1 linked guardian. Evidence lists students without.

### Cross-cutting

- **FR-34.** Every rule MUST supply English `Title`/`Why` copy plus an i18n key, and MUST cite ≥ 1 source in
  `Sources`.
- **FR-35.** Every text-matching rule MUST use a **locale-aware keyword set** loaded from
  `server/internal/service/coursechecklist/lexicons/` (one JSON per locale), not inline English regexes, and
  MUST fall back to the English set for unsupported locales.
- **FR-36.** Every rule MUST be `not_applicable` rather than `todo` when the feature it inspects is disabled
  on the course.
- **FR-37.** All 33 rules MUST land at `Tier: recommended`; promotion of the 17 marked **essential** above
  happens in a follow-up change gated on the §15 dogfood criteria.

## 6. Non-Functional Requirements

- **Performance** — All rules in this pack MUST be pure snapshot functions with no lazy loaders; the whole
  pack MUST add < 40 ms to `Evaluate` p95. Text-matching rules MUST compile their regexes once at package
  init, never per evaluation.
- **Security** — Rules read only snapshot data already authorised for course staff. Evidence naming students
  (FR-31, FR-33) MUST use display name + opaque ID only.
- **Privacy & Compliance** — FR-33 touches guardian relationships (FERPA); evidence MUST reuse the existing
  parent-portal visibility rules and MUST NOT expose guardian contact details. FR-29 evidence MUST NOT
  include invitee email addresses beyond what the enrollments page already shows to staff.
- **Accessibility** — Copy MUST be plain language (target grade 8), avoid "you failed" framing, and never
  rely on colour alone (status is also a word). FR-26 exists precisely to push courses toward accessibility.
- **Scalability** — Text rules operate on syllabus markdown capped at 512 KB; larger syllabi are scanned
  over the first 512 KB and the finding carries `Detail: "checked first 512 KB"`.
- **Reliability** — A malformed syllabus JSON blob MUST yield `unknown` for syllabus rules, not an error.
- **Observability** — Per-rule duration and error counters come free from CC.1. CC.10 adds per-rule
  pass-rate and dismissal-rate reporting; CC.3 MUST ensure every rule ID is stable enough to chart.
- **Maintainability** — Rules live in `rules_foundations.go`, `rules_orientation.go`, `rules_syllabus.go`,
  `rules_people.go`. Each has a sibling `_test.go` with a table of snapshots.
- **Internationalization** — FR-35 lexicons; ship `en` at minimum plus `es`, `fr`, `ar` to match the
  existing locale set in `clients/mobile/locales/`. Missing lexicon ⇒ English fallback, never `unknown`.
- **Backward compatibility** — Two additive columns (FR-10, FR-27); no behaviour change for existing
  endpoints beyond stamping a timestamp.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course with `starts_at` after `ends_at`, *When* the checklist is read, *Then*
  `course.dates` is `todo` with detail "End date is before the start date".
- **AC-2.** *Given* a self-paced course (`schedule_mode = 'relative'`), *When* the checklist is read, *Then*
  `course.dates` is `not_applicable` and `course.relative-schedule` is present.
- **AC-3.** *Given* an unpublished course starting in 3 days, *When* the checklist is read, *Then*
  `course.published` is `in_progress` and its detail names the days remaining.
- **AC-4.** *Given* an announcements channel with one 400-character staff post, *When* the checklist is read,
  *Then* `orientation.welcome-message` is `done`.
- **AC-5.** *Given* an announcements channel whose only post is a 12-character student message, *Then*
  `orientation.welcome-message` is `todo`.
- **AC-6.** *Given* a syllabus containing "I respond to email within 24 hours", *Then*
  `orientation.response-time` is `done`; *Given* a syllabus with no such statement, *Then* it is `todo`.
- **AC-7.** *Given* a Spanish-locale course whose syllabus says "responderé en 24 horas", *Then*
  `orientation.response-time` is `done` (lexicon test, FR-35).
- **AC-8.** *Given* a course with 12 students and no section rows while `sections_enabled` is true, *Then*
  `people.sections` is `todo` with 12 evidence rows; *Given* `sections_enabled` false, *Then* it is
  `not_applicable`.
- **AC-9.** *Given* a course with zero student enrollments and 3 pending invitations, *Then*
  `people.students-enrolled` is `in_progress` with 3 evidence rows and a resend target.
- **AC-10.** *Given* the features page has never been saved for a course, *Then* `course.features-reviewed`
  is `todo`; *Given* a `PATCH /features` call, *Then* it becomes `done` on the next recheck.
- **AC-11.** *Given* a syllabus stating "no late work accepted" while five assignments carry
  `late_submission_policy = 'allow'`, *Then* `syllabus.late-policy` is `in_progress` with those five as
  evidence.
- **AC-12.** *Given* a homeschool-mode course, *Then* `people.staff-roles` and `people.guardian-links` are
  `not_applicable`.
- **AC-13.** *Given* a syllabus row whose `sections` JSON fails to parse, *Then* every syllabus rule is
  `unknown` and no other rule is affected.
- **AC-14.** *Given* the registry integrity test, *Then* all 33 rules have non-empty `Sources`, a valid
  target route, and unique IDs.

## 8. Data Model

Two additive columns, in `server/migrations/462_course_checklist_review_markers.sql`:

```sql
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS features_reviewed_at TIMESTAMPTZ;

ALTER TABLE course.course_syllabus
    ADD COLUMN IF NOT EXISTS acceptance_decided_at TIMESTAMPTZ;

COMMENT ON COLUMN course.courses.features_reviewed_at IS
    'Set when course feature switches are saved; drives checklist item course.features-reviewed.';
COMMENT ON COLUMN course.course_syllabus.acceptance_decided_at IS
    'Set when require_syllabus_acceptance is explicitly chosen; drives syllabus.acceptance-decision.';
```

- **Backfill**: leave `NULL`. A course whose features were configured before this ships shows the item as
  `todo` until the next save — acceptable, and cheap to clear. Alternative (backfill from
  `courses.updated_at`) is rejected because it would mark unreviewed courses as reviewed.
- **Writers**: `handlePatchCourseFeatures` (`course_features.go`) stamps `features_reviewed_at = now()`;
  `UpsertSyllabus` (`repos/course/syllabus.go`) stamps `acceptance_decided_at` when the
  `requireSyllabusAcceptance` field is present in the request body.
- **Copy semantics**: `coursecopy` MUST copy `features_reviewed_at` (the decision carries over) and MUST NOT
  copy `acceptance_decided_at` when the syllabus is not copied.
- New lexicon assets live in `server/internal/service/coursechecklist/lexicons/{locale}.json`, embedded with
  `go:embed`; no DB storage.

## 9. API Surface

No new routes. Two existing handlers gain a side effect:

- `PATCH /api/v1/courses/{course_code}/features` — stamps `features_reviewed_at`. Response shape unchanged.
- `PUT /api/v1/courses/{course_code}/syllabus` — stamps `acceptance_decided_at` when the acceptance flag is
  present. Response shape unchanged.

Both are additive and require no OpenAPI change beyond a description note.

## 10. UI / UX

No new UI. CC.3 fixes the **copy contract** that CC.7 renders. Copy rules:

- `Title` is an imperative, ≤ 60 chars: "Post a welcome announcement", not "Welcome message missing".
- `Why` is one sentence naming the learner benefit: "Students who read a welcome post in week 1 are less
  likely to disengage — every major quality rubric asks for one."
- `Detail` names the concrete gap and, where possible, the number: "11 of 24 assessments are not mapped."
- Done state reads as a struck-through past-tense fact: "Welcome announcement posted".
- No rule may use guilt framing, exclamation marks or "you should have".
- `Sources` render as small citation chips ("QM 1.1", "OSCQR 1") linking to
  [research.md](../../help/course-checklist/research.md).

## 11. AI / ML Considerations

**Deliberately none.** Every text check in this pack is a locale-aware keyword/regex heuristic (FR-35), not
a model call, so that evaluation stays free, deterministic, offline-safe and privacy-neutral. The known cost
is recall: a syllabus that states a response-time commitment in unusual phrasing will read as `todo`. That is
mitigated by (a) dismissal with reason `done_elsewhere`, and (b) CC.10 tracking dismissal-by-reason per rule
so a high `done_elsewhere` rate flags a heuristic to widen. An optional AI second-pass for the text rules is
recorded as §18 Q3, not scoped here.

## 12. Integration Points

- Internal: `server/internal/service/coursechecklist` (rules + lexicons), `server/internal/repos/course`
  (syllabus, features, dates), `server/internal/repos/coursefeed` (announcements),
  `server/internal/repos/enrollment` (+ invitations), `server/internal/repos/coursesections`,
  `server/internal/relativeschedule`, `server/internal/l10n`, `server/internal/aidisclosure` (FR-25 link),
  parent-portal repos for FR-33.
- Handlers touched: `server/internal/httpserver/course_features.go`, syllabus handler.
- Nav targets referenced (must exist): `/settings/general`, `/settings/features`, `/settings/grading`,
  `/syllabus`, `/feed`, `/enrollments`, `/settings/sections`, `/modules`.
- No external services.

## 13. Dependencies & Sequencing

- Must ship after: CC.1 (registry/engine), CC.2 (API + persistence).
- Must ship before: CC.7 GA (the page needs a real catalog), CC.10.
- Migration `462` must land before the rules referencing the new columns evaluate.
- Coordinate with CC.4 on shared snapshot `DataNeeds` so the two packs do not each add a query for the same
  table.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Keyword heuristics produce false "todo" and erode trust | **H** | H | Ship all rules `recommended` first (FR-37); dismissal with `done_elsewhere`; CC.10 per-rule dismissal dashboards; widen lexicons before promoting |
| 33 items at once feels overwhelming | M | M | Categories collapse by default in CC.7 with counts; only `essential` items drive the badge |
| Non-English courses fail text rules wholesale | M | H | FR-35 lexicons for en/es/fr/ar + AC-7; unsupported locale ⇒ rule reports `unknown`, not `todo` |
| `features_reviewed_at` NULL backfill nags every existing course | **H** | M | Rule ships `recommended`; one features save clears it; help copy explains |
| Rules encode US higher-ed assumptions | M | M | K-12 and homeschool applicability predicates (FR-30, FR-33); demo-course review covers all three markets |
| Two migrations (461, 462) land out of order across branches | L | M | 462 is additive and independent of 461; verify numbering at merge |

## 15. Rollout Plan

**No feature flag.** Staging is done through `Tier` and the registry, not a toggle:

1. Migration `462` ships first (additive columns, no reads).
2. Rules ship as `recommended` — visible on the page, **not** counted in the nav badge.
3. Dogfood two weeks against: the three seeded demo courses, Lextures-internal courses, and a pilot cohort of
   ~20 volunteer instructors across K-12 / HE / homeschool.
4. Promotion gate per rule (not per pack): `todo`-rate plausible on manual review **and** dismissal rate with
   reason `disagree` < 10% **and** with reason `done_elsewhere` < 15%. Rules that fail the gate get their
   lexicon widened or stay `recommended`.
5. Promote the 17 `essential` rules in one follow-up change; announce in-product via the existing banner
   system so the badge appearing is not a surprise.
6. Rollback: move a rule ID into `RETIRED_ITEM_IDS` (server-only release) or demote its tier.

## 16. Test Plan

- **Unit** — One table-driven test per rule with ≥ 4 snapshots (done / todo / edge / not-applicable), plus
  lexicon tests per supported locale. Copy tests asserting `Title` ≤ 60 chars and no banned words
  ("failed", "should have", "!").
- **Integration** — DB tests for the two new columns and their writers; snapshot loader returns the fields
  each rule declares; malformed-syllabus containment (AC-13).
- **End-to-end** — Playwright: seed a bare course, assert the checklist shows the expected `todo` set; post a
  welcome announcement, assert `orientation.welcome-message` flips to `done` after recheck; enroll a student,
  assert `people.students-enrolled` flips.
- **Security** — Assert evidence rows for FR-31/FR-33 contain no email/contact fields; assert the pack adds
  no route.
- **Accessibility** — Copy review against the plain-language target; verify status words accompany every
  colour in the CC.7 render (asserted in CC.7's axe suite).
- **Performance / load** — Benchmark asserting the pack adds < 40 ms p95 to `Evaluate` on the 300-item
  fixture; regex-compilation-at-init test.
- **Manual exploratory** — Instructional-design review of all 33 items' copy and rubric citations before
  promotion; run against 10 real (consented) pilot courses and hand-score every finding.

## 17. Documentation & Training

- [research.md](../../help/course-checklist/research.md) — the rubric-to-rule mapping table for all four
  packs; CC.3 populates its A-sections.
- Help-centre article "Getting your course ready" walking the foundation items in order.
- `docs/dev/course-checklist-engine.md` gains a "writing a text-heuristic rule" section covering lexicons.
- Instructor-facing per-item help anchors (`HelpRef`) authored alongside each rule.

## 18. Open Questions

1. Should `course.features-reviewed` be satisfied by *any* features save, or require the instructor to have
   seen each feature group? Proposed: any save; revisit if it reads as a checkbox-ticking exercise.
2. Is 120 characters the right floor for a course description, and 600 for a syllabus? Proposed values are
   from a sample of the seeded courses; confirm with pilot data.
3. Should an optional AI pass re-check `todo` text rules before showing them (reducing false positives at a
   token cost)? Deferred to CC.10; would need the AI-provider plumbing from `docs/plan/ai-providers/`.
4. Does `people.guardian-links` belong here or in the parent-portal plan folder? Proposed: rule lives here,
   the underlying data comes from PP.
5. For cross-listed courses, do foundation rules evaluate on the parent or on each child? Proposed: each
   child, since dates and enrollments differ.

## 19. References

- Existing files this work touches: `server/internal/service/coursechecklist/rules_*.go` (new),
  `server/internal/httpserver/course_features.go`, `server/internal/repos/course/syllabus.go`,
  `server/internal/repos/coursefeed/`, `server/internal/repos/enrollment/`, `server/migrations/462_*`.
- External standards: [Quality Matters Higher Ed Rubric](https://www.qualitymatters.org/qa-resources/rubric-standards/higher-ed-rubric)
  GS1 (Course Overview and Introduction) and GS7 (Learner Support);
  [SUNY OSCQR](https://oscqr.suny.edu/) "Overview and Information" (1–10) and "Technology and Tools" (11–15);
  [NSQ Online Courses](https://nsqol.org/the-standards/quality-online-courses/) Standard A.
- Related plans: [CC.1](../../completed/checklist/CC.1-checklist-registry-and-evaluation-engine.md),
  [CC.2](CC.2-checklist-state-api-and-dismissals.md), [CC.4](CC.4-rule-pack-structure-outcomes-alignment.md),
  [CC.7](CC.7-web-checklist-page-and-nav-badge.md), [CC.10](CC.10-analytics-guidance-and-rollout.md).
