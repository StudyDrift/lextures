# ADR 0003 — Course Checklist: code registry, no feature flag, evidence in result

- **Status:** Accepted
- **Date:** 2026-08-04
- **Plan:** [CC.1 Checklist rule registry & evaluation engine](../completed/checklist/CC.1-checklist-registry-and-evaluation-engine.md)

## Context

Instructors need a course-scoped readiness checklist backed by external quality rubrics
(QM, OSCQR, NSQ, UDL, WCAG). We needed a place to define ~100 machine-checkable rules without
paying a migration/route cost per rule, and without a product feature flag (the checklist is
always on for staff).

## Decision

1. **Code registry over DB-driven rules.** Each item is an `ItemDescriptor` in
   `server/internal/service/coursechecklist` with a pure evaluator over an in-memory
   `CourseSnapshot`. Adding a rule is one registry entry + one function — no schema change.
2. **No feature flag.** Safety valves are structural: `RETIRED_ITEM_IDS`, tier demotion
   (`essential` → `recommended`), snapshot TTL (CC.2), and per-loader timeouts. New rules
   ship as `recommended` first.
3. **Evidence in the evaluation result.** Failing items may carry up to 200 `EvidenceRow`
   values in the same `Result`, so the UI can expand a table without a second round trip.

## Consequences

- Rule authors must keep evaluators pure and declare `DataNeeds` so Only-mode stays cheap.
- Persisted dismissals (CC.2) key off stable `ItemID`s; renames go through `ITEM_ID_ALIASES`.
- Catalog growth is bounded by review (`Sources` mandatory) and package file budgets, not by
  schema migrations.
- Clients must tolerate `unknown` rows (panic/error/timeout containment) and treat
  `not_applicable` as hidden / excluded from progress.
