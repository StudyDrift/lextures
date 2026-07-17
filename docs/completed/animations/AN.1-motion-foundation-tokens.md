# AN.1 — Motion Foundation & Shared Tokens

> Completed implementation plan. Source: [docs/plan/animations/README.md](../../plan/animations/README.md) (Motion & Animation Polish initiative).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AN.1 |
| **Section** | Motion & Animation Polish |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE — shared motion tokens, bubble spring, reduced-motion helpers, and reference adoptions on web / iOS / Android |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Design Systems / Frontend Platform |
| **Depends on** | — (this is the keystone) |
| **Unblocks** | AN.2, AN.3, AN.4, AN.5, AN.6, AN.7 |

---

## 1. Problem Statement

Every client re-implements motion locally: web hand-rolls one-off `@keyframes` in `index.css`,
iOS scatters raw `withAnimation(.easeOut(duration: 0.6))` literals, and Android hard-codes
`tween(450)` per call site — while Android has **no** reduced-motion handling at all. There is no
shared vocabulary of curves, durations, or stagger values, so nothing feels like it came from the
same product, and there is no single place to tune "the Lextures feel." This story creates that
foundation: a small, opinionated motion-token layer in each client's design system (curves,
durations, distances, springs, stagger, and a unified reduced-motion signal), plus the "bubble"
spring that defines the brand's signature. Without it, every other AN story would re-invent and
re-diverge.

## 2. Goals

- Define **one** motion specification (curves, durations, distances, stagger, springs) and express
  it as real, importable tokens in web, iOS, and Android design layers.
- Ship the signature **"bubble" spring** (slow-in → fast → small settle) as a first-class token on
  each platform.
- Provide **one** reduced-motion source of truth per client (hook/utility/environment value) that
  unifies OS `prefers-reduced-motion` with the existing in-app override, replacing ad-hoc checks.
- Establish an enforced **performance budget** (transform/opacity only, 60fps, no interactivity
  regression) and a lint/check to keep raw motion literals out of feature code.
- Make adoption cheap: helpers/modifiers so a feature author writes `motion.bubble` / `.lxBubbleIn()`
  / `Modifier.lxBubbleIn()` rather than raw specs.

## 3. Non-Goals

- Animating any specific surface — that is AN.2–AN.7. This story delivers *only* the shared layer
  plus a reference implementation on one screen per platform to prove the tokens.
- A third-party animation dependency on web (no `framer-motion`); we generate spring easings and use
  the Web Animations API / CSS + Tailwind v4 tokens to keep the bundle lean.
- Redesigning visuals, color, type, or layout. Motion only.
- Lottie/after-effects pipelines or video. (May be revisited in AN.7 for celebration assets.)

## 4. Personas & User Stories

- **As a feature engineer**, I want to import `bubble`/`base`/`stagger` tokens so that I animate a
  new surface in one line and it matches the rest of the app.
- **As a design-systems owner**, I want to tune the app's motion feel in one file per platform so
  that a curve change propagates everywhere instead of requiring a codebase sweep.
- **As a student/instructor/admin using any client**, I want motion to feel identical in spirit on
  web, iOS, and Android so that the product feels like one brand.
- **As a motion-sensitive user (any role)**, I want a single reliable reduced-motion switch so that
  turning it on calms *every* animation, not just the few that happened to check.
- **As a self-learner on a low-end device**, I want animations that never drop frames or delay a tap
  so that polish never costs responsiveness.

## 5. Functional Requirements

- **FR-1.** The system MUST define a shared **duration scale**: `instant=100ms`, `fast=150ms`,
  `base=220ms`, `slow=320ms`, `deliberate=480ms`, expressed as tokens on every client.
- **FR-2.** The system MUST define **easing curves**: `standard` = `cubic-bezier(0.2,0,0,1)`
  (emphasized decelerate), `exit` = `cubic-bezier(0.3,0,1,1)`, `emphasized` = `cubic-bezier(0.2,0,0,1)`
  with the `base`+ durations, and `bubble` (the signature overshoot spring).
- **FR-3.** The **bubble** spring MUST be defined as: web — a `linear()` easing generated from a
  spring solver (response ≈ 0.5s, damping ≈ 0.72) exposed as `--ease-bubble`; iOS —
  `Animation.lxBubble = .spring(response: 0.5, dampingFraction: 0.72)`; Android —
  `LexturesMotion.bubble = spring(dampingRatio = 0.72f, stiffness = Spring.StiffnessMediumLow)`.
  All three MUST feel perceptually equivalent (validated in §7).
- **FR-4.** The system MUST define **motion distance/scale tokens**: enter-translate `= 12px` (`8dp`
  on mobile), enter-scale-from `= 0.97`, press-scale `= 0.97`.
- **FR-5.** The system MUST define **stagger tokens**: `staggerStep = 40ms`, `staggerMax = 8` items,
  after which remaining items fade as one group.
- **FR-6.** Each client MUST expose **one** reduced-motion signal: web `usePrefersReducedMotion()`
  hook (OS query OR `html.reduced-motion` class); iOS an `@Environment`-driven `lxReduceMotion`
  helper wrapping `accessibilityReduceMotion` + app setting; Android a `LocalReduceMotion`
  `CompositionLocal` (OS `Settings.Global.ANIMATOR_DURATION_SCALE == 0` + app setting).
- **FR-7.** When reduced motion is active, motion helpers MUST resolve to an **opacity-only ≤100ms**
  or instant variant automatically — the caller MUST NOT need a branch.
- **FR-8.** The system SHOULD provide convenience helpers: web `motion.enter`/`motion.bubbleIn`
  (returns className or WAAPI keyframes+options); iOS `View.lxBubbleIn(_:)` / `.lxEnter(_:)`
  modifiers; Android `Modifier.lxBubbleIn()` / `lxEnter()`.
- **FR-9.** The system MUST enforce that motion uses only compositable properties; a CI check
  SHOULD flag animating `width`/`height`/`top`/`left`/`margin` in changed CSS and raw duration
  literals in feature code (extend `scripts/check-interface-polish.mjs`).
- **FR-10.** The tokens MUST be theme- and UI-mode aware where relevant (e.g. K-2 / elementary UI
  modes MAY scale durations up slightly; high-contrast/reduced-motion mode forces the reduced path).

## 6. Non-Functional Requirements

- **Performance** — All tokenized motion animates `transform`/`opacity`/`filter` only. Web: no
  main-thread layout thrash; helpers prefer WAAPI/CSS over JS rAF loops. Target 60fps on iPhone SE
  (2nd gen), a Pixel 6a, and a mid-tier Chromebook. No animation may delay first input or push LCP.
- **Security** — None (client-only, no data).
- **Privacy & Compliance** — Reduced-motion state is a local device/user preference; not
  transmitted or logged with PII.
- **Accessibility** — WCAG 2.3.3 (Animation from Interactions, AAA — targeted) and 2.2.2
  (Pause/Stop/Hide) informed; hard requirement is honoring reduced motion (2.3.1 no >3Hz flashing).
- **Scalability** — Token layer is compile-time/static; zero runtime cost beyond reading one media
  query / environment value.
- **Reliability** — Helpers must always leave the element in the correct final state even if the
  animation is interrupted (cancel → jump to end, never stick mid-transition).
- **Observability** — Add a dev-only warning when a motion helper receives a non-token duration.
  Optionally count reduced-motion activation in existing telemetry (see
  `memory/observability-telemetry-17-7.md`), no PII.
- **Maintainability** — One file per platform owns the tokens; feature code imports, never inlines.
- **Internationalization** — Motion direction MUST respect RTL (enter-from-inline-start flips for
  `ar`); durations are locale-independent.
- **Backward compatibility** — Existing `@keyframes`/`withAnimation` literals keep working; migrate
  opportunistically. No breaking change to shipped screens in this story.

## 7. Acceptance Criteria

- **AC-1.** *Given* a web component using `motion.bubbleIn()`, *When* it mounts, *Then* it eases in
  with the shared overshoot curve and, *When* `prefers-reduced-motion: reduce` is set, *Then* it
  instead fades in over ≤100ms with no transform. (Unit + jsdom media-query test.)
- **AC-2.** *Given* the iOS `.lxBubbleIn()` modifier, *When* `accessibilityReduceMotion` is on,
  *Then* the transition uses opacity only and no spring. (SwiftUI snapshot/unit via environment
  override.)
- **AC-3.** *Given* Android with `ANIMATOR_DURATION_SCALE = 0` (or app reduce-motion on), *When* a
  composable uses `Modifier.lxEnter()`, *Then* `LocalReduceMotion` is true and the enter animation
  is skipped/opacity-only. (Compose UI test.)
- **AC-4.** *Given* the three `bubble` definitions, *When* compared side-by-side on a reference
  square (translate+scale in), *Then* peak velocity timing and settle overshoot match within a
  documented tolerance (design QA sign-off checklist attached to the PR).
- **AC-5.** *Given* CI, *When* a changed file animates a layout property or hard-codes a duration in
  feature code, *Then* the interface-polish check fails with a pointer to the token.
- **AC-6.** *Given* any tokenized enter animation interrupted mid-flight (e.g. fast re-render),
  *When* it cancels, *Then* the element lands on its final opacity/transform (no stuck state).

## 8. Data Model

- No database changes. "Data model" here = the token catalog, defined once per platform:
  - **Web** — `clients/web/src/lib/motion.ts` (TS constants + WAAPI keyframe factories) and CSS
    custom properties in [`index.css`](../../../clients/web/src/index.css) (`--ease-standard`,
    `--ease-exit`, `--ease-bubble`, `--dur-fast|base|slow|deliberate`, `--stagger-step`). Tailwind
    v4 `@theme` entries expose `animate-*`/`ease-*`/`duration-*` utilities.
  - **iOS** — `clients/ios/Lextures/Core/Design/LexturesMotion.swift` (`enum LexturesMotion`
    with `Animation` statics + `View` modifier extensions).
  - **Android** — `clients/android/app/src/main/kotlin/com/lextures/android/core/design/LexturesMotion.kt`
    (`object LexturesMotion` with `AnimationSpec` vals + `Modifier` extensions + `LocalReduceMotion`).
- No migration; no backfill.

## 9. API Surface

- No HTTP/WebSocket surface. The "API" is the developer-facing token/helper API:
  - Web: `import { motion, durations, easings, useReducedMotion } from '@/lib/motion'`.
  - iOS: `LexturesMotion.bubble`, `.lxBubbleIn()`, `@Environment(\.lxReduceMotion)`.
  - Android: `LexturesMotion.bubble`, `Modifier.lxBubbleIn()`, `LocalReduceMotion.current`.
- No rate limits. No OpenAPI change.

## 10. UI / UX

- **New shared assets, not new screens.** One reference adoption per platform proves the layer:
  - Web: convert the `sidenav-item-in` / `tooltip-in` keyframes in `index.css` to consume the new
    tokens (behavior identical, values centralized).
  - iOS: refactor `AuthPrimaryButtonStyle` press animation (`LexturesTheme.swift:212`) to
    `LexturesMotion` and `SplashView` to `.lxEnter`.
  - Android: refactor `SplashScreen.kt`'s `tween(450)` to `LexturesMotion` tokens.
- **States** — helpers define enter, exit, and reduced-motion variants; loading/error/offline states
  are unaffected here (their motion is AN.3).
- **Mobile/responsive** — mobile distances use `dp`; K-2/elementary UI modes may scale durations.
- **Accessibility annotations** — reduced-motion path documented next to each helper; never animate
  focus outlines away.
- **Copy & i18n** — none (no user-facing strings) except a Settings label if AN.1 also surfaces the
  in-app reduce-motion toggle where one doesn't yet exist (web already has it; add read-through on
  iOS/Android if missing).

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- **Web** — [`clients/web/src/index.css`](../../../clients/web/src/index.css), Tailwind v4 theme
  config, `scripts/check-interface-polish.mjs`; consumers across `clients/web/src/components/**`.
- **iOS** — `clients/ios/Lextures/Core/Design/` (new `LexturesMotion.swift`), referenced by
  `LexturesTheme.swift`, `SplashView.swift`, and every Feature module thereafter.
- **Android** — `clients/android/app/src/main/kotlin/com/lextures/android/core/design/`
  (new `LexturesMotion.kt`), referenced by `SplashScreen.kt` and feature composables.
- **Desktop** — inherits the web layer via the Tauri bundle; no separate work.
- No webhooks/events.

## 13. Dependencies & Sequencing

- Must ship **before**: AN.2–AN.7 (all consume these tokens).
- Must ship **after**: nothing.
- Shared infra needed: none beyond the existing design-system files and CI check script.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Web spring via `linear()` easing has poor support on old browsers | L | M | Feature-detect; fall back to `--ease-standard` cubic-bezier when `linear()` unsupported |
| Three platforms drift and "bubble" stops matching | M | M | Single documented spec (response/damping) + AC-4 side-by-side QA gate on every change |
| Over-tokenization slows feature authors | L | M | Ship ergonomic one-line helpers; document the 5 durations + 2 curves people actually need |
| CI check produces false positives, annoys teams | M | L | Scope the check to changed lines; allowlist legitimate cases; warn before failing for a grace period |
| Android reduced-motion signal unreliable across OEMs | M | M | Combine `ANIMATOR_DURATION_SCALE==0` with an explicit in-app setting; default app setting wins |

## 15. Rollout Plan

- **Feature flag** — none required (adding tokens is inert until consumed). The one-screen reference
  adoptions ship behind normal review.
- **Sequencing** — land token files → land reference adoptions → land CI check in warn-only mode →
  flip CI check to failing after one release cycle.
- **Dogfood** — internal build; design QA runs the AC-4 side-by-side comparison.
- **GA criteria** — tokens documented, reference adoptions merged, reduced-motion verified on all
  three clients, CI check green.
- **Rollback** — revert token files; reference adoptions are behaviorally identical to prior code so
  low blast radius.

## 16. Test Plan

- **Unit** — web `motion.ts` returns reduced vs full variants per media query (jsdom `matchMedia`
  mock); spring `linear()` generator output snapshotted.
- **Integration** — iOS environment-override tests for `lxReduceMotion`; Android Compose test for
  `LocalReduceMotion` under `ANIMATOR_DURATION_SCALE=0`.
- **End-to-end** — Playwright: reference web component fades (not slides) with reduced motion forced
  via emulation; no layout shift recorded.
- **Security** — n/a.
- **Accessibility** — axe run unaffected; manual VoiceOver/TalkBack confirm focus is never lost
  during the reference transitions; reduced-motion OS toggle verified on device.
- **Performance / load** — DevTools/Instruments/Perfetto trace of the reference adoptions shows only
  compositor work; 60fps; no dropped frames on target low-end devices; Lighthouse unchanged.
- **Manual exploratory** — QA checklist toggling reduced motion mid-animation to confirm correct
  final state (AC-6).

## 17. Documentation & Training

- **Internal** — a "Motion" page in the design-system docs: the token catalog, the 2-curve/5-duration
  cheat-sheet, the bubble spec, and "how to animate a new surface" recipes per platform.
- **API reference** — inline doc comments on every token/helper.
- **Runbook** — how to tune the feel (which value maps to which perceived effect).
- No end-user docs (except a one-line help-center note if a new reduce-motion toggle is surfaced).

## 18. Open Questions

1. Do K-2 / elementary UI modes want *more* bounce (playful) or the same restraint? (Design call.)
2. Should the in-app reduce-motion setting be unified into one cross-platform account preference, or
   stay per-device? (Product + platform.)
3. Web: ship the spring as a precomputed `linear()` string, or generate at build time from a solver
   for tunability? (Frontend platform.)
4. Do we adopt View Transitions API on web now (AN.2) or keep AN.1 purely token-level? (Sequencing.)

## 19. References

- Existing: [`clients/web/src/index.css`](../../../clients/web/src/index.css),
  [`clients/web/src/components/layout/notifications-drawer.tsx`](../../../clients/web/src/components/layout/notifications-drawer.tsx),
  [`clients/ios/Lextures/Core/Design/LexturesTheme.swift`](../../../clients/ios/Lextures/Core/Design/LexturesTheme.swift),
  [`clients/android/app/src/main/kotlin/com/lextures/android/features/splash/SplashScreen.kt`](../../../clients/android/app/src/main/kotlin/com/lextures/android/features/splash/SplashScreen.kt),
  `clients/web/scripts/check-interface-polish.mjs`.
- Standards: WCAG 2.1 §2.3.1 / §2.3.3 / §2.2.2; Material 3 motion (easing & duration guidance);
  Apple HIG "Motion"; MDN Web Animations API & `linear()` easing.
- Related plans: [AN.2](../../plan/animations/AN.2-launch-navigation-transitions.md)–[AN.7](../../plan/animations/AN.7-delight-progress-moments.md);
  web reduced-motion override ([12.7](../12-accessibility/12.7-high-contrast-reduced-motion.md)).

## Implementation notes (shipped)

| Platform | Tokens / helpers | Reference adoption |
|---|---|---|
| **Web** | `clients/web/src/lib/motion.ts`, CSS `--ease-*` / `--dur-*` / `--stagger-step` in `index.css`, `usePrefersReducedMotion()` | `sidenav-item-in` / `tooltip-in` + notifications drawer use tokens; `npm run interface-polish:check` flags layout animations |
| **iOS** | `LexturesMotion.swift`, `.lxBubbleIn` / `.lxEnter`, `@Environment(\.lxReduceMotion)` | `SplashView`, `AuthPrimaryButtonStyle` |
| **Android** | `LexturesMotion.kt`, `Modifier.lxBubbleIn` / `lxEnter`, `LocalReduceMotion` | `SplashScreen` |
