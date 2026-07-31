# CT.M8 — Mobile Tool Pack 4: Media & Procedural (Media Checkpoints, Worked Example, Parameter Explorer, Code Sandbox)

> Implementation plan. Source: mobile renderers for [CT.19](CT.19-tool-media-checkpoints.md), [CT.18](CT.18-tool-step-through-worked-example.md), [CT.16](CT.16-tool-parameter-explorer.md), [CT.17](CT.17-tool-code-sandbox.md). Folder overview: [README](../../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.M8 |
| **Section** | Content Tools (CT) — Mobile |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M |
| **Owner (proposed)** | Mobile squad |
| **Depends on** | CT.M3; CT.M4 for `code_sandbox` |
| **Unblocks** | — (completes native coverage of the shipped shelf) |

---

## 1. Problem Statement

This pack finishes the shelf. `media_checkpoints` injects questions at timestamps in a video — the tool
most likely to be *used* on a phone and the one that most needs native player integration, because a
web player in a WebView on mobile means no picture-in-picture, no background audio, no AirPlay/Cast and
poor seek behaviour. `worked_example` walks a student through a procedure one checked step at a time.
`parameter_explorer` drives a live model from sliders. `code_sandbox` runs code — the one tool where a
native renderer is the wrong answer, and CT.M4's sandbox is the right one. Together they turn "open in
browser" into a rarity rather than the default.

## 2. Goals

- Ship native renderers for `media_checkpoints`, `worked_example` and `parameter_explorer` against the
  CT.M3 contract.
- Deliver `code_sandbox` on mobile via the CT.M4 WebView sandbox, with a mobile-appropriate editing
  experience and honest limits.
- Integrate media checkpoints with the platform's native player, captions and progress semantics rather
  than reimplementing playback.
- Keep every step check, parameter checkpoint and code run server-side and idempotent.
- Make each tool usable in one phone column: sliders, step cards and code panes that do not require
  horizontal page scrolling.

## 3. Non-Goals

- Authoring media checkpoints, worked-example steps, parameter models or starter code (web-only).
- Building a mobile code editor from scratch — `code_sandbox` uses the sandboxed web bundle (CT.M4).
- Offline video download or offline code execution.
- Instructor analytics for any of these tools (CT.7, web).
- Changing checkpoint scheduling, step verification, model evaluation or code execution semantics.

## 4. Personas & User Stories

- **As a student**, I want to watch the lecture video on my phone and answer the checkpoint questions
  as they come up, with my progress saved if I stop halfway.
- **As a student**, I want to keep listening to the video with the screen off and pick up at the next
  checkpoint.
- **As a student**, I want to work through the derivation one step at a time on my phone, with a hint
  when I am stuck.
- **As a student**, I want to move the slider and watch the graph change, then answer the noticing
  prompt.
- **As a student**, I want to at least *read and run* the code exercise on my phone even if I will
  write the real answer on a laptop.
- **As a deaf student**, I want captions on the checkpoint video.
- **As an instructor**, I want checkpoint completion from phone viewers to count exactly like desktop.

## 5. Functional Requirements

**Shared**

- **FR-1.** Each native renderer MUST register in CT.M3's registry and use only the host's state and
  action APIs; conflict policy is `server_wins` for all four.
- **FR-2.** No renderer may grade locally: `answer_checkpoint`, `check_step`, `verify`,
  `submit_answer` and `check` all resolve server-side (`scoring.mode: auto` where scored).
- **FR-3.** All prompt, step, hint and explanation text MUST render through CT.M1 (math and code are
  the norm in this pack).

**`media_checkpoints`** (caps: `state`, `scoring`, `media`; scoring `auto`)

- **FR-4.** MUST play the configured media with the **native** player (iOS `AVPlayer`, Android
  ExoPlayer/Media3) for direct media, and MUST use the provider's embed only where a provider (YouTube/
  Vimeo) requires it — reusing the app's existing `ContentVideoPlayer` / `CaptionedPlayerView`
  patterns.
- **FR-5.** MUST pause at each configured timestamp and present the checkpoint question, submitting via
  `answer_checkpoint`; where the config requires a correct answer to continue, playback MUST stay
  blocked until the server says otherwise.
- **FR-6.** MUST report playback progress via `record_progress` on a throttle (not per frame or per
  second), and MUST resume from the stored position when the student returns.
- **FR-7.** MUST honour the config's seek policy: where seeking past unanswered checkpoints is
  disallowed, the scrubber MUST enforce it, including after backgrounding.
- **FR-8.** MUST support captions where available and MUST respect the app's existing captions
  preference (`platformFeatures.immersiveReader.captionsEnabled`).
- **FR-9.** MUST support background audio and picture-in-picture where the course permits it, and MUST
  correctly re-present a checkpoint reached while in PiP or on the lock screen (pause and require
  return to the app rather than silently skipping).
- **FR-10.** MUST handle interruptions (call, route change, headphone unplug) without losing progress
  or double-submitting a checkpoint answer.

**`worked_example`** (caps: `state`, `scoring`; scoring `auto`)

- **FR-11.** MUST present one step at a time with the student's entry field, calling `check_step` to
  validate and advancing only on the server's verdict.
- **FR-12.** MUST support `hint` (progressive hints), `reveal_step` and `reveal_all` with the config's
  gating, and `prepare`/`verify` where the tool declares them.
- **FR-13.** MUST show step history above the current step so the student can see their own reasoning
  so far, and MUST preserve partially typed step input across backgrounding.
- **FR-14.** MUST make faded-scaffolding state visible: which steps were revealed vs solved, conveyed
  by icon and text.

**`parameter_explorer`** (caps: `state`, `aggregate`)

- **FR-15.** MUST render the configured parameters as accessible sliders with numeric readouts and a
  direct-entry alternative (a slider alone is not sufficient for precision or for assistive tech).
- **FR-16.** MUST render the model's output visualisation and any data table (the table via CT.M1's
  table component, horizontally scrollable inside its own container).
- **FR-17.** MUST submit guided-noticing answers via `submit_answer`, record `checkpoint` states, and
  support `reset_defaults`.
- **FR-18.** MUST throttle recomputation while a slider is dragged and MUST autosave only on settle.

**`code_sandbox`** (caps: `state`, `scoring`, `code_execution`; scoring `auto`)

- **FR-19.** MUST be delivered through the CT.M4 sandboxed WebView; there is no native code editor.
- **FR-20.** MUST route `run`, `check`, `reset_code` and `try_reference` through CT.M3's action
  dispatch (the WebView never calls the API), and MUST show execution results including stderr and
  timeouts.
- **FR-21.** MUST provide a mobile-appropriate editor experience: an accessory bar for characters the
  phone keyboard buries, horizontal code scrolling, and a run control that stays reachable above the
  keyboard.
- **FR-22.** Where CT.M4 is unavailable (flag off, unsupported WebView), MUST fall back to CT.M3's
  placeholder with "Open in browser" — never a broken editor.

## 6. Non-Functional Requirements

- **Performance** — Video starts in ≤ 2 s on Wi-Fi and streams (never fully buffers first); slider
  recomputation is throttled to ≤ 30 Hz with autosave on settle; a step check round-trip feels
  immediate (optimistic UI for the entry, authoritative verdict from the server).
- **Security** — Code execution is entirely server-side and sandboxed (CT.17); the device never
  executes student code. Media is fetched via authorized URLs; no answer key or reference solution
  reaches the client before the server releases it.
- **Privacy & Compliance** — Playback progress and code submissions are education records (S01/S02);
  progress reporting is throttled and coarse, not a keystroke-level behavioural log.
- **Accessibility** — WCAG 2.1 AA: native player controls with captions (§1.2.2) and audio description
  where authored; checkpoint questions announced when playback pauses; sliders expose value, min, max
  and step with a text-entry alternative (§1.4.13, §4.1.2); step progress announced; code results
  exposed as text. Reduced-motion honoured in visualisations.
- **Scalability** — `record_progress` throttling is the load-sensitive path; the interval must be
  agreed with backend before launch.
- **Reliability** — Checkpoint answers are idempotent (a retry after a dropped connection must not
  double-submit); playback position survives interruption; code runs are idempotent by key.
- **Observability** — Counters for checkpoint presented/answered/blocked, media start failures, PiP
  checkpoint encounters, step hint usage, slider interactions, and code run outcomes — labelled
  `tool_id`.
- **Internationalization** — `mobile.contentTools.tools.{media_checkpoints,worked_example,
  parameter_explorer,code_sandbox}.*`; RTL slider direction and step layout; localized number
  formatting in readouts and tables.
- **Backward compatibility** — State schemas identical to web; a session started on web resumes on
  mobile at the same position and step, and vice versa (explicit AC).

## 7. Acceptance Criteria

- **AC-1.** *Given* a `media_checkpoints` instance, *When* playback reaches a checkpoint, *Then* it
  pauses, the question is presented and announced, and the answer is graded server-side.
- **AC-2.** *Given* a checkpoint requiring a correct answer, *Then* playback cannot resume until the
  server allows it, including after backgrounding and relaunch.
- **AC-3.** *Given* a seek-restricted config, *When* the student drags the scrubber past an unanswered
  checkpoint, *Then* the seek is clamped.
- **AC-4.** *Given* a student stops halfway and returns the next day, *Then* playback resumes at the
  stored position and answered checkpoints are not re-asked.
- **AC-5.** *Given* PiP or lock-screen playback, *When* a checkpoint is reached, *Then* playback pauses
  and the student is prompted to return to the app; the checkpoint is not skipped.
- **AC-6.** *Given* a dropped connection during a checkpoint submit, *When* it retries, *Then* the
  idempotency key prevents a double answer.
- **AC-7.** *Given* captions are available and the preference is on, *Then* captions display.
- **AC-8.** *Given* `worked_example`, *When* a student enters a step and it is wrong, *Then* the server
  verdict renders, a hint is available per config, and the step does not advance.
- **AC-9.** *Given* a partially typed step, *When* the app is backgrounded and killed, *Then* the input
  is restored on relaunch.
- **AC-10.** *Given* `reveal_step`, *Then* the revealed state is visibly distinct from a solved step in
  icon and text, and the server records it.
- **AC-11.** *Given* `parameter_explorer`, *When* a slider is dragged, *Then* recomputation is throttled
  and exactly one state save occurs on settle.
- **AC-12.** *Given* a screen-reader user, *Then* each slider announces value/min/max/step and can be
  set precisely via the text-entry alternative.
- **AC-13.** *Given* a parameter model with a data table, *Then* the table renders via CT.M1 and scrolls
  inside its own container.
- **AC-14.** *Given* `code_sandbox` with CT.M4 enabled, *Then* the student can edit, run and check code,
  and results including stderr and timeouts render.
- **AC-15.** *Given* CT.M4 disabled or unsupported, *Then* `code_sandbox` shows the placeholder with
  "Open in browser".
- **AC-16.** *Given* a session started on web, *Then* mobile resumes at the same position/step, and vice
  versa.
- **AC-17.** *Given* CI, *Then* iOS build, Android compile and the renderer logic suites pass.

## 8. Data Model

**No server schema change, no migration.** State follows each manifest under
`server/internal/service/contenttools/tools/{media_checkpoints,worked_example,parameter_explorer,
code_sandbox}/manifest.json`. Cross-platform resume (AC-16) depends on mobile writing the same
position/step/parameter fields web writes — mobile MUST NOT introduce client-specific keys, and MUST
preserve unknown fields on write.

## 9. API Surface

**No new endpoints.** CT.M3's state routes plus:

| Tool | Actions |
|---|---|
| `media_checkpoints` | `answer_checkpoint`, `record_progress` |
| `worked_example` | `check_step`, `hint`, `prepare`, `reveal_step`, `reveal_all`, `verify` |
| `parameter_explorer` | `checkpoint`, `reset_defaults`, `submit_answer` |
| `code_sandbox` | `run`, `check`, `reset_code`, `try_reference` (dispatched natively on behalf of the CT.M4 WebView) |

Media assets are fetched via the existing authorized media URLs.

## 10. UI / UX

- **New (iOS)** — `Features/ContentTools/Tools/{MediaCheckpointsToolView,WorkedExampleToolView,
  ParameterExplorerToolView}.swift`, `Core/LMS/ContentToolPack4Logic.swift` (pure: checkpoint
  scheduling and seek clamping, resume position resolution, step state machine, slider throttle/settle
  logic — unit-tested).
- **New (Android)** — `features/contenttools/tools/{MediaCheckpointsTool,WorkedExampleTool,
  ParameterExplorerTool}.kt`, `core/lms/ContentToolPack4Logic.kt`.
- **Reused** — `ContentVideoPlayer` / `CaptionedPlayerView` (iOS `CourseMarkdownContentView.swift`) and
  the Android player equivalents; CT.M4's sandbox host for `code_sandbox`; CT.M1's table component for
  parameter data tables.
- **Key flows** — (1) Play → pause at checkpoint → answer → resume → finish → completion recorded.
  (2) Read step 1 → enter → check → advance → hint on step 3 → finish. (3) Move sliders → observe →
  answer noticing prompt. (4) Read starter code → edit → run → check.
- **States** — *Media*: loading, playing, paused-for-checkpoint, blocked (must answer), completed,
  failed-to-load (with retry and a link-out). *Steps*: current, solved, revealed, all-complete,
  attempts exhausted. *Explorer*: computing (throttled), settled, checkpoint answered, defaults reset.
  *Code*: sandbox loading, running, passed, failed, timed out, unavailable (placeholder).
- **Accessibility annotations** — player uses native accessible controls; checkpoint presentation moves
  focus to the question and announces it; sliders expose full value semantics with a text field
  alternative; step transitions announced; code output is a labelled readable region.
- **Copy & i18n** — per-tool namespaces; player and checkpoint strings reuse existing media keys where
  they already exist.

## 11. AI / ML Considerations

None of these four declare the `ai` capability, and `code_sandbox` executes code in the server's own
sandbox rather than through a model. Where `worked_example`'s `hint` action is AI-backed in a given
configuration, the AI disclosure and consent obligations from CT.M9 apply exactly as they do in CT.M6 —
verify the tool's declared capabilities at mount rather than assuming.

## 12. Integration Points

- **Internal (iOS)** — `AVKit`/`AVFoundation`, the existing `ContentVideoPlayer` and
  `CaptionedPlayerView`, `Core/Accessibility` (captions preference), CT.M3 host, CT.M4 sandbox,
  CT.M1 renderer.
- **Internal (Android)** — Media3/ExoPlayer, the existing player wrapper, `core/accessibility`,
  CT.M3 host, CT.M4 sandbox, CT.M1 renderer.
- **Server (unchanged)** — `server/internal/service/contenttools/{media_checkpoints,worked_example,
  parameter_explorer,code_sandbox}_actions.go`.
- **Events** — server-side CT.7 analytics; media progress is a tool action, not a separate telemetry
  channel.

## 13. Dependencies & Sequencing

- Must ship after: **CT.M3**; `code_sandbox` additionally after **CT.M4**.
- Independent of CT.M5–CT.M7.
- Recommended order: `worked_example` (pure form/state, no media) → `parameter_explorer` →
  `media_checkpoints` (largest platform surface) → `code_sandbox` (gated on CT.M4).
- Shared infra: native media playback and the CT.M4 sandbox.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Checkpoints can be skipped via PiP, lock screen, or backgrounding | H | H | FR-9 pauses and requires return to the app; explicit AC-5 covering PiP and lock screen; seek clamping re-applied after foreground |
| `record_progress` chatter creates avoidable server load | M | M | Throttled, coarse reporting with the interval agreed with backend before launch |
| Double-submitted checkpoint answers on flaky networks | M | M | Idempotency keys (CT.M3 FR-9); AC-6 |
| Provider embeds (YouTube/Vimeo) do not expose reliable timing for checkpoints | M | H | Use the provider player's API where it exposes time; where it does not, degrade to the placeholder rather than shipping unreliable checkpoints — decide per provider, see Open Question 1 |
| Code editing on a phone keyboard is genuinely bad | H | M | Set expectations honestly: accessory bar and run/check work; the plan does not claim phone-first authoring, and "open in browser" stays one tap away |
| Slider-driven recomputation drains battery or thrashes saves | M | M | Throttle + settle-only autosave (FR-18, AC-11) |
| Cross-platform resume mismatches | M | M | Same state keys, unknown-field preservation, explicit AC-16 |

## 15. Rollout Plan

- **Feature flag** — `mobileContentToolsEnabled` plus per-tool allowlist entries; `code_sandbox` also
  requires `mobileContentToolsSandboxEnabled`.
- **Sequencing** — worked example → parameter explorer → media checkpoints → code sandbox, each behind
  its entry.
- **Dogfood** — an internal course with a 10-minute video carrying three checkpoints, a multi-step
  derivation, a slider model with a data table, and one code exercise.
- **GA criteria** — all ACs green; no checkpoint-skip path found in adversarial testing; media start
  and progress metrics within target; a11y sign-off including captions and slider semantics.
- **Rollback** — per-tool allowlist removal; learner progress is server-side and unaffected.

## 16. Test Plan

- **Unit** — checkpoint scheduling and seek clamping across seeks, rate changes and interruptions;
  resume-position resolution; step state machine including reveal vs solve; slider throttle/settle;
  unknown-field preservation.
- **Integration** — each action round-trip including blocked-resume, attempt limits, idempotent
  retries; cross-platform resume fixtures (web-written state resumed on mobile and vice versa).
- **End-to-end (device)** — watch a video with checkpoints including a call interruption, a background,
  a PiP transition and a relaunch; complete a worked example with hints and a reveal; drive the
  explorer and answer its prompt; run and check code in the sandbox.
- **Security** — no reference solution or answer key in client payloads before release; code never
  executes on device; media URLs require auth.
- **Accessibility** — captions on; screen-reader pass through a checkpoint pause and question; slider
  value announcement and text-entry alternative; step-advance announcements; code output readability.
- **Performance / load** — media start time and rebuffering on cellular; `record_progress` request
  rate measured against the agreed interval; slider recomputation frame rate; battery over a
  20-minute session.
- **Manual exploratory** — headphone unplug mid-checkpoint, airplane mode mid-video, seek spamming,
  rotation during a checkpoint, very long code output, extremely small slider ranges, tablet layout.

## 17. Documentation & Training

- End-user: watching checkpoint videos on a phone (background audio, PiP behaviour, resume), and the
  honest note that code exercises are best finished on a larger screen.
- Instructor: authoring guidance — checkpoint spacing, seek policy implications for mobile, and which
  video providers support reliable checkpoints.
- Internal runbook: interpreting checkpoint-blocked and media-failure counters.

## 18. Open Questions

1. Which video providers expose timing reliable enough for checkpoints on mobile, and what do we do for
   those that do not? (Owner: mobile squad + content; decide before the media-checkpoints PR.
   Recommendation: support direct media and any provider whose player API exposes time; placeholder
   otherwise.)
2. What `record_progress` interval does backend want, given class-scale simultaneous viewing?
3. Does the course-level PiP/background-audio permission already exist, or does CT.M8 need to define
   one? (Verify against the existing media settings before assuming.)
4. Is a phone-sized `code_sandbox` worth shipping at all, or is "open in browser" the honest answer
   until tablets? (Recommendation: ship read-and-run on phones; it has real value for reading and
   verifying, and the sandbox path costs little once CT.M4 exists.)
5. Should `parameter_explorer` visualisations follow the platform charting approach used elsewhere in
   the apps, or render the model's own output? (Prefer the existing app charting for consistency and
   accessibility.)

## 19. References

- Web plans: [CT.19](CT.19-tool-media-checkpoints.md),
  [CT.18](CT.18-tool-step-through-worked-example.md),
  [CT.16](CT.16-tool-parameter-explorer.md),
  [CT.17](CT.17-tool-code-sandbox.md).
- Web renderers: `clients/web/src/components/content-tools/tools/{media_checkpoints,worked_example,
  parameter_explorer,code_sandbox}/`.
- Server: `server/internal/service/contenttools/{media_checkpoints,worked_example,parameter_explorer,
  code_sandbox}_actions.go`.
- Existing mobile media: `clients/ios/Lextures/Features/Courses/CourseMarkdownContentView.swift`
  (`ContentVideoPlayer`, `CaptionedPlayerView`), the Android player equivalents.
- Related plans: [CT.M3](CT.M3-mobile-content-tool-host-and-state.md),
  [CT.M4](CT.M4-mobile-sandboxed-webview-tool-host.md),
  [CT.M1](CT.M1-mobile-markdown-engine-tables-code-math.md).
- Standards: WCAG 2.1 AA §1.2.2 (captions), §1.4.13, §4.1.2 (slider semantics), §2.2.2 (pause, stop,
  hide for auto-updating visualisations).
