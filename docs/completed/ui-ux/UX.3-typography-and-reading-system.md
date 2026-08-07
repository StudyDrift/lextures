# UX.3 — Typography and Reading System

> Implementation plan. Source: [audit.md](../../plan/ui-ux/audit.md) §2 G-7.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.3 |
| **Section** | UI/UX — Foundations |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE — type roles, lint ratchet, measure, textScale prefs, ffTypeScale (2026-03) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Design Systems |
| **Depends on** | UX.1 |
| **Unblocks** | UX.9, UX.10, UX.11, UX.14 |

---

## 1. Problem Statement

The product's body text is **14px**, with **2,775 instances of 12px or smaller**
(`text-xs` 2,546; `text-[10px]`/`text-[11px]` 229), against **168** uses of
`text-base` (16px) across 795 files. The most common secondary-text colour,
`text-slate-500`, sits at ~4.76:1 on white — barely above the AA floor — and is
routinely applied at 12px. There is no type scale, no semantic type roles, and no
line-length constraint on long-form course content. For a product whose primary
users include K-12 students, learners with dyslexia and learners with low vision,
this is the wrong default at the wrong scale, and it compounds the extraneous
cognitive load documented in **R-1/R-2**.

## 2. Goals

- Establish a **semantic type scale** with named roles, so authors choose meaning
  (`text-body`, `text-caption`) rather than size.
- Raise the default body size to **16px** and eliminate sub-12px text from
  non-decorative contexts.
- Constrain measure (line length) on long-form reading surfaces to 60–75
  characters.
- Guarantee that small type is never paired with near-floor contrast.
- Make the shipped reading-preference controls (font, spacing, ruler) operate on
  the type system rather than beside it.

## 3. Non-Goals

- Changing the `Lextures` typeface or commissioning new weights.
- Redesigning any page layout (that is UX.9/UX.10).
- Localised typographic tuning for CJK scripts (not currently shipped).
- The marketing site `www/` — it may legitimately use expressive display type.

## 4. Personas & User Stories

- **As a student with dyslexia**, I want body copy at a comfortable size with
  adjustable spacing so that I can read a module without fatigue.
- **As a student on a laptop**, I want course content to stop at a readable line
  length instead of stretching to 1,600px.
- **As an instructor scanning a gradebook**, I want dense tabular type that is
  still legible, with numerals that align.
- **As a low-vision user**, I want to zoom to 200% without content being clipped
  or overlapping.
- **As an engineer**, I want `text-caption` to mean something so that I stop
  guessing between `text-xs` and `text-[11px]`.

## 5. Functional Requirements

- **FR-1.** The system MUST define a type scale as UX.1 tokens with these roles:

  | Role | Size / line-height | Use |
  |---|---|---|
  | `display` | 36 / 40 | Marketing-adjacent hero moments only |
  | `title-lg` | 28 / 34 | Page title |
  | `title` | 22 / 28 | Section title |
  | `subtitle` | 18 / 26 | Card / panel heading |
  | `body-lg` | 18 / 28 | Long-form course content |
  | `body` | **16 / 24** | Default UI and prose |
  | `body-sm` | 14 / 20 | Dense UI, table cells, secondary rows |
  | `caption` | 13 / 18 | Metadata, timestamps, helper text — **floor** |
  | `overline` | 12 / 16 | Uppercase section labels only, never sentences |
  | `code` | 14 / 22 | Monospace |

- **FR-2.** `body` (16px) MUST be the inherited default for `.lms-scope`.
- **FR-3.** No user-visible **sentence or paragraph** may render below 13px.
  `overline` at 12px is permitted only for short uppercase labels.
- **FR-4.** Long-form reading surfaces (content pages, syllabus, discussions,
  assignment descriptions, notebook) MUST constrain measure to `65ch`
  (min 45ch, max 75ch).
- **FR-5.** Type roles MUST be exposed as Tailwind utilities (`text-body`,
  `text-caption`, …). A lint rule MUST forbid raw `text-xs`/`text-sm`/
  `text-[Npx]` in `src/**/*.tsx`, with a ratcheting allowlist.
- **FR-6.** Tabular data MUST use `font-variant-numeric: tabular-nums`.
- **FR-7.** Any type role at or below `caption` MUST be restricted by the UX.1
  contrast validator to foreground tokens meeting **≥7:1** in the default themes,
  not merely 4.5:1.
- **FR-8.** The system MUST preserve and extend `text-wrap: balance` on headings
  and `text-wrap: pretty` on body copy (already present in `index.css`).
- **FR-9.** The existing `ReadingPreferencesPanel` (font family, letter spacing,
  word spacing, line height) MUST drive the type tokens via CSS variables so
  preferences apply across **all** learning surfaces, not only the reading view.
- **FR-10.** The system MUST support text resize to **200%** (WCAG 1.4.4) and
  text spacing overrides (WCAG 1.4.12: line-height 1.5×, paragraph 2×,
  letter 0.12em, word 0.16em) without loss of content or function.
- **FR-11.** `k2` and `elementary` UI modes MUST scale the type ramp via token
  override (larger base, looser leading), not bespoke rules.
- **FR-12.** Headings MUST form a correct document outline; a lint/test check
  SHOULD flag skipped heading levels on route-level components.

## 6. Non-Functional Requirements

- **Performance** — No additional font files. `font-display: swap` retained.
  Type token CSS ≤1 KB gzip. Raising the base size MUST NOT introduce CLS —
  measured at 0 on the top 20 routes.
- **Security** — None applicable.
- **Privacy & Compliance** — Delivers WCAG 2.1 SC 1.4.4 (Resize Text), 1.4.12
  (Text Spacing), 1.4.8 (Visual Presentation, AAA — partially), and supports the
  `docs/vpat/` claim.
- **Accessibility** — Primary driver of this plan; see FR-3, FR-7, FR-10.
- **Scalability** — Adding a role requires editing one token file.
- **Reliability** — Migration is mechanical; a codemap from current classes to
  roles is committed and reviewable.
- **Observability** — CI emits `type_role_violations` and
  `sub_13px_text_instances`.
- **Maintainability** — Type roles documented in `docs/design-tokens.md`.
- **Internationalization** — Arabic (`ar`) requires a larger effective size for
  equivalent legibility; the token layer MUST allow a per-locale size multiplier.
- **Backward compatibility** — Density increases where 14px→16px. Layouts that
  break MUST be fixed, not exempted; any role downgrade needs written
  justification in the PR.

## 7. Acceptance Criteria

- **AC-1.** *Given* the migrated codebase, *When* the type lint runs, *Then* there
  are **0** raw `text-xs`/`text-sm`/`text-[Npx]` occurrences in `src/**/*.tsx` and
  the allowlist is empty.
- **AC-2.** *Given* any rendered page, *When* every text node's computed
  `font-size` is sampled, *Then* no node containing a sentence renders below 13px.
- **AC-3.** *Given* a course content page at 1,920px viewport, *When* measured,
  *Then* the prose column is between 45ch and 75ch.
- **AC-4.** *Given* browser zoom at 200%, *When* the top 20 routes are exercised,
  *Then* no content is clipped, overlapped or made inoperable.
- **AC-5.** *Given* the WCAG text-spacing bookmarklet values, *When* applied,
  *Then* no content is lost or functionality broken on the top 20 routes.
- **AC-6.** *Given* the contrast validator, *When* a `caption`-role token is
  paired with a foreground below 7:1, *Then* the build fails.
- **AC-7.** *Given* a user sets line height 2.0 in Reading Preferences, *When*
  they open a module, an assignment, a discussion and the notebook, *Then* the
  preference applies in all four.
- **AC-8.** *Given* the gradebook, *When* numeric columns render, *Then* digits
  are tabular and vertically aligned.
- **AC-9.** *Given* the top 20 routes, *When* CLS is measured after the base-size
  change, *Then* CLS ≤0.02 at p75.
- **AC-10.** *Given* `ar` locale, *When* a content page renders, *Then* the
  locale size multiplier applies and measure constraints hold in RTL.

## 8. Data Model

One additive column to persist reading preferences server-side so they follow a
user across devices (today they are client-local).

```sql
-- server/migrations/NNN_user_reading_preferences.sql
ALTER TABLE users
  ADD COLUMN reading_preferences jsonb NOT NULL DEFAULT '{}'::jsonb;
```

- Shape: `{ fontFamily, letterSpacing, wordSpacing, lineHeight, textScale }`.
- **Backfill** — none; `{}` means "product defaults".
- **Constraint** — validated server-side against an allowlist of values; no
  free-form CSS is accepted.

## 9. API Surface

```ts
// GET /api/v1/settings/reading-preferences   (auth: self)
// PUT /api/v1/settings/reading-preferences   (auth: self)
type ReadingPreferences = {
  fontFamily: 'lextures' | 'system' | 'serif' | 'mono' | 'dyslexic'
  letterSpacing: 'normal' | 'wide' | 'wider'
  wordSpacing: 'normal' | 'wide'
  lineHeight: 1.5 | 1.75 | 2.0
  textScale: 1.0 | 1.125 | 1.25 | 1.5
}
```

- Rejects any value outside the enums with `400`.
- No WebSocket events. Standard per-user settings rate limit.
- **OpenAPI** — both routes MUST be documented; `make openapi-check` must pass.

## 10. UI / UX

- **New pages** — none. The `/design/tokens` gallery from UX.1 gains a type
  specimen section.
- **Modified pages** — `ReadingPreferencesPanel` gains a text-scale control and a
  "applies everywhere" note; effectively all content surfaces inherit new sizes.
- **Key user flows**
  1. Learner opens `Aa` in the top bar → adjusts size/spacing/font → change is
     immediate and persists across devices.
  2. Author writes prose in a content page → measure is constrained automatically.
- **States** — Reading panel: loading (skeleton), error (save failed → retry, with
  local preference still applied), offline (applies locally, syncs later).
- **Mobile/responsive** — the scale uses `clamp()` between 390px and 1280px so
  `title-lg` does not overwhelm small screens.
- **Accessibility** — the panel is a labelled dialog with a live preview; each
  control announces its current value; changes are announced via the existing
  `live-region.tsx`.
- **Copy & i18n** — new keys under `common.reading.*`, added to all four locales at
  parity.

## 11. AI / ML Considerations

Not AI-touching. *(Note for UX.16: AI-generated content rendered into these
surfaces inherits the measure and scale constraints automatically — no separate
handling required.)*

## 12. Integration Points

- **External** — none. If an OpenDyslexic-style face is added for
  `fontFamily: 'dyslexic'`, licensing MUST be cleared first (see §18).
- **Internal**
  - `clients/web/src/index.css`, `src/styles/tokens/*` — scale definition
  - `clients/web/src/components/a11y/ReadingPreferencesPanel.tsx`
  - `clients/web/src/styles/ui-modes/{k2,elementary}.css`
  - `clients/web/src/components/markdown/**`, `components/content-page/**`,
    `components/syllabus/**` — measure constraints
  - `clients/web/src/pages/lms/gradebook/**` — tabular numerals
  - `server/internal/httpserver` — settings routes
- **Events** — none.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.1](./UX.1-semantic-design-token-system.md) (roles are
  tokens).
- **Must ship before** — UX.9, UX.10, UX.11, UX.14 (all lay out against the scale).
- **Runs in parallel with** — UX.2.
- **Shared infra** — none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| 14px→16px breaks dense layouts (gradebook, enrollments, admin tables) | **H** | M | Dense surfaces legitimately use `body-sm` (14px); the change is to the *default*, not to every surface. Audit the 10 densest screens explicitly |
| Increased size causes overflow/truncation in fixed-width chrome | H | M | Visual regression on top 40 routes; sidebar and top bar audited first |
| Team treats `body-sm` as the new default, re-creating the problem | M | H | Lint rule requires justification comment for `body-sm` outside tables/dense lists; UX.18 tracks the ratio |
| Per-locale multiplier causes RTL layout breakage | M | M | RTL visual baselines in the gallery; `ar` included in the top-20 route sweep |
| CLS regression from font-size shift during load | M | M | Sizes are set in CSS, not JS; `font-display: swap` with metric-compatible fallback; AC-9 gate |

## 15. Rollout Plan

- **Feature flag** — `ffTypeScale` gates the **base-size raise only** (16px
  default), so it can be reverted independently of the token/role migration.
  Roles and lint ship unflagged.
- **Sequencing**
  1. Type tokens + roles + specimen in the gallery.
  2. Codemod current classes → roles (mechanical, size-preserving). Nothing
     visibly changes.
  3. Lint rule as warning; allowlist generated.
  4. Measure constraints on long-form surfaces.
  5. Flip `ffTypeScale` for the internal org → 2 weeks → 10% → GA.
  6. Lint flipped to error; allowlist deleted.
  7. Reading-preference persistence.
- **Dogfood** — internal org with `ffTypeScale` on.
- **GA criteria** — AC-1…AC-10 green; no increase in truncation-related support
  tickets; design sign-off on the 10 densest screens.
- **Rollback** — `ffTypeScale` off restores 14px base; roles remain (harmless).

## 16. Test Plan

- **Unit** — role→CSS resolution; `clamp()` boundaries; locale multiplier;
  preference validation (valid + rejected values).
- **Integration** — preferences GET/PUT authz (self only); persistence across
  sessions; local-first application when offline.
- **End-to-end** — Playwright: set each preference → assert computed styles on
  content page, assignment, discussion, notebook; 200% zoom sweep; text-spacing
  override sweep.
- **Security** — attempt to inject arbitrary CSS via preference values; assert
  server rejection and that no value reaches a style attribute unvalidated.
- **Accessibility** — automated: axe + a custom rule asserting no sentence node
  <13px (AC-2); manual: NVDA/VoiceOver read-through of a content page at each
  preference combination; zoom and reflow (WCAG 1.4.10) at 320px equivalent.
- **Performance / load** — CLS and LCP on the top 20 routes before/after
  `ffTypeScale`; bundle delta gate.
- **Manual exploratory** — QA checklist: gradebook, enrollments, admin tables,
  quiz taking, live quiz presenter view, print stylesheet, k2 and elementary modes.

## 17. Documentation & Training

- **End-user** — help-centre: "Making text easier to read" covering the `Aa`
  panel and browser zoom.
- **Admin / instructor** — note in the course-design guide that authored content
  inherits measure constraints.
- **Engineer** — `docs/design-tokens.md` type section: the roles, when to use
  each, the 13px floor rule, the `body-sm` justification requirement.
- **API reference** — OpenAPI for reading-preferences routes.
- **Runbook** — "Type lint failed: choosing the right role".
- **Update** `docs/design.md` — its "Plus Jakarta Sans or Inter" line is factually
  wrong today and must be replaced with the `Lextures` stack and the role table.

## 18. Open Questions

1. Do we license a dyslexia-optimised face for `fontFamily: 'dyslexic'`, or map it
   to a system fallback? Evidence for such faces is contested; spacing controls
   may matter more. Needs an accessibility-lead decision.
2. What is the correct `ar` size multiplier? Requires review by an Arabic-reading
   speaker; 1.05–1.1 is the usual starting point.
3. Should `body-lg` (18px) be the default for *learner-facing course content*
   while `body` (16px) stays the default for chrome? *Recommendation: yes — test
   in dogfood.*
4. Does the K-12 product owner want a larger base in `elementary` mode, or only in
   `k2`?
5. Do reading preferences belong in `users` or in the existing settings store? To
   be confirmed against `server/internal/repos` conventions.

## 19. References

- Existing files: `clients/web/src/index.css` (lines defining
  `--reading-*` variables and `text-wrap` rules),
  `clients/web/src/components/a11y/ReadingPreferencesPanel.tsx`,
  `clients/web/src/components/a11y/ReadingRuler.tsx`,
  `clients/web/src/styles/ui-modes/k2.css`,
  `clients/web/src/styles/ui-modes/elementary.css`,
  `clients/web/public/fonts/lextures-*.woff2`,
  `clients/web/src/pages/typeface-page.tsx`
- Research: [research.md](../../plan/ui-ux/research.md) R-1, R-2, R-3, §9
- Audit: [audit.md](../../plan/ui-ux/audit.md) G-7, G-1
- External: [WCAG 2.2 SC 1.4.4 / 1.4.12 / 1.4.10](https://www.w3.org/TR/WCAG22/)
- Related plans: [UX.1](./UX.1-semantic-design-token-system.md),
  [UX.10](../../plan/ui-ux/UX.10-course-home-and-learning-flow.md),
  [UX.11](../../plan/ui-ux/UX.11-data-table-and-gradebook-system.md),
  `../../completed/12-accessibility/`
