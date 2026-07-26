# TD.9 — Enforce Layering: No Raw Database Access in the HTTP Layer

> Implementation plan. Source: technical-debt static analysis, 2026-07-25. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.9 |
| **Section** | Technical Debt Remediation |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (internal) |
| **Status (today)** | PARTIAL |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team, per-domain engineers |
| **Depends on** | TD.6, TD.8 |
| **Unblocks** | Completes the backend layering contract; empties a TD.2 allowlist |

---

## 1. Problem Statement

The codebase has a repository layer — `internal/repos` — and mostly uses it. But **58 files in `internal/httpserver` bypass it**, executing **130 direct `d.Pool.Query` / `QueryRow` / `Exec` calls** with SQL embedded in HTTP handlers. Each is a small breach with compounding cost: the query is untestable without spinning the whole HTTP stack, it is invisible to anyone auditing data access for a table, it cannot participate in a service-owned transaction, and it will not benefit from TD.8's instrumentation seam. It also sets a precedent — the fastest way to ship is to add one more inline query, so the count grows. For a platform under FERPA and SOC 2 obligations, "where is every read of the submissions table?" should have a mechanical answer, and today it does not.

## 2. Goals

- Relocate all 130 raw database call sites out of `internal/httpserver` into `internal/repos`.
- Make the layering rule **compiler- or CI-enforced** so the count cannot grow again.
- Preserve every query's exact SQL and behaviour — this is a move, not an optimisation.
- Give data-access auditing a single, complete answer.
- Empty the `layering` allowlist created in TD.2.

## 3. Non-Goals

- Rewriting, optimising, or re-indexing any query. The SQL text moves verbatim.
- Changing repo package structure or naming conventions.
- Introducing a service layer between handlers and repos where none exists — handlers may call repos directly, as they do today.
- Addressing raw SQL in other layers (jobs, background workers) unless a shared query is discovered.
- Consolidating duplicate queries found during the move (record them; fix separately).

## 4. Personas & User Stories

- **As a compliance engineer**, I want every query against learner data to live in one layer, so that a data-access audit is mechanical rather than archaeological.
- **As a backend engineer**, I want to test a query without constructing an HTTP request, so that debugging is direct.
- **As an SRE**, I want every query to flow through the instrumented `Querier`, so that slow-query logging has no blind spots.
- **As a reviewer**, I want CI to reject inline SQL in a handler, so that the rule is not a matter of my vigilance.

## 5. Functional Requirements

- **FR-1.** All 130 raw `d.Pool.*` call sites in `internal/httpserver` MUST be relocated to functions in the appropriate `internal/repos` package.
- **FR-2.** Relocated queries MUST keep their SQL text **byte-identical**; only the surrounding function signature changes.
- **FR-3.** New repo functions MUST take `db.Querier` (per [TD.8](TD.8-querier-abstraction-for-repos.md)), not `*pgxpool.Pool`.
- **FR-4.** Each relocated query MUST land in the repo package owning its primary table; where ownership is ambiguous, the choice MUST be recorded in the PR.
- **FR-5.** Where a relocated query duplicates an existing repo function, the team MUST record the duplication and MAY reuse the existing function **only** if it is provably identical in SQL and scan behaviour; otherwise a new function is added and the duplicate noted for separate follow-up.
- **FR-6.** The TD.2 layering check MUST be promoted from allowlisted-warn to blocking once the allowlist is empty.
- **FR-7.** The layering check MUST detect both direct `pgx` usage and raw SQL string literals in `internal/httpserver` and its sub-packages.
- **FR-8.** Relocation MUST proceed in reviewable batches by domain, with TD.1 goldens unchanged at every step.
- **FR-9.** Each new repo function MUST carry at least one test at the repo layer.
- **FR-10.** Queries embedded in **WebSocket** handlers (e.g. `canvas_import_ws.go`, `quizgame_game_ws.go`) MUST be included; these files are among the largest and are easy to overlook.

## 6. Non-Functional Requirements

- **Performance** — no query changes, so no performance change. Verify no accidental N+1 is introduced by splitting a handler's query into multiple repo calls — this is the realistic failure mode.
- **Security** — every relocated query MUST remain parameterised. Any query found using string concatenation with request data is a **SQL-injection finding** requiring immediate escalation, not a routine relocation. Audit each of the 130 sites for this during the move.
- **Privacy & Compliance** — the story's compliance value: complete, auditable data access. Record the affected tables so the data map (`docs/isms/`, RoPA) can be verified against reality.
- **Accessibility** — n/a.
- **Scalability** — n/a.
- **Reliability** — behaviour-preserving; TD.1 goldens are the gate.
- **Observability** — post-migration, all queries flow through the `Querier` seam and gain instrumentation.
- **Maintainability** — the goal.
- **Internationalization** — n/a.
- **Backward compatibility** — internal only; HTTP contract unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* the migration is complete, *When* `internal/httpserver` is scanned, *Then* zero direct `pgx` query calls and zero raw SQL literals remain.
- **AC-2.** *Given* a relocation PR, *When* the SQL text is diffed before and after, *Then* it is byte-identical (AC-2 is verified mechanically, not by eye).
- **AC-3.** *Given* a relocation PR, *When* CI runs, *Then* TD.1 route inventory and characterization goldens are unchanged.
- **AC-4.** *Given* the migration is complete, *When* the TD.2 layering allowlist is inspected, *Then* it is empty and the check is blocking.
- **AC-5.** *Given* a new handler with inline SQL, *When* CI runs, *Then* the build fails citing the layering rule.
- **AC-6.** *Given* each new repo function, *When* the test suite runs, *Then* the function has direct coverage.
- **AC-7.** *Given* the audit in §6, *When* complete, *Then* every one of the 130 sites is confirmed parameterised, or a security finding is filed.
- **AC-8.** *Given* a handler previously issuing one query now calling multiple repo functions, *When* its request pattern is measured, *Then* the query count per request has not increased.

## 8. Data Model

No schema change. Code moves from `internal/httpserver/*.go` into `internal/repos/<domain>/*.go`.

The team SHOULD produce a mapping table (site → target repo package → tables touched) before starting; it doubles as the compliance artefact for §6.

## 9. API Surface

**No HTTP API change.** Handlers keep their signatures and responses; only their internals change from inline SQL to a repo call.

## 10. UI / UX

No UI.

## 11. AI / ML Considerations

Not applicable, though AI-adjacent handlers (grading agent, adaptive content) are likely among the 58 files; their queries move like any other.

## 12. Integration Points

- Internal: 58 files in `internal/httpserver` (post-TD.6, distributed across domain packages).
- Internal: `internal/repos/*` — destination packages.
- Internal: `internal/db` (TD.8) — `Querier`.
- CI: TD.2 layering check, promoted to blocking.
- Compliance: `docs/isms/` data map; `docs/soc2/` evidence.

## 13. Dependencies & Sequencing

- Must ship after: **TD.6** (relocations are simpler once handlers sit in domain packages; doing it before means doing it twice), **TD.8** (new repo functions should take `Querier` from the start).
- Must ship before: nothing hard; completes the backend layering contract.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A query is subtly altered during the move | M | H | FR-2 byte-identical rule; AC-2 mechanical SQL diff |
| Splitting a handler's inline query into multiple repo calls introduces N+1 | M | H | AC-8 per-request query-count assertion on affected endpoints |
| A concatenated (injectable) query is discovered | L | **H** | §6 mandates the audit; escalate as a security finding rather than relocating quietly |
| Wrong repo package chosen; churn later | M | L | FR-4 records the decision; a later move is cheap once the layering rule holds |
| Duplicate repo functions proliferate | M | M | FR-5 records duplicates for separate consolidation; do not merge them opportunistically inside a relocation PR |
| WebSocket handlers overlooked | M | M | FR-10 names them explicitly |
| Layering check has false positives on legitimate SQL-like strings | M | L | Tune during the TD.2 warn-only period before promoting to blocking |

## 15. Rollout Plan

- **Feature flag** — none.
- **Sequencing** — (1) build the site→package mapping table and complete the §6 security audit; (2) relocate domain by domain, one PR each; (3) shrink the TD.2 allowlist with each PR; (4) promote the check to blocking when empty.
- **Dogfood** — first domain validates the process; confirm goldens and query-count assertions work as intended.
- **GA criteria** — allowlist empty, check blocking, one week in production with no attributable incident.
- **Rollback** — per-domain revert.

## 16. Test Plan

- **Unit** — each new repo function tested directly (FR-9/AC-6).
- **Integration** — affected endpoints exercised end-to-end against a live database; per-request query-count assertions for AC-8.
- **End-to-end** — `make e2e` green after each domain.
- **Security** — the §6 parameterisation audit across all 130 sites; re-run any existing SQL-injection tests.
- **Accessibility** — n/a.
- **Performance / load** — compare query counts and p95 for affected endpoints before and after.
- **Manual exploratory** — smoke the affected features on staging per domain.

Baseline:

```bash
cd server
grep -rlE 'd\.Pool\.(Query|QueryRow|Exec)' internal/httpserver/*.go | grep -v _test | wc -l   # 58 files
grep -rhoE 'd\.Pool\.(Query|QueryRow|Exec)' internal/httpserver/*.go | wc -l                  # 130 sites
```

## 17. Documentation & Training

- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — state the layering rule as settled architecture, now enforced.
- `docs/ARCH.md` — update the layering diagram.
- Compliance: file the tables-touched mapping as data-map evidence.
- Note to engineers when the check goes blocking, with the "where does my query go?" decision guide.

## 18. Open Questions

1. Are any of the 130 queries genuinely handler-specific (one-off aggregations for a single endpoint)? If so, does a thin per-domain repo package suffice, or should they join the main table repo?
2. How many of the 130 duplicate existing repo functions? Answering early may shrink the work materially.
3. Should background jobs and workers be audited for the same pattern in this story, or a follow-up? (Proposed: follow-up — scope discipline.)
4. Does the compliance team want the tables-touched mapping in a specific format for the data map?
5. Should the layering check also forbid `internal/service` from raw pool access, or is that legitimate there today?

## 19. References

- `server/internal/httpserver/` — 58 files, 130 raw call sites
- `server/internal/httpserver/canvas_import_ws.go` (1,693 LOC), `quizgame_game_ws.go` (884 LOC) — WebSocket handlers per FR-10
- `server/internal/repos/` — destination layer
- `docs/isms/`, `docs/soc2/` — compliance artefacts consuming §6 output
- Related plans: [TD.2](../../completed/tech_debt/TD.2-convention-charter-and-enforcement.md), [TD.6](TD.6-decompose-httpserver-package.md), [TD.8](TD.8-querier-abstraction-for-repos.md)
