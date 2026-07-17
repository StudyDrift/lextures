# MOB.5 — Interactive Quizzes (mobile)

> Implementation plan. Source: Mobile ↔ web parity gap analysis (2026-07-17).
> Web reference: [`clients/web/src/pages/live-quiz-play-page.tsx`](../../../clients/web/src/pages/live-quiz-play-page.tsx),
> `clients/web/src/components/live-quiz/*`, `clients/web/src/lib/live-quiz-api.ts`.
> Backend: shipped plans [`docs/completed/interactive-quizzes/IQ.1–IQ.11`](../../completed/interactive-quizzes/).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MOB.5 |
| **Section** | Mobile parity |
| **Severity** | BLOCKER (K12) / MAJOR (HE, SL) |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Mobile team |
| **Depends on** | IQ.1–IQ.11 (shipped) |
| **Unblocks** | MOB.3 live-quizzes governance item |

## 1. Problem Statement

Interactive (game-based, live) quizzing shipped end-to-end on web and backend
(IQ.1–IQ.11): authors build quiz **kits** (8 question types), hosts run **live
games** with join codes + leaderboards, and there are team / student-paced /
async-homework modes, reports/gradebook, library/templates/sharing, moderation,
and AI generation. There is **no** interactive-quiz code on mobile at all.
Since live quizzing is a phone-first activity — students overwhelmingly join on
their own devices — the missing mobile client is the single biggest live-quiz
adoption blocker, especially for K-12 classrooms.

## 2. Goals

- Deliver a first-class **student play** experience on iOS/Android: join by
  code, answer all question types, see the live leaderboard.
- Support the game modes that are phone-native: live-hosted, team,
  student-paced, and async homework.
- Give instructors mobile **host/present** and basic **kit authoring**.
- Surface per-student **results/reports**.
- Match web's moderation, safety, and accessibility guarantees on mobile.

## 3. Non-Goals

- Re-implementing any server logic (IQ.1–IQ.11 are shipped).
- Full desktop-class kit authoring on phones (bring core authoring; leave the
  richest editing to web/tablet — see Open Questions).
- Admin governance UI beyond linking (owned by
  [MOB.3](MOB.3-system-settings-parity.md)).
- Projector/"big screen" present mode fidelity beyond a functional host view.

## 4. Personas & User Stories

- **As a student (K-12)**, I want to join a live quiz with a code on my phone
  and answer fast so I can compete on the leaderboard.
- **As a self-learner**, I want to play an async-homework quiz at my own pace.
- **As an instructor**, I want to host/advance a live game from my phone while
  walking the room.
- **As an instructor**, I want to spin up a quick kit or import one from the
  library on mobile.
- **As an instructor**, I want to see who struggled after the game.

## 5. Functional Requirements

- **FR-1.** The app MUST let a user join by code:
  `lookupJoinCode` → `POST /api/v1/live-quizzes/join/{code}` (student) or
  guest join, with nickname, mirroring web's code → nickname → play flow.
- **FR-2.** The play surface MUST render and accept answers for all question
  types: `mc_single`, `mc_multiple`, `true_false`, `type_answer`, `numeric`,
  `poll`, `ordering`, `word_cloud`.
- **FR-3.** The app MUST reflect live game state in real time (question start,
  timer, reveal, leaderboard) over the live-game realtime channel, reusing the
  mobile WebSocket client.
- **FR-4.** Scoring MUST honour server scoring modes (`standard`, `double`,
  `no_points`) and show streaks/points/leaderboard as web does.
- **FR-5.** The app MUST support game modes: live-hosted, **team** (team assign),
  **student-paced** (`…/games/{id}/paced/start`), and **async homework**
  (assignment-based).
- **FR-6.** Instructors MUST be able to **host** a game from a kit and
  advance/end it (`…/games/{id}`, `…/end`), and see the join code + roster
  (`…/games/{id}/players`, rename).
- **FR-7.** Instructors MUST be able to browse course kits and the shared library
  (`…/live-quizzes/kits`, `/live-quizzes/library`, `/templates`) and start a game
  or an assignment (`…/live-quizzes/assignments`, `…/start`).
- **FR-8.** The app SHOULD support core kit authoring (create/edit kit,
  add questions, duplicate, save-as-template) — phased.
- **FR-9.** The app MUST show per-player results (`…/games/{id}/my-results`) and
  instructor reports (`…/games/{id}` report data).
- **FR-10.** Moderation & safety parity: nickname filtering, flag
  (`…/games/{id}/flag`), safety events (`…/safety`, `…/safety-events`), and the
  ability to remove/rename players; force-end respected.
- **FR-11.** All actions MUST respect the per-course interactive-quizzes feature
  flag and the relevant permissions.

## 6. Non-Functional Requirements

- **Performance** — answer submit round-trip p95 < 300 ms on 4G; question
  transitions render < 150 ms of the server event; smooth 60 fps countdown.
- **Security** — guest joins are code-scoped and rate-limited; authenticated
  joins carry the token; hosting/authoring permission-gated server-side; no
  answer keys leaked to players before reveal.
- **Privacy & Compliance** — guest nicknames filtered (COPPA-aware); no PII
  collected from guests beyond nickname; FERPA-safe reports.
- **Accessibility** — WCAG 2.1 AA: color-blind-safe answer tiles with
  shapes/labels, not color alone; timer has non-visual cues; reduced-motion
  honours AN.\*; screen-reader answer flow; adjustable timers respected.
- **Scalability** — one WS per player; server fan-out already sized (IQ.3).
- **Reliability** — reconnect mid-game rejoins the same player/session
  idempotently; late/dup answers rejected server-side; offline shows a clear
  "reconnecting" state.
- **Observability** — `live_quiz_{join,answer,reconnect,host_start,host_advance,end}`
  with mode + question type (no answer content).
- **Maintainability** — new `LMSAPILiveQuiz` (iOS) / `LiveQuizApi.kt` (Android)
  + a `LiveGameLogic` state machine shared in shape across platforms.
- **Internationalization** — `mobile.liveQuiz.*` keys.
- **Backward compatibility** — no API change.

## 7. Acceptance Criteria

- **AC-1.** *Given* a live game code, *when* a student joins with a nickname,
  *then* they enter the lobby and see the leaderboard when the host starts.
- **AC-2.** *Given* each of the 8 question types, *when* presented, *then* the
  student can answer and the answer is scored correctly per the scoring mode.
- **AC-3.** *Given* a dropped connection mid-game, *when* it recovers, *then* the
  player rejoins the same session without losing prior points.
- **AC-4.** *Given* an instructor, *when* they host a kit, *then* a join code is
  shown and they can advance and end the game from mobile.
- **AC-5.** *Given* a finished game, *when* a student opens results, *then* they
  see their score/rank; the instructor sees the report.
- **AC-6.** *Given* an inappropriate nickname, *then* it is filtered/blocked per
  safety rules.
- **AC-7.** *Given* the course flag is off, *then* no interactive-quiz UI is
  reachable.

## 8. Data Model

- **No new tables.** All quiz kits, questions, games, players, answers, and
  reports exist server-side (IQ.1–IQ.11). Client adds transient live-game state
  (current question, timer, local answer, leaderboard snapshot).

## 9. API Surface

Existing endpoints (reused). Key set:

- Join: `lookupJoinCode`, `POST /api/v1/live-quizzes/join/{code}`,
  `…/join/{code}/players`.
- Game: `…/courses/{code}/live-quizzes/games/{gameId}` (state/report), `…/end`,
  `…/players`, `…/players/{id}/rename`, `…/paced/start`, `…/teams/assign`,
  `…/flag`, `…/safety`, `…/safety-events`, `…/my-results`.
- Kits: `…/live-quizzes/kits`, `…/kits/{id}` (+ `/duplicate`,
  `/save-as-template`, `/shares`, `/submit-to-catalog`, `/games`, `/archive`,
  `/restore`).
- Assignments: `…/live-quizzes/assignments`, `…/assignments/{id}/start`.
- Library/templates: `/live-quizzes/library`, `/library/{kitId}/preview|import`,
  `/live-quizzes/templates`, `/templates/{id}/create-kit`.
- Realtime: live-game WS channel (reuse mobile `WebSocketClient`).

No new server routes.

## 10. UI / UX

- **New screens (both platforms):**
  - Join (code) → Nickname → Lobby → Play (answer surface per type) →
    Leaderboard → My results.
  - Host: kit picker → start game → live host controls (join code, advance,
    end, roster/moderation).
  - Kits: course kit list, library/templates browse, kit preview, core kit
    editor (phased).
  - Reports: instructor game report.
- **Flows:** deep link `/play/{code}` (and a scanned/entered code) → play;
  in-course "Live quizzes" tab → kits/games.
- **States:** lobby waiting, question active/answered/reveal, between-questions
  leaderboard, reconnecting, ended, empty (no kits), error, flag/removed.
- **Mobile/responsive:** big tappable answer tiles (shape + color + label);
  countdown ring; haptics on submit/reveal.
- **Accessibility:** non-color answer differentiation; VoiceOver/TalkBack order;
  reduced-motion; adjustable-timer support.
- **Copy & i18n:** `mobile.liveQuiz.*`.

## 11. AI / ML Considerations

- AI kit generation (IQ.10) MAY be surfaced on mobile as a later phase by
  calling the existing generation endpoint; no new model. PII redaction and cost
  budget already defined in IQ.10 — mobile inherits.

## 12. Integration Points

- iOS: new `Core/LMS/LMSAPILiveQuiz.swift` + `LiveQuizLogic.swift` /
  `LiveGameLogic.swift`; `Features/LiveQuiz/*`; reuse
  `Core/Realtime/WebSocketClient.swift`.
- Android: new `core/lms/LiveQuizApi.kt` + logic; `features/livequiz/*`; reuse
  `core/realtime`.
- Course workspace nav gains a "Live quizzes" destination (parity with web
  `live-quizzes` routes).
- Governance links to [MOB.3](MOB.3-system-settings-parity.md).

## 13. Dependencies & Sequencing

- Must ship after: IQ.1–IQ.11 (done).
- **Phase 1 — Student play:** join, answer surface (all types), leaderboard,
  reconnect, my-results, safety on nicknames.
- **Phase 2 — Modes + host:** team/student-paced/async; instructor host/present;
  roster moderation; flag/safety events.
- **Phase 3 — Kits & library:** browse course kits, library/templates, start
  game/assignment, core authoring, duplicate/save-as-template.
- **Phase 4 — Reports + AI:** instructor reports; optional AI generation.
- Shared infra: realtime WS gateway (exists).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Realtime latency/jitter on classroom Wi-Fi | H | H | Optimistic UI + server-authoritative reveal; reconnect-resume; load test |
| Cheating (client tampering) | M | H | Server-authoritative scoring/timing; never trust client answer timing |
| Accessibility of fast-timed answers | M | M | Adjustable timers; non-color cues; reduced-motion; screen-reader path |
| Guest abuse / bad nicknames (COPPA) | M | H | Server nickname filter + rate limit + host moderation (AC-6) |
| Authoring too heavy for phones | M | M | Phase authoring; keep rich editing on web/tablet |

## 15. Rollout Plan

- Flag: reuse per-course `ff_interactive_quizzes`; add `ff_mobile_live_quiz`
  client gate.
- Sequence: Phase 1 (play) behind flag → K-12 pilot classroom → enable phases.
- GA criteria: AC-1..7 pass; classroom load test green (30+ concurrent players);
  crash-free ≥ 99.5%.
- Rollback: client flag off hides all live-quiz UI.

## 16. Test Plan

- **Unit** — answer serialization per type; scoring-mode math; game state
  machine; reconnect/resume.
- **Integration** — join → play → results against a hosted test game; each mode.
- **End-to-end** — multi-device: host on one device, 2+ players on others.
- **Security** — guest rate limit; no pre-reveal answer leak; host permission
  gating; tamper attempts rejected.
- **Accessibility** — screen-reader answer flow; color-blind simulation;
  reduced-motion; adjustable timer.
- **Performance / load** — concurrent-player load test; answer latency.
- **Manual** — flaky-network reconnect; force-end; nickname moderation.

## 17. Documentation & Training

- Student "How to join a live quiz" (with code/QR).
- Instructor "Host a live quiz from your phone" + modes explainer.
- Accessibility notes (timers, cues).

## 18. Open Questions

1. How much kit authoring belongs on phone vs. tablet-only vs. web-only?
2. Do we support QR-code join (camera) in addition to typed codes in v1?
3. Present mode: is a phone-as-projector-controller enough, or do we need a
   dedicated large-screen host view?
4. Which realtime transport does the shipped live-game engine use, and does the
   mobile WebSocket client need protocol changes to match?

## 19. References

- Web: `clients/web/src/pages/live-quiz-play-page.tsx`,
  `clients/web/src/components/live-quiz/*`, `clients/web/src/lib/live-quiz-api.ts`,
  `live-quiz-api-schemas.ts`.
- Backend plans: `docs/completed/interactive-quizzes/IQ.1–IQ.11`.
- iOS realtime: `clients/ios/Lextures/Core/Realtime/WebSocketClient.swift`.
- Related: [MOB.3](MOB.3-system-settings-parity.md).
