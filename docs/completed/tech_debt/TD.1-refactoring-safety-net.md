# TD.1 — Refactoring Safety Net: Route Inventory & API Characterization Harness

> Implementation plan — **completed 2026-07-25**. Source: technical-debt static analysis. Programme overview: [tech_debt README](../../plan/tech_debt/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.1 |
| **Section** | Technical Debt Remediation |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS (internal — no market-facing change) |
| **Status (today)** | DONE (2026-07-25) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | — |
| **Unblocks** | TD.3, TD.4, TD.5, TD.6, TD.7, TD.9, TD.10 |

---

## 1. Problem Statement

The remediation programme moves and deletes code across a **125,266-LOC** flat Go package serving **1,407 distinct routes**. Today nothing mechanically proves that a refactor left the HTTP contract intact: the 802 Go test files assert behaviour they were written for, but there is no inventory of what the API *is*, so a route silently dropped during a file move, or a JSON field renamed during a struct split, would reach `main` unnoticed. The 190 e2e specs cover happy paths for shipped features, not the full surface. Without a characterization harness the honest risk assessment for TD.6 is "unacceptable", and the programme cannot start.

## 2. Goals

- Produce a **machine-checked inventory** of every registered route (method, path, middleware chain, auth posture) that fails CI when it changes without an accompanying snapshot update.
- Produce **response-shape characterization tests** for the highest-traffic and highest-risk endpoints, pinning status codes, headers, and JSON key sets.
- Make the inventory diff **reviewable**: a PR that intentionally changes the API shows a small, readable snapshot delta.
- Give every later story a single, cheap command to answer "did I change the contract?".
- Keep the harness fast enough to run on every PR (target < 60 s).

## 3. Non-Goals

- Achieving full behavioural coverage of all 1,407 routes — this is a *change-detector*, not a substitute for feature tests.
- Refactoring any handler (TD.5–TD.10 own that).
- Fixing the OpenAPI spec (TD.3 owns that); this story only *records* what is served.
- Performance or load benchmarking.
- Changing e2e or unit-test frameworks.

## 4. Personas & User Stories

- **As a platform engineer**, I want a single command that tells me whether my refactor changed the HTTP contract, so that I can move 125K LOC without fear.
- **As a reviewer**, I want file-move PRs to carry mechanical proof that no route changed, so that I can approve on structure rather than re-reading every handler.
- **As a release manager**, I want CI to block an accidental route deletion, so that a refactor cannot cause a client outage.
- **As an on-call engineer**, I want to diff today's route inventory against the last release, so that I can correlate an incident with an API change.

*(Student / instructor / admin / parent personas are not directly involved — this is internal infrastructure whose success is defined by their experience being unchanged.)*

## 5. Functional Requirements

- **FR-1.** The system MUST provide a Go test that walks the `chi` router returned by `httpserver.NewHandler` and emits a deterministic, sorted inventory of every route as `(method, pattern)`.
- **FR-2.** The inventory MUST be committed as a golden file (`server/internal/httpserver/testdata/route_inventory.golden`) and the test MUST fail when the live router diverges from it.
- **FR-3.** The test MUST support `-update` (or `UPDATE_GOLDEN=1`) to regenerate the golden file, so intentional changes are a deliberate, reviewable act.
- **FR-4.** The inventory MUST record, per route, the **auth posture** — whether the route is reachable unauthenticated, requires a session, or requires elevated scope — derived by probing rather than by annotation, so it cannot drift from reality.
- **FR-5.** The system MUST provide characterization tests that, for a curated set of endpoints, assert HTTP status, `Content-Type`, and the **sorted set of JSON keys** (recursively) of the response body, without asserting volatile values (IDs, timestamps).
- **FR-6.** The curated set MUST cover, at minimum: auth, courses CRUD, course modules, enrollments, gradebook, quiz take/submit, assignments, and the platform features endpoint.
- **FR-7.** Characterization fixtures MUST be seeded deterministically so runs are reproducible.
- **FR-8.** CI MUST run the harness on every PR and MUST fail on any un-snapshotted divergence.
- **FR-9.** The harness SHOULD emit a human-readable summary (routes added / removed / changed) into the CI job output.
- **FR-10.** The web client SHOULD gain an equivalent guard: a test asserting the exported symbol set of each `src/lib/*-api.ts` module, so TD.11/TD.12 module splits cannot silently drop an export.

## 6. Non-Functional Requirements

- **Performance** — route-inventory test < 5 s (no DB); characterization suite < 60 s against a seeded test database.
- **Security** — the harness MUST NOT commit real credentials or PII into golden files; auth probing uses synthetic accounts. Golden files are reviewed as code.
- **Privacy & Compliance** — fixtures use synthetic learner data only; no production data is copied into `testdata/`.
- **Accessibility** — n/a (no UI).
- **Scalability** — inventory generation is O(routes); must stay linear as routes grow.
- **Reliability** — zero flakes. Any nondeterminism (map ordering, time, UUIDs) MUST be normalized before comparison; a flaky guard is worse than no guard because it trains reviewers to ignore it.
- **Observability** — CI output names every diverging route explicitly.
- **Maintainability** — golden files are plain sorted text, diffable in review. No binary snapshots.
- **Internationalization** — responses probed with a fixed `Accept-Language` so locale changes do not churn snapshots.
- **Backward compatibility** — this story adds tests only; no production code path changes.

## 7. Acceptance Criteria

- **AC-1.** *Given* the router is unchanged, *When* CI runs the inventory test, *Then* it passes and reports 1,407 routes (± intentional drift recorded in the golden file).
- **AC-2.** *Given* an engineer deletes a route registration, *When* CI runs, *Then* the build fails naming the removed `(method, pattern)`.
- **AC-3.** *Given* an engineer moves a handler between files or packages without changing logic, *When* CI runs, *Then* the inventory test passes with no golden-file change.
- **AC-4.** *Given* a handler's response struct gains a JSON field, *When* the characterization suite runs, *Then* it fails naming the endpoint and the added key.
- **AC-5.** *Given* an engineer runs the harness with `-update`, *When* they commit, *Then* the diff shows only the intended routes and is reviewable in under a minute.
- **AC-6.** *Given* a route's auth posture changes from authenticated to anonymous, *When* CI runs, *Then* the build fails — this is the single highest-value assertion in the story.
- **AC-7.** *Given* the suite runs twice on an unchanged tree, *When* results are compared, *Then* they are byte-identical (no flakes).

## 8. Data Model

No schema change. This story adds:

- `server/internal/httpserver/testdata/route_inventory.golden` — sorted `METHOD\tPATTERN\tAUTH` lines.
- `server/internal/httpserver/testdata/characterization/*.golden` — per-endpoint key-set snapshots.

Fixture seeding reuses the existing test-database helpers; **no new migration**. If seeding helpers do not currently produce a deterministic dataset, add a `testsupport` seed function rather than mutating existing fixtures used by other suites.

## 9. API Surface

**No API change.** This story observes the API; it does not alter it. The inventory is derived from `chi.Walk` over the router built by `httpserver.NewHandler(Deps{...})` with test dependencies.

Auth posture is probed by issuing each route a request with (a) no credentials, (b) a learner session, (c) an admin session, and recording the resulting status class. Routes with side effects MUST be probed inside a transaction that is rolled back, or excluded via an explicit, reviewed allowlist.

## 10. UI / UX

No user-facing UI. Developer-facing surface:

1. `make route-inventory` prints the current inventory.
2. `make route-inventory-update` regenerates the golden file.
3. CI failure output lists added/removed/changed routes as a bulleted diff.

Document both commands in `AGENTS.md` under **Commands reference**.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- Internal: `server/internal/httpserver/server.go` (`NewHandler`, `Deps`), all `*_routes.go` files.
- Internal: `server/internal/httpserver/testdata/` (new).
- CI: `.github/workflows/` — add harness step to the existing Go job.
- `Makefile` — new targets.
- Web: `clients/web/src/lib/__tests__/api-surface.test.ts` (new, FR-10).

## 13. Dependencies & Sequencing

- Must ship after: nothing.
- Must ship before: **every other TD story**.
- Shared infra: the existing Postgres test database; no new infrastructure.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Auth probing mutates data via side-effecting routes | M | H | Probe inside rolled-back transactions; explicit reviewed allowlist for routes that cannot be probed safely |
| Golden files churn constantly, training reviewers to rubber-stamp `-update` | M | H | Keep inventory to stable fields only (method, pattern, auth class) — never response bodies or counts; require PR body to justify any golden diff |
| Characterization suite becomes flaky on timestamps/UUIDs | M | H | Normalize volatile values before snapshotting; assert key *sets*, never values; AC-7 gates on double-run determinism |
| Suite too slow, gets skipped | L | M | 60 s budget; inventory test runs DB-free and always |
| Harness gives false confidence for non-HTTP behaviour (jobs, WebSockets) | M | M | Document scope explicitly in the harness README; WebSocket routes recorded in inventory but behaviour left to existing tests |

## 15. Rollout Plan

- **Feature flag** — none (test-only code).
- **Sequencing** — (1) inventory test + golden, non-blocking in CI for one week to shake out flakes; (2) flip to blocking; (3) add characterization suite; (4) add web export-surface guard.
- **Dogfood** — run against the last 10 merged PRs retroactively to confirm it would have caught real changes and produced no false positives.
- **GA criteria** — two consecutive weeks blocking in CI with zero flakes.
- **Rollback** — remove the CI step; no production impact.

## 16. Test Plan

- **Unit** — inventory generator produces sorted, deterministic output; normalizer strips UUIDs/timestamps correctly.
- **Integration** — characterization suite against a seeded database; auth-probe matrix.
- **End-to-end** — unchanged; `make e2e` must stay green as a control.
- **Security** — verify the auth-posture probe correctly classifies a known-public route (`/health`) and a known-protected route (`/api/v1/courses`); deliberately weaken one route in a scratch branch and confirm AC-6 fires.
- **Accessibility** — n/a.
- **Performance** — assert harness wall-clock in CI; fail over budget.
- **Manual exploratory** — engineer deletes a random route locally and confirms a clear failure message.

Baseline commands:

```bash
cd server && go test ./internal/httpserver/ -run TestRouteInventory -count=1
make route-inventory | wc -l      # expect 1407
```

## 17. Documentation & Training

- `AGENTS.md` — add `make route-inventory` / `-update` to the commands table.
- `docs/plan/tech_debt/README.md` — link the harness as the programme's definition-of-done gate.
- New `server/internal/httpserver/testdata/README.md` — what the golden files mean, when to regenerate, what a legitimate diff looks like.
- Short internal note for reviewers: "how to review a golden-file diff".

## 18. Open Questions

1. Should the auth-posture probe live in the same test as the inventory, or a separate slower job? (Leaning: separate, so the fast inventory check always runs.)
2. Which endpoints belong in the curated characterization set beyond the FR-6 minimum — driven by request volume from telemetry, or by blast radius?
3. Do WebSocket upgrade routes need their own inventory dimension (subprotocol, auth), or is `(method, pattern)` sufficient for this programme?
4. Should the web export-surface guard (FR-10) block or warn initially?

## 19. References

- `server/internal/httpserver/server.go` — `Deps`, `NewHandler`
- `server/internal/httpserver/courses_routes.go` — largest route registrar (455 lines)
- `e2e/` — 190 existing Playwright specs (control suite)
- `chi` router `Walk` API — <https://pkg.go.dev/github.com/go-chi/chi/v5#Walk>
- Michael Feathers, *Working Effectively with Legacy Code* — characterization-test technique
- Related plans: [TD.2](TD.2-convention-charter-and-enforcement.md), [TD.6](../../plan/tech_debt/TD.6-decompose-httpserver-package.md)

## 20. Implementation notes (2026-07-25)

Shipped as test-only harness (no production path changes):

| Deliverable | Location |
|---|---|
| Route inventory test + print | `server/internal/httpserver/route_inventory_*.go` |
| Golden inventory (1559 routes) | `server/internal/httpserver/testdata/route_inventory.golden` |
| Characterization suite | `server/internal/httpserver/characterization_*.go` + `testdata/characterization/` |
| Harness docs | `server/internal/httpserver/testdata/README.md` |
| Web export surface guard | `clients/web/src/lib/__tests__/api-surface.test.ts` + `api-surface.golden.json` |
| Make targets | `make route-inventory`, `make route-inventory-update` |
| CI | `.github/workflows/ci.yml` step *Route inventory & characterization (TD.1)* |

**Open-question resolutions for this ship:**

1. Auth posture for the fast inventory is `anonymous` \| `session` from unauthenticated probes (DB-free). Elevated scope is covered by characterization + existing RBAC tests rather than a full side-effect-safe probe of every mutator.
2. Characterization set is the FR-6 minimum (auth, courses, modules, enrollments, gradebook, quiz take/submit, assignments, platform features) plus `/health/detailed`.
3. WebSocket routes appear in the inventory as `(method, pattern)`; behaviour remains with existing tests.
4. Web export-surface guard is **blocking** from day one (fails CI on missing/dropped exports).
