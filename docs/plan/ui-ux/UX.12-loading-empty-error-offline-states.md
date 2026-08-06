# UX.12 — Loading, Empty, Error and Offline States

> Implementation plan. Source: [audit.md](audit.md) §6 G-9.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.12 |
| **Section** | UI/UX — Interaction |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN — three competing loading idioms, 4 error boundaries for 200 routes |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web + Product Design |
| **Depends on** | UX.1, UX.2 |
| **Unblocks** | UX.9, UX.10, UX.11, UX.17 |

---

## 1. Problem Statement

Three loading idioms coexist — 106 files render literal `Loading…` text, 90 use a
spinner, 44 use a hand-rolled `animate-pulse` skeleton — while the *system*
skeleton set is used in **5** files. There are **4 error boundaries for 200
routes**, so a render error in any of ~310 page components propagates to the app
root and white-screens the user. Only **8 files** reference offline state despite a
full Workbox PWA, background sync and Dexie storage: the offline *infrastructure*
exists and the offline *experience* does not. Empty states are ad-hoc strings —
490 "empty" mentions and hundreds of one-off "No X yet" lines — against 15 uses of
the `EmptyState` component. Per **R-16/R-18/R-19**, these three states are the ones
most reliably omitted in fast-moving codebases, and their absence is felt hardest
by exactly our least-confident users.

## 2. Goals

- Make **skeleton-first loading** the single default idiom, matching final layout
  to eliminate shift and reduce perceived wait (**R-16**).
- Treat empty states as **onboarding moments**, not dead ends (**R-18**).
- Contain failures: an error boundary per route and per widget, never a white
  screen.
- Deliver a real **offline experience** on top of the infrastructure already
  shipped.
- Establish a **four-state contract** that every data-bearing surface must satisfy.

## 3. Non-Goals

- Changing data-fetching architecture — that is
  [`TD.13`](../tech_debt/TD.13-adopt-server-state-management.md), with which this
  plan coordinates.
- Building new offline *capabilities* (e.g. offline quiz authoring).
- Performance optimisation itself — that is
  [UX.17](UX.17-perceived-performance-and-web-vitals-budget.md); this plan owns
  *perceived* performance.
- Error *reporting* infrastructure (Sentry is already wired via
  `server/internal/telemetry`).

## 4. Personas & User Stories

- **As a student on a slow connection**, I want to see the shape of the page
  arriving rather than a spinner, so I know it is working.
- **As a new instructor**, I want an empty gradebook to tell me what to do next,
  not just say "No data".
- **As any user**, I want one failing panel not to destroy the whole page.
- **As any user**, I want an error I can act on — retry, go back, or report — not
  a blank screen.
- **As a student on a train**, I want previously-opened content to still be
  readable, and my work queued until I reconnect.
- **As a support engineer**, I want an error to carry a reference I can trace.

## 5. Functional Requirements

### The four-state contract

- **FR-1.** Every data-bearing surface MUST implement **loading**, **empty**,
  **error** and **offline** states. A CI check MUST flag route-level components
  that render data without all four.
- **FR-2.** States MUST be provided by shared components (`Skeleton`,
  `EmptyState`, `ErrorState`, `OfflineNotice`) from UX.2. Hand-rolled variants MUST
  be forbidden by lint with a ratcheting allowlist.

### Loading

- **FR-3.** **Skeletons are the default.** Each skeleton MUST match the final
  layout's dimensions and element count to prevent shift.
- **FR-4.** Spinners are permitted only for **in-place, indeterminate, short**
  operations (a button's own loading state, an inline save).
- **FR-5.** Literal `Loading…` text MUST be eliminated as a primary loading
  indicator.
- **FR-6.** For operations expected to exceed ~5 s, honest progress MUST be shown
  instead of an indefinite skeleton (**R-17**) — a determinate bar or a staged
  description ("Importing 3 of 12 modules").
- **FR-7.** Loading regions MUST set `aria-busy` and announce completion politely.
- **FR-8.** A minimum display duration MUST prevent skeleton flash on fast
  responses (suggested: don't show below 150 ms; once shown, hold ≥400 ms).

### Empty

- **FR-9.** Every empty state MUST answer three questions: **what belongs here**,
  **why it is empty**, and **the one next action** (**R-18**).
- **FR-10.** "Empty because there is no data yet" MUST be visually and textually
  distinct from "empty because your filter matched nothing" — the latter must
  offer "clear filters".
- **FR-11.** Empty states MUST be **role-aware**: an instructor sees "Create your
  first assignment"; a student sees "Your instructor hasn't posted work yet".
- **FR-12.** Empty-state copy MUST come from i18n keys and MUST NOT be phrased as
  an error.

### Error

- **FR-13.** An error boundary MUST wrap **every route** and every dashboard/course
  widget. A boundary failure MUST NOT propagate past its own region.
- **FR-14.** Error states MUST offer: a plain-language description, a **retry**,
  a way back, and a **support reference id** correlating to the Sentry/telemetry
  event.
- **FR-15.** Error copy MUST be classified and specific: network, permission, not
  found, conflict, server, client. "Something went wrong" is permitted only as the
  last-resort fallback.
- **FR-16.** Permission errors MUST explain what access is needed and who to ask —
  never a bare 403.
- **FR-17.** Errors MUST be announced via a live region and MUST move focus to the
  error region when they replace the user's current context.
- **FR-18.** Errors MUST NOT discard user input. A failed save MUST preserve the
  form ([UX.6](UX.6-form-and-validation-system.md) FR-11).

### Offline

- **FR-19.** The app MUST detect and display connection state, distinguishing
  **offline** from **server unreachable**.
- **FR-20.** Previously-visited content MUST remain readable offline via the
  existing Workbox caches, with a visible "offline — last updated {time}"
  indicator.
- **FR-21.** Actions taken offline MUST be **queued** using the existing
  background-sync infrastructure, with a visible pending-count, and MUST replay on
  reconnect.
- **FR-22.** Actions that cannot work offline MUST be clearly disabled with an
  explanation — never fail silently.
- **FR-23.** On reconnect, the app MUST reconcile and inform the user what synced
  and what failed.
- **FR-24.** Stale cached data MUST be labelled with its age wherever accuracy
  matters (grades, due dates, submissions).

## 6. Non-Functional Requirements

- **Performance** — Skeletons MUST add ≤3 KB gzip total and MUST NOT delay actual
  content. CLS from loading→content transitions MUST be ≤0.02 (skeleton dimensions
  must match). Error boundaries MUST add no measurable render cost.
- **Security** — Error messages MUST NOT leak stack traces, SQL, internal
  hostnames, or the existence of records the user cannot access. The support
  reference id MUST be opaque and non-guessable. Offline caches MUST be cleared on
  sign-out and on unenrollment.
- **Privacy & Compliance** — Cached education records on-device are FERPA-relevant;
  cache scope, retention and clearing MUST be documented in the RoPA
  (`../standards/S05-ropa-data-inventory-mapping.md`) and honour
  `../standards/S02-data-retention-deletion-engine.md`.
- **Accessibility** — Skeletons MUST be `aria-hidden` with an accompanying
  `aria-busy` region so AT is not read a wall of placeholders; error regions are
  `role="alert"`; offline indicator is text + icon, never colour alone; all state
  changes announced politely except errors that take over context.
- **Scalability** — The contract is enforced by lint, so new surfaces inherit it.
- **Reliability** — Retry MUST use exponential backoff with jitter and a cap. The
  offline queue MUST be durable across reloads and MUST deduplicate.
- **Observability** — Emit `route_error_boundary_triggered` (with component and
  reference id), `state_empty_shown`, `state_error_shown`, `retry_clicked`,
  `retry_succeeded`, `offline_entered`, `offline_queue_depth`, `sync_failed`.
- **Maintainability** — One skeleton per surface, colocated with it, so they drift
  together.
- **Internationalization** — All state copy from i18n keys at parity across four
  locales; relative times ("last updated 5 minutes ago") locale-formatted; RTL
  layouts.
- **Backward compatibility** — Purely additive. Existing successful paths are
  unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* the codebase, *When* the state-contract check runs, *Then*
  every route-level data component declares all four states and the allowlist is
  empty.
- **AC-2.** *Given* any route loading, *When* observed, *Then* a skeleton matching
  the final layout is shown and CLS on the loading→content transition is ≤0.02.
- **AC-3.** *Given* a response arriving in under 150 ms, *When* observed, *Then* no
  skeleton flash occurs.
- **AC-4.** *Given* a component throws during render, *When* it does, *Then* only
  that region shows an error state and the rest of the page remains interactive.
- **AC-5.** *Given* an error state, *When* shown, *Then* it includes a
  plain-language cause, a working retry, a way back, and a support reference id
  that appears in telemetry.
- **AC-6.** *Given* a 403, *When* shown, *Then* the message explains the missing
  access and who to contact, and reveals nothing about the resource.
- **AC-7.** *Given* an empty list with no data, *When* shown, *Then* it states
  what belongs there and offers exactly one primary action, role-appropriately.
- **AC-8.** *Given* a filter matching nothing, *When* shown, *Then* the state is
  distinct from the no-data state and offers "clear filters".
- **AC-9.** *Given* the user goes offline, *When* they open previously-visited
  content, *Then* it renders from cache with an offline indicator and last-updated
  time.
- **AC-10.** *Given* the user acts offline, *When* they do, *Then* the action is
  queued with a visible pending count and replays on reconnect.
- **AC-11.** *Given* a queued action fails permanently on replay, *When* it does,
  *Then* the user is told which action failed and why, and can retry or discard.
- **AC-12.** *Given* sign-out, *When* it completes, *Then* offline caches and the
  action queue are cleared.
- **AC-13.** *Given* any state, *When* axe runs, *Then* 0 violations; skeletons are
  not read as content by a screen reader.
- **AC-14.** *Given* an error, *When* it is shown, *Then* no stack trace, SQL or
  internal hostname appears in the DOM.

## 8. Data Model

None server-side. The offline action queue lives in the existing IndexedDB store
(`clients/web/src/db/`, Dexie) with this shape:

```ts
type QueuedAction = {
  id: string                  // uuid, also the idempotency key
  createdAt: number
  method: 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  url: string
  body: unknown
  attempts: number
  lastError: string | null
  status: 'pending' | 'syncing' | 'failed'
}
```

No tables, columns, enums, indexes, migrations or backfill are required in
PostgreSQL.

## 9. API Surface

No new routes. Two cross-cutting requirements on existing endpoints:

- **Idempotency** — every mutating endpoint that the offline queue may replay MUST
  accept an `Idempotency-Key` header and MUST be safe to retry. This is a
  prerequisite for FR-21 and MUST be audited endpoint by endpoint.
- **Error envelope** — errors MUST return a stable machine `code` plus an opaque
  `referenceId` so FR-14/FR-15 can classify and correlate:

```ts
type ApiError = {
  error: string          // stable machine code, e.g. 'course_not_found'
  message: string        // safe, human-readable
  referenceId: string    // opaque; correlates to telemetry
}
```

- No WebSocket changes. Retry backoff MUST respect existing rate limits and honour
  `Retry-After`.
- **OpenAPI** — the error envelope MUST be a shared component schema;
  `make openapi-check` passes.

## 10. UI / UX

- **New pages** — none.
- **Modified pages** — every data-bearing surface; ~310 page components plus
  dashboard and course widgets.
- **Key user flows**
  1. Slow load → skeleton in the final shape → content fills in with no jump.
  2. Panel throws → that panel shows an error with retry → the rest of the page
     still works → retry succeeds.
  3. New instructor opens an empty gradebook → "No assignments yet. Create your
     first assignment." → one click.
  4. Student goes offline → banner appears → previously-read module still opens
     with "last updated 12 minutes ago" → they submit a note → "1 change waiting to
     sync" → reconnect → "1 change synced".
- **States** — this plan *is* the states. Each shared component documents its
  variants in the UX.2 gallery.
- **Mobile/responsive** — offline indicator is a persistent, unobtrusive bar; error
  states are full-width and reachable without horizontal scroll.
- **Accessibility annotations** — skeleton containers `aria-hidden="true"` inside an
  `aria-busy="true"` region with an off-screen "Loading {thing}" label; error
  regions `role="alert"` and focusable; offline status in a polite live region;
  empty states use `role="status"` (as `EmptyState` already does).
- **Copy & i18n** — a shared state-copy catalogue under `common.state.*`
  (`loading`, `emptyNoData`, `emptyFiltered`, `errorNetwork`, `errorPermission`,
  `errorNotFound`, `errorServer`, `offline`, `syncPending`, `syncFailed`) at
  parity across four locales. **Copy rules: name the thing, explain the cause,
  give one action, never blame the user.**

## 11. AI / ML Considerations

Not AI-touching. Two constraints on AI surfaces as *consumers*: streaming AI
responses MUST show a streaming state distinct from a loading skeleton, and MUST
be announced politely rather than by moving focus; AI features that require network
MUST be explicitly disabled offline with an explanation (FR-22), never left to
fail.

## 12. Integration Points

- **External** — Workbox (`workbox-*`, already a dependency), Dexie (already a
  dependency), Sentry via `server/internal/telemetry`.
- **Internal**
  - `clients/web/src/components/ui/lms-content-skeletons.tsx` — extended
  - `clients/web/src/components/ui/empty-state.tsx` — universally adopted
  - New: `components/ui/error-state.tsx`, `components/ui/offline-notice.tsx`,
    `components/ui/route-error-boundary.tsx`
  - `clients/web/src/sw.ts`, `vite.config.ts` (PWA config) — cache scoping
  - `clients/web/src/db/` — action queue
  - `clients/web/src/lib/api.ts`, `lib/errors.ts` — error envelope, backoff,
    idempotency keys
  - `clients/web/src/app.tsx` — per-route boundaries
  - `clients/web/src/components/a11y/live-region.tsx`
  - `server/internal/httpserver` — error envelope, idempotency
- **Events** — state telemetry into `server/internal/telemetry`.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.1](UX.1-semantic-design-token-system.md),
  [UX.2](UX.2-core-component-library-and-adoption-ratchet.md).
- **Should coordinate with** — [`TD.13`](../tech_debt/TD.13-adopt-server-state-management.md);
  if a server-state library is adopted, loading/error/retry semantics come largely
  for free and this plan should consume rather than duplicate them.
- **Must ship before** — [UX.9](UX.9-role-aware-dashboard.md) (per-widget error
  isolation is a hard requirement there), [UX.10](UX.10-course-home-and-learning-flow.md),
  [UX.11](UX.11-data-table-and-gradebook-system.md).
- **Shared infra** — telemetry pipeline; PWA cache configuration.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Offline queue replays a stale mutation and overwrites newer server state | M | **H** | Idempotency keys + server-side conflict detection; queued actions carry the version they were based on; conflicts surface to the user rather than auto-resolving |
| Cached education records persist on a shared device after sign-out | M | **H** | FR-19/AC-12: explicit cache and queue clearing on sign-out and unenrollment; covered in the security review and RoPA |
| Skeletons that don't match final layout make CLS worse, not better | **H** | M | Skeletons colocated with their surface so they drift together; AC-2 CLS gate; visual regression on loading state |
| Adding boundaries everywhere hides errors from telemetry | M | **H** | Every boundary reports to Sentry with the reference id before rendering; a caught error is still an error in the dashboards |
| "Retry" retries something that cannot succeed, frustrating users | M | M | Classify errors (FR-15); only offer retry for retryable classes; permission and not-found offer navigation instead |
| ~310 surfaces is a large migration | **H** | M | Lint ratchet; migrate by directory; highest-traffic first; the shared components make each site small |
| Idempotency audit across all mutating endpoints is substantial backend work | **H** | M | Scope the offline queue to a **defined allowlist** of actions in v1 (notes, annotations, quiz answers, discussion posts) rather than all mutations |

## 15. Rollout Plan

- **Feature flag** — `ffOfflineQueue` gates the offline action queue (the risky
  part). Skeletons, empty states and error boundaries ship unflagged — they are
  strict improvements.
- **Sequencing**
  1. Shared components (`Skeleton` set, `ErrorState`, `OfflineNotice`,
     `RouteErrorBoundary`) + gallery + copy catalogue.
  2. Per-route error boundaries across all 200 routes (highest value per effort —
     ends white screens immediately).
  3. Error envelope + classification + reference ids server-side.
  4. Skeleton migration by directory; `Loading…` text eliminated.
  5. Empty-state migration by directory.
  6. Offline read experience (cached content + indicators).
  7. Offline action queue behind `ffOfflineQueue`, scoped to the v1 allowlist.
  8. Lint flipped to error; allowlists deleted.
- **Dogfood** — internal org; deliberate offline testing sessions.
- **GA criteria** — AC-1…AC-14 green; `route_error_boundary_triggered` visible in
  telemetry and trending down; zero data-loss reports from the offline queue over
  30 days.
- **Rollback** — `ffOfflineQueue` off disables queueing (offline becomes
  read-only). Other work is additive and low-risk.

## 16. Test Plan

- **Unit** — skeleton min-display timing; error classification from status codes and
  envelope; backoff with jitter and cap; `Retry-After` honoured; queue enqueue,
  dedupe, replay ordering, permanent-failure handling; cache-clear on sign-out.
- **Integration** — error envelope end-to-end for each class (401/403/404/409/
  422/429/500); idempotent replay produces one effect; conflict detection on stale
  replay.
- **End-to-end** — Playwright: forced component throw → isolated boundary → retry;
  offline read of a previously-visited module; offline action → reconnect → sync
  confirmation; permanently-failing queued action → user-facing resolution;
  sign-out clears caches.
- **Security** — assert no stack trace, SQL or internal hostname in any error DOM
  (AC-14); reference id is opaque and non-enumerable; cached data inaccessible
  after sign-out; queue cannot be used to replay another user's action.
- **Accessibility** — axe on every state variant × 4 themes (AC-13);
  screen-reader scripts: enter a loading region (hear "Loading assignments", not
  placeholder soup); receive an error (hear the alert, land on it); go offline
  (hear the polite status).
- **Performance / load** — CLS on loading→content for the top 20 routes (AC-2);
  skeleton flash check under fast responses (AC-3); bundle delta gate.
- **Manual exploratory** — QA checklist: throttled 3G, airplane mode, server 500s,
  expired session mid-action, tab restored after long sleep.

## 17. Documentation & Training

- **End-user** — help-centre: "Using Lextures offline" — what works, what is
  queued, what is not available.
- **Admin / instructor** — note on offline caching for institutions with
  device-sharing policies.
- **Engineer** — `docs/guides/ui-states.md`: the four-state contract, when a
  spinner is allowed, how to write a skeleton that matches, the error-copy rules,
  how to add an action to the offline allowlist.
- **API reference** — the `ApiError` envelope and `Idempotency-Key` convention in
  OpenAPI and the API guide.
- **Runbook** — "A user reports a white screen" (now: find the boundary event by
  reference id); "Offline queue stuck": inspecting and clearing a user's queue.
- **Compliance** — document on-device caching in the RoPA and the privacy notice.

## 18. Open Questions

1. Does [`TD.13`](../tech_debt/TD.13-adopt-server-state-management.md) land first?
   If a server-state library is adopted, FR-3/FR-7/FR-14 retry semantics should be
   delegated to it rather than reimplemented. **This ordering decision should be
   made before implementation.**
2. What is the v1 offline action allowlist? *Proposed: notes, annotations,
   reading-log entries, discussion drafts, quiz answers. Explicitly excluded:
   grading, enrollment, settings, anything with authorisation nuance.*
3. How long do offline caches persist, and does an institution need a policy
   control to disable on-device caching entirely (shared-device K-12 labs)?
4. Should the support reference id be user-visible (helps support) or only in
   telemetry (avoids alarming users)? *Recommendation: visible but de-emphasised,
   with copy-to-clipboard.*
5. Do we need a distinct "server unreachable but network up" state, or is one
   offline state sufficient? *Recommendation: distinguish them — the user actions
   differ.*

## 19. References

- Existing files: `clients/web/src/components/ui/lms-content-skeletons.tsx`,
  `components/ui/empty-state.tsx`, `components/ui/load-reveal.tsx`,
  `clients/web/src/sw.ts`, `clients/web/vite.config.ts` (PWA/Workbox config),
  `clients/web/src/db/`, `clients/web/src/lib/api.ts`, `lib/errors.ts`,
  `clients/web/src/app.tsx`, `clients/web/src/components/a11y/live-region.tsx`
- Research: [research.md](research.md) R-16, R-17, R-18, R-19, R-23
- Audit: [audit.md](audit.md) G-9, G-13, G-2
- External: [NN/g — Skeleton Screens 101](https://www.nngroup.com/articles/skeleton-screens/),
  [LogRocket — Skeleton loading screen design](https://blog.logrocket.com/ux-design/skeleton-loading-screen-design/)
- Related plans: [UX.2](UX.2-core-component-library-and-adoption-ratchet.md),
  [UX.9](UX.9-role-aware-dashboard.md),
  [UX.13](UX.13-feedback-undo-and-destructive-actions.md),
  [UX.17](UX.17-perceived-performance-and-web-vitals-budget.md),
  [`../tech_debt/TD.13-adopt-server-state-management.md`](../tech_debt/TD.13-adopt-server-state-management.md),
  `../../completed/07-mobile-offline-cross-platform/`
