# AN.4 — Lists, Grids & Collection Motion

> Completed implementation plan. Source: [docs/plan/animations/README.md](../../plan/animations/README.md) (Motion & Animation Polish initiative).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AN.4 |
| **Section** | Motion & Animation Polish |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE — `AnimatedList` / `useListTransition` / list-motion + drag helpers, iOS `.lxListMotion` / `.lxListDragLift`, Android `Modifier.lxListMotion` / `lxListDragLift`, notifications + transcript order + pinned-course drag adoption, `ff_motion_lists` kill-switch |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Frontend Platform (web) + Mobile (iOS/Android) |
| **Depends on** | AN.1 |
| **Unblocks** | — |

---

## 1. Problem Statement

Once a screen is loaded, its collections mutate abruptly: an item deleted from a list makes the rows
below it **jump** up; a new notification, message, or feed post appears instantly; reordering (e.g.
[`transcript-order-builder.tsx`](../../../clients/web/src/components/lms/transcript-order-builder.tsx)
and other `@dnd-kit` surfaces) snaps rather than settling. There's no motion to communicate what
changed or where it went, so the user has to re-scan. Adding insert/remove/reorder choreography and a
tactile drag feel makes collections legible and the app feel physical.

**Scope note:** this story covers motion for **changes to an already-loaded collection**. The initial
skeleton→content reveal is [AN.3](AN.3-load-choreography.md); the two coordinate so a freshly loaded
list reveals (AN.3) and *subsequent* mutations animate (AN.4) without double-animating.

## 2. Goals

- Animate item **insertion** (enter), **removal** (exit + collapse the gap), and **reorder**
  (position transitions) in lists, grids, feeds, and carousels.
- Give drag-to-reorder a tactile feel: lift (scale/shadow), neighbors part to make room, drop settles
  with the bubble curve.
- Polish pull-to-refresh and infinite-scroll append so new content arrives smoothly.
- Deliver as reusable list-motion primitives; degrade cleanly under reduced motion.

## 3. Non-Goals

- Initial load reveal (AN.3).
- Full-screen navigation between list and detail (AN.2).
- Building new list/board features or changing data models behind them.
- Board canvas object motion beyond list-like rails (deep canvas physics is out of scope; board
  present-mode already has its own reduced-motion handling).

## 4. Personas & User Stories

- **As a student deleting a to-do / notification**, I want the item to slide out and the list to close
  the gap so I see exactly what was removed.
- **As an instructor reordering assignments or a transcript order**, I want items to lift and settle
  as I drag so reordering feels precise and physical.
- **As any user with a live feed/inbox**, I want new items to ease in at the top rather than shove
  existing content down.
- **As a motion-sensitive user**, I want mutations to be instant-or-fade with no sliding.
- **As a self-learner scrolling a long catalog**, I want appended pages to fade in without a jarring
  jump or scroll shift.

## 5. Functional Requirements

- **FR-1.** Inserting an item MUST animate it in (bubble enter) and, when inserted among existing
  items, MUST push neighbors via position transitions rather than an instant relayout.
- **FR-2.** Removing an item MUST animate it out (fade + collapse) and MUST animate the remaining
  items closing the gap, not jump.
- **FR-3.** Reordering (data-driven or drag) MUST animate items to their new positions using AN.1
  `standard`/bubble curves.
- **FR-4.** Drag-to-reorder MUST show a lift affordance (scale up slightly + elevated shadow) on
  grab, neighbors MUST make room, and drop MUST settle with the bubble curve; this applies to web
  `@dnd-kit` surfaces, iOS drag, and Android reorderable lists.
- **FR-5.** Infinite-scroll/pagination appends MUST fade/stagger new items in without shifting the
  user's current scroll position.
- **FR-6.** Pull-to-refresh MUST have a physical indicator (stretch/spinner tied to drag), and newly
  fetched items that differ from cache MUST animate in (not the whole list re-revealing).
- **FR-7.** All list motion MUST use stable item identity/keys so animations track the right element
  across data updates.
- **FR-8.** Under reduced motion, inserts/removes/reorders MUST be instant or opacity-only; drag lift
  reduces to a static elevation change with no scale.
- **FR-9.** List motion MUST remain 60fps for large/virtualized lists — only visible items animate;
  off-screen mutations apply without animation.

## 6. Non-Functional Requirements

- **Performance** — transform/opacity only; virtualization-aware; bounded concurrent animations
  (batch/stagger large diffs); no scroll-anchor breakage on append (FR-5). 60fps on target devices.
- **Security** — None.
- **Privacy & Compliance** — None.
- **Accessibility** — Live regions announce added/removed items regardless of animation; focus follows
  a sensible target after removal (next item); reordering exposes accessible move controls (not
  drag-only); reduced-motion honored.
- **Scalability** — Handles large diffs by capping simultaneous animations and applying the rest
  without motion.
- **Reliability** — Interrupted drags/rapid diffs never strand a ghost item or wrong order; final DOM/
  view state always matches the data.
- **Observability** — none required.
- **Maintainability** — one list-motion primitive per platform; surfaces adopt by wrapping their
  iteration.
- **Internationalization** — RTL-aware enter/exit direction; drag axis respects layout.
- **Backward compatibility** — Un-migrated lists keep instant behavior; incremental adoption.

## 7. Acceptance Criteria

- **AC-1.** *Given* a list, *When* an item is removed, *Then* it animates out and the gap closes
  smoothly; *When* an item is inserted, *Then* it eases in and neighbors shift via transition.
- **AC-2.** *Given* drag-reorder on a web `@dnd-kit` surface / mobile reorderable list, *When* I grab
  an item, *Then* it lifts; *When* I drop, *Then* it settles with the bubble curve and neighbors
  animate to final positions.
- **AC-3.** *Given* infinite scroll, *When* the next page loads, *Then* new items fade in and my scroll
  position does not jump.
- **AC-4.** *Given* reduced motion, *When* any mutation occurs, *Then* it is instant/opacity-only with
  no sliding and no drag scale.
- **AC-5.** *Given* a virtualized 500-item list, *When* items mutate, *Then* only visible items animate
  and the list stays 60fps.
- **AC-6.** *Given* an item removed, *When* it leaves, *Then* a screen reader announces the removal and
  focus moves to a valid neighbor.

## 8. Data Model

- No database changes. Requires stable, unique item keys/identity from existing data (audit lists that
  key by index and fix).
- No migration; no backfill.

## 9. API Surface

- No HTTP/WebSocket surface. Internal primitives:
  - Web: an `<AnimatedList>` / `useListTransition()` around mapped children; `@dnd-kit` drag styling
    hooks tuned with AN.1 tokens.
  - iOS: rely on `List`/`ForEach` identity + `.animation(_:value:)` and `.transition`, wrapped in a
    `lxListMotion` helper; `.onMove` drag polish.
  - Android: `Modifier.animateItemPlacement()` / `AnimatedVisibility` in `LazyColumn`/`LazyGrid`, plus
    reorderable-list drag helpers.

## 10. UI / UX

- **Modified surfaces** — notifications/inbox, feed, to-do/planner lists, gradebook/roster rows,
  course/catalog grids, dashboard carousels, transcript order builder
  ([`transcript-order-builder.tsx`](../../../clients/web/src/components/lms/transcript-order-builder.tsx)),
  any `@dnd-kit` reorder surface, discussion threads, live-quiz leaderboards (coordinate with AN.7).
- **Key flows** — (1) delete/add an item; (2) drag-reorder; (3) infinite-scroll append; (4)
  pull-to-refresh; (5) live update arriving via realtime.
- **Empty/loading/error/offline** — removing the last item animates into the empty state; failed
  optimistic insert animates back out.
- **Mobile/responsive** — native reorder gestures on mobile; hover-reveal drag handles on web.
- **Accessibility** — live-region announcements; keyboard reorder controls; focus management on
  removal.
- **Copy & i18n** — reuse existing add/remove strings; no new copy.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- **Web** — `@dnd-kit/*` (already a dependency), list-rendering components across
  `clients/web/src/components/**` and `pages/**`, realtime update handlers.
- **iOS** — `List`/`ForEach`/`LazyVStack` sites, drag `.onMove` handlers, realtime `onChange` reloads
  in dashboard/inbox.
- **Android** — `LazyColumn`/`LazyVerticalGrid` sites, reorderable list helpers, realtime flows.
- Consumes AN.1 tokens; coordinates with AN.3 (reveal vs mutate) and AN.7 (leaderboard motion).

## 13. Dependencies & Sequencing

- Must ship **after**: AN.1 (and coordinate with AN.3 for load-vs-mutate handoff).
- Must ship **before**: nothing.
- Shared infra: none beyond AN.1 and existing `@dnd-kit`.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Index-based keys cause wrong items to animate | M | H | Audit and fix to stable IDs before enabling (FR-7) |
| Large realtime diffs cause animation storms | M | M | Cap concurrent animations; apply overflow without motion (FR-9) |
| Append shifts scroll position | M | M | Scroll-anchoring; insert below the fold without moving viewport (FR-5) |
| Drag + reorder + reduced-motion interaction bugs | M | M | Explicit reduced-motion drag path (static elevation); test matrix |
| Double animation when AN.3 reveal overlaps a mutation | L | M | Coordinate "has revealed" state; mutations only animate post-reveal |

## 15. Rollout Plan

- **Feature flag** — `ff_motion_lists` (default off → on after QA), per-surface adoption.
- **Sequencing** — land primitives behind flag → adopt on inbox/notifications + transcript order
  builder first → expand to feeds/gradebooks/grids → enable by default.
- **Dogfood** — internal; watch for scroll-jump and key-mismatch reports.
- **GA criteria** — no scroll-position jumps, stable keys everywhere adopted, reduced-motion verified,
  60fps on virtualized lists.
- **Rollback** — flip `ff_motion_lists` off.

## 16. Test Plan

- **Unit** — list-transition primitive computes enter/exit/move states from a keyed diff; reduced-
  motion path is instant.
- **Integration** — iOS/Android reorder + insert/remove animate to correct final order; virtualized
  list only animates visible items.
- **End-to-end** — Playwright: `@dnd-kit` reorder lifts/settles; infinite scroll append keeps scroll
  position; reduced-motion emulation removes sliding.
- **Security** — n/a.
- **Accessibility** — live-region announcements on add/remove; keyboard reorder; focus on removal;
  axe clean.
- **Performance / load** — 500-item virtualized list mutation at 60fps; frame traces on low-end
  devices.
- **Manual exploratory** — rapid add/remove, interrupted drag, realtime burst updates, RTL.

## 17. Documentation & Training

- **Internal** — "Animating a list" recipe; keying requirements; reduced-motion drag guidance.
- **End-user** — none.
- **Runbook** — `ff_motion_lists` scope and kill-switch.

## 18. Open Questions

1. Which lists are truly virtualized today, and which need stable-key remediation first?
2. Do we adopt a shared reorderable-list helper on mobile or use per-screen native drag?
3. Should realtime-arriving items always animate, or only when the list is scrolled to where they
   land (to avoid off-screen churn)?

## 19. References

- Existing: [`clients/web/src/components/lms/transcript-order-builder.tsx`](../../../clients/web/src/components/lms/transcript-order-builder.tsx),
  `@dnd-kit/*` in [`clients/web/package.json`](../../../clients/web/package.json), list/feed
  components across `clients/web/src/**`, iOS `ForEach`/`LazyVStack` sites, Android `LazyColumn` sites.
- Standards: WCAG 4.1.3 (Status Messages), 2.1.1 (Keyboard), 2.3.1; Material 3 list motion; Apple HIG
  "Lists & drag".
- Related plans: [AN.1](AN.1-motion-foundation-tokens.md), [AN.3](AN.3-load-choreography.md),
  [AN.7](../../plan/animations/AN.7-delight-progress-moments.md).
