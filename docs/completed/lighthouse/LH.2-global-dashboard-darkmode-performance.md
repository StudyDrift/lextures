# LH.2 — Global Dashboard Dark Mode: Performance

> Completed implementation plan. Source: [docs/lighthouse/global-dashboard-darkmode.json](../../lighthouse/global-dashboard-darkmode.json).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LH.2 |
| **Section** | Lighthouse remediation |
| **Severity** | MAJOR |
| **Markets** | All |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Frontend platform team |
| **Depends on** | LH.1 (valid Lighthouse baseline) |
| **Unblocks** | 21.2 (mobile Lighthouse ≥ 70 target), 17.7 (Web Vitals in observability) |

## Shipped

- **Route code splitting** — All LMS pages lazy-loaded via `src/lazy-pages.ts` and `Suspense` in `src/app.tsx` (FR-2).
- **Dashboard fetch deferral** — Review stats, recommendations, catalog schedule, and academic calendar deferred with `scheduleIdleTask` (FR-3).
- **Progressive dashboard render** — Quick links and page title render when the course catalog resolves; course rows hydrate with section skeletons (FR-4, AC-4).
- **Mobile course prefetch** — First 3 courses on narrow viewports; remainder via "Load more" (FR-6).
- **Bundle budgets** — Entry chunk ≤ 200 KB gzip; dashboard chunk regression ≤ 10 KB (FR-5, AC-3, AC-5). See `clients/web/CONTRIBUTING.md`.
- **Observability marks** — `performance.mark` for catalog load, enrichment, and row hydration (NFR observability).
- **Tests** — Unit tests for prefetch/idle helpers; Playwright dashboard quick-links test; lazy-route test fixes.

## Baseline (2026-06-15 build)

| Metric | Before LH.2 | After LH.2 |
| --- | --- | --- |
| Entry JS gzip | ~1.19 MB (monolithic) | 192 KB (`index-*.js`) |
| Dashboard lazy chunk gzip | (bundled) | 12 KB (`dashboard-*.js`) |
| Lighthouse Performance (dark, mobile) | 0.26 | Re-run via `npm run lighthouse:dashboard:dark` after merge |

## References

- Dashboard: `clients/web/src/pages/lms/dashboard.tsx`
- Lazy routes: `clients/web/src/lazy-pages.ts`, `clients/web/src/app.tsx`
- Bundle check: `clients/web/scripts/check-bundle-size.mjs`
- Related: [LH.1](LH.1-global-dashboard-darkmode-audit-harness.md), [LH.3](../completed/lighthouse/LH.3-global-dashboard-darkmode-accessibility.md)
