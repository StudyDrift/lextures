# Lextures — visual design system

This document describes the product UI direction for the Lextures learning management system. It is inspired by modern **SaaS dashboard** patterns: calm, structured, and content-first.

**Implementation of colour, space, radius, and elevation is the semantic token system (UX.1).** See [design-tokens.md](design-tokens.md). Feature code must use semantic utilities (`bg-surface-raised`, `text-fg-muted`, …), never raw Tailwind palette literals.

## Design intent

- **Light-first**: Primary workspace uses `surface-raised` / `surface-base` with generous whitespace.
- **Layered surfaces**: Navigation and secondary regions use sunken/base elevation so hierarchy is structural, not arbitrary greys.
- **Soft depth**: Cards use semantic elevation tokens (`elevation-1` / `shadow-card`) and `border-border-default`—enough hierarchy, not neumorphism.
- **Friendly geometry**: Radius tokens (`radius-md` / `radius-lg`) on cards, inputs, and primary controls.

## Color

Colour is **semantic**. Authors pick intent; themes supply values.

| Role | Semantic token / utility | Notes |
| --- | --- | --- |
| **Page background** | `bg-surface-base` | Dark themes elevate surfaces (base darkest) |
| **Card / panel** | `bg-surface-raised` | |
| **Body text** | `text-fg-default` | AA by construction against surfaces |
| **Secondary text** | `text-fg-muted` / `text-fg-subtle` | |
| **Primary accent** | `bg-accent-solid` / `text-accent-fg` | Org can override via brand accent OKLCH |
| **On accent** | `text-fg-on-accent` | Guaranteed AA on solid |
| **Borders** | `border-border-default` / `border-border-strong` | Strong meets SC 1.4.11 |
| **Status** | `info` / `success` / `warning` / `danger` | Fixed vocabulary — learn once |

Do not encode meaning with colour alone (SC 1.4.1): pair with icon or text.

## Typography

- **Font stack**: **Lextures** (self-hosted humanist sans) with system-ui fallbacks. Dyslexia-friendly options (OpenDyslexic, Atkinson Hyperlegible) via Reading Preferences.
- **Type scale (UX.3)**: Authors pick **roles**, not raw sizes — `text-title-lg`, `text-title`, `text-subtitle`, `text-body-lg`, `text-body` (16px default when `ffTypeScale` is on), `text-body-sm`, `text-caption` (13px floor), `text-overline` (uppercase labels only), `text-code`. See [design-tokens.md](design-tokens.md).
- **Headings**: Role utilities include weight and `text-wrap: balance`.
- **Body**: `text-body` / `text-body-lg` with `text-wrap: pretty`; long-form columns use `lex-measure` (~65ch).
- **Dense UI**: Tables and gradebook use `text-body-sm` + `lex-tabular` for aligned numerals.
- **Do not** use `text-xs` / `text-sm` / `text-[Npx]` in feature code (`npm run type:purity`).

## Layout

- **App shell**: Fixed **left navigation** + **scrollable main column**. Main column may include a **top bar** (search, workspace context, primary actions).
- **Content**: Prefer **card grids** for lists and summaries; align to a consistent horizontal rhythm (`px-6` / `p-8` on desktop).

## Components

### Navigation (sidebar)

- Light gray background, **right border** only.
- Items: icon + label, **rounded** hover (`hover:bg-surface-sunken`).
- **Active** state: `bg-accent-surface` + `text-accent-fg`—not a heavy filled bar unless the pattern is icon-only.

### Top bar

- Raised surface, **bottom border**, optional elevation shadow.
- **Search**: Rounded field, sunken fill, `text-fg-subtle` placeholder.

### Buttons

- **Primary**: `bg-accent-solid` + `text-fg-on-accent` (component token `--lx-button-primary-*`).
- **Secondary**: Outline `border-border-default` or ghost on raised surfaces.

### Cards

- Raised surface, large radius, `border-border-default`, `shadow-card`.
- Optional header art; metadata row in **muted** type (`text-fg-muted`).

### Forms (auth and settings)

- Centered **card** on surface-base / sunken.
- Inputs: raised fields, `border-border-default`, focus via `--lx-border-focus` / `ring-focus-ring`.

## Iconography

- **Line-style** icons (e.g. Lucide), consistent stroke; active state inherits accent token.

## Accessibility

- Contrast is a **build invariant** (`npm run contrast:check`); do not add pairs that fail AA.
- Visible **focus** via `border-focus` / focus-ring tokens; semantic headings and `nav` labels.
- High-contrast themes cover every route via `data-theme`, not a one-off override sheet.

## Implementation notes

- **Never** write raw palette literals (`slate-*`, `neutral-*`, arbitrary hex) in `clients/web/src` feature code — enforced by `npm run tokens:purity`.
- Token CSS: `clients/web/src/styles/tokens/`; theme application: `lib/ui-theme.ts`.
- Global font and document background use semantic tokens in `clients/web/src/index.css`; shells under `clients/web/src/components/layout/`.
- Full reference: [design-tokens.md](design-tokens.md).