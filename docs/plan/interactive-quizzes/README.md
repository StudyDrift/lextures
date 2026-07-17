# Interactive Quizzes — Implementation Plans

> Goal: ship an in-house, real-time **game-based quizzing** experience so institutions no longer pay a
> separate third-party per-seat licence for live classroom quiz games. A teacher builds a **quiz kit** (a
> reusable set of timed questions with media), then **hosts a live game**: students join from any device
> with a short **join code**, answer against a countdown, earn points for speed + accuracy, and climb a
> live **leaderboard** shown on the projector. The same kit can also be assigned as **student-paced** or
> **async homework**. It is delivered as a **per-course feature flag** — the same on/off model as the
> Whiteboard (`whiteboard_enabled`), Collaborative Documents (`collab_docs_enabled`), and Collaboration
> Boards (`visual_boards_enabled`) apps.

## Why this folder exists

Lextures already ships a deep **static assessment** stack we build on, but nothing for **live, gamified,
whole-class play**:

- **Question bank** (`course.questions`, migration `075`, flag `question_bank_enabled`) — a reusable pool of
  questions with a rich `course.question_type` enum (`mc_single`, `mc_multiple`, `true_false`,
  `short_answer`, `numeric`, `matching`, `ordering`, `hotspot`, `formula`, `code`, …) plus pools
  (`course.question_pools`) and per-attempt sampling. Interactive Quizzes **reuses** this bank as a question
  source rather than reinventing item authoring.
- **Module quizzes & attempts** (`course.quiz_attempts`, `coursemodulequizzes`, `quiz_delivery_http.go`) — a
  self-paced, individually-timed, gradebook-integrated delivery engine. This is the *homework quiz* surface;
  it is **not** a synchronous, host-driven, leaderboard-based classroom game.
- **Item analysis** (`itemanalysis`), **grade sync** (`quiz_grade_sync.go`), **learner progress**
  (`learnerprogress`) — analytics/gradebook plumbing Interactive Quizzes plugs into.
- **Collaborative Documents** (`collab_docs_ws.go`) — a **Y.js CRDT** WebSocket relay. It proves our
  WebSocket auth/room plumbing, but a game show is **authoritative** (the server owns the truth, the host
  drives the clock, players submit answers) — **not** peer-merged CRDT. IQ.3 therefore builds a purpose-built
  authoritative game hub reusing the WS *transport* but not the CRDT *model*.

The gap these plans close: a **synchronous, host-driven, competitive quiz game** with join codes, live
leaderboards, speed-based scoring, team play, and celebratory feedback — plus student-paced and async modes,
rich reports, sharing, moderation, accessibility, and AI generation.

## Product naming

- **User-facing:** "Live Quizzes" (menu label). Umbrella term across modes: "Interactive Quizzes". A saved
  template is a **quiz kit**; a hosted instance is a **game**; students join with a **join code** and pick a
  **nickname**. (Deliberately distinct from static "Quizzes" (module quizzes) and "Live Sessions" (plan 6.4).)
- **Internal id / flag:** `interactive_quizzes` — per-course column
  `course.courses.interactive_quizzes_enabled` (default `FALSE`), plus a platform master flag
  `FFInteractiveQuizzes` (`settings.platform_app_settings.ff_interactive_quizzes`) in
  `server/internal/repos/platformconfig/features.go`.
- **Feature-ID prefix:** `IQ` (Interactive Quizzes), mirroring `VC`/`AP.#`/`W##`/`M##`/`S##`.
- **Schema:** new tables live in a dedicated `quizgame` Postgres schema (`quizgame.kits`,
  `quizgame.questions`, `quizgame.sessions`, …), keeping the surface self-contained and droppable behind the
  flag — the same containment strategy the `board` schema uses.

## Conventions

- **File naming:** `IQ.{N}-{kebab-slug}.md`. Every plan follows [`../_TEMPLATE.md`](../_TEMPLATE.md).
- A plan is **ready** when every template section is filled (no `…` placeholders).
- **Migrations** continue the repo's global sequence. Boards currently consume `378_*`–`380_*` (and reserve
  through ~`385`), so these plans reserve `390_*` onward; each plan states its number. **Renumber on merge**
  if the sequence has advanced.
- **HTTP:** handlers in `server/internal/httpserver/quizgame_*.go`, repos in
  `server/internal/repos/quizgame/`, routes under `/api/v1/courses/{course_code}/live-quizzes/*` and a
  public join surface under `/api/v1/live-quizzes/join/*`, using `apierr.WriteJSON`, `requireCourseAccess`,
  and `courseroles.UserHasPermission` exactly as the module-quiz / board handlers do.
- **Web:** page components in `clients/web/src/pages/lms/live-quiz-*.tsx`, shared UI in
  `clients/web/src/components/live-quiz/`, API client in `clients/web/src/lib/live-quiz-api.ts`, flag
  surfaced through `clients/web/src/context/course-nav-features-context.tsx` and toggled in
  `clients/web/src/pages/lms/course-features-section.tsx`.

## Severity legend

- **BLOCKER** — an institution cannot retire its incumbent live-quiz tool (and its per-seat spend) without it.
- **MAJOR** — parity gap that loses the head-to-head evaluation.
- **MINOR** — polish / nice-to-have / defence-in-depth.

## Story index

| ID | Plan | Severity | Depends on | Est. |
|---|---|---|---|---|
| IQ.1 | ~~Foundation, data model & feature flag~~ → [completed](../../completed/interactive-quizzes/IQ.1-foundation-and-feature-flag.md) | BLOCKER | — | M |
| IQ.2 | ~~Quiz-kit authoring & question types~~ → [completed](../../completed/interactive-quizzes/IQ.2-kit-authoring-and-question-types.md) | BLOCKER | IQ.1 | L |
| IQ.3 | ~~Live game hosting engine (real-time)~~ → [completed](../../completed/interactive-quizzes/IQ.3-live-game-hosting-engine.md) | BLOCKER | IQ.1, IQ.2 | L |
| IQ.4 | ~~Player join & gameplay experience~~ → [completed](../../completed/interactive-quizzes/IQ.4-player-join-and-gameplay.md) | BLOCKER | IQ.3 | L |
| IQ.5 | ~~Scoring, leaderboards & game mechanics~~ → [completed](../../completed/interactive-quizzes/IQ.5-scoring-leaderboards-mechanics.md) | MAJOR | IQ.3, IQ.4 | M |
| IQ.6 | ~~Game modes: team, student-paced & async homework~~ → [completed](../../completed/interactive-quizzes/IQ.6-game-modes-team-paced-async.md) | MAJOR | IQ.3, IQ.4 | L |
| IQ.7 | ~~Reports, results & gradebook/analytics~~ → [completed](../../completed/interactive-quizzes/IQ.7-reports-results-gradebook.md) | MAJOR | IQ.3 | M |
| IQ.8 | ~~Content library, templates, sharing & discovery~~ → [completed](../../completed/interactive-quizzes/IQ.8-library-templates-sharing.md) | MAJOR | IQ.1, IQ.2 | M |
| IQ.9 | ~~Moderation, safety, accessibility & fair play~~ → [completed](../../completed/interactive-quizzes/IQ.9-moderation-safety-accessibility.md) | BLOCKER (K12) | IQ.3, IQ.4 | M |
| IQ.10 | ~~AI-assisted quiz generation~~ → [completed](../../completed/interactive-quizzes/IQ.10-ai-assisted-generation.md) | MAJOR | IQ.2 | M |
| IQ.11 | ~~Admin governance, quotas, analytics & lifecycle~~ → [completed](../../completed/interactive-quizzes/IQ.11-admin-governance-quotas-lifecycle.md) | MAJOR | IQ.1 | M |

## Recommended sequencing

1. **IQ.1** ships the flag, schema, kit list, and an empty "Live Quizzes" page — nothing else lands without it.
2. **IQ.2 → IQ.3 → IQ.4** are the MVP: author a kit, host it live, students join and play. These three let a
   class replace its incumbent tool for the flagship "host a live quiz on the projector" use case.
3. **IQ.5** makes scoring and leaderboards feel great; it can land close behind the MVP.
4. **IQ.9** must ship **before** any public join surface is exposed to minors (never open guest join without
   nickname moderation and anti-cheat controls).
5. **IQ.6, IQ.7, IQ.8, IQ.10, IQ.11** are parity/expansion layers that can land in any order once the MVP is
   stable.

## Cross-cutting requirements (apply to every plan)

- **Privacy / FERPA / COPPA:** student answers, nicknames, and scores are education records; deletion,
  export, and retention must honour the shipped compliance engines (see [`../standards/`](../standards/) —
  especially [S01 DSAR](../standards/S01-unified-data-subject-rights-orchestration.md),
  [S02 retention](../standards/S02-data-retention-deletion-engine.md), and
  [S08 children's privacy](../standards/S08-childrens-privacy-age-assurance-design-codes.md)). Guest joins by
  under-13s require the same age-assurance guardrails as any child-facing surface.
- **Accessibility:** WCAG 2.1 AA on every surface — answers distinguishable by **shape + label**, not colour
  alone; keyboard-operable answering; ARIA live regions for countdowns and results; reduced-motion and
  no-flashing options (photosensitivity); captions for question media.
- **Internationalization:** all copy externalised to the web i18n catalog; locale/timezone-aware timestamps
  and number/score formatting.
- **Observability:** metrics, traces, and structured logs via `server/internal/telemetry` — game hub gauges
  (live games, connected players), answer-latency histograms, and reconnection counters.
- **Reuse first:** the **question bank** (`course.questions`) is the canonical item store; Live Quizzes
  imports/links rather than duplicating item authoring where practical. Gradebook writes go through the
  existing `coursegrades` path; item analytics reuse `itemanalysis`.
