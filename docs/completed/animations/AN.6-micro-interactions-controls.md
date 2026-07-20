# AN.6 — Micro-interactions & Controls

> Completed implementation plan. Source: [docs/plan/animations/README.md](../../plan/animations/README.md) (Motion & Animation Polish initiative).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AN.6 |
| **Section** | Motion & Animation Polish |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE — web `control-motion` + press/toggle/segmented/validation/loading CSS, `Button`/`FeatureToggleRow`/`SegmentedControl` adoption, iOS `Haptics` + `LXPressableButtonStyle`/`LXControlMotion`, Android `Haptics` + `lxPressable`, `ff_motion_controls` kill-switch (collapsed into motion master) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Design Systems / Frontend Platform + Mobile |
| **Depends on** | AN.1 |
| **Unblocks** | — |

---

## 1. Problem Statement

Individual controls are where the "bubble" feel lives or dies — they're touched constantly. Today
feedback is inconsistent: iOS's `AuthPrimaryButtonStyle` has a nice press scale
([`LexturesTheme.swift:211`](../../../clients/ios/Lextures/Core/Design/LexturesTheme.swift)) but most
other buttons, toggles, checkboxes, radio groups, tab bars, segmented controls, and inputs give no
motion feedback at all; web relies on `transition-*` colors but rarely on tactile scale; mobile has
no consistent haptics. Controls feel flat and it's not always obvious a tap registered. This story
makes every interactive control respond with a small, consistent, bubble-flavored reaction and adds
standardized haptics on mobile.

## 2. Goals

- A consistent press/tap reaction on all tappable controls (buttons, cards, list rows, chips, icon
  buttons): quick scale-down on press, bubble settle on release.
- Motion on state controls: toggle/switch thumb travel, checkbox/radio check-in, segmented/tab
  indicator that slides between options.
- Input affordances: focus ring/label motion, and a gentle validation "shake"/color pulse on error.
- Standardized, tasteful haptics on iOS/Android tied to key interactions (tap, toggle, success,
  error, selection).
- Everything degrades under reduced motion (color/opacity only, no scale/shake; haptics respect the
  system setting).

## 3. Non-Goals

- Overlay/menu open animations (AN.5) and celebratory/gamification feedback (AN.7).
- Redesigning control visuals, sizes, or the component API.
- Sound effects (haptics only on mobile).
- A full new component library — enhance the existing controls in place.

## 4. Personas & User Stories

- **As any user tapping a button**, I want it to "give" slightly and spring back so I feel the tap
  land.
- **As a student toggling a setting**, I want the switch thumb to glide and a light haptic to confirm.
- **As an instructor moving between tabs/segments**, I want the active indicator to slide so the
  change reads as continuous.
- **As a user submitting an invalid form**, I want the field to nudge/pulse so I notice the error
  without a jarring jump.
- **As a motion-sensitive or low-vision user**, I want feedback via color/opacity (and reduced/att
  haptics) rather than movement.

## 5. Functional Requirements

- **FR-1.** Tappable controls MUST scale down (~0.97) on press and settle back with the bubble curve on
  release; icon buttons and cards/rows included. Web uses `:active`/pointer state; mobile uses press
  state.
- **FR-2.** Toggle/switch MUST animate the thumb travel and track color together; checkbox/radio MUST
  animate the check/dot in (draw or scale), not snap.
- **FR-3.** Segmented controls, tab bars, and pill/underline tab indicators MUST slide the active
  indicator between positions using the AN.1 `standard`/bubble curve.
- **FR-4.** Text inputs MUST animate focus affordance (ring/border/label) smoothly; on validation
  error, MUST give a small shake or color pulse (single, ≤ `base` duration), not a layout jump.
- **FR-5.** Mobile controls MUST fire standardized haptics: light impact on tap of primary actions,
  selection feedback on toggles/segment changes, success/error notification haptics on submit
  outcomes — mapped to a small shared haptics helper, not scattered literals.
- **FR-6.** Loading buttons MUST transition into a spinner/disabled state smoothly (crossfade label↔
  spinner), and back, without width jump.
- **FR-7.** Under reduced motion, press reactions reduce to opacity/color change (no scale), validation
  uses color/icon only (no shake), and indicators cut/crossfade instead of sliding; haptics follow the
  OS haptics setting.
- **FR-8.** All control motion MUST be centralized in the shared button/control styles so every
  instance inherits it (web `button.tsx` + control components; iOS `ButtonStyle`/`ToggleStyle`;
  Android component defaults).
- **FR-9.** Motion MUST NOT delay the control's action — the effect runs alongside the handler, never
  gating it.

## 6. Non-Functional Requirements

- **Performance** — transform/opacity only; press reactions are GPU-cheap; no layout thrash on
  loading-button width; 60fps.
- **Security** — None.
- **Privacy & Compliance** — None.
- **Accessibility** — motion never replaces a text/color state (WCAG 1.4.1 not by motion alone);
  validation error remains programmatically associated (`aria-invalid`, message) regardless of shake;
  focus ring must remain visible (2.4.7); reduced motion + haptics settings honored.
- **Scalability** — centralized styles mean thousands of control instances inherit motion at no
  per-site cost.
- **Reliability** — rapid taps/toggles never leave a control stuck mid-scale; indicator always lands
  on the true active option.
- **Observability** — none required.
- **Maintainability** — one place per control type; no per-screen motion code.
- **Internationalization** — indicator slide direction respects RTL; validation shake is horizontal
  and mirror-safe; no text.
- **Backward compatibility** — visual states unchanged; motion is additive; components not yet
  centralized keep working.

## 7. Acceptance Criteria

- **AC-1.** *Given* any primary/secondary/icon button, *When* pressed and released, *Then* it scales
  down and springs back with the bubble curve.
- **AC-2.** *Given* a toggle, *When* switched, *Then* the thumb glides, the track color crossfades, and
  (mobile) a selection haptic fires.
- **AC-3.** *Given* a tab/segmented control, *When* I change selection, *Then* the active indicator
  slides to the new option.
- **AC-4.** *Given* an invalid form submit, *When* it fails, *Then* the offending field shakes/pulses
  once and remains `aria-invalid` with its message.
- **AC-5.** *Given* reduced motion, *When* I interact with any control, *Then* feedback is color/opacity
  only — no scale, slide, or shake — and error state is still conveyed by icon/text/color.
- **AC-6.** *Given* a loading button, *When* it enters loading, *Then* label↔spinner crossfade without
  the button changing width.

## 8. Data Model

- No database changes. Transient interaction state only.
- No migration; no backfill.

## 9. API Surface

- No HTTP/WebSocket surface. Internal:
  - Web: enhanced [`button.tsx`](../../../clients/web/src/components/ui/button.tsx) + shared toggle/
    checkbox/tabs/input styles; a `useHaptics()` no-op on web.
  - iOS: shared `ButtonStyle`/`ToggleStyle` (extend `AuthPrimaryButtonStyle`), a `Haptics` helper
    (Core/UI or Core/Design).
  - Android: component defaults (Material 3) tuned to AN.1 + a `Haptics`/`HapticFeedback` helper.

## 10. UI / UX

- **Modified surfaces** — all buttons ([`button.tsx`](../../../clients/web/src/components/ui/button.tsx),
  iOS `AuthPrimaryButtonStyle` and siblings, Android buttons), toggles/switches/checkboxes/radios,
  segmented & tab controls (side-nav links, settings tabs, quiz answer chips coordinate with AN.7),
  text inputs & form fields, icon buttons, chips/tags, tappable cards & list rows.
- **Key flows** — (1) tap a button; (2) toggle a setting; (3) switch tabs; (4) submit invalid form;
  (5) button loading state.
- **Empty/loading/error/offline** — loading buttons (FR-6); disabled controls show a static state
  with no press reaction.
- **Mobile/responsive** — haptics on mobile only; larger touch targets keep the same motion.
- **Accessibility** — visible focus, non-motion state signaling, `aria-invalid`, reduced-motion +
  haptics settings.
- **Copy & i18n** — none new.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- **Web** — `components/ui/button.tsx` and form/control components across `clients/web/src/**`;
  `index.css` focus/validation styles.
- **iOS** — `Core/Design/LexturesTheme.swift` button/toggle styles, a new `Haptics` helper, control
  sites across `Features/**`.
- **Android** — `core/design` control theming, a `Haptics` helper, Material 3 component usage.
- Consumes AN.1 tokens; coordinates with AN.7 for quiz-answer control feedback.

## 13. Dependencies & Sequencing

- Must ship **after**: AN.1.
- Must ship **before**: nothing (but AN.7 reuses the press/haptic helpers).
- Shared infra: none beyond AN.1.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Haptics feel excessive/annoying | M | M | Conservative mapping; honor OS haptics setting; user setting to disable; taste review |
| Press scale interferes with scroll/drag on mobile | M | M | Cancel press reaction when a gesture becomes a scroll/drag |
| Loading-button crossfade causes width jump | M | M | Reserve width / fixed min-width; crossfade in place (FR-6) |
| Validation shake perceived as error-by-motion-only | L | M | Always pair with color+icon+message; reduced-motion drops shake (AC-5) |
| Centralizing control styles regresses an edge-case control | M | M | Incremental adoption per control type; visual regression tests |

## 15. Rollout Plan

- **Feature flag** — `ff_motion_controls` (default off → on after QA); haptics behind a user setting
  too.
- **Sequencing** — land shared button press first (broadest impact) → toggles/checkboxes → tab/segment
  indicators → inputs/validation → haptics → enable by default.
- **Dogfood** — internal; gather haptics taste feedback.
- **GA criteria** — no scroll/drag conflicts, no width jumps, reduced-motion + haptics settings
  verified, focus visibility intact.
- **Rollback** — flip `ff_motion_controls` off; haptics setting independent.

## 16. Test Plan

- **Unit** — button/control style resolves press vs reduced-motion variants; indicator computes target
  offset; validation shake fires once.
- **Integration** — iOS/Android press + haptic mapping; toggle thumb travel; tab indicator lands on
  active option.
- **End-to-end** — Playwright: button press scale, tab indicator slide, invalid-submit shake with
  `aria-invalid`, loading-button crossfade without width change, reduced-motion emulation → color-only.
- **Security** — n/a.
- **Accessibility** — focus visible, error state conveyed without motion, axe clean, haptics respect OS
  setting.
- **Performance / load** — press reactions 60fps; no layout thrash; frame traces on low-end.
- **Manual exploratory** — rapid tap/toggle, press-then-scroll, RTL tab indicator, haptics on device.

## 17. Documentation & Training

- **Internal** — control-motion + haptics guidelines (the mapping table); how to add a new control
  that inherits motion.
- **End-user** — help-center note for the haptics/reduce-motion settings if newly surfaced.
- **Runbook** — `ff_motion_controls` scope and kill-switch.

## 18. Open Questions

1. Exact haptics mapping (which interactions, which intensity) — needs a taste pass on device.
2. Should tappable cards/rows get the same press scale as buttons, or a subtler version?
3. Do we add a dedicated "reduce haptics" setting or piggyback on OS + reduce-motion?

## 19. References

- Existing: [`clients/web/src/components/ui/button.tsx`](../../../clients/web/src/components/ui/button.tsx),
  [`clients/ios/Lextures/Core/Design/LexturesTheme.swift`](../../../clients/ios/Lextures/Core/Design/LexturesTheme.swift)
  (`AuthPrimaryButtonStyle`, focus animation at `:255`), Android `core/design` control theming,
  form/validation styles in [`index.css`](../../../clients/web/src/index.css).
- Standards: WCAG 1.4.1 (Use of Color), 2.4.7 (Focus Visible), 4.1.3 (Status Messages), 2.3.1;
  Apple HIG "Feedback / Haptics"; Material 3 "State layers / Haptics".
- Related plans: [AN.1](AN.1-motion-foundation-tokens.md), [AN.5](AN.5-overlays-surfaces.md),
  [AN.7](../../plan/animations/AN.7-delight-progress-moments.md).
