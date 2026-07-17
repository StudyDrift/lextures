# AN.2 — App Launch & Navigation Transitions

> Completed implementation plan. Source: [docs/plan/animations/README.md](../../plan/animations/README.md) (Motion & Animation Polish initiative).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AN.2 |
| **Section** | Motion & Animation Polish |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE — splash handoff, route/section transitions, tab motion, `ff_motion_navigation` kill-switch on web / iOS / Android |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Frontend Platform (web) + Mobile (iOS/Android) |
| **Depends on** | AN.1 |
| **Unblocks** | — |

---

## 1. Problem Statement

The splash screen eases in cleanly on iOS and Android, then the app **hard-cuts** to the dashboard —
the exact whiplash the initiative was raised to fix. Beyond launch, navigation everywhere is
instantaneous and un-choreographed: web route changes swap under `Suspense` with no transition
([`app.tsx`](../../../clients/web/src/app.tsx)), iOS `NavigationStack` and the tab bar use system
defaults, and Android navigation composes without enter/exit transitions. Users lose spatial
continuity (where did I come from, where am I going), and the product feels like a series of
disconnected screens rather than one moving surface.

## 2. Goals

- Replace the splash→app hard cut with a continuous handoff (shared brand element and/or crossfade)
  on iOS and Android, and the web boot/first-paint equivalent.
- Give route/screen changes a consistent directional transition (forward = advance, back = retreat)
  driven by AN.1 tokens.
- Animate tab switching and top-level section changes so context shifts read as movement, not blinks.
- Ensure every transition respects reduced motion (crossfade/instant) and never delays interactivity.

## 3. Non-Goals

- Per-card/content entrance once a screen is shown — that is [AN.3](AN.3-load-choreography.md).
- Modal/sheet/drawer presentation — that is [AN.5](../../plan/animations/AN.5-overlays-surfaces.md).
- Redesigning navigation structure, IA, or the splash artwork.
- Complex hero/shared-element morphs between arbitrary screens (a single splash→home shared element
  is in scope; a general shared-element framework is a follow-up).

## 4. Personas & User Stories

- **As any user launching the app**, I want the splash to *become* the home screen so that startup
  feels like one continuous motion instead of a flash-then-cut.
- **As a student navigating course → assignment → submission**, I want forward moves to slide/scale
  forward and back to reverse so that I keep my bearings.
- **As an instructor switching tabs** (Teach / Home / More), I want the switch to animate so the app
  feels responsive and alive.
- **As a motion-sensitive user**, I want these transitions to become simple crossfades so navigation
  never induces discomfort.
- **As a self-learner on web**, I want route changes to feel smooth without adding perceptible load
  latency.

## 5. Functional Requirements

- **FR-1.** iOS and Android MUST transition from splash to the first authenticated screen with a
  continuous motion: the brand mark/logo settles into place (or crossfades) rather than the screen
  being replaced instantly.
- **FR-2.** The splash MUST NOT extend total cold-start time to achieve the effect; the handoff runs
  during/over existing boot work and is capped (e.g. ≤ `deliberate` 480ms), skippable once content
  is ready.
- **FR-3.** Web route changes SHOULD use a shared transition (View Transitions API where supported,
  falling back to an AN.1 crossfade wrapper around the `Suspense` boundary in
  [`app.tsx`](../../../clients/web/src/app.tsx)); the `RouteFallback` skeleton MUST crossfade into
  loaded content, not pop.
- **FR-4.** iOS `NavigationStack` push/pop and Android nav MUST use directional transitions (forward
  advances, back reverses) using AN.1 `standard`/`exit` curves; the system back-gesture on both must
  remain interactive and interruptible.
- **FR-5.** Tab switches (iOS `MainTabView`, Android bottom nav, web section changes) MUST animate
  the content swap (directional slide or crossfade by tab distance) and the active-tab indicator.
- **FR-6.** All navigation transitions MUST resolve to a ≤100ms crossfade (or instant) under reduced
  motion, and MUST NOT block the first tap on the destination.
- **FR-7.** Transitions MUST respect RTL — "forward" travels toward the inline-end edge, mirrored for
  `ar`.
- **FR-8.** Deep links / programmatic navigation (notifications, `navigationDestination(isPresented:)`)
  MUST animate consistently with user-initiated navigation, or intentionally skip animation when
  arriving cold.

## 6. Non-Functional Requirements

- **Performance** — Transitions animate transform/opacity only; no jank on target low-end devices;
  web transitions must not increase INP or cause layout shift (CLS = 0 from the transition).
  Cold-start time budget unchanged (FR-2).
- **Security** — None.
- **Privacy & Compliance** — None.
- **Accessibility** — Focus moves to the destination's logical start after transition; screen-reader
  announces the new screen once (not mid-animation); reduced-motion honored; no >3Hz flashing.
- **Scalability** — Transition logic centralized so new routes/screens inherit it without bespoke code.
- **Reliability** — Interrupting a transition (rapid back-forth, gesture cancel) always lands on a
  valid, fully-rendered screen with correct focus.
- **Observability** — Optionally record transition frame drops in dev builds; no PII.
- **Maintainability** — One navigation-transition wrapper per platform; feature screens opt in via
  convention, not per-screen code.
- **Internationalization** — Directionality respects locale (RTL); no text in transitions.
- **Backward compatibility** — Screens that opt out (e.g. full-screen player) keep working; default
  is the shared transition.

## 7. Acceptance Criteria

- **AC-1.** *Given* app cold start on iOS/Android, *When* the first screen is ready, *Then* the logo
  transitions into the home layout continuously with no visible hard cut, *And* total time-to-interactive
  is not increased beyond the measured baseline.
- **AC-2.** *Given* a push from course → assignment on iOS/Android, *When* I tap, *Then* the incoming
  screen advances in and the outgoing retreats; *When* I use the back gesture, *Then* it reverses and
  tracks my finger.
- **AC-3.** *Given* a web route change, *When* it occurs, *Then* the outgoing view crossfades/slides
  to the incoming view using AN.1 tokens, *And* Lighthouse CLS remains 0.
- **AC-4.** *Given* reduced motion is on (any client), *When* I navigate or launch, *Then* all
  transitions become ≤100ms crossfades or instant swaps.
- **AC-5.** *Given* a tab switch, *When* I tap another tab, *Then* content and the active indicator
  animate, *And* rapidly tapping between tabs never leaves a half-rendered or misfocused screen.
- **AC-6.** *Given* RTL locale (`ar`), *When* I navigate forward, *Then* motion direction is mirrored.

## 8. Data Model

- No database changes. State touched: navigation state only (router location on web; `NavigationStack`
  path on iOS; nav back-stack on Android; `shell` active-section state in
  [`app-shell.tsx`](../../../clients/web/src/components/layout/app-shell.tsx) /
  [`MainTabView.swift`](../../../clients/ios/Lextures/Features/Home/MainTabView.swift)).
- No migration; no backfill.

## 9. API Surface

- No HTTP/WebSocket surface. Internal APIs:
  - Web: a `<RouteTransition>` wrapper around the `Suspense`/`Routes` tree in `app.tsx`, plus a
    `useViewTransition()` helper.
  - iOS: a `navigationTransition`/custom `AnyTransition` helper + `MainTabView` content transition.
  - Android: `enterTransition`/`exitTransition`/`popEnterTransition` on the nav host in
    [`RootScreen.kt`](../../../clients/android/app/src/main/kotlin/com/lextures/android/app/RootScreen.kt).

## 10. UI / UX

- **Modified surfaces** — splash→home handoff ([`SplashView.swift`](../../../clients/ios/Lextures/Features/Splash/SplashView.swift),
  [`SplashScreen.kt`](../../../clients/android/app/src/main/kotlin/com/lextures/android/features/splash/SplashScreen.kt),
  web boot/`RouteFallback`); route/screen transitions; tab bar
  ([`MainTabView.swift`](../../../clients/ios/Lextures/Features/Home/MainTabView.swift), Android
  bottom nav, web [`side-nav.tsx`](../../../clients/web/src/components/layout/side-nav.tsx) section
  changes).
- **Key flows** — (1) cold launch → home; (2) forward drill-in and back; (3) tab/section switch;
  (4) deep-link arrival.
- **Empty/loading/error/offline** — a route that resolves to an error/empty state still transitions
  in; the transition wraps whatever the destination renders (skeleton handoff belongs to AN.3).
- **Mobile/responsive** — mobile uses directional push/pop; web wide layouts may prefer crossfade
  over slide to avoid large-canvas travel.
- **Accessibility** — post-transition focus target defined per destination; single SR announcement.
- **Copy & i18n** — none.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- **Web** — [`app.tsx`](../../../clients/web/src/app.tsx),
  [`route-fallback.tsx`](../../../clients/web/src/components/route-fallback.tsx),
  [`app-shell.tsx`](../../../clients/web/src/components/layout/app-shell.tsx), `react-router-dom` v7.
- **iOS** — `App/RootView.swift`, `Features/Splash/SplashView.swift`, `Features/Home/MainTabView.swift`,
  all `Features/**` `NavigationStack` roots.
- **Android** — `app/RootScreen.kt`, `features/splash/SplashScreen.kt`, the nav host & bottom nav.
- **Desktop** — inherits web; optionally add a Tauri window fade-in on show.
- Consumes AN.1 tokens throughout.

## 13. Dependencies & Sequencing

- Must ship **after**: AN.1.
- Must ship **before**: nothing, but pairs naturally with AN.3 (launch→landing is fully polished
  only when both land).
- Shared infra: none beyond AN.1.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Splash handoff adds perceived startup latency | M | H | Run the handoff over existing boot work; cap duration; skip-to-ready when content arrives before the cap (FR-2) |
| View Transitions API uneven browser support | M | M | Progressive enhancement — crossfade wrapper fallback; feature-detect |
| Back-gesture interruption leaves broken state on mobile | M | H | Use platform-native interactive transitions; test rapid cancel/repeat (AC-5) |
| Directional transitions disorient when navigation isn't hierarchical | L | M | Use crossfade (not slide) for lateral/tab moves; reserve slide for true push/pop |
| Focus lost after transition (a11y regression) | M | H | Explicit post-transition focus target + SR test per destination |

## 15. Rollout Plan

- **Feature flag** — `ff_motion_navigation` (default off → on after QA) so transitions can be
  disabled instantly if a regression appears; reduced-motion path always available.
- **Sequencing** — land helpers behind flag → adopt on splash+home first (highest impact) → expand to
  drill-in flows → tabs → enable by default.
- **Dogfood** — internal cohort on the flag; watch crash-free rate and startup metrics.
- **GA criteria** — no startup-time regression, no a11y focus regressions, reduced-motion verified,
  crash-free rate steady.
- **Rollback** — flip `ff_motion_navigation` off.

## 16. Test Plan

- **Unit** — transition-wrapper selects correct direction (forward/back/lateral) given nav intent;
  reduced-motion resolves to crossfade.
- **Integration** — iOS/Android instrumentation: push/pop/back-gesture land on correct screen with
  correct focus; tab rapid-switch stability.
- **End-to-end** — Playwright: route change produces no CLS, correct focus target, reduced-motion
  emulation yields crossfade.
- **Security** — n/a.
- **Accessibility** — VoiceOver/TalkBack: one announcement per navigation; focus correct; axe clean.
- **Performance / load** — cold-start timing before/after (no regression); frame traces of push/pop
  and route change on low-end devices; Lighthouse CLS = 0.
- **Manual exploratory** — rapid navigation, gesture cancel, deep-link arrival, RTL locale.

## 17. Documentation & Training

- **Internal** — "How navigation transitions work" per platform; how to opt a screen out; how to set
  a destination's focus target.
- **End-user** — none (behavioral polish); reduce-motion setting already documented.
- **Runbook** — the kill-switch flag and what it disables.

## 18. Open Questions

1. Do we invest in a true shared-element splash→home logo morph, or a crossfade with a settling logo?
   (Design + effort trade-off.)
2. Adopt the View Transitions API on web now, or a lighter crossfade wrapper until support broadens?
3. Should lateral tab switches slide (directional) or always crossfade to avoid disorientation?
4. Which screens explicitly opt out (full-screen video player, live quiz play, board present mode)?

## 19. References

- Existing: [`clients/web/src/app.tsx`](../../../clients/web/src/app.tsx),
  [`clients/web/src/components/route-fallback.tsx`](../../../clients/web/src/components/route-fallback.tsx),
  [`clients/ios/Lextures/Features/Splash/SplashView.swift`](../../../clients/ios/Lextures/Features/Splash/SplashView.swift),
  [`clients/ios/Lextures/Features/Home/MainTabView.swift`](../../../clients/ios/Lextures/Features/Home/MainTabView.swift),
  [`clients/android/app/src/main/kotlin/com/lextures/android/app/RootScreen.kt`](../../../clients/android/app/src/main/kotlin/com/lextures/android/app/RootScreen.kt),
  [`clients/android/app/src/main/kotlin/com/lextures/android/features/splash/SplashScreen.kt`](../../../clients/android/app/src/main/kotlin/com/lextures/android/features/splash/SplashScreen.kt).
- Standards: MDN View Transitions API; Apple HIG "Navigation & Motion"; Material 3
  "Navigation transitions"; WCAG 2.3.1 / 2.4.3 (Focus Order).
- Related plans: [AN.1](AN.1-motion-foundation-tokens.md), [AN.3](AN.3-load-choreography.md),
  [AN.5](../../plan/animations/AN.5-overlays-surfaces.md).
