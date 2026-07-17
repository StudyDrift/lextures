# IQ.7 — Reports, Results & Gradebook / Analytics

> Completed implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](../../plan/interactive-quizzes/README.md) (active plans) / [completed index](README.md). Reuses item analysis (`itemanalysis`), grade sync (`coursegrades` / grade audit), and learner-progress plumbing already in the platform.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.7 |
| **Section** | Interactive Quizzes |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | SHIPPED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment squad + Analytics |
| **Depends on** | IQ.3 |
| **Unblocks** | — |

---

## 1. Problem Statement

A live quiz's value to *teaching* (not just fun) is what it reveals: which questions the class missed, which
students struggled, and whether it should nudge grades. Today those signals evaporate when the game ends. IQ.7
turns every game into **reports**: a per-game summary, per-question item analysis ("everyone got Q7 wrong —
reteach it"), per-student results, a "questions to review" list for each learner, CSV/PDF export, and an
optional push of scores into the **gradebook**. It reuses the platform's existing item-analysis and grade-sync
machinery rather than building analytics from scratch.

## 2. Goals

- A **post-game report** for instructors: participation, average/median score, per-question correctness &
  timing, hardest/easiest questions, and a full leaderboard.
- **Per-question item analysis** reusing `itemanalysis` (difficulty, distractor analysis, discrimination where
  applicable) so live games feed the same analytics as formal assessments.
- **Per-student results**: each learner's answers, correctness, points, and a personalised "review these"
  list; visible to the student (their own) and the instructor (all).
- Optional **gradebook push**: map a game/assignment score to a `coursegrades` item (raw, percentage, or
  participation), configurable and reversible.
- **Export** (CSV + printable/PDF) and **learner-progress** integration (mastery signals, at-risk inputs).

## 3. Non-Goals

- Real-time in-game leaderboard (IQ.5) — IQ.7 is post-game analysis and durable results.
- Building a new analytics engine — IQ.7 adapts existing item-analysis/grade-sync/learner-progress.
- Cross-course/longitudinal dashboards (admin analytics is IQ.11).

## 4. Personas & User Stories

- **As an instructor**, right after a game I want to see which questions the class missed, so I know what to
  reteach.
- **As an instructor**, I want to push the game score to my gradebook as a participation grade, so it counts.
- **As a student**, I want to review the questions I got wrong with the correct answers and explanations, so I
  learn from the game.
- **As an instructor**, I want to export results to CSV, so I can analyse or archive them.
- **As an advisor / the at-risk system**, I want game performance to feed the learner model, so struggling
  students surface early.

## 5. Functional Requirements

- **FR-1.** On game end, the system MUST compute and persist a **game report**: player count, completion,
  score distribution (avg/median/max), per-question aggregates (correct %, avg `response_ms`, answer
  distribution), and final rankings.
- **FR-2.** Per-question analysis MUST feed the existing `itemanalysis` pipeline keyed by the source question
  (where a kit question links to `course.questions` via `source_question_id`), so difficulty/distractor stats
  accumulate across live + formal use; kit-only questions get game-local stats.
- **FR-3.** Per-student results MUST be viewable: an instructor sees every student; a student sees **only their
  own** answers, correctness, points, and a "questions to review" list (their incorrect/slow items with the
  correct answer + explanation).
- **FR-4.** The instructor MUST be able to **push to gradebook**: create/update a `coursegrades` item from a
  game or assignment (IQ.6) with a configurable mapping — `raw_points`, `percent_correct`, or
  `participation` (answered ≥ X%); the push MUST be idempotent and reversible (unlink without deleting the
  game).
- **FR-5.** Gradebook writes MUST go through the existing grade path (`quiz_grade_sync.go`/`coursegrades`) and
  respect existing gradebook permissions, posting policies, and audit (`gradeauditevents`).
- **FR-6.** For async/homework games (IQ.6), the gradebook item MUST reflect the attempt policy
  (best/last/average) already resolved in IQ.6.
- **FR-7.** The system MUST export a game report as **CSV** (per-student rows + per-question columns) and a
  printable **PDF/HTML** summary; exports respect the requester's scope (instructor = full; student = own).
- **FR-8.** Game performance SHOULD emit signals to **learner progress / mastery** (`learnerprogress`,
  `masteryheatmap`) and, where standards-aligned questions are used, to outcomes reporting.
- **FR-9.** Reports MUST be available for a retained window and then aged/anonymised per IQ.11/S02; a student's
  DSAR export MUST include their game responses/results.
- **FR-10.** Guest (non-enrolled) player results MUST be included in the instructor's game report but MUST NOT
  create gradebook rows or learner-model records (no linked identity).
- **FR-11.** All aggregates MUST be reproducible from the raw `session_responses` + scoring version (IQ.5), so
  a recomputation matches the stored report.

## 6. Non-Functional Requirements

- **Performance** — report generation for a 200-player, 30-question game < 3 s; CSV export streamed.
- **Security** — strict scoping (student sees only own results); gradebook writes permission-checked; export
  authorized.
- **Privacy & Compliance** — results are education records; student view is self-only; DSAR/retention honoured;
  guest results excluded from persistent per-student records.
- **Accessibility** — report tables are semantic, sortable, screen-reader friendly; charts have data-table
  equivalents (per the dataviz standards); PDF is tagged.
- **Scalability** — aggregates computed incrementally during the game where possible; heavy reports run in a
  job for very large games.
- **Reliability** — report derives from durable `session_responses`; recompute is deterministic; gradebook
  push idempotent.
- **Observability** — counters: reports generated, gradebook pushes, exports; timing of report jobs.
- **Maintainability** — one report builder over the shared response tables; reuse item-analysis adapters.
- **Internationalization** — report labels localised; number/date formatting locale-aware; CSV UTF-8 BOM safe.
- **Backward compatibility** — additive; item-analysis contribution is additive to existing stats.

## 7. Acceptance Criteria

- **AC-1.** *Given* a finished game, *when* the instructor opens its report, *then* they see participation,
  score distribution, per-question correctness/timing, and the full leaderboard.
- **AC-2.** *Given* a kit question linked to a bank item, *when* the game ends, *then* that item's
  `itemanalysis` difficulty stats include the game's responses.
- **AC-3.** *Given* a student opens their results, *when* the page loads, *then* they see only their own
  answers and a "review these" list with correct answers/explanations — not classmates' data.
- **AC-4.** *Given* the instructor pushes the game to the gradebook as `participation`, *when* it completes,
  *then* a `coursegrades` item exists with the right values and an audit event; unlinking removes the item
  without deleting the game.
- **AC-5.** *Given* an async assignment with `policy=best`, *when* pushed, *then* the gradebook reflects each
  student's best attempt.
- **AC-6.** *Given* a CSV export request, *when* generated, *then* it contains one row per student and one
  column per question with correctness/points, scoped to the requester.
- **AC-7.** *Given* a guest player, *when* the report renders, *then* their nickname appears in the game report
  but no gradebook/learner-model row is created for them.
- **AC-8.** *Given* a recomputation, *when* run against stored responses + scoring version, *then* it matches
  the persisted report exactly.

## 8. Data Model

Migration `401_interactive_quizzes_reports.sql` (plan originally said `396`, already taken by player client):

```sql
CREATE TABLE quizgame.game_reports (
  session_id     UUID PRIMARY KEY REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
  player_count   INTEGER NOT NULL,
  answered_count INTEGER NOT NULL,
  score_avg      NUMERIC(10,2),
  score_median   NUMERIC(10,2),
  score_max      INTEGER,
  per_question   JSONB NOT NULL DEFAULT '[]'::jsonb, -- [{index, correctPct, avgMs, distribution}]
  generated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE quizgame.gradebook_links (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id        UUID REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
  assignment_id     UUID REFERENCES quizgame.assignments (id) ON DELETE CASCADE, -- IQ.6
  course_id         UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
  gradebook_item_id UUID NOT NULL,           -- coursegrades item
  mapping           TEXT NOT NULL DEFAULT 'participation', -- raw_points | percent_correct | participation
  points_possible   NUMERIC(6,2),
  created_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (session_id IS NOT NULL OR assignment_id IS NOT NULL)
);
CREATE INDEX idx_quizgame_gradebook_links_course ON quizgame.gradebook_links (course_id);
```

- The report is a **cache** of what's derivable from `session_responses` (FR-11); it can be dropped/rebuilt.
- Gradebook rows themselves live in `coursegrades` (not duplicated here); `gradebook_links` records the linkage
  for idempotent push/unlink.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| GET | `/live-quizzes/games/{game_id}/report` | host/instructor |
| GET | `/live-quizzes/games/{game_id}/my-results` | enrolled player (self) |
| GET | `/live-quizzes/games/{game_id}/report/export?format=csv\|pdf` | scoped |
| POST | `/live-quizzes/games/{game_id}/gradebook-link` `{mapping, pointsPossible}` | grade permission |
| DELETE | `/live-quizzes/games/{game_id}/gradebook-link/{id}` | grade permission |
| POST | `/live-quizzes/games/{game_id}/report/rebuild` | instructor |

- **OpenAPI:** document report shape, export formats, gradebook-link mapping.
- **Rate-limit:** exports rate-limited; large exports run as jobs and return a download link.

## 10. UI / UX

- **Report page** `clients/web/src/pages/lms/live-quiz-report-page.tsx`: summary tiles, per-question bars
  (with data-table equivalents per dataviz standards), "hardest questions" callouts, leaderboard, per-student
  table (sortable, searchable), export + "Push to gradebook" actions.
- **Student results page** `live-quiz-my-results-page.tsx`: score, rank, and a "review these" accordion with
  each missed/slow question, correct answer, and explanation.
- **Gradebook-link dialog:** mapping picker (participation/percent/raw), points-possible, preview of what each
  student would receive, and an unlink control.
- **Flows:** end game → report auto-opens → (optional) push to gradebook → export; student opens their results
  from the game or their grades.
- **States:** report generating (job), empty (no players), guest-only note, gradebook already-linked, export
  preparing.
- **Accessibility:** every chart has an accessible data table; tables sortable via keyboard; PDF tagged.
- **Copy & i18n:** `liveQuiz.report.*`, `liveQuiz.myResults.*`, `liveQuiz.gradebook.*`.

## 11. AI / ML Considerations

Optional (deferred): an AI "reteach suggestion" summarising the hardest questions could reuse the IQ.10/AP
provider path; the "questions to review" list itself is deterministic (no AI required).

## 12. Integration Points

- **Reuse:** `itemanalysis` (per-question stats), `quiz_grade_sync.go` + `coursegrades` (gradebook),
  `gradeauditevents` (audit), `learnerprogress` / `masteryheatmap` (mastery signals), `courseoutcomes` /
  outcomes reporting (standards-aligned questions), export/PDF utilities used elsewhere.
- **Server new:** `repos/quizgame/reports.go`, `httpserver/quizgame_reports.go`, a report-build job in
  `background/`, item-analysis adapter.
- **Web new:** report + my-results pages, gradebook-link dialog, export.

## 13. Dependencies & Sequencing

- Must ship after: IQ.3 (responses exist); benefits from IQ.5 (points) and IQ.6 (async attempt policy).
- Must ship before: nothing hard-depends.
- Shared infra: gradebook, item analysis, export/PDF, job runner.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Student sees classmates' data via report endpoint | L | H | Strict self-scoping on `/my-results`; authz tests |
| Item-analysis pollution from low-stakes games | M | M | Tag game responses; allow excluding game data from formal item stats via a flag |
| Gradebook double-push / drift | M | H | Idempotent `gradebook_links`; reversible unlink; audit events |
| Large-game report blocks a request | M | M | Job-based build + cached report; rebuild endpoint |
| Guest data leaking into learner model | M | M | Guests excluded from gradebook/learner records by identity check |

## 15. Rollout Plan

- **Flag:** `interactive_quizzes_enabled`; gradebook push behind a sub-flag until verified against known
  scores.
- **Sequencing:** migration `401` → report build + pages → export → gradebook link.
- **Dogfood:** run a game, verify report matches manual counts, push to a test gradebook, export CSV.
- **GA criteria:** AC-1..AC-8 pass; recompute==stored; gradebook values verified.
- **Rollback:** disable gradebook-push sub-flag (reports still work); links retained.

## 16. Test Plan

- **Unit** — aggregate math (avg/median/correct%/avgMs); mapping computations; recompute determinism.
- **Integration** — item-analysis contribution; gradebook idempotent push/unlink + audit; async best/last/avg.
- **End-to-end** — Playwright: finish game → report → push to gradebook → student sees own results → CSV.
- **Security** — student cannot access others' results or the full report; export scoping; grade permissions.
- **Accessibility** — chart data-tables; sortable tables; tagged PDF.
- **Performance** — 200×30 report build time; streamed CSV.
- **Manual** — guest-only game report; unlink after grades posted.

## 17. Documentation & Training

- Instructor: "Read your game report", "Push a game to the gradebook", exporting.
- Student: "Review your quiz-game results".
- API reference: report/export/gradebook-link endpoints.
- Runbook: report job, recompute, item-analysis tagging, gradebook linkage.

## 18. Open Questions

1. Should low-stakes live games contribute to formal item-analysis by default? (Recommendation: contribute but
   tagged, with an instructor/admin toggle to exclude.)
2. Default gradebook mapping — participation or percent-correct? (Recommendation: `participation` for live
   classic; `percent_correct` for async homework.)
3. How long are game reports retained before anonymisation? (Recommendation: align with the course's
   assessment retention policy via IQ.11/S02.)

## 19. References

- Existing files: `server/internal/repos/itemanalysis/`, `server/internal/httpserver/quiz_grade_sync.go`,
  `server/internal/httpserver/quiz_analytics.go`, `server/internal/repos/coursegrades/`,
  `server/internal/repos/gradeauditevents/`, `server/internal/repos/learnerprogress/`,
  `server/internal/repos/masteryheatmap/`.
- Related plans: [IQ.3](IQ.3-live-game-hosting-engine.md), [IQ.5](IQ.5-scoring-leaderboards-mechanics.md),
  [IQ.6](IQ.6-game-modes-team-paced-async.md), [IQ.11](../../plan/interactive-quizzes/IQ.11-admin-governance-quotas-lifecycle.md).
