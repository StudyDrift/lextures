# CT.M4 — Mobile Sandboxed WebView Tool Host (long-tail & third-party tools)

> Implementation plan. Source: mobile port of the [CT.5](../../completed/content_tools/CT.5-tool-sdk-sandboxing-and-versioning.md) sandbox bridge. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.M4 |
| **Section** | Content Tools (CT) — Mobile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad + web platform (consult) |
| **Depends on** | CT.M3 (web: CT.5, CT.9) |
| **Unblocks** | CT.M8 (code sandbox), marketplace tools on mobile |

---

## 1. Problem Statement

Content Tools is designed so that "adding a tool = a manifest + a renderer bundle" — hundreds of tools,
plus a third-party marketplace (CT.9). Native mobile renderers do not scale to that: CT.M5–CT.M8
deliver fourteen, and every marketplace tool after that would ship a placeholder forever. CT.5 already
solved this on web with a sandboxed iframe and a versioned `postMessage` bridge; the same tool bundle
can run inside a WebView on iOS and Android against a natively-implemented bridge. CT.M4 makes the
long tail work on phones without a mobile release per tool, and gives CT.M3's "Open in browser"
placeholder a real fallback: run it here, in place, with the app owning persistence and permissions.

## 2. Goals

- Run any CT.5-conformant tool bundle inside an embedded, origin-isolated WebView on both platforms.
- Implement the CT.5 bridge protocol (`init` / `stateAccepted` / `actionResult` / `error` host→tool;
  `ready` / `save` / `runAction` / `resize` / `announce` tool→host) natively, with the same size, rate
  and schema guards the web host enforces.
- Keep persistence, auth, networking and governance **native** — the WebView never sees the session
  token, never calls the API, and never reaches the device.
- Make the sandbox the automatic fallback in CT.M3's renderer resolution: native renderer → sandbox →
  placeholder.
- Preserve the CT.M3 frame contract: same status chips, same save/offline semantics, same live-region
  announcements, same read-only reasons.

## 3. Non-Goals

- Replacing native renderers. Where a native renderer exists it wins — it is faster, works offline, and
  is properly accessible. The sandbox is for the tail.
- Offline execution of sandboxed tools in v1 (bundle caching is Open Question 2).
- Mobile tool *authoring* or a mobile developer experience for tool authors.
- Changing the CT.5 protocol. If mobile needs a protocol change, that is a CT.5 revision with a version
  bump, not a mobile fork.
- Arbitrary third-party network access from inside the sandbox.

## 4. Personas & User Stories

- **As a student**, I want the tool my instructor placed to work on my phone even if the app has no
  native version of it yet.
- **As an instructor**, I want to place any tool from the catalog without checking a mobile support
  matrix first.
- **As a tool developer**, I want to write one bundle to the published SDK contract and have it run on
  web and mobile.
- **As a security reviewer**, I want assurance that third-party tool code cannot read the session
  token, the device, or another course's data.
- **As a screen-reader user**, I want a sandboxed tool to announce itself and its saves like every
  other tool.

## 5. Functional Requirements

- **FR-1.** CT.M3's renderer resolution MUST become: native renderer if registered and contract-compatible
  → sandbox host if the instance is sandboxable → placeholder. The decision MUST be a pure, tested
  function.
- **FR-2.** The sandbox host MUST load the tool document served at `/tool-sandbox/{toolId}.html` and
  mount it in a WebView with an **opaque origin** and no access to app cookies or storage: iOS
  `WKWebView` with a non-persistent `WKWebsiteDataStore` loading the fetched document via
  `loadHTMLString(_:baseURL: nil)`; Android `WebView` with `loadDataWithBaseURL(null, …)`.
- **FR-3.** The document bytes MUST be fetched **natively** (authenticated, pinned, integrity-checked
  per CT.5 versioning) and handed to the WebView; the WebView MUST NOT perform the fetch itself.
- **FR-4.** The bridge MUST implement protocol version 1 exactly as `clients/web/src/lib/tool-sdk/
  bridge/protocol.ts` defines it, including the `v: 1` discriminator and message-type allowlists.
- **FR-5.** Host→tool transport MUST be `postMessage` semantics: Android via `WebMessagePort`
  (`WebViewCompat`), iOS via an injected `window.postMessage` shim delivered through
  `evaluateJavaScript`. Tool→host MUST be `WKScriptMessageHandler` (iOS) and a `WebMessagePort`
  listener (Android). `addJavascriptInterface` MUST NOT be used.
- **FR-6.** Ingress MUST enforce the CT.5 guards: reject messages over `BRIDGE_MAX_MESSAGE_BYTES`
  (64 KB), rate-limit to `BRIDGE_MAX_MESSAGES_PER_SEC` (20) with a sliding window, and drop any
  message failing `isBridgeFromTool` validation. Each rejection MUST be counted by reason
  (`oversized` / `rate_limited` / `malformed` / `unknown_type`).
- **FR-7.** On mount the host MUST send `init` with `instanceId`, redacted `config`, current `state`,
  `revision`, `locale`, `dir`, `readOnly` and an **opaque** `participantId` — never a user id, email or
  enrollment id.
- **FR-8.** A `save` message MUST route through CT.M3's state store (debounce, revision, conflict,
  offline queue) and MUST be answered with `stateAccepted` carrying the new revision. Saves MUST be
  ignored when the frame is read-only.
- **FR-9.** A `runAction` message MUST route through CT.M3's action dispatch (idempotency key, rate
  limits) and MUST be answered with `actionResult` or a typed `error`.
- **FR-10.** A `resize` message MUST set the WebView height, clamped to 80–2000 pt/dp, without causing
  layout thrash in the enclosing scroll view.
- **FR-11.** An `announce` message MUST route to CT.M3's shared live region, subject to a rate cap so a
  misbehaving tool cannot flood assistive technology.
- **FR-12.** If `ready` does not arrive within 10 s the host MUST fall back to the CT.M3 placeholder
  with a "sandbox timeout" reason (parity with the web host's `READY_TIMEOUT_MS`).
- **FR-13.** The WebView MUST block **all** navigation: no top-level loads, no new windows, no
  `target=_blank`; a link tap inside a tool MUST be intercepted and routed through the app's link
  handler with its normal scheme allowlist and confirmation.
- **FR-14.** The WebView MUST have no access to camera, microphone, geolocation, clipboard-read,
  file pickers, notifications or Bluetooth unless the instance's manifest `capabilities` declare it and
  the app has an OS permission; declared capabilities MUST be surfaced to the user before first use.
- **FR-15.** The WebView MUST NOT receive the auth token, cookies, or any header the app uses for API
  calls, and MUST NOT be able to reach the API origin directly (enforced by opaque origin + navigation
  block + CSP in the served document).
- **FR-16.** `breakerOpen`, `tombstone`, `deprecated` and CT.M9's kill-switch MUST prevent the sandbox
  from mounting at all, falling back to the read-only placeholder with the appropriate reason.
- **FR-17.** The host MUST enforce the CT.5 pinned tool **version** in the document URL and reject a
  document whose reported `contract` on `ready` is outside the supported range.
- **FR-18.** Each mounted sandbox MUST be torn down deterministically when scrolled far out of view or
  when the screen is dismissed: bridge disposed, WebView released, no retained JS timers.
- **FR-19.** The frame MUST show a visible "runs in a sandbox" badge, matching the web
  `contentTools.sdk.sandboxBadge` treatment, so learners and instructors can tell third-party content
  apart.

## 6. Non-Functional Requirements

- **Performance** — At most **3** live WebViews per screen, recycled from a pool; mount-to-`ready`
  p95 ≤ 1.5 s on a mid-range device over Wi-Fi; a sandboxed tool must not drop the enclosing scroll
  below 50 fps; memory ceiling enforced by the pool, with off-screen tools torn down (FR-18).
- **Security** — Threat model is *hostile tool code*. Defences: opaque origin, no same-origin access,
  no session material in the WebView, navigation blocked, no JS interface object, schema/size/rate
  guards on ingress, capability-gated OS permissions, CSP on the served document, and native-only
  networking. A compromised tool can at worst write malformed state for its own instance — which the
  server rejects with 422.
- **Privacy & Compliance** — `participantId` is opaque and per-instance (mirroring
  `opaqueParticipantId`); no PII crosses the bridge. Third-party tools carry CT.9 consent and CT.M9
  disclosure. Non-persistent data store means nothing the tool writes survives the mount.
- **Accessibility** — WCAG 2.1 AA is the tool's obligation, but the host MUST: name the WebView region,
  route `announce` to the shared live region, keep the sandbox reachable and escapable by
  VoiceOver/TalkBack and switch control, honour font scale by forwarding it in `init`, and never trap
  focus. The conformance signal from `GET /api/v1/content-tools/conformance` MUST be surfaced where a
  tool is known non-conformant.
- **Scalability** — Client-only; no new server load beyond serving static tool documents.
- **Reliability** — Timeout fallback (FR-12); a crashed WebView content process MUST be detected and
  replaced with the placeholder + Retry, never a blank frame; the enclosing screen never crashes with
  the tool.
- **Observability** — Counters for sandbox mounts, ready latency, timeouts, bridge rejections by
  reason, content-process crashes, and teardowns — all labelled `tool_id`.
- **Maintainability** — The protocol lives in **one** place conceptually; mobile ports it verbatim and
  a shared fixture suite asserts message-level equivalence with the web host.
- **Internationalization** — `locale` and `dir` forwarded on `init`; sandbox chrome strings under
  `mobile.contentTools.sandbox.*`; RTL frames verified.
- **Backward compatibility** — Protocol version negotiated on `ready`; unsupported versions degrade to
  the placeholder. Adding message types is additive; removing one requires a CT.5 major bump.

## 7. Acceptance Criteria

- **AC-1.** *Given* an instance whose `toolId` has no native renderer but is sandboxable, *When* it
  mounts, *Then* the tool runs in a WebView inside the normal CT.M3 frame with the sandbox badge shown.
- **AC-2.** *Given* a native renderer exists for the same `toolId`, *Then* the native renderer is used
  and no WebView is created.
- **AC-3.** *Given* a tool that never sends `ready`, *When* 10 s pass, *Then* the placeholder with a
  timeout reason replaces it and no WebView remains alive.
- **AC-4.** *Given* a tool sends `save`, *Then* the state goes through the CT.M3 store (debounced,
  revisioned, queued offline) and the tool receives `stateAccepted` with the new revision.
- **AC-5.** *Given* the frame is read-only, *When* the tool sends `save`, *Then* nothing is written and
  the tool receives no `stateAccepted`.
- **AC-6.** *Given* a tool sends 100 messages in one second, *Then* messages beyond the limit are
  dropped, counted as `rate_limited`, and the app stays responsive.
- **AC-7.** *Given* a tool sends a 1 MB message, *Then* it is rejected as `oversized` and nothing is
  parsed into app memory beyond the size check.
- **AC-8.** *Given* a tool attempts `window.location = 'https://evil.example'` or opens a new window,
  *Then* the navigation is blocked and the tool remains mounted.
- **AC-9.** *Given* a tool attempts to read cookies, `localStorage`, or fetch the API origin, *Then*
  all fail (opaque origin, non-persistent store) and no session token is present in the WebView.
- **AC-10.** *Given* a tool requests the camera without a declared capability, *Then* the request is
  denied without an OS prompt.
- **AC-11.** *Given* a tool sends `resize` with 100000, *Then* the height is clamped to 2000 and layout
  stays stable.
- **AC-12.** *Given* a tool sends `announce`, *Then* the message is spoken through the shared live
  region, rate-capped.
- **AC-13.** *Given* a page with 6 sandboxed tools, *When* the student scrolls, *Then* at most 3
  WebViews are alive at once and memory stays within budget.
- **AC-14.** *Given* the WebView content process crashes, *Then* the placeholder with Retry appears and
  the screen survives.
- **AC-15.** *Given* a tombstoned, breaker-open or killed tool, *Then* no WebView is created.
- **AC-16.** *Given* the shared bridge fixture suite, *When* it runs against the iOS, Android and web
  hosts, *Then* all three produce identical accept/reject decisions per message.

## 8. Data Model

**No server schema change, no migration, no new persisted client model.** CT.M4 reuses CT.M3's
`ToolInstance` (notably `sandboxMode`, `contract`, `capabilities`, `breakerOpen`, `tombstone`) and
`ToolStateEnvelope`. Bridge messages are transient, mirroring `BridgeToTool` / `BridgeFromTool`:

```swift
enum BridgeToTool {                      // host → tool
  case initTool(instanceId: String, config: JSONValue, state: JSONValue, revision: Int64,
                locale: String, dir: String, readOnly: Bool, participantId: String)
  case stateAccepted(revision: Int64)
  case actionResult(id: String, result: JSONValue)
  case error(id: String?, code: String, message: String)
}

enum BridgeFromTool {                    // tool → host
  case ready(contract: String)
  case save(state: JSONValue, revision: Int64)
  case runAction(id: String, action: String, input: JSONValue)
  case resize(height: Double)
  case announce(message: String, assertive: Bool)
}
```

## 9. API Surface

**No new endpoints.** CT.M4 uses:

| Verb | Path | Purpose |
|---|---|---|
| GET | `/tool-sandbox/{toolId}.html` (+ pinned version) | The sandboxed tool document, fetched natively |
| GET | `/api/v1/courses/{course_code}/content-tools/manifests/{tool_id}` | Capabilities, contract range, sandbox mode |
| GET | `/api/v1/content-tools/conformance` | Accessibility conformance signal for the badge |
| — | CT.M3's state/submit/action routes | Reached only through the native host, never by the WebView |

## 10. UI / UX

- **New (iOS)** — `Features/ContentTools/Sandbox/{SandboxWebViewHost,SandboxBridge,SandboxWebViewPool}.swift`,
  `Core/LMS/ContentToolSandboxLogic.swift` (pure: message validation, rate limiter, size guard,
  resolution order — unit-tested).
- **New (Android)** — `features/contenttools/sandbox/{SandboxWebViewHost,SandboxBridge,
  SandboxWebViewPool}.kt`, `core/lms/ContentToolSandboxLogic.kt`.
- **Modified** — CT.M3's `ToolRendererRegistry` gains the sandbox branch; `ToolFrame` gains the sandbox
  badge and the conformance note.
- **Key flows** — (1) Tool with no native renderer → frame → skeleton → document fetched natively →
  WebView mounts → `ready` → tool visible with sandbox badge. (2) Student interacts → tool `save` →
  native store → `stateAccepted` → "Saved ✓". (3) Tool needs the camera → capability check → OS
  permission explainer → grant or deny. (4) Student scrolls away → WebView torn down → scrolls back →
  remounted with current state.
- **States** — *Loading*: fixed-height skeleton until `ready` (never a flash of empty WebView).
  *Timeout*: placeholder + Retry. *Crashed*: placeholder + Retry. *Read-only*: overlay that blocks
  input and a reason string. *Blocked*: killed/tombstoned placeholder. *Offline*: placeholder stating
  the tool needs a connection (until Open Question 2 is resolved).
- **Accessibility annotations** — WebView container named by tool title; sandbox badge included in the
  container label; `announce` routed to the shared region; font scale forwarded on `init`; focus can
  always leave the WebView.
- **Copy & i18n** — `mobile.contentTools.sandbox.badge`, `.timeout`, `.crashed`, `.needsConnection`,
  `.capabilityPrompt`, `.nonConformant` across all locale files.

## 11. AI / ML Considerations

The sandbox itself runs no model. AI-backed tools call `runAction`, which the **native** host forwards
to the server's `aigateway` — so budgets, disclosure, logging and PII redaction are unchanged and the
untrusted bundle never holds a provider key or sees a prompt template. The AI disclosure required by
CT.M9 is rendered by the native frame, above the WebView, where a hostile tool cannot cover it.

## 12. Integration Points

- **Internal (iOS)** — `WKWebView`/`WebKit`, `Features/Courses/WebItemView.swift` (existing
  `AuthenticatedWebView` is the reference for the *authenticated* case and deliberately **not** reused
  here — the sandbox must be unauthenticated), CT.M3 host and state store, `Core/Networking`,
  `Core/Accessibility`.
- **Internal (Android)** — `androidx.webkit` (`WebViewCompat`, `WebMessagePort`, `WebViewFeature`
  checks with a graceful placeholder when unsupported), `features/courses/WebItemScreen.kt` (reference
  only), CT.M3 host, `core/network`, `core/accessibility`.
- **Web (normative)** — `clients/web/src/lib/tool-sdk/bridge/protocol.ts`,
  `clients/web/src/components/content-tools/host/sandbox/{bridge.ts,sandbox-iframe-host.tsx}`.
- **Server (unchanged)** — the `/tool-sandbox` document route and
  `server/internal/service/contenttools/{breaker,policy,conformance}.go`.
- **Events** — none client-side.

## 13. Dependencies & Sequencing

- Must ship after: **CT.M3** (frame, state store, action dispatch, placeholder).
- Must ship before: **CT.M8** (Code Sandbox is delivered via this path) and any marketplace tool on
  mobile.
- Can ship in parallel with: CT.M5–CT.M7 native packs.
- Shared infra: the `/tool-sandbox` document route (already serving web).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Hostile third-party tool code escapes the sandbox | L | H | Opaque origin, no session material, no JS interface, navigation blocked, capability gating, native-only networking; external security review before enabling third-party (non-first-party) tools |
| WebView memory/perf makes tool-heavy pages unusable | H | M | Pool of ≤ 3, teardown off-screen, fixed-height skeletons, explicit perf ACs |
| Android `WebMessagePort` unavailable on old WebView versions | M | M | `WebViewFeature.isFeatureSupported` check; unsupported devices fall back to the CT.M3 placeholder rather than a weaker bridge |
| Sandbox becomes the default and native renderers never get built | M | M | Resolution order is native-first by construction; track sandbox-vs-native mount ratio as a roadmap signal |
| Protocol drift between web and mobile hosts | M | H | Shared fixture suite asserting identical accept/reject decisions (AC-16) |
| Accessibility inside third-party tools is poor | H | M | Conformance signal surfaced in the frame; instructors can disable tools per course via the allowlist; the native frame owns naming and announcements |
| Sandboxed tools do not work offline, confusing students who could work offline yesterday | M | M | Explicit offline state copy; bundle caching evaluated in Open Question 2 |

## 15. Rollout Plan

- **Feature flag** — client capability `mobileContentToolsSandboxEnabled`, default off, enabled per
  environment. Independent of `mobileContentToolsEnabled` so the native path can ship without the
  sandbox.
- **Sequencing** — bridge + fixtures (no UI) → WebView mount with `noop_probe` → save/action wiring →
  pooling and teardown → capability gating → security review → enable for **first-party** tools only →
  after review, enable for marketplace tools.
- **Dogfood** — an internal course placing one first-party tool that has no native renderer yet.
- **GA criteria** — all ACs green; security review signed off; ready-latency p95 within target; no
  content-process crash regression over a week of dogfood.
- **Rollback** — flip the capability off; tools revert to the CT.M3 placeholder. No data effect.

## 16. Test Plan

- **Unit** — message validation (type allowlist, `v` check, unknown types), size guard, sliding-window
  rate limiter, resolution order (native → sandbox → placeholder), height clamping, teardown
  bookkeeping, contract-range gating.
- **Integration** — bridge round-trip against a fixture tool document: `init` → `ready` → `save` →
  `stateAccepted` → `runAction` → `actionResult`; error paths for each guard; shared fixture suite run
  against iOS, Android and web hosts (AC-16).
- **End-to-end (device)** — a seeded page with a sandboxed tool: interact, save, background/foreground,
  rotate, scroll away and back, go offline.
- **Security** — dedicated hostile-tool fixture attempting: navigation, new windows, cookie/storage
  access, API fetch, oversized/flooded messages, camera without capability, focus trapping, covering
  the AI disclosure. Each MUST fail closed. Plus an external review before third-party enablement.
- **Accessibility** — screen-reader traversal into and out of a sandboxed tool; `announce` routing;
  font-scale forwarding; switch-control reachability; RTL.
- **Performance / load** — 6-sandbox page: pool behaviour, memory ceiling, scroll FPS, ready latency
  distribution.
- **Manual exploratory** — old Android WebView versions, low-memory devices, WebView updating
  mid-session, slow network during document fetch, tool that never resizes, tool that resizes every
  frame.

## 17. Documentation & Training

- Tool-developer docs: "your bundle runs on mobile too" — the mobile bridge parity guarantees, the
  capability model, and the performance envelope to target.
- Instructor docs: what the sandbox badge means; that third-party tools are isolated.
- Security runbook: the mobile threat model, what the kill switch stops, and how to revoke a tool.
- Update `docs/MOBILE_PLAN.md` and both client READMEs.

## 18. Open Questions

1. Do we enable the sandbox for **third-party/marketplace** tools at launch, or first-party only until
   an external security review lands? (Recommendation: first-party only at launch.)
2. Should tool bundles be cached on-device for offline use? It is a meaningful integrity and staleness
   surface (pinned version + hash). (Recommendation: defer to a fast-follow; ship online-only.)
3. Is a WebView pool of 3 right, or should it be device-tier dependent? (Measure on a low-end Android
   device before fixing the number.)
4. Where does the shared bridge fixture suite live so all three hosts run it? (Recommendation:
   `clients/mobile/fixtures/content-tools/bridge/`, consumed by web tests as well — same location
   proposed in CT.M3 Open Question 4.)
5. Does the iOS `postMessage` shim need to be signed/verified to prevent a tool from spoofing host
   messages to itself? (Recommendation: yes — nonce per mount, checked on ingress.)

## 19. References

- Web plans: [CT.5](../../completed/content_tools/CT.5-tool-sdk-sandboxing-and-versioning.md),
  [CT.9](../../completed/content_tools/CT.9-tool-marketplace-and-third-party-tools.md),
  [CT.3](../../completed/content_tools/CT.3-student-runtime-and-state-persistence.md).
- Web implementation: `clients/web/src/lib/tool-sdk/bridge/protocol.ts`,
  `clients/web/src/components/content-tools/host/sandbox/bridge.ts`,
  `clients/web/src/components/content-tools/host/sandbox/sandbox-iframe-host.tsx`.
- Existing mobile WebView precedent: `clients/ios/Lextures/Features/Courses/WebItemView.swift`,
  `clients/android/.../features/courses/WebItemScreen.kt`,
  `clients/ios/Lextures/Features/Courses/LaunchContainerView.swift`.
- Related plans: [CT.M3](CT.M3-mobile-content-tool-host-and-state.md),
  [CT.M8](CT.M8-mobile-tools-media-and-procedural.md),
  [CT.M9](CT.M9-mobile-tools-governance-a11y-telemetry.md).
- Standards: WCAG 2.1 AA §2.1.2 (no keyboard trap), §4.1.3 (status messages); OWASP MASVS-PLATFORM
  (WebView hardening); S13 (EU AI Act disclosure) via CT.M9.
