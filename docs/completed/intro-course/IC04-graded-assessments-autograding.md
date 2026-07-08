# IC04 — Graded Assessments & Automated Grading

> Implementation plan. Source: product direction — *"For all the students, they will be given
> grades and assignments."* Follows [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IC04 |
| **Section** | Intro Course |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | IC01 (course + assignment groups), IC03 (items to grade) |
| **Unblocks** | IC05 (completion is computed from these grades) |

---

## 1. Problem Statement

Students in the intro course must be *given grades and assignments*, but there is no human
instructor to grade at platform scale. This plan makes the intro course's quizzes and assignments
real, gradable items whose scores are produced **automatically** — quizzes auto-score against
their answer keys, and assignments auto-complete to full credit on submission (optionally routed
through the grader agent when enabled). The result is a genuine, populated gradebook for every
student, using the platform's real grading machinery so nothing about the experience is a mock.

## 2. Goals

- Attach **graded items** to IC03's structure: one auto-scored knowledge-check quiz per module,
  plus a small number of assignments (a "try it" task and a capstone reflection).
- Score quizzes **automatically** via the existing quiz-attempt scoring, writing points to
  `course.course_grades`.
- Grade assignments **automatically** on submission — default *completion → full points* — with an
  **optional** grader-agent path when `grader_agent_enabled`, so no human grading is required.
- Configure **assignment groups, weights, and a grading scale** so the gradebook shows a coherent
  running grade and a final grade at completion.
- Keep everything **idempotent and replayable** alongside IC03's content sync.

## 3. Non-Goals

- Authoring the *content* of pages/quizzes (IC03 owns fixtures; this plan owns their grading
  config: points, groups, due dates, answer keys' scoring semantics).
- Completion detection, credential, or progress UI (IC05).
- Human/instructor grading workflows, SpeedGrader, moderation, or rubric authoring beyond a
  simple auto-applied rubric (out of scope; the course is instructor-free).
- New grading infrastructure — reuses `course.course_grades`, quiz scoring, and the grader agent.

## 4. Personas & User Stories

- **As a student**, I want my quiz answers and submitted tasks to immediately show a grade, so the
  course feels real and I get feedback.
- **As a student**, I want a visible running grade and a final grade, so completing the course is
  rewarding and legible.
- **As the platform**, I want these grades produced with zero human effort, so the course scales
  to every user.
- **As a self-learner**, I want low-stakes grading (retries allowed, generous credit) so the
  intro course teaches without punishing.

## 5. Functional Requirements

- **FR-1.** Each module MUST have one **auto-scored quiz** (IC03 fixture) with a point value; on
  attempt submission the existing quiz scorer MUST compute the score and persist points to
  `course.course_grades` for `(student, module_item)`.
- **FR-2.** The course MUST include **assignments** (≥2): at least one interactive "try it" task
  and one **capstone reflection** (text entry). Assignment bodies/settings are synced as fixtures
  (extends IC03's sync to `course.module_assignments`).
- **FR-3.** Assignment grading MUST be **automatic**. Default policy: on a valid submission
  (text/URL/file per the assignment's `submission_allow_*` flags), award **full points**
  (completion-based) and write to `course.course_grades`. This MUST require no instructor action.
- **FR-4.** When `grader_agent_enabled` (and text-entry grading enabled), the capstone reflection
  MAY be routed through the **grader agent** for a scored/feedback result instead of flat
  completion; failure MUST fall back to completion-full-credit (never leave ungraded).
- **FR-5.** The course MUST define **assignment groups + weights** (from IC01 defaults, refined
  here) and a **grading scale**, so the gradebook computes a running and final percentage/letter.
- **FR-6.** Grading config (points, group, due date, submission modes, answer scoring) MUST be
  synced **idempotently** with IC03's content sync and versioned; re-sync MUST NOT wipe existing
  student grades for unchanged items.
- **FR-7.** Quizzes SHOULD allow **multiple attempts** (keep-highest) with immediate feedback,
  suiting a low-stakes onboarding context.
- **FR-8.** Due dates SHOULD be **relative/rolling** (e.g. "no hard due date" or "N days after
  enrollment") so a shared course never shows every student as "overdue"; if set, they MUST be
  soft (no penalty).
- **FR-9.** All grade writes MUST flow through the normal gradebook path so existing gradebook,
  what-if grades, and analytics surfaces reflect them without special-casing.

## 6. Non-Functional Requirements

- **Performance** — Auto-grade on submit completes synchronously in < 200 ms for quizzes and
  completion assignments; grader-agent path is async (queued) with a pending state. Gradebook read
  reuses existing queries (no new hot path).
- **Security** — A student may only submit/score their own attempts; auto-grade runs server-side
  (a client cannot self-award points). Answer keys MUST NOT be exposed to the client before
  submission. Grader-agent runs under existing authz/cost controls.
- **Privacy & Compliance** — Grades are FERPA education records — exported/erased via existing
  machinery. Grader-agent submission of reflection text follows existing AI-disclosure + PII
  handling; disclose AI grading when used.
- **Accessibility** — Quiz and assignment submission UIs are the existing WCAG 2.1 AA components;
  feedback states are screen-reader announced.
- **Scalability** — One shared course means `course.course_grades` gains ~ (items × students)
  rows. Bounded by student count; indexed by `(student, item)` already. Completion grading is O(1)
  per submit. Grader-agent usage is capped (v1: only the single capstone, only when enabled).
- **Reliability** — Auto-grade is idempotent per `(student, item)` (upsert); grader-agent has a
  guaranteed completion-credit fallback so no item is ever stuck ungraded.
- **Observability** — `intro_course_autograde_total{item_type,result}`,
  `intro_course_grade_write_total`, grader-agent-fallback counter. Log at DEBUG per grade write
  (hashed student id, item, points).
- **Maintainability** — Grading config co-located with content fixtures (front-matter fields:
  `points`, `group`, `submission_modes`, `attempts`, `due_offset_days`). No bespoke grader.
- **Internationalization** — Feedback strings are i18n keys (IC08).
- **Backward compatibility** — Additive; reuses existing grade tables and quiz/assignment models.

## 7. Acceptance Criteria

- **AC-1.** *Given* a student submits a module quiz, *when* scoring runs, *then* a
  `course.course_grades` row for `(student, quiz_item)` is written with the computed points and is
  visible in the gradebook.
- **AC-2.** *Given* a student submits the capstone reflection (text), *when* the flag path is
  completion-based, *then* full points are awarded automatically with no instructor action.
- **AC-3.** *Given* `grader_agent_enabled` and text-entry grading on, *when* the capstone is
  submitted, *then* it is routed to the grader agent; *and given* the agent errors, *then* the
  student still receives completion full-credit (fallback).
- **AC-4.** *Given* a student completes all quizzes and assignments, *when* the gradebook computes
  the final grade, *then* it reflects the configured group weights and grading scale.
- **AC-5.** *Given* a content re-sync with unchanged grading config, *when* it runs, *then* no
  existing student grades are altered or deleted.
- **AC-6.** *Given* a client attempts to POST a grade directly, *when* the request is made, *then*
  it is rejected (grades are server-computed only).
- **AC-7.** *Given* quizzes allow multiple attempts, *when* a student retakes and scores higher,
  *then* the kept grade is the highest attempt.

## 8. Data Model

No new grade schema — reuses:
`course.course_grades` (points per student per gradable item),
`course.assignment_groups` (+ `weight_percent`), `course.courses.grading_scale`,
`course.module_quizzes` (`questions_json` answer keys + delivery settings),
`course.module_assignments` (`available_from/until`, `submission_allow_*`, points/due via existing
assignment delivery settings).

Grading config is carried in the IC03 fixture front-matter and mapped by
`settings.intro_course_items.slug`. If a per-item "auto-grade policy" needs persistence beyond
existing columns, add:

```sql
-- server/migrations/373_intro_course_grading.sql  (renumber on merge)
ALTER TABLE settings.intro_course_items
    ADD COLUMN grade_policy TEXT;   -- 'quiz_autoscore' | 'completion_full' | 'grader_agent'
```

Assignment-group defaults (refined from IC01): `Quizzes` 50%, `Assignments` 40%,
`Participation` 10% (participation auto-credited by reading all pages — computed in IC05).

## 9. API Surface

No new endpoints — submissions/scoring go through existing quiz-attempt and assignment-submission
routes; grades read via the existing gradebook API. Internal hooks:

```go
// server/internal/service/introcourse/grade.go
func OnQuizAttempt(ctx, exec, studentID, itemID, attempt) error      // upsert highest score
func OnAssignmentSubmit(ctx, exec, studentID, itemID, submission) error // completion or grader-agent
func SyncGradingConfig(ctx, tx, courseID) error                       // groups/weights/scale/policies
```

`OnAssignmentSubmit` hooks the existing submission handler for the intro course id only, so other
courses are unaffected.

## 10. UI / UX

Reuses existing quiz, assignment-submission, feedback, and gradebook UIs. Additions:

1. Immediate **auto-feedback** after quiz submit (existing quiz result view) and an "Assignment
   received — full credit" confirmation for completion assignments.
2. A visible **running grade** in the intro course (existing student grades view).
3. The capstone reflection shows a "graded automatically" note (and "AI-assisted feedback" when
   the grader-agent path is used, with disclosure).

Empty/pending states: grader-agent capstone shows "grading…" until the async result or fallback
lands.

## 11. AI / ML Considerations

- **Optional** grader-agent grading of the single capstone reflection when `grader_agent_enabled`
  + `grader_agent_text_entry_grading_enabled`. Prompt/eval reuse the existing grader-agent
  service; **fallback path** = completion full-credit on any error/timeout/cost-cap. PII: student
  reflection text handled per existing grader-agent privacy rules; AI disclosure shown. Cost
  budget: capped to one item per student, only when the flag is on.

## 12. Integration Points

- **Grading:** `course.course_grades` + existing gradebook service; quiz scorer
  (`course.module_quizzes`); assignment submissions (`course.module_assignment_submissions`,
  migration 098).
- **Grader agent:** existing grader-agent service + flags (`grader_agent_enabled`,
  `grader_agent_text_entry_grading_enabled`, run filters/cost estimate).
- **Content sync:** extends IC03 `content_sync` with grading config (`SyncGradingConfig`).
- **Analytics/what-if:** existing gradebook, what-if grades (`ff_whatif_grades`) read the grades
  with no change.

## 13. Dependencies & Sequencing

- **After:** IC01 (assignment groups/scale), IC03 (items to grade).
- **Before:** IC05 (completion computed from grades/progress).
- **Shared infra:** quiz engine, gradebook, submissions, optional grader agent — present.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Grader agent cost/latency on every student | M | H | Only 1 capstone, only when flag on; async + cost cap; completion fallback |
| Shared-course due dates mark everyone overdue | H | M | No hard due dates / relative soft offsets (FR-8) |
| Client forges a grade | L | H | Server-side scoring only; reject client grade writes (AC-6) |
| Re-sync wipes student grades | L | H | Idempotent config sync; never delete items with grades (IC03 FR-6) |
| `course_grades` row growth (items × students) | M | M | Bounded, indexed; monitor; acceptable for onboarding-sized item count |

## 15. Rollout Plan

- **Flag:** gated by `intro_course_enabled`; grader-agent path additionally by
  `grader_agent_enabled` (else completion-only).
- **Sequencing:** define grading config + groups/weights → wire quiz/assignment auto-grade hooks →
  verify populated gradebook on staging → enable grader-agent path on staging (validate fallback)
  → prod.
- **Dogfood:** internal students complete the course; verify running/final grades and gradebook.
- **GA criteria:** every item auto-grades; final grade computes per weights; grader-agent fallback
  proven; no client-forgeable grades.
- **Rollback:** disable grader-agent path (completion-only) without disabling the course; or
  disable `intro_course_enabled`.

## 16. Test Plan

- **Unit** — quiz score → points mapping; completion-full award; grader-agent fallback; weight
  math; keep-highest attempts.
- **Integration (DB)** — submit quiz/assignment → grade rows written; re-sync preserves grades;
  final-grade computation across groups.
- **End-to-end** — student completes all items → populated gradebook with correct final grade;
  grader-agent capstone happy path + forced-error fallback.
- **Security** — direct grade POST rejected; answer keys not leaked pre-submit; cross-student
  scoring blocked.
- **Accessibility** — feedback/pending states announced; gradebook navigable.
- **Performance** — auto-grade latency targets; grader-agent async under cost cap.
- **Manual exploratory** — retake quizzes; submit empty/oversized assignments; toggle grader-agent
  flag mid-course.

## 17. Documentation & Training

- Update `docs/guides/intro-course-content.md` with grading front-matter fields (`points`,
  `group`, `submission_modes`, `attempts`, `due_offset_days`, `grade_policy`).
- Admin doc: "intro-course grades are automatic; enabling the grader agent adds AI feedback on the
  capstone."
- Runbook: how to inspect/repair a student's intro-course grades.

## 18. Open Questions

1. Group weights: final values for `Quizzes` / `Assignments` / `Participation`? (Proposed
   50/40/10; product to confirm.)
2. Should the capstone always be completion-credit, with grader-agent producing *feedback only*
   (not affecting the grade)? (Leaning: feedback-only when enabled, to keep grades deterministic.)
3. Multiple attempts: keep-highest vs. keep-latest for quizzes? (Leaning keep-highest.)
4. Do we want any *failing* path (a student who submits nothing) to leave the course incomplete,
   or auto-credit participation for reading? (Ties to IC05 completion definition.)

## 19. References

- Existing files: `server/migrations/067_course_grades.sql`, `.../027_course_grading.sql`,
  `.../098_module_assignment_submissions.sql`, `.../063_module_assignment_delivery_settings.sql`,
  `.../033_module_quizzes.sql`, grader-agent handlers (`server/internal/httpserver/grading_agent_*`).
- Related plans: [IC03](IC03-curriculum-content.md), [IC05](IC05-progress-completion-credential.md),
  completed what-if grades / grade curving under `docs/completed/03-submissions-grading-integrity/`.
