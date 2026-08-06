# TD.5 — Remove Unreachable In-Handler Method Dispatch

> Implementation plan — **completed 2026-08-05**. Source: technical-debt static analysis, 2026-07-25. Programme overview: [tech_debt README](../../plan/tech_debt/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.5 |
| **Section** | Technical Debt Remediation |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (internal) |
| **Status (today)** | DONE (2026-08-05) |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | TD.1 (safety net) |
| **Unblocks** | TD.6, TD.7 |

---

## 1. Problem Statement

Handlers in `internal/httpserver` re-implement HTTP method dispatch that the `chi` router already performed. A representative handler opens with an `OPTIONS` short-circuit and a `MethodNotAllowed` branch — but it is registered via `r.Post(...)`, so `chi` will never route another method to it. Across the package there are **261 `r.Method == http.MethodOptions` checks in 95 files** and **776 `StatusMethodNotAllowed` sites**, against just **3 method-agnostic registrations** out of 1,407 routes. The overwhelming majority is unreachable. It inflates every handler by 8–10 lines, obscures the actual logic, and — worse — creates a false impression that handlers are method-polymorphic, which makes the TD.6 package split harder to reason about and invites copy-paste propagation into new handlers.

## 2. Goals

- Remove method-dispatch prologues from handlers registered against a single HTTP method.
- Preserve exact observable behaviour for the small number of genuinely method-agnostic routes.
- Verify that `chi`'s own 405/OPTIONS handling produces responses equivalent to what the in-handler code produced — and where it does not, fix it centrally rather than per-handler.
- Shrink the surface TD.6 must relocate and TD.7 must standardise.

## 3. Non-Goals

- Introducing the typed handler toolkit — that is [TD.7](TD.7-handler-toolkit.md).
- Moving handlers between files or packages — that is [TD.6](TD.6-decompose-httpserver-package.md).
- Changing CORS policy (the `corsAll` middleware stays exactly as-is).
- Changing route registration or paths.
- Touching the 3 method-agnostic routes' behaviour.

## 4. Personas & User Stories

- **As an engineer reading a handler**, I want the first line of the body to be the actual work, so that I can understand the endpoint without skipping boilerplate.
- **As a reviewer**, I want handlers short enough to read in one screen, so that logic errors are visible.
- **As an API consumer**, I want `OPTIONS` and wrong-method responses to behave exactly as they do today, so that nothing in my client breaks.
- **As the TD.6 engineer**, I want ~8 fewer lines per handler to move, so that the split diff is smaller.

## 5. Functional Requirements

- **FR-1.** The team MUST first establish, by test, the **current** observable behaviour for: (a) `OPTIONS` on a single-method route, (b) a wrong-method request on a single-method route, for a representative sample across the package.
- **FR-2.** The team MUST confirm `chi`'s default `MethodNotAllowed` and `OPTIONS` behaviour matches FR-1's recorded behaviour; where it differs, a **central** `chi` `MethodNotAllowed` handler MUST be configured to reproduce the current response (status, `Allow` header, body) before any handler code is removed.
- **FR-3.** Method-dispatch prologues MUST be removed **only** from handlers proven to be registered against exactly one HTTP method.
- **FR-4.** Handlers registered via `r.Handle`, `r.HandleFunc`, or `r.Mount` (3 sites) MUST retain their dispatch logic; each MUST be individually reviewed and annotated with a comment explaining why the check is load-bearing.
- **FR-5.** A static check MUST prove the single-method claim per handler — mapping each handler function to its registration site(s) — rather than relying on manual inspection across 390 files.
- **FR-6.** Handlers registered at **more than one** route with **different** methods MUST be treated as method-agnostic and left alone.
- **FR-7.** The TD.1 route inventory and characterization snapshots MUST be unchanged by this work.
- **FR-8.** After completion, a lint rule SHOULD flag new in-handler method dispatch in single-method handlers.
- **FR-9.** Removal MUST proceed in reviewable batches (by file or domain group), not one repository-wide commit.

## 6. Non-Functional Requirements

- **Performance** — marginally fewer branches per request; no measurable change expected. Confirm no regression in p95.
- **Security** — a handler whose method check is load-bearing but is wrongly classified as single-method would become reachable by unintended verbs. FR-5's mechanical proof is the control; this is the story's principal risk.
- **Privacy & Compliance** — n/a.
- **Accessibility** — n/a.
- **Scalability** — n/a.
- **Reliability** — `OPTIONS` behaviour feeds CORS preflight. A regression here breaks the browser client silently for cross-origin cases. Preflight behaviour MUST be covered by an e2e test before and after.
- **Observability** — no change to logging or metrics; verify access-log output for 405s is unchanged.
- **Maintainability** — the point of the story: ~2,000 lines of noise removed from the package that most needs to be readable.
- **Internationalization** — the current 405 body uses `http.StatusText`; preserve exactly.
- **Backward compatibility** — strictly behaviour-preserving. Any observable difference is a defect in this story.

## 7. Acceptance Criteria

- **AC-1.** *Given* a single-method route (e.g. `POST /api/v1/courses/{course_code}/outcomes`), *When* a `GET` is issued, *Then* the status, `Allow` header, and body are byte-identical to the pre-change behaviour.
- **AC-2.** *Given* the same route, *When* an `OPTIONS` request is issued, *Then* the status and headers are byte-identical to pre-change behaviour.
- **AC-3.** *Given* a browser CORS preflight against a mutating endpoint, *When* it is issued cross-origin, *Then* it succeeds exactly as before (covered by e2e).
- **AC-4.** *Given* the 3 method-agnostic registrations, *When* the code is inspected, *Then* their dispatch logic is intact and each carries an explanatory comment.
- **AC-5.** *Given* the static check from FR-5, *When* it runs, *Then* it lists every handler with its registered methods, and no handler flagged single-method is in fact registered with two verbs.
- **AC-6.** *Given* all batches merged, *When* CI runs, *Then* the TD.1 inventory and characterization goldens are unchanged.
- **AC-7.** *Given* the work is complete, *When* the counts are re-measured, *Then* `MethodOptions` checks and `StatusMethodNotAllowed` sites in `internal/httpserver` are reduced to the load-bearing remainder, and the residual count is documented.

## 8. Data Model

No schema change, no new persisted artefacts. One temporary analysis script (`scripts/analyze-handler-methods.go` or similar) supporting FR-5; it may be kept as a lint helper for FR-8.

## 9. API Surface

**No intended API change.** Routes, paths, auth, and success responses are untouched.

The only responses in scope are the *error* paths:

| Case | Today | After |
|---|---|---|
| Wrong method on single-method route | In-handler 405 + `Allow` header | `chi` 405, configured centrally to match |
| `OPTIONS` on single-method route | In-handler 204 | `chi`/`corsAll`, configured to match |

If `chi`'s default differs from the current output in **any** observable way, FR-2 requires the central handler be configured to match — the API contract does not change to suit the refactor.

## 10. UI / UX

No UI. The web client is affected only if preflight or 405 handling changes — which AC-2 and AC-3 forbid.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- Internal: all 95 files in `internal/httpserver` containing `MethodOptions` checks; `server.go` (`NewHandler`) for the central `MethodNotAllowed`/`OPTIONS` configuration.
- Internal: `corsAll` middleware — interacts with `OPTIONS`; must be understood before changing anything.
- Web: `clients/web/src/lib/api.ts` (`authorizedFetch`) — the primary preflight-triggering consumer.
- e2e: cross-origin scenarios.

## 13. Dependencies & Sequencing

- Must ship after: **TD.1** — this story's entire safety argument rests on the characterization harness.
- Must ship before: **TD.6** (smaller diffs to move), **TD.7** (toolkit should not codify boilerplate that is about to be deleted).
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A handler misclassified as single-method becomes reachable by another verb | M | **H** | FR-5 mechanical registration mapping, not manual review; AC-5 asserts it |
| `chi`'s default 405/`OPTIONS` output differs subtly (missing `Allow`, different body) | **H** | M | FR-1 records current behaviour *first*; FR-2 configures the central handler to match before any deletion |
| CORS preflight regression breaks the SPA cross-origin | M | H | AC-3 e2e coverage; `corsAll` untouched; staged rollout |
| Handler registered at multiple paths with different methods | M | H | FR-6 explicitly excludes these; the FR-5 map surfaces them |
| Large mechanical diff hides an unintended logic change | M | H | FR-9 batching; reviewers diff with whitespace-insensitive comparison and confirm only prologues were removed |

## 15. Rollout Plan

- **Feature flag** — none.
- **Sequencing** — (1) build the FR-5 handler→method map and publish it; (2) write FR-1 characterization tests for 405/`OPTIONS`; (3) configure central `chi` handlers to match (no handler edits yet — tests must stay green, proving equivalence); (4) remove prologues in batches; (5) add the FR-8 lint rule.
- **Dogfood** — after step 3, deploy to staging and confirm zero change in 405/preflight metrics before touching any handler. Step 3 is where the real risk lives, and it is independently revertible.
- **GA criteria** — all batches merged; one week in production with no change in 4xx rate attributable to method handling.
- **Rollback** — revert the batch PR; step 3 is separately revertible.

## 16. Test Plan

- **Unit** — per-handler tests asserting 405 and `OPTIONS` behaviour for a sample spanning each route family.
- **Integration** — full request path through the middleware chain, including `corsAll`, for both error cases.
- **End-to-end** — cross-origin preflight against a mutating endpoint; `make e2e` green.
- **Security** — verb-tampering probe: for a sample of mutating endpoints, issue every HTTP verb and assert only the registered one is accepted. This is the key control for the story's main risk and SHOULD be kept permanently.
- **Accessibility** — n/a.
- **Performance / load** — confirm no p95 regression on a representative endpoint.
- **Manual exploratory** — exercise the SPA against a cross-origin API host and confirm no console CORS errors.

Measure the baseline and track progress:

```bash
cd server
grep -rh "r.Method == http.MethodOptions" internal/httpserver/*.go | wc -l   # 261
grep -rh "StatusMethodNotAllowed" internal/httpserver/*.go | wc -l           # 776
grep -rhoE '\b[a-z]+\.(Handle|HandleFunc|Mount)\(' internal/httpserver/*.go | sort | uniq -c  # the 3 exceptions
```

## 17. Documentation & Training

- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — add: "handlers do not check `r.Method`; the router owns dispatch. Method-agnostic handlers are registered with `Handle` and must document why."
- PR-template note for the batch PRs describing what reviewers should verify.
- Record the residual load-bearing count (AC-7) in the TD README baseline table.

## 18. Open Questions

1. Does `chi`'s default 405 emit an `Allow` header matching the current hand-written one? Must be answered empirically in step 1 — it determines how much central configuration FR-2 needs.
2. Does `corsAll` already fully handle `OPTIONS` before routing, making the in-handler `OPTIONS` branches unreachable even on the 3 method-agnostic routes?
3. Should the verb-tampering security probe (§16) become a permanent CI suite, or a one-off validation?
4. Can the FR-5 analysis be done with an AST tool rather than grep, given handler-returning-closure indirection (`d.handleFoo()` returns `http.HandlerFunc`)?

## 19. References

- `server/internal/httpserver/course_outcomes.go:334` — representative handler with the prologue
- `server/internal/httpserver/courses_routes.go` — single-method registrations (`r.Get`, `r.Post`, …)
- `server/internal/httpserver/server.go` — `NewHandler`, `corsAll` middleware
- `chi` routing and `MethodNotAllowed` — <https://pkg.go.dev/github.com/go-chi/chi/v5>
- RFC 9110 §15.5.6 (405 Method Not Allowed), §9.3.7 (OPTIONS)
- Related plans: [TD.1](TD.1-refactoring-safety-net.md), [TD.6](TD.6-decompose-httpserver-package.md), [TD.7](TD.7-handler-toolkit.md)

---

## Residual counts (AC-7) — post completion

Measured 2026-08-05 after prologue removal:

| Metric | Baseline (plan) | Residual (non-test) |
|---|---|---|
| `r.Method == http.MethodOptions` | 261–322 | **7** |
| `StatusMethodNotAllowed` sites | 776–786 | **40** |
| Method-agnostic registrations (`Handle`/`HandleFunc`) | 3 | **1** live (`handleCalDAVCollection`); `http501Handler` Handle stubs empty |
| Multi-method handlers (same func, ≥2 verbs) | — | **39** (keep `switch` / multi-or dispatch) |

**Where residuals live**

- **Infrastructure (keep):** `cors.go`, `not_found_response.go` (central chi 404/405), `unimplemented_v1.go` (Handle stub).
- **Multi-method handlers (FR-6):** e.g. course sections, assignment overrides, SCIM, attendance, org settings, magic-link consume, mobile link policy (PUT+PATCH).
- **CalDAV (FR-4):** `calendar_http.go` `HandleFunc` — OPTIONS/PROPFIND/GET dispatch is load-bearing.

**Guards**

- `python3 scripts/analyze-handler-methods.py --assert-single-ok`
- `bash scripts/check-handler-method-dispatch.sh` (wired into `make lint-structure`)
- Characterization: `TestTD5_*` in `server/internal/httpserver/method_dispatch_characterization_test.go`
