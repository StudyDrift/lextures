# UX.9 — Role-Aware, Prioritised Dashboard

> Implementation plan. Source: [audit.md](audit.md) §5 G-6.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.9 |
| **Section** | UI/UX — Core Surfaces |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN — 18 co-equal full-width banners in a single vertical stack |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Product Design + Web |
| **Depends on** | UX.1, UX.2, UX.3, UX.7, UX.12 |
| **Unblocks** | UX.16; measurable improvement in day-one engagement |

---

## 1. Problem Statement

`pages/lms/dashboard.tsx` (1,487 lines) renders a **single vertical stack of ~18
independent, full-width, equally-weighted sections**, each gated by its own
feature flag, each with its own arbitrary accent colour (teal, violet, emerald,
amber, indigo) carrying no semantic meaning. There is no grid, no priority, no
personalisation, no density control, and no role differentiation — an instructor
scrolls past student-motivation cards to reach the grading backlog, which renders
**last**. Enabling a feature appends another banner. This is the textbook
condition for choice overload (**R-8**) and an extraneous-load tax on the first
screen every user sees (**R-1**). Against the industry reference model — KPIs at a
glance, detail on click, configuration on intent (**R-32**) — the dashboard
implements one layer, eighteen times.

## 2. Goals

- Replace the banner stack with a **prioritised, role-aware layout** answering one
  question per persona: *what should I do next?*
- Apply the three-layer disclosure contract (**R-32**): glance → detail → configure.
- Give users **autonomy** over their dashboard — reorder, collapse, dismiss
  (**R-4**).
- Make **competence** legible: progress and mastery as the primary signal, not
  points and badges (**R-5**).
- Cut the dashboard's request fan-out and improve LCP/INP.

## 3. Non-Goals

- Removing any dashboard capability. Everything currently surfaced remains
  reachable; this plan changes **prominence and disclosure**.
- Course-level dashboards (`/courses/:code`) — that is [UX.10](UX.10-course-home-and-learning-flow.md).
- The recommendation engine itself (`fetchLearnerRecommendations`) — consumed
  as-is.
- Building new analytics.
- The parent dashboard beyond applying the same shell.

## 4. Personas & User Stories

- **As a student**, I want to see what is due and where I left off in the first
  screen, without scrolling past six promotional cards.
- **As a student**, I want to see that I am making progress, so that I keep going.
- **As an instructor**, I want my grading backlog and at-risk students first,
  because that is the job.
- **As an instructor teaching four courses**, I want per-course rollups I can
  scan, not one long undifferentiated list.
- **As a parent**, I want my child's upcoming work and recent results, and nothing
  else.
- **As an administrator**, I want organisation health, not my own coursework.
- **As any user**, I want to hide the cards I do not care about, permanently.

## 5. Functional Requirements

### Structure

- **FR-1.** The dashboard MUST be composed from a **widget registry**: id, title
  key, audience, required permission, required feature flag, default rank,
  default size (`sm` = 1 col, `md` = 2 col, `lg` = full width), and a data
  dependency declaration.
- **FR-2.** Layout MUST be a **responsive grid**, not a vertical stack: 3 columns
  ≥1280px, 2 columns ≥768px, 1 column below. Widget size comes from the registry,
  so not everything is full-width.
- **FR-3.** The dashboard MUST render a **single primary "Next action" region**
  above the fold, visually dominant, distinct from all other widgets. For students
  this is the recommendation/what's-next; for instructors, the grading backlog and
  at-risk signal; for parents, upcoming work; for admins, org health.
- **FR-4.** Below the primary region, widgets MUST be ordered by registry rank
  within audience — never by feature-flag declaration order.
- **FR-5.** The dashboard MUST show at most **6 widgets by default**; the
  remainder go behind a "Show more" disclosure that persists its state (**R-9**).
- **FR-6.** Each widget MUST implement the three-layer contract (**R-32**):
  a glanceable summary; a click-through to detail; configuration only via the
  customise sheet.
- **FR-7.** Widget accent colour MUST come from the UX.1 **semantic status
  vocabulary** (info/success/warning/danger/accent) and MUST reflect meaning.
  Decorative per-widget colour is forbidden.

### Personalisation (autonomy — R-4)

- **FR-8.** Users MUST be able to reorder, resize (where the registry allows),
  collapse and **permanently dismiss** widgets.
- **FR-9.** Dismissal MUST be honoured indefinitely and MUST be reversible from a
  "Hidden widgets" list.
- **FR-10.** Personalisation MUST persist server-side per user and sync across
  devices.
- **FR-11.** A **density control** (comfortable / compact) MUST be offered and
  persisted.
- **FR-12.** "Reset to default" MUST be available.

### Role awareness

- **FR-13.** Audience MUST be derived from the viewer's actual roles across their
  enrollments, and MUST support users who are **both** student and instructor —
  such users get a segmented dashboard with an explicit switch, not a merged
  stack.
- **FR-14.** A user with no student enrollments MUST NOT see student-motivation
  widgets at all.

### Data and state

- **FR-15.** Widgets MUST load **independently and progressively**. One failing
  widget MUST NOT blank the dashboard; it MUST render its own error state with
  retry (see [UX.12](UX.12-loading-empty-error-offline-states.md)).
- **FR-16.** The current per-course fan-out (structure + grades + gradebook grid +
  feed channels + feed messages for **every** enrolled course) MUST be replaced
  by **server-side aggregation**: one summary endpoint per widget.
- **FR-17.** Above-the-fold widgets MUST be fetched eagerly; below-the-fold
  widgets MUST be deferred until visible.
- **FR-18.** Every widget MUST define its empty state as an onboarding moment
  (**R-18**) — what belongs here, why it is empty, one next action.
- **FR-19.** The first-run dashboard (new user, no data) MUST be a designed
  experience, not eighteen empty cards.

## 6. Non-Functional Requirements

- **Performance** — LCP ≤2.0 s at p75 on a mid-tier laptop over 4G; INP ≤200 ms;
  CLS ≤0.05. Initial dashboard payload MUST be **one aggregate request** for
  above-the-fold content. Dashboard route chunk ≤20 KB gzip (current 12 KB — do
  not regress badly).
- **Security** — Aggregation endpoints MUST enforce the same authorisation as the
  underlying resources; a widget MUST NOT expose data the user could not fetch
  directly. Widget registry filtering is presentation only.
- **Privacy & Compliance** — At-risk and behaviour signals shown to instructors
  are sensitive; display MUST follow the existing at-risk governance and be
  covered in the RoPA. Student-facing progress MUST NOT expose peer comparison
  without opt-in (**R-6**).
- **Accessibility** — Grid MUST be a logical DOM order matching visual order; each
  widget is a `section` with an accessible name; the primary region is announced
  first; drag-reorder MUST have a single-pointer and keyboard alternative
  ([UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md) FR-5/FR-6).
- **Scalability** — Adding a widget is a registry entry plus a summary endpoint.
  The layout must hold at 30 registered widgets without becoming a stack again.
- **Reliability** — Partial failure is normal and designed for (FR-15). Aggregate
  endpoints MUST degrade to partial results rather than 500.
- **Observability** — Emit `dashboard_widget_view`, `dashboard_widget_click`,
  `dashboard_widget_dismiss`, `dashboard_show_more`, `dashboard_reorder`,
  `dashboard_time_to_first_click`. Dismissal rate is the primary signal that a
  widget does not deserve its rank.
- **Maintainability** — One file per widget, ≤200 lines. The 1,487-line dashboard
  file is decomposed as part of this work, aligned with
  [`TD.14`](../tech_debt/TD.14-decompose-god-components.md).
- **Internationalization** — All copy from i18n keys (the dashboard is already
  well i18n'd — preserve that); grid mirrors in RTL.
- **Backward compatibility** — No capability is removed. Users who relied on a
  specific card can pin it.

## 7. Acceptance Criteria

- **AC-1.** *Given* a student at 1280×800, *When* the dashboard loads, *Then* the
  primary "next action" is visible without scrolling, and at most 6 widgets render
  before "Show more".
- **AC-2.** *Given* an instructor, *When* the dashboard loads, *Then* the grading
  backlog and at-risk signal are in the primary region, above any
  student-motivation widget.
- **AC-3.** *Given* a user who is both student and instructor, *When* the
  dashboard loads, *Then* an explicit role switch is presented and each view is
  coherent.
- **AC-4.** *Given* a user with no student enrollments, *When* the dashboard
  renders, *Then* no student-motivation widget appears in the DOM.
- **AC-5.** *Given* a user dismisses a widget, *When* they return on another
  device, *Then* it is still dismissed and recoverable from "Hidden widgets".
- **AC-6.** *Given* one widget's endpoint returns 500, *When* the dashboard loads,
  *Then* every other widget renders and the failing one shows an error state with
  a working retry.
- **AC-7.** *Given* a new user with no courses, *When* the dashboard loads,
  *Then* a designed first-run experience renders — not empty cards.
- **AC-8.** *Given* the dashboard, *When* measured on a mid-tier laptop over
  simulated 4G, *Then* LCP ≤2.0 s, INP ≤200 ms, CLS ≤0.05 at p75.
- **AC-9.** *Given* the dashboard, *When* the network panel is inspected, *Then*
  above-the-fold content comes from **one** aggregate request, not N per course.
- **AC-10.** *Given* widget accent colours, *When* audited, *Then* every one maps
  to a semantic status token and no widget uses a decorative colour.
- **AC-11.** *Given* keyboard-only operation, *When* a user reorders widgets,
  *Then* it is possible without dragging, and the result is announced.
- **AC-12.** *Given* the dashboard, *When* axe runs in all four themes, *Then* 0
  violations; DOM order matches visual order.
- **AC-13.** *Given* moderated testing with ≥8 participants per persona, *When*
  asked "what should you do next?", *Then* ≥80% answer correctly within 10 seconds.
- **AC-14.** *Given* the decomposition, *When* measured, *Then* no dashboard
  widget file exceeds 200 lines and `dashboard.tsx` is under 300.

## 8. Data Model

```sql
-- server/migrations/NNN_user_dashboard_preferences.sql
CREATE TABLE user_dashboard_preferences (
  user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  audience    text        NOT NULL,          -- 'student' | 'instructor' | 'parent' | 'admin'
  layout      jsonb       NOT NULL DEFAULT '[]'::jsonb,  -- [{ widgetId, rank, size, collapsed }]
  dismissed   jsonb       NOT NULL DEFAULT '[]'::jsonb,  -- widget ids
  density     text        NOT NULL DEFAULT 'comfortable',
  updated_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, audience),
  CONSTRAINT user_dashboard_preferences_density_chk
    CHECK (density IN ('comfortable', 'compact'))
);
```

- **Backfill** — none; absent row means defaults.
- Unknown widget ids dropped on read against the registry, so retired widgets
  self-clean.
- Cascade delete satisfies `../standards/S02-data-retention-deletion-engine.md`.

## 9. API Surface

```ts
// GET /api/v1/dashboard/summary?audience=student      (auth: self)
// One aggregate call for above-the-fold content. Degrades to partial results.
type DashboardSummary = {
  audience: 'student' | 'instructor' | 'parent' | 'admin'
  primary: PrimaryAction | null       // next action / grading backlog / org health
  widgets: Record<string, WidgetPayload | { error: string }>
  generatedAt: string
}

// GET /api/v1/dashboard/widgets/{widgetId}            (auth: self)
// Deferred, below-the-fold widgets fetch individually.

// GET  /api/v1/dashboard/preferences?audience=student (auth: self)
// PUT  /api/v1/dashboard/preferences                  (auth: self)
// DELETE /api/v1/dashboard/preferences?audience=...   (auth: self) — reset
```

- The summary endpoint MUST return `200` with per-widget error entries rather than
  failing wholesale.
- No WebSocket events. Standard per-user rate limits; summary is cacheable for a
  short TTL per user.
- **OpenAPI** — all routes documented; `make openapi-check` passes.

## 10. UI / UX

- **New pages** — none. New UI: a "Customise dashboard" sheet (reorder, resize,
  collapse, dismiss, density, reset).
- **Modified pages** — `pages/lms/dashboard.tsx` decomposed into
  `components/dashboard/widgets/*`; `pages/lms/parent/parent-dashboard.tsx` and
  `pages/admin/AdminOverview.tsx` adopt the same shell.
- **Key user flows**
  1. Student signs in → sees "Continue: *Module 4 — Cell Division*, due Thursday"
     as the dominant element → clicks → lands in the module.
  2. Instructor signs in → sees "12 submissions to grade · 3 students at risk" →
     clicks → lands in the grading queue.
  3. User opens Customise → drags (or keyboard-moves) a widget up → dismisses two
     → sets compact density.
  4. New user with no courses → sees a first-run experience with a single clear
     action.
- **States** — per widget: loading (skeleton matching final shape, **R-16**),
  empty (onboarding, **R-18**), error (message + retry, does not blank siblings),
  offline (last-known values with a staleness note), partial (some data missing,
  labelled).
- **Mobile/responsive** — single column; primary region first; "Show more" is more
  aggressive on small screens (3 widgets before disclosure).
- **Accessibility annotations** — `main` landmark; primary region has `h2` and is
  first in DOM; each widget is a `section` with `aria-labelledby`; reorder is
  keyboard-operable with live-region announcements; density control is a labelled
  radio group.
- **Copy & i18n** — reuse and extend the existing `dashboard.json` namespace;
  keep all four locales at parity. Widget titles must be **nouns describing the
  user's world** ("Due this week"), not feature names ("Learner recommendations").

## 11. AI / ML Considerations

The primary region for students consumes the existing recommendation engine
(`fetchLearnerRecommendations`, `postRecommendationEvent`) and the shipped
`ProfileRationaleChip`.

- **Model** — existing; no new model introduced.
- **Prompts** — n/a.
- **Eval metric** — click-through on the primary action, and
  `dashboard_time_to_first_click`. If CTR on the AI-chosen primary action is not
  above a recency-ordered baseline, fall back to recency.
- **Fallback path** — if recommendations are unavailable or degraded, the primary
  region MUST fall back to "most recently visited" / "soonest due" — this already
  exists as `whatsNext.degraded` and MUST be preserved.
- **Explainability** — the rationale chip MUST remain; a recommendation without a
  visible reason is not shippable (autonomy, **R-4**).
- **PII redaction / cost** — no new inference at render time; the summary endpoint
  reads precomputed recommendations. No per-render model cost.

## 12. Integration Points

- **External** — none.
- **Internal**
  - `clients/web/src/pages/lms/dashboard.tsx` — decomposed
  - `clients/web/src/components/dashboard/**` — existing cards become widgets
  - `clients/web/src/components/{study-stats,gamification,credentials,self-paced,study-reminders,intro-course,onboarding,research,notebook}/**`
    — existing cards adopt the widget contract
  - `clients/web/src/lib/courses-api.ts` — the per-course fan-out is replaced by
    aggregation (coordinates with [`TD.12`](../tech_debt/TD.12-split-courses-api-module.md))
  - `clients/web/src/lib/dashboard-course-prefetch.ts`, `lib/schedule-idle.ts`
  - `clients/web/src/lib/recommendation-nav.ts`
  - `server/internal/httpserver` — summary, widget, preference routes
- **Events** — dashboard telemetry into `server/internal/telemetry`.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.1](UX.1-semantic-design-token-system.md) (status
  colour), [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md) (cards,
  disclosure), [UX.3](UX.3-typography-and-reading-system.md) (scale),
  [UX.7](../../completed/ui-ux/UX.7-navigation-information-architecture.md) (audience model reused),
  [UX.12](UX.12-loading-empty-error-offline-states.md) (per-widget states).
- **Should ship alongside** — [`TD.14`](../tech_debt/TD.14-decompose-god-components.md)
  for the 1,487-line decomposition.
- **Must ship before** — [UX.16](UX.16-progress-motivation-and-learner-agency.md).
- **Shared infra** — server-side aggregation capability; participant recruitment.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Demoting a feature's card is read internally as demoting the feature | **H** | M | Ranks are set from `dashboard_widget_click` telemetry, not opinion; every widget remains available and pinnable; publish the ranking rule |
| Server-side aggregation is a substantial backend effort | **H** | **H** | Ship the layout first against existing endpoints; aggregation as phase 2 with the widget contract already in place. Do **not** block the UX on the backend |
| Users personalise into a broken state and blame the product | M | M | "Reset to default"; support tooling to view/clear preferences; sane guardrails (primary region cannot be dismissed) |
| Dual-role (student + instructor) users get a confusing switch | M | M | Explicit, persistent switch mirroring the existing "View as" affordance they already know |
| Partial-failure design is harder to test than all-or-nothing | M | M | Widget error states are a first-class E2E case; MSW fixtures include per-widget failures |
| Grid layout regresses on very wide screens (widgets stretch) | L | M | Max content width; widget size classes from the registry |

## 15. Rollout Plan

- **Feature flag** — `ffDashboardV2`, default off. This is a visible experience
  change and must be reversible.
- **Sequencing**
  1. Instrument the current dashboard; collect widget view/click for **4 weeks**
     to derive ranks.
  2. Widget registry + contract; wrap existing cards **without changing layout**
     (pure refactor, unflagged). Decomposes the god component.
  3. Grid layout + primary region + "Show more" behind `ffDashboardV2`.
  4. Per-widget independent loading and error states.
  5. Personalisation + density.
  6. Server-side aggregation replaces fan-out (perf phase).
  7. First-run experience.
  8. Internal → 10% → 50% → GA, watching engagement and time-to-first-click.
- **Dogfood** — internal org, 3 weeks, both personas.
- **GA criteria** — AC-1…AC-14 green; AC-13 comprehension ≥80%; no decrease in
  engagement with any individual widget beyond the expected redistribution;
  CWV targets met.
- **Rollback** — `ffDashboardV2` off. The registry refactor (step 2) stays.

## 16. Test Plan

- **Unit** — registry filtering by audience/permission/flag; rank ordering;
  layout computation per breakpoint; personalisation merge (defaults + user
  layout + dismissals); unknown-widget dropping; density.
- **Integration** — summary endpoint authz; partial-failure envelope; preference
  CRUD authz (self only); aggregation correctness against seeded multi-course data.
- **End-to-end** — Playwright per persona: primary action above the fold;
  "Show more"; dismiss → reload → still dismissed; one widget 500 → others render
  → retry succeeds; first-run with zero courses; dual-role switch.
- **Security** — assert a widget cannot surface data the user cannot fetch
  directly; cross-user preference access; at-risk data not exposed to students.
- **Accessibility** — axe × 4 themes (AC-12); DOM-order-matches-visual-order
  assertion; screen-reader script: land on dashboard, hear the primary action
  first, traverse widgets, reorder by keyboard; reduced-motion honoured on
  reveal/stagger.
- **Performance / load** — Lighthouse CI on the dashboard route (extend the
  existing `lighthouse:dashboard:dark` harness) gating LCP/INP/CLS (AC-8); request
  count assertion (AC-9); aggregation endpoint p95 under a 20-course fixture.
- **User research** — 8+ moderated sessions per persona on the "what should you do
  next?" task (AC-13); dismissal-rate review after 4 weeks.
- **Manual exploratory** — QA matrix of audience × feature-flag combinations at
  the 10 most common org configurations; parent and admin shells.

## 17. Documentation & Training

- **End-user** — help-centre: "Your dashboard" and "Customising your dashboard";
  a one-time in-product tour for existing users on first exposure to
  `ffDashboardV2`.
- **Admin / instructor** — note that dashboard content follows enabled features
  and the viewer's roles.
- **Engineer** — `docs/guides/dashboard-widgets.md`: the widget contract, how to
  register one, size classes, the mandatory four states, and why a widget may not
  choose its own accent colour.
- **API reference** — OpenAPI for summary/widget/preference routes.
- **Runbook** — "A user's dashboard is empty/broken": inspect and reset
  preferences.

## 18. Open Questions

1. Should dual-role users get a **switch** or a **merged** dashboard with two
   sections? *Recommendation: switch — merging is how the current stack got long.*
   Validate in moderated testing.
2. Does server-side aggregation belong in a new `dashboard` service package or in
   existing handlers? Coordinate with
   [`TD.6`](../tech_debt/TD.6-decompose-httpserver-package.md).
3. What is the default density — comfortable or compact? Likely persona-dependent
   (instructors want compact, students comfortable). Test.
4. Should the primary region be dismissible? *Recommendation: no — it is the
   dashboard's reason to exist.*
5. How long is the 4-week instrumentation window allowed to gate the work? Can the
   registry refactor proceed in parallel? *Recommendation: yes, in parallel.*
6. Do we surface a leaderboard widget at all by default, given **R-6**?
   *Recommendation: off by default, opt-in only — decide with the K-12 product
   owner and record in [UX.16](UX.16-progress-motivation-and-learner-agency.md).*

## 19. References

- Existing files: `clients/web/src/pages/lms/dashboard.tsx` (1,487 lines; the
  banner stack is at lines 699–960), `clients/web/src/components/dashboard/**`,
  `clients/web/src/lib/dashboard-course-prefetch.ts`,
  `clients/web/src/lib/recommendation-nav.ts`,
  `clients/web/src/pages/lms/parent/parent-dashboard.tsx`,
  `clients/web/src/pages/admin/AdminOverview.tsx`,
  `docs/lighthouse/global-dashboard-darkmode.json`
- Research: [research.md](research.md) R-1, R-4, R-5, R-6, R-8, R-9, R-16, R-18,
  R-31, R-32
- Audit: [audit.md](audit.md) G-6, G-1, G-9, G-15, G-17
- Related plans: [UX.7](../../completed/ui-ux/UX.7-navigation-information-architecture.md),
  [UX.12](UX.12-loading-empty-error-offline-states.md),
  [UX.16](UX.16-progress-motivation-and-learner-agency.md),
  [UX.17](UX.17-perceived-performance-and-web-vitals-budget.md),
  [`../tech_debt/TD.14-decompose-god-components.md`](../tech_debt/TD.14-decompose-god-components.md),
  [`../tech_debt/TD.12-split-courses-api-module.md`](../tech_debt/TD.12-split-courses-api-module.md)
