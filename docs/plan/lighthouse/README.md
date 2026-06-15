# Lighthouse Remediation Plans

Implementation plans derived from Lighthouse reports stored in [`docs/lighthouse/`](../../lighthouse/). Each plan follows the structure in [`../_TEMPLATE.md`](../_TEMPLATE.md).

## Conventions

- File naming: `LH.{number}-{kebab-slug}.md` (e.g. `LH.1-global-dashboard-darkmode-audit-harness.md`).
- **Feature ID** prefix: `LH.{number}` (Lighthouse remediation track; separate from `docs/MISSING_FEATURES.md` sections).
- Source report: every plan cites the JSON (or HTML) artifact under `docs/lighthouse/` that motivated it.
- A plan is "ready" when every section in the template is filled (no `…` placeholders).
- Re-run Lighthouse after each plan ships and commit an updated baseline JSON beside the original report.

## Severity legend

- **BLOCKER** — Lighthouse cannot produce a valid baseline (e.g. `NO_FCP`); blocks all other LH plans for that page.
- **MAJOR** — Category score below target or weighted audit failure affecting real users.
- **MINOR** — Opportunistic improvement; passes thresholds but leaves measurable savings on the table.

## Reports

| Report | Page | Theme | Status | Plans |
|---|---|---|---|---|
| [`global-dashboard-darkmode.json`](../../lighthouse/global-dashboard-darkmode.json) | Global dashboard (`/`) | Dark (`.dark`) | **Valid** — run via `npm run lighthouse:dashboard:dark` (see [LH.1](../../completed/lighthouse/LH.1-global-dashboard-darkmode-audit-harness.md)) | [LH.2](LH.2-global-dashboard-darkmode-performance.md), [LH.3](LH.3-global-dashboard-darkmode-accessibility.md) |

## Plans

- [LH.1 — Global dashboard dark mode: reproducible Lighthouse harness](../../completed/lighthouse/LH.1-global-dashboard-darkmode-audit-harness.md) (completed)
- [LH.2 — Global dashboard dark mode: performance](LH.2-global-dashboard-darkmode-performance.md)
- [LH.3 — Global dashboard dark mode: accessibility](LH.3-global-dashboard-darkmode-accessibility.md)
