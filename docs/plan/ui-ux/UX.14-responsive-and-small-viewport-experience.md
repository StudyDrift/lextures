# UX.14 — Responsive and Small-Viewport Experience

> Implementation plan. Source: [audit.md](audit.md) §6 G-11.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.14 |
| **Section** | UI/UX — Cross-cutting |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN — 32% of components have any responsive prefix |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Web + Product Design |
| **Depends on** | UX.1, UX.2, UX.3, UX.5, UX.11 |
| **Unblocks** | Tablet/Chromebook adoption; WCAG 1.4.10 Reflow conformance |

---

## 1. Problem Statement

Only **257 of 795 component files (32%)** contain any responsive breakpoint
prefix, and usage is heavily skewed to `sm:` (659 occurrences) with `lg:` at 118
and `xl:` at 19 — meaning most of the product has a single fixed layout. **18 of
99 tables** lack `overflow-x-auto` and will force horizontal page scroll. Touch
targets are unverified: 177 uses of `h-6`/`h-7`/`h-8` against 140 uses of a ≥44px
class. Native iOS/Android clients absorb much of the phone traffic, but the web
app is used daily on Chromebooks, tablets and small laptops — the dominant K-12
device profile — and per **R-2**, small-screen UI problems raise learners'
extraneous cognitive load specifically. WCAG 1.4.10 (Reflow) requires content to
work at 320 CSS px equivalent without two-dimensional scrolling; that is currently
unverified across the product.

## 2. Goals

- Guarantee every route is **usable and conformant** from 320 px to 2560 px.
- Eliminate page-level horizontal scrolling everywhere.
- Meet touch-target minimums on touch-primary devices.
- Establish **responsive coverage** as a measured, ratcheting CI metric.
- Optimise deliberately for the **Chromebook/tablet** profile, not just phone and
  desktop.

## 3. Non-Goals

- Native iOS/Android clients (`../mobile/`, `../../completed/mobile/`).
- Building a separate mobile web app or `m.` site.
- Feature parity trade-offs — the aim is that everything works, not that everything
  is identical at every size.
- Offline behaviour — that is [UX.12](UX.12-loading-empty-error-offline-states.md).

## 4. Personas & User Stories

- **As a K-12 student on a Chromebook (1366×768)**, I want the course page to fit
  without horizontal scrolling.
- **As an instructor on an iPad**, I want to grade without pinch-zooming.
- **As a student on a phone browser**, I want to read a module and submit an
  assignment — the two things I actually do on a phone.
- **As a low-vision user zooming to 400%**, I want content to reflow into one
  column rather than clip.
- **As a user in a split-screen window**, I want the layout to adapt to the
  viewport, not the device.
- **As an engineer**, I want to know before merge that my page breaks at 390 px.

## 5. Functional Requirements

- **FR-1.** A **breakpoint contract** MUST be defined and documented as UX.1
  tokens, with a named purpose per breakpoint:

  | Token | Min width | Target |
  |---|---|---|
  | `base` | 320 | Phone portrait; WCAG 1.4.10 floor |
  | `sm` | 640 | Phone landscape / small tablet |
  | `md` | 768 | Tablet portrait |
  | `lg` | 1024 | **Chromebook / tablet landscape — the primary K-12 target** |
  | `xl` | 1280 | Laptop |
  | `2xl` | 1536 | Desktop |

- **FR-2.** Every route MUST be usable at **320 CSS px** with no two-dimensional
  scrolling (WCAG 1.4.10 Reflow).
- **FR-3.** **No page may scroll horizontally at any breakpoint.** Wide content
  (tables, code, diagrams, wide media) MUST scroll inside its own container with a
  visible affordance.
- **FR-4.** All 99 tables MUST route through the
  [UX.11](UX.11-data-table-and-gradebook-system.md) `DataTable`, which owns
  responsive behaviour (card reflow or contained scroll).
- **FR-5.** Touch targets MUST be ≥44×44 CSS px on touch-primary devices
  (`pointer: coarse`), and ≥24×24 everywhere per
  [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md).
- **FR-6.** Layout MUST respond to **container size where the component is
  reusable**, using container queries, rather than viewport width alone — so a
  card behaves correctly in a sidebar and in a main column.
- **FR-7.** The app shell MUST adapt: sidebar → drawer below `md`; top bar
  condenses without losing the search affordance
  ([UX.7](../../completed/ui-ux/UX.7-navigation-information-architecture.md) FR-16); focus/reading bars
  remain usable.
- **FR-8.** Modals MUST become **bottom sheets** below `md`, with drag-to-dismiss
  **and** a visible close control (drag alone is not accessible).
- **FR-9.** Multi-column layouts MUST collapse in a defined, tested order, and DOM
  order MUST match visual order at every breakpoint.
- **FR-10.** Long forms MUST be single-column below `md` with a sticky save action.
- **FR-11.** Text MUST reflow, never truncate, in content contexts. Truncation is
  permitted only in dense list/table cells and MUST expose the full value on
  focus/hover and to AT.
- **FR-12.** The viewport meta MUST permit zoom (`user-scalable` MUST NOT be
  disabled, `maximum-scale` MUST NOT be set) — WCAG 1.4.4.
- **FR-13.** Safe-area insets MUST be honoured (`env(safe-area-inset-*)`) for
  notched devices and the iOS/Android in-app browsers.
- **FR-14.** A **responsive coverage** CI metric MUST be computed — the percentage
  of route-level components with a verified ≤390 px layout — and MUST NOT decrease.
- **FR-15.** Visual regression MUST run at **390 / 768 / 1024 / 1440 px** for the
  top 40 routes.

## 6. Non-Functional Requirements

- **Performance** — Small viewports imply weaker devices. LCP ≤2.5 s at p75 on a
  mid-tier Chromebook over 4G; INP ≤200 ms. Images MUST be responsive
  (`srcset`/`sizes`); off-screen media MUST be lazy.
- **Security** — None specific. Bottom sheets MUST not expose content behind them
  to screen readers (`inert`, per [UX.4](../../completed/ui-ux/UX.4-aria-widget-and-focus-management-remediation.md)).
- **Privacy & Compliance** — Delivers WCAG 2.1 SC 1.4.4 (Resize Text), 1.4.10
  (Reflow), 1.3.4 (Orientation), and 2.2 SC 2.5.8 (Target Size).
- **Accessibility** — Orientation MUST NOT be locked (SC 1.3.4). Reflow at 400%
  zoom MUST be verified as part of the same work.
- **Scalability** — Responsive behaviour lives in UX.2 components, so new features
  inherit it.
- **Reliability** — Layout MUST not depend on JS measurement where CSS can express
  it; resize MUST not cause layout thrash.
- **Observability** — Emit viewport-bucketed CWV and a `horizontal_overflow_detected`
  beacon (a small runtime check in non-production builds) to catch regressions
  early.
- **Maintainability** — One breakpoint contract; arbitrary breakpoint values
  forbidden by lint.
- **Internationalization** — German and Arabic strings are materially longer/wider;
  responsive testing MUST include `de`-length pseudo-locale and `ar` RTL.
- **Backward compatibility** — Desktop layouts must not regress; visual regression
  at 1440 px is part of the gate.

## 7. Acceptance Criteria

- **AC-1.** *Given* every route at 320 px, *When* rendered, *Then* there is no
  two-dimensional scrolling and all functionality is available.
- **AC-2.** *Given* every route at 390 / 768 / 1024 / 1440 px, *When* rendered,
  *Then* `document.documentElement.scrollWidth <= clientWidth` (no page-level
  horizontal scroll).
- **AC-3.** *Given* any table at 390 px, *When* rendered, *Then* it reflows to
  cards or scrolls within its own container with a visible affordance.
- **AC-4.** *Given* `pointer: coarse`, *When* targets are measured, *Then* all
  interactive targets are ≥44×44 CSS px or on the justified exception list.
- **AC-5.** *Given* 400% browser zoom at 1280 px, *When* the top 20 routes are
  exercised, *Then* content reflows to a single column with no loss of content or
  function.
- **AC-6.** *Given* a modal below `md`, *When* opened, *Then* it presents as a
  bottom sheet with a visible close control and correct focus management.
- **AC-7.** *Given* any breakpoint, *When* the DOM is inspected, *Then* source
  order matches visual order.
- **AC-8.** *Given* the viewport meta tag, *When* inspected, *Then* zoom is not
  disabled and `maximum-scale` is unset.
- **AC-9.** *Given* device rotation, *When* it occurs, *Then* content is available
  in both orientations (no orientation lock).
- **AC-10.** *Given* the responsive coverage check, *When* CI runs, *Then* coverage
  ≥95% and the gate fails on any decrease.
- **AC-11.** *Given* a mid-tier Chromebook over simulated 4G, *When* the top 10
  routes load, *Then* LCP ≤2.5 s and INP ≤200 ms at p75.
- **AC-12.** *Given* `ar` (RTL) and a long-string pseudo-locale, *When* the top 40
  routes render at 390 px, *Then* no overflow, clipping or overlap occurs.
- **AC-13.** *Given* moderated testing with ≥6 participants on Chromebooks and
  tablets, *When* they complete the 8 critical journeys, *Then* ≥90% succeed
  without zooming or horizontal scrolling.

## 8. Data Model

None. UX.14 is entirely client-side. No tables, columns, enums, indexes,
migrations or backfill.

## 9. API Surface

None. No HTTP or WebSocket changes, no rate-limit considerations, no OpenAPI
changes. *(Responsive images may require the media service to expose multiple
renditions; if so that is an existing capability of the media pipeline and is
consumed, not extended.)*

## 10. UI / UX

- **New pages** — none.
- **Modified pages** — potentially all; prioritised by traffic: dashboard, course
  home, module list, item pages, quiz taking, gradebook, enrollments, settings,
  admin lists.
- **Key user flows** (all validated at 390 / 768 / 1024 px)
  1. Student reads a module and submits an assignment on a phone.
  2. Student takes a quiz on a Chromebook.
  3. Instructor grades a column on an iPad.
  4. Instructor changes a course setting on a tablet.
  5. Admin reviews a user list on a small laptop.
- **States** — every state from [UX.12](UX.12-loading-empty-error-offline-states.md)
  MUST be verified at small viewports; error and empty states are frequently the
  ones that break.
- **Mobile/responsive behaviour** — this plan *is* the responsive behaviour.
  Specific decisions: sidebar→drawer at `md`; modal→sheet at `md`; table→cards or
  contained scroll at `md`; multi-column forms→single column at `md`; primary
  actions become sticky bottom bars on small viewports.
- **Accessibility annotations** — bottom sheets follow the UX.4 overlay contract;
  sticky bars must not obscure focus (UX.5 FR-8); drag-to-dismiss always has a
  button alternative (UX.5 FR-5).
- **Copy & i18n** — no new copy, but layouts MUST be tested with the longest
  translated strings; the pseudo-locale harness is part of this work.

## 11. AI / ML Considerations

Not AI-touching. Constraint on AI surfaces as consumers: the tutor panel,
ask-AI and grader-agent popovers MUST become sheets below `md` rather than
fixed-width popovers, and streaming responses must not cause horizontal overflow
in code blocks or tables.

## 12. Integration Points

- **External** — none.
- **Internal**
  - `clients/web/src/components/layout/app-shell.tsx`, `side-nav.tsx`,
    `top-bar.tsx`, `resizable-split-pane.tsx`
  - `clients/web/src/components/ui/**` — responsive behaviour lives here
  - `clients/web/src/index.css`, `styles/tokens/*` — breakpoint contract
  - `clients/web/index.html` — viewport meta verification
  - All 99 table-bearing files via
    [UX.11](UX.11-data-table-and-gradebook-system.md)
  - `clients/web/src/components/{media,video-player,markdown,math}/**` — wide
    content containment
  - `e2e/` — multi-viewport visual regression
  - `docs/accessibility/mobile-audit-checklist.md` — extended to cover web
- **Events** — viewport-bucketed CWV into `server/internal/telemetry`.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.1](UX.1-semantic-design-token-system.md) (breakpoint
  tokens), [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md) (components
  own responsiveness), [UX.3](UX.3-typography-and-reading-system.md) (`clamp()`
  scale), [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md) (target size),
  [UX.11](UX.11-data-table-and-gradebook-system.md) (tables are the single largest
  source of overflow).
- **Runs in parallel with** — [UX.9](UX.9-role-aware-dashboard.md),
  [UX.10](UX.10-course-home-and-learning-flow.md) — both should be built responsive
  from the start rather than retrofitted.
- **Shared infra** — device lab or cloud device testing; Chromebook and tablet
  participants for AC-13.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Retrofitting 538 non-responsive files is a very large effort | **H** | **H** | Most responsiveness is delivered *by* UX.2 components and UX.11 tables. This plan owns the shell, the audit, the gates, and the long tail — sequence it **after** those so the tail is small |
| Container queries have uneven support in the org's supported browser matrix | L | M | Verify the matrix; fall back to viewport queries where needed; the contract permits both |
| Dense instructor surfaces become unusable when reflowed to cards | M | **H** | For the gradebook specifically, offer purpose-built single-student / single-assignment views rather than a card list (UX.11 §14) |
| Visual regression at 4 viewports × 40 routes is slow and flaky | M | M | Run the full matrix nightly and on release branches; run 390 + 1440 on every PR |
| Fixing small viewports regresses desktop | M | M | 1440 px is in the regression matrix; desktop screenshots gate every PR |
| Long translations break layouts only in production locales | M | M | Pseudo-locale (`de`-length) in the regression matrix (AC-12) |

## 15. Rollout Plan

- **Feature flag** — none. Responsive fixes are strict improvements and flagging
  two layouts doubles the test matrix.
- **Sequencing**
  1. Breakpoint contract + lint against arbitrary breakpoints + viewport meta
     verification.
  2. **Audit**: automated overflow and target-size sweep across all 200 routes to
     produce the prioritised defect list. *Deliverable: a ranked backlog.*
  3. App shell responsiveness (sidebar, top bar, sheets) — fixes the chrome for
     every route at once.
  4. Container-query adoption in UX.2 components.
  5. Table responsiveness via UX.11.
  6. Route-by-route defect burn-down, traffic-ordered.
  7. Responsive coverage gate flipped on; visual regression matrix enabled.
- **Dogfood** — internal org, with a "Chromebook week" where the team works
  primarily at 1366×768.
- **GA criteria** — AC-1…AC-13 green; coverage ≥95%; zero page-level horizontal
  overflow across the route inventory.
- **Rollback** — per-PR revert; low risk since changes are additive CSS.

## 16. Test Plan

- **Unit** — breakpoint token resolution; container-query fallbacks; sheet vs modal
  selection logic; truncation-with-full-value-exposed behaviour.
- **Integration** — shell drawer behaviour across breakpoints; sheet focus
  management; sticky-bar offset interaction with UX.5 focus-not-obscured.
- **End-to-end** — Playwright at 320 / 390 / 768 / 1024 / 1440 px across the top 40
  routes asserting no page-level horizontal scroll (AC-2) and completing the 8
  critical journeys at 390 px.
- **Security** — verify `inert` behind bottom sheets so background content is not
  exposed to AT.
- **Accessibility** — reflow at 400% zoom (AC-5); orientation lock check (AC-9);
  target-size measurement under `pointer: coarse` (AC-4); axe at each viewport;
  screen-reader pass on the mobile drawer and bottom sheets; RTL at 390 px.
- **Performance / load** — Lighthouse CI with a mid-tier Chromebook profile on the
  top 10 routes (AC-11); image weight budget; long-task profiling on the module
  list at 390 px.
- **Visual regression** — 40 routes × 4 viewports × light/dark, plus RTL and
  pseudo-locale runs (AC-12).
- **Manual exploratory** — real-device session matrix: Chromebook, iPad (Safari),
  Android tablet (Chrome), iPhone (Safari), split-screen Windows.
- **User research** — 6+ moderated sessions on Chromebooks and tablets (AC-13).

## 17. Documentation & Training

- **End-user** — none.
- **Admin / instructor** — a supported-device note in the help centre, naming the
  Chromebook/tablet profile explicitly.
- **Engineer** — `docs/guides/responsive.md`: the breakpoint contract and each
  breakpoint's purpose, when to use container vs viewport queries, the
  no-page-horizontal-scroll rule, the truncation rule, how the coverage gate works.
- **API reference** — n/a.
- **Runbook** — "Responsive coverage check failed on my PR".
- **Update** `docs/accessibility/mobile-audit-checklist.md` to cover the web client
  alongside native.

## 18. Open Questions

1. What is the actual device mix in production? Telemetry should answer this before
   we finalise which breakpoint is "primary". *Current assumption: Chromebook
   1366×768 is the K-12 primary — verify.*
2. Do we support 320 px as a genuine product target, or only as a WCAG Reflow
   conformance floor? *Recommendation: conformance floor; design target is 390 px.*
3. Which surfaces are legitimately desktop-only (live-quiz presenter view,
   whiteboard authoring, screen-share console)? These need an explicit, friendly
   "best on a larger screen" message rather than a broken layout — and that list
   must be short and justified.
4. Is container-query support sufficient across the supported browser matrix, or do
   we need a polyfill?
5. Should the gradebook get a purpose-built tablet interface rather than a
   responsive one? Instructors on iPads are a real segment.

## 19. References

- Existing files: `clients/web/src/components/layout/app-shell.tsx`,
  `side-nav.tsx`, `top-bar.tsx`, `resizable-split-pane.tsx`,
  `clients/web/index.html` (viewport meta), `clients/web/src/index.css`,
  `docs/accessibility/mobile-audit-checklist.md`, `e2e/`
- Research: [research.md](research.md) R-2, R-31, R-35
- Audit: [audit.md](audit.md) G-11, G-5b, G-7
- External: [WCAG 2.2 SC 1.4.10 Reflow](https://www.w3.org/TR/WCAG22/#reflow),
  [SC 1.3.4 Orientation](https://www.w3.org/TR/WCAG22/#orientation),
  [SC 2.5.8 Target Size](https://www.w3.org/TR/WCAG22/#target-size-minimum)
- Related plans: [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md),
  [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md),
  [UX.11](UX.11-data-table-and-gradebook-system.md),
  [UX.17](UX.17-perceived-performance-and-web-vitals-budget.md),
  `../mobile/`, `../../completed/mobile/`, `../../completed/07-mobile-offline-cross-platform/`
