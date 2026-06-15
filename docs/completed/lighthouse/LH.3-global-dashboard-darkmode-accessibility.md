# LH.3 — Global Dashboard Dark Mode: Accessibility

> Completed implementation plan. Source: [docs/lighthouse/global-dashboard-darkmode.json](../../lighthouse/global-dashboard-darkmode.json).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LH.3 |
| **Section** | Lighthouse remediation |
| **Severity** | MAJOR |
| **Markets** | All |
| **Status (today)** | COMPLETED |
| **Estimated effort** | S (1–2w) |
| **Owner (proposed)** | Frontend platform team |
| **Depends on** | LH.1 (valid Lighthouse baseline), 12.3 (design-token contrast contract) |
| **Unblocks** | VPAT evidence for dashboard, 10.7 WCAG program |

## Shipped

- **Lighthouse accessibility baseline** — Dark-mode mobile report achieves score **1.0** (≥ 0.95 target); weighted audits (`button-name`, `color-contrast`, `heading-order`, `link-name`, `image-alt`) all pass (AC-1).
- **Harness threshold** — `runLighthouseDashboard` enforces `LH_MIN_A11Y_SCORE` (default 0.95) and reports weighted audit failure count (FR-1, NFR observability).
- **E2E regression** — `e2e/tests/lighthouse-dashboard-a11y.spec.ts` validates committed baseline JSON, collapse-toggle `aria-label` (AC-3), and heading outline (AC-6).
- **Shared helpers** — `e2e/lib/lighthouse-a11y.ts` parses Lighthouse JSON and surfaces failing selectors for CI output (FR-7).
- **CI gate** — `.github/workflows/lighthouse.yml` fails when accessibility score < 0.95; performance remains informational.
- **Existing axe coverage** — `e2e/tests/contrast.spec.ts` dashboard dark test continues to gate `color-contrast` (AC-2).

## Baseline (2026-06-15 build)

| Metric | Score |
| --- | --- |
| Lighthouse Accessibility (dark, mobile) | **100** |
| Weighted a11y audit failures | 0 |
| Lighthouse Performance (dark, mobile) | 26 (LH.2 scope; not gated) |

Re-run: `npm run lighthouse:dashboard:dark` from `clients/web/` or `e2e/` with stack up.

## References

- Baseline: `docs/lighthouse/global-dashboard-darkmode.json`
- Dashboard: `clients/web/src/pages/lms/dashboard.tsx`
- Contrast CI: `e2e/tests/contrast.spec.ts`, `clients/web/contrast-config.json`
- Related: [LH.1](LH.1-global-dashboard-darkmode-audit-harness.md), [LH.2](LH.2-global-dashboard-darkmode-performance.md), [12.3 color-contrast compliance](../12-accessibility/12.3-color-contrast-compliance.md)
