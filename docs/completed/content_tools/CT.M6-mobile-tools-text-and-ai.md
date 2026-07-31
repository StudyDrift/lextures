# CT.M6 — Mobile Tool Pack 2: Text & AI (Ask Questions, Explain It Back, Inline Discussion)

> Implementation plan. Source: mobile renderers for [CT.10](CT.10-tool-ask-questions.md), [CT.20](CT.20-tool-explain-it-back.md), [CT.22](CT.22-tool-inline-discussion.md). Folder overview: [README](../../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.M6 |
| **Section** | Content Tools (CT) — Mobile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad |
| **Depends on** | CT.M3 (web: CT.6, CT.8) |
| **Unblocks** | — (parallel with CT.M5, CT.M7, CT.M8) |

---

## 1. Problem Statement

The three tools in this pack are where a student *writes* inside the reading: asking the grounded AI
about the passage in front of them, explaining a concept back in their own words for AI formative
feedback, and arguing with classmates in a thread anchored to a paragraph. They are also the pack with
the sharpest mobile stakes — free-text entry on a phone keyboard, streaming responses over flaky
mobile networks, AI disclosure and consent, and peer-visible content that needs reporting and
moderation controls a student can actually reach. Without them, the moment-of-confusion question that
Content Tools exists to capture is a browser-only feature, which on a phone means it does not happen.

## 2. Goals

- Ship native renderers for `ask_questions`, `explain_it_back` and `inline_discussion` against the
  CT.M3 contract.
- Make free-text entry good on a phone: keyboard-aware layout, draft preservation across
  backgrounding, no lost work on a dropped connection.
- Handle AI honestly: visible disclosure, consent gating, budget/rate-limit errors explained in plain
  language, graceful degradation on provider failure with the transcript intact.
- Ship peer-visible safety controls in reach: report, and moderate for staff.
- Meet accessibility for conversational UI: announced arrivals without focus theft, readable
  transcripts, and no reliance on hover or gesture.

## 3. Non-Goals

- Prompt engineering, model choice, grounding strategy or evaluation — all server-side and unchanged
  (CT.6/CT.10/CT.20 own them).
- Instructor analytics: response clustering, misconception surfaces, discussion insights (CT.7, web).
- Authoring or configuring these tools on mobile.
- Replacing the course-wide AI tutor (`Features/Tutor`) — this is scoped, in-place Q&A about *this*
  activity, and reuses tutor chat patterns without merging surfaces.
- Voice input beyond what the OS keyboard already offers.

## 4. Personas & User Stories

- **As a student**, I want to ask "what does this paragraph mean?" right where I am confused, on my
  phone, and get an answer grounded in this activity.
- **As a student**, I want to type my own explanation and get feedback on it without leaving the page.
- **As a student**, I want to see and reply to what my classmates said about this paragraph.
- **As a student**, I want my half-written answer to still be there after a phone call interrupts me.
- **As a TA**, I want to endorse a good reply and hide an abusive one from my phone.
- **As a parent**, I want to know when my child's work is being sent to an AI model.
- **As a screen-reader user**, I want new messages announced without losing my place in the text field.

## 5. Functional Requirements

**Shared**

- **FR-1.** Each renderer MUST register in CT.M3's registry and use only the host's state and action
  APIs.
- **FR-2.** Free-text inputs MUST preserve drafts locally across backgrounding, rotation, low-memory
  eviction and app restart, and MUST NOT lose a draft when an action fails.
- **FR-3.** The composer MUST be keyboard-aware: the input stays visible above the keyboard, the send
  control is thumb-reachable, and the enclosing page does not jump.
- **FR-4.** All rendered text — questions, AI responses, peer posts — MUST go through the CT.M1
  renderer so code, math and tables in responses display correctly.
- **FR-5.** Actions MUST NOT be queued offline (CT.M3 FR-11); an offline composer MUST keep the draft
  and show a clear "will send when you are back online" affordance that the student triggers.
- **FR-6.** Conflict policy MUST be honoured as declared: `merge` for `ask_questions` and
  `explain_it_back`, `server_wins` for `inline_discussion`.

**`ask_questions`** (caps: `state`, `ai`, `network`)

- **FR-7.** MUST dispatch the `ask` action and render the grounded answer with its citations/sources as
  returned by the server; the client MUST NOT fetch or resolve links itself (SSRF safety lives in CT.6).
- **FR-8.** MUST render the AI disclosure required by CT.M9 before the first question, and MUST block
  the composer where AI consent is required and not granted.
- **FR-9.** MUST support the `clear` action to reset the transcript, with confirmation.
- **FR-10.** MUST handle streaming or long responses without blocking the enclosing scroll view, and
  MUST show a cancellable in-flight state.
- **FR-11.** MUST distinguish and explain, in plain language, the error classes the server returns:
  rate limited, budget exhausted, provider unavailable, content filtered — never a raw error code.

**`explain_it_back`** (caps: `state`, `ai`)

- **FR-12.** MUST accept a free-text explanation and submit it via the `submit` action, rendering the
  returned formative feedback; the client MUST NOT score.
- **FR-13.** MUST show attempt history where the config allows revision, with each attempt's feedback.
- **FR-14.** MUST display the instructor note (`instructor_note` action output) when present.
- **FR-15.** MUST enforce the configured minimum/maximum length client-side as guidance before submit,
  while treating the server's validation as authoritative.

**`inline_discussion`** (caps: `state`, `peer_visible`)

- **FR-16.** MUST render the thread via the `thread` action with pagination, and post replies via
  `post`; `get_post` MUST back deep links into a single post.
- **FR-17.** MUST support `edit` and `delete` for the student's own posts, and `endorse` and `moderate`
  for staff, hiding controls the viewer is not entitled to and handling a server `403` gracefully.
- **FR-18.** MUST support `upvote` and `report`, with report reachable in at most two taps from any
  post.
- **FR-19.** MUST reflect server-side moderation states (hidden, removed, flagged) rather than
  rendering removed content; a moderated post MUST show its tombstone, not its body.
- **FR-20.** MUST refresh the thread on foreground and on pull-to-refresh; no WebSocket in v1.
- **FR-21.** MUST show authorship per the tool's configured attribution (named or anonymous) exactly as
  the server returns it, and MUST NOT expose any identity the server withheld.

## 6. Non-Functional Requirements

- **Performance** — Composer input latency imperceptible (no re-render of the transcript per
  keystroke); thread pages of 20 render in ≤ 150 ms; an in-flight AI action never blocks scrolling.
- **Security** — No prompt, model id, key or grounding corpus on the device; peer content is rendered
  as markdown only (never HTML); authorization is server-enforced and the client merely hides controls;
  report/moderate calls are authenticated actions, not client-side state changes.
- **Privacy & Compliance** — Student free text is sent to a model: disclosure (S13/EU AI Act) and
  consent (CT.8/CT.M9) gate the composer; AI usage is logged server-side to `analytics.ai_usage_log`;
  transcripts are education records subject to DSAR (S01) and retention (S02); COPPA-relevant courses
  rely on the server's consent state, never a client default. Drafts stored on-device live in the
  CT.M3 encrypted cache and are purged on sign-out.
- **Accessibility** — WCAG 2.1 AA: transcript is a labelled list with per-message authorship in the
  accessible name; new messages announced politely without moving focus out of the composer; in-flight
  state exposed via `aria-busy`-equivalent semantics; report/moderate are labelled buttons, never
  icon-only without a name; 200% font scale in a chat layout.
- **Scalability** — Thread pagination server-side; AI calls rate-limited per manifest (default 10/min
  for AI-backed actions) on top of the platform limiter.
- **Reliability** — A failed AI call preserves the question and the transcript and offers retry with
  the same idempotency key; a failed post preserves the draft; no action is silently dropped.
- **Observability** — Counters for composer opens, action outcomes by error class, AI latency,
  report submissions, and draft recoveries — labelled `tool_id`.
- **Internationalization** — `mobile.contentTools.tools.{ask_questions,explain_it_back,
  inline_discussion}.*`; RTL chat bubbles and thread indentation; localized relative timestamps.
- **Backward compatibility** — Unknown state/config fields preserved; unknown moderation states render
  as a generic tombstone rather than exposing content.

## 7. Acceptance Criteria

- **AC-1.** *Given* an `ask_questions` instance, *When* a student asks a question, *Then* the grounded
  answer and its sources render, and the AI disclosure was visible beforehand.
- **AC-2.** *Given* a course requiring AI consent that has not been granted, *Then* the composer is
  blocked with an explanation and no model call is made.
- **AC-3.** *Given* the AI budget is exhausted, *Then* the student sees a plain-language message (not a
  code) and the transcript is intact.
- **AC-4.** *Given* an AI call that times out, *When* the student retries, *Then* the same idempotency
  key is reused and no duplicate spend occurs.
- **AC-5.** *Given* a half-typed explanation, *When* the app is backgrounded and killed by the OS,
  *Then* the draft is restored on relaunch.
- **AC-6.** *Given* `explain_it_back`, *When* a student submits, *Then* server-generated formative
  feedback renders and the client sets no score.
- **AC-7.** *Given* an `inline_discussion` thread, *When* a student posts a reply, *Then* it appears in
  the thread and persists across a refresh.
- **AC-8.** *Given* another student's post, *Then* edit and delete controls are absent, and a forced
  API call returns 403 handled gracefully.
- **AC-9.** *Given* any post, *Then* Report is reachable within two taps and submits successfully.
- **AC-10.** *Given* a server-moderated post, *Then* a tombstone renders and the original body is not
  present in the client payload.
- **AC-11.** *Given* an anonymous-attribution discussion, *Then* no author identity appears anywhere in
  the UI or the payload.
- **AC-12.** *Given* offline, *When* the student types and tries to send, *Then* the draft is kept, a
  clear offline state appears, and nothing is queued.
- **AC-13.** *Given* VoiceOver/TalkBack, *When* a response arrives, *Then* it is announced politely and
  focus stays in the composer.
- **AC-14.** *Given* an AI response containing code and math, *Then* both render via CT.M1.
- **AC-15.** *Given* a read-only frame, *Then* transcripts and threads render with no composer.
- **AC-16.** *Given* CI, *Then* iOS build, Android compile and the renderer logic suites pass.

## 8. Data Model

**No server schema change, no migration.** State documents follow each tool's manifest
(`server/internal/service/contenttools/tools/{ask_questions,explain_it_back,inline_discussion}/
manifest.json`). Discussion posts are returned by the `thread`/`get_post` actions, not by a new
endpoint. Client-side drafts are stored per `instanceId` in the CT.M3 encrypted cache, keyed
separately from state so a draft is never mistaken for saved work.

## 9. API Surface

**No new endpoints.** CT.M3's state routes plus these registered actions:

| Tool | Actions |
|---|---|
| `ask_questions` | `ask`, `clear` |
| `explain_it_back` | `submit`, `instructor_note`, `test_sample` (staff) |
| `inline_discussion` | `post`, `edit`, `delete`, `thread`, `get_post`, `upvote`, `endorse`, `report`, `moderate` |

Plus CT.M9's consent and report routes (`…/content-tools/ai-consent`,
`…/content-tools/instances/{instance_id}/report`) where the frame surfaces them.

## 10. UI / UX

- **New (iOS)** — `Features/ContentTools/Tools/{AskQuestionsToolView,ExplainItBackToolView,
  InlineDiscussionToolView}.swift`, shared `ToolComposerView.swift` (keyboard-aware input + draft
  persistence), `Core/LMS/ContentToolPack2Logic.swift` (pure: draft lifecycle, error classification,
  permission-derived control visibility, pagination cursoring — unit-tested).
- **New (Android)** — `features/contenttools/tools/{AskQuestionsTool,ExplainItBackTool,
  InlineDiscussionTool}.kt`, `features/contenttools/ToolComposer.kt`,
  `core/lms/ContentToolPack2Logic.kt`.
- **Reused patterns** — `Features/Tutor/TutorChatView.swift` and `features/tutor/TutorChatScreen.kt`
  are the reference for transcript layout and streaming behaviour; the code is reused as patterns, not
  as a shared surface.
- **Key flows** — (1) Read → tap Ask → disclosure → type → send → answer with sources. (2) Read →
  explain in own words → submit → feedback → optionally revise. (3) Read → open thread → reply →
  upvote → report if needed.
- **States** — *Empty*: prompt inviting the first question/explanation/post. *In flight*: cancellable
  spinner with the question echoed. *Error*: classified message + Retry, transcript preserved.
  *Consent required*: blocked composer + explanation + link to consent. *Offline*: draft kept,
  send disabled with reason. *Read-only*: transcript only. *Moderated post*: tombstone.
- **Accessibility annotations** — transcript as a semantic list; each message's accessible name
  includes author and role; polite announcements for arrivals; composer keeps focus; destructive
  actions (clear, delete) confirm in a focus-trapped dialog.
- **Copy & i18n** — per-tool namespaces plus shared `mobile.contentTools.ai.*` disclosure strings.

## 11. AI / ML Considerations

- **Models & prompts** — server-side only, via `aigateway` feature ids `content_tool_ask`,
  `content_tool_explain_back`. Mobile never sees a prompt or a model id.
- **Disclosure** — rendered natively by the frame per CT.M9 and the course's `aiDisclosureMode`.
- **Consent** — the composer is gated on the server's consent state
  (`GET …/content-tools/ai-consent`); a client default is never assumed.
- **PII redaction & filtering** — server-side; the client renders the filtered result and, when
  `freeTextFilterAction` blocks a submission, explains it without echoing the blocked content back as
  an error string.
- **Fallback** — provider failure degrades to a retry affordance with the transcript preserved; there
  is no on-device model and no silent substitution.
- **Cost** — per-manifest action rate limits plus the course's `monthlyAiTokenBudget` and
  `dailyAiCallsPerUser`; the client surfaces exhaustion honestly rather than retrying.

## 12. Integration Points

- **Internal** — CT.M3 host/state/actions/live region; CT.M1 renderer; CT.M9 disclosure, consent and
  report chrome; existing tutor chat patterns; keyboard and draft utilities.
- **Server (unchanged)** — `server/internal/service/contenttools/{ask_questions,explain_it_back,
  inline_discussion}_actions.go`, the grounded context service (CT.6), `aigateway`, and the governance
  handlers in `content_tools_governance.go`.
- **Events** — server-side CT.7 analytics and `analytics.ai_usage_log`; no client emission.

## 13. Dependencies & Sequencing

- Must ship after: **CT.M3**. CT.M9 should land alongside or before the AI tools so disclosure and
  consent are not stubbed; if CT.M9 has not landed, the AI composers stay disabled behind their
  allowlist entries.
- Independent of CT.M4, CT.M5, CT.M7, CT.M8.
- Recommended order: `inline_discussion` (no AI dependency) → `explain_it_back` → `ask_questions`.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A student's long typed answer is lost to a background kill or failed send | H | H | FR-2 draft persistence with an explicit restart AC; drafts stored separately from state |
| AI disclosure/consent shipped as a stub to unblock the renderer | M | H | Sequencing rule above: composer stays disabled until CT.M9 lands |
| Peer-visible content reaches a student before moderation | M | H | Server-side filtering and moderation states are authoritative; client renders tombstones and never caches removed bodies |
| Report is buried, so abuse goes unreported on mobile | M | H | Two-tap requirement (FR-18) with an explicit AC |
| Streaming responses fight the enclosing scroll view | M | M | Transcript renders in its own bounded, independently scrolling container with a fixed max height |
| Duplicate AI spend from retries on flaky mobile networks | M | M | Idempotency key persisted with the pending action (CT.M3 FR-9) |
| Anonymous discussions leak identity through a payload field | L | H | AC-11 asserts on the payload, not just the UI |

## 15. Rollout Plan

- **Feature flag** — `mobileContentToolsEnabled` plus per-tool allowlist entries; the two AI tools have
  their own entries so they can be held back independently of the discussion tool.
- **Sequencing** — discussion → explain-it-back → ask-questions, each behind its entry, each gated on
  CT.M9 for the AI pair.
- **Dogfood** — an internal course with a discussion anchored to a paragraph and both AI tools placed;
  deliberately exercise budget exhaustion and provider failure in staging.
- **GA criteria** — all ACs green; zero draft-loss findings; disclosure and consent verified with
  privacy review; report flow verified end to end with a moderator.
- **Rollback** — per-tool allowlist removal; existing transcripts and threads remain server-side and
  reappear when re-enabled.

## 16. Test Plan

- **Unit** — draft lifecycle (save, restore, clear on successful send, retain on failure); error
  classification from server responses; control visibility from viewer permissions; pagination
  cursoring; consent gating logic.
- **Integration** — each action round-trip including 403 on foreign-post edit, rate-limit and budget
  errors, filtered content, moderated post rendering, consent-required blocking.
- **End-to-end (device)** — ask a question and read the answer; write an explanation, background the
  app, force-quit, relaunch, confirm the draft, submit; post a reply, upvote, report; staff endorses
  and moderates.
- **Security** — verify no prompt/model/key on device; verify removed post bodies are absent from
  payloads; verify anonymous attribution; verify authorization is server-enforced by calling the action
  directly.
- **Accessibility** — screen-reader pass on transcript arrival announcements and focus retention;
  labelled report/moderate controls; 200% font scale in chat; RTL threads.
- **Performance / load** — typing latency with a 50-message transcript; thread pagination; AI in-flight
  scroll behaviour.
- **Manual exploratory** — poor network mid-stream, token refresh mid-action, emoji and RTL text in
  posts, extremely long single post, rapid double-send, keyboard with a hardware keyboard attached.

## 17. Documentation & Training

- End-user: how inline AI help works, what is sent to a model, and how to report a post.
- Instructor/TA: moderating and endorsing from a phone; what students see when the AI budget runs out.
- Privacy: update the AI disclosure copy inventory to include the mobile surfaces.
- Internal runbook: reading AI error-class counters to distinguish budget from provider incidents.

## 18. Open Questions

1. Does mobile support **streaming** AI responses, or render only the final answer? (Recommendation:
   final answer in v1 with a cancellable in-flight state; streaming is a fast-follow — it multiplies
   the flaky-network failure modes.)
2. Should inline discussion get realtime updates via the existing mobile realtime layer, or stay
   pull-to-refresh? (Recommendation: pull-to-refresh in v1, matching web.)
3. Where does the AI consent prompt live — inside the tool frame, or as a course-level gate the student
   hits once? (Decide with CT.M9; recommendation: course-level with an in-frame link.)
4. Do we let a student attach an image to a discussion post on mobile (the phone-camera argument from
   VC.M2)? (Recommendation: out of scope for v1; it widens the moderation surface.)

## 19. References

- Web plans: [CT.10](CT.10-tool-ask-questions.md),
  [CT.20](CT.20-tool-explain-it-back.md),
  [CT.22](CT.22-tool-inline-discussion.md),
  [CT.6](CT.6-grounded-context-and-link-ingestion.md),
  [CT.8](CT.8-governance-safety-privacy-accessibility.md).
- Web renderers: `clients/web/src/components/content-tools/tools/{ask_questions,explain_it_back,
  inline_discussion}/renderer.tsx`.
- Server: `server/internal/service/contenttools/{ask_questions,explain_it_back,inline_discussion}_actions.go`,
  `server/internal/httpserver/content_tools_governance.go`.
- Existing mobile patterns: `clients/ios/Lextures/Features/Tutor/TutorChatView.swift`,
  `clients/android/.../features/tutor/TutorChatScreen.kt`.
- Related plans: [CT.M3](CT.M3-mobile-content-tool-host-and-state.md),
  [CT.M9](../../plan/content_tools/CT.M9-mobile-tools-governance-a11y-telemetry.md).
- Standards: WCAG 2.1 AA §4.1.3 (status messages), §3.3.1 (error identification); EU AI Act
  transparency (S13); S01/S02 (DSAR, retention); S08 (children's privacy).
