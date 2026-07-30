# CT.M3 — Mobile Content Tool Host & State Persistence

> Implementation plan. Source: mobile half of [CT.3 FR-17 / AC-12](../../completed/content_tools/CT.3-student-runtime-and-state-persistence.md). Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.M3 |
| **Section** | Content Tools (CT) — Mobile |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad + backend platform (consult) |
| **Depends on** | CT.M1, CT.M2 (web: CT.1, CT.2, CT.3) |
| **Unblocks** | CT.M4, CT.M5, CT.M6, CT.M7, CT.M8, CT.M9 |

---

## 1. Problem Statement

Fifteen Content Tools shipped to the web (CT.10–CT.23), and CT.3 §FR-17 promised mobile would "render
a first-class placeholder for renderers they do not implement, and render supported tools natively."
Neither half exists: a repo-wide search for `lex-tool`, `contentTool` or `content_tool` in
`clients/ios` and `clients/android` returns nothing, and mobile does not even read the
`contentToolsEnabled` course flag. Today a student who opens a tool-bearing page on a phone sees a
code block containing `{"instanceId":"9f0c…","toolId":"inline_questions","v":1}` — leaked internal
JSON where an activity should be. CT.M3 ships the mobile runtime host: fence → instance → renderer,
with the state persistence, conflict, offline and accessibility contract shared by every tool that
will ever exist.

## 2. Goals

- Resolve a `lex-tool` fence to a live tool instance and mount a renderer inside a consistent frame,
  on every surface CT.M2 migrated.
- Implement the CT.3 **state contract** natively: load, debounced autosave with optimistic
  concurrency, explicit submit, 409/413/422 handling, and self-reset.
- Implement the CT.3 **action contract** with idempotency keys, so AI-backed and auto-graded tools
  need no mobile-specific endpoint.
- Make offline first-class: cached config and state render offline, writes queue in the existing
  outbox and replay in order.
- Guarantee graceful degradation — unimplemented tool, disabled flag, breaker open, tombstoned,
  archived, past due, or renderer crash each produce a clear, labelled, non-destructive state.
- Ship the accessibility baseline (labelled group per tool, live-region announcements, focus order)
  once, in the host, not fifteen times.

## 3. Non-Goals

- Any individual tool renderer — CT.M5–CT.M8 deliver those. CT.M3 ships the host plus one trivial
  renderer (`noop_probe`) to prove the contract end to end.
- The WebView sandbox for long-tail and third-party tools — that is CT.M4.
- Authoring, configuring or placing tools on mobile (CT.2 stays web-only; mobile is read/interact).
- The instructor state console and roster reset (CT.4 stays web-only for now).
- Instructor analytics and insights on mobile (CT.7 surfaces stay web-only).
- Any server change. CT.M3 consumes the shipped API exactly as the web host does.

## 4. Personas & User Stories

- **As a student**, I want the inline activity my instructor placed in the reading to actually work on
  my phone, because my phone is where I do my reading.
- **As a student**, I want my answers saved as I type, and to see clearly when they are saved.
- **As a student on the bus with no signal**, I want to keep working and have it sync when I reconnect.
- **As a student**, I want a tool the app cannot render yet to say so plainly and offer to open it in a
  browser — not to show me raw JSON or a blank space.
- **As an instructor previewing on my phone**, I want to interact with a tool as myself without
  polluting class aggregates.
- **As a screen-reader user**, I want each activity announced as a named region with its status.
- **As a parent/observer**, I want to see my student's tool work read-only where I already have that
  permission.

## 5. Functional Requirements

- **FR-1.** The apps MUST read `contentToolsEnabled` from the course features payload and MUST NOT
  fetch instances or render tool chrome for a course with the flag off; a `lex-tool` fence in a
  flag-off course renders nothing at all (parity with web).
- **FR-2.** On opening a surface, the host MUST issue **exactly one** batched request per item:
  `GET …/content-tools/instances?itemId={id}&withState=1`, and map fences to instances by `instanceId`.
- **FR-3.** A fence whose `instanceId` is absent from the response (deleted, or not visible to the
  viewer) MUST render nothing — never an error, never raw JSON.
- **FR-4.** Each mounted tool MUST be wrapped in a `ToolFrame` showing title, status chip
  (`not_started` / `in_progress` / `submitted` / `completed`), save indicator, score when scored, and
  an overflow with Help, Reset (when `studentResetAllowed`) and Report (CT.M9).
- **FR-5.** State writes MUST use `PUT …/instances/{instanceId}/state` with the last-known `revision`,
  debounced at 1.5 s and coalesced so rapid input produces one write.
- **FR-6.** A `409 revision_conflict` MUST be resolved using the tool's declared conflict policy
  (mirroring `clients/web/src/components/content-tools/host/conflict-policy.ts`) and MUST NOT silently
  discard learner input; where the policy is "server wins", the local document is preserved and offered.
- **FR-7.** `422 schema_invalid` MUST surface a non-destructive inline error and MUST NOT clear local
  state; `413 state_too_large` MUST block further growth with a clear message.
- **FR-8.** Explicit submit MUST call `POST …/instances/{instanceId}/submit`; the client MUST NOT set
  scores — `score.raw` / `score.max` come from the server only.
- **FR-9.** Actions MUST call `POST …/instances/{instanceId}/actions/{action}` with an
  `idempotencyKey`; a retry of the same logical action MUST reuse the key so no duplicate AI call or
  side effect occurs.
- **FR-10.** Offline: instance config and state MUST be cached via the existing `OfflineService`
  (`cachedFetch` + a new `OfflineCacheKey.contentToolInstances(courseCode:itemId:)`), state writes MUST
  queue through `enqueueMutation` and replay **in order per instance**, and the frame MUST show an
  "unsynced" chip while queued.
- **FR-11.** Actions MUST NOT be queued offline (they can have server side effects and AI cost); an
  action attempted offline MUST show a "needs connection" state and offer retry.
- **FR-12.** A tool with no native renderer MUST render a first-class placeholder naming the tool, with
  an "Open in browser" action that deep-links to the web activity (CT.3 FR-17 / AC-12). CT.M4 replaces
  this path for sandboxable tools.
- **FR-13.** Read-only rendering MUST be enforced for: archived instances, closed activities, past-due
  when the tool declares `respectsDueDate`, `tombstone: true`, `breakerOpen: true`, and
  observer/parent viewers — each with a distinct reason string.
- **FR-14.** A renderer that throws MUST be contained: a labelled error card with Retry replaces that
  tool only; every other tool and the surrounding page keep working.
- **FR-15.** The host MUST expose an `announce(message, assertive:)` API backed by one shared live
  region per screen (iOS `UIAccessibility.post(.announcement:)` / Android `LiveRegion` semantics), used
  for saves, scores and errors.
- **FR-16.** The host MUST honour `contract` (runtime contract version) from the instance payload and
  refuse to mount a renderer outside its supported range, falling back to FR-12.
- **FR-17.** The host MUST NOT trust or send client timestamps; interaction telemetry is recorded
  server-side on write.
- **FR-18.** Self-reset MUST call `POST …/instances/{instanceId}/self-reset` when the course allows it,
  with a confirm dialog naming what will be cleared.
- **FR-19.** Instructors and TAs interacting on mobile MUST write against their own enrollment exactly
  as the web host does; no role-specific client branching.

## 6. Non-Functional Requirements

- **Performance** — Host mount ≤ 30 ms per instance on a mid-range device; the batched instances
  request p95 ≤ 120 ms; a page with 20 tools must not stall scrolling — renderers hydrate on approach
  to viewport, not all at mount.
- **Security** — `instanceId` is the only client-supplied key and is validated server-side against the
  course; students read and write only their own state (enforced in SQL, not by a body field); the app
  MUST NOT send `enrollmentId` for its own writes. TLS pinning and auth-token handling follow existing
  app conventions.
- **Privacy & Compliance** — `state_json` is student work: it appears in DSAR exports (S01) and follows
  retention policy (S02) server-side. The offline cache holds it on-device, so it MUST live in the
  existing encrypted cache store and MUST be purged on sign-out and on enrollment loss.
- **Accessibility** — WCAG 2.1 AA baseline in the host: `role=group` equivalent with an accessible name
  per tool, focus order matching visual order, one polite live region per screen (assertive reserved
  for errors), ≥ 44 pt / 48 dp targets, Dynamic Type / font-scale to 200%, reduced-motion honoured,
  colour never the sole carrier of correctness.
- **Scalability** — Single-row upserts; autosave debounce keeps write QPS ≈ page views. No new server
  load pattern beyond what the web host already produces.
- **Reliability** — Optimistic concurrency prevents lost updates; the outbox prevents lost work;
  actions are idempotent; an AI/provider outage degrades to "try again" without losing the transcript.
- **Observability** — Client counters mirroring the server metric names where the app has analytics:
  tool mounts by `tool_id`, render errors, save outcomes, conflicts, offline replays, unsupported-tool
  placeholders shown (the last is the roadmap signal for CT.M5–CT.M8 ordering).
- **Maintainability** — One host, one state store, one frame. A new renderer registers in a map and
  implements one interface; it never touches networking.
- **Internationalization** — `mobile.contentTools.runtime.*` keys (`saved`, `saving`, `unsynced`,
  `retry`, `unavailable`, `readOnlyArchived`, `readOnlyPastDue`, `openInBrowser`, `resetConfirm`) in
  `clients/mobile/locales/*.json`; RTL verified; numbers/dates through existing locale helpers.
- **Backward compatibility** — Additive: courses with the flag off are byte-identical to today.
  Unknown `toolId`, unknown `status`, and future contract versions all degrade to FR-12 rather than
  crashing, so the apps survive server-side tool launches without a release.

## 7. Acceptance Criteria

- **AC-1.** *Given* a page with two placed tools, *When* a student opens it, *Then* both mount with
  their config and prior state, and exactly one instances request is issued.
- **AC-2.** *Given* a course with `contentToolsEnabled: false`, *When* a page containing a fence
  renders, *Then* nothing tool-related appears and no content-tools request is made.
- **AC-3.** *Given* a student interacts with a tool, *When* 1.5 s elapse, *Then* one `PUT …/state` is
  sent and a "Saved" indicator appears and is announced politely.
- **AC-4.** *Given* the same instance edited on web and then saved from a stale mobile client, *Then*
  the client receives 409, resolves per the declared policy, and no learner input is silently lost.
- **AC-5.** *Given* a state document above `maxStateBytes`, *When* the client saves, *Then* it handles
  413 with a clear message and the stored document is unchanged.
- **AC-6.** *Given* airplane mode, *When* a student interacts and then reconnects, *Then* queued writes
  replay in order and the server state matches the last local state.
- **AC-7.** *Given* an action attempted offline, *Then* the tool shows "needs connection" and no write
  is queued.
- **AC-8.** *Given* the same action retried after a timeout, *When* the idempotency key is reused,
  *Then* the second call returns the first result with no additional side effect.
- **AC-9.** *Given* a `toolId` the app has no renderer for, *When* it mounts, *Then* the placeholder
  names the tool and "Open in browser" deep-links to the web activity — no crash, no blank space, no
  raw JSON.
- **AC-10.** *Given* a renderer that throws during mount, *Then* only that tool shows an error card
  with Retry; the other tools and the page still work.
- **AC-11.** *Given* an archived instance, a past-due `respectsDueDate` tool, a tombstoned tool and an
  open breaker, *Then* each renders read-only with its own reason string and no write is attempted.
- **AC-12.** *Given* VoiceOver/TalkBack, *When* the student reaches a tool, *Then* it is announced as a
  named region with its status, and save/score changes are announced without stealing focus.
- **AC-13.** *Given* the course flag is switched off while the page is open, *When* the student next
  interacts, *Then* the API returns 404 and the tool becomes read-only without discarding stored work.
- **AC-14.** *Given* sign-out, *Then* every cached instance config and state document is purged from
  the device.
- **AC-15.** *Given* CI, *Then* iOS build, Android compile, and the host logic unit suites pass.

## 8. Data Model

**No server schema change, no migration.** CT.M3 reads `course.content_tool_instances` and reads/writes
`course.content_tool_states` through the shipped API. New client models mirror
`server/internal/models/contenttools/types.go` exactly:

```kotlin
@Serializable data class ToolInstance(
  val id: String, val toolId: String, val toolVersion: String, val hostKind: String,
  val structureItemId: String? = null, val sectionKey: String? = null, val title: String? = null,
  val config: JsonElement, val status: String, val updatedAt: String,
  val state: ToolStateEnvelope? = null,
  val sandboxMode: String? = null, val contract: Int = 0, val breakerOpen: Boolean = false,
  val deprecated: Boolean = false, val sunsetAt: String? = null,
  val capabilities: List<String> = emptyList(), val tombstone: Boolean = false,
)

@Serializable data class ToolStateEnvelope(
  val instanceId: String, val revision: Long, val status: String, val state: JsonElement,
  val score: ToolScore? = null, val updatedAt: String? = null,
  val resetCount: Int = 0, val lastResetAt: String? = null,
  val stateSchemaVersion: Int = 0, val quarantined: Boolean = false,
)

@Serializable data class ToolScore(val raw: Double, val max: Double)
```

(iOS: equivalent `Codable` structs in `Core/LMS/LMSContentToolModels.swift`.)

**On-device storage** — instance config + state cached under
`OfflineCacheKey.contentToolInstances(courseCode:itemId:)` in the existing encrypted `CacheStore`;
queued writes ride the existing mutation outbox. Both are purged on sign-out (AC-14).

## 9. API Surface

**No new endpoints.** Mobile consumes the shipped routes:

| Verb | Path | Auth |
|---|---|---|
| GET | `/api/v1/courses/{course_code}/content-tools/instances?itemId=&withState=1` | course member |
| GET | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/state` | own state |
| PUT | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/state` | own state |
| POST | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/submit` | own state |
| POST | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/actions/{action}` | course member |
| POST | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/self-reset` | own state, when allowed |
| GET | `/api/v1/courses/{course_code}/content-tools/manifests/{tool_id}` | course member |
| GET | `/api/v1/courses/{course_code}/content-tools/settings` | course member (allowlist, reset policy) |
| GET | `/api/v1/courses/{course_code}/content-tools/my-progress` | own progress |

Error contract handled client-side: `409 revision_conflict` (with `current` envelope),
`422 schema_invalid` (`errors[{path,message}]`), `413 state_too_large` (`maxBytes`), `404` when the
flag is off or the instance is not visible. Rate limits (120 state writes/min/user/instance;
per-manifest action limits) are server-enforced; the client backs off on 429 rather than retrying hot.

## 10. UI / UX

- **New (iOS)** — `Core/LMS/LMSContentToolModels.swift`, `Core/LMS/LMSAPIContentTools.swift`,
  `Core/LMS/ContentToolHostLogic.swift` (pure: fence→instance mapping, debounce/coalesce decisions,
  conflict resolution, read-only reason derivation — unit-tested per repo convention),
  `Features/ContentTools/{ContentToolHostView,ToolFrameView,ToolPlaceholderView,ToolErrorCardView,
  ToolLiveRegion,ToolRendererRegistry}.swift`.
- **New (Android)** — `core/lms/ContentToolModels.kt`, `core/lms/ContentToolsApi.kt`,
  `core/lms/ContentToolHostLogic.kt`,
  `features/contenttools/{ContentToolHost,ToolFrame,ToolPlaceholder,ToolErrorCard,ToolRegistry}.kt`.
- **Modified** — the CT.M1 renderer's `toolFence` case now delegates to the host; the surfaces CT.M2
  migrated pass `courseCode` + `itemId` context down so the host can batch-fetch once per screen.
- **Key flows** — (1) Open page → one batched fetch → each fence mounts in a frame → interact →
  debounce → save → "Saved ✓" chip + polite announcement. (2) Explicit submit → status chip advances →
  server score appears. (3) Offline interact → amber "Saved on this device" chip → reconnect → replay →
  chip clears. (4) Unsupported tool → placeholder → "Open in browser" → in-app browser at the web
  activity.
- **States** — *Loading*: frame with a fixed-height skeleton, no layout shift. *Idle*: the tool's own
  empty UI. *Saving / Saved / Unsynced / Error*: chips in the frame header. *Read-only*: dimmed with a
  reason ("Archived by your instructor", "Past due", "Preview", "Temporarily unavailable"). *Unsupported*:
  named placeholder + open-in-browser. *Crashed*: error card + Retry.
- **Accessibility annotations** — one accessible container per tool named by title; header chips are
  part of the container label, not separate stops; one live region per screen; the reset confirm dialog
  is modal with focus trapped and returned.
- **Copy & i18n** — `mobile.contentTools.runtime.*` (per NFR) across all five locale files.

## 11. AI / ML Considerations

CT.M3 dispatches AI-backed actions but owns no prompt or model. Every AI call goes through the server's
`aigateway` under feature ids `content_tool*`, is budgeted and logged to `analytics.ai_usage_log`, and
is disclosed by the frame when the instance's `capabilities` include `ai` (disclosure wording and
consent gating land in CT.M9). Fallback on provider failure is the tool's own "try again" state with
the transcript preserved. No on-device inference; no PII leaves the device beyond the state document
the server already stores.

## 12. Integration Points

- **Internal (iOS)** — `Core/Offline/{OfflineService,CacheStore,OfflineModels}.swift` (new cache key +
  outbox use), `Core/Networking`, `Core/Auth/AuthSession.swift`, `Core/Routing/ContentLinkRouter.swift`
  (open-in-browser deep link), `Core/Accessibility`, `Core/I18n`, plus Xcode project regeneration.
- **Internal (Android)** — `core/offline`, `core/network`, `core/auth`, `core/routing`,
  `core/accessibility`, `core/i18n`.
- **Server (unchanged)** — `server/internal/httpserver/content_tools_{state,actions,instances,reset}.go`,
  `server/internal/service/contenttools/`.
- **Web (reference)** — `clients/web/src/components/content-tools/host/` (`content-tool-host.tsx`,
  `use-tool-state.ts`, `use-tool-action.ts`, `conflict-policy.ts`, `tool-frame.tsx`) is the normative
  behaviour spec; mobile mirrors it rather than inventing semantics.
- **Events** — none emitted client-side; the server emits xAPI/Caliper on write (CT.7).

## 13. Dependencies & Sequencing

- Must ship after: **CT.M1** (the `toolFence` block) and **CT.M2** (surfaces on one renderer).
- Must ship before: **CT.M4** (sandbox host plugs into this frame) and every tool pack
  **CT.M5–CT.M8**, and **CT.M9** (governance chrome hangs off the frame).
- Shared infra: the offline cache and mutation outbox; no new backend infrastructure.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| The mobile host drifts from web semantics (conflict, debounce, status) | H | H | Port `conflict-policy.ts` and the `use-tool-state` state machine as a **spec table** into shared fixtures both platforms test against |
| Student work lost on conflict or offline replay | M | H | Never overwrite local state on 409; preserve-and-offer; per-instance ordered replay; explicit AC-4/AC-6 tests |
| State documents in the on-device cache leak after sign-out | M | H | Encrypted cache store + purge on sign-out, verified by AC-14 |
| 20 tools on a page tank scroll performance | M | M | Viewport-approach hydration, fixed-height skeletons, per-tool mount budget in the perf test |
| Placeholder fatigue: most tools unimplemented at launch | H | M | Ship CT.M4 (sandbox) close behind, and use the "unsupported placeholder shown by tool_id" counter to order CT.M5–CT.M8 |
| Server launches a new tool the app has never heard of | H | L | Unknown `toolId`/`contract` always degrades to FR-12; no release needed |
| Duplicate AI spend from retries | M | M | Idempotency keys generated once per logical action and persisted with the pending action |

## 15. Rollout Plan

- **Feature flag** — server-side per-course `contentToolsEnabled` (already shipped) **plus** a client
  capability `mobileContentToolsEnabled` so mobile can be dark-launched independently of web. Default
  off; internal builds on.
- **Sequencing** — models + API client → fence→instance resolution → frame + placeholder (this alone
  fixes the raw-JSON leak and is worth shipping first) → state load/save/conflict → offline queue →
  actions → self-reset → `noop_probe` renderer proving the contract → flip on for dogfood.
- **Dogfood** — an internal course with tools placed on a content page, an assignment and a quiz;
  exercised on iOS and Android, online and offline.
- **GA criteria** — all ACs green; zero data-loss findings in the offline/conflict test matrix; render
  error rate < 1% per `tool_id` over a week of dogfood; a11y pass signed off.
- **Rollback** — flip `mobileContentToolsEnabled` off; the apps fall back to the CT.M1 placeholder.
  Nothing server-side changes, and no learner state is affected.

## 16. Test Plan

- **Unit** — fence parsing → instance mapping (including missing/duplicate/unknown ids); debounce and
  coalescing; conflict resolution per policy; read-only reason precedence (tombstone > breaker >
  archived > past due > observer); idempotency-key lifecycle; offline queue ordering per instance;
  contract-range gating.
- **Integration** — against a seeded server: load → save → conflict (409) → oversize (413) → invalid
  (422) → submit → action → self-reset; flag-off returns 404 and the tool goes read-only.
- **End-to-end (device)** — a page with three tools including one unsupported: interact, background the
  app, go offline, interact more, reconnect, verify final state; renderer-crash injection verifies
  isolation.
- **Security** — attempt a write against another enrollment (expect 403 and no write); verify no
  `enrollmentId` is sent on self-writes; verify cache purge on sign-out; verify no state in logs or
  crash reports.
- **Accessibility** — automated scan plus scripted VoiceOver and TalkBack passes over a three-tool
  page: region naming, status announcement, focus order, no trap, 200% font-scale, reduced motion.
- **Performance / load** — 20-tool page: mount time per tool, scroll FPS, memory; batched-request
  latency measured against the p95 target.
- **Manual exploratory** — flaky network, token refresh mid-save, force-quit with queued writes, device
  clock skew, low storage, tablet split view, RTL locale.

## 17. Documentation & Training

- End-user: "Activities inside your reading on mobile" — saving, offline behaviour, reset, and what
  "Open in browser" means.
- Instructor: which tools work natively on mobile today and what students see for the rest; a note that
  placing a tool does not break mobile.
- Internal runbook: how to read the unsupported-placeholder counter, how to interpret conflict and
  offline-replay metrics, how to kill mobile tools without a release.
- Update the CT folder README's mobile index and `clients/{ios,android}/README.md` feature lists.

## 18. Open Questions

1. Should the app cache instance **config** for offline first-render, or refuse to render tools offline
   until first fetched? (Recommendation: cache config + state, render read-write offline; it is the
   whole point of the outbox.)
2. Does "Open in browser" open the in-app `AuthenticatedWebView` at the activity anchor or an external
   browser? (Recommendation: in-app webview with the session, so the student is not asked to log in
   again — and it is the same surface CT.M4 will reuse.)
3. Do we need a mobile analogue of CT.4's instructor reset console, or is self-reset plus web enough
   for v1? (Recommendation: self-reset only in v1.)
4. Where does the shared web/mobile behaviour spec live so the three hosts cannot drift — a fixture
   file, or a prose contract in `docs/`? (Recommendation: fixtures under `clients/mobile/fixtures/
   content-tools/`, consumed by web tests too.)
5. Is a per-instance WebSocket ever needed on mobile for live tools (CT.21 Class Pulse), or does the
   cached-aggregate poll suffice? (Recommendation: poll, matching web; revisit in CT.M5.)

## 19. References

- Web plan and normative behaviour:
  [CT.3](../../completed/content_tools/CT.3-student-runtime-and-state-persistence.md) (§5 FR-17, §7
  AC-12 are the mobile obligations this story discharges),
  [CT.1](../../completed/content_tools/CT.1-foundations-registry-and-data-model.md),
  [CT.5](../../completed/content_tools/CT.5-tool-sdk-sandboxing-and-versioning.md).
- Web implementation: `clients/web/src/components/content-tools/host/*`.
- Server: `server/internal/httpserver/content_tools_state.go`, `…_actions.go`, `…_instances.go`,
  `…_reset.go`; models `server/internal/models/contenttools/types.go`.
- Existing mobile infra: `clients/ios/Lextures/Core/Offline/{OfflineService,CacheStore,OfflineModels}.swift`,
  `clients/ios/Lextures/Features/Courses/WebItemView.swift` (`AuthenticatedWebView`),
  `clients/android/.../core/offline/*`, `.../features/courses/WebItemScreen.kt`.
- Related plans: [CT.M1](../../completed/content_tools/CT.M1-mobile-markdown-engine-tables-code-math.md),
  [CT.M2](CT.M2-mobile-rich-content-parity-assignments-quizzes.md),
  [CT.M4](CT.M4-mobile-sandboxed-webview-tool-host.md), [CT.M9](CT.M9-mobile-tools-governance-a11y-telemetry.md).
- Standards: WCAG 2.1 AA §4.1.3 (status messages), §2.4.3 (focus order); S01/S02 (DSAR, retention).
