# IQ.4 — Player Join & Gameplay Experience

> Implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](../../plan/interactive-quizzes/README.md). Consumes the authoritative game protocol from [IQ.3](IQ.3-live-game-hosting-engine.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.4 |
| **Section** | Interactive Quizzes |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Assessment squad + Web |
| **Depends on** | IQ.3 |
| **Unblocks** | IQ.5, IQ.6, IQ.9 |

---

## 1. Problem Statement

The engine (IQ.3) is invisible without a delightful, fast **player experience**. IQ.4 is what the student
actually touches: enter a join code, pick a nickname, and then — on their own phone or laptop — see the answer
buttons, tap fast, feel the countdown, and get instant "correct! +840" feedback with their rank. It must work
on a low-end phone over classroom Wi-Fi, be reachable without deep navigation, and be fully accessible
(answers distinguishable by shape and label, not colour alone).

## 2. Goals

- A frictionless **join flow**: code → nickname → lobby, reachable from a short URL and from inside the course.
- A responsive **answer surface** per question type (big tap targets, live countdown, submit-once feedback).
- Immediate per-question **feedback**: correct/incorrect, points earned, streak, and current rank/position.
- Robust **reconnection**: dropping and rejoining keeps the player's identity and score (using IQ.3's player
  token), with clear connection status.
- First-class **accessibility** and **mobile** behaviour (the primary device is a student phone/browser).

## 3. Non-Goals

- The scoring maths and leaderboard algorithm (IQ.5) — IQ.4 renders what the server sends.
- Team UI specifics and student-paced/async player flows (IQ.6) — IQ.4 ships the classic live player.
- Nickname content policy and anti-cheat enforcement (IQ.9) — IQ.4 wires the inputs/hooks.
- A native mobile app player (mobile `M##` series) — IQ.4 targets responsive web; native is a follow-up.

## 4. Personas & User Stories

- **As a student**, I want to join a game by typing a short code and a nickname, so I'm playing in seconds.
- **As a student**, I want big, clearly-labelled answer buttons and a visible countdown, so I can answer fast
  without misclicks.
- **As a student**, I want to instantly know if I was right and how many points I got, so it feels like a game.
- **As a student using a screen reader**, I want the question, options, timer, and result announced, so I can
  play independently.
- **As a student whose connection blips**, I want to rejoin and keep my score, not lose my place.
- **As an enrolled student**, I want my play to be tied to my account (for the gradebook), while a guest player
  is only a nickname.

## 5. Functional Requirements

- **FR-1.** The system MUST provide a **join page** (`/play` and `/play/{code}`) where a user enters a join
  code, then a nickname; on success it opens the player WS and enters the lobby.
- **FR-2.** Enrolled users MUST join **authenticated** (player row linked to `user_id`), so results reach the
  gradebook (IQ.7); guest join (nickname only) is allowed **only** when the game permits it and per IQ.9 rules.
- **FR-3.** Nicknames MUST be validated client- and server-side (length, allowed charset) and pass the IQ.9
  moderation hook; duplicates within a game MUST be rejected (`session_players` unique on `(session, nickname)`).
- **FR-4.** The player WS `GET /live-quizzes/games/{game_id}/player-ws` MUST authenticate (JWT for enrolled;
  signed player token for guests) and reconnect using the player token so a rejoin resumes identity+score.
- **FR-5.** For each `question_open` state, the client MUST render the type-appropriate answer UI:
  MC (2–6 shape+colour+label tiles), true/false, type-answer (text field + submit), numeric (number field),
  ordering (reorderable list), poll (same as MC, no "correct"), word-cloud (short text). Options render in the
  server-provided (shuffled) order.
- **FR-6.** Answer submission MUST send `{type:"answer", questionIndex, answer, clientSentAt}` and then lock the
  UI ("answer received"); the server's server-clock timing (IQ.3 FR-6) is authoritative — `clientSentAt` is
  telemetry only. Re-submission MUST be prevented client-side and is idempotent server-side.
- **FR-7.** On the server's per-question **result** frame, the client MUST show correct/incorrect, points
  earned, streak, and rank/position; on **reveal** it MAY show the correct answer and explanation.
- **FR-8.** The client MUST show a **countdown** synchronized to the server `deadline` (using round-trip clock
  offset estimation), degrading gracefully if the tab was backgrounded.
- **FR-9.** The client MUST surface connection status (connected / reconnecting / disconnected) and auto-retry
  with backoff; on reconnect it re-syncs to current server state via `seq`.
- **FR-10.** Between questions and at the end, the player MUST see their standing (podium for top 3, "you
  placed Nth" for everyone) and, if enrolled, a link to their per-question review (IQ.7).
- **FR-11.** If the host kicks the player (IQ.3 FR-12), the client MUST show a clear "removed by host" state and
  close the WS.
- **FR-12.** The join surface MUST refuse codes for ended/nonexistent games with a friendly error and respect
  the join-code lookup rate limit.

## 6. Non-Functional Requirements

- **Performance** — join-to-lobby < 3 s on a mid-range phone; answer tap → "received" < 150 ms; bundle for the
  player route code-split and lightweight (target < 150 KB gz for the play path).
- **Security** — guest tokens are per-game, signed, expiring, and carry no account privileges; enrolled join
  uses the normal session; no correct answers are ever sent before reveal.
- **Privacy & Compliance** — guest nicknames are transient and minimised; enrolled play ties to the roster;
  under-13 guest play follows [S08](../../plan/standards/S08-childrens-privacy-age-assurance-design-codes.md).
- **Accessibility** — WCAG 2.1 AA: answers distinguishable by **shape + text**, not colour; all controls
  keyboard-operable; ARIA live for countdown/results; visible focus; reduced-motion; targets ≥ 44px; supports
  200% zoom/reflow.
- **Scalability** — 200 concurrent players/game; the client holds minimal state and trusts server frames.
- **Reliability** — reconnect resumes; backgrounded tab recovers; double-tap cannot double-submit.
- **Observability** — client emits (privacy-safe) join, answer, reconnect, error events; server counts joins,
  guest vs enrolled, kicks.
- **Maintainability** — one `useLiveGame` player hook; per-type answer components share a registry with IQ.2's
  editor components.
- **Internationalization** — all player copy localised; numeric/score formatting locale-aware; RTL-safe answer
  layout.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a live game's code, *when* a student enters it and a nickname on `/play`, *then* they land
  in the lobby and appear in the host's player list within ~1 s.
- **AC-2.** *Given* a duplicate nickname, *when* a second student tries it, *then* they're asked to choose
  another (server-rejected).
- **AC-3.** *Given* an open MC question, *when* the student taps an option, *then* the UI locks to "answer
  received" and a second tap does nothing.
- **AC-4.** *Given* the reveal, *when* the result frame arrives, *then* the student sees correct/incorrect,
  points earned, streak, and their rank.
- **AC-5.** *Given* a screen-reader user, *when* a question opens, *then* the prompt, options, and countdown are
  announced and the options are operable by keyboard.
- **AC-6.** *Given* the student's connection drops mid-question, *when* it recovers, *then* they resume with
  their score and see the current question state (or the "answer received"/reveal state if they'd already
  answered).
- **AC-7.** *Given* colour-blind simulation, *when* answers render, *then* each option is still distinguishable
  by its shape and label alone.
- **AC-8.** *Given* the host kicks the player, *when* the frame arrives, *then* the player sees "removed" and
  the connection closes.

## 8. Data Model

No new tables — reuses `quizgame.session_players`, `quizgame.session_responses`, and `player_token` from
[IQ.3](IQ.3-live-game-hosting-engine.md). IQ.4 adds:

- A signed **guest player token** (JWT-like, per game, short-lived) minted at join and used for reconnect; the
  hash is stored in `session_players.player_token` (already defined in IQ.3's migration).
- Optional `session_players` columns via a tiny migration `393_interactive_quizzes_player_client.sql` if
  needed: `last_seen_at TIMESTAMPTZ`, `client_meta JSONB` (coarse device/browser for support; no fingerprinting).

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| GET | `/live-quizzes/join/{code}` | public, rate-limited → `{gameId, kitTitle, requiresAuth, allowsGuests}` |
| POST | `/live-quizzes/games/{game_id}/join` | JWT (enrolled) or none (guest) → `{playerId, playerToken}` |
| GET (WS) | `/live-quizzes/games/{game_id}/player-ws` | first-msg `authToken`(enrolled) or `playerToken`(guest) |

- **Player WS frames:** client→server `{type:"answer", questionIndex, answer, clientSentAt}`,
  `{type:"hello", resumeSeq}`; server→player `{type:"state"|"result"|"reveal"|"kicked"|"standing", …}` (same
  `seq` contract as IQ.3).
- **Answer payloads** are type-specific: MC → option id(s); type_answer → string; numeric → number;
  ordering → ordered id list; poll → option id; word_cloud → string.
- **Rate-limit:** join lookups and answer frames are rate-limited per IP/connection (anti-abuse, IQ.9).

## 10. UI / UX

- **Join page** `clients/web/src/pages/live-quiz-play-page.tsx` (note: outside `/lms` course chrome — reachable
  unauthenticated for guests): code step → nickname step → lobby → gameplay → standing/podium.
- **Answer components** in `components/live-quiz/play/`: `answer-grid` (shape+colour+label tiles),
  `answer-truefalse`, `answer-type`, `answer-numeric`, `answer-ordering`, `answer-poll`, `answer-wordcloud`;
  `countdown-ring`, `result-card`, `standing-card`, `connection-badge`.
- **Flows:** (1) enter code → (2) enter nickname → (3) lobby ("you're in!") → (4) answer each question →
  (5) see result → (6) final standing/podium.
- **States:** joining, nickname-taken, waiting-for-host, question-open, answered/locked, reveal, between-rounds,
  reconnecting, kicked, game-ended, error.
- **Mobile:** phone-first; large tap targets; sticky countdown; no horizontal scroll; works portrait/landscape.
- **Accessibility:** answers = shape icon + text (colour is redundant); ARIA live for countdown (polite,
  assertive in final 5 s) and results; keyboard number/letter shortcuts (1–6, T/F); focus trap-free; reduced
  motion; no flashing.
- **Copy & i18n:** `liveQuiz.play.*`, `liveQuiz.answer.*`, `liveQuiz.standing.*`.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Reuse:** IQ.3 player WS + `useLiveGame` hook, the shared question-type registry from IQ.2, the app's design
  system (buttons, focus), i18n catalog.
- **Server new:** `httpserver/quizgame_join.go` (join lookup + join + guest token), player-WS role in
  `quizgame_game_ws.go`.
- **Web new:** the public play route (registered in `app.tsx` outside the authed LMS shell), play components.

## 13. Dependencies & Sequencing

- Must ship after: IQ.3 (protocol + engine).
- Must ship before: IQ.5 (feedback surfaces scores it computes), IQ.6 (extends player flows), IQ.9 (moderates
  the join inputs).
- Shared infra: WebSocket, rate limiter, design system.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Player bundle too heavy for low-end phones | M | M | Code-split the `/play` route; minimal deps; perf budget in CI |
| Countdown drift vs server deadline | M | M | Estimate clock offset on connect; server is authoritative for scoring regardless |
| Colour-only answers fail a11y | M | H | Shape+label required by design; axe + colour-blind test in CI |
| Double-submit from fast double-tap | M | M | Client lock + server idempotency (IQ.3) |
| Guest join abused for spam/impersonation | M | M | Per-game signed tokens, rate limits, IQ.9 moderation, host kick |
| Backgrounded tab misses frames | M | M | `seq`-based re-sync on visibility/reconnect |

## 15. Rollout Plan

- **Flag:** gated by `interactive_quizzes_enabled` + IQ.3's `iq_live_hosting`; guest-join sub-flag off until
  IQ.9 ships.
- **Sequencing:** deploy join API + play route → enrolled-only join first → enable guest join after IQ.9.
- **Dogfood:** 30-player classroom test on mixed devices; force reconnects; screen-reader pass.
- **GA criteria:** AC-1..AC-8 pass; a11y audit clean; perf budget met on a throttled mid-range device.
- **Rollback:** disable hosting sub-flag; play route shows "no active games".

## 16. Test Plan

- **Unit** — per-type answer components (render + submit-lock); countdown offset math; reconnect `seq` resume;
  nickname validation.
- **Integration** — join lookup/join/guest-token; duplicate nickname rejection; answer idempotency with IQ.3.
- **End-to-end** — Playwright multi-context: 3 players join + play a full game; reconnect mid-question;
  kicked-player state; ended state.
- **Security** — guest token scope/expiry; no early correct-answer leakage to players; rate-limit enforcement.
- **Accessibility** — axe on every play state; screen-reader script for a full question; colour-blind
  simulation; keyboard-only play; 200% zoom.
- **Performance** — throttled-device join time and bundle size; 200 simulated players answering at once.
- **Manual** — Wi-Fi drop mid-question; tab background/foreground; portrait/landscape.

## 17. Documentation & Training

- End-user (student): "Join and play a live quiz" (with/without an account).
- Instructor: sharing the join code/URL; enabling guest join (once available); accessibility notes for students.
- API reference: join + player-WS contract.
- Runbook: guest-token issuance/expiry; play-route CSP/CORS for the unauthenticated surface.

## 18. Open Questions

1. Should the public `/play` route allow fully-anonymous guests at GA, or require a name + course-provided
   access? (Recommendation: enrolled-only at first GA; anonymous guests behind IQ.9.)
2. Keyboard answer shortcuts (1–6 / A–F / T–F) — enable by default? (Recommendation: yes, with an on-screen hint.)
3. Show the correct answer to players on reveal, or only on the projector? (Recommendation: instructor-toggle,
   default show on player device for study value.)

## 19. References

- Existing files: `clients/web/src/app.tsx` (route registration, incl. non-LMS public routes),
  `clients/web/src/lib/` API client patterns, design-system components.
- Related plans: [IQ.3](IQ.3-live-game-hosting-engine.md), [IQ.5](../../plan/interactive-quizzes/IQ.5-scoring-leaderboards-mechanics.md),
  [IQ.6](../../plan/interactive-quizzes/IQ.6-game-modes-team-paced-async.md), [IQ.9](../../plan/interactive-quizzes/IQ.9-moderation-safety-accessibility.md).
