# CT.M7 — Mobile Tool Pack 3: Direct Manipulation (Sort & Sequence, Highlight & Annotate, Labeled Diagram)

> Implementation plan. Source: mobile renderers for [CT.14](CT.14-tool-sort-and-sequence.md), [CT.13](CT.13-tool-highlight-and-annotate.md), [CT.15](CT.15-tool-labeled-diagram-and-hotspot.md). Folder overview: [README](../../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.M7 |
| **Section** | Content Tools (CT) — Mobile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad + design (consult) |
| **Depends on** | CT.M3 |
| **Unblocks** | — (parallel with CT.M5, CT.M6, CT.M8) |

---

## 1. Problem Statement

These three tools are the ones a touchscreen should be *best* at and that a naive port makes worst.
`sort_sequence` drags items into categories or order; `highlight_annotate` asks a student to select
passages of prose against a prompt; `diagram_hotspot` drops labels onto an image. On the web they are
mouse-precise. On a phone, drag-and-drop fights the page scroll, text selection fights the OS's own
selection UI, and a 12 px hotspot is smaller than a fingertip. Each also has a hard accessibility
obligation: WCAG 2.1 §2.5.7 requires that any dragging action have a single-pointer alternative.
CT.M7 delivers these tools with touch interactions designed for a phone and a non-drag path that is a
first-class way to complete them, not a compliance afterthought.

## 2. Goals

- Ship native renderers for `sort_sequence`, `highlight_annotate` and `diagram_hotspot` against the
  CT.M3 contract.
- Design touch interactions that do not fight scrolling, selection or zoom.
- Provide a **tap-to-assign** alternative to every drag, usable by everyone (not a hidden a11y mode),
  satisfying WCAG 2.1 AA §2.5.7 and §2.5.1.
- Keep grading, correctness and aggregation server-side; the client never holds an answer key.
- Handle images responsibly: fit-to-width, pinch-zoom, hotspot targets sized for fingers, and correct
  coordinate mapping across zoom, rotation and screen density.

## 3. Non-Goals

- Authoring: defining categories, prompts, diagrams or hotspot regions (web-only).
- Instructor heat maps and annotation aggregates (CT.7, web).
- Freehand drawing or ink annotation — that is the whiteboard/board feature, not these tools.
- Changing the stored state or coordinate schema; mobile writes exactly what web writes.
- OCR or image analysis of any kind.

## 4. Personas & User Stories

- **As a student**, I want to sort the terms into categories with my thumb without the page scrolling
  away underneath me.
- **As a student with a motor impairment**, I want to complete the sort by tapping an item and then
  tapping its category — no dragging at all.
- **As a student**, I want to highlight the sentence that answers the prompt without fighting the copy/
  paste bubble.
- **As a student**, I want to zoom into the diagram to place a label accurately.
- **As a screen-reader user**, I want to place labels through a list-based picker with clear positions.
- **As an instructor**, I want the same evidence quality from a phone as from a laptop.

## 5. Functional Requirements

**Shared**

- **FR-1.** Each renderer MUST register in CT.M3's registry and use only the host's state and action
  APIs. Conflict policy is `server_wins` for `sort_sequence` and `diagram_hotspot`, `merge` for
  `highlight_annotate`.
- **FR-2.** Every drag interaction MUST have an equivalent **single-pointer, non-path-based**
  alternative that is visible by default — select-then-place — satisfying WCAG 2.1 §2.5.7 and §2.5.1.
- **FR-3.** No renderer may compute correctness locally or receive an answer key; `check` actions
  (`sort_sequence`, `diagram_hotspot`) grade server-side and the client renders the returned result.
- **FR-4.** Drag gestures MUST NOT conflict with page scroll: a long-press-to-lift threshold engages
  the drag, the enclosing scroll view is locked during a drag, and edge auto-scroll is bounded.
- **FR-5.** Interactive state MUST autosave through the host on settle (not per-pixel), and every
  reorder/assignment MUST be announced through the live region.
- **FR-6.** Haptic feedback MUST accompany lift, valid drop and invalid drop, and MUST respect the OS
  reduced-motion / haptics settings.

**`sort_sequence`** (caps: `state`, `scoring`; scoring `auto`)

- **FR-7.** MUST support both configured modes — categorize (items → buckets) and order (items into a
  sequence) — with a layout that works in a single phone column.
- **FR-8.** MUST provide tap-to-assign: tap an item to select it, then tap a category or an insertion
  point to place it; and "move up"/"move down" actions in ordering mode.
- **FR-9.** MUST call the `check` action for grading and `reset_attempt` to start over, honouring the
  configured attempt limit.
- **FR-10.** MUST render item text through CT.M1 (items are often code or formulas).

**`highlight_annotate`** (caps: `state`, `aggregate`)

- **FR-11.** MUST let a student select a passage of the configured text and tag it against the prompt's
  categories, storing selection **offsets against the canonical source text** so a mobile-created
  annotation lands in the same place on web.
- **FR-12.** MUST use a purpose-built selection UI rather than fighting the OS text-selection menu:
  selection handles are the tool's own, and the system copy/paste callout MUST be suppressed inside the
  passage.
- **FR-13.** MUST offer a **sentence/paragraph tap** mode as the non-drag alternative — tap a sentence
  to select it whole, then choose a tag.
- **FR-14.** MUST support adding a note to an annotation, editing and deleting one's own annotations,
  and MUST render the categories with icon + text (colour alone is insufficient).
- **FR-15.** MUST handle text reflow correctly: offsets are computed against the source, never against
  the laid-out glyph positions, so font scale and rotation cannot corrupt stored ranges.

**`diagram_hotspot`** (caps: `state`, `scoring`, `media`; scoring `auto`)

- **FR-16.** MUST render the configured image fit-to-width with pinch-to-zoom and pan, and MUST map
  label placements to **normalised image coordinates** so placements are density- and zoom-independent
  and interoperate with web.
- **FR-17.** MUST enforce a minimum effective touch target for hotspots (≥ 44 pt / 48 dp) regardless of
  the authored region size, expanding the hit area without changing the stored coordinates.
- **FR-18.** MUST provide a list-based placement alternative: choose a label, then choose its target
  from a named list of hotspots (using the author's hotspot labels), for both non-drag and
  screen-reader use.
- **FR-19.** MUST call `check` for grading and `reset_attempt` to retry, honouring attempt limits, and
  MUST render the image's alt text and any per-hotspot accessible names the author provided.
- **FR-20.** MUST show placement feedback positionally *and* as a text list, so a student who cannot
  perceive the image still gets the result.

## 6. Non-Functional Requirements

- **Performance** — Drag tracks at 60 fps with up to 30 items; pinch-zoom on a 4000 px image stays
  smooth via downsampled decoding; autosave on settle, coalesced, never per-frame.
- **Security** — Diagram images load through the authenticated image loader only; no answer key,
  correct-region set, or peer annotation reaches the client except as the server returns it.
- **Privacy & Compliance** — Annotations and placements are education records (S01/S02). Peer
  annotations are only visible where the server exposes them, aggregated and anonymised.
- **Accessibility** — WCAG 2.1 AA, with §2.5.7 (dragging movements) and §2.5.1 (pointer gestures) as
  first-class requirements rather than fallbacks: every task completable by single taps; every state
  change announced; categories and correctness never colour-only; hit targets ≥ 44 pt / 48 dp;
  reduced-motion honoured; 200% font scale supported in item lists and annotation UI.
- **Scalability** — Client-side; annotation aggregates are server-computed.
- **Reliability** — A drag interrupted by a call, rotation or background restores to the last settled
  state, never a half-applied one; attempt limits are server-enforced.
- **Observability** — Counters for drag vs tap-to-assign completion rate (the signal that tells us
  whether the touch design works), check outcomes, and image load failures — labelled `tool_id`.
- **Internationalization** — `mobile.contentTools.tools.{sort_sequence,highlight_annotate,
  diagram_hotspot}.*`; RTL: category columns and ordering direction mirror, and annotation offsets are
  direction-agnostic because they are source offsets.
- **Backward compatibility** — Coordinate and offset schemas are identical to web's; a mobile-created
  annotation or placement MUST render identically on web and vice versa (this is an explicit AC).

## 7. Acceptance Criteria

- **AC-1.** *Given* a `sort_sequence` categorize task, *When* a student drags items into buckets and
  checks, *Then* the server grades it and per-item feedback renders.
- **AC-2.** *Given* the same task, *When* the student uses tap-to-assign only, *Then* it is fully
  completable with no dragging and the resulting state is identical.
- **AC-3.** *Given* a drag in progress, *Then* the enclosing page does not scroll, and an interrupted
  drag restores the last settled state.
- **AC-4.** *Given* an ordering task, *Then* "move up"/"move down" actions exist and are labelled.
- **AC-5.** *Given* `highlight_annotate`, *When* a student selects a passage on mobile and tags it,
  *Then* opening the same activity on web shows the annotation over the same characters.
- **AC-6.** *Given* the passage, *Then* the OS copy/paste callout does not appear during tool selection.
- **AC-7.** *Given* sentence-tap mode, *Then* a whole sentence can be selected and tagged without a
  drag.
- **AC-8.** *Given* 200% font scale and rotation after annotating, *Then* the annotation still covers
  the same source text.
- **AC-9.** *Given* `diagram_hotspot`, *When* a label is placed on mobile, *Then* web renders it in the
  same position (normalised coordinates).
- **AC-10.** *Given* an authored hotspot smaller than 44 pt, *Then* the effective touch target is at
  least 44 pt and the stored coordinates are unchanged.
- **AC-11.** *Given* a screen-reader user, *Then* every label can be placed via the list-based picker
  and the result is read back as text.
- **AC-12.** *Given* any of the three tools, *Then* correctness and categories are conveyed with icon
  or text in addition to colour.
- **AC-13.** *Given* attempt limits, *Then* `check` beyond the limit is rejected server-side and the
  client reflects it.
- **AC-14.** *Given* a read-only frame, *Then* placements and annotations render with no editing
  affordance.
- **AC-15.** *Given* CI, *Then* iOS build, Android compile and the renderer logic suites pass.

## 8. Data Model

**No server schema change, no migration.** State follows each manifest under
`server/internal/service/contenttools/tools/{sort_sequence,highlight_annotate,diagram_hotspot}/
manifest.json`. Two invariants are load-bearing for cross-platform parity and are called out as
requirements rather than implementation detail:

- `highlight_annotate` stores **character offsets into the canonical source text** (start, end, tag,
  note), never rendered-glyph positions.
- `diagram_hotspot` stores **normalised coordinates** in [0,1] relative to the intrinsic image size,
  never device pixels.

## 9. API Surface

**No new endpoints.** CT.M3's state routes plus:

| Tool | Actions |
|---|---|
| `sort_sequence` | `check`, `reset_attempt` |
| `highlight_annotate` | `filter_note` |
| `diagram_hotspot` | `check`, `reset_attempt` |

Diagram images are fetched through the existing authenticated course-file/image path.

## 10. UI / UX

- **New (iOS)** — `Features/ContentTools/Tools/{SortSequenceToolView,HighlightAnnotateToolView,
  DiagramHotspotToolView}.swift`, shared `Features/ContentTools/Interaction/{DragOrTapAssign,
  ZoomableImageCanvas,PassageSelectionView}.swift`, `Core/LMS/ContentToolPack3Logic.swift` (pure:
  offset math, normalised-coordinate mapping, hit-target expansion, reorder operations — unit-tested).
- **New (Android)** — `features/contenttools/tools/{SortSequenceTool,HighlightAnnotateTool,
  DiagramHotspotTool}.kt`, `features/contenttools/interaction/*`, `core/lms/ContentToolPack3Logic.kt`.
- **Key flows** — (1) Sort: lift → drag → drop, *or* tap item → tap bucket → Check → feedback.
  (2) Annotate: select (drag handles or sentence tap) → choose tag → optional note → saved.
  (3) Diagram: zoom → drag label onto the image, *or* pick label → pick hotspot from list → Check.
- **States** — *Not started*: instructions + items. *In progress*: selection highlighted, unassigned
  count shown. *Checked*: per-item/per-placement correctness with icon + text. *Attempts exhausted*:
  read-only with the last result. *Image failed to load*: alt text + retry, tool still completable via
  the list picker where possible. *Read-only*: rendered result, no controls.
- **Accessibility annotations** — every item and target is a named control; selection state is in the
  accessible name ("Selected. Photosynthesis. Double-tap a category to place"); reorder actions exposed
  as custom actions on iOS and accessibility actions on Android; the image exposes alt text and each
  hotspot exposes its author-provided name.
- **Copy & i18n** — per-tool namespaces plus shared `mobile.contentTools.interaction.*` strings for
  lift/place/move affordances.

## 11. AI / ML Considerations

None of the three declare the `ai` capability. Grading is deterministic and server-side. (Authoring
these tools may use AI on the web; mobile only renders and submits.)

## 12. Integration Points

- **Internal** — CT.M3 host/state/actions/live region; CT.M1 renderer for item and passage text;
  `AuthorizedNotebookImage` / the Android authenticated image loader for diagrams;
  `Core/Design/Haptics.swift` and its Android counterpart; `Core/Accessibility` for custom actions.
- **Server (unchanged)** — `server/internal/service/contenttools/{sort_sequence,highlight_annotate,
  diagram_hotspot}_actions.go`.
- **Web (parity reference)** — `clients/web/src/components/content-tools/tools/{sort_sequence,
  highlight_annotate,diagram_hotspot}/renderer.tsx` define the coordinate and offset semantics mobile
  must match exactly.
- **Events** — server-side CT.7 analytics only.

## 13. Dependencies & Sequencing

- Must ship after: **CT.M3**.
- Independent of CT.M4, CT.M5, CT.M6, CT.M8.
- Recommended order: `sort_sequence` (establishes the shared drag-or-tap interaction) →
  `diagram_hotspot` (reuses it plus the zoom canvas) → `highlight_annotate` (the hardest, because of
  OS text-selection conflicts).
- Design partnership is a real dependency here, not a nicety: the touch interaction model should be
  prototyped and usability-tested before the second tool is built.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Drag-and-drop fights page scroll and feels broken | H | H | Long-press-to-lift threshold, scroll lock during drag, bounded edge auto-scroll; prototype and usability-test before building all three |
| Text selection fights the OS copy/paste UI | H | H | Custom selection layer with system callout suppressed (FR-12); sentence-tap mode as the reliable path |
| Annotation offsets drift between mobile and web | M | H | Offsets against canonical source text only (FR-15); explicit cross-platform AC-5 |
| Hotspot coordinates drift across density/zoom | M | H | Normalised coordinates (FR-16); explicit cross-platform AC-9 |
| The non-drag alternative is built as a hidden a11y mode and rots | M | H | FR-2 makes it visible by default; the drag-vs-tap completion counter monitors real use |
| Authored hotspots too small for fingers | H | M | Hit-target expansion without changing stored coordinates (FR-17) |
| Large diagram images blow memory on low-end devices | M | M | Downsampled decoding sized to the viewport; explicit low-end device test |

## 15. Rollout Plan

- **Feature flag** — `mobileContentToolsEnabled` plus per-tool allowlist entries.
- **Sequencing** — interaction prototype + usability test → `sort_sequence` → `diagram_hotspot` →
  `highlight_annotate`, each behind its entry.
- **Dogfood** — an internal course with all three placed, tested on a small phone, a large phone and a
  tablet, and with a motor-impairment simulation (switch control / Voice Control only).
- **GA criteria** — all ACs green; cross-platform parity verified by creating on mobile and reading on
  web *and* vice versa; a11y sign-off explicitly covering §2.5.7; usability test shows tap-to-assign
  completion is not slower than drag by more than a small margin.
- **Rollback** — per-tool allowlist removal; stored annotations and placements are unaffected and
  remain readable on web.

## 16. Test Plan

- **Unit** — offset arithmetic across insertions and multi-byte characters (emoji, combining marks,
  RTL); normalised-coordinate mapping across zoom/rotation/density; hit-target expansion; reorder
  operations; drag-state restoration after interruption.
- **Integration** — `check`/`reset_attempt` round-trips with attempt limits; annotation create/edit/
  delete; cross-platform fixtures: a state document written by web renders identically on mobile and
  vice versa.
- **End-to-end (device)** — complete each tool twice: once by dragging, once entirely by tapping;
  interrupt a drag with a rotation and with a simulated call; zoom then place a label.
- **Security** — no answer key or correct-region set in any client payload; images require auth.
- **Accessibility** — scripted VoiceOver and TalkBack completion of all three tools end to end;
  switch-control and Voice Control passes; colour-blind simulation; 200% font scale; RTL.
- **Performance / load** — 30-item drag frame rate; 4000 px image zoom on a low-end Android device;
  memory ceiling.
- **Manual exploratory** — annotate then change font scale; annotate the same passage twice; place all
  labels then reset; rapidly toggle between drag and tap modes; landscape and split view.

## 17. Documentation & Training

- End-user: "Two ways to do it" — document tap-to-assign alongside dragging so students discover it.
- Instructor: authoring guidance for mobile — hotspot size guidance, passage length, item count.
- Accessibility: record the §2.5.7 conformance argument in the accessibility conformance report.
- Internal: the shared interaction components and their contract, in the mobile README.

## 18. Open Questions

1. Should `highlight_annotate` on mobile default to **sentence-tap** mode with free selection as the
   opt-in, rather than the reverse? (Recommendation: yes on phones, free selection on tablets —
   validate in usability testing.)
2. Does the annotation offset scheme survive ACE-rewritten variants, where the learner's prose differs
   from the canonical source? (Check CT.13's answer and mirror it; do not invent a mobile rule.)
3. Do we need a "review my placements" list view for `diagram_hotspot` on small screens even for
   sighted users? (Recommendation: yes — it is FR-18's picker reused, and it costs nothing.)
4. Is edge auto-scroll during drag worth the complexity on phones, or should long lists page instead?
   (Decide from the prototype.)

## 19. References

- Web plans: [CT.14](CT.14-tool-sort-and-sequence.md),
  [CT.13](CT.13-tool-highlight-and-annotate.md),
  [CT.15](CT.15-tool-labeled-diagram-and-hotspot.md).
- Web renderers: `clients/web/src/components/content-tools/tools/{sort_sequence,highlight_annotate,
  diagram_hotspot}/renderer.tsx`.
- Server: `server/internal/service/contenttools/{sort_sequence,highlight_annotate,diagram_hotspot}_actions.go`.
- Related plans: [CT.M3](CT.M3-mobile-content-tool-host-and-state.md),
  [CT.M9](../../plan/content_tools/CT.M9-mobile-tools-governance-a11y-telemetry.md);
  accessibility checklist `../../accessibility/mobile-audit-checklist.md`.
- Standards: WCAG 2.1 AA §2.5.7 (dragging movements), §2.5.1 (pointer gestures), §2.5.5 (target size),
  §1.4.1 (use of colour), §1.3.1 (info & relationships).
