# IQ.6 — Game Modes: Team, Student-Paced & Async Homework

> Implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](README.md). Extends the state machine of [IQ.3](../../completed/interactive-quizzes/IQ.3-live-game-hosting-engine.md), the player flows of [IQ.4](../../completed/interactive-quizzes/IQ.4-player-join-and-gameplay.md), and the scoring of [IQ.5](IQ.5-scoring-leaderboards-mechanics.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.6 |
| **Section** | Interactive Quizzes |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Assessment squad |
| **Depends on** | IQ.3, IQ.4 |
| **Unblocks** | (enhances IQ.7) |

---

## 1. Problem Statement

The flagship "host drives every question live" mode fits a synchronous classroom, but teachers need more: a
**team mode** for collaborative play, a **student-paced** mode where each learner moves through the same
questions at their own speed (great for stations or mixed-pace rooms), and an **async homework** mode where a
kit is assigned with a due date and students play any time — with results still flowing to the gradebook. IQ.6
adds these three modes on top of the IQ.3 engine so one kit serves the whole spectrum from party-game to
graded homework.

## 2. Goals

- **Team mode:** players are grouped into named teams; answers/scores roll up to a team leaderboard while
  individual responses are still recorded (for gradebook + item analysis).
- **Student-paced mode:** a hosted game where each player advances through the question set independently (host
  sees aggregate progress); optional shuffle and time budget.
- **Async homework mode:** assign a kit with open/close windows and attempt rules; students play solo any time;
  results feed IQ.7/gradebook like a module quiz — but with the game feedback loop.
- Reuse one engine, one scoring model, and one report pipeline across all modes (mode is a parameter, not a
  fork).
- Clear instructor setup for each mode with sensible defaults.

## 3. Non-Goals

- New question types (IQ.2) or new scoring formulae (IQ.5 — team aggregation is defined here but reuses per-
  player points).
- Replacing the existing module-quiz homework engine — async mode is the *game-flavoured* option, not a
  migration of `coursemodulequizzes`.
- Cross-class tournaments/ladders (a possible future story).

## 4. Personas & User Stories

- **As an instructor**, I want to split the class into teams and see a team leaderboard, so collaboration is
  rewarded.
- **As an instructor with a mixed-pace class**, I want each student to move at their own speed through the same
  questions, so nobody waits or feels rushed.
- **As an instructor**, I want to assign a quiz kit as homework due Friday, so students play at home and I get
  a gradebook column.
- **As a student**, I want to play the homework game on my own time and still get the fun feedback and my score.
- **As a student on a team**, I want to see my team's standing and my own contribution.

## 5. Functional Requirements

- **FR-1.** Games MUST carry a `mode ∈ {live_classic, team, student_paced, homework}` (enum already on
  `quizgame.sessions` from IQ.3); the engine branches behaviour on mode without duplicating the reducer.
- **FR-2. Team mode:** the host MUST be able to create teams (named, or auto-balanced), assign/auto-assign
  players, and the leaderboard MUST rank **teams** by aggregated score (sum or average, configurable) while
  `session_responses` still records each individual. Team membership persists on `session_players.team_id`.
- **FR-3.** In team mode, the answer rule MUST be configurable: `each_member_answers` (each scores; team =
  sum/avg) or `one_device_per_team` (a shared team device submits once). Default `each_member_answers`.
- **FR-4. Student-paced mode:** on start, each player MUST receive the full (optionally shuffled) question set
  and advance independently; per-question timing is still server-measured from *that player's* question-open;
  the host view MUST show aggregate progress (e.g. "18/25 finished Q4") and may end the round for everyone.
- **FR-5.** Student-paced MUST support an optional overall **time budget** and optional per-question timers;
  when time expires, remaining questions auto-lock for that player.
- **FR-6. Async homework mode:** an **assignment** MUST bind a kit to a course with `opens_at`, `due_at`,
  `closes_at`, `attempts_allowed`, and `shuffle`; it appears in the student's to-do/assignments surface.
- **FR-7.** Async play MUST create a per-student session (or a lightweight per-student run) that records
  responses, scores via IQ.5, and on completion writes to IQ.7/gradebook honouring `attempts_allowed`
  (best/last/average configurable, reusing the module-quiz grade policy conventions).
- **FR-8.** Async mode MUST enforce windows: no play before `opens_at`, late attempts flagged after `due_at`,
  no play after `closes_at`; accommodations/time extensions MUST honour the existing overrides engines
  (`assignmentoverrides`, `enrollmentquizzesoverrides`).
- **FR-9.** All modes MUST reuse the same **reports** (IQ.7): per-question item analysis, per-student results,
  and "questions to review" work regardless of mode.
- **FR-10.** Mode selection and its config MUST be set at game/assignment creation and immutable once players
  have joined/started (to keep scoring reproducible).
- **FR-11.** Guest players MUST be disallowed in `homework` mode (identity required for grading); allowed in
  `team`/`student_paced` only if the game permits and IQ.9 rules pass.

## 6. Non-Functional Requirements

- **Performance** — team leaderboard aggregation O(players); student-paced supports 200 independent players
  without central-clock contention; async play is ordinary request/response (no room needed at rest).
- **Security** — async windows/attempts enforced server-side; accommodations authorized; team assignment is
  host-controlled.
- **Privacy & Compliance** — async results are graded education records (FERPA); windows/overrides audited;
  homework attempts retained per policy.
- **Accessibility** — team and paced UIs meet the same AA bar as IQ.4; async play is self-paced (kinder to AT
  users); no colour-only team encoding.
- **Scalability** — student-paced/async avoid a single synchronized clock, easing fan-out; async scales like
  normal quiz traffic.
- **Reliability** — async attempts resumable (save progress); team membership survives reconnect; window
  checks idempotent.
- **Observability** — per-mode metrics: teams/game, paced completion distribution, async attempt counts, late
  submissions.
- **Maintainability** — mode is a strategy object over the shared engine; no parallel code paths for scoring or
  reporting.
- **Internationalization** — team names/UI localised; due dates in the student's timezone.
- **Backward compatibility** — additive; `live_classic` behaviour unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* team mode with 4 teams, *when* members answer, *then* the leaderboard ranks teams by the
  configured aggregate, and individual responses are still recorded per player.
- **AC-2.** *Given* `one_device_per_team`, *when* the team device submits, *then* the team scores once and no
  per-member duplicate is created.
- **AC-3.** *Given* student-paced mode, *when* two students play, *then* each advances independently and the
  host sees aggregate progress, not a single shared question.
- **AC-4.** *Given* a student-paced time budget, *when* it expires, *then* that student's remaining questions
  auto-lock and their game finalises.
- **AC-5.** *Given* an async assignment opening tomorrow, *when* a student tries today, *then* play is refused
  as not-yet-open.
- **AC-6.** *Given* async `attempts_allowed=2, policy=best`, *when* a student plays twice, *then* the higher
  score is written to the gradebook.
- **AC-7.** *Given* a student with a time-extension accommodation, *when* they play the async game, *then* the
  extended window/timers apply.
- **AC-8.** *Given* any mode, *when* the game ends, *then* IQ.7 reports render identically (per-question + per-
  student).

## 8. Data Model

Migration `395_interactive_quizzes_modes.sql`:

```sql
CREATE TABLE quizgame.teams (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  color      TEXT,
  total_score INTEGER NOT NULL DEFAULT 0,
  UNIQUE (session_id, name)
);
-- session_players.team_id (declared in IQ.3) references quizgame.teams(id) logically.

CREATE TABLE quizgame.assignments (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kit_id           UUID NOT NULL REFERENCES quizgame.kits (id) ON DELETE RESTRICT,
  course_id        UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
  title            TEXT NOT NULL,
  opens_at         TIMESTAMPTZ,
  due_at           TIMESTAMPTZ,
  closes_at        TIMESTAMPTZ,
  attempts_allowed INTEGER NOT NULL DEFAULT 1,
  grade_policy     TEXT NOT NULL DEFAULT 'best',  -- best | last | average
  shuffle          BOOLEAN NOT NULL DEFAULT TRUE,
  points_possible  NUMERIC(6,2),                  -- gradebook mapping (IQ.7)
  gradebook_item_id UUID,                         -- link to coursegrades item
  created_by       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_quizgame_assignments_course ON quizgame.assignments (course_id, due_at);

CREATE TABLE quizgame.assignment_attempts (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  assignment_id UUID NOT NULL REFERENCES quizgame.assignments (id) ON DELETE CASCADE,
  user_id       UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
  session_id    UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE, -- per-attempt run
  attempt_no    INTEGER NOT NULL,
  score         INTEGER NOT NULL DEFAULT 0,
  submitted_at  TIMESTAMPTZ,
  is_late       BOOLEAN NOT NULL DEFAULT FALSE,
  UNIQUE (assignment_id, user_id, attempt_no)
);
```

- Async attempts reuse `quizgame.sessions` (mode `homework`, single player) so scoring/reporting are shared.
- Overrides/accommodations resolved via existing `assignmentoverrides`/`enrollmentquizzesoverrides`.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| POST | `/live-quizzes/games` `{mode, teamConfig?, pacedConfig?}` | host |
| POST | `/live-quizzes/games/{game_id}/teams` / `.../teams/assign` | host |
| POST | `/live-quizzes/kits/{kit_id}/assignments` | `item:create` |
| GET | `/live-quizzes/assignments/{id}` / `.../my-attempts` | enrolled |
| POST | `/live-quizzes/assignments/{id}/start` → creates attempt session | enrolled (window-checked) |
| POST | `/live-quizzes/assignments/{id}/attempts/{aid}/submit` | enrolled |

- **WS:** student-paced reuses the player WS but with per-player question advancement; team mode adds
  `team.leaderboard` frames.
- **OpenAPI:** document mode configs, assignment CRUD, window/attempt rules.

## 10. UI / UX

- **Host start dialog:** mode selector (Live / Team / Student-paced / Homework) with mode-specific config
  (team count & aggregate, paced time budget, homework windows/attempts/policy).
- **Team UI:** team assignment screen (drag/auto-balance), team-coloured player badges, team leaderboard/podium.
- **Student-paced host view:** aggregate progress grid; "end round" control.
- **Async student UI:** the kit appears in assignments/to-do; a "Play" button within the window; attempt
  history; results after submit. Reuses the assignment card patterns already in the app.
- **States:** not-yet-open, closed, out-of-attempts, in-progress-resumable, late, graded.
- **Accessibility:** team colour is paired with a team name/label; async play is self-paced and AT-friendly.
- **Copy & i18n:** `liveQuiz.mode.*`, `liveQuiz.team.*`, `liveQuiz.assignment.*`.

## 11. AI / ML Considerations

Not AI-touching. (Auto-balancing teams by prior performance is a possible future ML nicety, out of scope.)

## 12. Integration Points

- **Reuse:** IQ.3 engine (mode strategy), IQ.5 scoring, IQ.7 reports/gradebook, `assignmentoverrides` /
  `enrollmentquizzesoverrides` (accommodations), `studenttodos` / assignments surface, `coursegrades`.
- **Server new:** `repos/quizgame/{teams,assignments,attempts}.go`, mode strategies in the engine, window/
  attempt enforcement, `httpserver/quizgame_assignments.go`.
- **Web new:** mode selector, team assignment UI, async assignment + attempt UI.

## 13. Dependencies & Sequencing

- Must ship after: IQ.3, IQ.4 (engine + player), and needs IQ.5 for scoring, IQ.7 for grading of async.
- Must ship before: nothing hard-depends; enriches IQ.7.
- Shared infra: gradebook, overrides engines, assignments/to-do surface.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Mode branching forks the engine into spaghetti | M | H | Mode = strategy object; scoring/report/reducer stay shared; tests per mode |
| Student-paced loses the central clock's anti-cheat benefit | M | M | Still server-measured per player; shuffle; async anti-cheat settings (IQ.9) |
| Async grading conflicts with module-quiz gradebook conventions | M | M | Reuse `coursegrades` item + best/last/average policy already used by quizzes |
| Team aggregation ambiguity (sum vs avg with uneven teams) | M | M | Configurable aggregate; default average to be fair to smaller teams |
| Window/accommodation bugs cause unfair lockouts | M | H | Reuse the proven overrides engines; explicit window tests |

## 15. Rollout Plan

- **Flag:** `interactive_quizzes_enabled`; each mode behind a small sub-flag so they can ship incrementally
  (team → student-paced → homework).
- **Sequencing:** migration `395` → team mode → student-paced → async homework (needs IQ.7 gradebook path).
- **Dogfood:** run one game in each mode; assign a homework kit and grade it.
- **GA criteria:** AC-1..AC-8 pass; async gradebook write verified against a known score.
- **Rollback:** disable a mode's sub-flag; `live_classic` unaffected.

## 16. Test Plan

- **Unit** — team aggregation (sum/avg, uneven teams); paced per-player timing; window/attempt/policy logic.
- **Integration** — team assignment + leaderboard; paced independent advancement; async window enforcement +
  accommodations + best/last/average gradebook write.
- **End-to-end** — Playwright: team game to team podium; two students paced independently; async assign →
  play twice → gradebook shows best.
- **Security** — async windows/attempts/accommodations enforced server-side; guest blocked in homework.
- **Accessibility** — team labels not colour-only; async self-paced AT pass.
- **Performance** — 200 independent paced players; async load like normal quiz traffic.
- **Manual** — late submission flagging; resume an interrupted async attempt.

## 17. Documentation & Training

- End-user: "Play in teams", "Play at your own pace", "Complete a quiz-game homework".
- Instructor: choosing a mode; setting windows/attempts/policy; team setup; accommodations.
- API reference: mode configs + assignment endpoints.
- Runbook: async attempt/session relationship; gradebook item linkage.

## 18. Open Questions

1. Team aggregate default — sum or average? (Recommendation: average, to keep uneven teams fair; instructor-
   overridable.)
2. Does async mode reuse `quizgame.sessions` per attempt (chosen) or a lighter attempt-only table?
   (Recommendation: reuse sessions so scoring/reports are identical; accept a few extra rows.)
3. Should student-paced show a live leaderboard (spoilers) or only final? (Recommendation: final-only by
   default; instructor toggle for a live board.)

## 19. References

- Existing files: `server/internal/repos/assignmentoverrides/`, `server/internal/repos/enrollmentquizzesoverrides/`,
  `server/internal/repos/coursegrades/`, `server/internal/httpserver/quiz_grade_sync.go`,
  `server/internal/repos/studenttodos/`.
- Related plans: [IQ.3](../../completed/interactive-quizzes/IQ.3-live-game-hosting-engine.md), [IQ.4](../../completed/interactive-quizzes/IQ.4-player-join-and-gameplay.md),
  [IQ.5](IQ.5-scoring-leaderboards-mechanics.md), [IQ.7](IQ.7-reports-results-gradebook.md),
  [IQ.9](IQ.9-moderation-safety-accessibility.md).
