# CT.15 — Tool: Labeled Diagram & Hotspot (place labels on an image, or click the right region)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.15 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | CT.1, CT.2, CT.3, CT.14 (placement engine) |
| **Unblocks** | Anatomy, geography, circuit, map and chart-reading practice inline |

---

## 1. Problem Statement

Enormous amounts of curriculum are visual — label the parts of a cell, identify the region on a map,
click the resistor in the circuit, find the peak on the graph — and Lextures can render the image but
cannot ask anything about it. The question bank has a `hotspot` type usable only inside a formal quiz.
Inline, an author's only option is prose ("look at the third structure from the left"), which is both
worse teaching and worse accessibility than a real interaction. This tool makes image-anchored practice
placeable, gradable and — critically — operable without a mouse.

## 2. Goals

- Two modes on one surface: **label placement** (drag labels onto regions) and **hotspot identification**
  (click/tap the region that answers a prompt).
- Author regions on the image directly, with rectangle, circle and polygon shapes.
- Grade server-side against regions, with per-label feedback.
- Provide a genuinely equivalent non-visual path: every region has a required text label and description,
  and the whole activity is completable from a list-based keyboard interface.
- Give instructors a click/placement heat map over the image.

## 3. Non-Goals

- Image editing or annotation of arbitrary uploaded documents (the shipped PDF/whiteboard tools cover
  adjacent needs).
- Freehand drawing responses (the shipped whiteboard block is the right surface; a drawing-response tool
  is a future story).
- Pixel-perfect precision tasks (radiology-grade); tolerance is region-based by design.
- Auto-detecting regions from an image with AI (reserved, not v1).

## 4. Personas & User Stories

- **As a biology teacher**, I want students to label a cell diagram inline so that they practise before
  the lab practical.
- **As a geography teacher**, I want students to click the country a paragraph describes so that reading
  connects to the map.
- **As a physics teacher**, I want students to identify the component in a circuit so that symbols become
  fluent.
- **As a blind student**, I want an equivalent way to complete the activity so that a visual task is not
  an exclusion.
- **As an instructor**, I want to see where the class clicked so that I know whether they confused two
  adjacent structures.

## 5. Functional Requirements

- **FR-1.** The author MUST upload or reference an image (course files) and define regions as rectangle,
  circle or polygon in normalized coordinates (0–1), so the activity is resolution-independent.
- **FR-2.** Every region MUST require a `label` and a short `description`; the description is the
  non-visual equivalent and MUST NOT be optional.
- **FR-3.** The image MUST require meaningful alt text; the CT.2 alt-text enforcement path applies.
- **FR-4.** `mode: 'label'` MUST let a learner place each label chip onto a region (drag, tap-to-place,
  or keyboard select-and-assign).
- **FR-5.** `mode: 'hotspot'` MUST present one or more prompts, each answered by selecting a region.
- **FR-6.** Correct region mappings and feedback MUST be `x-lex-sensitive`; checking MUST be a server action.
- **FR-7.** A **list mode** MUST be available to every learner (not only assistive-technology users),
  presenting regions as a labelled list with their descriptions, so the activity can be completed
  entirely without spatial interaction. It MUST be reachable by an always-visible control, not hidden
  behind a "accessibility mode" stigma.
- **FR-8.** Placement/selection MUST support keyboard: `Tab` to a label, `Enter` to pick up, arrow keys
  to cycle regions (announced by label position: "top-left region, 2 of 8"), `Enter` to place.
- **FR-9.** The author MUST configure attempts, per-item correctness display, and whether correct
  placements lock.
- **FR-10.** Scoring MUST be per label/prompt with a fraction-correct default, reported to CT.7.
- **FR-11.** The instructor MUST see a heat map of clicks/placements over the image plus a per-region
  table, and MUST be able to spot "labels most often swapped".
- **FR-12.** The image MUST be zoomable and pannable on small screens, with placement accuracy preserved
  under zoom.
- **FR-13.** The tool MUST honour `prefers-reduced-motion` and MUST not rely on hover for any affordance.
- **FR-14.** CT.4 reset MUST return all labels to the tray and clear attempts.

## 6. Non-Functional Requirements

- **Performance** — Image loaded responsively (srcset from the shipped image pipeline); interaction at
  60 fps; check p95 ≤ 200 ms; renderer ≤ 38 KB gz excluding the image.
- **Security** — Region answers server-side only; images served through existing course-file
  authorization so a shared link cannot leak course media.
- **Privacy & Compliance** — Placements are student work; no AI, no egress.
- **Accessibility** — WCAG 2.1 AA. The declaration states: full keyboard operation, a non-spatial list
  mode, required region descriptions, and no colour-only encoding. Known limitation documented: learners
  who cannot perceive the image rely on author-written descriptions, so authoring guidance and a
  description-quality warning in the editor are part of the feature.
- **Scalability** — Heat map aggregated server-side (CT.7) into a coarse grid, not raw point storage.
- **Reliability** — Draft placements autosave; idempotent checks.
- **Observability** — `lextures_content_tool_checks_total{tool_id="diagram_hotspot",mode,outcome}`,
  list-mode usage rate, per-region error rate.
- **Maintainability** — Reuses the CT.14 placement engine; region geometry is a small, tested module.
- **Internationalization** — Labels and descriptions are author content; RTL layout mirrors the tray but
  not the image; announcements localized.
- **Backward compatibility** — Additive; independent of quiz `hotspot`.

## 7. Acceptance Criteria

- **AC-1.** *Given* an unchecked learner, *Then* the payload contains regions (shape + label +
  description) but no correct mapping.
- **AC-2.** *Given* a learner places all labels and checks, *Then* per-label correctness and score are
  returned by the server and stored.
- **AC-3.** *Given* keyboard-only operation, *When* the learner cycles regions and places a label,
  *Then* each step is announced with the region's label position and the result equals a pointer drag.
- **AC-4.** *Given* list mode, *When* a learner assigns every label via the list, *Then* the activity
  completes and scores identically to spatial placement.
- **AC-5.** *Given* an author saves a region without a description, *Then* the editor blocks the save
  with a clear message.
- **AC-6.** *Given* an image without alt text, *Then* the CT.2 alt-text enforcement blocks or warns per
  platform setting.
- **AC-7.** *Given* the learner zooms to 300% on a phone and places a label, *Then* the placement maps
  to the correct normalized coordinates.
- **AC-8.** *Given* 30 learners, *When* the instructor opens the heat map, *Then* densities match the
  aggregate and the per-region table shows the same numbers.
- **AC-9.** *Given* `prefers-reduced-motion`, *Then* no animated transitions play.
- **AC-10.** *Given* a CT.4 reset, *Then* labels return to the tray and attempts are cleared.

## 8. Data Model

**No migration.**

```ts
// configSchema
type DiagramHotspotConfig = {
  mode: 'label' | 'hotspot'
  prompt: string
  image: { url: string; alt: string; naturalWidth: number; naturalHeight: number }
  regions: Array<{
    id: string
    label: string                 // visible name of the region
    description: string           // REQUIRED non-visual equivalent
    shape:
      | { kind: 'rect'; x: number; y: number; w: number; h: number }        // normalized 0..1
      | { kind: 'circle'; cx: number; cy: number; r: number }
      | { kind: 'polygon'; points: Array<[number, number]> }
  }>
  labels?: Array<{ id: string; text: string }>                 // label mode
  correctRegionByLabel?: Record<string, string>                // x-lex-sensitive
  prompts?: Array<{ id: string; text: string }>                // hotspot mode
  correctRegionByPrompt?: Record<string, string>               // x-lex-sensitive
  feedbackByRegion?: Record<string, string>                    // x-lex-sensitive
  attempts: number | 'unlimited'   // default 3
  lockCorrect: boolean             // default true
  showRegionOutlines: 'always' | 'on_focus' | 'after_check'    // default 'on_focus'
}

// stateSchema
type DiagramHotspotState = {
  v: 1
  assignments: Record<string, string | null>     // labelId|promptId → regionId
  attempts: Array<{ at: string; correctIds: string[]; scorePct: number }>
  lockedIds: string[]
  usedListMode?: boolean
  completedAt?: string
}
```

`scoring.mode = 'auto'`; `capabilities = ['state','scoring','media']`; `maxStateBytes = 24000`.

## 9. API Surface

**No new routes.**

- `PUT .../state` — draft assignments.
- `POST .../actions/check` — `{assignments, idempotencyKey}` → `{perItem, scorePct, attemptsRemaining, state}`.
- Heat map via CT.7 facets `regionId`, `assignedTo`, `correct` (aggregated to a coarse grid server-side).

## 10. UI / UX

**Label mode** — image on the left/top with region outlines shown per config; a tray of label chips;
drag or tap-to-place; placed labels render as pinned chips with leader lines.
**Hotspot mode** — a prompt above the image; clicking/tapping a region selects it, with the selection
echoed as text ("Selected: upper-left chamber").

Always-visible **List view** toggle switches to a two-column list (labels ↔ regions with descriptions)
with select menus — the same activity, no spatial interaction.

**States** — *Unassigned*, *Assigning (draft saved)*, *Checked*, *Exhausted*, *Read-only*, *Image
failed to load* (list mode offered automatically), *Error (retry, assignments preserved)*.

**Mobile** — pinch-zoom and pan; tap-to-place default; tray becomes a horizontally scrolling strip;
list view is one tap away.

**Accessibility** — regions are focusable elements with `aria-label` = "{label}: {description}"; the
image has meaningful alt text describing the whole diagram; announcements on pick-up, cycle and place;
outlines have ≥ 3:1 contrast; correctness by icon + text; no hover-only affordances.

**Copy & i18n** — `contentTools.tools.diagramHotspot.*`.

**Authoring** — custom editor: upload/pick image → draw regions with rect/circle/polygon tools →
required label + description per region (save blocked without them) → assign correct mappings →
preview in both spatial and list modes.

## 11. AI / ML Considerations

None in v1. Reserved and explicitly deferred: (a) AI-suggested region descriptions from the image
(would reuse the shipped `service/alttextai`), which is attractive for authoring speed but must not
become an excuse for unreviewed descriptions — any generated description would be pre-filled and
require author confirmation; (b) auto-detection of regions in common diagram types.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/diagramhotspot/`, CT.14 placement engine,
  `service/filestorage` + course-file image serving, `service/imagealt` / alt-text enforcement,
  `clients/web/src/components/editor/block-editor/course-aware-tip-tap-image.tsx` patterns for image
  handling, `clients/web/src/components/content-tools/tools/diagram-hotspot/`.
- **CT.7** — region facets, grid heat map.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.14 (placement engine), CT.1–CT.3.
- **Must ship before:** nothing.
- **Shared infra needed:** image storage/serving already shipped.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Non-visual learners excluded despite list mode | M | H | Required region descriptions enforced at authoring, description-quality warning, list mode always visible, screen-reader validation per release |
| Authors write useless descriptions ("region 1") | H | H | Editor blocks save on empty and warns on descriptions matching the label or under N characters; documentation with examples |
| Region drawing is fiddly for authors | M | M | Snap-to-grid, keyboard nudging, polygon point editing, undo, preview |
| Zoom breaks placement accuracy | M | M | Normalized coordinates throughout; zoom tested at 100/200/400% |
| Large images hurt performance on low-end devices | M | M | Responsive srcset, dimension caps, lazy load, compression guidance |
| Copyrighted images used as diagrams | M | M | Existing course-file policy and guidance apply |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist.
- **Sequencing** — region geometry module → authoring editor → renderer (spatial + list + keyboard) →
  grading → heat map → pilot in an anatomy/geography course.
- **Dogfood** — one anatomy unit and one map unit; blind-user validation session before GA.
- **GA criteria** — list mode proven equivalent; a11y audit passed; author guidance published.
- **Rollback** — remove from the allowlist.

## 16. Test Plan

- **Unit** — point-in-shape math for rect/circle/polygon (including edges and concave polygons);
  normalization under zoom; grading; lock/attempt accounting.
- **Integration** — key redaction; draft persistence; reset; alt-text enforcement interaction.
- **End-to-end** — Playwright: pointer, touch, keyboard and list paths each complete the activity with
  identical resulting state; image-load failure falls back to list mode.
- **Security** — payload inspection; tampered region ids; image authorization (cross-course URL).
- **Accessibility** — axe; screen-reader script for spatial and list modes; contrast of outlines;
  zoom to 400%; reduced motion.
- **Performance** — interaction frame rate with a 4000px source image; chunk budget.
- **Manual exploratory** — polygon-heavy maps, overlapping regions, very small regions on phones, RTL.

## 17. Documentation & Training

- **Instructor** — authoring good regions and descriptions (with examples of bad ones); choosing
  outline visibility; when list mode changes the task.
- **Student** — keyboard and list-mode instructions surfaced in-tool.
- **Developer** — region geometry module and its reuse.

## 18. Open Questions

1. Should overlapping regions be allowed (a region inside a region)? Proposed: yes, with an explicit
   z-order and "smallest match wins" documented.
2. Should list mode be the default on touch devices under a certain width? Proposed: offer it
   prominently, do not force it.
3. Should we allow SVG images with `<title>`/`<desc>` per shape as an accessibility-superior path?
   Proposed: yes as a follow-up — SVG-native regions would remove the description-quality risk.

## 19. References

- Existing files this work touches: `clients/web/src/components/editor/block-editor/course-aware-tip-tap-image.tsx`,
  `server/internal/service/filestorage/`, `server/internal/service/imagealt/`,
  `clients/web/src/components/content-tools/`.
- Precedents: quiz `hotspot` question type (`server/migrations/075_question_bank.sql`); alt-text
  enforcement (`clients/web/src/components/editor/block-editor/alt-text-*`).
- External standards: WCAG 2.1 AA — 1.1.1 Non-text Content, 1.4.11 Non-text Contrast, 2.1.1, 2.5.7.
- Related plans: [CT.14](CT.14-tool-sort-and-sequence.md),
  [CT.8](CT.8-governance-safety-privacy-accessibility.md),
  [CT.7](CT.7-analytics-insights-and-gradebook.md).
