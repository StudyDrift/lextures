# AN.5 — Overlays & Surfaces (Modals, Sheets, Drawers, Toasts, Menus)

> Completed implementation plan. Source: [docs/plan/animations/README.md](../../plan/animations/README.md) (Motion & Animation Polish initiative).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AN.5 |
| **Section** | Motion & Animation Polish |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE — overlay state machine + `OverlaySurface` / `useOverlayPresence`, dialog/sheet/menu/toast/tooltip CSS, confirm + fullscreen shell + command palette + notifications drawer + sonner toaster adoption, iOS `.lxSheet`/`.lxDialog`, Android `lxSheet`/`lxDialog`, `ff_motion_overlays` kill-switch |
| **Estimated effort** | S (1w)–M (2–4w) |
| **Owner (proposed)** | Frontend Platform (web) + Mobile (iOS/Android) |
| **Depends on** | AN.1 |
| **Unblocks** | — |

---

## 1. Problem Statement

Transient surfaces are where motion matters most — they appear *over* content and need to explain
where they came from — yet most of ours just appear and disappear. The web notifications drawer is a
good exemplar (it slides and respects reduced motion,
[`notifications-drawer.tsx`](../../../clients/web/src/components/layout/notifications-drawer.tsx)),
and `tooltip-in` has a keyframe, but modals (`fullscreen-modal-shell`), popovers, dropdown menus,
context menus, toasts (`sonner` via [`lms-toaster.tsx`](../../../clients/web/src/components/lms-toaster.tsx)),
and the mobile equivalents (SwiftUI `.sheet`, Compose dialogs/bottom sheets) largely pop in and cut
out. This story gives every overlay a consistent, origin-aware enter/exit with a scrim fade.

## 2. Goals

- Give every overlay class a consistent enter/exit: dialogs scale+fade from center, sheets/drawers
  slide+settle from their edge, popovers/menus grow from their anchor, toasts slide+fade in a stack.
- Fade the scrim/backdrop in and out in sync with the overlay (never a hard black flash).
- Use the AN.1 bubble curve for entrances and the `exit` curve for dismissals; support interactive/
  swipe-to-dismiss on mobile sheets with a settling snap.
- Honor reduced motion (fade-only) and preserve focus-trap/return semantics through the animation.

## 3. Non-Goals

- Full-screen route/navigation transitions (AN.2) and in-content reveal (AN.3).
- Redesigning overlay layouts, sizes, or z-index architecture.
- New overlay components; this animates the ones that exist.
- Toast content/logic changes (keep `sonner`); only its motion is tuned.

## 4. Personas & User Stories

- **As any user opening a dialog**, I want it to scale up from center with the scrim fading so it
  feels like it emerged, not blinked on.
- **As a mobile user on a bottom sheet**, I want to drag it and have it settle or dismiss with a
  physical snap.
- **As an instructor getting a toast confirmation**, I want it to slide in and auto-dismiss smoothly
  so it's noticeable but not jarring.
- **As a user opening a dropdown/context menu**, I want it to grow from where I clicked so the origin
  is obvious.
- **As a motion-sensitive user**, I want overlays to simply fade so nothing flies across the screen.

## 5. Functional Requirements

- **FR-1.** Modals/dialogs MUST enter with scale (from ~0.97) + fade using the bubble curve and exit
  with fade + slight scale-down using the `exit` curve; the scrim MUST fade in/out in sync.
- **FR-2.** Sheets and drawers (side & bottom) MUST slide from their originating edge and settle; on
  mobile they MUST support interactive drag-to-dismiss that tracks the finger and snaps open/closed.
- **FR-3.** Popovers, dropdown menus, and context menus MUST grow/fade from their anchor point
  (transform-origin at the trigger) rather than appearing fully formed.
- **FR-4.** Toasts (`sonner`) MUST enter with slide+fade, stack/reflow with motion when multiple are
  present, and exit with fade; the existing stacking behavior is preserved.
- **FR-5.** Tooltips MUST use the shared tooltip enter (migrate the existing `tooltip-in` keyframe to
  AN.1 tokens) with a short delay-in and quick fade-out.
- **FR-6.** Every overlay MUST keep a working focus trap during and after the enter animation, and
  MUST return focus to the trigger on exit — animation must not break focus management.
- **FR-7.** Under reduced motion, all overlays MUST fade only (≤100ms), no slide/scale; scrim still
  fades.
- **FR-8.** Dismiss (Esc, scrim tap, swipe, back gesture) MUST animate out (not instant-remove) unless
  reduced motion, and MUST be interruptible (re-open mid-exit lands correctly).
- **FR-9.** Overlay enter/exit MUST use a portal/layer that doesn't cause layout shift in the page
  behind it (CLS = 0).

## 6. Non-Functional Requirements

- **Performance** — transform/opacity only; scrim uses opacity (not backdrop-filter animation on
  low-end); 60fps; no CLS on the underlying page.
- **Security** — None.
- **Privacy & Compliance** — None.
- **Accessibility** — WCAG 2.4.3 focus order & focus return; `aria-modal`/roles intact; SR announces
  the overlay once on open; Esc/back always work; reduced motion honored; toast is a polite live
  region.
- **Scalability** — one overlay-motion layer reused by all overlay types.
- **Reliability** — rapid open/close, double-dismiss, and swipe-cancel never strand a scrim or trap
  focus off-screen.
- **Observability** — none required.
- **Maintainability** — a single `<Overlay>`/presentation helper per platform; existing overlays adopt
  it.
- **Internationalization** — side sheets originate from the inline-start/-end edge per RTL; no text.
- **Backward compatibility** — un-migrated overlays keep working; incremental adoption; `sonner`
  stays.

## 7. Acceptance Criteria

- **AC-1.** *Given* a dialog opens, *When* it appears, *Then* it scales+fades in with the scrim fading
  in sync; *When* dismissed, *Then* both animate out and focus returns to the trigger.
- **AC-2.** *Given* a mobile bottom sheet, *When* I drag it down, *Then* it tracks my finger and
  snaps to dismissed/open with a settle; releasing past the threshold dismisses.
- **AC-3.** *Given* a dropdown/context menu, *When* it opens, *Then* it grows from the trigger's
  location (correct transform-origin).
- **AC-4.** *Given* a toast, *When* it appears and later auto-dismisses, *Then* it slides+fades in and
  fades out, and stacking multiple reflows with motion.
- **AC-5.** *Given* reduced motion, *When* any overlay opens/closes, *Then* it fades only with no
  slide/scale.
- **AC-6.** *Given* an overlay mid-exit, *When* it is re-opened, *Then* it returns to the open state
  without a stuck scrim or lost focus trap.

## 8. Data Model

- No database changes. Overlay open/close/animating state is local component/view state only.
- No migration; no backfill.

## 9. API Surface

- No HTTP/WebSocket surface. Internal:
  - Web: an overlay presentation wrapper (or enhancement of `fullscreen-modal-shell` + a shared
    `<Popover>`/`<Menu>` motion), `sonner` `<Toaster>` motion options in
    [`lms-toaster.tsx`](../../../clients/web/src/components/lms-toaster.tsx).
  - iOS: reusable `.lxSheet`/`.lxDialog` presentation modifiers with interactive dismiss.
  - Android: `ModalBottomSheet`/`Dialog` wrappers with AN.1 specs + predictive-back where available.

## 10. UI / UX

- **Modified surfaces** — `fullscreen-modal-shell.tsx`, confirm dialogs (`use-confirm`),
  [`notifications-drawer.tsx`](../../../clients/web/src/components/layout/notifications-drawer.tsx)
  (align to tokens), all tooltips
  ([`icon-action-tooltip.tsx`](../../../clients/web/src/components/ui/icon-action-tooltip.tsx),
  [`action-error-tooltip.tsx`](../../../clients/web/src/components/ui/action-error-tooltip.tsx),
  `side-nav-tooltip.tsx`), the `tooltip-in`/`sidenav-item-in` keyframes in
  [`index.css`](../../../clients/web/src/index.css), `sonner` toasts, dropdown/command-palette
  ([`side-nav-command-palette.tsx`](../../../clients/web/src/components/layout/side-nav-command-palette.tsx)),
  iOS `.sheet`/`.confirmationDialog`/`.popover` sites (e.g. `IntroCompletionCelebrationSheet`),
  Android dialogs/bottom sheets.
- **Key flows** — (1) open/close a dialog; (2) drag-dismiss a sheet; (3) open a menu/popover;
  (4) toast lifecycle; (5) tooltip hover.
- **Empty/loading/error/offline** — an overlay whose body is still loading shows AN.3 skeleton inside;
  error dialogs animate identically.
- **Mobile/responsive** — bottom sheets on mobile, centered dialogs on web/desktop; interactive
  dismiss mobile-only.
- **Accessibility** — focus trap, focus return, `aria-modal`, Esc/back, polite toast region.
- **Copy & i18n** — none new.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- **Web** — `components/ui/fullscreen-modal-shell.tsx`, `components/__tests__/use-confirm`,
  `components/lms-toaster.tsx` (`sonner`), tooltip components, `index.css` keyframes,
  command palette.
- **iOS** — `.sheet`/`.popover`/`.confirmationDialog` sites across `Features/**`, the
  `IntroCompletionCelebrationSheet` presentation.
- **Android** — `ModalBottomSheet`/`AlertDialog`/`Popup` sites, predictive-back integration.
- Consumes AN.1 tokens.

## 13. Dependencies & Sequencing

- Must ship **after**: AN.1.
- Must ship **before**: nothing.
- Shared infra: none beyond AN.1 and existing `sonner`.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Exit animation delays focus return / traps focus | M | H | Focus returns on exit-start, not exit-end; a11y tests (AC-1) |
| Interactive sheet dismiss conflicts with scroll inside sheet | M | M | Threshold + directional gesture disambiguation; platform-native where possible |
| `sonner` motion customization limited | L | M | Use its supported animation hooks; fall back to its defaults tuned to AN.1 timing |
| Scrim opacity animation janky with backdrop-filter | M | M | Animate opacity of a solid/blurred layer, not the filter itself, on low-end |
| Re-open mid-exit leaves stuck scrim | M | M | Idempotent open/close state machine; interruption tests (AC-6) |

## 15. Rollout Plan

- **Feature flag** — `ff_motion_overlays` (default off → on after QA), per overlay class.
- **Sequencing** — land overlay wrapper behind flag → adopt dialogs + toasts first → sheets/drawers →
  menus/popovers/tooltips → enable by default.
- **Dogfood** — internal; watch for focus-return and stuck-scrim reports.
- **GA criteria** — focus management verified, no stuck scrims, reduced-motion fade verified, CLS = 0.
- **Rollback** — flip `ff_motion_overlays` off.

## 16. Test Plan

- **Unit** — overlay state machine transitions (closed→opening→open→closing→closed) and reduced-motion
  fade path.
- **Integration** — iOS/Android sheet drag-dismiss threshold + settle; dialog focus trap/return.
- **End-to-end** — Playwright: dialog scale+scrim in/out, focus returns to trigger, Esc animates out;
  toast lifecycle; reduced-motion emulation → fade only.
- **Security** — n/a.
- **Accessibility** — VoiceOver/TalkBack single announcement, focus trap+return, Esc/back, toast live
  region; axe clean.
- **Performance / load** — 60fps overlay open/close on low-end; no CLS behind the overlay.
- **Manual exploratory** — rapid open/close, swipe-cancel, nested overlays, RTL side sheets.

## 17. Documentation & Training

- **Internal** — "Presenting an overlay" recipe per platform; focus-management rules; when to use
  dialog vs sheet vs popover.
- **End-user** — none.
- **Runbook** — `ff_motion_overlays` scope and kill-switch.

## 18. Open Questions

1. How much of `sonner`'s enter/exit can we customize vs. accept its defaults tuned to our timing?
2. Do we standardize all mobile modals on bottom sheets, or keep centered dialogs for destructive
   confirms?
3. Adopt Android predictive-back for sheets now or later?

## 19. References

- Existing: [`clients/web/src/components/layout/notifications-drawer.tsx`](../../../clients/web/src/components/layout/notifications-drawer.tsx),
  [`clients/web/src/components/ui/fullscreen-modal-shell.tsx`](../../../clients/web/src/components/ui/fullscreen-modal-shell.tsx),
  [`clients/web/src/components/lms-toaster.tsx`](../../../clients/web/src/components/lms-toaster.tsx),
  [`clients/web/src/index.css`](../../../clients/web/src/index.css) (`tooltip-in`, `sidenav-item-in`),
  iOS `IntroCompletionCelebrationSheet` and `.sheet` sites, Android dialog/bottom-sheet sites.
- Standards: WCAG 2.4.3 (Focus Order), 4.1.3 (Status Messages), 2.1.2 (No Keyboard Trap); Material 3
  "Dialogs / Bottom sheets"; Apple HIG "Sheets / Popovers / Alerts".
- Related plans: [AN.1](AN.1-motion-foundation-tokens.md), [AN.2](AN.2-launch-navigation-transitions.md),
  [AN.3](AN.3-load-choreography.md), [AN.4](AN.4-lists-collections-motion.md).
