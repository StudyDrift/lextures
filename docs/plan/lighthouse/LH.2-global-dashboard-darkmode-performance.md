# LH.2 — Global Dashboard Dark Mode: Performance

> Implementation plan. Source: [docs/lighthouse/global-dashboard-darkmode.json](../../lighthouse/global-dashboard-darkmode.json) — performance category (blocked by `NO_FCP` until LH.1 ships).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LH.2 |
| **Section** | Lighthouse remediation |
| **Severity** | MAJOR |
| **Markets** | All |
| **Status (today)** | MISSING (baseline not yet collected) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Frontend platform team |
| **Depends on** | LH.1 (valid Lighthouse baseline) |
| **Unblocks** | 21.2 (mobile Lighthouse ≥ 70 target), 17.7 (Web Vitals in observability) |

---

## 1. Problem Statement

The global dashboard (`clients/web/src/pages/lms/dashboard.tsx`) is the default landing route after sign-in. It eagerly loads course metadata, then fans out parallel API calls per course (structure, grades, announcements, gradebook grid, grading backlog) via `mapPool`. The app shell imports every page synchronously in `clients/web/src/app.tsx` with no `React.lazy` route splitting, inflating the initial JavaScript payload. The failed 2026-06-15 Lighthouse report could not measure FCP, LCP, TBT, CLS, or Speed Index due to `NO_FCP`, but the architecture matches patterns Lighthouse typically flags: **unused JavaScript**, **render-blocking resources**, **main-thread work**, and **LCP breakdown** delays. Students and instructors on mid-tier mobile hardware experience slow first paint in dark mode, especially at semester start when dashboards hydrate many courses.

## 2. Goals

- Achieve Lighthouse **Performance ≥ 70** on mobile throttling for `/` in dark mode (aligned with plan 21.2 AC-6).
- Bring **LCP ≤ 2.5 s** and **TBT ≤ 300 ms** on the seeded e2e dashboard fixture.
- Reduce initial JS transferred for the `/` route by introducing route-level code splitting.
- Cut dashboard API waterfall latency so content sections populate without blocking LCP.
- Re-baseline `docs/lighthouse/global-dashboard-darkmode.json` after fixes.

## 3. Non-Goals

- Server-side caching (plan 17.5).
- Rewriting dashboard UX or removing features.
- Desktop-only performance tuning without mobile throttling validation.
- Native mobile app WebView performance (plan 21.1).

## 4. Personas & User Stories

- **As a student on a phone**, I want the dashboard to show my due items within a few seconds on LTE so that I can check deadlines between classes.
- **As an instructor**, I want the teaching overview and grading backlog to appear without a long blank skeleton so that I can triage work quickly.
- **As a frontend engineer**, I want bundle budgets on the dashboard route so that PRs cannot regress Lighthouse Performance.
- **As a platform engineer**, I want LCP and INP reported to observability (plan 17.7) from the same URLs Lighthouse audits.

## 5. Functional Requirements

- **FR-1.** After LH.1 baseline, the dashboard MUST meet Lighthouse Performance score ≥ 0.70 with mobile throttling and dark mode.
- **FR-2.** The `/` route MUST be lazy-loaded via `React.lazy` + `Suspense` so initial entry JS excludes unrelated LMS pages.
- **FR-3.** The dashboard MUST defer non-critical fetches (recommendations, review stats, academic calendar) until after first paint or `requestIdleCallback`.
- **FR-4.** The dashboard course fan-out (`mapPool` over structure/grades/announcements) MUST NOT block rendering of the page header and quick links; show partial content progressively.
- **FR-5.** The build MUST enforce a gzip budget for the initial `/` chunk (target ≤ 200 KB gzipped per `docs/ARCH.md` P1.6).
- **FR-6.** The system SHOULD prefetch only the first N courses (e.g. 3) for detailed rows on mobile, with "load more" or intersection observer for the rest.
- **FR-7.** The system SHOULD add `fetchpriority="high"` or equivalent for the LCP text node container if LCP remains text-bound after splitting.

## 6. Non-Functional Requirements

- **Performance** — LCP p75 < 2.5 s; TBT < 300 ms; CLS < 0.1 on dashboard fixture per Lighthouse mobile.
- **Security** — No change to auth model; lazy chunks served with same CSP as main bundle.
- **Privacy & Compliance** — No new tracking; Web Vitals optional and anonymized.
- **Accessibility** — Skeleton loading states remain announced; no removal of loading indicators without replacement.
- **Scalability** — Dashboard fan-out concurrency limits (`mapPool` pool size) remain bounded under load.
- **Reliability** — Partial API failures still render available sections (existing `detailError` pattern preserved).
- **Observability** — Log dashboard hydration timing (`performance.mark`) for future 17.7 export.
- **Maintainability** — Bundle budget in `vite.config.ts`; documented in `clients/web/CONTRIBUTING.md`.
- **Internationalization** — Lazy-loaded chunks include locale strings for dashboard copy.
- **Backward compatibility** — Desktop experience unchanged or improved; no removal of dashboard sections.

## 7. Acceptance Criteria

- **AC-1.** *Given* LH.1 harness runs after this work ships, *When* the JSON report is generated, *Then* `categories.performance.score ≥ 0.70`.
- **AC-2.** *Given* mobile throttling, *When* the dashboard loads for a user with 5 courses, *Then* LCP element is visible within 2.5 s (Lighthouse audit or Playwright trace).
- **AC-3.** *Given* `npm run build`, *When* bundle analysis runs for the `/` entry, *Then* initial JS gzip ≤ 200 KB (excluding lazily loaded route chunks).
- **AC-4.** *Given* a throttled network, *When* the dashboard loads, *Then* quick links and page title render before course detail rows finish loading.
- **AC-5.** *Given* a PR increases dashboard entry chunk by > 10 KB gzip, *When* CI runs, *Then* the build fails with a budget error.

## 8. Data Model

No database changes. Optional server aggregation endpoint (future) to replace N+1 client fetches — out of scope unless LH.1 baseline shows API wait dominates LCP.

## 9. API Surface

No required API changes for initial scope. **MAY** add `GET /api/v1/me/dashboard-summary` in a follow-up if client fan-out remains the LCP bottleneck after JS splitting.

## 10. UI / UX

Primary file: `clients/web/src/pages/lms/dashboard.tsx`.

Key flows:
1. Shell renders → lazy dashboard chunk loads → header + quick links paint.
2. Course list fetch → progressive section hydration (student/staff rows).
3. Secondary widgets (What's next, review stats, calendar) hydrate after idle.

Loading states:
- Keep `DashboardLoadingSkeleton` until `catalog` and `courses` resolve; ensure skeleton paints immediately (fixes `NO_FCP` class of issues).
- Section-level skeletons for slow course rows.

Dark mode: no performance-specific UI changes; verify gradients and shadows do not force expensive repaints.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- `clients/web/src/app.tsx` — route lazy loading.
- `clients/web/src/pages/lms/dashboard.tsx` — fetch orchestration, `mapPool` usage.
- `clients/web/vite.config.ts` — bundle budget.
- `clients/web/src/lib/async-pool.ts` — concurrency tuning.
- `docs/lighthouse/global-dashboard-darkmode.json` — post-fix baseline.
- Related: `docs/ARCH.md` § P1.6 Frontend Performance Baseline.

## 13. Dependencies & Sequencing

- Must ship after: LH.1.
- Must ship before: 21.2 Lighthouse mobile gate, 17.7 Web Vitals.
- Shared infra: Lighthouse harness, Vite bundle analyzer.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Lazy loading causes layout shift | M | M | Fixed skeleton dimensions; reserve min-height for sections |
| Deferring fetches hides critical deadlines | L | H | Never defer "due this week" primary list; only defer tertiary widgets |
| Bundle budget too aggressive for feature growth | M | M | Start at 200 KB; raise only with perf review |
| API fan-out still dominates after JS split | H | M | Measure in LH.1 baseline; escalate aggregation endpoint if LCP TTFB-bound |

## 15. Rollout Plan

- Feature flag not required for code splitting.
- Sequence: bundle analysis → lazy routes → dashboard fetch deferral → Lighthouse re-baseline.
- Dogfood on real mid-tier Android device before merge.
- GA criteria: AC-1 through AC-5; updated JSON in `docs/lighthouse/`.
- Rollback: revert lazy loading commit if fatal route error; budgets can be relaxed temporarily.

## 16. Test Plan

- **Unit** — Vitest for any extracted dashboard data hooks.
- **Integration** — Mock API latency; assert progressive render order.
- **End-to-end** — Playwright: dashboard visible < 5 s on throttled network; no regression on course row content.
- **Security** — N/A.
- **Accessibility** — Loading skeleton still meets contrast in dark mode.
- **Performance / load** — LH.1 harness in CI; bundle size check on PR.
- **Manual exploratory** — Chrome Performance panel on 4× CPU throttle; compare before/after filmstrip.

## 17. Documentation & Training

- Update `docs/plan/lighthouse/README.md` report table with Performance score.
- `clients/web/CONTRIBUTING.md` — document route lazy-loading and bundle budget rules.

## 18. Open Questions

1. Should the first LH.2 milestone target only JS splitting, or combine with API deferral in one PR?
2. Is a server-side dashboard aggregation endpoint justified if LCP remains API-bound after client optimizations?
3. What seeded course count should the Lighthouse fixture use — 2, 5, or 10?
4. Should we add `rollup-plugin-visualizer` to CI artifacts permanently?

## 19. References

- Source report: [`docs/lighthouse/global-dashboard-darkmode.json`](../../lighthouse/global-dashboard-darkmode.json) (performance audits errored: `first-contentful-paint`, `largest-contentful-paint`, `total-blocking-time`, `speed-index`, `unused-javascript`, `render-blocking-insight`, `lcp-breakdown-insight`).
- Dashboard: `clients/web/src/pages/lms/dashboard.tsx` (`mapPool`, `DashboardLoadingSkeleton`).
- App routes: `clients/web/src/app.tsx` (synchronous imports).
- Related plans: [LH.1](LH.1-global-dashboard-darkmode-audit-harness.md), [LH.3](LH.3-global-dashboard-darkmode-accessibility.md), [21.2](../21-mobile-offline-cross-platform/21.2-mobile-responsive-review.md), [17.5](../17-platform-performance-operability/17.5-caching-layer.md).
- External: [Web Vitals thresholds](https://web.dev/articles/vitals), [Lighthouse performance scoring](https://developer.chrome.com/docs/lighthouse/performance/performance-scoring/).
