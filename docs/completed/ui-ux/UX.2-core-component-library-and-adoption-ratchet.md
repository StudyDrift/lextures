# UX.2 — Core Component Library and Adoption Ratchet

> Implementation plan. Source: [audit.md](../../plan/ui-ux/audit.md) §1, §2 G-2.
> **Shipped** as the core library + gallery + coverage/allowlist ratchet + pilot
> adoptions (`EmptyState` → `Button`, `ConfirmDialog` → `AlertDialog`,
> `Dialog` → `OverlaySurface`). Full call-site migration continues under
> `npm run ds:coverage` / `raw-interactive-allowlist.json` (counts may only
> decrease; coverage may only increase). Guide: [docs/guides/component-library.md](../../guides/component-library.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.2 |
| **Section** | UI/UX — Foundations |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | SHIPPED (library + ratchet; product migration ongoing) |
| **Estimated effort** | XL (>2mo) |
| **Owner (proposed)** | Design Systems |
| **Depends on** | UX.1 |
| **Unblocks** | UX.4, UX.6, UX.9, UX.10, UX.11, UX.12, UX.13, UX.14, UX.18 |

---

## 1. Problem Statement

Lextures has a component library that nobody uses. `components/ui/button.tsx` is a
well-built control with four variants, loading state, `aria-busy` and reduced-
motion handling — and it is imported by **2 of 795 component files**, against
**2,016 hand-rolled `<button>` elements**. `components/ui/overlay-surface.tsx` has
**zero** importers while **129 files** hand-roll `role="dialog"`. The library does
not even compose with itself: `empty-state.tsx` re-implements button styling
inline rather than importing `Button`. The result is that the product's quality
ceiling is set not by the design system but by whatever each file happened to
hand-roll, and every accessibility, theming, motion and copy fix must be applied
2,016 times instead of once.

## 2. Goals

- Ship a **complete, accessible, token-driven core library** covering the
  interaction primitives the product actually uses.
- Migrate the codebase to it, taking **design-system coverage from ~0.25% to
  ≥90%** of interactive elements.
- Make every accessibility contract (focus, keyboard, ARIA, target size) a
  property of the component, so it cannot be forgotten at a call site.
- Install a **ratchet**: coverage can only go up, enforced in CI.
- Delete `overlay-surface.tsx`-style dead primitives by either adopting or
  removing them — no component ships without a migration path.

## 3. Non-Goals

- Building a *public* / open-source design system. This is internal.
- Redesigning any component's appearance beyond what UX.1 tokens imply.
- Replacing domain components (gradebook grid, quiz panel, editor) — those are
  UX.10/UX.11 and consume this library.
- Replacing TipTap, CodeMirror, xyflow, dnd-kit or other third-party UI. They are
  wrapped, not rewritten.
- Migrating native clients.

## 4. Personas & User Stories

- **As a web engineer**, I want `<Button variant="danger" loading>` so that I
  never re-derive focus rings, disabled states, spinners or haptics.
- **As a new engineer**, I want one gallery showing every component and its props
  so that I do not copy-paste from whichever file I happened to open.
- **As a keyboard user**, I want every menu, tab set and dialog in the product to
  behave identically, because they are literally the same component.
- **As a screen-reader user**, I want a dialog to trap focus everywhere, not in
  the three places someone remembered.
- **As a designer**, I want a change to the secondary button to ship everywhere at
  once.
- **As a QA engineer**, I want to test the component once instead of testing 2,016
  buttons.

## 5. Functional Requirements

- **FR-1.** The library MUST live at `clients/web/src/components/ui/` with a
  single barrel export, and MUST consume UX.1 semantic/component tokens
  exclusively.
- **FR-2.** The library MUST provide, at minimum:

  | Group | Components |
  |---|---|
  | Actions | `Button`, `IconButton`, `LinkButton`, `ButtonGroup`, `SplitButton` |
  | Forms | `Field`, `Input`, `Textarea`, `Select`, `Combobox`, `Checkbox`, `Radio`, `RadioGroup`, `Switch`, `SegmentedControl`, `DatePicker`, `FileInput`, `Fieldset` |
  | Overlays | `Dialog`, `AlertDialog`, `Sheet`, `Drawer`, `Popover`, `Tooltip`, `Menu`, `ContextMenu` |
  | Navigation | `Tabs`, `Breadcrumbs`, `Pagination`, `NavLink`, `Disclosure` |
  | Display | `Card`, `Badge`, `Avatar`, `Tag`, `Callout`, `Separator`, `ProgressBar`, `Meter`, `Table` (primitives), `DescriptionList` |
  | Feedback | `Toast`, `EmptyState`, `Skeleton`, `Spinner`, `ErrorState`, `InlineAlert` |
  | Layout | `Stack`, `Inline`, `Grid`, `PageHeader`, `Section`, `Toolbar` |

- **FR-3.** Every interactive component MUST implement the full **WAI-ARIA APG**
  keyboard contract for its pattern — including roving `tabindex`, arrow keys,
  `Home`/`End`, typeahead where specified, and Escape.
- **FR-4.** `Dialog`, `AlertDialog`, `Sheet` and `Drawer` MUST trap focus, move
  focus in on open, restore focus to the trigger on close, mark background
  content `inert`, and close on Escape. Focus trapping MUST use the existing
  `lib/a11y/focus-trap.ts`.
- **FR-5.** Every component MUST meet **WCAG 2.2 SC 2.5.8** — a minimum 24×24 CSS
  px target — by construction, and MUST expose a `size` prop whose smallest value
  still satisfies it.
- **FR-6.** Every component MUST render a visible, token-driven focus indicator
  meeting 3:1 against adjacent colours (SC 1.4.11, 2.4.7).
- **FR-7.** All user-visible strings in components MUST come from props or i18n
  keys. No hardcoded English.
- **FR-8.** Components MUST support `dir="rtl"` using logical properties only.
- **FR-9.** Components MUST integrate the AN.1 motion tokens and respect
  `prefers-reduced-motion` and the `data-motion-controls` kill switch.
- **FR-10.** A **coverage analyser** MUST compute
  `design-system-coverage = (interactive elements rendered by ui/* components) /
  (all interactive elements)` from the source tree, and MUST fail CI if coverage
  decreases relative to a committed baseline.
- **FR-11.** Lint rules MUST forbid, in `src/**` outside `components/ui/`:
  raw `<button>`, raw `<input>`/`<select>`/`<textarea>`, `role="dialog"`,
  `role="menu"`, `role="tablist"`, and `title=` used as a tooltip. Each rule ships
  with a ratcheting allowlist.
- **FR-12.** Every component MUST appear in an interactive **component gallery**
  route with props, variants, states (default/hover/focus/disabled/loading/error),
  RTL preview, and all four themes.
- **FR-13.** The library MUST NOT contain a component with zero importers for more
  than one release — such components MUST be adopted or deleted.
- **FR-14.** `Button` MUST remain API-compatible with today's props (`variant`,
  `static`, `loading`) so the 2 existing call sites do not break.
- **FR-15.** The library SHOULD be tree-shakeable; importing `Button` MUST NOT
  pull in `DatePicker`.

## 6. Non-Functional Requirements

- **Performance** — The core set (`Button`, `Field`, `Input`, `Card`, `Badge`,
  `Skeleton`) MUST add ≤12 KB gzip to the entry bundle. Heavy components
  (`DatePicker`, `Combobox`, `Table`) MUST be separately chunked. Net entry bundle
  MUST **decrease** post-migration as duplication is removed.
- **Security** — Components rendering user content MUST NOT use
  `dangerouslySetInnerHTML` except in the existing sanitised markdown path.
- **Privacy & Compliance** — Delivers WCAG 2.1/2.2 AA and EN 301 549 obligations
  structurally; directly supports the `docs/vpat/` claim.
- **Accessibility** — Every component ships with axe-clean tests and a documented
  keyboard contract. This is the acceptance bar, not a follow-up.
- **Scalability** — Adding a component requires: implementation, tests, gallery
  entry, docs. Enforced by a CI check that every export appears in the gallery.
- **Reliability** — Codemods MUST be idempotent and emit an unmigrated report.
- **Observability** — CI emits `design_system_coverage`,
  `raw_button_count`, `raw_dialog_count`, `aria_contract_coverage`.
- **Maintainability** — One component per file; no component file over 300 lines;
  conforms to `docs/ARCHITECTURE_CONVENTIONS.md` budgets.
- **Internationalization** — Gallery renders every component in `ar` (RTL) as a
  standing check.
- **Backward compatibility** — Migration is mechanical and behaviour-preserving.
  Where a hand-rolled control had a *bug* the component fixes (e.g. missing focus
  trap), that is an intentional change and MUST be noted in the PR.

## 7. Acceptance Criteria

- **AC-1.** *Given* the migrated codebase, *When* the coverage analyser runs,
  *Then* `design-system-coverage ≥ 90%` and the CI gate fails on any decrease.
- **AC-2.** *Given* `npm run lint`, *When* it runs on `src/**` outside
  `components/ui/`, *Then* raw `<button>` count is **0** and the allowlist is
  empty.
- **AC-3.** *Given* any `Dialog` in the product, *When* it opens, *Then* focus
  moves inside, Tab cycles within it, background is `inert`, Escape closes it, and
  focus returns to the trigger — verified by an E2E test that runs against **every**
  dialog registered in the gallery.
- **AC-4.** *Given* any `Tabs`, *When* the user presses `ArrowRight`/`ArrowLeft`/
  `Home`/`End`, *Then* selection moves per the WAI-ARIA APG and `tabindex` roves.
- **AC-5.** *Given* any `Menu`, *When* opened, *Then* focus enters the first item
  and arrow keys, `Home`/`End`, typeahead and Escape all behave per the APG.
- **AC-6.** *Given* the component gallery, *When* axe runs across every component
  × 4 themes × 2 directions, *Then* there are **0** violations.
- **AC-7.** *Given* every component with a pointer target, *When* measured, *Then*
  the effective target is ≥24×24 CSS px in its smallest size.
- **AC-8.** *Given* the built app post-migration, *When* the entry bundle is
  measured, *Then* it is **≤ the pre-migration baseline** of 245,104 B gzip.
- **AC-9.** *Given* the library exports, *When* the gallery-coverage check runs,
  *Then* every exported component has a gallery entry and ≥1 importer outside
  `components/ui/`.
- **AC-10.** *Given* visual baselines captured pre-migration, *When* migration
  lands, *Then* per-page diff is ≤3% except on the explicitly-listed a11y repairs.
- **AC-11.** *Given* `components/ui/empty-state.tsx`, *When* inspected, *Then* it
  imports `Button` rather than re-implementing button styles.

## 8. Data Model

None. UX.2 is entirely client-side. No tables, columns, enums, indexes,
migrations or backfill.

## 9. API Surface

None. No new or changed HTTP routes, no WebSocket events, no rate-limit or quota
considerations, no OpenAPI changes.

## 10. UI / UX

- **New pages** — component gallery at `/design/components` (staff-gated,
  alongside `/design/tokens` from UX.1). Routes are excluded from the production
  sitemap.
- **Modified pages** — ultimately most of the 795 component files, mechanically.
- **Key user flows**
  1. Engineer opens the gallery → finds the component → copies the usage snippet.
  2. Engineer writes a raw `<button>` → lint fails with the replacement named.
  3. Reviewer opens a PR → the coverage delta is reported in the CI summary.
- **States** — every component documents default, hover, focus-visible, active,
  disabled, loading, error, empty and read-only in the gallery.
- **Mobile/responsive** — components are responsive by default; the gallery
  renders each at 390 / 768 / 1280 px.
- **Accessibility annotations** — each gallery entry states the ARIA pattern
  implemented, the keyboard contract, and the focus-order guarantee.
- **Copy & i18n** — components take strings as props. Gallery copy lives under
  `common.gallery.*` and is `en`-only (staff tool) but MUST not break the i18n
  parity check.

## 11. AI / ML Considerations

Not AI-touching. AI-specific surfaces (`components/ai/`, tutor panel, grader
agent) consume this library like any other feature area.

## 12. Integration Points

- **External** — none added. Third-party UI (`@dnd-kit`, `@tiptap`,
  `@uiw/react-codemirror`, `@xyflow/react`, `sonner`, `hls.js`, `pdfjs-dist`) is
  wrapped behind library components where it surfaces interaction primitives.
- **Internal**
  - `clients/web/src/components/ui/**` — the library
  - `clients/web/src/lib/a11y/focus-trap.ts` — consumed by overlays
  - `clients/web/src/lib/control-motion.ts`, `lib/motion.ts`,
    `lib/overlay-motion.ts` — consumed for AN.1 motion
  - `clients/web/src/components/use-confirm.tsx` — refactored onto `AlertDialog`
  - `clients/web/src/lib/lms-toast.ts` — becomes the `Toast` surface (see UX.13)
  - `clients/web/eslint-rules/` + `.oxlintrc.json` — new rules
  - `clients/web/scripts/check-design-system-coverage.mjs` — new
- **Events** — none.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.1](UX.1-semantic-design-token-system.md).
- **Must ship before** — UX.4 (ARIA remediation is *delivered by* migrating to
  these components), UX.6, UX.9, UX.10, UX.11, UX.12, UX.13, UX.14.
- **Runs in parallel with** — [UX.3](../../plan/ui-ux/UX.3-typography-and-reading-system.md).
- **Shared infra** — none beyond CI.
- **Internal sequencing**: overlays and forms first (they carry the accessibility
  debt), then actions, then display/layout. Migrate **by directory**, highest-
  traffic first: `components/layout/` → `pages/lms/dashboard.tsx` →
  `components/settings/` → `pages/lms/` → the long tail.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Migration of ~700 files stalls at 60% and the codebase ends with two systems | **H** | **H** | Ratchet (FR-10) makes regression impossible; per-directory ownership with named owners; coverage published weekly in UX.18 |
| Codemods cannot handle bespoke call sites | H | M | Codemod handles the mechanical 80%; emits a triage list; budget explicit manual time for the remainder |
| Merge conflicts with concurrent feature work | H | M | Small per-directory PRs; announce directory freezes; land continuously rather than in a big bang |
| New components are over-engineered before anyone needs them | M | M | FR-13: no component survives a release with zero importers. Build from real call sites, not speculation |
| Component API churn breaks call sites mid-migration | M | M | API freeze per component once it has >20 importers; changes go through a deprecation cycle |
| Bundle grows instead of shrinking | M | M | AC-8 gate; heavy components separately chunked; measure at each batch |
| "Design system tax" perception slows feature delivery | M | H | Pair each migration batch with a visible win (a11y fix, dark-mode fix); publish time-saved evidence in UX.18 |

## 15. Rollout Plan

- **Feature flag** — none. This is a refactor; flagging two implementations of
  every control would double the surface and the risk. Safety comes from
  behaviour-preserving codemods, visual regression gates, and small batches.
- **Sequencing**
  1. Library skeleton + gallery + coverage analyser (baseline recorded, no
     migration).
  2. Overlays (`Dialog`, `AlertDialog`, `Sheet`, `Popover`, `Menu`, `Tooltip`) —
     built and unit/axe-tested.
  3. Migrate `components/layout/` as the pilot. Measure diff, bundle, coverage.
  4. Forms group; migrate `components/settings/` (48 files, high form density).
  5. Actions/display/layout groups; rolling migration by directory.
  6. Lint rules flipped from warning to **error**, allowlists deleted.
  7. Dead primitives (`overlay-surface.tsx`) deleted or adopted.
- **Dogfood** — internal org runs each migrated directory for one week before the
  next batch.
- **GA criteria** — AC-1…AC-11 green; two consecutive weeks with no
  component-attributed Sentry regressions.
- **Rollback** — per-batch PR revert. Because batches are directory-scoped and
  behaviour-preserving, reverting one does not block the others.

## 16. Test Plan

- **Unit** — every component: rendering, all variants and states, prop contracts,
  controlled/uncontrolled behaviour, RTL, reduced motion. Vitest +
  `@testing-library/react`.
- **Keyboard contract tests** — per ARIA pattern, a shared conformance suite
  asserting the APG contract; each overlay/menu/tabs component runs it.
- **Integration** — `useConfirm` on `AlertDialog`; toast host; form composition
  (`Field` + `Input` + error) with real validation.
- **End-to-end** — Playwright: the gallery drives an automated sweep asserting
  focus trap, focus restore, Escape and arrow keys for every registered overlay
  and menu (AC-3/AC-4/AC-5).
- **Security** — verify no component introduces an unsanitised HTML sink; check
  that `Tooltip`/`Popover` content cannot escape its container.
- **Accessibility** — axe on every gallery entry × 4 themes × LTR/RTL (AC-6);
  manual NVDA (Windows/Firefox) and VoiceOver (macOS/Safari, iOS) scripts for
  Dialog, Menu, Tabs, Combobox, Toast; target-size measurement (AC-7).
- **Performance / load** — `check-bundle-size.mjs` per batch; React Profiler on
  `Table` and `Combobox` with 1,000 rows/options; INP measured on the gradebook
  after migration.
- **Visual regression** — Playwright screenshots of every gallery entry ×
  4 themes; per-page baselines on the top 40 product routes (AC-10).
- **Manual exploratory** — QA checklist per migrated directory covering the
  states matrix in §10.

## 17. Documentation & Training

- **End-user** — none (no user-visible feature).
- **Admin / instructor** — none.
- **Engineer** — `docs/guides/component-library.md`: what exists, how to choose,
  how to add a component, how the ratchet works, how to read a coverage failure.
  Each gallery entry carries inline usage docs and the keyboard contract.
- **API reference** — n/a.
- **Runbook** — "Coverage check failed on my PR" and "My component has no
  importers and CI is failing".
- **AGENTS.md / CLAUDE.md** — add: *use `components/ui/*`; never hand-roll a
  button, dialog, menu or tab set.*

## 18. Open Questions

1. Build on a headless primitive library (Radix / Ark / React Aria) or from
   scratch? *Recommendation: **React Aria Components** — it supplies the APG
   keyboard contracts that are precisely our G-3/G-4 debt, is unstyled so UX.1
   tokens drive appearance, and has first-class RTL. Cost: ~15 KB gzip for the
   core set. Decision needed before implementation starts.*
2. Is the gallery Storybook or a route inside the app? *Recommendation: an in-app
   route — it inherits real providers, themes, i18n and routing, and avoids a
   second build pipeline. Revisit if design needs standalone publishing.*
3. What is the exact denominator for `design-system-coverage`? Source-level
   element counting is cheap but crude; the Preply "visual coverage" approach
   (R-27) is more honest but needs runtime instrumentation. *Recommendation: ship
   source-level in UX.2, evaluate visual coverage in UX.18.*
4. Do we wrap `sonner` or replace it? (UX.13 depends on the answer.)
5. Should `Table` be a primitive set (`Table.Root/Head/Row/Cell`) or a data-driven
   component? UX.11 needs the latter; this plan should ship the former and let
   UX.11 build `DataTable` on top.
6. What is the deprecation window for a component API change once it has >100
   importers?

## 19. References

- Existing files: `clients/web/src/components/ui/` (all 19 files, especially
  `button.tsx`, `empty-state.tsx`, `overlay-surface.tsx`,
  `lms-content-skeletons.tsx`), `clients/web/src/lib/a11y/focus-trap.ts`,
  `clients/web/src/lib/control-motion.ts`, `clients/web/src/components/use-confirm.tsx`,
  `clients/web/src/lib/lms-toast.ts`, `clients/web/eslint-rules/`,
  `clients/web/.oxlintrc.json`
- Research: [research.md](../../plan/ui-ux/research.md) R-24, R-25, R-27, R-28, R-35
- Audit: [audit.md](../../plan/ui-ux/audit.md) §1, G-2, G-3, G-4, G-5a, G-5c
- External: [WAI-ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/),
  [React Aria Components](https://react-spectrum.adobe.com/react-aria/),
  [Instructure UI](https://github.com/instructure/instructure-ui) (LMS peer)
- Related plans: [UX.1](UX.1-semantic-design-token-system.md),
  [UX.4](../../plan/ui-ux/UX.4-aria-widget-and-focus-management-remediation.md),
  [UX.6](../../plan/ui-ux/UX.6-form-and-validation-system.md),
  [UX.18](../../plan/ui-ux/UX.18-design-system-governance-and-measurement.md),
  [`../tech_debt/TD.14-decompose-god-components.md`](../../plan/tech_debt/TD.14-decompose-god-components.md)
