# MOB.6 — Whiteboards (mobile authoring)

> Implementation plan. Source: Mobile ↔ web parity gap analysis (2026-07-17).
> Web reference: [`clients/web/src/components/whiteboard/*`](../../../clients/web/src/components/whiteboard/)
> (`whiteboard-toolbar.tsx`, `use-whiteboard-canvas.ts`), whiteboard endpoints in
> `clients/web/src/lib/courses-api.ts` (≈L7604–7660).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MOB.6 |
| **Section** | Mobile parity |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | THIN (view-only) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile team |
| **Depends on** | — |
| **Unblocks** | — |

## 1. Problem Statement

Course/meeting **whiteboards** — a freeform drawing canvas (tools: select, pen,
line, rect, circle, triangle, eraser, text; colors; pan/zoom) — are fully
authorable on web. On mobile they are **read-only**: iOS `Live/WhiteboardView`
renders `CourseWhiteboard.canvasData` with pan/zoom but no editing, and Android
has a `WhiteboardRenderer` only. Instructors and students can look at a
whiteboard on their phone but cannot draw, annotate, or create one, which
undercuts live-meeting collaboration and tablet-based teaching (a natural
pen/stylus use case).

## 2. Goals

- Turn the mobile whiteboard from a viewer into a full authoring surface: draw,
  add shapes/text, erase, choose colors, pan/zoom, undo/redo.
- Support creating, saving, and deleting whiteboards on mobile.
- Persist edits via the existing whiteboard endpoints (`canvasData`).
- Provide a great stylus/touch experience (Apple Pencil / stylus, palm rejection
  where the OS allows).
- Keep the on-wire element schema identical to web so boards created on either
  platform render everywhere.

## 3. Non-Goals

- Changing the whiteboard element schema or server storage.
- Building a new realtime protocol if the web whiteboard is save-based (see
  Open Questions / §14 for the realtime decision).
- Visual **collaboration boards** (Padlet-style) — that is
  [MOB.8](MOB.8-collaboration-boards-completion.md), a different feature.
- Infinite-canvas advanced features beyond web parity (layers, etc.).

## 4. Personas & User Stories

- **As an instructor**, I want to sketch a diagram on my tablet during a live
  meeting so students can follow along.
- **As a student**, I want to annotate the shared whiteboard to answer a prompt.
- **As a tutor**, I want to create a quick whiteboard to work a problem with a
  learner.
- **As a presenter**, I want to pan/zoom and erase without fighting the touch UI.

## 5. Functional Requirements

- **FR-1.** The whiteboard MUST support the same tools as web:
  select, pen, line, rect, circle, triangle, eraser, text.
- **FR-2.** The user MUST be able to set stroke color/width (and text color)
  matching web's toolbar options.
- **FR-3.** The user MUST be able to pan and zoom the canvas (existing viewer
  gesture) while drawing.
- **FR-4.** The app MUST support undo/redo of edits.
- **FR-5.** The app MUST create a whiteboard
  (`POST /api/v1/courses/{code}/whiteboards`) and delete one
  (`DELETE …/whiteboards/{boardId}`) where permitted.
- **FR-6.** Edits MUST persist by writing `canvasData` back
  (`PUT …/whiteboards/{boardId}`), producing elements byte-compatible with web's
  schema.
- **FR-7.** The app MUST handle stylus input (Apple Pencil / Android stylus)
  distinctly from finger where the platform exposes it, with reasonable palm
  handling.
- **FR-8.** If the whiteboard is collaborative in a live meeting, remote edits
  MUST appear in near-real-time and local edits MUST propagate (see §14 realtime
  decision).
- **FR-9.** Access MUST respect the meeting/course permissions that govern who
  may edit vs. view.

## 6. Non-Functional Requirements

- **Performance** — 60 fps ink while drawing; input-to-ink latency < 50 ms with
  stylus; large boards (hundreds of elements) pan/zoom smoothly.
- **Security** — edit gated by permission; saves authenticated; no injection via
  text elements (sanitise on render).
- **Privacy & Compliance** — whiteboard content may include student work; treat
  as course data (FERPA); no third-party sync.
- **Accessibility** — WCAG 2.1 AA: tool buttons labelled and ≥44 pt; provide a
  non-drawing way to add text; respect reduced-motion for canvas animations;
  ensure color is not the only differentiator for created shapes.
- **Scalability** — element writes batched/debounced; delta save where possible.
- **Reliability** — autosave with retry; edits survive backgrounding; conflict
  strategy defined for concurrent edits.
- **Observability** — `whiteboard_{created,edited,saved,deleted,undo}` (counts,
  no content).
- **Maintainability** — new `WhiteboardLogic` + `LMSAPIWhiteboard` (iOS) /
  `WhiteboardApi.kt` (Android); share element (de)serialization with the shipped
  viewer.
- **Internationalization** — `mobile.whiteboard.*` keys.
- **Backward compatibility** — element schema unchanged; older clients still
  render new boards.

## 7. Acceptance Criteria

- **AC-1.** *Given* edit permission, *when* the user draws with the pen and
  adds a rectangle and text, *then* on save-and-reopen the elements persist and
  render identically on web.
- **AC-2.** *Given* a mistake, *when* the user taps undo, *then* the last edit is
  reverted; redo re-applies it.
- **AC-3.** *Given* a stylus, *when* the user draws, *then* ink follows the pen
  with < 50 ms perceived latency and palm touches don't create strokes.
- **AC-4.** *Given* view-only permission, *then* tools are hidden and the canvas
  is read-only (current behaviour preserved).
- **AC-5.** *Given* a new whiteboard created on mobile, *then* it appears in the
  course/meeting whiteboard list and opens on web.
- **AC-6.** *(If realtime)* *Given* two participants editing, *then* each sees
  the other's strokes within the agreed latency budget.

## 8. Data Model

- **No new tables.** Whiteboards + `canvasData` (element array) already exist.
  Client adds an editable element model + undo/redo stack in memory. Element
  shape must match the shipped `WhiteboardElement` used by the viewer.

## 9. API Surface

Existing endpoints (reused):

- `GET /api/v1/courses/{code}/whiteboards` — list.
- `POST /api/v1/courses/{code}/whiteboards` — create.
- `GET /api/v1/courses/{code}/whiteboards/{boardId}` — load.
- `PUT /api/v1/courses/{code}/whiteboards/{boardId}` — save `canvasData`.
- `DELETE /api/v1/courses/{code}/whiteboards/{boardId}` — delete.
- Realtime channel: TBD (see §14) — only if the web whiteboard is collaborative.

No new server routes for save-based authoring; a delta/realtime channel would be
a server addition to be confirmed.

## 10. UI / UX

- **Modified screen:** `Live/WhiteboardView` (iOS) / Android renderer become an
  editor with a floating toolbar (tools, color, width, undo/redo, add text).
- **New:** whiteboard list + create/delete within the meeting/course.
- **Flows:** open board → pick tool → draw → autosave; create new board; delete.
- **States:** loading, empty canvas, saving/saved, save-failed (retry),
  read-only, offline (queue saves), (optional) presence indicators.
- **Mobile/responsive:** collapsible toolbar; two-finger pan/zoom; stylus-first
  ergonomics; color picker sheet.
- **Accessibility:** labelled tools; text-add without drawing; reduced-motion;
  shape/color not sole differentiator.
- **Copy & i18n:** `mobile.whiteboard.*`.

## 11. AI / ML Considerations

Not AI-touching. (Shape-recognition/handwriting is a possible future add;
out of scope.)

## 12. Integration Points

- iOS: extend `Features/Live/WhiteboardView.swift`; new `WhiteboardLogic.swift`
  + `LMSAPIWhiteboard.swift`. Reuse `Core/Realtime/WebSocketClient.swift` if
  collaborative.
- Android: extend `core/lms/WhiteboardRenderer.kt` into an editor; new
  `WhiteboardApi.kt` + logic; `features/live/*`.
- Ties into the Live meetings feature (`Features/Live`, `LiveMeetingsLogic`).

## 13. Dependencies & Sequencing

- Must ship after: —.
- Must ship before: —.
- Shared infra: whiteboard storage (exists); realtime gateway only if
  collaborative editing is in scope.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| **Realtime unknown**: is the web whiteboard collaborative or save-based? | H | H | Spike first (§18 Q1); if save-based, ship authoring + last-writer-wins; add realtime as a follow-up |
| Concurrent edits clobber each other | M | H | Element-level merge or per-user layers; conflict UX; or lock while editing |
| Ink latency/jank on large boards | M | M | Native canvas rendering; element culling; debounced saves |
| Stylus/palm handling varies by device | M | M | Use platform pencil APIs; per-device QA matrix |

## 15. Rollout Plan

- Flag: `ff_mobile_whiteboard_edit` (default off).
- Sequence: authoring (save-based) behind flag → dogfood on tablets → GA;
  realtime as a separate flagged follow-up if warranted.
- GA criteria: AC-1..5 pass; ink performance target met on target devices.
- Rollback: flag off returns to read-only viewer.

## 16. Test Plan

- **Unit** — element (de)serialization round-trips web schema; undo/redo stack;
  tool state machine.
- **Integration** — create → edit → save → reload; cross-client render parity
  (mobile-authored board opens on web, AC-5).
- **End-to-end** — draw all tool types on device; delete; permission gating.
- **Security** — text-element sanitisation; edit authz.
- **Accessibility** — tool labels; text-add path; reduced-motion.
- **Performance** — ink latency + fps on min-spec + stylus devices; large-board
  pan/zoom.
- **Manual** — palm rejection; offline save queue; (if realtime) two-device sync.

## 17. Documentation & Training

- "Draw on a whiteboard on mobile" help article (stylus tips).
- Note on collaboration behaviour (realtime vs. save-based) once decided.

## 18. Open Questions

1. **Realtime:** is the web whiteboard collaborative (WS) or single-editor
   save-based? This decides scope and the presence/merge model. (Spike.)
2. Is the whiteboard scoped to Live meetings only, or also a standalone course
   surface on mobile?
3. Do we need image insertion (photo) on mobile v1, or shapes/pen/text only?
4. Conflict policy: last-writer-wins, element-merge, or edit-lock?

## 19. References

- Web: `clients/web/src/components/whiteboard/*`,
  `clients/web/src/lib/courses-api.ts` (whiteboards).
- iOS: `clients/ios/Lextures/Features/Live/WhiteboardView.swift`.
- Android: `.../core/lms/WhiteboardRenderer.kt`, `.../features/live/*`.
- Related (distinct feature): [MOB.8](MOB.8-collaboration-boards-completion.md).
