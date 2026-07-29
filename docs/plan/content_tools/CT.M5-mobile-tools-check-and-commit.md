# CT.M5 — Mobile Tool Pack 1: Check & Commit (Inline Questions, Predict & Reveal, Class Pulse, Flashcards)

> Implementation plan. Source: mobile renderers for [CT.11](../../completed/content_tools/CT.11-tool-inline-questions.md), [CT.12](../../completed/content_tools/CT.12-tool-predict-and-reveal.md), [CT.21](../../completed/content_tools/CT.21-tool-class-pulse.md), [CT.23](../../completed/content_tools/CT.23-tool-flashcards-and-spaced-recall.md). Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.M5 |
| **Section** | Content Tools (CT) — Mobile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad |
| **Depends on** | CT.M3 |
| **Unblocks** | — (parallel with CT.M6, CT.M7, CT.M8) |

---

## 1. Problem Statement

CT.M3 gives mobile a host but only a probe renderer, so every real tool falls back to "Open in
browser". The four tools in this pack are the ones authors reach for most and the ones that suit a
phone best: they are tap-first, short-session, and produce small state documents. `inline_questions`
is the BLOCKER-severity formative check; `predict_reveal` and `class_pulse` are single-commit probes;
`flashcards` feeds the shipped spaced-repetition system that mobile students already use in the Review
tab. Shipping these four converts the majority of placed instances from a placeholder into a working
activity.

## 2. Goals

- Ship native iOS and Android renderers for `inline_questions`, `predict_reveal`, `class_pulse` and
  `flashcards` against the CT.M3 contract.
- Match web behaviour for scoring, reveal gating, aggregate display and SRS scheduling — the server is
  the authority in every case.
- Make each tool genuinely phone-native: large tap targets, thumb-reachable primary actions, haptics
  on commit, and no horizontal scrolling.
- Full offline support for the two tools that allow it (`inline_questions`, `flashcards` review) and an
  honest connection-required state for the two that need a server action.
- Meet the accessibility baseline per tool: labelled controls, announced results, and correctness never
  signalled by colour alone.

## 3. Non-Goals

- Authoring or configuring any of these tools on mobile (web-only).
- Instructor-facing aggregates, insights or heat maps (CT.7 stays web).
- Changing scoring, SRS scheduling, or aggregate computation — all server-side and unchanged.
- The other ten tools (CT.M6–CT.M8) and the sandbox path (CT.M4).
- Live/real-time class pulse via WebSocket — mobile polls the cached aggregate exactly as web does.

## 4. Personas & User Stories

- **As a student**, I want to answer the two-question check in my reading on my phone and see straight
  away whether I was right.
- **As a student**, I want to commit a prediction and my confidence before the answer is revealed, so
  the reveal actually teaches me something.
- **As a student**, I want to vote in a class poll during lecture from my phone and see how the class
  answered.
- **As a student**, I want to run the inline flashcard deck on the bus and have it count toward my
  review streak.
- **As an instructor**, I want the same evidence from phone users as from laptop users.
- **As a screen-reader user**, I want to know I answered correctly without relying on a green tint.

## 5. Functional Requirements

**Shared**

- **FR-1.** Each renderer MUST register in CT.M3's registry by stable `toolId` and implement the host
  interface only — no direct networking, no direct persistence.
- **FR-2.** Each renderer MUST honour the frame's `readOnly` state, the declared conflict policy
  (`server_wins` for `inline_questions`, `predict_reveal`, `class_pulse`; `merge` for `flashcards`),
  and MUST never write `score` — scoring is server-side (`scoring.mode: auto`).
- **FR-3.** Every result, score and reveal change MUST be announced through the host's live region.

**`inline_questions`** (caps: `state`, `scoring`)

- **FR-4.** MUST render the configured 1–2 questions with their supported response types, using the
  CT.M1 renderer for prompts and choices so tables, code and math inside a question work.
- **FR-5.** MUST call the `submit` action to grade and the `reveal` action to disclose the answer,
  honouring the config's attempt limit and reveal policy; the client MUST NOT compute correctness
  locally or hold the answer key.
- **FR-6.** MUST show per-question feedback and rationale after grading, rendered through CT.M1, with
  correctness conveyed by icon + text as well as colour.
- **FR-7.** MUST support offline answering: responses persist locally, submit queues, and the UI states
  plainly that grading happens on reconnect.

**`predict_reveal`** (caps: `state`, `aggregate`)

- **FR-8.** MUST take a prediction plus a confidence rating and commit it with the `commit` action;
  once committed the prediction MUST be immutable and the reveal MUST become available.
- **FR-9.** MUST NOT reveal the answer before commit under any client state — including on relaunch,
  after backgrounding, or offline.
- **FR-10.** MUST support the post-reveal `reflect` step and, where the config enables it, show the
  anonymised class distribution.

**`class_pulse`** (caps: `state`, `aggregate`)

- **FR-11.** MUST cast a vote via the `vote` action and then display the anonymised distribution from
  the `aggregate` action.
- **FR-12.** MUST poll the cached aggregate on a backoff while the tool is on screen and stop polling
  when it is not — never a WebSocket, never a tight poll.
- **FR-13.** MUST suppress the distribution until the viewer has voted where the config requires it,
  and MUST hide small-N distributions per the server's own thresholds (the client displays what the
  server returns and never reconstructs counts).
- **FR-14.** MUST require a connection to vote, with a clear "needs connection" state (actions are not
  queued — CT.M3 FR-11).

**`flashcards`** (caps: `state`)

- **FR-15.** MUST run a review session through `start_session` → `rate` (per card) → `end_session`, and
  MUST read `status` for due counts; scheduling is server-side.
- **FR-16.** MUST render card fronts/backs through CT.M1 (math and code on cards are common) and
  support a flip gesture plus an accessible flip button.
- **FR-17.** MUST expose the rating scale as discrete labelled buttons, not a swipe-only interaction.
- **FR-18.** MUST reconcile with the existing mobile Review/SRS surface so an inline deck contributes
  to the same streak and due counts the student sees there, without double-counting.
- **FR-19.** MUST support offline review with `merge` conflict semantics: ratings queue in order and
  replay on reconnect.

## 6. Non-Functional Requirements

- **Performance** — Each renderer mounts in ≤ 30 ms; a flashcard flip animates at 60 fps; class-pulse
  aggregate polling costs at most one request per 15 s while visible, with backoff.
- **Security** — No answer key, no correct-answer field, and no aggregate raw counts ever reach the
  client except as the server chooses to return them; all grading and thresholding is server-side.
- **Privacy & Compliance** — Votes and predictions are education records; class distributions are
  anonymised server-side. Nothing identifies a peer. Offline state lives in the CT.M3 encrypted cache
  and is purged on sign-out.
- **Accessibility** — WCAG 2.1 AA: every choice is one control with a complete label; correctness is
  icon + text + colour; results announced politely; flip has a button equivalent; rating buttons meet
  target size; distributions expose their values as text, not chart geometry alone.
- **Scalability** — Class Pulse is the hot path (a whole lecture voting at once); mobile relies on the
  server's aggregate cache and MUST NOT poll faster than the documented interval.
- **Reliability** — Committed predictions are immutable; a failed submit never loses the typed
  response; queued ratings replay in order.
- **Observability** — Per-tool mount, interaction, submit-outcome and offline-replay counters labelled
  `tool_id`.
- **Internationalization** — `mobile.contentTools.tools.{toolId}.*` keys in all five locale files;
  RTL layouts for choices, distributions and cards; confidence and rating scales localised.
- **Backward compatibility** — Unknown config fields ignored; unknown state fields preserved on write
  so a newer web client's state is never truncated by an older mobile client.

## 7. Acceptance Criteria

- **AC-1.** *Given* an `inline_questions` instance, *When* a student answers and submits, *Then* the
  server grades it, per-question feedback appears, the score chip updates, and the result is announced.
- **AC-2.** *Given* an attempt limit of 1, *When* the student submits, *Then* re-answering is blocked
  client-side and a second submit is rejected server-side.
- **AC-3.** *Given* a question containing a table and inline math, *When* it renders, *Then* both
  render (CT.M1 integration).
- **AC-4.** *Given* offline, *When* a student answers `inline_questions`, *Then* the response persists,
  the UI says grading is pending, and on reconnect the submit runs and feedback appears.
- **AC-5.** *Given* `predict_reveal` before commit, *When* the student force-quits and relaunches,
  *Then* the answer is still hidden.
- **AC-6.** *Given* a committed prediction, *Then* it cannot be edited and the reveal is available.
- **AC-7.** *Given* `class_pulse` and a student who has not voted, *When* the config requires voting
  first, *Then* no distribution is shown.
- **AC-8.** *Given* a vote attempted offline, *Then* a "needs connection" state appears and nothing is
  queued.
- **AC-9.** *Given* `class_pulse` on screen for 10 minutes, *Then* aggregate requests respect the
  backoff and stop entirely when the tool leaves the screen.
- **AC-10.** *Given* a `flashcards` deck, *When* a student reviews 5 cards and ends the session, *Then*
  ratings post in order, the due count updates, and the Review tab reflects the same session with no
  double-count.
- **AC-11.** *Given* offline flashcard review, *When* the student reconnects, *Then* ratings replay in
  order and the resulting schedule matches a same-order online session.
- **AC-12.** *Given* VoiceOver/TalkBack on each of the four tools, *Then* controls are labelled,
  results are announced, and no interaction requires colour perception or a swipe-only gesture.
- **AC-13.** *Given* a read-only frame (archived, past due, observer), *Then* each tool renders its
  state without any write affordance.
- **AC-14.** *Given* CI, *Then* iOS build, Android compile and the renderer logic suites pass.

## 8. Data Model

**No server schema change, no migration.** Each tool's config and state are JSON documents validated
server-side against its manifest (`server/internal/service/contenttools/tools/{toolId}/manifest.json`).
Mobile models mirror those schemas as typed structs, with unknown-field preservation (NFR) so state
written by a newer client survives a mobile round-trip. No new cache keys beyond CT.M3's.

## 9. API Surface

**No new endpoints.** All four use CT.M3's state routes plus their registered actions:

| Tool | Actions |
|---|---|
| `inline_questions` | `submit`, `reveal` |
| `predict_reveal` | `commit`, `reflect` |
| `class_pulse` | `vote`, `aggregate` |
| `flashcards` | `start_session`, `rate`, `end_session`, `status` |

All dispatched via `POST /api/v1/courses/{course_code}/content-tools/instances/{instance_id}/actions/
{action}` with an idempotency key, per CT.M3 FR-9.

## 10. UI / UX

- **New (iOS)** — `Features/ContentTools/Tools/{InlineQuestionsToolView,PredictRevealToolView,
  ClassPulseToolView,FlashcardsToolView}.swift`, plus `Core/LMS/ContentToolPack1Logic.swift` (pure:
  attempt gating, reveal gating, poll backoff, rating queue ordering — unit-tested).
- **New (Android)** — `features/contenttools/tools/{InlineQuestionsTool,PredictRevealTool,
  ClassPulseTool,FlashcardsTool}.kt`, `core/lms/ContentToolPack1Logic.kt`.
- **Modified** — CT.M3's `ToolRendererRegistry` gains four entries; the Review/SRS surface gains the
  reconciliation from FR-18.
- **Key flows** — (1) Read → answer inline check → submit → feedback. (2) Predict + confidence →
  commit → reveal → reflect. (3) Vote → distribution. (4) Start deck → flip → rate → next → summary.
- **States** — *Not started*: the tool's idle prompt. *Answered, ungraded*: pending chip. *Graded*:
  feedback + score. *Committed*: locked prediction. *Voted*: distribution or "waiting for the class".
  *Offline*: unsynced chip (questions/flashcards) or needs-connection (pulse/commit). *Read-only*:
  values shown, controls disabled with a reason.
- **Accessibility annotations** — choices are single controls with merged labels; the flashcard flip is
  a button with state in its label; the distribution exposes "{option}: {percent}%" as text; result
  announcements are polite, errors assertive.
- **Copy & i18n** — `mobile.contentTools.tools.inline_questions.*`, `.predict_reveal.*`,
  `.class_pulse.*`, `.flashcards.*`.

## 11. AI / ML Considerations

None of these four declare the `ai` capability. Grading is deterministic and server-side; no model call
is made from this pack. (AI-backed tools are CT.M6.)

## 12. Integration Points

- **Internal** — CT.M3 host, state store and live region; CT.M1 renderer for prompts, choices, feedback
  and card faces; the existing mobile Review/SRS feature (`Features/Review/ReviewSessionView.swift`,
  `features/review/*`) for FR-18; haptics helpers (`Core/Design/Haptics.swift`).
- **Server (unchanged)** — `server/internal/service/contenttools/{inline_questions,predict_reveal,
  class_pulse,flashcards}_actions.go` and the SRS service the flashcards actions feed.
- **Events** — server emits CT.7 analytics on write; no client emission.

## 13. Dependencies & Sequencing

- Must ship after: **CT.M3**.
- Independent of CT.M4 and of the other packs; can ship tool-by-tool.
- Recommended order within the pack: `inline_questions` (highest usage, BLOCKER on web) →
  `flashcards` (reuses the Review surface) → `predict_reveal` → `class_pulse` (needs the poll budget
  settled).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Reveal leaks before commit through a client state we did not consider | M | H | Reveal gated on server-confirmed commit, not local state; explicit relaunch/offline ACs |
| Flashcard ratings double-count against the Review streak | M | M | FR-18 reconciliation designed with the SRS owner; integration test asserting single-count |
| Class Pulse polling storms during a live lecture | M | H | Visible-only polling with backoff, server aggregate cache, documented interval, and a load test |
| Offline replay produces a different SRS schedule than online | M | M | Ordered per-instance replay; server recomputes the schedule from the rating sequence |
| Small-N distributions let a student infer a peer's vote | L | H | Client never reconstructs counts; thresholding stays server-side |
| Rich content inside choices breaks tap targets | M | M | CT.M2's FR-3 pattern reused: one control per choice with a merged label |

## 15. Rollout Plan

- **Feature flag** — reuse `mobileContentToolsEnabled`, plus a per-tool client allowlist so a single
  renderer can be disabled (falling back to CT.M3's placeholder) without a release.
- **Sequencing** — one renderer per PR in the recommended order, each behind its allowlist entry.
- **Dogfood** — an internal course with all four placed on one content page.
- **GA criteria** — all ACs green per tool; no state-loss findings; a11y sign-off per tool; Class Pulse
  load test passed.
- **Rollback** — remove the tool from the client allowlist; the placeholder returns; learner state is
  untouched.

## 16. Test Plan

- **Unit** — attempt-limit and reveal gating; commit immutability; poll backoff and visibility
  lifecycle; rating queue ordering and replay; unknown-field preservation on state write.
- **Integration** — each action round-trip against a seeded server, including 409/413/422 and
  read-only paths; flashcard session reconciliation with the Review surface.
- **End-to-end (device)** — answer/submit/reveal; predict/commit/reveal/reflect with a relaunch in the
  middle; vote and observe the distribution; a 10-card offline deck replayed on reconnect.
- **Security** — verify no answer key or raw counts in any response the client receives; verify a
  second submit past the attempt limit is rejected; verify no peer identity in pulse data.
- **Accessibility** — scripted screen-reader pass per tool; colour-blind simulation on correctness and
  distributions; 200% font scale; RTL.
- **Performance / load** — Class Pulse with a simulated 300-student lecture; flashcard flip frame rate.
- **Manual exploratory** — mid-submit token refresh, force-quit before commit, airplane mode toggling
  during a deck, tablet layout, very long choice text.

## 17. Documentation & Training

- End-user: "Activities in your reading" gains per-tool notes (what saves offline, what needs signal).
- Instructor: mobile support matrix updated — these four are native.
- Internal: renderer-authoring notes in the mobile README (how to add a tool to the registry).

## 18. Open Questions

1. What is the sanctioned Class Pulse poll interval and backoff curve, given the server aggregate
   cache? (Owner: backend platform; needed before the Class Pulse PR.)
2. Does the inline flashcard deck feed the *same* SRS queue as the Review tab, or a scoped one? (Read
   the CT.23 decision and mirror it exactly; do not re-decide on mobile.)
3. Should `inline_questions` allow offline *grading* of objective types for instant feedback?
   (Recommendation: no — it would put the answer key on the device.)
4. Does `predict_reveal` show the class distribution on mobile when the config enables it, or is that
   web-only for screen-size reasons? (Recommendation: show it; it is a small bar list.)

## 19. References

- Web plans: [CT.11](../../completed/content_tools/CT.11-tool-inline-questions.md),
  [CT.12](../../completed/content_tools/CT.12-tool-predict-and-reveal.md),
  [CT.21](../../completed/content_tools/CT.21-tool-class-pulse.md),
  [CT.23](../../completed/content_tools/CT.23-tool-flashcards-and-spaced-recall.md).
- Web renderers: `clients/web/src/components/content-tools/tools/{inline_questions,predict_reveal,
  class_pulse,flashcards}/renderer.tsx`.
- Server: `server/internal/service/contenttools/{inline_questions,predict_reveal,class_pulse,
  flashcards}_actions.go`; manifests under `server/internal/service/contenttools/tools/*/manifest.json`.
- Related plans: [CT.M1](CT.M1-mobile-markdown-engine-tables-code-math.md),
  [CT.M3](CT.M3-mobile-content-tool-host-and-state.md),
  [CT.M9](CT.M9-mobile-tools-governance-a11y-telemetry.md).
- Standards: WCAG 2.1 AA §1.4.1 (use of colour), §2.5.1 (pointer gestures — flip must have a
  non-gesture equivalent), §4.1.3 (status messages).
