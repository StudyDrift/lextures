# TD.11 — Consolidate the Web HTTP Client Foundation

> Implementation plan. Source: technical-debt static analysis, 2026-07-25. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.11 |
| **Section** | Technical Debt Remediation |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (internal) |
| **Status (today)** | PARTIAL |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | TD.2 |
| **Unblocks** | TD.12, TD.13 |

---

## 1. Problem Statement

`clients/web/src/lib` contains **113 API client modules** built on a good foundation — `api.ts` provides `authorizedFetch` with session-refresh handling, and `errors.ts` provides `readApiErrorMessage`. But the layer above that foundation was never shared: **six modules each define a byte-identical private `apiJson<T>` helper** (`av-scan`, `captions`, `demographics`, `email-templates`, `scheduler`, `system-email-templates`), and the remaining modules hand-roll the same ok-check / 204-handling / `res.json()` sequence inline. Separately, **25 component and page files call `fetch()` directly**, bypassing `authorizedFetch` entirely and therefore its session-refresh logic — a correctness gap, not just a style one. There is no single place to add retry, request cancellation, timeout, or error normalisation, which is exactly what [TD.13](TD.13-adopt-server-state-management.md) will need.

## 2. Goals

- Establish **one** shared typed request helper and delete the six duplicates.
- Bring the 25 direct `fetch()` call sites onto the shared foundation so session refresh applies uniformly.
- Create the single seam where timeout, cancellation, retry, and error normalisation can later be added once.
- Add enforcement so a seventh duplicate cannot appear.
- Preserve every request's observable behaviour exactly.

## 3. Non-Goals

- Splitting `courses-api.ts` — that is [TD.12](TD.12-split-courses-api-module.md).
- Introducing a data-fetching/caching library — that is [TD.13](TD.13-adopt-server-state-management.md).
- Changing endpoint URLs, payloads, or error-message copy.
- Rewriting the 113 modules' function signatures.
- Adopting generated OpenAPI types (unblocked by [TD.3](../../completed/tech_debt/TD.3-repair-and-verify-openapi-contract.md)).

## 4. Personas & User Stories

- **As a web engineer**, I want one import for typed API calls, so that I do not copy a helper from a neighbouring file.
- **As a user whose session expires mid-task**, I want every request to refresh transparently, so that I do not lose work on the 25 endpoints that currently bypass refresh.
- **As a web engineer adding retry or timeout**, I want one place to add it, so that it applies everywhere.
- **As a reviewer**, I want a lint rule to catch raw `fetch()` in a component, so that the foundation cannot be bypassed.

## 5. Functional Requirements

- **FR-1.** `src/lib/api.ts` (or a sibling `src/lib/http.ts`) MUST export a single generic request helper handling: `authorizedFetch` delegation, non-OK → typed error via `readApiErrorMessage`, `204` → `undefined`, and JSON parsing.
- **FR-2.** The six duplicate `apiJson<T>` implementations MUST be deleted and their modules switched to the shared helper.
- **FR-3.** The shared helper MUST be behaviourally identical to the duplicates it replaces — same error type, same message, same 204 handling.
- **FR-4.** The 25 files calling `fetch()` directly MUST be audited; each MUST either move to the shared helper or carry a comment justifying why raw `fetch` is correct (e.g. streaming, external origin, file upload progress).
- **FR-5.** The helper MUST support `AbortSignal` so callers can cancel, laying the groundwork for TD.13.
- **FR-6.** The helper MUST preserve existing behaviour for non-JSON responses (blobs, text, streams) — either by supporting them explicitly or by leaving those callers on `authorizedFetch`.
- **FR-7.** A lint rule MUST flag raw `fetch(` in `src/components` and `src/pages`, with an allowlist for justified exceptions per FR-4.
- **FR-8.** A test MUST assert the exported surface of each migrated module is unchanged (complements TD.1 FR-10).
- **FR-9.** Migration MUST be incremental, one module or small group per PR.
- **FR-10.** The helper SHOULD normalise errors into a typed shape (status, code, message) rather than a bare `Error`, so callers can branch on status without string matching.

## 6. Non-Functional Requirements

- **Performance** — no additional network round-trips; the helper is a thin wrapper. Bundle size must not grow measurably (`npm run bundle:check` gates this).
- **Security** — the 25 raw `fetch()` sites are the security-relevant part: they may not attach credentials consistently or handle 401 refresh. Audit each for token handling before migrating. The helper MUST NOT log request bodies or tokens.
- **Privacy & Compliance** — error messages surfaced to users must not echo server internals; FR-10's typed errors help keep raw server text out of the UI.
- **Accessibility** — unchanged; error copy reaching users must remain announceable by the existing error-display components.
- **Scalability** — n/a.
- **Reliability** — session refresh applying uniformly (FR-4) is a reliability improvement for the affected 25 sites.
- **Observability** — the single seam SHOULD later carry client-side request telemetry; not implemented here, but the design must not preclude it.
- **Maintainability** — the story's purpose.
- **Internationalization** — error copy must continue to flow through the existing i18n path; the helper must not hard-code English strings.
- **Backward compatibility** — no module's exported signatures change.

## 7. Acceptance Criteria

- **AC-1.** *Given* the migration is complete, *When* `src/lib` is scanned, *Then* exactly one request helper definition exists and the six duplicates are gone.
- **AC-2.** *Given* a migrated module, *When* its exported symbols are compared before and after, *Then* they are identical (AC verified by the FR-8 test).
- **AC-3.** *Given* a request returning 204, *When* it resolves, *Then* the caller receives `undefined`, exactly as before.
- **AC-4.** *Given* a request returning 4xx with an error body, *When* it rejects, *Then* the message equals what `readApiErrorMessage` produced before.
- **AC-5.** *Given* an expired session, *When* any migrated endpoint is called, *Then* the session refreshes and the request retries — including on the 25 formerly-raw-`fetch` sites.
- **AC-6.** *Given* a raw `fetch(` added to a component, *When* CI runs, *Then* the FR-7 lint rule fails unless allowlisted.
- **AC-7.** *Given* a caller passes an `AbortSignal` and aborts, *When* the request is in flight, *Then* it cancels and rejects with an abort error.
- **AC-8.** *Given* the migration is complete, *When* `npm run bundle:check` runs, *Then* it passes with no meaningful size increase.

## 8. Data Model

No schema change. File changes:

```
clients/web/src/lib/http.ts          # the shared helper (new, or folded into api.ts)
clients/web/src/lib/errors.ts        # extended for FR-10 typed errors
clients/web/src/lib/*-api.ts         # six duplicates removed; callers switched
scripts/allowlists/raw-fetch.txt     # FR-4/FR-7 justified exceptions
```

## 9. API Surface

**No server API change.** Client-side module surface is unchanged (AC-2).

## 10. UI / UX

No intended UI change. The one user-visible improvement is that expired sessions on the formerly-raw-`fetch` endpoints now refresh instead of failing.

- **Error states** — messages must render identically (AC-4).
- **Loading states** — unchanged.
- **Offline** — verify the offline/service-worker path (`src/sw.ts`, `src/db/`) is unaffected; some raw `fetch` sites may be intentional offline-cache interactions and belong on the FR-4 allowlist.
- **Accessibility** — error announcement unchanged.

## 11. AI / ML Considerations

Some AI endpoints stream responses. FR-6 explicitly covers non-JSON responses — streaming callers must stay on `authorizedFetch` or get first-class streaming support. Do not force a streaming call site through a JSON helper.

## 12. Integration Points

- `clients/web/src/lib/api.ts` — `authorizedFetch`, `tryRefreshSession`, `apiUrl`, `wsUrl`.
- `clients/web/src/lib/errors.ts` — `readApiErrorMessage`.
- Six duplicate modules: `av-scan-api.ts`, `captions-api.ts`, `demographics-api.ts`, `email-templates-api.ts`, `scheduler-api.ts`, `system-email-templates-api.ts`.
- 25 component/page files with raw `fetch()`.
- `clients/web/src/sw.ts`, `src/db/` — offline paths.
- CI: `oxlint` config, `bundle:check`.

## 13. Dependencies & Sequencing

- Must ship after: **TD.2** (lint infrastructure).
- Must ship before: **TD.12** (splitting a god-module is cleaner on a shared foundation), **TD.13** (needs one seam to wrap).
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A raw-`fetch` site was intentional (streaming, offline, external origin) and is broken by migration | **H** | M | FR-4 requires per-site audit and justification, not blanket migration; allowlist for legitimate cases |
| Error message wording changes, breaking user-facing copy or tests | M | M | AC-4 asserts message equality; FR-3 requires behavioural identity |
| Session-refresh behaviour changes subtly for the 25 sites | M | H | These sites gain refresh they lacked — verify each intentionally; AC-5 covers it |
| Typed errors (FR-10) become a large refactor of every caller | M | M | Make the typed shape additive — keep `message` so existing callers keep working; adopt branching incrementally |
| Bundle size grows | L | L | AC-8 gate via existing `bundle:check` |

## 15. Rollout Plan

- **Feature flag** — none.
- **Sequencing** — (1) ship the shared helper alongside existing duplicates; (2) migrate the six duplicate modules; (3) audit and migrate the 25 raw-`fetch` sites, allowlisting justified exceptions; (4) enable the FR-7 lint rule; (5) add typed errors (FR-10) as an additive change.
- **Dogfood** — migrate one duplicate module first and verify in staging before the rest.
- **GA criteria** — lint rule active, allowlist stable, one week with no attributable client errors.
- **Rollback** — per-PR revert; the helper is additive.

## 16. Test Plan

- **Unit** — helper: ok path, 204, 4xx with body, 4xx without body, network error, abort; error-message equality against the old implementations.
- **Integration** — migrated modules exercised against a mocked server (MSW or the existing test setup); exported-surface test (FR-8).
- **End-to-end** — `make e2e` green; specifically exercise a formerly-raw-`fetch` feature and a session-expiry scenario.
- **Security** — verify credentials are attached consistently post-migration; verify no token or body logging; confirm no external-origin request accidentally gains an `Authorization` header.
- **Accessibility** — confirm error copy still renders through accessible error components.
- **Performance / load** — `npm run bundle:check`.
- **Manual exploratory** — exercise the offline path and a streaming AI endpoint to confirm FR-6 handling.

Baseline:

```bash
cd clients/web
grep -rn 'async function apiJson\|const apiJson' src/lib/*.ts | wc -l   # 6
grep -rln 'fetch(' --include='*.tsx' src/components src/pages | wc -l   # 25
ls src/lib/*api*.ts | wc -l                                             # 113
```

## 17. Documentation & Training

- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — "all API calls go through the shared helper; raw `fetch` requires justification."
- Helper's TSDoc as the primary reference.
- Note on when raw `fetch` is legitimate (streaming, offline, external origin) with the allowlist process.

## 18. Open Questions

1. Should the helper live in `api.ts` or a new `http.ts`? (Leaning `http.ts` — `api.ts` already mixes URL building, WebSocket URLs, and session refresh.)
2. Which of the 25 raw-`fetch` sites are legitimate? The audit is the first task and may materially change the story's size.
3. Should typed errors (FR-10) ship here or with TD.13, which will consume them more heavily?
4. Is MSW already available for the tests, or does the request-helper suite need new infrastructure?
5. Should the helper add a default timeout? It would be a behaviour change — likely defer to TD.13 rather than smuggle it in here.

## 19. References

- `clients/web/src/lib/api.ts` — `authorizedFetch`, `tryRefreshSession`
- `clients/web/src/lib/errors.ts` — `readApiErrorMessage`
- Duplicates: `av-scan-api.ts:6`, `captions-api.ts:4`, `demographics-api.ts:4`, `email-templates-api.ts:44`, `scheduler-api.ts:4`, `system-email-templates-api.ts:45`
- Related plans: [TD.2](../../completed/tech_debt/TD.2-convention-charter-and-enforcement.md), [TD.12](TD.12-split-courses-api-module.md), [TD.13](TD.13-adopt-server-state-management.md), [TD.3](../../completed/tech_debt/TD.3-repair-and-verify-openapi-contract.md)
