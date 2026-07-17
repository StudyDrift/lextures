# MOB.2 — Canvas Course Import (mobile)

> Implementation plan. Source: Mobile ↔ web parity gap analysis (2026-07-17).
> Web reference: `fetchCanvasCourses` / `postCourseImportCanvas` in
> [`clients/web/src/lib/courses-api.ts`](../../../clients/web/src/lib/courses-api.ts)
> (≈L5719–5830) and the Canvas import panel used from `/courses/create`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MOB.2 |
| **Section** | Mobile parity |
| **Severity** | MAJOR |
| **Markets** | HE / K12 |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile team |
| **Depends on** | MOB.1 (create entry point) |
| **Unblocks** | — |

## 1. Problem Statement

Instructors migrating from Canvas can, on web, connect with a Canvas base URL +
access token, list their Canvas courses, choose what to bring over (modules,
assignments, quizzes, enrollments, grades, settings, files, announcements), and
watch a live import job. None of this exists on mobile — there is no Canvas
import code path in either client. This blocks the single most common onboarding
motion for higher-ed and K-12 instructors evaluating the app on a tablet, and
forces every migration to happen on a laptop.

## 2. Goals

- Let an instructor import a Canvas course into a new (or existing) Lextures
  course entirely on iOS/Android.
- Match web's selectable import scope and live progress.
- Handle the Canvas access token securely: sent once, never persisted on device.
- Integrate as a first-class branch of the MOB.1 create flow.

## 3. Non-Goals

- Building or changing the server import pipeline (RabbitMQ worker exists).
- Ongoing Canvas LTI/sync beyond the optional grade-sync toggle (that is a
  course-settings concern).
- Bulk/admin multi-course import (web `admin-console/imports`) — future plan.
- Editing imported content (handled by existing course screens post-import).

## 4. Personas & User Stories

- **As an HE instructor**, I want to pull my Canvas course shell into Lextures
  from my iPad so I can evaluate without a laptop.
- **As a K-12 teacher**, I want to import only modules + assignments (not
  grades) so I start clean.
- **As a privacy-conscious instructor**, I want assurance my Canvas token is not
  stored on my phone.

## 5. Functional Requirements

- **FR-1.** The app MUST provide a "credentials" step collecting Canvas base URL
  + access token, and call `POST /api/v1/integrations/canvas/courses` to list
  the instructor's Canvas courses.
- **FR-2.** The app MUST show a "select" step: pick one Canvas course and toggle
  the eight include categories (`CanvasImportInclude`; all default on).
- **FR-3.** The app MUST support import mode (new course vs. into an existing
  course — `CourseBundleImportMode`) consistent with web.
- **FR-4.** On confirm, the app MUST `POST /api/v1/courses/{courseCode}/import/canvas`
  and receive a `jobId`.
- **FR-5.** The app MUST stream progress from
  `wss://…/api/v1/ws/canvas-import/{jobId}` and render an "importing" step with
  live status messages.
- **FR-6.** The app MUST allow cancelling an in-flight import (abort) and show
  the cancelled state.
- **FR-7.** The access token MUST be held only in memory for the request
  lifetime and MUST NOT be written to Keychain/DataStore, logs, or crash
  reports.
- **FR-8.** An optional "push grades back to Canvas" toggle
  (`canvasGradeSyncEnabled`) MAY be offered, defaulting off.
- **FR-9.** On success, the app MUST route to the imported course workspace.
- **FR-10.** The entry point MUST be gated behind the Canvas integration being
  enabled for the org and the `course:create` permission.

## 6. Non-Functional Requirements

- **Performance** — course listing p95 < 3 s (Canvas-bound); WS progress renders
  within 500 ms of each server event.
- **Security** — token in memory only; TLS/WSS only; redact token from all
  telemetry; server enforces auth + org scope. Threat model: token leakage,
  MITM, replay — mitigated by no-store + WSS + short-lived job.
- **Privacy & Compliance** — imported enrollments may carry PII; respect
  existing consent/FERPA handling on the server; surface a notice that
  enrollments/grades will be copied.
- **Accessibility** — WCAG 2.1 AA; progress announced via live region;
  token field marked secure/no-autocorrect.
- **Scalability** — one job per import; server queue handles load.
- **Reliability** — WS reconnect with resume by `jobId`; idempotent — a
  reconnect does not double-import.
- **Observability** — `canvas_import_{listed,started,progress,succeeded,failed,cancelled}`
  with category counts (never the token).
- **Maintainability** — new `LMSAPICanvasImport` wrapper mirroring the web fns.
- **Internationalization** — `mobile.canvasImport.*` keys.
- **Backward compatibility** — no API change.

## 7. Acceptance Criteria

- **AC-1.** *Given* valid Canvas URL + token, *when* the user submits
  credentials, *then* their Canvas courses list appears.
- **AC-2.** *Given* a selected course with grades unchecked, *when* the import
  runs, *then* the resulting course has modules/assignments but no grades.
- **AC-3.** *Given* a running import, *when* the server emits progress, *then*
  the UI shows live status and a completion state, and lands on the course.
- **AC-4.** *Given* a running import, *when* the user cancels, *then* the job is
  aborted and a cancelled message is shown.
- **AC-5.** *Given* any import, *when* inspecting device storage/logs, *then* the
  Canvas token is absent (verified in a security test).
- **AC-6.** *Given* an org without Canvas enabled, *then* the import entry point
  is hidden.

## 8. Data Model

- **No new tables.** Server import worker writes into existing course/module/
  assignment/enrollment tables. Client holds transient credentials + job state
  only (no persistence).

## 9. API Surface

Existing endpoints (reused):

- `POST /api/v1/integrations/canvas/courses` → `{ courses: CanvasCourseListItem[] }`
  (body `{canvasBaseUrl, accessToken}`; token not stored).
- `POST /api/v1/courses/{courseCode}/import/canvas` → `{ jobId, message }`
  (body: `mode`, `canvasBaseUrl`, `canvasCourseId`, `accessToken`, `include`,
  `canvasGradeSyncEnabled?`).
- `WS /api/v1/ws/canvas-import/{jobId}` → progress messages.
- To create the target shell first: `POST /api/v1/courses` (via MOB.1) or the
  new-course mode of the import.

No new/changed server routes. OpenAPI already documents these.

## 10. UI / UX

- **New screens (both platforms):** Canvas Import — 3 steps mirroring web:
  Credentials → Select course + scope → Importing (live).
- **Flows:** (1) Create → "Import from Canvas"; (2) enter URL+token → list;
  (3) pick course, toggle categories, choose mode; (4) confirm → live progress →
  open course.
- **States:** loading (list), empty (no Canvas courses), error (bad
  token/URL with actionable message), importing (streamed), cancelled, success,
  offline (block start).
- **Mobile/responsive:** secure text entry for token; category toggles as a
  checklist; progress as a step log + spinner.
- **Accessibility:** token field `textContentType`/no autofill; progress in an
  ARIA-live-equivalent announcer; cancel reachable.
- **Copy & i18n:** `mobile.canvasImport.*`; include a "your token isn't stored"
  reassurance line.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- New iOS `Core/LMS/LMSAPICanvasImport.swift` + `CanvasImportLogic.swift`;
  `Features/Courses/Create/CanvasImportView.swift`.
- New Android `core/lms/CanvasImportApi.kt` + `CanvasImportLogic.kt`;
  `features/courses/create/CanvasImportScreen.kt`.
- Reuse `Core/Realtime/WebSocketClient.swift` (iOS) / `core/realtime` (Android)
  for the job WS.
- Entry from MOB.1 create screen.

## 13. Dependencies & Sequencing

- Must ship after: MOB.1 (entry point) and org Canvas integration enablement.
- Must ship before: —.
- Shared infra: existing import worker/queue, WS gateway.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Token accidentally persisted/logged | M | H | No-store policy; security test (AC-5); scrub telemetry |
| Long imports vs. app backgrounding kills WS | H | M | Reconnect by jobId; allow leaving and re-opening progress |
| Canvas API rate limits / large courses | M | M | Server-side queueing; user messaging; retry |
| Users confused by mode (new vs existing) | M | M | Clear labels; default to "new course" |

## 15. Rollout Plan

- Flag: `ff_mobile_canvas_import` (default off).
- Sequence: ship behind flag → pilot with a migrating HE org → GA.
- GA criteria: AC-1..6 pass; token-absence security test green; <2% import
  failure rate in pilot.
- Rollback: flag off hides the entry point.

## 16. Test Plan

- **Unit** — credential validation, include-map serialization, WS message
  parsing, cancel/abort.
- **Integration** — list → import → progress against a Canvas sandbox/mock.
- **End-to-end** — full import happy path + grades-excluded path on device.
- **Security** — token-not-persisted audit (Keychain/DataStore/logs/crash);
  WSS enforcement; authz.
- **Accessibility** — screen-reader run of all three steps + progress.
- **Performance** — list latency; progress render latency.
- **Manual** — background/foreground during import; poor-network reconnect.

## 17. Documentation & Training

- "Import from Canvas on mobile" help article with token-generation steps.
- Note on what each include category copies.
- Security FAQ: token handling.

## 18. Open Questions

1. Do we allow import into an existing course on mobile, or restrict to
   new-course to reduce error surface in v1?
2. Should the token entry offer OAuth (if the org has a Canvas developer key)
   instead of a manual token?
3. Surface `canvasGradeSyncEnabled` at import, or only later in course settings?

## 19. References

- Web: `clients/web/src/lib/courses-api.ts` (`fetchCanvasCourses`,
  `postCourseImportCanvas`, `CanvasImportInclude`), `pages/admin/AdminImport.tsx`.
- iOS realtime: `clients/ios/Lextures/Core/Realtime/WebSocketClient.swift`.
- Android realtime: `clients/android/app/src/main/kotlin/com/lextures/android/core/realtime/`.
- Related: [MOB.1](MOB.1-course-creation-wizard.md).
