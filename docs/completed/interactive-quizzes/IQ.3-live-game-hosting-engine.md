# IQ.3 — Live Game Hosting Engine (Real-Time)

> Implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](../../plan/interactive-quizzes/README.md). Reuses the WebSocket auth/room *transport* proven by Collaborative Documents (`collab_docs_ws.go`, `server.go`), but builds an **authoritative** game state machine — deliberately **not** the Y.js CRDT model, which is wrong for a host-driven competition.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.3 |
| **Section** | Interactive Quizzes |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Assessment squad + Realtime |
| **Depends on** | IQ.1, IQ.2 |
| **Unblocks** | IQ.4, IQ.5, IQ.6, IQ.7, IQ.9 |

---

## 1. Problem Statement

The magic of a live quiz is the shared clock: the question appears on the projector, every phone shows answer
buttons, a countdown ticks, and the instant it locks the room sees who was fastest. That requires an
**authoritative real-time server** — one source of truth for the current question, the deadline, and every
player's score — because unlike a collaborative document there is a *right answer*, a *timer*, and a
*competition* that no peer may be trusted to adjudicate. IQ.3 stands up that engine: a host starts a game from
a kit, gets a **join code**, and drives a per-question lifecycle over WebSockets while the server scores every
submission against a monotonic server clock.

## 2. Goals

- Let an instructor start a **game** from a kit, producing a short numeric **join code** and a host control
  surface (lobby → per-question flow → podium).
- Run an **authoritative** per-game state machine on the server: the server owns current question, deadline,
  lock, reveal, and scoring; clients render what the server tells them.
- Provide a WebSocket **game hub** (one room per game) that reuses our WS upgrade + token/enrollment auth, and
  fans out state to host, players, and an optional big-screen/projector view.
- Enforce the **server clock** for scoring: answer timing is measured server-side (receipt vs. deadline), so
  client lag/latency cannot be gamed.
- Persist an append-only **event log** and player/response rows so a game survives host or player reconnects
  and can be reconstructed, reported on (IQ.7), and audited.
- Handle the messy realities: reconnection, host handover/disconnect grace, late joiners, network partitions,
  and duplicate submissions (idempotent per player+question).

## 3. Non-Goals

- The player device UI and join UX (IQ.4) — IQ.3 defines the protocol/events they consume.
- Scoring formulae, streaks, power-ups, leaderboards presentation (IQ.5) — IQ.3 records timing + correctness
  and exposes hooks; IQ.5 owns the maths and mechanics.
- Team / student-paced / async modes (IQ.6) — IQ.3 ships the **classic live, individual** mode; it is
  structured so IQ.6 extends the state machine.
- Nickname moderation / anti-cheat policy (IQ.9) — IQ.3 provides enforcement hooks; IQ.9 owns the rules.

## 4. Personas & User Stories

- **As an instructor**, I want to start a game from my kit and show a join code, so students can join in
  seconds.
- **As an instructor**, I want to control the pace — reveal the question, watch the answer count climb, then
  lock and show results — so I facilitate the room.
- **As a student**, I want the question and my answer buttons to appear the instant the teacher starts it,
  with a live countdown.
- **As a student whose phone dropped Wi-Fi**, I want to rejoin the same game and keep my score, not start over.
- **As an instructor whose laptop crashed**, I want to reopen the host screen and resume the game where it was.
- **As an instructor**, I want a separate "big screen" view for the projector that never shows answers early.

## 5. Functional Requirements

- **FR-1.** `POST /live-quizzes/kits/{kit_id}/games` MUST create a `quizgame.sessions` row (status `lobby`),
  snapshot the kit's questions into the session (so mid-game kit edits can't mutate a running game), and
  return a unique, human-typeable **join code** (see FR-9).
- **FR-2.** The system MUST expose `GET /live-quizzes/games/{game_id}/ws` (host + projector) and a
  player WS (IQ.4) that upgrade to WebSocket and authenticate via the existing handshake (first message
  `{"authToken":…}` → `JWTSigner`), plus a game-membership/role check.
- **FR-3.** The engine MUST maintain one **in-process room per game** (registry mirroring
  `getOrCreateRoom`) holding host connection(s), player connections, and the authoritative game state.
- **FR-4.** The server MUST own a **state machine** per game:
  `lobby → question_intro → question_open(deadline) → question_locked → question_reveal → (next|leaderboard) → … → podium → ended`.
  Only the host (or an auto-advance timer, if configured) may drive transitions.
- **FR-5.** On `question_open`, the server MUST record a monotonic `opened_at` and compute a `deadline`
  (`opened_at + time_limit`); it MUST broadcast the question payload (prompt, shuffled options **without**
  correctness, deadline) to players and the projector.
- **FR-6.** Player answer submissions MUST be accepted only while `question_open` and before `deadline`
  (server clock); the server MUST record `response_ms = received_at − opened_at`, correctness, and a
  first-write-wins guard so a player scores at most once per question (idempotent on `(session, question,
  player)`).
- **FR-7.** On lock/reveal, the server MUST compute correctness and hand timing+correctness to the scoring
  module (IQ.5) to award points, then broadcast per-player result (correct/incorrect, points, rank delta) and
  aggregate answer distribution (for the projector).
- **FR-8.** The system MUST persist an **append-only event log** (`quizgame.session_events`) and
  `quizgame.session_responses`; a reconnecting host or player MUST be able to reconstruct current state by
  replaying/reading the latest snapshot (the DB is the durable source of truth, the in-process room is a
  cache).
- **FR-9.** Join codes MUST be short (default 6 digits), unique among **active** games, unguessable enough
  (random, not sequential), rate-limited on lookup, expired on game end, and never reused while active.
- **FR-10.** The engine MUST tolerate **host disconnect** with a grace window (game pauses, players see
  "waiting for host"); on host return within the window it resumes; on expiry it auto-ends and finalises.
- **FR-11.** The engine MUST tolerate **player disconnect/reconnect**: a player rejoining with their session
  token resumes with their score; late joiners MAY be admitted to the lobby or (if configured) mid-game
  starting at the current question with zero back-score.
- **FR-12.** The relay MUST cap message size/rate per connection and drop/throttle abusive connections without
  affecting the room; the host MUST be able to kick a player (hook for IQ.9).
- **FR-13.** Starting/joining a game MUST be refused when the platform or course flag is off, the kit is not
  "ready" (IQ.2 validation), or the game has ended.
- **FR-14.** The engine MUST support **auto-advance** (timed) and **manual-advance** (host clicks Next) pacing,
  selectable at game start.
- **FR-15.** A game MUST finalise on the last question or host "End game", writing final scores and marking the
  session `ended` (results owned by IQ.7).

## 6. Non-Functional Requirements

- **Performance** — question fan-out p95 < 200 ms to a room of 200 players; answer ack < 150 ms; server clock
  drift irrelevant (single authority). Support ≥ 200 concurrent players/game and many concurrent games/instance.
- **Security** — token-verified, role-checked (host vs player vs projector); players cannot receive correct
  answers before reveal; the server never trusts client-reported timing or client-claimed identity for authz;
  join codes are rate-limited and non-enumerable.
- **Privacy & Compliance** — responses/scores are education records (deletion/export via
  [S01](../standards/S01-unified-data-subject-rights-orchestration.md)/[S02](../standards/S02-data-retention-deletion-engine.md));
  guest (non-enrolled) players handled per IQ.9 (nickname only, consent/age rules).
- **Accessibility** — countdown announced via ARIA live (polite→assertive near end); no reliance on colour;
  reduced-motion host/projector option; no flashing (photosensitivity).
- **Scalability** — single-process rooms match the collab-doc design; a horizontal fan-out path (Postgres
  LISTEN/NOTIFY or Redis pub/sub keyed by game_id, with sticky routing) is specified as the multi-instance
  follow-up.
- **Reliability** — DB is durable truth; in-process room is reconstructable; submissions idempotent; host/player
  reconnect within grace; auto-finalise on abandonment; exactly-once scoring per player+question.
- **Observability** — gauges: live games, players/game, host-connected; counters: answers received/late/dup,
  reconnects, kicks; histograms: fan-out latency, answer `response_ms`; traces around transitions.
- **Maintainability** — extract shared WS helpers from `collab_docs_ws.go` into a reusable package; game state
  machine is a pure, unit-testable reducer separate from transport.
- **Internationalization** — server sends structured payloads (ids + values), never localised strings; clients
  localise.
- **Backward compatibility** — additive; no change to existing quiz/collab behaviour.

## 7. Acceptance Criteria

- **AC-1.** *Given* a ready kit, *when* the instructor starts a game, *then* a unique join code is issued and a
  lobby appears; the projector view shows "join at … code ……".
- **AC-2.** *Given* players in the lobby, *when* the host opens Q1, *then* every player sees the question and a
  synchronized countdown within ~200 ms.
- **AC-3.** *Given* a player answers 3.2 s after open on a 20 s timer, *when* the server records it, *then*
  `response_ms ≈ 3200` measured server-side, independent of the client's claimed time.
- **AC-4.** *Given* a player submits twice for the same question, *when* both arrive, *then* only the first
  counts (idempotent) and the second is rejected.
- **AC-5.** *Given* the deadline passes, *when* a late submission arrives, *then* it is rejected as late and the
  player is marked unanswered for that question.
- **AC-6.** *Given* a player's phone drops and rejoins mid-game, *when* they reconnect, *then* they resume with
  their existing score and the current question state — no duplicate player, no score reset.
- **AC-7.** *Given* the host's browser crashes, *when* they reopen the host screen within the grace window,
  *then* the game resumes at the same question; players saw a "waiting for host" pause, not an end.
- **AC-8.** *Given* the last question is revealed, *when* the host advances, *then* the game reaches the podium,
  finalises scores, and marks the session ended.
- **AC-9.** *Given* the course flag is off (or kit not ready), *when* a start is attempted, *then* it is refused
  with a clear error.
- **AC-10.** *Given* a projector view, *when* a question is open, *then* it never renders which option is
  correct until reveal.

## 8. Data Model

Migration `392_interactive_quizzes_sessions.sql`:

```sql
CREATE TYPE quizgame.session_status AS ENUM ('lobby','running','paused','ended','abandoned');
CREATE TYPE quizgame.session_mode   AS ENUM ('live_classic','team','student_paced','homework'); -- IQ.6 extends

CREATE TABLE quizgame.sessions (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kit_id         UUID NOT NULL REFERENCES quizgame.kits (id) ON DELETE RESTRICT,
  course_id      UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
  host_id        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  join_code      TEXT,                          -- NULL once ended/expired
  mode           quizgame.session_mode   NOT NULL DEFAULT 'live_classic',
  status         quizgame.session_status NOT NULL DEFAULT 'lobby',
  pacing         TEXT NOT NULL DEFAULT 'manual', -- manual | auto
  kit_snapshot   JSONB NOT NULL,                -- frozen questions at start
  current_index  INTEGER NOT NULL DEFAULT -1,   -- -1 = lobby
  current_phase  TEXT NOT NULL DEFAULT 'lobby',
  question_opened_at TIMESTAMPTZ,               -- server clock for current question
  settings       JSONB NOT NULL DEFAULT '{}'::jsonb, -- late-join, shuffle, anti-cheat (IQ.9)
  started_at     TIMESTAMPTZ,
  ended_at       TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_quizgame_active_join_code
  ON quizgame.sessions (join_code) WHERE join_code IS NOT NULL AND status IN ('lobby','running','paused');
CREATE INDEX idx_quizgame_sessions_course ON quizgame.sessions (course_id, created_at DESC);

CREATE TABLE quizgame.session_players (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id   UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
  user_id      UUID REFERENCES "user".users (id) ON DELETE SET NULL, -- NULL = guest
  nickname     TEXT NOT NULL,
  team_id      UUID,                          -- IQ.6
  player_token TEXT NOT NULL,                 -- reconnect secret (hashed)
  total_score  INTEGER NOT NULL DEFAULT 0,
  streak       INTEGER NOT NULL DEFAULT 0,
  connected    BOOLEAN NOT NULL DEFAULT TRUE,
  joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  removed_at   TIMESTAMPTZ,                   -- kicked/left
  UNIQUE (session_id, nickname)
);
CREATE INDEX idx_quizgame_players_session ON quizgame.session_players (session_id);

CREATE TABLE quizgame.session_responses (
  session_id   UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
  question_index INTEGER NOT NULL,
  player_id    UUID NOT NULL REFERENCES quizgame.session_players (id) ON DELETE CASCADE,
  answer       JSONB NOT NULL,
  is_correct   BOOLEAN NOT NULL,
  response_ms  INTEGER NOT NULL,              -- server-measured
  points       INTEGER NOT NULL DEFAULT 0,    -- filled by IQ.5
  answered_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (session_id, question_index, player_id)   -- idempotency guard (FR-6)
);

CREATE TABLE quizgame.session_events (
  id         BIGSERIAL PRIMARY KEY,
  session_id UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
  seq        INTEGER NOT NULL,
  type       TEXT NOT NULL,                   -- question_open, lock, reveal, player_join, ...
  payload    JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (session_id, seq)
);
CREATE INDEX idx_quizgame_events_session ON quizgame.session_events (session_id, seq);
```

- **Idempotency:** the composite PK on `session_responses` enforces one scored answer per player+question.
- **Reconstruction:** current state derives from `sessions.current_*` + latest events; the room is a cache.
- **Retention/backfill:** ended sessions retained for reports (IQ.7) then aged per IQ.11/S02; none to backfill.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| POST | `/live-quizzes/kits/{kit_id}/games` | `item:create` (host) → returns `{gameId, joinCode}` |
| GET | `/live-quizzes/games/{game_id}` | host/enrolled |
| GET (WS) | `/live-quizzes/games/{game_id}/ws` | first-msg `authToken` → JWT + host/projector role |
| POST | `/live-quizzes/games/{game_id}/end` | host |
| GET | `/live-quizzes/join/{code}` | public (rate-limited) → `{gameId, kitTitle, requiresAuth}` |

- **WS protocol (JSON frames):** host→server `{type:"open"|"lock"|"reveal"|"next"|"skip"|"kick"|"pause", …}`;
  server→client `{type:"state", phase, questionIndex, question?, deadline?, distribution?, leaderboard?, …}`;
  player→server answer frames are defined in IQ.4. All server frames carry a monotonically increasing `seq`
  for gap detection and replay.
- **Auth roles:** the WS query/first-message names role (`host` | `projector`); host requires the course
  permission, projector is a read-only token derived from the game (no answer leakage).
- **Rate-limit:** `join/{code}` lookups strictly rate-limited (anti-enumeration); answer frames rate-limited
  per connection.
- **OpenAPI:** document REST endpoints + the WS handshake/frame catalogue (as collab docs is documented).

## 10. UI / UX

- **Host console** `clients/web/src/pages/lms/live-quiz-host-page.tsx`: lobby (join code, joined-player list,
  Start), per-question controls (Open/Lock/Reveal/Next, live answer count, skip), pause/kick, End game.
- **Projector view** `live-quiz-present-page.tsx`: full-screen, large type — join instructions in lobby, the
  question + answer distribution live, correct answer + leaderboard on reveal. Answer-blind until reveal.
- **`useLiveGame` hook** (`clients/web/src/lib/live-quiz-realtime.ts`): opens the WS, applies server `state`
  frames to a local reducer, exposes `phase`, `question`, `deadline`, `players`, and host actions. Handles
  reconnect with `seq` resume.
- **States:** connecting, reconnecting (banner + auto-retry with backoff), host-paused ("waiting for host"),
  ended. Optimistic host buttons disabled until the server confirms the transition.
- **Accessibility:** countdown as ARIA live region; keyboard shortcuts for host (space = advance); reduced
  motion; projector high-contrast preset; no flashing transitions.
- **Copy & i18n:** `liveQuiz.host.*`, `liveQuiz.present.*`, `liveQuiz.state.*`.

## 11. AI / ML Considerations

Not AI-touching. (An optional AI "explain this answer" on reveal could reuse IQ.10's provider path later.)

## 12. Integration Points

- **Reuse / refactor:** extract the WS core from `server/internal/httpserver/collab_docs_ws.go` into a shared
  package (`server/internal/wsroom` or reuse the one VC.4 proposes as `yrelay`, generalised) — upgrade, token
  handshake, room registry, size/rate caps. Reuse `JWTSigner`, `enrollment.UserHasAccess`,
  `course.GetIDByCourseCode`, telemetry.
- **Server new:** `httpserver/quizgame_game_ws.go`, `httpserver/quizgame_games.go`,
  `repos/quizgame/{sessions,players,responses,events}.go`, a pure `internal/quizgame/engine` reducer package,
  and a background finaliser/reaper (`server/internal/background/`) for abandoned games + join-code expiry.
- **Web new:** host + projector pages, `live-quiz-realtime.ts`.

## 13. Dependencies & Sequencing

- Must ship after: IQ.1, IQ.2 (a ready kit to host).
- Must ship before: IQ.4 (player experience consumes this protocol), and it unblocks IQ.5–IQ.7, IQ.9.
- Shared infra: WebSocket support (already in `server.go`), background job runner, telemetry.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Single-process rooms don't fan out across instances | M | H | Sticky routing by `game_id` for GA; specify + implement LISTEN/NOTIFY (or Redis) bridge before multi-node |
| Client-timing/answer spoofing to win | H | H | Server clock is sole authority; correct answers withheld until reveal; server-stamped identity |
| Host disconnect kills a live class game | M | H | Grace-window pause + resume; DB is durable; auto-finalise only on expiry |
| Join-code enumeration / crashing others' games | M | H | Random non-sequential codes, active-only uniqueness, strict lookup rate-limit, role-scoped WS |
| Duplicate/late submissions corrupt scores | M | M | Composite-PK idempotency + server-clock deadline check |
| Thundering herd at question open (200 phones) | M | M | Single broadcast frame, backpressure, per-conn caps; load test at 200 players |
| In-process room ↔ DB divergence | M | M | DB authoritative on reconnect; events are append-only; reducer is deterministic and tested |

## 15. Rollout Plan

- **Flag:** `interactive_quizzes_enabled`; an internal `iq_live_hosting` sub-flag lets IQ.1/IQ.2 (authoring)
  ship before hosting is enabled.
- **Sequencing:** migrations `392` → deploy engine + WS behind sub-flag → load test → dogfood a real class →
  enable.
- **Dogfood:** run a live game in an internal class of 30+; force host/player reconnects; verify finalise.
- **GA criteria:** AC-1..AC-10 pass; 200-player load test meets latency targets; reconnect scenarios green.
- **Rollback:** disable `iq_live_hosting` (authoring remains); in-flight games finalise gracefully; DB retained.

## 16. Test Plan

- **Unit** — the state-machine reducer (all transitions, illegal transitions rejected); server-clock deadline
  math; join-code generation/uniqueness; idempotent scoring guard.
- **Integration** — multi-client WS: host opens → players answer → lock/reveal; late/dup submissions; host and
  player reconnect replay; abandonment finalise; flag/kit-ready refusal.
- **End-to-end** — Playwright multi-context: host + 3 players full game to podium; reconnection mid-game;
  projector answer-blindness.
- **Security** — projector cannot see answers early; join-code enumeration blocked; spoofed timing ignored;
  role escalation attempts fail.
- **Accessibility** — countdown live-region; keyboard host control; reduced-motion/no-flash.
- **Performance / load** — 200-player room fan-out + answer throughput; many concurrent games/instance.
- **Manual** — network partition + heal; two host tabs; Wi-Fi drop on player mid-question.

## 17. Documentation & Training

- End-user: "Host your first live game"; projector setup; what happens if you disconnect.
- API reference: game REST + WS frame catalogue + reconnection contract.
- Runbook: room registry, finaliser/reaper jobs, join-code expiry, and the multi-instance fan-out plan +
  scaling limits.

## 18. Open Questions

1. Ship horizontal fan-out (sticky + LISTEN/NOTIFY vs Redis) for GA, or accept single-instance until scale
   demands it? (Recommendation: sticky single-instance for GA; land the bridge before multi-node WS.)
2. Host grace window length and auto-advance defaults? (Recommendation: 90 s host grace; manual pacing default,
   auto optional.)
3. Are guest (non-enrolled) players allowed at all, or enrolled-only for v1? (Recommendation: enrolled-only for
   the first GA; guest join gated behind IQ.9 moderation before opening publicly.)
4. Snapshot the kit at start (chosen) vs. live-reference — confirm snapshot to prevent mid-game mutation.
   (Recommendation: snapshot, as specified.)

## 19. References

- Existing files: `server/internal/httpserver/collab_docs_ws.go` (WS upgrade/auth/room registry),
  `server/internal/httpserver/server.go` (WS setup), `enrollment.UserHasAccess`,
  `server/internal/repos/quizattempts/` (attempt/response persistence patterns), `server/internal/telemetry`.
- Related plans: [IQ.1 (completed)](IQ.1-foundation-and-feature-flag.md), [IQ.2 (completed)](IQ.2-kit-authoring-and-question-types.md), [IQ.4 (completed)](IQ.4-player-join-and-gameplay.md),
  [IQ.5](IQ.5-scoring-leaderboards-mechanics.md), [IQ.6](IQ.6-game-modes-team-paced-async.md),
  [IQ.9](../../plan/interactive-quizzes/IQ.9-moderation-safety-accessibility.md); Collaborative Documents (plan 6.5) / [VC.4](../../plan/visual-collaboration/VC.4-realtime-collaboration-and-presence.md) for WS transport reuse.
