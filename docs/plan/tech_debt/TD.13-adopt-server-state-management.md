# TD.13 — Adopt Server-State Management

> Implementation plan. Source: technical-debt static analysis, 2026-07-25. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.13 |
| **Section** | Technical Debt Remediation |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | TD.11, TD.12 |
| **Unblocks** | TD.14 |

---

## 1. Problem Statement

The web client has **no data-fetching or state-management library** — no react-query, zustand, or redux. Every screen therefore hand-rolls the same machinery in `useState` + `useEffect`: request, loading flag, error flag, refetch, and cache invalidation. The result is visible in the numbers: `course-modules.tsx` holds **99 `useState` declarations across 3,372 lines**, `course-module-quiz-page.tsx` **99 across 3,383**, and `course-module-assignment-page.tsx` **75 across 1,041**. Only **16 shared hooks** exist for **718 components and pages**, so almost none of this is reused. Beyond volume, the pattern causes real defects: no request deduplication, no cancellation on unmount (a classic race where a slow response overwrites newer state), no shared cache so navigating between screens refetches everything, and stale data after mutations unless each call site remembers to refetch. This is the direct cause of the god components in [TD.14](TD.14-decompose-god-components.md) — those files are large mostly because they are 99 copies of the same four lines.

## 2. Goals

- Adopt a server-state library so fetching, caching, deduplication, invalidation, and cancellation are handled once.
- Eliminate the hand-rolled loading/error/refetch triads that dominate large components.
- Fix the class of latent bugs the current pattern permits — unmount races, duplicate in-flight requests, stale-after-mutation reads.
- Make [TD.14](TD.14-decompose-god-components.md) tractable by removing the bulk of what those components contain.
- Adopt incrementally, with the old pattern coexisting indefinitely.

## 3. Non-Goals

- Introducing a *client*-state library (zustand/redux) for UI state — this story is about **server** state only. UI state stays in `useState`.
- Rewriting all 718 components — adoption is incremental and prioritised.
- Changing any API endpoint or payload.
- Changing routing or the lazy-route setup.
- Server-side rendering.

## 4. Personas & User Stories

- **As a student**, I want the page to show current data after I submit, so that I am not confused by stale content.
- **As an instructor navigating between course screens**, I want instant back-navigation from cache, so that the app feels responsive.
- **As a user on a slow connection**, I want in-flight requests deduplicated and cancelled on navigation, so that the app does not thrash.
- **As a web engineer**, I want to write `useQuery(key, fn)` instead of a `useState`/`useEffect` triad, so that a screen is domain logic rather than plumbing.
- **As a reviewer**, I want data fetching to look the same everywhere, so that I can spot the one place that is different.

## 5. Functional Requirements

- **FR-1.** The team MUST select a server-state library and record the decision in an ADR (`docs/adr/`), with TanStack Query as the presumptive default given React 19 and the existing stack.
- **FR-2.** The library MUST be wired to TD.11's shared HTTP helper, so auth refresh, error normalisation, and cancellation flow through one path.
- **FR-3.** A **query-key convention** MUST be defined and documented, organised along TD.12's module boundaries so keys are predictable and invalidation is reliable.
- **FR-4.** Per-domain query hooks MUST live beside their API module (e.g. `src/lib/courses/queries.ts`), not in components.
- **FR-5.** Mutations MUST declare their invalidations explicitly, so post-mutation staleness is a design decision rather than an omission.
- **FR-6.** Adoption MUST be incremental — the existing `useState`/`useEffect` pattern MUST keep working during migration.
- **FR-7.** Migration MUST be prioritised by pain: the highest-`useState` components first (`course-modules.tsx`, `course-module-quiz-page.tsx`, `course-module-assignment-page.tsx`).
- **FR-8.** Each migrated screen MUST preserve its observable behaviour — same data, same loading and error presentation, same empty states.
- **FR-9.** Caching defaults (stale time, refetch-on-focus, retry) MUST be chosen deliberately and documented; defaults that change perceived freshness for gradebooks or live quizzes MUST be considered per domain, not set globally and forgotten.
- **FR-10.** Real-time surfaces (WebSocket-driven feed, boards, live quiz, Canvas import progress) MUST integrate with the cache rather than maintaining parallel state; the integration pattern MUST be documented.
- **FR-11.** Offline behaviour (`src/sw.ts`, `src/db/`, Dexie) MUST continue to work; the interaction between the library's cache and the existing offline store MUST be specified before migrating any offline-capable screen.
- **FR-12.** Bundle-size impact MUST be measured and stay within the existing `bundle:check` budget.
- **FR-13.** A lint rule SHOULD flag new `useEffect`-based fetching in components once adoption is established.

## 6. Non-Functional Requirements

- **Performance** — deduplication and caching should reduce request volume; measure requests-per-session on key flows before and after. Library bundle cost must stay within budget (FR-12).
- **Security** — cached responses hold learner data in memory. Cache MUST be cleared on logout and on user switch (impersonation is a supported feature — `ImpersonationBanner` exists, so cache isolation between impersonated identities is a hard requirement, not a nicety).
- **Privacy & Compliance** — FERPA: cached learner data must not persist beyond the session or leak across users. If the library's persistence plugins are used, they must not write learner data to durable storage without explicit review.
- **Accessibility** — loading and error states must remain announced to assistive technology. Migrating a screen must not turn an announced error into a silent one; WCAG 2.1 AA conformance is preserved per screen.
- **Scalability** — n/a server-side.
- **Reliability** — the story *fixes* reliability issues (races, staleness). Regressions would be severe, so per-screen verification is mandatory.
- **Observability** — the library's devtools should be available in development only, never shipped to production.
- **Maintainability** — the goal.
- **Internationalization** — error and loading copy continues through i18n.
- **Backward compatibility** — no API change; per-screen behaviour preserved (FR-8).

## 7. Acceptance Criteria

- **AC-1.** *Given* the ADR, *When* reviewed, *Then* the library choice, caching defaults, and query-key convention are documented and approved.
- **AC-2.** *Given* a migrated screen, *When* compared to its pre-migration behaviour, *Then* data, loading, error, and empty states are equivalent.
- **AC-3.** *Given* two components requesting the same data simultaneously, *When* they mount, *Then* exactly one network request is made.
- **AC-4.** *Given* a component unmounts while a request is in flight, *When* the response arrives, *Then* no state update occurs and no console warning is produced.
- **AC-5.** *Given* a mutation succeeds, *When* it completes, *Then* the declared queries are invalidated and the UI reflects current data without a manual refetch.
- **AC-6.** *Given* a user logs out, *When* the cache is inspected, *Then* it is empty.
- **AC-7.** *Given* an admin starts and stops impersonating a user, *When* the cache is inspected, *Then* no data from one identity is visible to the other.
- **AC-8.** *Given* a migrated offline-capable screen, *When* the device is offline, *Then* offline behaviour is unchanged from before migration.
- **AC-9.** *Given* a migrated screen with a WebSocket feed, *When* a real-time event arrives, *Then* the cache updates and the UI re-renders without a duplicate fetch.
- **AC-10.** *Given* the library is added, *When* `npm run bundle:check` runs, *Then* it passes.
- **AC-11.** *Given* the three FR-7 priority screens are migrated, *When* their `useState` counts are measured, *Then* each has fallen substantially from 99 / 99 / 75.
- **AC-12.** *Given* a migrated screen, *When* audited with axe and a screen reader, *Then* loading and error states remain announced.

## 8. Data Model

No schema change. New structure:

```
clients/web/src/lib/query-client.ts       # configured client, cache lifecycle, logout/impersonation reset
clients/web/src/lib/courses/queries.ts    # per-domain hooks, co-located with the API module (TD.12)
docs/adr/NNNN-server-state-management.md  # FR-1 decision record
```

## 9. API Surface

**No server API change.** Request volume and timing change (fewer duplicates, cancellation on unmount) — worth noting for anyone reading server metrics during rollout, since a drop in request rate is expected and is not an outage.

## 10. UI / UX

No intended visual change. Per-screen requirements during migration:

1. **Loading** — same skeleton/spinner as before, same placement.
2. **Error** — same copy, same recovery affordance, still announced to assistive tech.
3. **Empty** — unchanged.
4. **Offline** — unchanged (FR-11, AC-8).
5. **Refetch-on-focus** — this is a *new* behaviour if enabled by default and is user-visible. Decide per domain (FR-9); enabling it silently on a gradebook could surprise instructors mid-edit.
6. **Mobile/responsive** — unchanged.

Accessibility: focus must not be stolen when cached data renders instantly instead of after a spinner.

## 11. AI / ML Considerations

AI-backed screens (grading agent workflow, adaptive content, tutor session) often involve long-running or streaming operations. These MUST NOT be forced into a simple query model; the ADR should state how long-running AI operations are represented — polling with a query, streaming outside the cache, or a mutation with progress state. `use-grader-agent-workflow.ts` (1,251 lines) is the reference case and should be examined before the pattern is fixed.

## 12. Integration Points

- `clients/web/src/lib/api.ts`, `http.ts` (TD.11) — the fetch layer the library wraps.
- `clients/web/src/lib/courses/` (TD.12) — where query hooks live.
- `clients/web/src/auth/`, `ImpersonationBanner` — cache lifecycle on identity change (AC-6, AC-7).
- `clients/web/src/sw.ts`, `src/db/` (Dexie) — offline interaction (FR-11).
- WebSocket hooks: `use-course-structure-ws.ts`, feed/boards/live-quiz/Canvas-import subscriptions (FR-10).
- `clients/web/scripts/check-bundle-size.mjs`.
- `docs/adr/`.

## 13. Dependencies & Sequencing

- Must ship after: **TD.11** (one fetch seam to wrap), **TD.12** (module boundaries define query keys).
- Must ship before: **TD.14** — decomposing god components before removing the state machinery would mean decomposing the machinery too, then deleting it.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Cached learner data leaks across identities during impersonation | M | **H** | AC-7 explicit test; cache reset wired to identity change, not just logout; security review of `query-client.ts` |
| Caching defaults change perceived freshness (stale gradebook, stale submission status) | **H** | H | FR-9 per-domain defaults, decided deliberately; AC-2 per-screen behaviour comparison |
| Refetch-on-focus surprises users mid-edit | M | M | Default off; enable per domain with explicit reasoning (§10.5) |
| Offline behaviour breaks on migrated screens | M | H | FR-11 requires the interaction be specified before migrating offline screens; AC-8 gate |
| Real-time screens end up with two sources of truth | **H** | M | FR-10 documented integration pattern before migrating any WebSocket screen |
| Accessibility regression — errors no longer announced | M | H | AC-12 axe + screen-reader check per migrated screen |
| Adoption stalls, leaving two patterns forever | M | M | FR-7 prioritisation delivers visible wins early; FR-13 lint rule once established; track migrated-screen count |
| Bundle size grows beyond budget | L | M | AC-10 gate; the library is small relative to current hand-rolled code being deleted |

## 15. Rollout Plan

- **Feature flag** — none at the library level; migration is per screen, so each screen is independently revertible.
- **Sequencing** — (1) ADR and library selection; (2) wire the client with cache-lifecycle handling (logout, impersonation) and prove AC-6/AC-7 before migrating anything; (3) migrate one small, non-offline, non-real-time screen as a pilot; (4) document the WebSocket and offline patterns (FR-10, FR-11); (5) migrate the three priority screens; (6) continue by pain order; (7) add the FR-13 lint rule.
- **Dogfood** — internal users on staging for each priority screen; watch for staleness complaints specifically.
- **GA criteria** — three priority screens migrated, no regression reports for two weeks, bundle check green.
- **Rollback** — per-screen revert; the library can remain installed and unused.

## 16. Test Plan

- **Unit** — query hooks: success, error, loading; mutation invalidation (AC-5); cache reset on logout and identity change (AC-6, AC-7).
- **Integration** — component tests per migrated screen against a mocked server; deduplication (AC-3) and unmount cancellation (AC-4).
- **End-to-end** — `make e2e` green; per-screen Playwright flows for the three priority screens; an explicit stale-after-mutation scenario.
- **Security** — impersonation cache-isolation test (AC-7); confirm no learner data written to durable storage.
- **Accessibility** — axe scan plus screen-reader script per migrated screen (AC-12); verify focus is not stolen on instant cache render.
- **Performance / load** — requests-per-session on key flows before and after; `bundle:check`; Lighthouse on the dashboard via the existing harness.
- **Manual exploratory** — offline toggling, slow-network throttling, rapid navigation between course screens.

Baseline:

```bash
cd clients/web
for f in src/pages/lms/course-modules.tsx src/pages/lms/course-module-quiz-page.tsx src/pages/lms/course-module-assignment-page.tsx; do
  echo "$f useState=$(grep -oE 'useState[<(]' $f | wc -l) lines=$(wc -l < $f)"
done   # 99/3372, 99/3383, 75/1041
ls src/hooks/ | wc -l    # 16 shared hooks for 718 components+pages
```

## 17. Documentation & Training

- ADR in `docs/adr/` — library choice, caching defaults, query-key convention, AI/long-running pattern.
- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — server state via the library; UI state via `useState`; no `useEffect` fetching in new code.
- Migration guide with a worked before/after for one real screen.
- WebSocket-integration and offline-integration patterns documented before those screens migrate.
- Team session on query keys and invalidation — the two things teams most often get wrong.

## 18. Open Questions

1. TanStack Query is the presumptive choice — confirm against React 19 compatibility, bundle cost, and the team's familiarity in the ADR.
2. What are the right stale times per domain? Gradebook, live quiz, and course content have very different freshness expectations.
3. How should the cache interact with the existing Dexie offline store — as the source of truth, a fallback, or kept entirely separate?
4. What is the pattern for long-running AI operations (§11)? Examine `use-grader-agent-workflow.ts` first.
5. Should refetch-on-focus be off globally and opted into, or on with opt-outs? (Leaning off — the safer default for an app with editing surfaces.)
6. Is impersonation identity change already emitting an event the cache can subscribe to, or does one need adding?

## 19. References

- `clients/web/src/pages/lms/course-modules.tsx` — 3,372 LOC, 99 `useState`
- `clients/web/src/pages/lms/course-module-quiz-page.tsx` — 3,383 LOC, 99 `useState`
- `clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts` — 1,251 LOC, the AI long-running reference case
- `clients/web/src/hooks/` — 16 shared hooks
- `clients/web/src/sw.ts`, `src/db/` — offline layer
- TanStack Query — <https://tanstack.com/query/latest>
- Related plans: [TD.11](TD.11-consolidate-http-client-foundation.md), [TD.12](TD.12-split-courses-api-module.md), [TD.14](TD.14-decompose-god-components.md)
