# Motion & Animation Polish (AN.1–AN.7)

Give Lextures a single, coherent **motion language** across every client — web, desktop
(Tauri), iOS (SwiftUI), and Android (Jetpack Compose) — so the product feels physical,
responsive, and hand-finished instead of assembled. The signature feel is a
**"bubble" spring**: motion that eases in slowly, accelerates through the middle, and
settles with a small, damped overshoot — never a linear slide, never an abrupt pop.

> Source: product-quality initiative (not a `MISSING_FEATURES.md` gap). This folder is the
> canonical spec for animation work; every story follows [`../_TEMPLATE.md`](../_TEMPLATE.md).

## Why now

The apps already ship a splash animation, skeleton loaders, and scattered transitions, but
motion is applied **inconsistently and locally**. The clearest symptom: the mobile splash
eases in cleanly, then the dashboard **hard-cuts** and its cards/data **pop in** the instant
the network resolves ([`SplashView.swift`](../../../clients/ios/Lextures/Features/Splash/SplashView.swift)
→ [`DashboardView.swift`](../../../clients/ios/Lextures/Features/Dashboard/DashboardView.swift),
and identically on Android [`SplashScreen.kt`](../../../clients/android/app/src/main/kotlin/com/lextures/android/features/splash/SplashScreen.kt)
→ [`DashboardTab.kt`](../../../clients/android/app/src/main/kotlin/com/lextures/android/features/dashboard/DashboardTab.kt)).
That whiplash between a polished intro and an unpolished landing reads as "cheap," and it
repeats on nearly every screen: content swaps, list mutations, sheets, and control feedback
all arrive without choreography.

## What exists today (baseline)

| Platform | Motion infra that exists | Gaps |
|---|---|---|
| **Web** (React 19 + Tailwind v4) | Hand-rolled CSS `@keyframes` in [`index.css`](../../../clients/web/src/index.css) (`sidenav-item-in`, `tooltip-in`, canvas-import); ~153 `animate-*` and ~675 `transition-*` class uses; `prefers-reduced-motion` honored in CSS + a few components ([`notifications-drawer.tsx`](../../../clients/web/src/components/layout/notifications-drawer.tsx), [`intro-completion-celebration.tsx`](../../../clients/web/src/components/intro-course/intro-completion-celebration.tsx)) | No shared motion tokens; no spring/"bubble" curve; no route transitions (`Suspense` hard-swaps in [`app.tsx`](../../../clients/web/src/app.tsx)); skeleton→content hard-cuts; no list mutation motion; each component re-implements reduced-motion detection |
| **Desktop** (Tauri) | Inherits the web bundle | Same as web + no window-level polish |
| **iOS** (SwiftUI) | ~29 `withAnimation`, 12 `.transition`, only 2 spring uses, 13 `reduceMotion` reads; `AuthPrimaryButtonStyle` has a press scale ([`LexturesTheme.swift:188`](../../../clients/ios/Lextures/Core/Design/LexturesTheme.swift)); skeletons via `redacted`/`LMSSkeletonList` | No `Motion` token layer in `Core/Design`; content swaps un-animated; near-zero springs; navigation uses default push/pop; reduced-motion checked ad hoc in 5 files only |
| **Android** (Compose) | ~11 `animate*AsState`/`AnimatedVisibility`, ~12 `spring/tween`, 334 shimmer/placeholder refs; splash uses `tween` | No `LexturesMotion` tokens; **no** `reduceMotion` handling anywhere; no `Crossfade`/`AnimatedContent` on content swaps; nav transitions default |

## Design language (applies to every story)

The full specification lives in **[AN.1](../../completed/animations/AN.1-motion-foundation-tokens.md)** (completed); the essentials:

- **Signature "bubble" spring** — a spring with a small overshoot for *entrances, expansions,
  and delight*. Per-platform equivalents:
  - Web: a generated `linear()` spring easing (Web Animations API / CSS) — token `--ease-bubble`.
  - iOS: `.spring(response: 0.5, dampingFraction: 0.72)` (≈ SwiftUI `.bouncy` tuned).
  - Android: `spring(dampingRatio = 0.72f, stiffness = Spring.StiffnessMediumLow)`.
- **Standard ease** — `cubic-bezier(0.2, 0, 0, 1)` (emphasized-decelerate) for *most* enter/move,
  `cubic-bezier(0.3, 0, 1, 1)` for exits. Used where a bounce would be noise (dismissals, exits).
- **Duration scale** — `instant 100ms · fast 150ms · base 220ms · slow 320ms · deliberate 480ms`.
  Springs are defined by response, not duration.
- **Stagger** — list/grid children enter offset by `30–50ms` (cap the cascade at ~8 items, then
  fade the remainder as a group) so choreography never feels slow.
- **Distance & scale** — entrances translate ≤ 12px and scale from 0.96–0.98; big travel reads as
  laggy.
- **Reduced-motion contract** — every animation MUST degrade to an **opacity-only ≤ 100ms** (or
  instant) form when the OS/app requests reduced motion. This is the single hard requirement that
  gates every story's acceptance.
- **Performance budget** — transform/opacity only (no layout-animating properties); 60fps on the
  low-end target devices; no animation may delay interactivity or Largest Contentful Paint.

## Stories

| ID | Plan | Severity | Surface breadth | One-line |
|---|---|---|---|---|
| **AN.1** | ~~Motion foundation & shared tokens~~ → [completed](../../completed/animations/AN.1-motion-foundation-tokens.md) | MAJOR | All platforms (design systems) | Done — shared tokens, bubble spring, reduced-motion helpers, reference adoptions |
| **AN.2** | ~~App launch & navigation transitions~~ → [completed](../../completed/animations/AN.2-launch-navigation-transitions.md) | MAJOR | Splash, routes, screens, tabs | Done — splash handoff, route/section/tab transitions, `ff_motion_navigation` |
| **AN.3** | ~~Load choreography: skeleton → content~~ → [completed](../../completed/animations/AN.3-load-choreography.md) | MAJOR | Every data-backed screen | Done — skeleton→content crossfade, staggered reveal, `ff_motion_reveal` |
| **AN.4** | ~~Lists, grids & collection motion~~ → [completed](../../completed/animations/AN.4-lists-collections-motion.md) | MINOR | Every list/feed/board | Done — insert/remove/reorder, drag lift, `ff_motion_lists` |
| **AN.5** | ~~Overlays & surfaces~~ → [completed](../../completed/animations/AN.5-overlays-surfaces.md) | MINOR | Modals, sheets, drawers, toasts, menus | Done — dialog/sheet/menu/toast/tooltip enter-exit, `ff_motion_overlays` |
| **AN.6** | [Micro-interactions & controls](AN.6-micro-interactions-controls.md) | MINOR | Every interactive control | Press/tap "bubble," toggle/checkbox/tab-indicator motion, input focus & validation, haptics |
| **AN.7** | [Delight & progress moments](AN.7-delight-progress-moments.md) | MINOR | Gamification, quizzes, progress, mastery | Celebrations, streaks/badges/XP, quiz answer feedback, progress-ring & mastery fills |

## Sequencing

```
AN.1 (foundation)  ──┬──▶ AN.2 launch & navigation
                     ├──▶ AN.3 load choreography   ◀── flagship; do right after AN.1
                     ├──▶ AN.4 lists & collections
                     ├──▶ AN.5 overlays & surfaces
                     ├──▶ AN.6 micro-interactions
                     └──▶ AN.7 delight & progress
```

**AN.1 ships first** — every other story consumes its tokens. AN.2 and AN.3 are the highest-impact
follow-ups (they fix the launch-to-landing whiplash the user called out). AN.4–AN.7 are
independent of one another and can land in any order or in parallel by surface owner.

## Cross-cutting requirements (inherited by every story)

- **Accessibility** — respect OS reduced-motion + the in-app reduced-motion override (web plan
  12.7 / `html.reduced-motion`); never trap focus during a transition; never remove content from
  the a11y tree mid-animation; no flashing > 3Hz (WCAG 2.3.1).
- **Performance** — animate `transform`/`opacity` only; GPU-compositable; measured against a
  per-platform frame-time budget; must not regress Lighthouse or app cold-start.
- **Consistency** — no story may hand-roll a curve or duration; all values come from AN.1 tokens.
- **Testability** — reduced-motion and "animation completes and leaves correct final state" are
  asserted in automated tests on every story.
