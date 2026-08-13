# TD.6 — Decompose `internal/httpserver` into Domain Packages

> Implementation plan. Source: technical-debt static analysis, 2026-07-25. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.6 |
| **Section** | Technical Debt Remediation |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS (internal) |
| **Status (today)** | MISSING |
| **Estimated effort** | XL (>2mo) |
| **Owner (proposed)** | Backend platform team, with a named engineer per domain slice |
| **Depends on** | TD.1, TD.2, TD.5 |
| **Unblocks** | TD.9, TD.10 |

---

## 1. Problem Statement

`server/internal/httpserver` is a single flat Go package holding **390 non-test files, 679 files in total, and 125,266 non-test lines** — roughly 37% of the server's Go source. Every handler for every domain (courses, quizzes, grading, transcripts, badges, SCIM, Canvas import, admin, billing, boards, portfolios) shares one namespace, one `Deps` god-struct with **32 fields**, and one set of package-private helpers. The consequences are concrete: any identifier collides across the whole domain space, so names grow long and defensive; `registerCourseRoutes` alone is a **455-line function** registering **419 routes**; a change to a shared helper has 390 files of blast radius with no compiler-enforced boundary; and Go's package-level compilation means routine edits recompile the whole package. New engineers cannot form a mental model, and code review cannot rely on structure. This is the single largest structural obstacle in the codebase and the root cause behind TD.7, TD.9, and TD.10.

## 2. Goals

- Split `internal/httpserver` into **domain-oriented sub-packages** with explicit, compiler-enforced boundaries.
- Keep the split **purely mechanical** — file relocation and visibility adjustment only, zero logic change, proven by TD.1.
- Establish a shared kernel (routing contracts, middleware, common helpers) that domain packages depend on, with no domain→domain dependencies.
- Break `registerCourseRoutes` and its siblings into per-domain registrars composed at the root.
- Bring every resulting package under the TD.2 package-size budget (≤ 40 non-test files) and remove `internal/httpserver` from the allowlist.

## 3. Non-Goals

- Changing any route, path, request/response shape, status code, or auth rule.
- Refactoring handler *internals* — TD.7 (toolkit) and TD.9 (layering) own that, and both run after or alongside, never inside the move commits.
- Decomposing `Deps` — that is [TD.10](TD.10-composition-root-decomposition.md); this story may pass the existing struct through unchanged.
- Rewriting the repo or service layers.
- Introducing a new HTTP framework. `chi` stays.

## 4. Personas & User Stories

- **As a new backend engineer**, I want to find the grading handlers in a `grading` package, so that I can orient in minutes rather than days.
- **As a domain owner**, I want a compiler-enforced boundary around my code, so that unrelated changes cannot reach into my helpers.
- **As a reviewer**, I want a PR touching only `httpserver/quizzes/`, so that I know its blast radius without reading 390 files.
- **As an engineer running tests**, I want to compile and test one domain, so that my feedback loop is seconds not minutes.
- **As a tech lead**, I want CODEOWNERS to work per domain, so that review routing is automatic.

## 5. Functional Requirements

- **FR-1.** The team MUST publish a **target package map** — every one of the 390 files assigned to a destination package — and get it reviewed **before** any file moves.
- **FR-2.** The package map MUST be derived from the route taxonomy and existing file-name clustering, and MUST be validated against the actual import/symbol graph rather than by name alone.
- **FR-3.** A shared kernel package (proposed `internal/httpserver/kernel`) MUST hold cross-domain concerns: middleware, auth guards, error writing, pagination, and shared request/response helpers.
- **FR-4.** Domain packages MUST NOT import one another. Cross-domain needs go through the kernel or a service package. A CI check MUST enforce this.
- **FR-5.** Each domain package MUST expose a single route-registration entry point (e.g. `courses.Routes(deps) func(chi.Router)`), composed at the root.
- **FR-6.** Each move MUST be a **pure relocation**: the diff contains only file paths, package clauses, import lines, and identifier-visibility changes. Any logic change MUST be a separate PR.
- **FR-7.** After every move PR, the TD.1 route inventory and characterization goldens MUST be unchanged.
- **FR-8.** `git log --follow` MUST continue to work for moved files (use `git mv`; avoid delete+create).
- **FR-9.** Tests MUST move with their subjects; each domain package MUST carry its own tests.
- **FR-10.** Package-private helpers used by multiple domains MUST be promoted to the kernel with a recorded rationale, **not** duplicated into each domain.
- **FR-11.** Every resulting package MUST satisfy the TD.2 budget (≤ 40 non-test files); packages that cannot MUST be split further or granted a documented, time-boxed exception.
- **FR-12.** `CODEOWNERS` SHOULD be updated per domain package as it lands.
- **FR-13.** The split MUST be executed **incrementally** — one domain per PR — with `main` releasable at every point.

## 6. Non-Functional Requirements

- **Performance** — no runtime change. Build and test times SHOULD improve; measure before/after as a success metric.
- **Security** — auth guards (`requireCourseAccess`, used **329 times**) move to the kernel. A guard that silently changes visibility or behaviour during the move is the highest-severity failure mode; kernel promotion PRs need security review.
- **Privacy & Compliance** — no data handling changes. Audit-logging call sites must move intact; confirm SOC 2 audit-trail coverage is unaffected.
- **Accessibility** — n/a.
- **Scalability** — n/a at runtime; developer scalability is the goal.
- **Reliability** — `main` must remain releasable throughout. No long-lived refactor branch.
- **Observability** — logging and metrics call sites move unchanged. Verify metric label values that embed package or function names do not shift, which would break dashboards and alerts.
- **Maintainability** — the entire point.
- **Internationalization** — n/a.
- **Backward compatibility** — `internal/` packages have no external consumers; the HTTP contract is what must be preserved, and TD.1 proves it.

## 7. Acceptance Criteria

- **AC-1.** *Given* the package map, *When* it is reviewed, *Then* all 390 files have a named destination and no file is unassigned.
- **AC-2.** *Given* a domain move PR, *When* CI runs, *Then* the TD.1 route inventory golden is **unchanged** and all characterization snapshots match.
- **AC-3.** *Given* a domain move PR, *When* a reviewer diffs it ignoring whitespace and import lines, *Then* no statement inside any function body has changed.
- **AC-4.** *Given* the split is complete, *When* the import graph is analysed, *Then* there are **zero** domain→domain edges.
- **AC-5.** *Given* the split is complete, *When* package sizes are measured, *Then* every package is ≤ 40 non-test files and `internal/httpserver` is removed from the TD.2 allowlist.
- **AC-6.** *Given* a moved file, *When* `git log --follow` is run, *Then* full history is preserved.
- **AC-7.** *Given* the split is complete, *When* `registerCourseRoutes` is inspected, *Then* it no longer exists as a 455-line function; route registration is distributed across domain registrars.
- **AC-8.** *Given* any point during the programme, *When* `main` is deployed, *Then* it is releasable — no intermediate state is broken.
- **AC-9.** *Given* the split is complete, *When* build and test wall-clock are measured, *Then* the figures are recorded (improvement expected, not required).

## 8. Data Model

No schema change. Structural target:

```
server/internal/httpserver/
  server.go              # NewHandler: composes domain registrars only
  kernel/                # middleware, guards, error writing, pagination, shared helpers
  courses/               # course CRUD, settings, features, files, enrollments
  modules/               # course structure, content pages, modules
  assessment/            # quizzes, question banks, quiz game, live quiz
  grading/               # gradebook, grading agent, submissions, peer review, annotation
  identity/              # auth, me, SSO/OIDC/SAML/LTI, SCIM provisioning
  org/                   # orgs, roles/RBAC, multi-tenancy, consortium
  admin/                 # admin console, platform settings, scheduler
  integrations/          # Canvas import/sync, third-party connectors, webhooks, bots
  credentials/           # transcripts, diplomas, badges, eportfolio, report cards
  collaboration/         # boards, discussions, feed, conferences, broadcasts
  adaptive/              # adaptive paths, adaptive content, diagnostics, learner profile
  billing/               # wallet, billing, payments, marketplace
```

Package names and boundaries are a **proposal**; FR-1's reviewed map is the contract. Expect the map exercise to move files between these buckets.

## 9. API Surface

**No API change whatsoever.** This is the story's defining constraint. All 1,407 routes keep identical paths, methods, auth, request shapes, response shapes, status codes, and headers.

Route registration is restructured internally:

```go
// before — server.go
d.registerCourseRoutes(r)   // 455 lines, 419 routes, one function

// after — server.go
r.Group(courses.Routes(deps))
r.Group(modules.Routes(deps))
r.Group(assessment.Routes(deps))
// …one registrar per domain
```

TD.1's inventory is the acceptance gate: identical golden file before and after.

## 10. UI / UX

No UI. Developer-facing outcome: a navigable tree where file location predicts responsibility.

## 11. AI / ML Considerations

Not directly applicable. Note that AI-touching handlers (grading agent, adaptive content, tutor session, AI reports) are spread across the current flat package; the map should co-locate them with their domain rather than creating an `ai` catch-all, so that AI features stay owned by the domain teams that ship them.

## 12. Integration Points

- Internal: all 390 files in `internal/httpserver`; `server.go` (`NewHandler`, `Deps`).
- Internal: `internal/app/app.go` (440 lines) — builds `Deps`; must keep compiling as registrars change.
- Internal: `internal/repos/*`, `internal/service/*` — imported by handlers; unchanged by this story.
- CI: TD.2 package-budget and layering checks; new domain-isolation check (FR-4).
- `CODEOWNERS`.

## 13. Dependencies & Sequencing

- Must ship after: **TD.1** (the only thing making this safe), **TD.2** (budgets and boundary checks), **TD.5** (removes ~2,000 lines of boilerplate that would otherwise be moved).
- Should ship after: **TD.4** (do not relocate code destined for deletion).
- Must ship before: **TD.9** (layering enforcement is per-package), **TD.10** (`Deps` decomposition follows package boundaries).
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A move PR quietly changes behaviour | M | **H** | FR-6 pure-move rule; AC-3 body-diff review; AC-2 golden gate; move and change never in one PR |
| Shared helper promoted to kernel changes semantics for one caller | M | H | Kernel promotions are separate, small, individually reviewed PRs — never bundled into a domain move |
| Auth guard behaviour drifts during kernel promotion | L | **H** | Security review required for guard promotions; verb/authz matrix tests re-run |
| Domain boundaries drawn wrong; churn as files move twice | **H** | M | FR-1 reviewed map first; accept that the map will be revised — revising a document is cheap, re-moving files is not |
| Long-lived refactor branch diverges from active feature work | **H** | H | FR-13 one domain per PR, merged continuously; **no** refactor branch |
| Merge conflicts with in-flight feature work in the same files | **H** | M | Sequence domains against the feature roadmap; move quiet domains first; announce each domain's move window |
| Circular dependencies discovered mid-split | M | M | FR-2 validates against the real symbol graph before moving; cycles resolved by kernel promotion or interface extraction |
| Metric/log labels embedding package names shift, breaking dashboards | M | M | §6 requires an explicit check; grep for `runtime.Caller`, `%T`, and package-name-derived labels before moving |

## 15. Rollout Plan

- **Feature flag** — none (compile-time restructuring).
- **Sequencing** —
  1. Build and review the package map (FR-1). Nothing moves until this is signed off.
  2. Create the `kernel` package; promote shared helpers in small, individually reviewed PRs.
  3. Move domains one PR at a time, starting with the **smallest, least active** domain to validate the process end-to-end.
  4. Split route registration per domain as each lands.
  5. After each domain, verify goldens, deploy to staging, soak for one business day.
  6. Remove `internal/httpserver` from the TD.2 allowlist once all packages are under budget.
- **Dogfood** — the first domain move is the pilot; hold a retrospective before proceeding to the rest.
- **GA criteria** — all domains moved, AC-4/AC-5 satisfied, two weeks in production with no attributable incident.
- **Rollback** — per-domain `git revert`. Because each PR is a pure move, reverts are mechanical.

## 16. Test Plan

- **Unit** — all existing tests move with their subjects (FR-9) and pass unmodified. A test requiring modification to compile after a move signals the move was not pure — investigate rather than adjusting the test.
- **Integration** — full server suite with a live database after each domain.
- **End-to-end** — `make e2e` green after every domain PR; this is the primary behavioural gate alongside TD.1.
- **Security** — re-run the authz matrix after every kernel guard promotion; verb-tampering probe from TD.5 stays green.
- **Accessibility** — n/a.
- **Performance / load** — record build time, test time, and binary size before and after (AC-9); smoke a p95 latency check per domain on staging.
- **Manual exploratory** — per-domain smoke checklist on staging before proceeding to the next domain.

Baseline measurement:

```bash
cd server
find internal/httpserver -maxdepth 1 -name '*.go' ! -name '*_test.go' | wc -l   # 390
find internal/httpserver -maxdepth 1 -name '*.go' ! -name '*_test.go' -exec cat {} + | wc -l   # 125266
grep -c 'requireCourseAccess' internal/httpserver/*.go | awk -F: '{s+=$2} END{print s}'        # 329
```

## 17. Documentation & Training

- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — document the package map, the kernel's role, and the no-domain-to-domain rule as the standing architecture.
- `docs/ARCH.md` — update the server architecture section to reflect the new tree.
- `AGENTS.md` — update navigation guidance for agents and engineers.
- `CODEOWNERS` — per-domain ownership.
- Team session walking through the map before step 3 begins.

## 18. Open Questions

1. Is the §8 package map right? It is a proposal from route taxonomy and filename clustering; FR-1's validated map supersedes it.
2. Should the kernel be one package or several (`kernel/middleware`, `kernel/guards`, `kernel/render`)? Start as one; split if it exceeds the budget.
3. How do domains share request/response DTOs that legitimately cross boundaries (e.g. a course summary embedded in a gradebook response)? Kernel types, or per-domain duplication with explicit mapping?
4. Should this story wait for TD.7's toolkit so handlers are moved in their final form, or move first and refactor in place? (Leaning: move first — TD.7 on a split tree is far more parallelisable.)
5. What is the sequencing against the active feature roadmap (AC, AP, IQ, VC plans all have in-flight work in this package)? Needs a joint plan with those owners before step 3.
6. Does `internal/app/app.go` also need splitting, or does TD.10 cover it?

## 19. References

- `server/internal/httpserver/` — 390 non-test files, 125,266 LOC
- `server/internal/httpserver/server.go` — `Deps` (32 fields), `NewHandler`
- `server/internal/httpserver/courses_routes.go:5` — `registerCourseRoutes`, 455 lines
- `server/internal/app/app.go` — dependency construction
- `docs/ARCH.md` — current architecture documentation
- Related plans: [TD.1](../../completed/tech_debt/TD.1-refactoring-safety-net.md), [TD.2](../../completed/tech_debt/TD.2-convention-charter-and-enforcement.md), [TD.5](../../completed/tech_debt/TD.5-remove-unreachable-method-dispatch.md), [TD.7](../../completed/tech_debt/TD.7-handler-toolkit.md), [TD.9](TD.9-enforce-repo-layering.md), [TD.10](TD.10-composition-root-decomposition.md)
