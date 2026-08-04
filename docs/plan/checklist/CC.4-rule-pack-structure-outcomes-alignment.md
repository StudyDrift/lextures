# CC.4 — Rule Pack B: Structure, Content & Outcome Alignment

> Implementation plan. Source: Course Checklist product request. Folder overview: [README](README.md).
> Rubric mapping: [course-design-research.md](course-design-research.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CC.4 |
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

Alignment is the single idea every course-quality rubric is built on: stated outcomes, the assessments that
measure them, and the content that prepares learners for those assessments must point at each other (QM
General Standards 2–5; NSQ Standard C). Lextures already stores outcomes (`course.course_learning_outcomes`)
and their mappings (`course.course_outcome_links`, with `measurement_level` and `intensity_level`), but
nothing tells an instructor that eleven of their twenty-four gradable items map to nothing, or that one of
their outcomes is never assessed. The same is true of plain structural hygiene: empty modules, items still
titled "Untitled page", unpublished items sitting inside a published module past the start date. CC.4 ships
the 22 rules that cover course structure, content quality and outcome alignment — including the
**assignment/quiz outcomes-mapping** item the product request called out by name, with an expandable table
of every unmapped item.

## 2. Goals

- Ship the **22 structure and alignment rules** in §5, each with a rubric citation and a nav target.
- Make `outcomes.assessment-mapping` a first-class, **evidence-rich** item: a table of every unmapped
  assignment and quiz, each row deep-linking to that item's outcomes panel.
- Cover **both directions** of alignment: items with no outcome, and outcomes with no assessment.
- Detect the structural defects that actually break a course on day one (empty modules, unpublished items,
  placeholder titles) rather than only design-theory concerns.
- Keep the pack inside the CC.1 snapshot budget: no new N+1 queries, no per-item round trips.

## 3. Non-Goals

- No assessment-configuration rules (rubrics, weights, due dates) — CC.5.
- No accessibility rules on content (alt text, headings, captions) — CC.6.
- No link-health crawling in this pack; external link checking is a lazy loader defined in CC.6.
- No auto-generation of outcomes or mappings. AI-assisted mapping is CC.10 and reuses the existing
  Build-with-AI plumbing rather than being reinvented here.
- No changes to the outcomes data model or the outcomes authoring UI.

## 4. Personas & User Stories

- **As an instructor**, I want a list of exactly which assignments and quizzes have no learning outcome, so
  that I can fix them in one sitting instead of auditing 24 items by hand.
- **As an instructor**, I want to know when an outcome I wrote is never actually measured, so that my
  assessment plan is honest.
- **As a program assessor**, I want alignment coverage to be visible before the term rather than at
  accreditation time, so that reporting is not a scramble.
- **As a K-12 teacher**, I want standards alignment checked when my district uses standards, and silent when
  it does not.
- **As a student** (indirectly), I want no empty modules and no "Untitled page" links, so that the course
  does not look abandoned.
- **As a course designer**, I want to be told when a module has no overview, so that learners know what a
  week is for before they start it.

## 5. Functional Requirements

Each rule MUST be registered per CC.1 FR-2. `essential` items drive the nav badge once promoted (CC.4 FR-23).

### B1 — Course structure hygiene

- **FR-1.** `structure.modules-exist` (**essential**; QM 1.2, OSCQR 16, NSQ C) — DONE when ≥ 1 module exists.
  Target: `/courses/{code}/modules`.
- **FR-2.** `structure.empty-modules` (**essential**; OSCQR 16) — DONE when every module contains ≥ 1 child
  item. Evidence: one row per empty module, each linking to that module.
- **FR-3.** `structure.placeholder-titles` (**essential**; OSCQR 20) — DONE when no structure item's title
  matches the placeholder lexicon (`untitled`, `new page`, `new assignment`, `test`, `asdf`, `tbd`,
  `lorem`, plus locale variants). Evidence lists offenders with their type.
- **FR-4.** `structure.module-overviews` (**recommended**; QM 1.2, OSCQR 2, NSQ A) — DONE when every module
  has an overview: a description on the module row, or a first child content page matching an overview
  heuristic. Evidence lists modules without one. `in_progress` reports `done/total`.
- **FR-5.** `structure.unpublished-items` (**essential**; OSCQR 7) — DONE when no unpublished item sits
  inside a published module whose `visible_from` has passed. Evidence lists them with a publish target.
  `not_applicable` before the course is published.
- **FR-6.** `structure.orphan-items` (**recommended**) — DONE when no structure item has a `parent_id`
  pointing at an archived or deleted module. Evidence lists orphans.
- **FR-7.** `structure.pacing-signal` (**recommended**; QM 8.6, OSCQR 30) — DONE when every module carries a
  duration/pacing signal (estimated time, or a date range derived from its items' due dates). Evidence lists
  modules with neither.
- **FR-8.** `structure.content-variety` (**recommended**; OSCQR 29, UDL Representation) — DONE when the
  course contains ≥ 2 distinct item kinds beyond plain content pages (assignment, quiz, discussion, external
  link, H5P, SCORM, LTI, library resource, video, content tool). `Detail` reports the observed mix; evidence
  lists modules that are text-only.
- **FR-9.** `structure.interactive-elements` (**recommended**; OSCQR 30/31, UDL Engagement) — DONE when at
  least 50% of content pages contain ≥ 1 interactive element (a Content Tool instance from
  `course.content_tool_instances`, an embedded quiz, or an H5P item). `not_applicable` when
  `content_tools_enabled` is false and no H5P/quiz embeds exist.
- **FR-10.** `structure.attribution` (**recommended**; OSCQR 32/33, copyright) — DONE when every external
  link / library resource / textbook resource item has a non-empty attribution or source field. Evidence
  lists items missing it.
- **FR-11.** `structure.file-references` (**recommended**; OSCQR 37) — DONE when every `course.file_items`
  reference embedded in module content resolves to an existing, non-archived file. Evidence lists broken
  internal references with the page they appear on.
- **FR-12.** `structure.gating-review` (**recommended**, applies only when `module_gating_enabled` or any
  `course.module_prerequisites` / `course.item_completion_rules` row exists) — DONE when every gated module
  has at least one satisfiable prerequisite path (no cycle, no prerequisite on an unpublished item).
  Evidence lists unsatisfiable gates. This is a **correctness** check, not a style check.

### B2 — Learning outcomes

- **FR-13.** `outcomes.defined` (**essential**; QM 2.1, NSQ C, OSCQR 9) — DONE when ≥ 3 course learning
  outcomes exist. `in_progress` at 1–2 with `done/total` progress. Target:
  `/courses/{code}/settings/outcomes`.
- **FR-14.** `outcomes.measurable` (**recommended**; QM 2.1/2.4) — DONE when every outcome's title begins
  with a measurable verb. Uses a **Bloom's-taxonomy verb lexicon** (locale-aware, per CC.3 FR-35) and flags
  non-measurable openers (`understand`, `know`, `learn about`, `be familiar with`, `appreciate`). Evidence
  lists each flagged outcome with a suggested replacement verb from the same Bloom level. Heuristic, no AI.
- **FR-15.** `outcomes.described` (**recommended**; QM 2.3) — DONE when every outcome has a non-empty
  `description`.
- **FR-16.** `outcomes.module-alignment` (**recommended**; QM 2.2, NSQ C) — DONE when every module contains
  at least one item mapped to at least one outcome. Evidence lists modules that teach toward nothing.
- **FR-17.** `outcomes.assessment-mapping` (**essential**; QM 3.1, NSQ C/D, OSCQR 45) — **the item named in
  the product request.** DONE when every gradable item (assignment or quiz) has ≥ 1 row in
  `course.course_outcome_links`. `in_progress` otherwise with `progress = {mapped, totalGradable}` and
  evidence columns `["Item", "Type", "Module", "Points"]`, one row per unmapped assignment or quiz, each row
  targeting that item's editor outcomes section (`assignment.outcomes-mapping` / `quiz.outcomes` in the PS.1
  settings registry — see CC.8). Evidence sorted by module order then item order.
- **FR-18.** `outcomes.coverage` (**essential**; QM 3.1, NSQ D) — the reverse direction: DONE when every
  outcome is measured by ≥ 1 link. Evidence lists orphan outcomes, each targeting the outcomes editor.
- **FR-19.** `outcomes.summative-coverage` (**recommended**; QM 3.1) — DONE when every outcome has ≥ 1 link
  with `measurement_level = 'summative'` (not only formative). Evidence lists outcomes measured only
  formatively.
- **FR-20.** `outcomes.mastery-scale` (**recommended**, applies only when `sbg_enabled`) — DONE when
  `sbg_proficiency_scale_json` is configured and a mastery threshold is set.
- **FR-21.** `outcomes.standards-alignment` (**recommended**, applies only when `standards_alignment_enabled`)
  — DONE when ≥ 1 `course.course_standards` row exists **and** every gradable item has ≥ 1 standard
  alignment. Evidence lists unaligned items. Target: `/courses/{code}/standards-coverage`.
- **FR-22.** `outcomes.syllabus-published` (**recommended**; QM 2.3, OSCQR 9) — DONE when the course outcomes
  also appear in the syllabus (a syllabus section referencing outcomes, or ≥ 60% token overlap with the
  outcome titles), so learners can actually read them.

### Cross-cutting

- **FR-23.** All 22 rules MUST land at `Tier: recommended`; the seven marked **essential** are promoted in a
  follow-up gated on §15.
- **FR-24.** Every rule producing evidence MUST cap at 200 rows (CC.1 FR-5) and MUST sort deterministically
  by `(module sort_order, item sort_order, title)`.
- **FR-25.** Rules MUST NOT issue their own queries; all data comes from `CourseSnapshot` `DataNeeds`
  (`structure`, `structure_detail`, `outcomes`, `outcome_links`, `standards`, `content_tool_instances`,
  `files`, `module_prerequisites`).
- **FR-26.** The snapshot loader MUST compute the gradable-item set once (`kind ∈ {assignment, quiz}`, not
  archived, not a survey) and share it between FR-17, FR-18, FR-19 and FR-21.

## 6. Non-Functional Requirements

- **Performance** — The pack MUST add ≤ 4 queries to the snapshot (outcomes, outcome links, standards,
  content-tool instance counts) and < 60 ms p95 to `Evaluate` on the 300-item fixture. FR-11's file-reference
  check MUST use a single pre-loaded file-id set, not a query per page. FR-12's cycle detection MUST be
  O(V+E) over the prerequisite graph.
- **Security** — Read-only, snapshot-only. Evidence names course content only, never learners.
- **Privacy & Compliance** — No learner data in this pack at all — evidence rows are course objects. That is
  a deliberate design constraint so the alignment evidence can be shown to accreditors without FERPA review.
- **Accessibility** — Evidence tables must be renderable as real tables with header cells; `EvidenceShape`
  MUST therefore always name its columns (CC.7 renders `<th scope="col">`).
- **Scalability** — Courses up to 2,000 structure items MUST evaluate within budget; above that, evidence is
  truncated and `Detail` states the true total.
- **Reliability** — A prerequisite cycle MUST NOT hang the evaluator (visited-set guard, AC-9). Missing
  optional tables (standards not provisioned) yield `not_applicable`.
- **Observability** — Per-rule metrics from CC.1. CC.10 charts `outcomes.assessment-mapping` mapped-ratio
  distribution as a product health metric.
- **Maintainability** — `rules_structure.go`, `rules_outcomes.go` + fixtures. The Bloom verb lexicon lives
  alongside the CC.3 lexicons and is versioned with the catalog.
- **Internationalization** — Placeholder-title lexicon (FR-3), overview heuristic (FR-4) and Bloom verbs
  (FR-14) are all locale-keyed; unsupported locale ⇒ `unknown` for FR-14 (never a false `todo` about
  someone's language) and English-only matching for FR-3.
- **Backward compatibility** — No schema change, no API change. Purely additive registry entries.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course with 24 gradable items of which 13 have outcome links, *When* the checklist is
  read, *Then* `outcomes.assessment-mapping` is `in_progress` with `progress = {13, 24}` and exactly 11
  evidence rows, ordered by module then item.
- **AC-2.** *Given* that evidence, *When* a row's target is followed, *Then* it resolves to that item's
  editor with the outcomes section focused (verified against the CC.8 target table).
- **AC-3.** *Given* a course where all gradable items are mapped, *Then* `outcomes.assessment-mapping` is
  `done` and renders struck-through.
- **AC-4.** *Given* an outcome with no links, *Then* `outcomes.coverage` is `todo` and lists that outcome.
- **AC-5.** *Given* an outcome linked only with `measurement_level = 'formative'`, *Then*
  `outcomes.summative-coverage` lists it while `outcomes.coverage` treats it as covered.
- **AC-6.** *Given* an outcome titled "Understand recursion", *Then* `outcomes.measurable` flags it and
  suggests a verb; *Given* "Implement a recursive solution", *Then* it does not.
- **AC-7.** *Given* a course with two modules containing zero items, *Then* `structure.empty-modules` is
  `todo` with two evidence rows.
- **AC-8.** *Given* a published course past its start date with three unpublished items inside published
  modules, *Then* `structure.unpublished-items` is `todo` with three rows; *Given* an unpublished course,
  *Then* it is `not_applicable`.
- **AC-9.** *Given* a prerequisite cycle A→B→A, *When* `structure.gating-review` runs, *Then* it returns
  `todo` naming the cycle and the evaluation terminates (no hang, no stack overflow).
- **AC-10.** *Given* `standards_alignment_enabled = false`, *Then* `outcomes.standards-alignment` is
  `not_applicable` and is excluded from the progress denominator.
- **AC-11.** *Given* a course with 2,000 items, *Then* evaluation stays within the §6 budget and evidence
  is truncated at 200 with the true count in `Detail`.
- **AC-12.** *Given* a content page embedding a file that was deleted, *Then* `structure.file-references`
  lists it with the containing page.
- **AC-13.** *Given* the snapshot, *Then* a test asserts the gradable-item set is computed once and reused
  by all four dependent rules (FR-26).

## 8. Data Model

**No schema changes.** CC.4 reads:

| Rule group | Tables |
|---|---|
| Structure hygiene | `course.course_structure_items` (+ `published`, `visible_from`, `parent_id`, `archived`), `course.module_content_pages`, `course.module_external_links`, `course.module_library_resources`, `course.module_textbook_resources` |
| Interactivity | `course.content_tool_instances`, H5P/SCORM/LTI item rows |
| Files | `course.course_files`, `course.file_items` |
| Gating | `course.module_prerequisites`, `course.item_completion_rules`, `course.module_requirements`, `course.structure_item_path_rules` |
| Outcomes | `course.course_learning_outcomes`, `course.course_outcome_links`, `course.course_outcome_sub_outcomes` |
| Standards | `course.course_standards`, `course.question_standard_alignments`, `course.concept_standard_alignments` |
| SBG | `course.courses.sbg_proficiency_scale_json` |

New snapshot `DataNeed` values: `outcomes`, `outcome_links`, `standards`, `content_tool_counts`,
`module_prerequisites`, `file_refs`. Each maps to exactly one batched query added to the CC.1 loader.

The Bloom verb lexicon and placeholder-title lexicon are embedded JSON assets, not database rows.

## 9. API Surface

None. Rules surface through the CC.2 endpoints. Two notes for client authors:

- `outcomes.assessment-mapping` is the first item whose `evidence.rows[].target` differs per row; CC.7 and
  CC.9 MUST support per-row targets (already in the CC.2 response shape).
- `progress` is populated for FR-4, FR-13, FR-17, FR-21 — clients render `13 / 24` next to the item.

## 10. UI / UX

No new pages. CC.4 defines the **expandable-evidence interaction** the product request asked for:

1. A `todo`/`in_progress` item with evidence renders a disclosure control ("Show the 11 items").
2. Expanding reveals a table with the declared columns; each row is a link to that specific item.
3. Following a row navigates and highlights (CC.8).
4. Returning to the checklist re-checks that single item (CC.2 `recheck`) so the row disappears once fixed.

Copy examples fixed by this plan:

- `outcomes.assessment-mapping` — Title: "Map every assessment to an outcome". Why: "Alignment is what makes
  a grade mean something — every rubric (QM 3.1, NSQ C) asks that each assessment measure a stated outcome."
  Detail (`in_progress`): "11 of 24 assessments aren't mapped."
- `structure.empty-modules` — Title: "Fill in or remove empty modules". Detail: "2 modules have no items."

## 11. AI / ML Considerations

None in the evaluators — FR-14's verb check is a lexicon lookup, deliberately, so it is deterministic and
free. The **remediation** side is where AI belongs: CC.10 will offer "suggest outcome mappings for these 11
items" using the existing Build-with-AI path (`handleBuildModuleContentPageWithAI` precedent) with
human approval before any link is written. CC.4 MUST NOT write outcome links under any circumstance
(CC.1 FR-15 read-only invariant).

## 12. Integration Points

- Internal: `server/internal/service/coursechecklist` (rules + lexicons), `server/internal/repos/coursestructure`,
  `server/internal/repos/courseoutcomes`, `server/internal/repos/standards`,
  `server/internal/repos/contenttools`, `server/internal/repos/coursefiles`,
  `server/internal/repos/conditionalrelease` (gating graph).
- Nav targets referenced: `/modules`, `/modules/{moduleId}`, assignment editor + `assignment.outcomes-mapping`
  setting anchor, quiz editor + `quiz.outcomes` setting anchor, `/settings/outcomes`, `/standards-coverage`,
  `/files`.
- Depends on the PS.1 settings-registry IDs for two evidence targets — CC.8 owns that binding.
- No external services.

## 13. Dependencies & Sequencing

- Must ship after: CC.1, CC.2.
- Should ship alongside or after CC.8 for the per-row deep links to land somewhere useful; if CC.8 is not
  ready, evidence rows target the item page without an anchor (graceful degradation, asserted by test).
- Must ship before: CC.10 (alignment metrics), CC.7 GA.
- Coordinate `DataNeeds` additions with CC.3 and CC.5 in one snapshot-loader change to avoid three
  overlapping refactors.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `outcomes.assessment-mapping` shows 40 unmapped items on every existing course and feels punitive | **H** | H | Ships `recommended`; `progress` framing ("13 of 24 mapped") rather than failure framing; dismissable; CC.10 AI-assisted bulk mapping |
| Snapshot query fan-out grows past the CC.1 budget | M | H | FR-25/FR-26 shared gradable set; ≤ 4 new queries; query-count test |
| Bloom verb lexicon is opinionated and wrong for a discipline | M | M | `recommended` tier only; suggestion not enforcement; dismiss reason `disagree` tracked |
| Prerequisite cycle detection hangs on pathological graphs | L | H | Visited-set + depth cap; AC-9 |
| Placeholder-title heuristic flags legitimate titles (a course *about* "Lorem Ipsum") | L | L | Lexicon requires whole-title or title-prefix match, not substring |
| Standards rules fire for orgs that enabled the flag but do not use standards | M | M | Requires ≥ 1 `course_standards` row before flagging items |

## 15. Rollout Plan

**No feature flag.** Staging via tier and registry, identical to CC.3:

1. Ship all 22 rules `recommended` in one server release, after the shared snapshot-loader change.
2. Dogfood 2 weeks on demo + internal + pilot courses; hand-score `outcomes.assessment-mapping` and
   `structure.*` findings for false positives.
3. Promotion gate per rule: manual review plausible **and** `disagree` dismissal rate < 10%. Promote the seven `essential` rules together with a product announcement, since this is the pack most likely to move a
   course's badge count.
4. Because these rules are the ones instructors will act on in bulk, ship CC.10's "suggest mappings" assist
   **before** promoting `outcomes.assessment-mapping` to `essential`.
5. Rollback: `RETIRED_ITEM_IDS` or tier demotion; both are server-only releases.

## 16. Test Plan

- **Unit** — Table-driven per rule (≥ 4 snapshots each). Bloom lexicon tests. Placeholder lexicon tests.
  Cycle detection. Evidence ordering and truncation. Shared-gradable-set test (AC-13).
- **Integration** — Seeded-DB tests for the new `DataNeeds` queries: outcome links across all four
  `target_kind` values including `quiz_question`; archived items excluded; standards tables absent.
- **End-to-end** — Playwright: seed a course with 5 assignments, map 2, assert the checklist shows 3
  evidence rows; click a row, assert it lands on that assignment's outcomes section; map it; return and
  assert the row is gone after recheck.
- **Security** — Assert no evidence row in this pack contains a user ID or name.
- **Accessibility** — Covered in CC.7, but CC.4 MUST supply column headers for every `EvidenceShape`
  (asserted by the registry integrity test).
- **Performance / load** — Benchmark on 300-item and 2,000-item fixtures; query-count assertion; cycle
  detection on a 500-node adversarial graph.
- **Manual exploratory** — Instructional-design review of all 22 items; run against the marketplace course
  `introduction-to-python` and hand-verify every finding.

## 17. Documentation & Training

- [course-design-research.md](course-design-research.md) B-sections: rubric mapping for all 22 rules.
- Help-centre article "Aligning assessments to outcomes" covering the mapping workflow, `measurement_level`
  semantics, and why both directions are checked.
- `docs/dev/course-checklist-engine.md` gains "evidence tables with per-row targets".
- Accreditation-facing note: the alignment evidence is FERPA-free by construction and can be exported.

## 18. Open Questions

1. Should surveys and ungraded practice items count as "gradable" for FR-17? Proposed: no — only
   `assignment` and `quiz` structure items with points > 0; ungraded practice is covered by
   `assessment.formative` in CC.5.
2. Is 3 the right floor for `outcomes.defined`? Proposed yes (QM expects module-level and course-level
   objectives); confirm with K-12 pilots where standards may replace outcomes entirely.
3. When `standards_alignment_enabled` is on, should `outcomes.defined` become `not_applicable` (standards
   *are* the outcomes)? Proposed: no, but lower its tier for K-12 orgs — needs a decision.
4. Should `outcomes.assessment-mapping` count a quiz as mapped when only some of its questions are mapped?
   Proposed: yes at quiz level, with a separate `recommended` rule for question-level coverage later.
5. Does FR-11 belong here or with the external link-health loader in CC.6? Proposed: internal file refs here
   (cheap, snapshot-only), external URLs in CC.6 (network, lazy).

## 19. References

- Existing files this work touches: `server/internal/service/coursechecklist/rules_structure.go` and
  `rules_outcomes.go` (new), `server/internal/repos/courseoutcomes/repo.go`,
  `server/internal/repos/coursestructure/`, `server/migrations/072_course_learning_outcomes.sql`,
  `server/migrations/074_course_outcome_link_levels.sql`.
- External standards: [Quality Matters](https://www.qualitymatters.org/qa-resources/rubric-standards/higher-ed-rubric)
  GS2 (Learning Objectives), GS3 (Assessment and Measurement), GS4 (Instructional Materials);
  [NSQ](https://nsqol.org/the-standards/quality-online-courses/) Standards B and C;
  [OSCQR](https://oscqr.suny.edu/) "Design and Layout" (16–28) and "Content and Activities" (29–37);
  [CAST UDL 3.0](https://udlguidelines.cast.org/) Representation.
- Related plans: [CC.1](../../completed/checklist/CC.1-checklist-registry-and-evaluation-engine.md),
  [CC.5](CC.5-rule-pack-assessment-feedback-interaction.md), [CC.8](CC.8-deep-link-and-highlight-targeting.md),
  [CC.10](CC.10-analytics-guidance-and-rollout.md).
