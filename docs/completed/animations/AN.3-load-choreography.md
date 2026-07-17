# AN.3 — Load Choreography: Skeleton → Content

> Completed implementation plan. Source: [docs/plan/animations/README.md](../../plan/animations/README.md) (Motion & Animation Polish initiative).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AN.3 |
| **Section** | Motion & Animation Polish |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE — `LoadReveal` / `StaggerReveal` / `useReveal`, iOS `.lxStaggeredReveal` + `LXLoadReveal`, Android `LoadReveal` / `StaggeredReveal` / `Modifier.lxReveal`, dashboard adoption, `ff_motion_reveal` kill-switch |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Frontend Platform (web) + Mobile (iOS/Android) |
| **Depends on** | AN.1 |
| **Unblocks** | — |

---

## 1. Problem Statement

This is the flagship problem the initiative was raised on: after a clean splash/route transition,
data-backed screens **pop**. On the mobile dashboard, the skeleton
([`LMSSkeletonList`](../../../clients/ios/Lextures/Features/Dashboard/DashboardView.swift)) is
replaced the instant the network resolves and every card/stat appears simultaneously with no
entrance — the `if model.loading { skeleton } else { cards }` swap in
[`DashboardView.swift:231`](../../../clients/ios/Lextures/Features/Dashboard/DashboardView.swift) is
un-animated, and the same is true on Android
([`DashboardTab.kt`](../../../clients/android/app/src/main/kotlin/com/lextures/android/features/dashboard/DashboardTab.kt))
and across web pages. The abruptness undoes the polish of everything leading up to it. Content
should **resolve into place**: skeletons crossfade to real content, and cards/rows/stats stagger in
with the AN.1 "bubble" so the screen feels composed, not dumped.

## 2. Goals

- Crossfade skeleton/placeholder → loaded content everywhere, replacing the hard swap.
- Stagger the entrance of cards, list rows, stat tiles, and sections as data resolves (capped so it
  never feels slow), using AN.1 tokens and the bubble curve.
- Animate content-size changes (a section growing/shrinking as data fills) rather than jumping.
- Apply the pattern as a reusable primitive so every current and future data screen inherits it.
- Guarantee reduced-motion degrades to a simple opacity crossfade with no stagger delay.

## 3. Non-Goals

- The route/screen transition that *precedes* content load — that is [AN.2](AN.2-launch-navigation-transitions.md).
- List *mutation* motion (insert/remove/reorder after initial load) — that is [AN.4](AN.4-lists-collections-motion.md).
- Redesigning skeleton shapes or building new empty-state art (motion of existing states only).
- Streaming/partial-hydration architecture changes; we animate whatever the existing data-loading
  produces.

## 4. Personas & User Stories

- **As a student opening the dashboard**, I want cards to ease in as they load so the screen feels
  like it's assembling itself, not glitching into existence.
- **As an instructor loading a gradebook/roster**, I want rows to resolve smoothly so a large table
  doesn't slam onto the screen.
- **As any user on a slow connection**, I want the skeleton→content handoff to feel intentional so
  waiting feels designed rather than broken.
- **As a motion-sensitive user**, I want content to simply fade in without staggered movement.
- **As a self-learner**, I want the first meaningful paint to still be fast — animation must not
  delay when I can read/act on content.

## 5. Functional Requirements

- **FR-1.** When a data region finishes loading, its skeleton/placeholder MUST crossfade to real
  content over `fast`–`base` (150–220ms), not swap instantly.
- **FR-2.** Collections of peers (dashboard cards, list rows, stat tiles, carousel items) MUST enter
  **staggered** using AN.1 `staggerStep`, capped at `staggerMax`; items beyond the cap fade in as one
  group so total choreography stays ≤ ~400ms.
- **FR-3.** Entrance uses the AN.1 **bubble** curve with ≤12px/8dp upward translate + scale-from 0.97;
  content MUST end at its exact final position (no residual offset).
- **FR-4.** A section whose height changes as data arrives MUST animate its size (web
  `grid`/`height` via transform-safe technique or `animateContentSize` equivalents on mobile), not
  jump.
- **FR-5.** Entrance animations MUST run **once** per data resolution, not on every re-render, scroll,
  or refresh-in-place (track "has entered" so a background refresh doesn't re-stagger the screen).
- **FR-6.** Under reduced motion, content MUST fade in ≤100ms with **no** stagger and **no** transform.
- **FR-7.** The pattern MUST be delivered as a reusable primitive: web `<StaggerReveal>` /
  `useReveal()`; iOS `.lxStaggeredReveal(index:)` / a reveal container; Android a `StaggeredReveal`
  composable / `Modifier.lxReveal(index)`.
- **FR-8.** Pull-to-refresh and background refetch MUST NOT re-trigger the full entrance; only genuinely
  new content animates (ties into AN.4 for inserts).
- **FR-9.** Animation MUST NOT delay interactivity: content is tappable as soon as it's laid out, even
  mid-fade.

## 6. Non-Functional Requirements

- **Performance** — Stagger uses transform/opacity only; on web, avoid animating layout — reserve
  space with the skeleton so content reveal causes **zero** layout shift (CLS = 0). 60fps on target
  low-end devices even with many items (cap + group-fade guarantees bounded work).
- **Security** — None.
- **Privacy & Compliance** — None.
- **Accessibility** — Content is present in the a11y tree immediately (fade is visual only); screen
  readers are not forced to wait for the stagger; `aria-busy` toggles off when loaded; reduced-motion
  honored.
- **Scalability** — Primitive handles 1 to hundreds of items via the cap; virtualized lists reveal
  only on-screen items.
- **Reliability** — If data resolves, animation state must not strand a skeleton on screen; error path
  reveals the error state with the same crossfade.
- **Observability** — No new metrics required; may reuse perf telemetry to ensure no INP regression.
- **Maintainability** — One primitive per platform; screens adopt by wrapping their content region.
- **Internationalization** — Reveal translate respects RTL; no text.
- **Backward compatibility** — Screens not yet migrated keep the current hard swap; migration is
  incremental, dashboard first.

## 7. Acceptance Criteria

- **AC-1.** *Given* the mobile dashboard loading, *When* data resolves, *Then* the skeleton crossfades
  out and cards/stat tiles ease in staggered with the bubble curve — no simultaneous pop.
  (iOS & Android UI test / recorded QA.)
- **AC-2.** *Given* reduced motion is on, *When* the dashboard loads, *Then* content fades in ≤100ms
  with no stagger or movement.
- **AC-3.** *Given* a web data page using `<StaggerReveal>`, *When* it loads, *Then* Lighthouse
  reports CLS = 0 (skeleton reserved the space) and content reveals staggered.
- **AC-4.** *Given* a background refresh (pull-to-refresh) with unchanged data, *When* it completes,
  *Then* the screen does **not** re-run the entrance animation.
- **AC-5.** *Given* many items (e.g. 50-row roster), *When* it loads, *Then* the first ~8 stagger and
  the rest group-fade, keeping total choreography ≤ ~400ms and 60fps.
- **AC-6.** *Given* content is mid-fade, *When* the user taps a card, *Then* the tap registers
  immediately (no animation-blocked interactivity).

## 8. Data Model

- No database changes. Adds a per-region "has revealed" flag in component/view state to satisfy FR-5.
- No migration; no backfill.

## 9. API Surface

- No HTTP/WebSocket surface. Internal primitives:
  - Web: `<StaggerReveal index={i}>` and `useReveal({ ready })` in `clients/web/src/lib/motion` or
    `components/ui`.
  - iOS: `View.lxStaggeredReveal(index:appeared:)` in `Core/Design/LexturesMotion.swift`.
  - Android: `StaggeredReveal` composable + `Modifier.lxReveal(index, appeared)` in
    `core/design/LexturesMotion.kt`.

## 10. UI / UX

- **Modified surfaces (highest-traffic first)** — student/instructor/parent dashboards
  ([`DashboardView.swift`](../../../clients/ios/Lextures/Features/Dashboard/DashboardView.swift),
  [`DashboardTab.kt`](../../../clients/android/app/src/main/kotlin/com/lextures/android/features/dashboard/DashboardTab.kt),
  [`clients/web/src/pages/lms/dashboard.tsx`](../../../clients/web/src/pages/lms/dashboard.tsx)),
  course lists/carousels, gradebooks/rosters, feeds/inbox, insights, catalog/marketplace, reading
  dashboards — anywhere a skeleton exists today
  ([`lms-content-skeletons.tsx`](../../../clients/web/src/components/ui/lms-content-skeletons.tsx),
  iOS `LMSSkeletonList`, Android's ~334 shimmer/placeholder sites).
- **Key flows** — (1) cold dashboard load; (2) navigate into a list-heavy screen; (3) pull-to-refresh
  (no re-stagger); (4) error resolution reveal.
- **Empty/loading/error/offline** — skeleton (loading) crossfades to content **or** to empty/error
  state using the same crossfade; offline cached content reveals without stagger if already seen.
- **Mobile/responsive** — mobile stagger tuned for smaller viewports; web reveals only above-the-fold
  + on-scroll for long pages.
- **Accessibility** — `aria-busy`/`redacted`/placeholder semantics clear on load; content readable
  immediately.
- **Copy & i18n** — none.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- **Web** — [`lms-content-skeletons.tsx`](../../../clients/web/src/components/ui/lms-content-skeletons.tsx),
  dashboard and list pages under `clients/web/src/pages/**`, data-fetching hooks that expose a
  `loading`/`ready` flag.
- **iOS** — `Features/Dashboard/*`, every view using `LMSSkeletonList`/`redacted`, the `model.loading`
  gates.
- **Android** — `features/dashboard/DashboardTab.kt`, the shimmer/placeholder sites, view-model
  loading flags.
- Consumes AN.1 tokens; coordinates with [AN.4](AN.4-lists-collections-motion.md) so post-load
  mutations don't double-animate.

## 13. Dependencies & Sequencing

- Must ship **after**: AN.1.
- Pairs with: [AN.2](AN.2-launch-navigation-transitions.md) (together they fix launch→landing).
- Must ship **before**: nothing.
- Shared infra: none beyond AN.1.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Stagger makes screens feel slower to fully appear | M | H | Cap at `staggerMax`, group-fade the rest, keep total ≤~400ms; measure perceived vs actual |
| Layout shift if skeleton size ≠ content size | M | M | Make skeletons space-accurate; CLS = 0 gate in CI (AC-3) |
| Re-stagger on every refresh feels glitchy | M | M | "Has revealed" flag (FR-5); only new items animate (AC-4) |
| Animation delays time-to-read on slow devices | L | H | Content laid out and interactive immediately; fade is cosmetic (FR-9/AC-6) |
| Virtualized lists re-trigger reveal on scroll | M | M | Track revealed keys; reveal once per item, not per viewport entry |

## 15. Rollout Plan

- **Feature flag** — `ff_motion_reveal` (default off → on after QA), applied per surface so the
  dashboard can pilot before wider rollout.
- **Sequencing** — land primitive behind flag → adopt on the three dashboards first → expand to
  lists/feeds/gradebooks → enable by default.
- **Dogfood** — internal cohort; watch INP/CLS and qualitative feedback.
- **GA criteria** — CLS = 0, no INP regression, reduced-motion verified, no re-stagger on refresh,
  dashboards signed off by design.
- **Rollback** — flip `ff_motion_reveal` off (reverts to current hard swap).

## 16. Test Plan

- **Unit** — reveal primitive computes correct per-index delay, caps at `staggerMax`, and returns
  no-stagger under reduced motion; "has revealed" prevents re-animation.
- **Integration** — iOS/Android: dashboard load reveals staggered; refresh does not re-stagger; error
  path crossfades.
- **End-to-end** — Playwright: web dashboard loads with CLS = 0, staggered reveal present, reduced-
  motion emulation yields plain fade; tap during fade registers.
- **Security** — n/a.
- **Accessibility** — content in a11y tree pre-fade; `aria-busy` clears; axe clean; SR reads content
  without waiting on stagger.
- **Performance / load** — 50-item load stays 60fps; INP/LCP not regressed; frame traces on low-end
  devices.
- **Manual exploratory** — slow-network throttling, rapid navigate-away during reveal, offline cached
  reveal.

## 17. Documentation & Training

- **Internal** — "Revealing loaded content" recipe per platform; how to make skeletons space-accurate;
  when to stagger vs plain-fade.
- **End-user** — none.
- **Runbook** — the `ff_motion_reveal` flag scope and kill-switch.

## 18. Open Questions

1. Ideal stagger step and cap by device class (mobile vs web) — tune during dogfood.
2. Should above-the-fold reveal on web be eager while below-the-fold reveals on scroll, or reveal the
   whole first viewport at once?
3. For cached/offline content the user has already seen, skip animation entirely?
4. Do we co-locate the "space-accurate skeleton" audit here or as a separate polish pass?

## 19. References

- Existing: [`clients/ios/Lextures/Features/Dashboard/DashboardView.swift`](../../../clients/ios/Lextures/Features/Dashboard/DashboardView.swift),
  [`clients/android/app/src/main/kotlin/com/lextures/android/features/dashboard/DashboardTab.kt`](../../../clients/android/app/src/main/kotlin/com/lextures/android/features/dashboard/DashboardTab.kt),
  [`clients/web/src/components/ui/lms-content-skeletons.tsx`](../../../clients/web/src/components/ui/lms-content-skeletons.tsx),
  [`clients/web/src/pages/lms/dashboard.tsx`](../../../clients/web/src/pages/lms/dashboard.tsx).
- Standards: WCAG 2.3.1; web CLS / INP (Core Web Vitals); Material 3 "Container transform / reveal".
- Related plans: [AN.1](AN.1-motion-foundation-tokens.md), [AN.2](AN.2-launch-navigation-transitions.md),
  [AN.4](AN.4-lists-collections-motion.md).
