# UX.18 — Design System Governance and Measurement

> Implementation plan. Source: [audit.md](audit.md) §1, §2 G-2, G-16.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.18 |
| **Section** | UI/UX — Governance |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING — no ownership, no metrics, no contribution path |
| **Estimated effort** | M (2–4w) + ongoing |
| **Owner (proposed)** | Design Systems (standing team) |
| **Depends on** | UX.1, UX.2 |
| **Unblocks** | Durability of every other UX.* plan |

---

## 1. Problem Statement

Lextures has been here before. `components/ui/button.tsx` is a good component that
nobody adopted; `overlay-surface.tsx` is dead code with zero importers;
`lms-toast.ts` implements `toastWithUndo` and has one caller; `focus-trap.ts` is
used three times against 126 modals; `docs/design.md` is a style guide contradicted
by the code it describes. **The failure was never authorship — it was governance.**
Without ownership, adoption measurement, a contribution path and enforcement, the
work in UX.1–UX.17 will decay exactly the same way: a beautiful token system with
33,000 raw literals growing back around it. This plan makes the design system a
governed asset rather than a directory of hopeful files.

## 2. Goals

- Establish clear **ownership** and a working **contribution model**.
- Make **adoption measurable** and published, so decay is visible within a week
  rather than a year (**R-27**).
- Make every foundational decision **enforceable in CI**, with ratchets that cannot
  loosen.
- Replace stale prose documentation with a **living, executable** reference.
- Institute a **design review** with explicit criteria, not taste.

## 3. Non-Goals

- Building components — that is [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md).
- Reorganising teams or reporting lines.
- Open-sourcing the design system.
- Design tooling procurement beyond what the token pipeline needs.

## 4. Personas & User Stories

- **As an engineer**, I want an obvious answer to "does a component for this
  exist?" so I do not hand-roll a 2,017th button.
- **As an engineer with a genuine gap**, I want a clear, fast path to contribute a
  component rather than building it locally.
- **As a designer**, I want Figma and code to share tokens so specs are not
  guesses.
- **As an engineering manager**, I want to know whether the design-system
  investment is paying back.
- **As a new joiner**, I want one place that shows what the product looks like and
  how to build in it.
- **As a product owner**, I want design review to be about criteria and evidence,
  not opinion.

## 5. Functional Requirements

### Ownership and contribution

- **FR-1.** A named **Design Systems owner** and a small standing group MUST be
  established, with an explicit remit: tokens, components, patterns, accessibility
  contracts, and the ratchets.
- **FR-2.** A documented **contribution path** MUST exist: propose → review →
  build → document → adopt, with a target turnaround (proposal answered within 3
  working days).
- **FR-3.** A **triage rule** MUST be documented for "the system does not have what
  I need": (a) compose from existing primitives, (b) contribute upstream, (c) build
  locally **with a recorded exemption and an expiry date**. Local builds MUST NOT be
  silent.
- **FR-4.** Every component MUST have an owner and a maintenance status
  (experimental / stable / deprecated).
- **FR-5.** A deprecation policy MUST exist: mark deprecated → lint warning →
  one-release grace → removal.

### Measurement (**R-27**)

- **FR-6.** The following metrics MUST be computed in CI on every build and
  published on a dashboard:

  | Metric | Definition | Target |
  |---|---|---|
  | `design-system-coverage` | Interactive elements from `ui/*` ÷ all interactive elements | ≥90% |
  | `token-purity` | Raw palette literals in `src/**/*.tsx` | 0 |
  | `type-role-purity` | Raw `text-*` size classes in `src/**/*.tsx` | 0 |
  | `aria-contract-coverage` | Declared ARIA roles with full APG keyboard contract | 100% |
  | `state-contract-coverage` | Data surfaces with all four states | 100% |
  | `i18n-coverage` | Externalised user-visible strings | ≥98% |
  | `responsive-coverage` | Route components with verified ≤390 px layout | ≥95% |
  | `a11y-violations` | axe violations across the top 40 routes | 0 |
  | `cwv-budget-breaches` | Route classes exceeding their budget | 0 |

- **FR-7.** Every metric MUST be a **ratchet**: CI fails if it worsens, regardless
  of whether the target is met yet.
- **FR-8.** Each metric MUST have an allowlist that can only shrink; a PR that adds
  an entry MUST require Design Systems approval.
- **FR-9.** Metrics MUST be published weekly to the engineering channel with
  trends, so decay is visible immediately.
- **FR-10.** A **payback measure** MUST be tracked: time-to-build for a
  representative UI task, sampled quarterly, and the count of accessibility defects
  originating in feature code vs component code.

### Living documentation

- **FR-11.** The component gallery (UX.2 FR-12) MUST be the **canonical**
  reference: every component, every variant, every state, all four themes, LTR and
  RTL, with usage code and the keyboard contract.
- **FR-12.** `docs/design.md` MUST be rewritten to describe **principles and
  decisions**, referencing tokens rather than restating hex values — the current
  version is factually contradicted by the code (wrong font stack, wrong radius,
  no colour semantics).
- **FR-13.** `docs/design-tokens.md` MUST be rewritten as the token specification
  (three layers, naming grammar, how to add and consume).
- **FR-14.** A **pattern library** MUST document composed patterns above the
  component level: page shell, form layout, empty/error/loading, destructive
  action, data table, wizard, settings page.
- **FR-15.** Documentation MUST be **generated from source** where possible (props
  from types, tokens from the token files) so it cannot drift.
- **FR-16.** A CI check MUST fail when an exported component lacks a gallery entry
  or documentation.

### Design/engineering alignment

- **FR-17.** Figma MUST consume the same tokens via the W3C DTFM artefact
  (UX.1 FR-11); a token change MUST propagate to both without manual re-entry.
- **FR-18.** A **design review** MUST run before significant UI work, with explicit
  criteria: does it use existing patterns? does it meet the accessibility contract?
  are all four states designed? is it responsive to 320 px? is copy i18n-ready?
  does it inform rather than manipulate?
- **FR-19.** Visual regression MUST run on the gallery and the top 40 routes, so
  unintended visual change is caught before merge.

### Preventing recurrence

- **FR-20.** A component with **zero importers outside `ui/`** for one release MUST
  be flagged and either adopted or deleted (UX.2 FR-13). The `overlay-surface.tsx`
  situation must not repeat.
- **FR-21.** Every new *feature* plan MUST state which existing components and
  patterns it uses, and justify anything new.
- **FR-22.** `AGENTS.md` / `CLAUDE.md` MUST carry the foundational rules so
  AI-assisted contributions follow them by default.

## 6. Non-Functional Requirements

- **Performance** — Metric computation MUST add ≤60 s to CI. Visual regression runs
  on a schedule for the full matrix and a reduced matrix per PR.
- **Security** — The gallery is staff-gated and MUST NOT be reachable in production
  by unauthenticated users; it MUST NOT contain real user data.
- **Privacy & Compliance** — Governance artefacts feed SOC 2 evidence
  (`../standards/S21-compliance-evidence-continuous-monitoring.md`) and the VPAT
  process (`docs/vpat/VPAT_UPDATE_CHECKLIST.md`).
- **Accessibility** — `a11y-violations` and `aria-contract-coverage` are permanent
  gates, not one-off project outcomes.
- **Scalability** — Metrics must run on a 795-file codebase in seconds and scale.
- **Reliability** — A flaky gate is worse than no gate: any check with >2% false
  positives MUST be fixed or removed, not tolerated.
- **Observability** — Metrics are the observability. History MUST be retained so
  trends are visible.
- **Maintainability** — One metrics script directory; one config file listing gates
  and targets.
- **Internationalization** — Gallery renders in RTL and pseudo-locale as a standing
  check.
- **Backward compatibility** — Ratchets start at *current* values, so nothing is
  blocked on day one; they only prevent regression.

## 7. Acceptance Criteria

- **AC-1.** *Given* the repository, *When* CI runs, *Then* all nine FR-6 metrics are
  computed and reported in the build summary.
- **AC-2.** *Given* a PR that worsens any metric, *When* CI runs, *Then* it fails
  with the specific regression named and the fix suggested.
- **AC-3.** *Given* a PR that adds an allowlist entry, *When* opened, *Then* it
  requires Design Systems review before merge.
- **AC-4.** *Given* the metrics dashboard, *When* viewed, *Then* it shows current
  values and 12-week trends for every metric.
- **AC-5.** *Given* an exported component without a gallery entry or docs, *When*
  CI runs, *Then* it fails.
- **AC-6.** *Given* a component with zero importers outside `ui/` for one release,
  *When* the check runs, *Then* it is flagged for adoption or deletion.
- **AC-7.** *Given* `docs/design.md` and `docs/design-tokens.md`, *When* reviewed,
  *Then* neither contains a claim contradicted by the code, and both reference
  tokens rather than literal values.
- **AC-8.** *Given* a token change in the source, *When* the pipeline runs, *Then*
  both the code and the Figma artefact reflect it without manual re-entry.
- **AC-9.** *Given* the contribution path, *When* an engineer files a proposal,
  *Then* it receives a decision within 3 working days.
- **AC-10.** *Given* a significant UI change, *When* it reaches design review,
  *Then* it is assessed against the written FR-18 criteria and the outcome is
  recorded.
- **AC-11.** *Given* the gallery, *When* visual regression runs, *Then* unintended
  changes are caught before merge with <2% false-positive rate.
- **AC-12.** *Given* the quarterly payback measure, *When* sampled, *Then*
  time-to-build for the representative task and the a11y-defect origin split are
  recorded and published.

## 8. Data Model

None in the product database. Metrics history is stored as CI artefacts and in the
existing observability stack. No tables, columns, enums, indexes, migrations or
backfill.

## 9. API Surface

None. No HTTP or WebSocket changes, no rate-limit considerations, no OpenAPI
changes. *(The gallery is a client route, gated by existing staff permissions.)*

## 10. UI / UX

- **New pages** — the metrics dashboard (may be a Grafana panel or a static CI
  artefact page rather than an in-product route).
- **Modified pages** — `/design/components` and `/design/tokens` (from UX.1/UX.2)
  become the canonical reference and gain the pattern library.
- **Key user flows**
  1. Engineer needs a control → opens the gallery → finds it → copies usage.
  2. Engineer finds a gap → files a proposal → gets a decision in ≤3 days.
  3. Engineer opens a PR → CI reports every metric delta inline.
  4. Team reviews the weekly metrics post and sees a trend before it becomes debt.
- **States** — gallery: loading, empty search, component not found (with the
  contribution path linked). Dashboard: loading, no-data (first run), error.
- **Mobile/responsive** — the gallery renders each component at 390/768/1280 px as
  part of its purpose.
- **Accessibility annotations** — the gallery is itself held to the accessibility
  bar; it is the reference implementation and must be exemplary.
- **Copy & i18n** — gallery and docs are `en`-only staff tooling, but MUST NOT break
  the i18n parity check.

## 11. AI / ML Considerations

Not AI-touching as a feature. One governance requirement: **AI-assisted development
is now a primary contributor to this codebase**, and FR-22 exists because rules that
live only in a wiki will not be followed. `AGENTS.md` and `CLAUDE.md` MUST carry
the enforceable rules — no raw palette literals, no hand-rolled buttons/dialogs/
menus/tabs, no hardcoded user-visible strings, all four states required — so that
generated code conforms by default rather than being corrected in review.

## 12. Integration Points

- **External** — Figma (token consumption via W3C DTFM); CI provider.
- **Internal**
  - `clients/web/scripts/check-*.mjs` — the existing ratchet family
    (`check-contrast`, `check-i18n-locales`, `check-bundle-size`,
    `check-interface-polish`, `check-entity-labels`,
    `check-platform-feature-toggles`) is the precedent and the home for new checks
  - `clients/web/eslint-rules/`, `.oxlintrc.json`
  - `clients/packages/tokens/` — the DTFM artefact
  - `docs/design.md`, `docs/design-tokens.md`, `docs/guides/**`
  - `AGENTS.md`, `CLAUDE.md`
  - `Makefile` — a `make ux-metrics` target alongside `make lint-structure`
  - `.github/` — CI workflows, required checks, CODEOWNERS for `components/ui/`
    and the token files
- **Events** — metric history into CI artefacts and the observability stack.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.1](UX.1-semantic-design-token-system.md) and
  [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md) exist enough to
  measure.
- **Must ship before the programme completes** — governance that arrives after the
  work is finished is an autopsy. **The metrics and ratchets should land as each
  plan lands**, not at the end.
- **Ongoing** — this is a standing function, not a project that completes.
- **Shared infra** — CI capacity for visual regression and metric computation.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Design Systems ownership is nominal and nobody has time | **H** | **Critical** | Named owner with allocated capacity, not a volunteer rota. If capacity cannot be allocated, **the rest of the programme will decay** — that trade-off must be made explicitly, in writing |
| Ratchets are disabled the first time they block a deadline | **H** | **H** | Ratchets start at *current* values so nothing is blocked initially; allowlists provide a legitimate escape hatch with review (FR-8); disabling a gate requires the same review as adding an allowlist entry |
| Metrics become the goal and the system is gamed (**R-28**) | M | M | Pair coverage with the payback measure (FR-10) and qualitative review; a component nobody wants gets fixed or deleted, not mandated |
| Gallery and docs drift from the code | M | **H** | FR-15 generation from source; FR-16 CI check; the gallery is the reference implementation, so it breaks visibly |
| Visual regression flakiness erodes trust in all gates | **H** | **H** | <2% false-positive requirement (AC-11); full matrix nightly, reduced matrix per PR; flakiness treated as a defect |
| Contribution path is too slow and engineers route around it | M | **H** | 3-working-day SLA (FR-2, AC-9); option (c) local build with a recorded, expiring exemption keeps velocity while keeping visibility |
| Figma/code token sync breaks silently | M | M | CI check that the DTFM artefact matches the source tokens |

## 15. Rollout Plan

- **Feature flag** — none. This is process and CI, not product.
- **Sequencing**
  1. Name the owner and the group; publish the remit and contribution path.
  2. Stand up the metrics scripts with **current values as baselines** — nothing
     blocked, everything visible.
  3. Publish the dashboard and start the weekly post.
  4. Turn ratchets on one at a time as each upstream plan lands (token purity with
     UX.1, coverage with UX.2, and so on).
  5. Rewrite `docs/design.md` and `docs/design-tokens.md`; retire the stale claims.
  6. Pattern library.
  7. Figma token pipeline.
  8. Design review process, with the criteria published.
  9. CODEOWNERS on `components/ui/` and the token files.
  10. `AGENTS.md` / `CLAUDE.md` rules.
- **Dogfood** — the team itself; the first month's weekly posts will be noisy and
  that is the point.
- **GA criteria** — AC-1…AC-12 green; four consecutive weeks of published metrics;
  at least three contributions through the documented path.
- **Rollback** — gates can be downgraded to warnings without code change. Rolling
  back governance is a decision to accept decay and should be recorded as such.

## 16. Test Plan

- **Unit** — each metric script against fixture repositories with known values
  (including deliberately-violating fixtures); allowlist parsing; ratchet
  comparison logic (must fail on regression, pass on improvement).
- **Integration** — CI workflow end-to-end: a PR that improves a metric passes; a
  PR that worsens it fails with a useful message; an allowlist addition requires
  review.
- **End-to-end** — the gallery renders every component in all four themes and both
  directions without error; a missing-docs component fails CI.
- **Security** — the gallery is not reachable by unauthenticated users in
  production; it contains no real user data.
- **Accessibility** — the gallery itself passes axe in all four themes and both
  directions — the reference implementation must be exemplary.
- **Performance / load** — metric computation ≤60 s; visual regression matrix
  completes within the CI budget; false-positive rate measured over 4 weeks (AC-11).
- **Manual exploratory** — a new-joiner exercise: given only the gallery and docs,
  build a small feature; record where they got stuck. This is the real test of the
  documentation.

## 17. Documentation & Training

- **End-user** — none.
- **Admin / instructor** — none.
- **Engineer** — this plan is largely a documentation deliverable:
  - `docs/design.md` — rewritten: principles and decisions
  - `docs/design-tokens.md` — rewritten: the token specification
  - `docs/guides/component-library.md`, `accessibility-patterns.md`, `forms.md`,
    `ui-states.md`, `data-table.md`, `responsive.md`, `i18n.md`,
    `navigation-registry.md`, `performance-budgets.md`,
    `feedback-and-destructive-actions.md`, `learner-motivation.md`,
    `dashboard-widgets.md`, `item-shell.md`, `settings-architecture.md` — the
    per-plan guides, indexed from one place
  - `docs/guides/design-system-governance.md` — ownership, contribution, triage,
    deprecation, metrics, review criteria
- **Design** — the Figma token pipeline and the review criteria.
- **API reference** — n/a.
- **Runbook** — one entry per gate: what it means, how to fix it, how to request an
  exemption.
- **Onboarding** — the design-system section of engineering onboarding.

## 18. Open Questions

1. Who is the Design Systems owner, and what capacity is allocated? **This is the
   single most important open question in the entire programme** — without an
   answer, UX.1–UX.17 will decay back to the current state.
2. Is the group a dedicated team or a federated model with rotating contributors?
   *Recommendation: a small dedicated core (1 design + 2 engineering) plus federated
   contribution — a pure federation has already been tried implicitly and produced
   the current state.*
3. Which visual regression tool? (Playwright screenshots in-repo vs a hosted
   service.) *Recommendation: start with Playwright in-repo — no new vendor, no new
   data agreement — and revisit if triage becomes the bottleneck.*
4. Should `design-system-coverage` move from source-level to Preply-style **visual
   coverage** (**R-27**) once the basics are in place? *Recommendation: evaluate at
   the 12-month mark; source-level is sufficient to drive the migration.*
5. How do native clients participate? They share tokens but not components; do they
   get their own coverage metric?
6. What is the escalation path when a team persistently requests exemptions?

## 19. References

- Existing files: `clients/web/scripts/check-contrast.mjs`,
  `check-i18n-locales.mjs`, `check-bundle-size.mjs`, `check-interface-polish.mjs`,
  `check-entity-labels.mjs`, `check-platform-feature-toggles.mjs`,
  `clients/web/eslint-rules/`, `clients/web/eslint-plugin-lextures-i18n.js`,
  `clients/web/.oxlintrc.json`, `docs/design.md`, `docs/design-tokens.md`,
  `AGENTS.md`, `Makefile` (`lint-structure` precedent),
  `docs/ARCHITECTURE_CONVENTIONS.md`
- Research: [research.md](research.md) R-24, R-25, R-26, R-27, R-28
- Audit: [audit.md](audit.md) §1, G-2, G-16, §9
- External: [Measuring design system adoption — Mews](https://developers.mews.com/design-system-adoption-metric-building/),
  [Visual Coverage — Into Design Systems](https://www.intodesignsystems.com/blog/measure-design-systems-impact),
  [Design System "Adoption" is a Red Herring](https://medium.com/@disco_lu/design-system-adoption-is-a-red-herring-6c6b5a504f43),
  [W3C Design Tokens Format Module](https://tr.designtokens.org/)
- Related plans: **all of UX.1–UX.17**; `../tech_debt/` (the structural-ratchet
  precedent this plan generalises), `../standards/S21-compliance-evidence-continuous-monitoring.md`
