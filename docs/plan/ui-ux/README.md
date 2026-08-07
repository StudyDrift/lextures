# UX — UI / UX Remediation Programme

> Eighteen plans (`UX.1`–`UX.18`) derived from a measured audit of `clients/web`
> against published UX research. Audit date **2026-08-05**, commit `aa0d523a`.

| Document | Purpose |
|---|---|
| [research.md](research.md) | The evidence base. 37 numbered findings (`R-n`) from cognitive-load theory, SDT, IA research, design-system practice, 2024–2026 trends, competitive analysis and accessibility law. |
| [audit.md](audit.md) | The gap analysis. 18 numbered findings (`G-n`), every one backed by a reproducible measurement. |
| `UX.1`–`UX.18` | Implementation plans, each conforming to [`_TEMPLATE.md`](../_TEMPLATE.md). |

---

## The one-sentence finding

**Lextures has built the right primitives and then not adopted them.**

| Primitive | Exists | Files using it | Files that should |
|---|---|---|---|
| `components/ui/button.tsx` | ✅ | **2** | ~600 |
| `components/ui/overlay-surface.tsx` | ✅ | **0** *(dead code)* | ~129 |
| `components/ui/lms-content-skeletons.tsx` | ✅ | **5** | ~200 |
| `components/ui/empty-state.tsx` | ✅ | **15** | ~150 |
| `lib/lms-toast.ts` (incl. `toastWithUndo`) | ✅ | **1** | ~300 |
| `lib/a11y/focus-trap.ts` | ✅ | **3** | **129** |
| Semantic colour tokens | ✅ UX.1 | migrating (purity ratchet) | all |

Against that: **2,016 hand-rolled `<button>` elements**, **129 hand-rolled
dialogs**, **33,331 raw colour literals in 698 of 795 component files**.

The one domain where tokens *were* enforced — **motion**, via the completed
AN.1–AN.7 plans, with a CI check that fails the build — is markedly consistent.
That is the proof the organisation can do this. It has simply only done it once,
for the least load-bearing domain. **[UX.18](UX.18-design-system-governance-and-measurement.md)
exists so it does not happen again.**

---

## Plans

### Foundations — nothing else is durable without these

| ID | Plan | Severity | Effort | Depends on |
|---|---|---|---|---|
| **UX.1** | [Semantic design token system](../../completed/ui-ux/UX.1-semantic-design-token-system.md) (shipped) | BLOCKER | L | — |
| **UX.2** | [Core component library and adoption ratchet](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md) (shipped — library + ratchet; migration ongoing) | BLOCKER | XL | UX.1 |
| **UX.3** | [Typography and reading system](../../completed/ui-ux/UX.3-typography-and-reading-system.md) ✅ | MAJOR | M | UX.1 |

### Accessibility — legal floor, not polish

| ID | Plan | Severity | Effort | Depends on |
|---|---|---|---|---|
| **UX.4** | [ARIA widget and focus management remediation](../../completed/ui-ux/UX.4-aria-widget-and-focus-management-remediation.md) (shipped — contracts + ratchet; modal tail ongoing) | BLOCKER | L | UX.2 |
| **UX.5** | [WCAG 2.2 AA conformance uplift](UX.5-wcag-2.2-aa-conformance-uplift.md) | MAJOR¹ | M | UX.2, UX.4 |
| **UX.6** | [Form and validation system](UX.6-form-and-validation-system.md) | MAJOR | L | UX.2 |

¹ BLOCKER for EU sales — the EAA deadline passed 28 June 2025 and enforcement is live (**R-36**).

### Information architecture

| ID | Plan | Severity | Effort | Depends on |
|---|---|---|---|---|
| **UX.7** | [Navigation information architecture](UX.7-navigation-information-architecture.md) | MAJOR | L | UX.2, UX.15 |
| **UX.8** | [Settings and admin IA unification](UX.8-settings-and-admin-ia-unification.md) | MAJOR | M | UX.6, UX.7 |

### Core surfaces

| ID | Plan | Severity | Effort | Depends on |
|---|---|---|---|---|
| **UX.9** | [Role-aware, prioritised dashboard](UX.9-role-aware-dashboard.md) | MAJOR | L | UX.1–3, UX.7, UX.12 |
| **UX.10** | [Course home and the learning flow](UX.10-course-home-and-learning-flow.md) | MAJOR | XL | UX.1–3, UX.7, UX.12, **TD.14** |
| **UX.11** | [Data table and gradebook system](UX.11-data-table-and-gradebook-system.md) | MAJOR | L | UX.1–3, UX.5 |

### Interaction and state

| ID | Plan | Severity | Effort | Depends on |
|---|---|---|---|---|
| **UX.12** | [Loading, empty, error and offline states](UX.12-loading-empty-error-offline-states.md) | MAJOR | M | UX.1, UX.2 |
| **UX.13** | [Feedback, undo and destructive actions](UX.13-feedback-undo-and-destructive-actions.md) | MAJOR | M | UX.2, UX.12 |

### Cross-cutting

| ID | Plan | Severity | Effort | Depends on |
|---|---|---|---|---|
| **UX.14** | [Responsive and small-viewport experience](UX.14-responsive-and-small-viewport-experience.md) | MAJOR | L | UX.1–3, UX.5, UX.11 |
| **UX.15** | [i18n coverage and RTL completion](UX.15-i18n-coverage-and-rtl-completion.md) | MAJOR | L | UX.2 |
| **UX.16** | [Progress, motivation and learner agency](UX.16-progress-motivation-and-learner-agency.md) | MINOR² | M | UX.9, UX.10 |
| **UX.17** | [Perceived performance and web vitals budget](UX.17-perceived-performance-and-web-vitals-budget.md) | MAJOR | M | UX.12 |

² Low severity, high strategic value — it is the interface expression of the
"learning environment that adapts" thesis.

### Governance

| ID | Plan | Severity | Effort | Depends on |
|---|---|---|---|---|
| **UX.18** | [Design system governance and measurement](UX.18-design-system-governance-and-measurement.md) | MAJOR | M + ongoing | UX.1, UX.2 |

---

## Sequencing

Four waves. Waves overlap; the arrows are hard dependencies, not scheduling.

```
Wave 0  ── measure first, or the programme is unaccountable ──────────────────
        UX.17a  RUM + CWV instrumentation (4-week collection)
        UX.18   metrics + ratchets at *current* baselines (nothing blocked)
        UX.7    nav telemetry collection + card sort / tree test  ← research gate
        UX.9    dashboard widget telemetry collection
        TD.14   god-component decomposition (hard prerequisite for UX.10/UX.11)
        Baseline SUS + task-time for the 8 critical journeys

Wave 1  ── foundations ───────────────────────────────────────────────────────
        UX.1 ──► UX.2 ──► UX.4 ──► UX.5
          └────► UX.3            └► UX.6
                 UX.15 (runs alongside; UX.7 needs its keys)
                 UX.12

Wave 2  ── structure and interaction ─────────────────────────────────────────
        UX.7 ──► UX.8
        UX.13
        UX.11
        UX.17b  optimisation against the Wave-0 budgets

Wave 3  ── surfaces ──────────────────────────────────────────────────────────
        UX.9 ──► UX.16
        UX.10 ─┘
        UX.14   (retrofit tail is small once UX.2/UX.11 have landed)
```

### Three sequencing rules

1. **Measure before you change.** UX.7, UX.9 and UX.17 all specify a telemetry or
   research window *before* implementation. Skipping these makes the resulting
   design unfalsifiable — and per **R-15**, IA must be tested, not argued.
2. **Governance lands with the work, not after it.** UX.18's ratchets start at
   current values so nothing is blocked on day one, and each gate turns on as its
   upstream plan lands. Governance that arrives at the end is an autopsy.
3. **TD.14 gates the learning flow.** `course-modules.tsx` (3,284 lines, 100
   `useState`) and `course-module-quiz-page.tsx` (3,403 lines) cannot be safely
   redesigned as they stand. See
   [`../tech_debt/TD.14-decompose-god-components.md`](../tech_debt/TD.14-decompose-god-components.md).

---

## Programme metrics

Every plan carries at least one. UX.18 makes them ratchets — CI fails on
regression regardless of whether the target is met yet.

| Metric | Today | Target | Owner plan |
|---|---|---|---|
| `token-purity` (raw palette literals) | **33,331** | 0 | UX.1 |
| `design-system-coverage` | **~0.25%** | ≥90% | UX.2 |
| Body text below 13px | **2,775** | 0 | UX.3 |
| `aria-contract-coverage` | **~21%**³ (ratchet live) | 100% | UX.4 |
| Modals with `aria-modal` but no focus trap | **~123** | 0 | UX.4 |
| `aria-invalid` per invalid field | **10 / 929 inputs** | 100% | UX.6 |
| Max in-course nav links | **40** | ≤7 + sections | UX.7 |
| Dashboard co-equal sections | **~18** | ≤6 + primary | UX.9 |
| Tables outside a table system | **99** | 0 | UX.11 |
| Error boundaries per route | **4 / 200** | 200 / 200 | UX.12 |
| `i18n-coverage` | **34%** | ≥98% | UX.15 |
| `responsive-coverage` | **32%** | ≥95% | UX.14 |
| Entry bundle (gzip) | **245,104 B** | ≤180 KB | UX.17 |
| `a11y-violations` (top 40 routes) | unmeasured | 0 | UX.18 |

³ 4 of 37 menus + 0 of 22 tablists implement their keyboard contract.

Attitudinal measures (SUS per persona, SEQ per task) **must be baselined before
UX.1 ships**. Without a pre-measurement the programme cannot be evaluated.

---

## What is already good

Worth stating plainly, because it establishes the standard is reachable.

- **Motion system** — 10 duration tokens, 3 easings, a documented bubble spring,
  per-UI-mode timing, reduced-motion handling throughout, and CI enforcement.
- **Command palette** — well-designed trigger with a `⌘K` hint; matches current
  best practice (**R-29**).
- **i18n infrastructure** — 4 locales at exact 3,746-key parity, ICU formatting,
  RTL registry, missing-key telemetry, lint plugin, CI check. *Infrastructure,
  not coverage.*
- **`useConfirm`** — consistently adopted at 41 sites, i18n'd, with a danger
  variant. The one feedback pattern that works.
- **Reading accessibility** — reading preferences, ruler, read-aloud with
  non-colour-only highlighting, caption variables, high-contrast sheet, K-2 and
  elementary UI modes.
- **Ratchet culture** — six `check-*.mjs` CI guards already exist. UX.18
  generalises a habit the team already has.

---

## Conventions

- File naming: `UX.{n}-{kebab-slug}.md`, per [`../README.md`](../README.md).
- Every plan fills all 19 sections of [`_TEMPLATE.md`](../_TEMPLATE.md). Where a
  section does not apply (no data model, no API surface, not AI-touching), it says
  so explicitly rather than leaving a placeholder.
- Research claims cite `R-n` from [research.md](research.md); gap claims cite `G-n`
  from [audit.md](audit.md).
- Severity: **BLOCKER** (cannot sell without it) · **MAJOR** (RFP-losing) ·
  **MINOR** (parity / nice-to-have).
- Effort: XS (≤1d) · S (1w) · M (2–4w) · L (1–2mo) · XL (>2mo).

## Related plans

- [`../tech_debt/`](../tech_debt/) — TD.12, TD.13 and **TD.14** are structural
  prerequisites for UX.9, UX.10, UX.11 and UX.12.
- [`../standards/S20-accessibility-legal-mandates.md`](../standards/S20-accessibility-legal-mandates.md)
  — owns the legal framing that UX.4 and UX.5 implement.
- [`../../completed/12-accessibility/`](../../completed/12-accessibility/) — the
  shipped WCAG 2.1 AA baseline this programme extends.
- [`../../completed/animations/`](../../completed/animations/) — AN.1–AN.7, the
  model this programme generalises to every other token domain.
- [`../../completed/11-i18n-l10n/`](../../completed/11-i18n-l10n/) — the pipeline
  UX.15 completes.
- [`../adaptive/`](../adaptive/), [`../content_tools/`](../content_tools/) — consumed
  by UX.10 and UX.16, not modified.
