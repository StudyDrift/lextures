# Accessibility patterns (UX.4)

Keyboard, focus, and ARIA contracts for `clients/web`. Prefer **WAI-ARIA APG**
patterns implemented once in `components/ui/*` over bespoke roles in feature code.

Related: [component library](./component-library.md), [design tokens](../design-tokens.md),
[forms](./forms.md) (UX.6),
plan [UX.4](../completed/ui-ux/UX.4-aria-widget-and-focus-management-remediation.md).

## Forms (UX.6)

- Compose controls with `Field` so label, description, and error share one `id` /
  `aria-describedby` / `aria-invalid` / `aria-required` contract.
- On failed submit, render `ErrorSummary` (`role="alert"`, focus target) with links
  to each field — do not leave focus on the submit button.
- Prefer `aria-describedby` for errors (broader AT support than `aria-errormessage`).
- Validation timing: blur (touched), change only once errored, full check on submit.
- See [forms.md](./forms.md) for zod, server 422 mapping, and dirty-form warnings.

## Rules of thumb

1. **Honest semantics.** If it is not a menu, do not put `role="menu"` on it.
   A list of navigation links is a list of links (or a disclosure), not a menu.
2. **Ship roles only with their keyboard contract.** Declaring `tablist` / `menu`
   / `aria-modal` without arrows, typeahead, or a focus trap is worse than plain
   buttons and links.
3. **Use the design system.** `Dialog`, `AlertDialog`, `Sheet`/`Drawer`, `Menu`,
   `Tabs`, `Tooltip`, `Popover` already implement APG contracts.
4. **Never use native `title=` as a tooltip.** Use `Tooltip` (keyboard-reachable,
   Escape-dismissible, hoverable for SC 1.4.13).

## Overlays (modal)

`Dialog`, `AlertDialog`, and `Sheet`/`Drawer`:

- Move focus into the panel on open
- Trap Tab / Shift+Tab via `createFocusTrap` (`lib/a11y/focus-trap.ts`)
- Mark `#root` `inert` via the **overlay stack** (`lib/a11y/overlay-stack.ts` +
  `useInertBackground`) so nested modals do not clear inert early
- Close on Escape
- Restore focus to the trigger on close, with fallback:
  **trigger → nearest focusable ancestor → `#main-content` / `main` → `body`**

Non-modal overlays (`Popover`, `Tooltip`, menus) **must not** set `aria-modal`.

Portaled content mounts under `document.body` as a sibling of `#root`, so inert
does not lock the overlay itself. Toasts must stay outside the trapped subtree.

```ts
import { Dialog } from '../components/ui'

<Dialog open={open} onClose={close} title="Rename" closeLabel={t('close')}>
  …
</Dialog>
```

## Tabs

Use `Tabs` / `TabList` / `Tab` / `TabPanel`. Contract:

- Roving `tabindex` (only the selected tab is tabbable)
- `ArrowLeft` / `ArrowRight` (horizontal) or `ArrowUp` / `ArrowDown` (vertical)
- `Home` / `End`
- RTL inverts horizontal arrows
- `aria-selected`, `aria-controls` / `aria-labelledby` wiring

Legacy hand-rolled tablists can use `handleTablistKeyDown` from
`lib/a11y/tablist-keyboard` while migrating.

## Menus

Use `Menu` with an anchor ref and item list. Contract:

- Focus first item on open
- `ArrowUp` / `ArrowDown` (wrap), `Home` / `End`
- Printable-character typeahead
- Escape closes and restores focus to the trigger
- Tab closes (APG)

Legacy hand-rolled menus can use `handleMenuKeyDown` + `focusFirstMenuitem`
from `lib/a11y/menu-keyboard`. Prefer full migration to `Menu` when practical.

### When to remove `role="menu"` (FR-3)

If items are primarily **navigation links** (e.g. legal footer, sidebar
shortcuts) and you are not implementing a true application menu, drop the role
and use a disclosure + list of links. Screen readers then get correct link
semantics without a broken menu contract.

## Tooltips

```ts
import { Tooltip } from '../components/ui'

<Tooltip content={t('help.truncated')}>
  <IconButton aria-label={t('help.truncated')}>…</IconButton>
</Tooltip>
```

- Reachable via keyboard focus
- Dismiss with Escape (does not close surrounding dialogs first if handled on the tip)
- Hoverable so pointer users can move onto the tip (SC 1.4.13)
- Never the sole carrier of essential information

## Route focus & landmarks

`AppShell` calls `useFocusOnRoute()` so each client navigation moves focus to
the page `h1` (preferred) or `#main-content` / `main`.

Landmarks expected on shell routes:

| Landmark | Element |
|---|---|
| banner | top bar `header` |
| navigation | side nav |
| main | `#main-content` |
| contentinfo | optional footers |

Skip link: `SkipLink` → `#main-content`.

## Live announcements

Use `announce()` / `LiveRegion` for async results, saves, and validation that
should not move focus. Prefer `polite` unless the message is urgent.

## Focus not obscured (WCAG 2.2 SC 2.4.11)

The shell maintains `--lx-sticky-offset` from the rendered sticky chrome
(`useStickyOffset` in `AppShell`). Focusable content under `.lms-scope` uses
`scroll-margin-block-start: var(--lx-sticky-offset)` so Tab never leaves a field
entirely under the top bar / quiz focus bar / reading focus bar.

- Mark sticky chrome with `header.lms-chrome` or `data-lx-sticky-chrome`.
- Toasts sit top-right with an offset under the sticky bar; focusables also get
  `scroll-margin-inline-end` so they are not entirely covered.

```ts
import { useStickyOffset, syncStickyOffset } from '../lib/a11y'
// useStickyOffset() — already mounted in AppShell
```

## Reorderable / dragging alternatives (WCAG 2.2 SC 2.5.7)

Every `@dnd-kit` surface needs a **single-pointer** alternative in addition to
keyboard reorder. Prefer the design-system helpers:

```ts
import { MoveToPositionMenu, useClickToMove } from '../components/ui'
import { moveItemToIndex } from '../lib/reorderable/move-to-index'
import { KeyboardSensor, defaultKeyboardSensorOptions } from '../lib/dnd/keyboardSensorConfig'
```

Contract:

1. **Drag** (optional) via PointerSensor.
2. **Keyboard** — `Space` lift, arrows move, `Space` drop, `Escape` cancel
   (`KeyboardSensor` + `defaultKeyboardSensorOptions`).
3. **Click / menu** — `MoveToPositionMenu` ("Move to…") or click-source →
   click-target via `useClickToMove`.
4. **Announce** results with `announce()` / live region
   (`"Module 3 moved to position 1 of 7"`).

Inventory + CI: `clients/web/drag-surfaces-inventory.json` and
`npm run a11y:drag-alt` (`drag_surfaces_without_alternative`).

## Target size (WCAG 2.2 SC 2.5.8)

UX.2 size tokens enforce ≥24×24 CSS px (`sizeClasses` / `iconSizeClasses` in
`components/ui/utils.ts`). Prefer those controls. In dense toolbars, use the
*spacing* exception rather than sub-24 targets.

```bash
npm run a11y:target-size   # target_size_violations ratchet
```

Justified exceptions: `clients/web/target-size-exceptions.json`.
Touch-primary surfaces should target **44×44** where practical (FR-4).

## Consistent help (WCAG 2.2 SC 3.2.6)

Authenticated shell routes expose a single help entry in the top bar
(`HelpWidgetMenu`, `data-lx-help-entry`) in a stable relative order (feedback →
help → notifications → account).

## Accessible authentication (WCAG 2.2 SC 3.3.8)

- Password fields: `autoComplete="current-password"` / `"new-password"`; **never**
  block paste (password managers).
- Username/email: `autoComplete="username"`.
- OTP: `autoComplete="one-time-code"`, `type="text"`, visible for re-reading.
- Passkeys are a first-class primary alternative on MFA challenge/setup.
- Magic link remains a passwordless alternative without a cognitive function test.

## CI ratchets

```bash
cd clients/web
npm run a11y:contracts     # fail on regression vs baseline
npm run a11y:baseline      # rewrite after intentional migration batches
npm run a11y:target-size   # UX.5 target size static ratchet
npm run a11y:drag-alt      # UX.5 drag single-pointer alternatives
npm run a11y:wcag22        # contracts + target-size + drag-alt
npm run ds:coverage        # UX.2 interactive coverage (includes raw role=menu/tablist)
```

Metrics:

| Metric | Meaning |
|---|---|
| `aria_contract_coverage` | Satisfied widget roles / total declared widgets |
| `aria_modal_without_trap` | `aria-modal` without focus-trap / DS dialog |
| `role_menu_without_keyboard` | `role=menu` without arrow contract |
| `role_tablist_without_keyboard` | `role=tablist` without arrow contract |
| `title_attribute_tooltips` | Quoted `title=` pseudo-tooltips in feature TSX |
| `target_size_violations` | Suspect sub-24px interactive hosts (static) |
| `drag_surfaces_without_alternative` | dnd-kit surfaces missing single-pointer alt |

Coverage may only increase; defect counts may only decrease.

## Runbook: ARIA contract check failed

1. Read the failing metric and sample paths from the script output.
2. Prefer fixing by switching to a `components/ui` primitive.
3. If you must keep hand-rolled markup, implement the APG contract (use the
   keyboard helpers above) or remove the role (FR-3).
4. After a deliberate migration batch that improves metrics:
   `npm run a11y:baseline`.
5. Do not raise defect counts “to make CI green” without an a11y lead sign-off.

## Screen-reader scripts (manual gate)

Before claiming VPAT re-attestation, run the scripts in UX.4 §16 on
NVDA/Firefox, JAWS/Chrome, and VoiceOver/Safari for: course sidebar, user menu,
course-settings tabs, enrollment dialog, save confirmation, validation recovery.

## Focus indicators

Interactive components use `focusRingClass` from `components/ui/utils`
(`ring-border-focus` on `surface-base` offset). Feature code should not invent
one-off focus rings; use the token ring or the DS control.
