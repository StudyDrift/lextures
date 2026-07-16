# IQ.5 — Scoring, Leaderboards & Game Mechanics

> Implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](README.md). Consumes the timing + correctness recorded by [IQ.3](../../completed/interactive-quizzes/IQ.3-live-game-hosting-engine.md) and drives the feedback rendered by [IQ.4](../../completed/interactive-quizzes/IQ.4-player-join-and-gameplay.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.5 |
| **Section** | Interactive Quizzes |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment squad |
| **Depends on** | IQ.3, IQ.4 |
| **Unblocks** | IQ.7 |

---

## 1. Problem Statement

Points and a live leaderboard are what turn a quiz into a *game*. IQ.5 owns the **scoring model** — rewarding
both correctness and speed, with streak bonuses and per-question multipliers — and the **leaderboard** that
updates between questions, culminating in a podium. It must be **deterministic, explainable, and fair** (based
solely on IQ.3's server-clock timing), and it must be configurable so instructors can dial competitiveness up
or down (or off, for low-pressure formative use).

## 2. Goals

- A configurable, deterministic scoring function: base points for correct + a speed bonus scaled by remaining
  time, with optional streak bonus and per-question points style (`standard` / `double` / `no_points`).
- Real-time **leaderboard** computation and fan-out (top-N + "your rank") between questions and a final podium.
- Optional **game mechanics** that add fun without breaking fairness: streaks, double-points questions, and a
  small set of opt-in power-ups (e.g. "second chance", "shield") behind a toggle.
- Instructor controls to choose the scoring profile per game (competitive, formative/participation, custom).
- Full transparency: every awarded point is reconstructable from the stored response + profile (for disputes
  and reports).

## 3. Non-Goals

- Recording raw timing/correctness (IQ.3) or rendering feedback (IQ.4) — IQ.5 computes the numbers.
- Gradebook mapping of scores to grades (IQ.7).
- Team aggregation rules (IQ.6 defines team scoring on top of IQ.5's per-player points).
- Cosmetic reward systems (avatars/coins/unlockables) — noted as a future nicety, not in scope.

## 4. Personas & User Stories

- **As a student**, I want faster correct answers to score more, so speed matters.
- **As a student**, I want a streak bonus for consecutive correct answers, so momentum is rewarded.
- **As an instructor**, I want to mark a hard question "double points", so the stakes vary.
- **As an instructor running a low-stakes review**, I want a participation profile where everyone who answers
  correctly gets equal points, so it isn't only about speed.
- **As an instructor fielding a dispute**, I want to see exactly how a student's points were computed.

## 5. Functional Requirements

- **FR-1.** The scoring function MUST be a pure, versioned function of `(is_correct, response_ms, time_limit,
  points_style, streak, profile)` → integer points, evaluated **server-side** at lock/reveal.
- **FR-2.** The default "competitive" profile MUST award: `0` for incorrect; for correct,
  `base + round(base * speed_factor)` where `speed_factor = max(0, 1 − response_ms/deadline_ms)` (fastest ≈ 2×
  base, at-the-buzzer ≈ base). `base` default `1000`, configurable.
- **FR-3.** `points_style` MUST modify the award: `double` → ×2; `no_points` → `0` (poll/opinion or ungraded).
- **FR-4.** A **streak bonus** MUST be supported: +`streak_step` per consecutive correct answer up to a cap,
  reset to 0 on incorrect/unanswered; on/off and magnitude per profile.
- **FR-5.** The engine MUST persist the awarded `points` on `quizgame.session_responses` and maintain
  `session_players.total_score` and `streak` transactionally (no double-count on reconnect/replay).
- **FR-6.** The system MUST compute a **leaderboard** after each question: ranked players by `total_score`
  (deterministic tie-break: fewer total `response_ms`, then earliest join), and fan out top-N + each player's
  own rank/delta.
- **FR-7.** The system MUST support at least three built-in **profiles**: `competitive` (speed+streak),
  `formative` (fixed points for correct, no speed, no streak), and `custom` (instructor sets base, speed
  weight, streak step/cap).
- **FR-8.** Optional **power-ups** (behind a per-game toggle, default off) MUST be server-adjudicated and
  fair: e.g. "double-or-nothing" (opt-in on a question: correct → ×2 that question, incorrect → 0), "shield"
  (protect a streak once). Power-ups MUST NOT let a client fabricate score; the server validates eligibility.
- **FR-9.** All scoring MUST be **explainable**: a per-response breakdown (`base`, `speedBonus`, `streakBonus`,
  `styleMultiplier`, `powerUp`, `total`) MUST be derivable and exposed to reports (IQ.7) and the host.
- **FR-10.** The scoring-profile **version** MUST be stored on the session so historical games reproduce
  identical results even after the default profile changes.
- **FR-11.** Leaderboards MUST honour privacy settings (IQ.9): an instructor MAY anonymise the projected
  leaderboard (nicknames only, or hide names) while keeping per-student results in reports.

## 6. Non-Functional Requirements

- **Performance** — leaderboard recompute + fan-out for 200 players < 100 ms; scoring is O(1) per response.
- **Security** — scoring/power-ups are server-authoritative; clients cannot submit points or claim ineligible
  power-ups; profile is fixed at game start.
- **Privacy & Compliance** — leaderboard display is configurable to avoid publicly ranking students where
  policy/FERPA norms require (IQ.9); per-student scores remain restricted to instructor + that student.
- **Accessibility** — leaderboard readable by screen reader; rank changes announced politely; no colour-only
  rank encoding; reduced-motion podium.
- **Scalability** — incremental leaderboard update (heap/ordered structure), not full re-sort where avoidable.
- **Reliability** — score mutations transactional + idempotent with response writes; reconnect never
  double-awards.
- **Observability** — distributions of points/question, streak lengths, power-up usage; scoring latency.
- **Maintainability** — profiles are data (a registry), not branching code; the scoring function is a single
  pure module with a version constant.
- **Internationalization** — score/number formatting locale-aware; rank labels localised.
- **Backward compatibility** — versioned profiles guarantee historical reproducibility.

## 7. Acceptance Criteria

- **AC-1.** *Given* two students answer correctly, one at 2 s and one at 8 s on a 10 s timer, *when* scored
  with the competitive profile, *then* the 2 s answer earns more, and both earn ≥ base.
- **AC-2.** *Given* a "double points" question, *when* a correct answer is scored, *then* the award is exactly
  twice the equivalent standard-question award.
- **AC-3.** *Given* a student answers 3 in a row correctly then misses one, *when* scored, *then* the streak
  bonus accrued for the run and reset to 0 after the miss.
- **AC-4.** *Given* the formative profile, *when* two students answer correctly at different speeds, *then* they
  receive **equal** points (speed ignored).
- **AC-5.** *Given* a player reconnects after a scored question, *when* state re-syncs, *then* their
  `total_score` is unchanged (no double-award).
- **AC-6.** *Given* the host requests a breakdown, *when* they open a response, *then* base/speed/streak/style
  components sum exactly to the stored total.
- **AC-7.** *Given* the default profile changes next release, *when* an old game's report is reopened, *then*
  its scores are recomputed identically via its stored profile version.
- **AC-8.** *Given* anonymised-leaderboard mode, *when* the projector shows rankings, *then* student legal
  names are not displayed.

## 8. Data Model

Migration `394_interactive_quizzes_scoring.sql` (small — most state lives on IQ.3 tables):

```sql
ALTER TABLE quizgame.sessions
  ADD COLUMN IF NOT EXISTS scoring_profile     TEXT NOT NULL DEFAULT 'competitive',
  ADD COLUMN IF NOT EXISTS scoring_profile_ver INTEGER NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS scoring_config      JSONB NOT NULL DEFAULT '{}'::jsonb, -- base, speedWeight, streakStep/cap, powerUps
  ADD COLUMN IF NOT EXISTS leaderboard_privacy TEXT NOT NULL DEFAULT 'names';     -- names | nicknames | hidden

-- points already on quizgame.session_responses (IQ.3); add the breakdown for explainability:
ALTER TABLE quizgame.session_responses
  ADD COLUMN IF NOT EXISTS points_breakdown JSONB NOT NULL DEFAULT '{}'::jsonb;

-- optional power-up ledger (server-adjudicated)
CREATE TABLE IF NOT EXISTS quizgame.player_powerups (
  session_id     UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
  player_id      UUID NOT NULL REFERENCES quizgame.session_players (id) ON DELETE CASCADE,
  question_index INTEGER NOT NULL,
  kind           TEXT NOT NULL,               -- double_or_nothing | shield | ...
  applied_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (session_id, player_id, question_index, kind)
);
```

- `session_players.total_score` / `streak` (from IQ.3) are the running aggregates.
- Reproducibility comes from `scoring_profile_ver` + `scoring_config` + the raw stored responses.

## 9. API Surface

- **Config at start:** `POST /live-quizzes/kits/{kit_id}/games` (IQ.3) accepts
  `{scoringProfile, scoringConfig, leaderboardPrivacy, powerUpsEnabled}`.
- **WS frames (server→client):** `state.leaderboard` (top-N + `you`), `result.pointsBreakdown`, `podium`.
- **WS frames (player→server):** `{type:"powerup", kind, questionIndex}` (validated server-side).
- **REST:** `GET /live-quizzes/games/{game_id}/leaderboard` (host/enrolled) and
  `GET .../responses/{player_id}` returns per-question breakdowns (feeds IQ.7).
- **OpenAPI:** document profiles/config schema and the breakdown shape.

## 10. UI / UX

- **Host/projector leaderboard** `components/live-quiz/leaderboard.tsx`: animated (reduced-motion aware) top-N,
  rank deltas, and a podium at the end.
- **Scoring-profile picker** in the host start dialog: Competitive / Formative / Custom (base, speed weight,
  streak), plus toggles for power-ups and leaderboard privacy.
- **Player feedback** (IQ.4 `result-card`) shows the breakdown: "+1000 base +640 speed +100 streak = 1740".
- **Power-up UI:** a small pre-answer opt-in on eligible questions (e.g. "Double or nothing?").
- **States:** leaderboard loading, tie display, hidden/anonymised mode, no-points (poll) result.
- **Accessibility:** leaderboard is a semantic ordered list; rank changes announced; podium reduced-motion;
  breakdown readable by screen reader.
- **Copy & i18n:** `liveQuiz.score.*`, `liveQuiz.leaderboard.*`, `liveQuiz.powerup.*`.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Reuse:** IQ.3 response/score tables + WS fan-out; IQ.4 result rendering.
- **Server new:** `internal/quizgame/scoring` (pure function + profile registry), leaderboard computation in
  the engine, `repos/quizgame/leaderboard.go`.
- **Web new:** leaderboard/podium components, profile picker, breakdown display.

## 13. Dependencies & Sequencing

- Must ship after: IQ.3 (timing/correctness), IQ.4 (feedback surface).
- Must ship before: IQ.7 (reports consume points + breakdowns); IQ.6 layers team scoring on top.
- Shared infra: none new.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Speed-only scoring feels unfair / excludes slower learners | M | M | Formative profile; configurable speed weight; instructor guidance |
| Power-ups create score exploits | M | H | Server-adjudicated, eligibility-checked, ledgered; default off |
| Leaderboard re-sort cost at scale | L | M | Incremental ordered structure; cap top-N fan-out |
| Public ranking raises FERPA/wellbeing concerns | M | M | Anonymise/hidden leaderboard modes; per-student results stay private |
| Profile change breaks old reports | L | M | Versioned profiles + stored config → deterministic replay |

## 15. Rollout Plan

- **Flag:** part of `interactive_quizzes_enabled`; power-ups behind a per-game toggle (default off).
- **Sequencing:** migration `394` → scoring module + leaderboard → profile picker → enable.
- **Dogfood:** run competitive and formative games; verify breakdowns and podium; test anonymised mode.
- **GA criteria:** AC-1..AC-8 pass; scoring reproducibility test green.
- **Rollback:** default to `formative` fixed scoring if a scoring bug is found; power-ups off.

## 16. Test Plan

- **Unit** — scoring function across the matrix (speed, streak, style, profile); tie-break ordering;
  reproducibility by version; power-up eligibility.
- **Integration** — score persistence + aggregate updates idempotent on reconnect; leaderboard fan-out.
- **End-to-end** — Playwright: full game shows correct leaderboard and podium; double-points question; streak.
- **Security** — client cannot inject points or claim ineligible power-ups; profile fixed at start.
- **Accessibility** — leaderboard/podium screen-reader + reduced-motion.
- **Performance** — 200-player leaderboard recompute latency.
- **Manual** — dispute walkthrough using the breakdown.

## 17. Documentation & Training

- End-user: "How scoring works" (speed + streak + multipliers), with the formula.
- Instructor: choosing a scoring profile; when to anonymise the leaderboard; power-ups.
- API reference: profile/config + breakdown schemas.
- Runbook: scoring version constant and how to add a profile.

## 18. Open Questions

1. Ship power-ups in the first GA or defer? (Recommendation: ship streaks + double-points; gate power-ups
   behind a follow-up toggle once fairness is proven.)
2. Default leaderboard privacy — `names` or `nicknames`? (Recommendation: `names` for enrolled classes,
   instructor-overridable; `nicknames` when guests are allowed.)
3. Should "no answer" ever earn participation points? (Recommendation: no by default; formative profile MAY
   award a small participation point for answering, correct or not — configurable.)

## 19. References

- Existing files: `server/internal/repos/quizattempts/`, `server/internal/repos/itemanalysis/` (analytics
  reuse in IQ.7), `server/internal/telemetry`.
- Related plans: [IQ.3](../../completed/interactive-quizzes/IQ.3-live-game-hosting-engine.md), [IQ.4](../../completed/interactive-quizzes/IQ.4-player-join-and-gameplay.md),
  [IQ.6](IQ.6-game-modes-team-paced-async.md), [IQ.7](IQ.7-reports-results-gradebook.md),
  [IQ.9](IQ.9-moderation-safety-accessibility.md).
