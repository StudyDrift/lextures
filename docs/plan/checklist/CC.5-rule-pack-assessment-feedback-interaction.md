# CC.5 — Rule Pack C: Assessment, Grading, Feedback & Interaction

> Implementation plan. Source: Course Checklist product request. Folder overview: [README](README.md).
> Rubric mapping: [course-design-research.md](course-design-research.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CC.5 |
| **Section** | Course Checklist |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Server / platform team + instructional design |
| **Depends on** | CC.1, CC.2 |
| **Unblocks** | CC.7, CC.10 |

---

## 1. Problem Statement

Assessment and feedback are where course quality is most visible to students and most often broken in
practice: assignment group weights that sum to 87%, a rubric-less 30% capstone, every assessment stacked in
the final two weeks, a quiz that shows correct answers before the late window closes. OSCQR devotes its
whole final category (standards 44–50) to grading, criteria, formative assessment, accommodations and the
gradebook; QM General Standard 3 covers measurement and General Standard 5 covers interaction. Lextures
stores all of the underlying facts — `course.assignment_groups.weight_percent`, `module_assignments`,
`module_quizzes`, `course.assignment_rubrics`, `course.discussion_forums`, `course.appointment_slots` — and
surfaces none of them as a readiness signal. CC.5 ships the 26 rules that cover assessment configuration,
grading integrity, feedback commitments and learner interaction.

## 2. Goals

- Ship the **26 assessment and interaction rules** in §5 (6 item configuration · 6 grading integrity · 6 criteria & feedback · 4 integrity & accommodations · 4 interaction), each with a rubric citation, nav target and, where
  useful, an evidence table of the specific offending items.
- Catch the **arithmetic and configuration defects** that silently produce wrong grades (weights ≠ 100%,
  zero-point gradable items, drop rules that exceed group size).
- Surface the **pedagogical** expectations that instructors are rarely told: rubrics on high-stakes work,
  low-stakes formative checks per module, distributed assessment across the term.
- Make interaction rules honest: silent when the course has no discussions, groups or office hours enabled.

## 3. Non-Goals

- No structure or outcome-alignment rules (CC.4); no accessibility or launch readiness (CC.6).
- No grading-workload analytics ("you have 42 ungraded submissions") — that is the existing grading backlog
  surface on the course dashboard, not a design checklist item.
- No changes to grading, rubric or discussion data models or UI.
- No AI rubric generation — the Build-with-AI rubric path already exists and CC.10 links to it.
- No enforcement: a course with weights ≠ 100% is flagged, never blocked.

## 4. Personas & User Stories

- **As an instructor**, I want to be told my assignment group weights sum to 87% before students see wrong
  grades, so that I fix arithmetic instead of apologising.
- **As an instructor**, I want a nudge that my 30%-of-grade essay has no rubric, so that grading is
  defensible and students know the criteria up front.
- **As a student** (indirectly), I want every graded thing to have a due date and points, so that the
  gradebook is not a mystery.
- **As a program lead**, I want "formative checks exist in each module" to be visible, so that our courses
  are not all high-stakes finals.
- **As a K-12 teacher**, I want accommodations to be flagged when a student has extended time but my quizzes
  have hard time limits, so that I do not accidentally violate an IEP/504 plan.
- **As a homeschool parent**, I want interaction rules about discussions and groups to disappear when I have
  one learner, so that the list stays honest.

## 5. Functional Requirements

### C1 — Gradable item configuration

- **FR-1.** `assessment.gradable-items` (**essential**; QM 3.1, OSCQR 45) — DONE when ≥ 1 non-archived,
  published-or-draft gradable item exists. `not_applicable` for courses in a non-graded mode.
- **FR-2.** `assessment.due-dates` (**essential**; QM 1.2, OSCQR 44) — DONE when every gradable item has a
  `due_at` **or** is explicitly marked undated. Evidence lists undated items. `not_applicable` for
  `schedule_mode = 'relative'` (covered by CC.3 FR-5).
- **FR-3.** `assessment.points` (**essential**; OSCQR 44/46) — DONE when no gradable item has null or zero
  points while belonging to a weighted group. Evidence lists them.
- **FR-4.** `assessment.dates-within-term` (**recommended**; OSCQR 7) — DONE when no due date falls outside
  `[starts_at, ends_at]`. Evidence lists out-of-range items — the single most common copy-forward defect.
- **FR-5.** `assessment.availability-windows` (**recommended**; OSCQR 44) — DONE when no item has
  `available_from > due_at` or `available_until < due_at`. Evidence lists contradictions.
- **FR-6.** `assessment.spread` (**recommended**; QM 3.4, OSCQR 47) — DONE when no single week holds more
  than 40% of the course's total points and the final week holds less than 50%. `Detail` names the
  overloaded weeks. Purely arithmetic over due dates and points.

### C2 — Grading model integrity

- **FR-7.** `grading.group-weights` (**essential**; QM 3.2, OSCQR 44) — applies only when weighted grading is
  in use. DONE when `SUM(course.assignment_groups.weight_percent) = 100` (±0.01). `todo` detail states the
  actual sum. Target: `/courses/{code}/settings/grading#grading.groups`.
- **FR-8.** `grading.empty-groups` (**recommended**) — DONE when no weighted group with weight > 0 contains
  zero items (a 20% group with nothing in it silently redistributes the grade). Evidence lists them.
- **FR-9.** `grading.drop-rules` (**recommended**) — DONE when no group's `drop_lowest + drop_highest` is ≥
  its item count. Evidence lists impossible drop rules.
- **FR-10.** `grading.scheme-coverage` (**recommended**; OSCQR 44) — DONE when the selected grading scheme's
  bands cover 0–100 with no gaps or overlaps.
- **FR-11.** `grading.posting-policy` (**recommended**; QM 3.5, OSCQR 38) — DONE when the course's
  grade-posting policy (manual vs automatic) has been explicitly chosen, and, if manual, the syllabus states
  when grades are released.
- **FR-12.** `grading.late-policy-configured` (**recommended**; OSCQR 44) — DONE when a course-level late
  policy exists **or** every gradable item's `late_submission_policy` was explicitly set. Complements CC.3
  FR-24, which checks that the syllabus *states* the policy.

### C3 — Criteria & feedback

- **FR-13.** `feedback.rubrics-on-high-stakes` (**essential**; QM 3.3, OSCQR 46) — DONE when every assignment
  worth ≥ 10% of the course total (or ≥ 100 points when unweighted) has a rubric in
  `course.assignment_rubrics` **or** a non-empty grading-criteria description. Evidence lists high-stakes
  items without criteria, each targeting that assignment's rubric section.
- **FR-14.** `feedback.criteria-published` (**recommended**; QM 3.3, OSCQR 46) — DONE when every gradable
  item has a non-empty description/instructions field. Evidence lists items with no instructions at all.
- **FR-15.** `feedback.formative-per-module` (**recommended**; QM 3.4, OSCQR 47, NSQ D) — DONE when every
  module contains ≥ 1 low-stakes formative element: an ungraded practice quiz, a survey, an inline
  comprehension Content Tool, an SRS deck, or an item worth < 2% of the total. Evidence lists modules with
  only high-stakes work.
- **FR-16.** `feedback.quiz-review-settings` (**recommended**; QM 3.5, OSCQR 45) — DONE when every quiz has
  had its scores-and-review settings explicitly configured (what students see, when). Flags quizzes that
  reveal correct answers immediately while the availability window is still open for other students —
  a genuine integrity defect, not a style note. Evidence lists them.
- **FR-17.** `feedback.attempts-policy` (**recommended**) — DONE when every quiz's attempts and
  `grade_attempt_policy` are consistent (e.g. not "1 attempt" with "highest of attempts").
- **FR-18.** `feedback.peer-review-config` (**recommended**, applies only when a peer-review config exists) —
  DONE when every peer-reviewed assignment has an allocation strategy, a review window that opens after its
  due date, and a rubric.

### C4 — Integrity & accommodations

- **FR-19.** `integrity.high-stakes-settings` (**recommended**; QM 3.x, OSCQR 45) — DONE when the course has
  explicitly reviewed integrity settings for items worth ≥ 20% (lockdown, proctoring, plagiarism,
  question shuffling). `not_applicable` when no item crosses the threshold. This is a *review* marker, not a
  demand that integrity tooling be switched on.
- **FR-20.** `integrity.ai-policy-alignment` (**recommended**) — DONE when, if any AI-assist feature is
  enabled for learners (AI tutor, adaptive content, content-tool AI), the syllabus academic-integrity section
  mentions AI use. Cross-checks CC.3 FR-25.
- **FR-21.** `accommodations.honored` (**essential**; OSCQR 48, ADA/§504) — applies when ≥ 1 row exists in
  `course.student_accommodations`. DONE when no timed quiz conflicts with an active extended-time
  accommodation that the platform cannot auto-apply. Evidence lists (quiz, accommodation type) pairs using
  **counts and accommodation type only — never a student name** (see §6 Privacy). Target:
  `/courses/{code}/settings/accessibility`.
- **FR-22.** `accommodations.reviewed` (**recommended**) — DONE when the accommodations surface has been
  opened for this course since the most recent accommodation was added. Requires the review marker in §8.

### C5 — Interaction & community

- **FR-23.** `interaction.discussion-exists` (**recommended**; QM 5.2, OSCQR 39/42, NSQ C) — applies only
  when discussions, feed, groups or boards are enabled and the course has ≥ 2 students. DONE when ≥ 1
  structured discussion prompt, forum or graded discussion exists.
- **FR-24.** `interaction.instructor-presence-plan` (**recommended**; QM 5.3, OSCQR 40) — DONE when ≥ 1
  announcement is scheduled or posted per 2 weeks of course duration, or the item is dismissed. Evaluated
  from `course.feed_messages` in the announcements channel plus scheduled sends.
- **FR-25.** `interaction.office-hours` (**recommended**, applies only when `office_hours_enabled`; QM 1.7,
  OSCQR 40) — DONE when ≥ 1 `course.appointment_slots` row exists in the future.
- **FR-26.** `interaction.groups-configured` (**recommended**, applies only when `enrollment_groups_enabled`
  or a group assignment exists; QM 5.2) — DONE when ≥ 1 group set exists and every student belongs to a
  group. Evidence: count of unassigned students (no names).

### Cross-cutting

- **FR-27.** All 26 rules land at `Tier: recommended`; the six marked **essential** are promoted per §15.
- **FR-28.** Rules MUST be pure snapshot functions; new `DataNeeds`: `assignment_groups`, `rubrics`,
  `quiz_settings`, `discussions`, `office_hours`, `accommodation_counts`, `peer_review_configs`.
- **FR-29.** Every arithmetic rule (FR-7, FR-9, FR-10, FR-6, FR-13) MUST state its computed number in
  `Detail` so the instructor can verify the claim.

## 6. Non-Functional Requirements

- **Performance** — ≤ 5 additional snapshot queries; < 50 ms p95 added to `Evaluate`. FR-6's weekly
  bucketing is O(n) over gradable items. FR-13's percentage-of-total computation reuses the CC.4 shared
  gradable set.
- **Security** — Read-only. Integrity settings (FR-19) are read, never changed.
- **Privacy & Compliance** — This is the only pack that touches disability data. `course.student_accommodations`
  is **special-category data**; FR-21 and FR-22 MUST expose only *counts and accommodation types*, never a
  student identifier, and MUST NOT be included in any export or analytics event (CC.10 excludes these two
  rule IDs from evidence-bearing telemetry). FR-26 similarly reports a count, not names. Reviewed against
  FERPA and the §504/ADA obligations tracked in `docs/plan/standards/`.
- **Accessibility** — FR-21 exists to protect accommodations; its copy MUST be non-alarmist and MUST link to
  the accommodations documentation rather than implying wrongdoing.
- **Scalability** — Weekly bucketing capped at 520 buckets (10 years) to bound pathological date data.
- **Reliability** — Division-by-zero guards on every percentage computation; a course with 0 total points
  yields `not_applicable` for FR-6 and FR-13, not `unknown`.
- **Observability** — Per-rule metrics from CC.1. `grading.group-weights` failure rate is worth a dedicated
  product dashboard in CC.10 — it is a data-integrity signal, not just a design one.
- **Maintainability** — `rules_assessment.go`, `rules_grading.go`, `rules_feedback.go`, `rules_interaction.go`.
- **Internationalization** — Copy keys per CC.1; no text heuristics in this pack except FR-11/FR-20, which
  reuse the CC.3 lexicons.
- **Backward compatibility** — One additive review-marker column (§8); no API change.

## 7. Acceptance Criteria

- **AC-1.** *Given* assignment groups weighted 40/30/17, *When* the checklist is read, *Then*
  `grading.group-weights` is `todo` with detail "Weights add up to 87%, not 100%".
- **AC-2.** *Given* groups summing to exactly 100.00, *Then* the item is `done`; *Given* unweighted grading,
  *Then* it is `not_applicable`.
- **AC-3.** *Given* a group weighted 20% containing zero items, *Then* `grading.empty-groups` lists it.
- **AC-4.** *Given* a group with 3 items and `drop_lowest = 2, drop_highest = 1`, *Then* `grading.drop-rules`
  is `todo` naming the group.
- **AC-5.** *Given* an assignment worth 30% with no rubric and an empty description, *Then*
  `feedback.rubrics-on-high-stakes` lists it and its row targets that assignment's rubric section.
- **AC-6.** *Given* every module contains an ungraded practice quiz, *Then* `feedback.formative-per-module`
  is `done`.
- **AC-7.** *Given* a due date two weeks after `ends_at`, *Then* `assessment.dates-within-term` lists it.
- **AC-8.** *Given* 60% of the course's points due in the final week, *Then* `assessment.spread` is `todo`
  with the computed percentage in `Detail`.
- **AC-9.** *Given* a quiz revealing correct answers immediately while its availability window is open,
  *Then* `feedback.quiz-review-settings` flags it.
- **AC-10.** *Given* a student with an extended-time accommodation and a hard-time-limited quiz, *Then*
  `accommodations.honored` is `todo` and the serialized evidence contains **no** user ID or name (asserted).
- **AC-11.** *Given* a course with 1 student, *Then* `interaction.discussion-exists` and
  `interaction.groups-configured` are `not_applicable`.
- **AC-12.** *Given* `office_hours_enabled = false`, *Then* `interaction.office-hours` is `not_applicable`.
- **AC-13.** *Given* a course with 0 total points, *Then* `assessment.spread` and
  `feedback.rubrics-on-high-stakes` are `not_applicable` and no division-by-zero panic occurs.
- **AC-14.** *Given* the pack, *When* the benchmark runs, *Then* it adds ≤ 5 queries and < 50 ms p95.

## 8. Data Model

One additive column, in `server/migrations/463_course_checklist_review_markers_2.sql`:

```sql
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS accommodations_reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS integrity_settings_reviewed_at TIMESTAMPTZ;

COMMENT ON COLUMN course.courses.accommodations_reviewed_at IS
    'Set when course staff open/save the course accommodations surface; drives accommodations.reviewed.';
COMMENT ON COLUMN course.courses.integrity_settings_reviewed_at IS
    'Set when course staff save integrity settings; drives integrity.high-stakes-settings.';
```

Read-only sources otherwise:

| Rule group | Tables |
|---|---|
| Items | `course.course_structure_items`, `course.module_assignments`, `course.module_quizzes`, `course.module_surveys` |
| Grading | `course.assignment_groups` (incl. `drop_lowest`/`drop_highest`), `course.grading_schemes`, `course.courses.grading_scale` |
| Criteria | `course.assignment_rubrics` |
| Quiz behaviour | `course.module_quizzes` (attempts, `grade_attempt_policy`, review settings, `available_from`/`available_until`, `late_submission_policy`) |
| Peer review | `course.peer_review_configs` |
| Accommodations | `course.student_accommodations` — **counts and types only** |
| Interaction | `course.discussion_forums`, `course.discussion_threads`, `course.feed_messages`, `course.appointment_slots`, `course.enrollment_groups`, `course.enrollment_group_memberships` |

- **Backfill**: none; NULL markers read as `todo` on the two review-marker rules, both `recommended`.
- **Writers**: the accommodations and integrity settings handlers stamp their marker on save.

## 9. API Surface

No new routes. Two existing handlers stamp a marker column on save (accommodations settings, integrity
settings); response shapes unchanged. OpenAPI descriptions updated only.

## 10. UI / UX

No new pages. CC.5 establishes two copy conventions CC.7 must honour:

1. **Arithmetic items state the number.** "Weights add up to 87%, not 100%" — never "check your weights".
2. **Accommodation items are advisory and count-only.** "1 timed quiz may conflict with an extended-time
   accommodation" with a link to the accommodations page — never a student name, never "you are violating".

Evidence tables in this pack use columns `["Item", "Type", "Module", "Points", "Issue"]` (assessment rules)
or `["Group", "Weight", "Items"]` (grading rules).

## 11. AI / ML Considerations

None in the evaluators. Remediation links to **existing** AI affordances rather than new ones: the
assignment rubric Build-with-AI path (`437_assignment_rubric_generation_prompt` / the one-click rubric build
shipped in the content-tools work) is the natural "fix" for `feedback.rubrics-on-high-stakes`, and CC.10
wires the checklist item's action button to it. No new model, prompt or cost budget is introduced by CC.5.

## 12. Integration Points

- Internal: `server/internal/service/coursechecklist`, `server/internal/repos/coursegrading`,
  `server/internal/repos/assignmentrubric`, `server/internal/repos/coursemodulequiz`,
  `server/internal/repos/coursemoduleassignment`, `server/internal/repos/peerreview`,
  `server/internal/repos/accommodations`, `server/internal/repos/communication` /
  `coursefeed` (announcement cadence), `server/internal/gradingdrops`, `server/internal/gradingdisplay`.
- Nav targets: `/settings/grading`, assignment editor (`assignment.rubric`, `assignment.scheduling`,
  `assignment.grading` anchors), quiz editor (`quiz.scores-review`, `quiz.attempts-grading`,
  `quiz.scheduling`), `/settings/accessibility`, `/discussions`, `/office-hours`, `/groups`.
- No external services.

## 13. Dependencies & Sequencing

- Must ship after: CC.1, CC.2. Should follow CC.4 so the shared gradable-item set and points-total
  computation already exist in the snapshot.
- Must ship before: CC.10 (grading-integrity dashboards), CC.7 GA.
- Migration `463` before the two review-marker rules evaluate.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Accommodation evidence leaks disability data | L | **H** | Counts-and-types only (FR-21); serialization test asserts no user identifiers (AC-10); excluded from CC.10 telemetry |
| `feedback.rubrics-on-high-stakes` fires on every essay-based HE course | **H** | M | Accepts a written grading-criteria description as satisfying it, not only a formal rubric; `recommended` tier; one-click rubric build as the fix |
| `assessment.spread` encodes a pacing opinion instructors reject | M | M | Thresholds (40% week / 50% final week) are generous and stated in `Detail`; `recommended` only; dismissal reason tracked |
| Weight rule fires on courses that intentionally use points, not weights | M | M | Applies only when weighted grading is in use, determined from the grading scheme, not from group presence |
| Quiz review-settings rule misreads a legitimate practice quiz | M | L | Only flags when the availability window is still open for others **and** the quiz is graded |
| Interaction rules nag single-learner homeschool courses | M | M | ≥ 2 students precondition on FR-23/FR-26 |

## 15. Rollout Plan

**No feature flag.** Same tiered path as CC.3/CC.4:

1. Migration `463`, then all 26 rules ship `recommended`.
2. Dogfood 2 weeks; hand-score the arithmetic rules first — they should be near-zero false positive and are
   the strongest candidates for early promotion.
3. Promotion order: `grading.group-weights`, `assessment.due-dates`, `assessment.points`,
   `assessment.gradable-items` (all objectively verifiable) → then `accommodations.honored` →
   then `feedback.rubrics-on-high-stakes` last, after the one-click rubric fix is wired (CC.10).
4. Promotion gate per rule: `disagree` dismissal rate < 10%; for `accommodations.honored`, an additional
   privacy sign-off that no identifier appears in any surface or log.
5. Rollback: `RETIRED_ITEM_IDS` or tier demotion.

## 16. Test Plan

- **Unit** — Table-driven per rule. Arithmetic edge cases: weights 99.99/100.00/100.01, zero items, zero
  points, single item, negative points. Drop-rule boundaries. Week-bucketing across DST and year boundaries.
  Availability-window contradiction matrix.
- **Integration** — Seeded-DB tests per `DataNeed`; accommodations query returns aggregates only (asserted at
  the repo layer, so a future caller cannot accidentally widen it).
- **End-to-end** — Playwright: set group weights to 87%, assert the item and its detail text; fix to 100%,
  recheck, assert `done`. Create a 30% assignment with no rubric, assert the evidence row targets the rubric
  section.
- **Security** — Serialization test over the full checklist payload asserting no field matches a user-id or
  email pattern for the accommodations and groups rules; authz unchanged (CC.2).
- **Accessibility** — Copy review for FR-21/FR-22 tone; evidence column headers present.
- **Performance / load** — Benchmark: ≤ 5 queries, < 50 ms p95; 2,000-item fixture for the bucketing rule.
- **Manual exploratory** — Run against a real weighted HE course, a K-12 standards course with
  accommodations, and a homeschool single-learner course; hand-verify every finding and every
  `not_applicable`.

## 17. Documentation & Training

- [course-design-research.md](course-design-research.md) C-sections.
- Help-centre: "Why do my weights need to total 100%?", "Writing grading criteria students can use",
  "Formative checks: what counts".
- Accommodations doc note explaining exactly what the checklist can and cannot see (privacy transparency).
- Runbook entry: how to retire `accommodations.honored` immediately if a privacy concern is raised.

## 18. Open Questions

1. Is 10% of course total the right "high-stakes" threshold for FR-13, or should it be configurable per org?
   Proposed: fixed at 10% in v1, org-configurable later alongside the CC.2 §18 Q3 policy table.
2. Should `assessment.spread` use points or item count? Proposed: points, since that is what students feel.
3. Should FR-21 exist at all given the privacy sensitivity, or should it live only inside the accommodations
   surface? Proposed: keep it, count-only — the failure mode it prevents (a 504 violation) is serious.
   Needs explicit privacy/legal sign-off before promotion.
4. Does the announcement-cadence rule (FR-24) belong to design (checklist) or to teaching (a during-term
   nudge)? Proposed: keep as `recommended` and revisit if dismissal rates are high.
5. Should peer-review config checks move to the peer-review plan folder instead? Proposed: keep the rule
   here, the fix path there.

## 19. References

- Existing files this work touches: `server/internal/service/coursechecklist/rules_{assessment,grading,feedback,interaction}.go`
  (new), `server/internal/repos/coursegrading/`, `server/internal/repos/assignmentrubric/`,
  `server/internal/gradingdrops/`, `server/migrations/027_course_grading.sql`,
  `server/migrations/070_assignment_rubrics.sql`, `server/migrations/109_assignment_group_drop_rules.sql`.
- External standards: [OSCQR](https://oscqr.suny.edu/) "Assessment and Feedback" (44–50) and "Interaction"
  (38–43); [Quality Matters](https://www.qualitymatters.org/qa-resources/rubric-standards/higher-ed-rubric)
  GS3 (Assessment and Measurement) and GS5 (Course Activities and Learner Interaction);
  [NSQ](https://nsqol.org/the-standards/quality-online-courses/) Standard D (Learner Assessment).
- Related plans: [CC.4](CC.4-rule-pack-structure-outcomes-alignment.md),
  [CC.6](CC.6-rule-pack-accessibility-and-launch-readiness.md),
  [CC.10](CC.10-analytics-guidance-and-rollout.md).
