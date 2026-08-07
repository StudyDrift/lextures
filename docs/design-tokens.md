# Design Tokens — Semantic System (UX.1)

Three-layer token architecture for `clients/web`. Feature code references **semantic** tokens only.

## Layers

| Layer | Path | Role |
|---|---|---|
| **Primitive** | `clients/web/src/styles/tokens/primitives.css` | Raw scales (`--lx-neutral-*`, `--lx-accent-*`, status hues, space, radius) in OKLCH with sRGB fallback |
| **Semantic** | `clients/web/src/styles/tokens/semantic.css` | Intent (`--lx-surface-raised`, `--lx-fg-muted`, status, elevation). Values switch by `[data-theme]` |
| **Component** | `clients/web/src/styles/tokens/component.css` | Defaults for UX.2 primitives (`--lx-button-primary-bg`, …) |
| **Motion** | `clients/web/src/styles/tokens/motion.css` | AN.1 durations/easings (names preserved as aliases) |
| **@theme** | `clients/web/src/styles/tokens/theme.css` | Tailwind v4 utilities |

Export for Figma / native: `clients/packages/tokens/tokens.json` (W3C Design Tokens Format Module).

## Themes

Set on `<html data-theme="…">` before first paint (inline script in `index.html`):

- `light`
- `dark` — **surface elevation** (base darkest → raised/overlay lighter)
- `high-contrast-light`
- `high-contrast-dark`

`prefers-contrast: more` and the in-app high-contrast toggle select HC themes. Legacy `.dark` / `.high-contrast` classes are still toggled for unmigrated call sites.

**Surface tint (personal preference):** `data-surface-tint` on `<html>` (`neutral` default = true gray/near-black; options: `slate`, `blue`, `teal`, `green`, `purple`, `red`, `amber`). Light mode uses soft pastels; dark mode uses deep near-black hues. Settings → Preferences → Background colour. See `lib/ui-surface-tint.ts` and `styles/tokens/surface-tints.css`.

API: `clients/web/src/lib/ui-theme.ts` (`applyUiTheme`, `applyHighContrast`, `resolveSemanticTheme`).

## Authoring (feature code)

```tsx
// ✅ semantic
<div className="bg-surface-raised text-fg-default border border-border-default">
  <p className="text-fg-muted">Secondary</p>
  <button className="bg-accent-solid text-fg-on-accent">Save</button>
  <span className="bg-danger-surface text-danger-fg">Error</span>
</div>

// ❌ raw palette — blocked by npm run tokens:purity
<div className="bg-white text-slate-900 dark:bg-neutral-900 dark:text-neutral-100" />
```

### Common utilities

| Intent | Utility |
|---|---|
| Page background | `bg-surface-base` |
| Card / panel | `bg-surface-raised` |
| Sunken well | `bg-surface-sunken` |
| Body text | `text-fg-default` |
| Secondary text | `text-fg-muted` |
| Placeholder | `text-fg-subtle` |
| On primary button | `text-fg-on-accent` |
| Default border | `border-border-default` |
| Strong / UI border | `border-border-strong` |
| Primary action | `bg-accent-solid` |
| Status | `bg-*-surface` + `text-*-fg` (`info` \| `success` \| `warning` \| `danger` \| `accent`) |

Gallery: `/design/tokens` (specimen + live contrast table). Components: `/design/components` (UX.2 library).

## Adding a token

1. Add primitive scale step only if needed (`primitives.css`).
2. Map semantic intent in `semantic.css` for **all four themes**.
3. Expose in `theme.css` as `--color-*` / radius / space.
4. Update `clients/packages/tokens/tokens.json` hex values.
5. Run `npm run contrast:check` — must stay green.
6. Document in this file if it is a new semantic family.

## Contrast (build invariant)

```bash
cd clients/web && npm run contrast:check
# Deliberate fail fixture (AC-2):
node scripts/check-contrast.mjs --fixture=failing
```

Declared semantic (fg, bg) pairs must meet WCAG 2.1 AA (4.5:1 text, 3:1 non-text UI). Pair list lives in `scripts/check-contrast.mjs`.

## Raw palette purity (ratchet)

```bash
npm run tokens:purity                 # fail on new / increased violations
node scripts/check-raw-palette.mjs --write-allowlist   # regenerate after codemod
```

Allowlist: `clients/web/raw-palette-allowlist.json`. Counts may only decrease; when empty, zero raw palette literals remain (AC-1).

## Codemod

```bash
npm run tokens:codemod:dry -- --dir=src/components/layout
npm run tokens:codemod -- --dir=src/components/layout
```

Idempotent. Emits unmapped literal report for manual triage.

## Org brand accent

Admins set `accentOklch` on `PUT /api/v1/orgs/{id}/branding`. Server:

1. Parses/re-serialises OKLCH only (no CSS injection).
2. Derives full accent ramp.
3. Rejects (422 `brand_accent_contrast`) if `fg-onAccent` on solid fails AA, with optional suggestion.

Columns: `tenant.org_branding.brand_accent_oklch`, `brand_tokens_version` (migration `465_org_brand_tokens.sql`).

Client applies ramp to `--lx-accent-*` via `OrgBrandingProvider`.

## UI modes (k2 / elementary)

Token overrides on `html.ui-mode-k2` / `html.ui-mode-elementary` (radius, space, motion, **type ramp** via `--lx-type-mode-mult`). Touch targets remain in `styles/ui-modes/*.css`.

## Typography (UX.3)

Semantic type roles live in `clients/web/src/styles/tokens/typography.css` and ship as Tailwind utilities:

| Role utility | Size / leading (base) | Use |
|---|---|---|
| `text-display` | 36 / 40 | Marketing-adjacent hero only |
| `text-title-lg` | 28 / 34 (clamped) | Page title |
| `text-title` | 22 / 28 | Section title |
| `text-subtitle` | 18 / 26 | Card / panel heading |
| `text-body-lg` | 18 / 28 | Long-form course content |
| `text-body` | **16 / 24** when `ffTypeScale` | Default UI and prose |
| `text-body-sm` | 14 / 20 | Dense UI, table cells |
| `text-caption` | 13 / 18 | Metadata, timestamps — **floor** for sentences |
| `text-overline` | 12 / 16 | Uppercase labels only, never sentences |
| `text-code` | 14 / 22 | Monospace |

### Rules

- **Default body** inherits on `.lms-scope`. `ffTypeScale` (`data-type-scale="on"`) raises body from 14px → 16px; roles and lint ship unflagged.
- **No sentences below 13px.** `text-overline` is for short uppercase labels only.
- **Caption contrast:** pair `text-caption` / `text-overline` with `text-fg-default` (or other ≥7:1 tokens), not `text-fg-muted`.
- **Measure:** long-form surfaces use `lex-measure` (target 65ch, min 45ch, max 75ch).
- **Tabular numbers:** dense numeric columns use `lex-tabular` / `lex-num` (`font-variant-numeric: tabular-nums`).
- **Reading preferences** (`Aa` panel) set `--lx-type-scale` (textScale 1–1.5), font family, letter/word spacing, and line height on the root so preferences apply across learning surfaces.
- **Locale:** `html[lang=ar]` multiplies sizes via `--lx-type-locale-mult` (1.08).
- **Lint:** `npm run type:purity` forbids raw `text-xs` / `text-sm` / `text-[Npx]` in `src/**/*.tsx` (ratcheting allowlist `type-role-allowlist.json`). Codemod: `npm run type:codemod`.

Gallery: `/design/tokens` type specimen section.

## Runbook: contrast CI failure

1. Read failing pair from `npm run contrast:check` output.
2. Adjust the **semantic** mapping for that theme in `semantic.css` (prefer not changing primitives).
3. Mirror hex in `tokens.json`.
4. Re-run check. Do not silence by deleting the pair.

## Related

- [design.md](design.md) — product design principles (references tokens, not raw hex)
- Plan: `docs/completed/ui-ux/UX.1-semantic-design-token-system.md`
- Plan: `docs/completed/ui-ux/UX.3-typography-and-reading-system.md`
- AN.1 motion: `docs/completed/animations/`
