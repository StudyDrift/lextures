# Component library (UX.2)

Internal design system for `clients/web`. Import from the barrel:

```ts
import { Button, Dialog, Field, Input } from '../components/ui'
// or a path alias equivalent
```

Gallery (staff): **`/design/components`** (tokens at `/design/tokens`).

## What exists

| Group | Components |
|---|---|
| Actions | `Button`, `IconButton`, `LinkButton`, `ButtonGroup`, `SplitButton` |
| Forms | `Field`, `Input`, `Textarea`, `Select`, `Combobox`, `Checkbox`, `Radio`/`RadioGroup`, `Switch`, `SegmentedControl`, `DatePicker`, `FileInput`, `Fieldset`, `ErrorSummary`, `UnsavedChangesBanner` |
| Overlays | `Dialog`, `AlertDialog`, `Sheet`/`Drawer`, `Popover`, `Tooltip`, `Menu`, `ContextMenu`, `OverlaySurface` |
| Navigation | `Tabs`/`TabList`/`Tab`/`TabPanel`, `Breadcrumbs`, `Pagination`, `NavLink`/`UiNavLink`, `Disclosure` |
| Display | `Card`, `Badge`, `Avatar`, `Tag`, `Callout`, `Separator`, `ProgressBar`, `Meter`, `Table` primitives, `DescriptionList` |
| Feedback | `toast` helpers, `EmptyState`, `Skeleton`, `Spinner`, `ErrorState`, `InlineAlert` |
| Layout | `Stack`, `Inline`, `Grid`, `PageHeader`, `Section`, `Toolbar` |

Every interactive component:

- Uses **UX.1 semantic tokens** only
- Meets **≥24×24 CSS px** targets (`size="sm"` included)
- Exposes a **token focus ring**
- Respects **reduced motion** / motion kill-switches where press or overlay motion applies
- Takes **user-visible strings as props** (no product English locked in)

## How to choose

1. Prefer the named primitive over a hand-rolled control.
2. `Button` for actions; `LinkButton` for navigation that looks like a button; `IconButton` only with `aria-label`.
3. `Dialog` for general modals; `AlertDialog` (or `useConfirm` → `ConfirmDialog`) for confirmations.
4. `Tooltip` instead of native `title=`.
5. `EmptyState` / `ErrorState` for non-table empty and error panels.
6. Domain widgets (gradebook grid, TipTap, xyflow) **wrap** these primitives; they are not rewritten here.

## Adding a component

1. One file under `clients/web/src/components/ui/` (≤300 lines).
2. Export from `components/ui/index.ts`.
3. Add a gallery section in `pages/design/components-gallery.tsx`.
4. Unit tests for variants + keyboard contract when interactive.
5. Run:

```bash
cd clients/web
npm run ds:gallery
npm run ds:coverage
```

## Coverage ratchet

```bash
npm run ds:coverage          # fail if coverage ↓ or raw counts ↑
npm run ds:baseline          # rewrite baseline + allowlist after a migration batch
npm run ds:gallery           # every barrel component appears in the gallery
npm run a11y:contracts       # UX.4 ARIA widget contract ratchet
npm run a11y:baseline        # rewrite after intentional a11y migration batches
```

Keyboard / focus / ARIA patterns: [accessibility-patterns.md](./accessibility-patterns.md).

Forms / validation (zod, error summary, dirty guard): [forms.md](./forms.md) (UX.6).

- **Baseline:** `clients/web/design-system-coverage-baseline.json`
- **Allowlist:** `clients/web/raw-interactive-allowlist.json` (per-file raw `<button>` / `<input>` / `role="dialog"` / … counts; may only decrease)
- **Form ratchet:** `npm run forms:check` (`form-fields-baseline.json`)

`design-system-coverage = ds_jsx_tags / (ds_jsx_tags + raw_interactive_tags)`.

## Runbooks

### Coverage check failed on my PR

1. You introduced a raw `<button>` / `<input>` / `role="dialog"` outside `components/ui/`, or coverage fell.
2. Replace with a library component, **or** if this is an intentional temporary exception after a large migration batch, regenerate only after review:

   `npm run ds:baseline` — **do not** raise raw counts without a linked cleanup plan.

3. Prefer fixing the call site over growing the allowlist.

### My component has no importers and CI is failing

FR-13: dead library code is deleted or adopted within a release. Either:

- Use it from a real feature path, or
- Show it in the gallery **and** keep a production importer, or
- Remove the export.

Gallery-only is enough for AC-9 importer check when the gallery imports the barrel.

## Related

- [Design tokens (UX.1)](../design-tokens.md)
- Plan (completed): `docs/completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md`
