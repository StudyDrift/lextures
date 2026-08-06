# UX.1 — Semantic Design Token System

> Implementation plan. Source: [audit.md](audit.md) §2 G-1, G-14, G-16.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.1 |
| **Section** | UI/UX — Foundations |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Design Systems (new virtual team: 1 design + 2 web) |
| **Depends on** | — (this is the root of the programme) |
| **Unblocks** | UX.2, UX.3, UX.9, UX.10, UX.11, UX.12, UX.14, UX.18; org white-labelling; high-contrast mode |

---

## 1. Problem Statement

`clients/web/src` contains **33,331 raw Tailwind palette literals across 698 of
795 component files (88%)**, with two competing neutral ramps (`slate-*` for
light, `neutral-*` for dark) written out by hand on every element. There is no
semantic token layer: `src/index.css` defines motion tokens only. As a result no
one can change a colour, ship a genuine high-contrast theme, honour an
organisation's branding, or guarantee contrast — because there are 33,331 places
to change and a hand-maintained CI allowlist that cannot see the pairs actually
rendered. Colour also carries no meaning (the dashboard assigns teal, violet,
emerald and amber to sibling sections arbitrarily), which per **R-1/R-3** is pure
extraneous cognitive load in a product whose job is learning.

## 2. Goals

- Establish a **three-layer token architecture** (primitive → semantic →
  component) per **R-25**, with feature code referencing **semantic tokens only**.
- Reduce raw palette literals in `.tsx` from 33,331 to **0**, enforced by CI.
- Collapse the `slate`/`neutral` dual ramp into one neutral scale with light and
  dark *value* mappings, so a component declares intent once.
- Make contrast **derivable rather than allowlisted** — replace pair-by-pair CI
  approval with OKLCH lightness rules that guarantee AA by construction.
- Give colour semantic meaning: a fixed status vocabulary (info / success /
  warning / danger / accent) that a learner can learn once.
- Ship **dark mode via surface elevation** (R-9 trend table) rather than
  per-element `dark:` pairs.

## 3. Non-Goals

- Visual redesign. UX.1 is a **refactor**: the rendered pixels should change as
  little as possible, except where the current value fails contrast.
- Component API changes — that is UX.2.
- Typography scale — that is UX.3.
- Motion tokens — already shipped by AN.1; UX.1 adopts and re-homes them
  unchanged.
- Native clients (iOS/Android/desktop). Token *values* are exported for them; the
  consuming work is out of scope.

## 4. Personas & User Stories

- **As a web engineer**, I want to write `bg-surface-raised text-fg-muted` so that
  I never make a light/dark or contrast decision by hand again.
- **As a designer**, I want one file to be the source of truth for colour so that
  a change I make in Figma reaches production without 698 file edits.
- **As an administrator at a partner institution**, I want our brand colour
  applied to the product so that it feels like our environment, without breaking
  contrast anywhere.
- **As a student with low vision**, I want a high-contrast theme that actually
  covers every screen, so that I am not returned to unreadable text as soon as I
  leave the reading view.
- **As a homeschool parent**, I want the K-2 and elementary UI modes to be
  visually distinct and calm, driven by the same token layer instead of bespoke
  CSS.
- **As a compliance owner**, I want to state in the VPAT that contrast is
  structurally guaranteed rather than sampled.

## 5. Functional Requirements

- **FR-1.** The system MUST define a **primitive** token layer in
  `clients/web/src/styles/tokens/primitives.css` containing raw scale values only
  (`--lx-neutral-0` … `--lx-neutral-1000`, `--lx-accent-50` … `--lx-accent-950`,
  and one ramp per status hue). Primitives MUST be expressed in **OKLCH** with an
  sRGB fallback.
- **FR-2.** The system MUST define a **semantic** token layer in
  `styles/tokens/semantic.css` covering at minimum:
  - Surfaces: `--lx-surface-base`, `-raised`, `-overlay`, `-sunken`, `-inverse`
  - Foregrounds: `--lx-fg-default`, `-muted`, `-subtle`, `-onAccent`, `-inverse`
  - Borders: `--lx-border-default`, `-subtle`, `-strong`, `-focus`
  - Status (× `-surface`, `-fg`, `-border`): `info`, `success`, `warning`,
    `danger`, `accent`
  - Interaction: `--lx-focus-ring`, `--lx-selection`, `--lx-overlay-scrim`
  - Elevation: `--lx-elevation-0` … `-3`
  - Radius: `--lx-radius-sm|md|lg|pill`
  - Space: `--lx-space-1` … `--lx-space-12` on a 4px base
- **FR-3.** Semantic tokens MUST resolve to different primitives per theme via a
  single `[data-theme]` switch. Feature code MUST NOT contain `dark:` colour
  variants.
- **FR-4.** Dark theme MUST use **surface elevation** — `surface-base` darkest,
  each elevation step lighter — rather than inverted per-element pairs.
- **FR-5.** The system MUST expose every semantic token as a Tailwind v4 utility
  via `@theme`, so authors write `bg-surface-raised`, `text-fg-muted`,
  `border-border-subtle`.
- **FR-6.** The system MUST define **component** tokens for the primitives shipped
  in UX.2 (e.g. `--lx-button-primary-bg`), defaulting to semantic values.
- **FR-7.** A build-time validator MUST compute the contrast ratio of every
  declared `(fg, bg)` semantic pairing in every theme and **fail** if any is below
  WCAG 2.1 AA (4.5:1 text, 3:1 large text and non-text UI).
- **FR-8.** A lint rule (`lextures/no-raw-palette`) MUST fail the build on any
  Tailwind palette literal or arbitrary hex in `.tsx` under `src/`, with a
  per-directory allowlist that ratchets downward and cannot increase.
- **FR-9.** The system MUST ship at least four themes from one token set: `light`,
  `dark`, `high-contrast-light`, `high-contrast-dark`. The existing
  `styles/high-contrast.css` MUST be reimplemented as a theme, not an override
  sheet.
- **FR-10.** The system MUST support a **per-organisation accent override**: an
  org supplies one brand hue; the build derives the full accent ramp in OKLCH and
  rejects any hue that cannot satisfy FR-7.
- **FR-11.** Tokens MUST be exported to a **W3C Design Tokens Format Module**
  (Oct 2025 stable) JSON artefact at `clients/packages/tokens/tokens.json` for
  Figma and native-client consumption.
- **FR-12.** The `k2` and `elementary` UI modes MUST be expressed as token
  overrides, not separate stylesheets.
- **FR-13.** AN.1 motion tokens MUST be relocated into the token layer unchanged,
  preserving every existing variable name as an alias so no motion code changes.
- **FR-14.** The system SHOULD provide a `/type`-style internal token gallery
  route rendering every token, its value per theme, and its computed contrast.

## 6. Non-Functional Requirements

- **Performance** — Token CSS MUST add ≤4 KB gzip to the entry bundle. Theme
  switching MUST be a single attribute write on `<html>` with no re-render and no
  FOUC; the active theme MUST be applied by an inline head script before first
  paint.
- **Security** — Org accent values are attacker-controlled input. They MUST be
  parsed and re-serialised as OKLCH; raw strings MUST NOT be interpolated into
  CSS. Reject anything that is not a valid colour.
- **Privacy & Compliance** — Supports WCAG 2.1/2.2 SC 1.4.3, 1.4.11, 1.4.1;
  EN 301 549; and the VPAT claim in `docs/vpat/`.
- **Accessibility** — Contrast conformance becomes a build invariant (FR-7).
  `prefers-contrast: more` and `forced-colors` MUST select the high-contrast
  themes automatically.
- **Scalability** — Adding a status hue MUST require editing exactly two files
  (primitives, semantic) and MUST NOT require touching feature code.
- **Reliability** — The codemod (§15) MUST be idempotent and produce a
  machine-readable report of every unmapped literal for manual triage.
- **Observability** — CI MUST emit `token_purity_violations` and
  `contrast_pairs_checked` as build metrics, tracked over time.
- **Maintainability** — One owner file per layer. `docs/design-tokens.md` is
  rewritten to describe the token system; `docs/design.md` is rewritten to
  reference tokens instead of hex values.
- **Internationalization** — No colour may encode meaning without a
  non-colour affordance (icon/text), preserving SC 1.4.1 across locales.
- **Backward compatibility** — Migration is mechanical and behaviour-preserving.
  Any intentional visual change (contrast repairs) MUST be listed explicitly in
  the PR description with before/after screenshots.

## 7. Acceptance Criteria

- **AC-1.** *Given* the repo at HEAD, *When* `npm run lint` runs in
  `clients/web`, *Then* `lextures/no-raw-palette` reports **0** violations in
  `src/**/*.tsx` and the allowlist file is empty.
- **AC-2.** *Given* the token definitions, *When* the contrast validator runs,
  *Then* every declared semantic pairing in all four themes meets WCAG AA, and a
  deliberately-failing pair added in a test fixture causes a non-zero exit.
- **AC-3.** *Given* a page in light mode, *When* `document.documentElement`
  switches to `data-theme="dark"`, *Then* the page renders correctly with **no**
  `dark:` class present anywhere in the rendered DOM.
- **AC-4.** *Given* an org with brand hue `#B3122B`, *When* an admin saves it,
  *Then* a full accent ramp is derived, all AA checks pass, and the sidebar active
  state, primary buttons and links adopt it without any other colour changing.
- **AC-5.** *Given* an org supplies a hue whose ramp cannot reach 4.5:1 for
  `fg-onAccent`, *When* they save, *Then* the save is rejected with a message
  naming the failing pair and the nearest acceptable hue.
- **AC-6.** *Given* `prefers-contrast: more`, *When* the app loads, *Then* a
  high-contrast theme is selected automatically and every route passes an axe
  contrast scan.
- **AC-7.** *Given* the AN.1 motion suite, *When* UX.1 merges, *Then*
  `npm run interface-polish:check` passes unchanged and no motion test is
  modified.
- **AC-8.** *Given* the token gallery route, *When* it is opened, *Then* every
  semantic token renders with its per-theme value and computed contrast ratio.
- **AC-9.** *Given* the built app, *When* the entry bundle is measured, *Then* the
  gzip delta attributable to tokens is ≤4 KB against
  `scripts/bundle-baseline.json`.
- **AC-10.** *Given* Percy/Playwright visual baselines captured before migration,
  *When* the migration lands, *Then* diffs are ≤2% per page except on the
  explicitly-listed contrast repairs.
- **AC-11.** *Given* `tokens.json`, *When* validated against the W3C Design Tokens
  Format Module schema, *Then* it conforms.

## 8. Data Model

Only org branding touches the database.

```sql
-- server/migrations/NNN_org_brand_tokens.sql
ALTER TABLE organizations
  ADD COLUMN brand_accent_oklch text,      -- e.g. 'oklch(0.55 0.18 264)'
  ADD COLUMN brand_tokens_version int NOT NULL DEFAULT 1;

-- CHECK enforces the serialised OKLCH shape; full validation is server-side.
ALTER TABLE organizations
  ADD CONSTRAINT organizations_brand_accent_oklch_fmt
  CHECK (brand_accent_oklch IS NULL OR brand_accent_oklch ~ '^oklch\(');
```

- **Indexes/constraints** — none needed beyond the format check; lookups are by
  existing org PK.
- **Backfill** — `NULL` means "use the product accent". No backfill required.
- **Migration naming** — follows the repo's `server/migrations/NNN_*.sql`
  convention.

## 9. API Surface

Extends the existing org-branding surface (`pages/lms/admin/org-branding.tsx`).

```ts
// GET /api/v1/orgs/{orgId}/branding   (auth: org admin; read: any org member)
type BrandingResponse = {
  accentOklch: string | null
  derivedRamp: Record<'50'|'100'|'200'|'300'|'400'|'500'|'600'|'700'|'800'|'900'|'950', string>
  tokensVersion: number
}

// PUT /api/v1/orgs/{orgId}/branding   (auth: org admin)
type BrandingRequest = { accentOklch: string | null }

// 422 when the ramp cannot satisfy AA:
type BrandingContrastError = {
  error: 'brand_accent_contrast'
  failingPairs: { fg: string; bg: string; ratio: number; required: number }[]
  suggestion: string | null   // nearest conforming OKLCH
}
```

- No WebSocket events.
- Rate limit: reuse the standard org-settings write limiter.
- **OpenAPI** — both routes MUST be documented; `make openapi-check` must pass.

## 10. UI / UX

- **New pages** — internal token gallery at `/design/tokens` (staff-gated, sits
  beside the existing `/type` route).
- **Modified pages** — `pages/lms/admin/org-branding.tsx` gains an accent picker
  with live contrast readout and an inline preview of nav/button/link.
- **Key user flows**
  1. Admin opens Org branding → picks accent → sees live contrast verdict per
     pair → saves → theme applies org-wide on next load.
  2. Any user with `prefers-contrast: more` → high-contrast theme auto-selected.
  3. Engineer opens `/design/tokens` → copies the semantic token name → uses it.
- **States** — Branding picker: loading (skeleton), empty (no brand set → product
  default preview), error (422 with named failing pair), offline (disabled with
  explanation).
- **Mobile/responsive** — Gallery and picker are single-column below `md`.
- **Accessibility** — Contrast readout is text + icon, never colour alone
  (SC 1.4.1). The picker is operable by keyboard with a hex/OKLCH text input, not
  only a colour surface.
- **Copy & i18n** — New keys under `common.branding.*` and `common.theme.*`;
  added to all four locale files at parity.

## 11. AI / ML Considerations

Not AI-touching. *(Optional follow-up, out of scope: an accent suggester that
proposes the nearest AA-conforming hue to a brand colour — pure computation, no
model required.)*

## 12. Integration Points

- **External** — none at runtime. Figma consumes `tokens.json` via the W3C DTFM
  format (design-time only).
- **Internal**
  - `clients/web/src/index.css` — token imports; `dark:` variant retired
  - `clients/web/src/styles/high-contrast.css` — reimplemented as a theme
  - `clients/web/src/styles/ui-modes/{k2,elementary}.css` — become token overrides
  - `clients/web/src/lib/ui-theme.ts` — theme application
  - `clients/web/contrast-config.json` — replaced by the derived validator
  - `clients/web/scripts/check-contrast.mjs` — rewritten against tokens
  - `clients/packages/tokens/` — new package
  - `server/internal/httpserver` — branding routes
  - `docs/design.md`, `docs/design-tokens.md` — rewritten
- **Events** — none.

## 13. Dependencies & Sequencing

- **Must ship after** — nothing. UX.1 is first.
- **Must ship before** — UX.2 (components consume component tokens), UX.3, UX.9,
  UX.10, UX.11, UX.12, UX.14, UX.18.
- **Shared infra** — none beyond existing CI.
- **Sequencing within UX.1**: tokens defined → validator green → codemod on one
  pilot directory (`components/layout/`) → measure → codemod remaining
  directories in reviewable batches (~30 files/PR) → lint rule flipped to error.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Codemod produces subtle visual regressions across 698 files | H | H | Playwright visual baselines on the top 40 routes before migration; ≤2% diff gate (AC-10); migrate in ~30-file PRs |
| `slate`→`neutral` collapse changes perceived hue in dark mode | M | M | Pick the unified ramp by measuring the *most-used* current values; review dark mode on the top 20 screens with design before merge |
| Merge conflicts against concurrent feature work in 698 files | H | M | Land in short-lived batches; announce a freeze window per directory; run the codemod, not hand edits, so conflicts re-resolve mechanically |
| Long tail of literals the codemod cannot map (62 arbitrary hex) | H | L | Codemod emits an unmapped report; triage list is finite and enumerated in the audit |
| OKLCH browser support | L | M | `@supports` fallback to sRGB values emitted at build time; baseline support is universal in our supported matrix |
| Org accent produces a ramp that is legal but ugly | M | L | Constrain chroma range; preview before save; admin can revert to product accent |
| Team keeps writing raw literals after migration | M | H | FR-8 lint rule as **error**, not warning; UX.18 tracks token purity as a standing metric |

## 15. Rollout Plan

- **Feature flag** — `ffSemanticTokens` gates only the *theme switching* and org
  branding behaviour. The token refactor itself is not flagged (it is
  behaviour-preserving and flagging 698 files would double the surface).
- **Sequencing**
  1. Token layers + validator + Tailwind `@theme` mapping (no consumer changes).
  2. Lint rule added as **warning**, allowlist auto-generated at current counts.
  3. Codemod pilot: `components/layout/` + `components/ui/`. Measure diffs.
  4. Rolling migration by directory; allowlist ratchets down each PR.
  5. Lint rule flipped to **error**; allowlist deleted.
  6. High-contrast themes replace the override sheet.
  7. Org branding behind `ffSemanticTokens`; dogfood on the internal org.
- **Dogfood** — internal Lextures org for two weeks with brand accent set.
- **GA criteria** — AC-1 through AC-11 green; zero token-related Sentry errors for
  14 days; design sign-off on the top 40 routes in all four themes.
- **Rollback** — Token CSS is additive; reverting the codemod PRs restores raw
  literals. Org branding rolls back by flag.

## 16. Test Plan

- **Unit** — token resolution per theme; OKLCH→sRGB conversion; contrast
  computation; accent ramp derivation; org accent validation and rejection.
- **Integration** — branding GET/PUT authz matrix (org admin vs member vs other
  org); 422 shape; theme application on load.
- **End-to-end** — Playwright: theme toggle across light/dark/HC on the top 10
  routes; org accent applied after save; `prefers-contrast: more` auto-selection.
- **Security** — CSS-injection attempts through `accentOklch` (`;`, `}`,
  `url()`, `expression()`); authz on branding write.
- **Accessibility** — axe contrast scan on the top 40 routes × 4 themes; manual
  NVDA + VoiceOver spot check that nothing regressed; `forced-colors` mode check.
- **Performance / load** — bundle-size gate via existing `check-bundle-size.mjs`;
  theme-switch measured at <16 ms with no layout shift (CLS 0).
- **Visual regression** — Playwright screenshot baselines, top 40 routes × 4
  themes, ≤2% diff.
- **Manual exploratory** — QA checklist covering k2 mode, elementary mode,
  reading view, print stylesheet, and the quiz focus shell in every theme.

## 17. Documentation & Training

- **End-user** — help-centre article: "Choosing a theme" and "High contrast mode".
- **Admin** — "Branding your organisation", including why some colours are
  rejected.
- **Engineer** — rewritten `docs/design-tokens.md`: the three layers, the naming
  grammar, how to add a token, how to consume one, what the lint rule enforces.
  Rewritten `docs/design.md` pointing at tokens rather than hex values.
- **API reference** — OpenAPI entries for the branding routes.
- **Runbook** — "A contrast check is failing CI": how to read the validator output
  and fix it.
- **AGENTS.md / CLAUDE.md** — add the rule: *never write a raw palette literal in
  `clients/web/src`.*

## 18. Open Questions

1. Do we keep `slate` or `neutral` as the base hue for the unified neutral ramp?
   (Recommendation: derive a new ramp in OKLCH tuned to the `Lextures` typeface's
   perceived weight, rather than adopting either wholesale.)
2. Should org branding allow a full secondary palette or accent-only in v1?
   (Recommendation: accent-only — it is the 90% case and bounds the contrast
   problem.)
3. Do the k2/elementary UI modes need distinct *hues* or only distinct
   size/spacing/motion? Requires a call with the K-12 product owner.
4. Does the desktop client (`clients/desktop`) consume web CSS directly, and does
   it therefore inherit this for free?
5. What is the deprecation policy for `contrast-config.json` — delete on flip, or
   keep one release as a cross-check?

## 19. References

- Existing files: `clients/web/src/index.css`,
  `clients/web/src/styles/high-contrast.css`,
  `clients/web/src/styles/ui-modes/*.css`, `clients/web/src/lib/ui-theme.ts`,
  `clients/web/contrast-config.json`, `clients/web/scripts/check-contrast.mjs`,
  `clients/web/src/pages/lms/admin/org-branding.tsx`
- Research: [research.md](research.md) R-1, R-3, R-24, R-25, R-26, R-27, §9
- Audit: [audit.md](audit.md) G-1, G-14, G-16
- External: [W3C Design Tokens Format Module](https://tr.designtokens.org/),
  [WCAG 2.2](https://www.w3.org/TR/WCAG22/),
  [Tailwind CSS v4 `@theme`](https://tailwindcss.com/docs/theme)
- Related plans: [UX.2](UX.2-core-component-library-and-adoption-ratchet.md),
  [UX.3](UX.3-typography-and-reading-system.md),
  [UX.18](UX.18-design-system-governance-and-measurement.md),
  `../../completed/animations/` (AN.1 motion tokens),
  `../../completed/12-accessibility/`
