# TD.12 — Split `courses-api.ts` and Restructure the API Client Layer

> Implementation plan. Source: technical-debt static analysis, 2026-07-25. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.12 |
| **Section** | Technical Debt Remediation |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (internal) |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | TD.11 |
| **Unblocks** | TD.13, TD.14 |

---

## 1. Problem Statement

`clients/web/src/lib/courses-api.ts` is **8,631 lines** exporting **521 symbols** — 319 functions and 202 types — and is imported by **215 files**, roughly a third of the web source tree. It is the frontend's counterpart to `internal/httpserver`: a single module that every feature reaches into, where any change has repository-wide blast radius, where merge conflicts are routine, and where a bundler cannot tree-shake meaningfully because everything is entangled. It is not alone — `boards-api.ts` is 1,453 lines and `live-quiz-api.ts` 1,583 — but it is the extreme case. Its 202 hand-written types also duplicate contracts the server already defines, drifting silently whenever the API changes, because until [TD.3](../../completed/tech_debt/TD.3-repair-and-verify-openapi-contract.md) the broken spec made generated types impossible (now repaired; migration still optional).

## 2. Goals

- Split `courses-api.ts` into **domain-scoped modules** mirroring the server's post-[TD.6](TD.6-decompose-httpserver-package.md) boundaries.
- Keep every existing import working during migration — no flag-day rewrite of 215 files.
- Apply the same treatment to the other oversized client modules.
- Reduce merge-conflict surface and improve tree-shaking.
- Position the layer to adopt **generated** types from the repaired OpenAPI spec, retiring hand-written duplicates.

## 3. Non-Goals

- Changing any function signature, endpoint, or payload.
- Introducing caching or a data-fetching library — that is [TD.13](TD.13-adopt-server-state-management.md).
- Migrating to generated types in this story — it *enables* that; adoption is separate and blocked on TD.3.
- Refactoring the components that consume these modules — that is [TD.14](TD.14-decompose-god-components.md).
- Splitting every one of the 113 client modules; only those over budget.

## 4. Personas & User Stories

- **As a web engineer**, I want to import course-enrollment functions from an enrollment module, so that my editor is not loading an 8,631-line file.
- **As an engineer on a feature branch**, I want my API changes not to conflict with every other team's, so that rebases are cheap.
- **As a user**, I want smaller bundles for the routes I visit, so that pages load faster.
- **As a reviewer**, I want an API-layer diff scoped to one domain, so that I can tell what changed.

## 5. Functional Requirements

- **FR-1.** `courses-api.ts` MUST be split into domain modules under `src/lib/courses/` (e.g. `courses.ts`, `modules.ts`, `enrollments.ts`, `settings.ts`, `files.ts`, `outcomes.ts`, `gradebook.ts`), each under the TD.2 size budget.
- **FR-2.** The split MUST be **pure relocation** — no signature, payload, or logic change.
- **FR-3.** During migration, `courses-api.ts` MUST remain as a **re-export barrel** so all 215 importers keep working unchanged.
- **FR-4.** After the split, consumers SHOULD be migrated to direct imports incrementally, and the barrel retired only once no importer remains.
- **FR-5.** A test MUST assert the barrel's exported symbol set is unchanged at every step (TD.1 FR-10 / TD.11 FR-8 mechanism).
- **FR-6.** Types MUST move with the functions that use them; shared types go to a `src/lib/courses/types.ts` (or reuse `courses-api-schemas.ts`).
- **FR-7.** The same treatment MUST be applied to other client modules over budget — `boards-api.ts` (1,453), `live-quiz-api.ts` (1,583), `transcripts-api.ts` (1,479), `courses-api-schemas.ts` (1,071), `build-search-items.ts` (1,020).
- **FR-8.** Module boundaries SHOULD mirror the server's domain packages from TD.6, so a feature maps to one client module and one server package.
- **FR-9.** The barrel MUST NOT be a permanent fixture — FR-4's retirement is part of done, tracked by a shrink-only importer count.
- **FR-10.** The split MUST NOT regress bundle size; it SHOULD improve per-route chunk sizes, verified by `npm run bundle:check` and the existing lazy-route setup.

## 6. Non-Functional Requirements

- **Performance** — better tree-shaking and per-route chunking is a goal, not merely a non-regression. Measure route chunk sizes before and after for the heaviest routes.
- **Security** — no change to request construction or auth; TD.11's shared helper is used throughout.
- **Privacy & Compliance** — n/a.
- **Accessibility** — n/a (no UI change).
- **Scalability** — developer scalability: parallel work on different domains without conflict.
- **Reliability** — a purely mechanical split verified by type checking and the exported-surface test.
- **Observability** — n/a.
- **Maintainability** — the goal.
- **Internationalization** — n/a.
- **Backward compatibility** — the barrel (FR-3) guarantees no consumer breaks during migration.

## 7. Acceptance Criteria

- **AC-1.** *Given* the split is complete, *When* module sizes are measured, *Then* every module under `src/lib/courses/` is within the TD.2 budget.
- **AC-2.** *Given* any migration step, *When* the exported-surface test runs, *Then* the barrel's symbol set is unchanged.
- **AC-3.** *Given* any migration step, *When* `npm run typecheck` runs, *Then* it passes with no consumer changes required.
- **AC-4.** *Given* a split PR, *When* diffed ignoring imports, *Then* no function body has changed.
- **AC-5.** *Given* the split is complete, *When* `npm run bundle:check` runs, *Then* it passes, and per-route chunk sizes for the heaviest routes are recorded (improvement expected).
- **AC-6.** *Given* consumers are migrated to direct imports, *When* the importer count of the barrel is measured, *Then* it decreases monotonically toward zero.
- **AC-7.** *Given* the barrel has zero importers, *When* it is deleted, *Then* `npm run typecheck` and `npm run test` pass.
- **AC-8.** *Given* all FR-7 modules are split, *When* file sizes are measured, *Then* none exceeds the budget and the TD.2 file-size allowlist shrinks accordingly.

## 8. Data Model

No schema change. Target structure:

```
clients/web/src/lib/courses/
  index.ts        # barrel during migration; deleted at the end
  types.ts
  courses.ts      modules.ts     enrollments.ts
  settings.ts     files.ts       outcomes.ts
  gradebook.ts    assignments.ts quizzes.ts
```

`courses-api.ts` becomes a re-export of `./courses` until FR-4 completes.

## 9. API Surface

**No server API change.** Client module structure changes; the public symbol set does not (AC-2).

## 10. UI / UX

No UI change. The only user-facing effect is potentially faster route loads from better chunking (§6).

## 11. AI / ML Considerations

Not applicable directly. Note that AI-related course functions (adaptive content, AI drafting, tutor session) should follow FR-8 and land in the module matching their server domain rather than a client-side `ai.ts` catch-all.

## 12. Integration Points

- `clients/web/src/lib/courses-api.ts` — 8,631 LOC, 521 exports, 215 importers.
- `clients/web/src/lib/courses-api-schemas.ts` — 1,071 LOC; reconcile with FR-6.
- 215 importing files across `src/pages`, `src/components`, `src/hooks`.
- `clients/web/src/lazy-pages.ts` — route-level code splitting; relevant to §6 measurement.
- CI: `bundle:check`, `typecheck`, TD.2 file-size allowlist.

## 13. Dependencies & Sequencing

- Must ship after: **TD.11** (shared helper first, so split modules do not each carry request boilerplate).
- Should align with: **TD.6** (FR-8 mirrors server domains — coordinate the boundary map).
- Must ship before: **TD.13** (query keys and hooks organise along these module boundaries), **TD.14**.
- Enables: generated-type adoption once **TD.3** repairs the spec.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| The barrel becomes permanent and the split delivers no real decoupling | **H** | M | FR-9 makes retirement part of done; AC-6 tracks importer count as a shrink-only ratchet |
| Circular imports between new modules | M | M | Shared types extracted to `types.ts` first (FR-6); dependency-cycle lint rule |
| A function is duplicated rather than moved during the split | M | M | AC-2 exported-surface test catches additions; AC-4 body diff |
| Boundaries drawn differently from the server's, so features map to two structures | M | M | FR-8 explicitly mirrors TD.6; coordinate the map with the backend team |
| Bundle size regresses from barrel re-exports defeating tree-shaking | M | M | AC-5 gate; retire the barrel promptly (FR-4) |
| Merge conflicts with in-flight feature work in `courses-api.ts` | **H** | M | Announce migration windows; split domain by domain; the file is the highest-traffic in the repo |

## 15. Rollout Plan

- **Feature flag** — none.
- **Sequencing** — (1) agree the module map with the backend team (FR-8); (2) extract shared types; (3) move one domain at a time behind the barrel; (4) migrate consumers to direct imports domain by domain; (5) delete the barrel; (6) repeat for the FR-7 modules.
- **Dogfood** — first domain validates the process end to end, including consumer migration.
- **GA criteria** — barrel deleted, all modules within budget, bundle check green, one week with no attributable client errors.
- **Rollback** — per-PR revert; the barrel keeps intermediate states safe.

## 16. Test Plan

- **Unit** — existing tests for these modules pass unchanged; exported-surface test at each step.
- **Integration** — component tests importing these modules pass without modification.
- **End-to-end** — `make e2e` green after each domain move; heavy course flows (modules, enrollments, gradebook) exercised explicitly.
- **Security** — n/a beyond confirming TD.11's helper is used uniformly.
- **Accessibility** — n/a.
- **Performance / load** — record per-route chunk sizes before and after for the heaviest routes; `npm run bundle:check` in CI.
- **Manual exploratory** — smoke the course area (modules, settings, files, enrollments, gradebook) after each domain move.

Baseline:

```bash
cd clients/web
wc -l src/lib/courses-api.ts                       # 8631
grep -cE '^export ' src/lib/courses-api.ts         # 521
grep -rl 'courses-api' src/ | wc -l                # 215
wc -l src/lib/boards-api.ts src/lib/live-quiz-api.ts src/lib/transcripts-api.ts
```

## 17. Documentation & Training

- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — client module boundaries mirror server domains; size budget; no barrels as permanent structure.
- Module map published alongside the TD.6 server map so both teams share one vocabulary.
- Note to engineers when each domain moves, with the new import path.

## 18. Open Questions

1. What is the correct domain decomposition for `courses-api.ts`? The §8 list is a proposal; derive it from actual function clustering and the TD.6 map.
2. Should `courses-api-schemas.ts` merge into the new `types.ts`, or stay separate? (It is separately over budget at 1,071 lines.)
3. Should generated OpenAPI types replace hand-written ones as part of this story, or strictly after TD.3? (Proposed: strictly after — do not couple a mechanical split to a contract change.)
4. Is there a dependency-cycle lint rule available for the web build, or does one need adding?
5. How aggressively should the barrel be retired — is a two-step (split, then migrate consumers) acceptable, or should consumers migrate in the same PR as each domain move?

## 19. References

- `clients/web/src/lib/courses-api.ts` — 8,631 LOC, 521 exports, 215 importers
- `clients/web/src/lib/courses-api-schemas.ts` — 1,071 LOC
- `clients/web/src/lazy-pages.ts` — route-level code splitting
- `clients/web/scripts/check-bundle-size.mjs` — existing bundle gate
- Related plans: [TD.11](TD.11-consolidate-http-client-foundation.md), [TD.13](TD.13-adopt-server-state-management.md), [TD.6](TD.6-decompose-httpserver-package.md), [TD.3](../../completed/tech_debt/TD.3-repair-and-verify-openapi-contract.md)
