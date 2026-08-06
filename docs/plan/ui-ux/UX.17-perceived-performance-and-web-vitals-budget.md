# UX.17 — Perceived Performance and Web Vitals Budget

> Implementation plan. Source: [audit.md](audit.md) §7 G-17.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.17 |
| **Section** | UI/UX — Cross-cutting |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL — bundle checks exist; no CWV budget, no RUM |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web + Platform |
| **Depends on** | UX.12 |
| **Unblocks** | UX.9, UX.10, UX.11, UX.14 quality gates |

---

## 1. Problem Statement

The entry bundle is **245,104 B gzipped** (~1 MB parsed) for an application whose
first meaningful screen is a dashboard. Route-level code splitting and bundle-size
CI checks exist — genuinely good — but there is **no Core Web Vitals budget per
route class**, no real-user monitoring, and the only Lighthouse artefact is a
single dark-mode dashboard run. The dashboard additionally fans out **N+1
requests**, fetching structure, grades, gradebook grid, feed channels and feed
messages for *every* enrolled course, which is what makes an already-noisy screen
slow. INP is the metric most at risk, given god components with 100 `useState`
hooks in the learning flow. Performance is a UX property before it is an
engineering one: per **R-1**, waiting and jank are extraneous load in a learning
context.

## 2. Goals

- Establish an enforced **Core Web Vitals budget per route class**.
- Instrument **real-user monitoring** so budgets are set on truth, not lab runs.
- Reduce the entry bundle and eliminate the dashboard N+1.
- Make **perceived** performance a first-class target alongside measured
  performance.
- Make every performance target a **CI gate**, not a report nobody reads.

## 3. Non-Goals

- Backend query optimisation beyond the aggregation endpoints these plans require.
- Infrastructure/CDN changes.
- Native client performance.
- Loading-state design — that is [UX.12](UX.12-loading-empty-error-offline-states.md);
  this plan owns the measurement and the budget.

## 4. Personas & User Stories

- **As a K-12 student on a school Chromebook over shared WiFi**, I want the app to
  become usable quickly.
- **As an instructor**, I want typing a grade to feel instant.
- **As any user**, I want the page not to jump as it loads.
- **As an engineer**, I want CI to tell me my change made the dashboard slower.
- **As a platform owner**, I want to know real users' p75 vitals, not a synthetic
  score.

## 5. Functional Requirements

### Budgets

- **FR-1.** Routes MUST be classified and given explicit budgets:

  | Route class | LCP (p75) | INP (p75) | CLS (p75) | JS (gzip) |
  |---|---|---|---|---|
  | **Auth** (login, signup) | 1.5 s | 200 ms | 0.05 | 80 KB |
  | **Dashboard / course home** | 2.0 s | 200 ms | 0.05 | 180 KB |
  | **Learning surfaces** (module, content, quiz) | 2.0 s | 200 ms | 0.05 | 220 KB |
  | **Data-dense** (gradebook, enrollments, admin) | 2.5 s | 200 ms | 0.10 | 260 KB |
  | **Editors** (syllabus, portfolio, whiteboard) | 3.0 s | 300 ms | 0.10 | 400 KB |

  Measured on a **mid-tier Chromebook profile over simulated 4G** — the K-12
  reality, not a developer laptop.

- **FR-2.** The entry bundle MUST be reduced to **≤180 KB gzip** from 245 KB, and a
  CI gate MUST prevent regression.
- **FR-3.** Budgets MUST be enforced in CI via Lighthouse CI, extending the
  existing `lighthouse:dashboard:dark` harness to cover one representative route
  per class.
- **FR-4.** A budget breach MUST **fail the build**, not warn.

### Real-user monitoring

- **FR-5.** RUM MUST collect LCP, INP, CLS, TTFB and FCP from production, attributed
  to route, viewport bucket, connection type and device class.
- **FR-6.** RUM MUST feed the existing observability stack
  (`server/internal/telemetry`) with dashboards and alerts on p75 regression.
- **FR-7.** RUM MUST be **privacy-preserving**: no PII, no full URLs with
  identifiers (route patterns only), sampled, and disclosed in the privacy notice.
- **FR-8.** Lab budgets (FR-1) MUST be reviewed against RUM p75 quarterly and
  adjusted to reality.

### Loading strategy

- **FR-9.** Route-level code splitting MUST continue; heavy dependencies
  (`pdfjs-dist`, `katex`, `@xyflow/react`, `@tiptap/*`, `@uiw/react-codemirror`,
  `hls.js`, `mark.js`, `react-syntax-highlighter`) MUST be loaded **only** on
  routes that need them, verified by a chunk-composition check.
- **FR-10.** Above-the-fold data MUST be fetched in **one** request per surface;
  the dashboard N+1 fan-out MUST be replaced by server-side aggregation
  ([UX.9](UX.9-role-aware-dashboard.md) FR-16).
- **FR-11.** Below-the-fold content MUST be deferred until visible.
- **FR-12.** Critical fonts MUST be preloaded; `font-display: swap` retained with a
  metric-compatible fallback to avoid layout shift.
- **FR-13.** Images MUST be responsive (`srcset`/`sizes`), lazy below the fold, and
  MUST reserve space to prevent CLS.
- **FR-14.** Long lists MUST virtualise beyond 200 items
  ([UX.11](UX.11-data-table-and-gradebook-system.md) FR-4).

### Perceived performance

- **FR-15.** Skeletons matching final layout MUST be the default loading treatment
  (**R-16**, [UX.12](UX.12-loading-empty-error-offline-states.md) FR-3).
- **FR-16.** Optimistic UI MUST be used for small mutations (**R-23**,
  [UX.13](UX.13-feedback-undo-and-destructive-actions.md) FR-21).
- **FR-17.** Likely next navigations MUST be prefetched on intent (hover/focus),
  respecting Save-Data and metered connections.
- **FR-18.** Any interaction that cannot respond within 100 ms MUST show immediate
  acknowledgement.

### Runtime

- **FR-19.** No long task >50 ms during typical interactions on the target device
  profile.
- **FR-20.** Re-render hot spots MUST be measured and fixed; the god components
  (100 `useState` in one file) are the known offenders and are addressed by
  [`TD.14`](../tech_debt/TD.14-decompose-god-components.md).
- **FR-21.** A CI performance-profiling check SHOULD run React Profiler on the 5
  heaviest surfaces and fail on a significant regression in render count.

## 6. Non-Functional Requirements

- **Performance** — this plan *is* the performance requirement.
- **Security** — RUM payloads MUST NOT contain identifiers, tokens or query
  parameters. Prefetching MUST NOT prefetch authenticated mutations or leak
  navigation intent to third parties (all telemetry is first-party).
- **Privacy & Compliance** — RUM is behavioural telemetry; it MUST appear in the
  RoPA (`../standards/S05-ropa-data-inventory-mapping.md`) and the privacy notice,
  MUST be sampled, and MUST respect Do-Not-Track/consent where applicable.
- **Accessibility** — Prefetching MUST NOT interfere with assistive technology.
  Skeletons MUST NOT be announced as content. Reduced-motion MUST be honoured in
  any progress animation.
- **Scalability** — Budgets are per route class, so new routes inherit one.
- **Reliability** — RUM collection MUST fail silently; telemetry must never break a
  page.
- **Observability** — this plan *is* the observability requirement for the frontend.
  Extends the shipped 17.7 telemetry layer.
- **Maintainability** — One budget file; one RUM module.
- **Internationalization** — Locale bundles MUST be lazily loaded per namespace
  ([UX.15](UX.15-i18n-coverage-and-rtl-completion.md) §6) and MUST NOT inflate the
  entry bundle.
- **Backward compatibility** — No user-visible behaviour change beyond speed.

## 7. Acceptance Criteria

- **AC-1.** *Given* the built app, *When* measured, *Then* the entry bundle is
  ≤180 KB gzip and CI fails on regression.
- **AC-2.** *Given* one representative route per class, *When* Lighthouse CI runs
  on the mid-tier Chromebook / 4G profile, *Then* every FR-1 budget is met and a
  breach fails the build.
- **AC-3.** *Given* production traffic, *When* RUM reports, *Then* p75 LCP, INP and
  CLS are available per route class, viewport bucket, connection type and device
  class.
- **AC-4.** *Given* a p75 regression beyond budget for 24 hours, *When* it occurs,
  *Then* an alert fires to the owning team.
- **AC-5.** *Given* the dashboard, *When* the network panel is inspected, *Then*
  above-the-fold content is one request, not N per course.
- **AC-6.** *Given* any route, *When* its chunk composition is analysed, *Then* no
  heavy dependency is present on a route that does not use it.
- **AC-7.** *Given* the top 20 routes, *When* loaded, *Then* CLS ≤0.05 and no
  font-swap-induced shift.
- **AC-8.** *Given* typical interactions on the target device, *When* profiled,
  *Then* there is no long task >50 ms.
- **AC-9.** *Given* a user on a metered or Save-Data connection, *When* they
  browse, *Then* prefetching is disabled.
- **AC-10.** *Given* RUM payloads, *When* inspected, *Then* they contain no PII,
  no tokens and no identifier-bearing URLs.
- **AC-11.** *Given* an interaction that cannot complete within 100 ms, *When*
  triggered, *Then* immediate acknowledgement is shown.
- **AC-12.** *Given* a list of 1,000 items, *When* rendered, *Then* it virtualises
  and scrolls at 60fps.

## 8. Data Model

RUM events are time-series telemetry, not relational data. They flow into the
existing observability stack (`server/internal/telemetry`, per the shipped 17.7
work) rather than PostgreSQL.

```ts
type WebVitalEvent = {
  metric: 'LCP' | 'INP' | 'CLS' | 'TTFB' | 'FCP'
  value: number
  routePattern: string          // '/courses/:code/modules' — never a concrete id
  routeClass: 'auth' | 'dashboard' | 'learning' | 'dense' | 'editor'
  viewportBucket: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  connection: string | null     // effectiveType
  deviceMemory: number | null
  sessionSample: boolean
}
```

- **Retention** — aggregated metrics retained per the existing telemetry retention
  policy; raw events short-lived.
- No tables, columns, enums, indexes, migrations or backfill in PostgreSQL.

## 9. API Surface

```ts
// POST /api/v1/telemetry/web-vitals          (auth: optional — works pre-login)
type WebVitalsBatch = { events: WebVitalEvent[] }
// Responds 204. Fire-and-forget; uses sendBeacon where available.
```

- MUST be heavily rate-limited and MUST reject oversized batches.
- MUST validate `routePattern` against a known-pattern allowlist server-side, so a
  malicious client cannot inject high-cardinality labels into metrics.
- No WebSocket events.
- **OpenAPI** — documented; `make openapi-check` passes.

## 10. UI / UX

- **New pages** — an internal performance dashboard (staff-gated), or Grafana
  panels in the existing observability stack — whichever the platform team prefers.
- **Modified pages** — no user-visible changes beyond speed and the FR-15/FR-18
  perceived-performance treatments already owned by UX.12/UX.13.
- **Key user flows**
  1. User navigates → skeleton appears immediately → content fills without shift.
  2. User hovers a course link → the route is prefetched → the click feels instant.
  3. Engineer opens a PR → CI reports the CWV and bundle delta for affected routes.
- **States** — n/a beyond UX.12.
- **Mobile/responsive** — the target device profile *is* a small, weak device;
  budgets are set there.
- **Accessibility annotations** — prefetch must not move focus or announce
  anything; skeletons must not be read as content.
- **Copy & i18n** — no new user-facing copy.

## 11. AI / ML Considerations

Not AI-touching. One constraint on AI surfaces as consumers: streaming AI
responses MUST not be counted against INP through continuous re-render — they MUST
batch updates (suggested ≤10 Hz) and MUST not cause layout shift as tokens arrive.

## 12. Integration Points

- **External** — `web-vitals` library (small); Lighthouse CI in the existing CI.
- **Internal**
  - `clients/web/vite.config.ts`, `scripts/check-bundle-size.mjs`,
    `check-tool-bundle-size.mjs`, `scripts/bundle-baseline.json`
  - `clients/web/src/lazy-pages.ts` — route splitting
  - `clients/web/src/lib/schedule-idle.ts`, `lib/dashboard-course-prefetch.ts`
  - `clients/web/src/main.tsx` — RUM init
  - `e2e/` — Lighthouse CI harness (extends `lighthouse:dashboard:dark`)
  - `server/internal/telemetry` — metrics ingestion (shipped 17.7)
  - `docs/lighthouse/` — artefacts
- **Events** — web-vitals into the existing metrics pipeline.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.12](UX.12-loading-empty-error-offline-states.md)
  (skeletons are the perceived-performance mechanism).
- **Should ship early** — the **measurement** half (RUM + budgets + CI gates) should
  land *before* UX.9/UX.10/UX.11 so those plans can be held to a budget rather than
  measured afterwards. **Recommendation: split this plan — instrumentation first,
  optimisation later.**
- **Depends for optimisation on** — [UX.9](UX.9-role-aware-dashboard.md) FR-16
  (aggregation), [`TD.14`](../tech_debt/TD.14-decompose-god-components.md) (INP).
- **Shared infra** — CI runners capable of consistent Lighthouse runs; the
  telemetry pipeline.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Lighthouse CI is noisy and the gate gets disabled | **H** | **H** | Multiple runs with median; generous variance thresholds; gate on a 3-run median and a trend, not a single number; treat flakiness as a bug in the harness |
| Budgets are set too tight and block all delivery | M | **H** | Set initial budgets from **measured current p75 + a small improvement**, then ratchet quarterly (FR-8). Never set an aspirational budget as a gate on day one |
| RUM adds weight and privacy surface | M | M | `web-vitals` is ~2 KB; sampled; route patterns only; disclosed in the privacy notice and RoPA |
| Prefetching wastes bandwidth for school networks | M | M | Respect Save-Data and `effectiveType`; prefetch on intent only, never speculatively at scale |
| Entry-bundle reduction from 245→180 KB proves hard | **H** | M | Analyse composition first (`npm run analyze` exists); most likely wins are moving i18n, icons and rarely-used providers out of the entry chunk; treat 180 KB as a target with staged milestones (220 → 200 → 180) |
| High-cardinality route labels blow up the metrics backend | M | M | Server-side allowlist of route patterns (§9) |
| Perf work competes with the rest of the UX programme | **H** | M | Instrumentation is cheap and should land early; heavy optimisation follows the surfaces it measures |

## 15. Rollout Plan

- **Feature flag** — `ffWebVitalsRum` gates RUM collection (so it can be disabled
  instantly if it misbehaves). Budgets and CI gates are not user-facing.
- **Sequencing**
  1. **Instrumentation first**: RUM + telemetry dashboards, behind
     `ffWebVitalsRum`, sampled at 10%.
  2. Collect **4 weeks** of production p75 per route class.
  3. Set budgets from measured reality + a modest improvement (FR-1 table becomes
     concrete).
  4. Lighthouse CI harness extended to one route per class; gates as **warnings**.
  5. Chunk-composition check; heavy-dependency isolation.
  6. Entry-bundle reduction in staged milestones.
  7. Dashboard aggregation (with UX.9); INP work (with TD.14).
  8. Gates flipped to **failing**; quarterly budget review begins.
- **Dogfood** — internal org at 100% RUM sampling.
- **GA criteria** — AC-1…AC-12 green; budgets met for 14 consecutive days in RUM;
  no CI flakiness above an agreed threshold.
- **Rollback** — `ffWebVitalsRum` off disables collection. CI gates can be
  downgraded to warnings without a code change.

## 16. Test Plan

- **Unit** — vitals collection and batching; route-pattern derivation (must never
  emit a concrete id); sampling logic; Save-Data/`effectiveType` gating of
  prefetch; sendBeacon fallback.
- **Integration** — telemetry ingestion authz and rate limiting; oversized-batch
  rejection; route-pattern allowlist rejection of unknown labels.
- **End-to-end** — Playwright + Lighthouse: one route per class against the budget
  (AC-2); network-panel request-count assertion on the dashboard (AC-5);
  chunk-composition assertion (AC-6).
- **Security** — inspect RUM payloads for PII, tokens and identifier-bearing URLs
  (AC-10); attempt label-cardinality injection; verify prefetch never issues a
  mutating request.
- **Accessibility** — verify prefetch does not move focus or announce; verify
  skeletons are not read as content; reduced-motion honoured.
- **Performance / load** — the plan's own acceptance criteria; plus a load test of
  the telemetry ingestion endpoint at expected volume.
- **Manual exploratory** — throttled 3G and mid-tier Chromebook sessions across the
  8 critical journeys, noting anything that *feels* slow even when it measures fine
  — perceived performance is the point.

## 17. Documentation & Training

- **End-user** — none.
- **Admin** — none.
- **Engineer** — `docs/guides/performance-budgets.md`: the route classes and their
  budgets, how to run Lighthouse CI locally, how to read the RUM dashboards, how to
  request a budget change, the heavy-dependency isolation rule.
- **API reference** — OpenAPI for the telemetry endpoint.
- **Runbook** — "A performance budget is failing CI": how to profile, read the
  bundle analysis, and identify the regressing chunk. Extend
  `docs/runbooks/` and `docs/monitoring/`.
- **Compliance** — add RUM to the RoPA and privacy notice.
- **Update** `AGENTS.md` with the Lighthouse CI command alongside the existing
  `lighthouse:dashboard:dark` entry.

## 18. Open Questions

1. Should this plan be **split** into UX.17a (instrumentation, early) and UX.17b
   (optimisation, after the surface plans)? *Recommendation: yes — measurement must
   precede the surfaces it will judge.*
2. What is the actual production device and connection mix? RUM answers this and
   should be collected before budgets are fixed.
3. Where do frontend vitals live — the existing Prometheus/Grafana stack from 17.7,
   or a dedicated RUM product? *Recommendation: existing stack — one pane of glass,
   no new vendor, no new data-processor agreement.*
4. Is 180 KB the right entry-bundle target, or should we aim lower by moving i18n
   and icons out entirely? Bundle analysis should set this.
5. Does RUM require consent under GDPR in our configuration, or is it legitimate-
   interest first-party performance monitoring? Needs a privacy-counsel answer
   before production rollout.
6. Should budgets differ by market segment (K-12 school networks vs higher-ed
   campus WiFi)? RUM segmentation will show whether one budget is defensible.

## 19. References

- Existing files: `clients/web/vite.config.ts`,
  `clients/web/scripts/check-bundle-size.mjs`, `scripts/bundle-baseline.json`
  (entry 245,104 B gzip), `clients/web/src/lazy-pages.ts`,
  `clients/web/src/lib/{schedule-idle,dashboard-course-prefetch}.ts`,
  `docs/lighthouse/global-dashboard-darkmode.json`, `e2e/`,
  `server/internal/telemetry`
- Research: [research.md](research.md) R-1, R-16, R-17, R-23
- Audit: [audit.md](audit.md) G-17, G-9, G-15
- External: [web.dev — Core Web Vitals](https://web.dev/articles/vitals),
  [Lighthouse CI](https://github.com/GoogleChrome/lighthouse-ci),
  [web-vitals library](https://github.com/GoogleChrome/web-vitals)
- Related plans: [UX.9](UX.9-role-aware-dashboard.md),
  [UX.11](UX.11-data-table-and-gradebook-system.md),
  [UX.12](UX.12-loading-empty-error-offline-states.md),
  [UX.14](UX.14-responsive-and-small-viewport-experience.md),
  [`../tech_debt/TD.14-decompose-god-components.md`](../tech_debt/TD.14-decompose-god-components.md),
  `../../completed/lighthouse/`, `../../completed/17-platform-performance-operability/`
