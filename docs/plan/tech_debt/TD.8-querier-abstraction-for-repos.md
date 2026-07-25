# TD.8 — `Querier` Abstraction for the Repository Layer

> Implementation plan. Source: technical-debt static analysis, 2026-07-25. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.8 |
| **Section** | Technical Debt Remediation |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (internal) |
| **Status (today)** | MISSING |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | TD.2 |
| **Unblocks** | TD.9; materially faster test feedback for all backend work |

---

## 1. Problem Statement

Every repository function in `server/internal/repos` takes a concrete `*pgxpool.Pool` — **2,199 such parameters** — and the codebase defines **no `Querier`, `DB`, or `Executor` interface anywhere**. Three consequences follow. First, no repo call can be substituted in a test, so **113 test files** depend on a live PostgreSQL instance and **194 `t.Skip` calls** disable tests when it is absent; the `-short` suite therefore cannot exercise service logic. Second, transaction composition is awkward: a service that must run two repo calls atomically cannot pass a `pgx.Tx` where a `*pgxpool.Pool` is demanded, pushing transaction logic into places it does not belong. Third, cross-cutting concerns — query timing, tracing, slow-query logging, read-replica routing — have no seam to attach to. The concrete type is load-bearing in 2,199 places, and nothing can be inserted between the repo and the database.

## 2. Goals

- Introduce a minimal `Querier` interface that both `*pgxpool.Pool` and `pgx.Tx` satisfy, and migrate repo signatures to accept it.
- Make transactional composition natural: a service passes a `Tx`, and every repo call inside participates.
- Create the seam for cross-cutting instrumentation (tracing, slow-query logging) without touching 2,199 call sites again.
- Enable meaningful `-short` testing of service logic without a database.
- Change **zero** SQL and **zero** behaviour.

## 3. Non-Goals

- Introducing an ORM or query builder. Hand-written SQL with `pgx` stays.
- Generating mocks for every repo — the interface enables test doubles where valuable; it does not mandate them.
- Rewriting queries, adding indexes, or tuning performance.
- Changing the repo package layout or function names.
- Achieving a specific coverage number — this story creates the *capability*; teams use it as they see fit.

## 4. Personas & User Stories

- **As a backend engineer**, I want to unit-test a service's branching logic without Docker running, so that my feedback loop is seconds.
- **As an engineer writing a multi-step mutation**, I want to pass a transaction to several repo calls, so that partial failure cannot leave inconsistent data.
- **As an SRE**, I want per-query tracing and slow-query logs, so that I can diagnose latency without adding instrumentation to 2,199 call sites.
- **As a CI maintainer**, I want the `-short` suite to be genuinely meaningful, so that PR feedback does not always require a database.

## 5. Functional Requirements

- **FR-1.** The system MUST define a `Querier` interface in a shared package (proposed `internal/db`) exposing exactly the `pgx` methods the repo layer actually uses — `Query`, `QueryRow`, `Exec`, and `CopyFrom`/`SendBatch` only if genuinely required.
- **FR-2.** `*pgxpool.Pool` and `pgx.Tx` MUST both satisfy `Querier` without adapters, or with a single trivial adapter if signatures differ.
- **FR-3.** Repo functions MUST be migrated to accept `Querier` in place of `*pgxpool.Pool`, package by package.
- **FR-4.** Migration MUST be **signature-only**: no SQL text, parameter, scan, or error-handling change in the same commit.
- **FR-5.** Existing call sites passing a `*pgxpool.Pool` MUST continue to compile and behave identically — the interface is a widening, not a break.
- **FR-6.** The system MUST provide a transaction helper (e.g. `db.InTx(ctx, pool, func(q Querier) error`) that begins, commits, and rolls back correctly, including on panic.
- **FR-7.** The `Querier` seam MUST support an instrumenting decorator adding tracing spans and slow-query logging, wired through the existing `internal/telemetry`.
- **FR-8.** The instrumenting decorator MUST be opt-in initially and MUST NOT change query semantics or error values.
- **FR-9.** After migration, a lint rule MUST prevent new repo functions from taking `*pgxpool.Pool` directly.
- **FR-10.** Migration MUST proceed one repo package at a time, with `main` releasable throughout.
- **FR-11.** The team SHOULD demonstrate the value by converting a representative sample of DB-dependent tests to run without a database, reducing the `t.Skip` count.
- **FR-12.** Any repo function requiring pool-specific capability (`Acquire`, pool statistics, `LISTEN/NOTIFY`) MUST be identified up front and MUST keep the concrete type, with a documented reason.

## 6. Non-Functional Requirements

- **Performance** — interface dispatch adds a pointer indirection per call; must be unmeasurable at p95. Benchmark a hot read path before and after. The instrumenting decorator, when enabled, must stay within an agreed overhead budget.
- **Security** — no change to query construction. The migration MUST NOT convert any parameterised query into string concatenation; reviewers should treat any SQL-text diff in a migration PR as a defect (FR-4).
- **Privacy & Compliance** — slow-query logging (FR-7) MUST NOT log query *arguments*, which routinely contain learner PII. Log the SQL text and timing only; this is a hard requirement, not a preference.
- **Accessibility** — n/a.
- **Scalability** — the seam enables future read-replica routing; not implemented here, but the interface must not preclude it.
- **Reliability** — FR-6's transaction helper must handle rollback on panic and on context cancellation. A leaked transaction holds a connection and degrades the pool.
- **Observability** — FR-7 is the first coherent per-query observability the codebase can have.
- **Maintainability** — 2,199 signatures change; the migration must be mechanical enough to review at scale.
- **Internationalization** — n/a.
- **Backward compatibility** — internal only; no external consumers.

## 7. Acceptance Criteria

- **AC-1.** *Given* the `Querier` interface, *When* a `*pgxpool.Pool` and a `pgx.Tx` are assigned to it, *Then* both compile without adapters (or with the single documented adapter).
- **AC-2.** *Given* a migrated repo package, *When* its PR is diffed, *Then* only parameter types changed — no SQL text, no scan logic, no error handling.
- **AC-3.** *Given* a migrated repo package, *When* the full test suite runs against a live database, *Then* all tests pass unmodified.
- **AC-4.** *Given* a service using `db.InTx`, *When* an inner repo call returns an error, *Then* the transaction rolls back and no partial write is visible.
- **AC-5.** *Given* a service using `db.InTx`, *When* an inner call panics, *Then* the transaction rolls back and the panic propagates.
- **AC-6.** *Given* the instrumenting decorator is enabled, *When* a query runs, *Then* a span is emitted with SQL text and duration and **no argument values**.
- **AC-7.** *Given* a new repo function taking `*pgxpool.Pool`, *When* CI runs, *Then* the FR-9 lint rule fails the build.
- **AC-8.** *Given* the sample of converted tests (FR-11), *When* `go test ./... -short` runs with no database available, *Then* those tests execute and pass, and the `t.Skip` count is measurably below 194.
- **AC-9.** *Given* a hot read path, *When* benchmarked before and after, *Then* the difference is within noise.

## 8. Data Model

No schema change. New package:

```
server/internal/db/
  querier.go     # the Querier interface
  tx.go          # InTx helper, rollback-on-panic semantics
  instrument.go  # tracing / slow-query decorator (opt-in)
```

## 9. API Surface

**No HTTP API change.** Internal Go signature change only:

```go
// before
func InsertOutcome(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, title, desc string) (*Row, error)

// after
func InsertOutcome(ctx context.Context, q db.Querier, courseID uuid.UUID, title, desc string) (*Row, error)
```

Existing callers passing `d.Pool` compile unchanged (FR-5).

## 10. UI / UX

No UI. Developer experience: faster tests, natural transactions.

## 11. AI / ML Considerations

Not applicable, except that AI services performing multi-step writes (adaptive content generation, grading agent workflows) are among the clearest beneficiaries of FR-6 — their multi-write flows currently have no clean transactional composition.

## 12. Integration Points

- Internal: all packages under `internal/repos/*` (2,199 parameters).
- Internal: `internal/service/*` — callers; unchanged signatures, may adopt `InTx`.
- Internal: `internal/httpserver` — passes `d.Pool`; unchanged.
- Internal: `internal/telemetry` — decorator integration.
- Internal: `internal/logging` — slow-query logging with the redaction constraint from §6.
- CI: FR-9 lint rule; TD.2 enforcement scripts.

## 13. Dependencies & Sequencing

- Must ship after: **TD.2** (lint infrastructure for FR-9).
- Can run in parallel with: TD.6 (different files — repos vs handlers), though coordinate merge windows.
- Must ship before: **TD.9** (handlers moving off raw pool access should land on repo functions that already take `Querier`).
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A migration PR silently alters SQL while changing signatures | M | **H** | FR-4 + AC-2: SQL-text diff in a signature PR is a defect; reviewers check with a text diff of query literals only |
| Slow-query logging leaks learner PII via query arguments | M | **H** | §6 hard requirement; AC-6 asserts no arguments logged; privacy review of `instrument.go` |
| Transaction helper leaks connections on panic or cancellation | M | H | AC-4/AC-5 explicit tests; connection-count assertion in integration tests |
| 2,199-signature change creates unreviewable PRs | **H** | M | FR-10 one package at a time; changes are mechanical and greppable |
| Interface too wide (mirrors all of `pgx`), providing no real seam | M | M | FR-1 restricts to methods actually used; audit usage first |
| Pool-specific capabilities discovered mid-migration | M | M | FR-12 identifies them up front; those functions keep the concrete type |
| Mocking becomes fashionable and tests assert on SQL strings rather than behaviour | M | M | Convention guidance: prefer real-DB integration tests for repos; use doubles for *service* logic, not to re-test SQL |
| Merge conflicts with TD.6 and active feature work | M | M | Coordinate merge windows; repos and handlers are largely disjoint files |

## 15. Rollout Plan

- **Feature flag** — none for the interface. The instrumenting decorator (FR-7) is behind a config flag, default off.
- **Sequencing** — (1) define `Querier`, audit `pgx` method usage across repos, identify FR-12 exceptions; (2) ship `InTx` and prove it on one service; (3) migrate repo packages one at a time, smallest first; (4) add the FR-9 lint rule once migration is substantially complete; (5) ship the decorator off by default, enable in staging, then production.
- **Dogfood** — first migrated package plus one converted test file demonstrate the value before committing to the full sweep.
- **GA criteria** — all non-exception repo packages migrated; lint rule active; decorator running in production for one week with no latency regression.
- **Rollback** — per-package revert; the decorator is config-flagged off.

## 16. Test Plan

- **Unit** — `InTx` commit, rollback-on-error, rollback-on-panic, rollback-on-cancellation; decorator emits correct spans and omits arguments.
- **Integration** — every migrated repo package's existing suite passes unmodified against a live database (AC-3); connection-leak assertions.
- **End-to-end** — `make e2e` green after each package.
- **Security** — confirm no query text changed during migration (mechanical diff of SQL literals before/after); confirm no argument logging.
- **Accessibility** — n/a.
- **Performance / load** — benchmark a hot read path (e.g. course listing) before and after; measure decorator overhead separately.
- **Manual exploratory** — run the `-short` suite with Postgres stopped and confirm the converted tests still execute.

Baseline:

```bash
cd server
grep -rho 'pool \*pgxpool\.Pool' internal/repos/*/*.go | wc -l    # 2199
grep -rh 't.Skip' internal/ --include='*_test.go' | wc -l          # 194
grep -rl 'DATABASE_URL\|testing.Short()' internal/ --include='*_test.go' | wc -l  # 113
```

## 17. Documentation & Training

- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — repo functions take `Querier`; services own transactions via `InTx`; repos never begin transactions themselves.
- `internal/db` package doc — the primary reference, with `InTx` usage examples.
- Testing guide — when to use a real database versus a double; explicitly discourage asserting on SQL strings.
- Runbook — enabling slow-query logging and reading the output.

## 18. Open Questions

1. Which `pgx` methods does the repo layer actually use? Audit before defining the interface (FR-1) — a wide interface defeats the purpose.
2. Does anything rely on pool-specific behaviour (`Acquire`, `Stat`, `LISTEN/NOTIFY`, `CopyFrom`)? FR-12 depends on this answer.
3. Should `Querier` live in `internal/db` or beside the repos? (Leaning `internal/db` — repos depend on it, not vice versa.)
4. Is the target to eliminate DB-dependent tests, or to make DB-free testing *possible*? (Proposed: the latter. Repo tests should keep hitting a real database — mocked SQL tests are worse than none.)
5. What is the slow-query threshold, and does it belong in config or platform settings?
6. Should read-replica routing be designed for now, or deliberately deferred so the interface stays minimal?

## 19. References

- `server/internal/repos/` — 2,199 `*pgxpool.Pool` parameters, no interfaces
- `server/internal/telemetry/` — existing observability layer for decorator integration
- `server/internal/logging/` — redaction path relevant to §6
- `pgx` v5 — <https://pkg.go.dev/github.com/jackc/pgx/v5>
- Related plans: [TD.2](TD.2-convention-charter-and-enforcement.md), [TD.9](TD.9-enforce-repo-layering.md), [TD.6](TD.6-decompose-httpserver-package.md)
