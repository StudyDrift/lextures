# UX.4 — ARIA Widget and Focus Management Remediation

> Implementation plan. Source: [audit.md](audit.md) §3 G-3, G-4, G-5a, G-5c.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.4 |
| **Section** | UI/UX — Accessibility |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | BROKEN — roles declared without their keyboard contracts |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Accessibility + Design Systems |
| **Depends on** | UX.2 |
| **Unblocks** | UX.5; VPAT re-attestation; EU/EN 301 549 contractual claims |

---

## 1. Problem Statement

The product declares ARIA widget roles it does not implement. **All 22
`role="tablist"` implementations lack arrow-key navigation** and **33 of 37
`role="menu"` implementations** do the same. **126 files declare `aria-modal`**
while only **3** import the focus-trap utility that already exists in the
repository — meaning ~123 dialogs tell assistive technology that background
content is inert when a keyboard user can Tab straight into it. This is worse
than shipping plain links and buttons: declaring the role sets an expectation the
implementation breaks, actively misleading screen-reader users. It fails WCAG
2.1.1 (Keyboard) and 4.1.2 (Name, Role, Value), contradicts the published VPAT
(`docs/vpat/VPAT_2.5_INT_Lextures_2026-05.md`), and is a live contractual exposure
under EN 301 549 (**R-36**).

## 2. Goals

- Bring every declared ARIA widget role to **full WAI-ARIA APG keyboard
  conformance**, or remove the role in favour of honest semantics.
- Guarantee focus management for every overlay: trap on open, restore on close,
  background inert, Escape closes.
- Replace all 289 `title=` pseudo-tooltips with accessible tooltip components.
- Establish a designed, contrast-verified focus indicator on **every** interactive
  element.
- Make ARIA conformance **structurally impossible to regress** by delivering it
  through UX.2 components plus CI enforcement.

## 3. Non-Goals

- A full manual screen-reader audit of every one of 200 routes (scoped, sampled —
  see §16).
- WCAG 2.2's *new* criteria — those are [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md).
- Colour contrast — delivered by [UX.1](UX.1-semantic-design-token-system.md).
- Form error association — delivered by [UX.6](UX.6-form-and-validation-system.md).
- Native clients.

## 4. Personas & User Stories

- **As a screen-reader user**, I want a tab set announced as tabs to actually
  respond to arrow keys, so that I am not stranded.
- **As a keyboard-only user**, I want a dialog to hold my focus so that I do not
  silently land on background controls I cannot see.
- **As a keyboard-only user**, I want focus to return to the button I pressed when
  a dialog closes, so that I do not lose my place on the page.
- **As a low-vision sighted keyboard user**, I want a clearly visible focus ring
  on every control on every background.
- **As a touch user**, I want tooltip information to be reachable, since `title`
  never appears on touch.
- **As a compliance owner**, I want the VPAT to be re-attestable with evidence.

## 5. Functional Requirements

- **FR-1.** Every `role="tablist"` MUST implement the APG Tabs pattern:
  `ArrowLeft`/`ArrowRight` (or `Up`/`Down` for vertical), `Home`, `End`, roving
  `tabindex`, `aria-selected`, `aria-controls`/`aria-labelledby` wiring, and
  correct RTL arrow inversion.
- **FR-2.** Every `role="menu"` MUST implement the APG Menu/Menu Button pattern:
  focus moves to the first item on open, `ArrowUp`/`ArrowDown` wrap, `Home`/`End`,
  printable-character typeahead, `Escape` closes and returns focus, `Tab` closes.
- **FR-3.** Roles that are *not* genuinely menus (e.g. a list of navigation links)
  MUST have the role **removed** and be marked up as a list of links. Honest
  semantics beat a wrong pattern.
- **FR-4.** Every modal overlay MUST: move focus into the dialog on open, trap
  focus for the dialog's lifetime, mark all sibling content `inert`, close on
  `Escape`, and restore focus to the invoking element on close.
- **FR-5.** Every modal MUST have an accessible name via `aria-labelledby`
  pointing at its visible title, or `aria-label` when there is no visible title.
- **FR-6.** Non-modal overlays (popovers, tooltips, comboboxes) MUST NOT declare
  `aria-modal` and MUST follow their own APG patterns.
- **FR-7.** All 289 `title=` tooltips MUST be replaced with the UX.2 `Tooltip`
  component, which MUST be reachable by keyboard focus, dismissible with `Escape`,
  hoverable (SC 1.4.13 Content on Hover or Focus), and MUST NOT be the sole
  carrier of essential information.
- **FR-8.** Every interactive element MUST render the token-driven focus indicator
  from UX.1, meeting ≥3:1 against both the element and its background.
- **FR-9.** Route changes MUST move focus to the new page's `h1` or main landmark.
  The existing `lib/a11y/use-focus-on-route.ts` MUST be applied on all routes.
- **FR-10.** Every page MUST expose correct landmarks (`banner`, `navigation`,
  `main`, `complementary`, `contentinfo`), and a working skip link to `main`.
- **FR-11.** Dynamic content changes (save results, validation, async loads) MUST
  announce via the existing `components/a11y/live-region.tsx` with correct
  politeness.
- **FR-12.** Custom controls built on `div`/`span` MUST be converted to native
  elements or given complete role + keyboard + state handling. A lint rule MUST
  forbid `onClick` on non-interactive elements without `role` + `tabIndex` +
  `onKeyDown`.
- **FR-13.** A CI check MUST compute **`aria-contract-coverage`**: for each
  declared ARIA widget role, whether the file (or the component it delegates to)
  implements the required keyboard handlers. It MUST fail on any decrease.
- **FR-14.** An automated E2E sweep MUST exercise every registered overlay and
  menu for focus trap, restore, Escape and arrow keys — driven from the UX.2
  gallery registry so new components are covered automatically.

## 6. Non-Functional Requirements

- **Performance** — Focus trapping and `inert` management MUST add no measurable
  INP cost; overlay open MUST stay within the AN.1 motion budget. `inert` is
  applied to a single container, not per element.
- **Security** — None specific. Focus restoration MUST NOT leak focus to a
  detached node (avoid holding strong references across unmount).
- **Privacy & Compliance** — Delivers WCAG 2.1 SC 2.1.1, 2.1.2, 2.4.3, 2.4.7,
  4.1.2, 1.4.13, and 4.1.3; underpins EN 301 549 and the VPAT.
- **Accessibility** — This plan *is* the accessibility requirement.
- **Scalability** — Conformance is a property of UX.2 components, so new features
  inherit it.
- **Reliability** — Focus restoration MUST handle the case where the trigger has
  unmounted (fall back to the nearest stable ancestor or `main`).
- **Observability** — CI emits `aria_contract_coverage`, `aria_modal_without_trap`,
  `title_attribute_tooltips`, `role_menu_without_keyboard`.
- **Maintainability** — Zero bespoke ARIA in feature code; everything routes
  through `components/ui/`.
- **Internationalization** — Arrow-key direction MUST invert in RTL. Typeahead
  MUST work with non-Latin input.
- **Backward compatibility** — Keyboard behaviour changes are additive. Where a
  role is *removed* (FR-3), the visual result is unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* any tab set in the product, *When* a keyboard user focuses a
  tab and presses `ArrowRight`, `ArrowLeft`, `Home`, `End`, *Then* selection moves
  per the APG and only the selected tab is in the tab order.
- **AC-2.** *Given* `dir="rtl"`, *When* the user presses `ArrowRight` on a tab
  set, *Then* selection moves to the *previous* tab.
- **AC-3.** *Given* any menu, *When* opened, *Then* focus is on the first item;
  arrow keys wrap; typing a character jumps to the matching item; `Escape` closes
  and returns focus to the trigger.
- **AC-4.** *Given* any modal dialog, *When* open, *Then* `Tab` and `Shift+Tab`
  never leave it, background content reports `inert`, and `Escape` closes it.
- **AC-5.** *Given* any modal dialog, *When* it closes, *Then* focus is on the
  element that opened it.
- **AC-6.** *Given* the codebase, *When* the ARIA contract check runs, *Then*
  `aria-contract-coverage = 100%` and `aria_modal_without_trap = 0`.
- **AC-7.** *Given* the codebase, *When* scanned, *Then* `title=` used as a
  tooltip appears **0** times in `src/**/*.tsx`.
- **AC-8.** *Given* a route change, *When* it completes, *Then* focus is on the
  new page's `h1`/main landmark and a screen reader announces the new page title.
- **AC-9.** *Given* every interactive element on the top 40 routes, *When* focused
  by keyboard, *Then* a visible indicator renders at ≥3:1 contrast.
- **AC-10.** *Given* the top 40 routes, *When* axe runs, *Then* there are **0**
  violations in the `cat.keyboard`, `cat.aria` and `cat.name-role-value`
  categories.
- **AC-11.** *Given* the manual screen-reader scripts (§16), *When* executed on
  NVDA/Firefox, JAWS/Chrome and VoiceOver/Safari, *Then* every scripted task
  completes without a blocking defect.
- **AC-12.** *Given* the E2E overlay sweep, *When* a new overlay is added to the
  gallery without focus management, *Then* CI fails.

## 8. Data Model

None. UX.4 is entirely client-side. No tables, columns, enums, indexes,
migrations or backfill.

## 9. API Surface

None. No HTTP or WebSocket changes, no rate-limit considerations, no OpenAPI
changes.

## 10. UI / UX

- **New pages** — none.
- **Modified pages** — all surfaces containing tabs, menus, dialogs or tooltips
  (≈180 files), migrating to UX.2 components.
- **Key user flows** (all keyboard-only)
  1. Open a menu from the top bar → arrow to an item → Enter → focus lands
     sensibly on the destination.
  2. Open a settings dialog → Tab through it → Escape → focus returns to the
     trigger.
  3. Move across course settings tabs with arrow keys.
  4. Focus a truncated label → tooltip appears → Escape dismisses it without
     closing the surrounding dialog.
- **States** — overlays: opening, open, closing, error-while-open (the dialog must
  not close and lose the user's input).
- **Mobile/responsive** — tooltips become press-and-hold or inline help on touch;
  essential information is never tooltip-only (FR-7).
- **Accessibility annotations** — the deliverable is a written focus-order
  specification per overlay class, committed alongside the components.
- **Copy & i18n** — new/normalised `aria-label` strings MUST be i18n keys, not
  literals. This intersects [UX.15](UX.15-i18n-coverage-and-rtl-completion.md):
  today's labels (e.g. `aria-label="User menu"`) are hardcoded English.

## 11. AI / ML Considerations

Not AI-touching. AI surfaces (tutor panel, ask-AI, grader agent popovers) are
in-scope *as consumers* — they contain menus and popovers that must conform.
Streaming AI responses MUST announce via a polite live region, not by moving
focus.

## 12. Integration Points

- **External** — none added. If React Aria Components is adopted in UX.2 §18, it
  supplies most FR-1…FR-6 behaviour directly.
- **Internal**
  - `clients/web/src/lib/a11y/focus-trap.ts` — becomes universally used
  - `clients/web/src/lib/a11y/use-focus-on-route.ts` — applied on all routes
  - `clients/web/src/lib/a11y/announcer.ts`,
    `components/a11y/live-region.tsx` — announcements
  - `clients/web/src/components/layout/top-bar.tsx`, `side-nav*.tsx`,
    `notifications-drawer.tsx`, `help-widget.tsx` — high-traffic menus
  - `clients/web/src/components/command-palette/command-palette-dialog.tsx`
  - `clients/web/src/components/use-confirm.tsx`
  - `clients/web/src/components/layout/side-nav-tooltip.tsx`,
    `components/ui/icon-action-tooltip.tsx`,
    `components/ui/action-error-tooltip.tsx`
  - `clients/web/eslint-rules/`, `.oxlintrc.json`
  - `clients/web/scripts/check-aria-contracts.mjs` — new
  - `e2e/` — overlay sweep
- **Events** — none.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md).
  UX.4 is largely *delivered by* the UX.2 migration; this plan owns the
  verification, the removal of wrong roles, the tooltip replacement, and the CI
  gates.
- **Must ship before** — [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md) and any
  VPAT re-attestation.
- **Shared infra** — CI runners capable of running axe and Playwright.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `inert` background management breaks nested overlays (dialog → confirm → toast) | **H** | **H** | Central overlay stack manager owning `inert`; explicit E2E tests for 3-deep nesting; toasts live outside the trapped subtree |
| Focus restoration fails when the trigger unmounts (row deleted, list re-sorted) | H | M | Documented fallback chain: trigger → nearest stable ancestor → `main`; unit-tested |
| Removing a wrong `role="menu"` (FR-3) is mistaken for a regression | M | L | Each removal documented in the PR with the APG rationale |
| Automated ARIA checks give false confidence — axe catches ~30% of issues | **H** | **H** | Manual screen-reader scripts (AC-11) are a **gate**, not a nice-to-have; budget real AT-user testing |
| Arrow-key handling conflicts with editors (TipTap, CodeMirror) inside dialogs | M | M | Components stop propagation only at their own boundary; explicit tests for the editor-in-dialog case |
| Long tail of bespoke ARIA outside UX.2's reach | M | M | FR-13 coverage check enumerates them; finite list, tracked to zero |
| Command palette regression — currently one of the better surfaces | L | M | It gets dedicated E2E coverage before migration |

## 15. Rollout Plan

- **Feature flag** — none. Accessibility fixes must not be flag-gated; a flagged
  a11y fix is an a11y bug for anyone in the control group.
- **Sequencing**
  1. ARIA contract analyser + baseline (no fixes) — establishes the ratchet.
  2. Overlay stack manager + `inert` + focus trap/restore in UX.2 overlays.
  3. Migrate the 129 dialogs, highest-traffic first.
  4. Tabs (22) and menus (37) migrated; wrong roles removed.
  5. Tooltip replacement (289 sites).
  6. Route focus + landmarks + skip link audit.
  7. Focus-indicator sweep.
  8. CI gates flipped to error; E2E overlay sweep enabled.
  9. VPAT re-attestation with fresh evidence.
- **Dogfood** — internal org, with a keyboard-only day for the team per batch.
- **GA criteria** — AC-1…AC-12 green, including the manual AT gate.
- **Rollback** — per-batch revert. Note that reverting restores a known
  accessibility defect, so rollback requires an accessibility-lead sign-off.

## 16. Test Plan

- **Unit** — per component: roving tabindex, arrow/Home/End, typeahead, RTL
  inversion, Escape, focus trap entry/exit, focus restore including the
  unmounted-trigger fallback.
- **Integration** — nested overlay stack (dialog → alert-dialog → toast);
  editor-inside-dialog key handling; command palette over a dialog.
- **End-to-end** — Playwright, keyboard-only: the 8 critical journeys (sign in,
  find a course, open a module, take a quiz, submit an assignment, grade a
  submission, change a course setting, invite a user) completed **without a
  mouse**. Plus the gallery-driven overlay sweep (FR-14).
- **Security** — none specific.
- **Accessibility**
  - *Automated*: axe on the top 40 routes + every gallery entry, gating on
    keyboard/aria/name-role-value categories (AC-10).
  - *Screen-reader scripts* (the real gate, AC-11): NVDA + Firefox (Windows),
    JAWS + Chrome (Windows), VoiceOver + Safari (macOS), VoiceOver (iOS Safari).
    Scripts cover: navigate the course sidebar; open and operate the user menu;
    move across course-settings tabs; open, complete and dismiss the enrollment
    dialog; receive a save confirmation; recover from a validation error.
  - *AT-user testing*: at least one session with a screen-reader user who is not
    on the team, before GA.
- **Performance / load** — INP on overlay open/close ≤200 ms at p75; no CLS from
  focus scroll-into-view.
- **Manual exploratory** — keyboard-only exploratory session per migrated
  directory; Windows High Contrast Mode pass.

## 17. Documentation & Training

- **End-user** — help-centre: "Using Lextures with a keyboard" and "Using
  Lextures with a screen reader", including the shortcut sheet already shipped in
  `components/keyboard-shortcuts/`.
- **Admin / instructor** — note in the accessibility statement.
- **Engineer** — `docs/guides/accessibility-patterns.md`: the APG contracts we
  implement, the overlay stack model, the focus-restoration fallback chain, when
  to remove a role rather than fix it.
- **API reference** — n/a.
- **Runbook** — "ARIA contract check failed" and "How to run the screen-reader
  scripts".
- **Compliance** — update `docs/vpat/VPAT_2.5_INT_Lextures_2026-05.md` and
  `docs/vpat/VPAT_UPDATE_CHECKLIST.md` with the new evidence.

## 18. Open Questions

1. Do we adopt React Aria Components (UX.2 §18 Q1)? It resolves most of FR-1…FR-6
   at the source and materially reduces this plan's effort. **This decision should
   be made before UX.2 implementation begins.**
2. Which of the 37 `role="menu"` sites are genuinely menus vs. navigation lists?
   Needs a one-pass triage with the accessibility lead; estimated 1 day.
3. Do we contract an external AT-user testing panel, or recruit through existing
   institutional partners?
4. Should the ARIA contract analyser be source-based (cheap, approximate) or
   runtime-based via the E2E sweep (accurate, slower)? *Recommendation: both —
   source for the fast ratchet, E2E for truth.*
5. Does `clients/desktop` embed the web app, and does it therefore inherit all of
   this?

## 19. References

- Existing files: `clients/web/src/lib/a11y/focus-trap.ts`,
  `clients/web/src/lib/a11y/use-focus-on-route.ts`,
  `clients/web/src/lib/a11y/announcer.ts`,
  `clients/web/src/components/a11y/live-region.tsx`,
  `clients/web/src/components/layout/top-bar.tsx` (lines 124–156, 219–259 —
  representative broken menus),
  `clients/web/src/components/layout/side-nav-tooltip.tsx`,
  `clients/web/src/components/command-palette/command-palette-dialog.tsx`,
  `clients/web/src/components/use-confirm.tsx`
- Research: [research.md](research.md) R-35, R-36, R-37
- Audit: [audit.md](audit.md) G-3, G-4, G-5a, G-5c
- External: [WAI-ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/),
  [WCAG 2.2](https://www.w3.org/TR/WCAG22/),
  [EN 301 549](https://www.deque.com/en-301-549-compliance/)
- Related plans: [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md),
  [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md),
  [UX.6](UX.6-form-and-validation-system.md),
  [UX.15](UX.15-i18n-coverage-and-rtl-completion.md),
  `../../completed/12-accessibility/`, `../standards/S20-accessibility-legal-mandates.md`
