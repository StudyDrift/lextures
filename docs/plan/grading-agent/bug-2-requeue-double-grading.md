# GA-B2 — Requeue causes double grading & duplicate results

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-B2 |
| **Section** | Grading Agent — Bugs |
| **Severity** | MAJOR |
| **Bug size** | Medium |
| **Markets** | HE / K12 / SL |
| **Status (today)** | BUG |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Platform / Grading squad |
| **Depends on** | — |
| **Unblocks** | trustworthy counts & cost |

## 1. Problem Statement

The RabbitMQ consumer requeues on handler error: `if err := handler(msg); err != nil { d.Nack(false, true) }`.
`HandleGradingAgentQueueMessage` swallows most errors (it calls `failGradingAgentItem`, which returns
`nil`), but it **returns the error from the final `IncrementRunProgress`** — which runs *after* the grade
has already been written via `UpsertCellWithFlags` and a result row inserted. So a transient DB blip on
that last call requeues a message whose grade is already applied. On redelivery the submission is graded
**again**: a second LLM call (extra cost), a **duplicate** `grading_agent_results` row, a second
gradebook write, and a second progress increment — which can push `completed_count` past `total_count`
and skew applied/failed tallies. There is no idempotency key on `(run_id, submission_id)`.

## 2. Goals

- Make per-submission grading idempotent: a redelivered message must not double-grade, double-insert, or double-count.
- Ensure progress accounting cannot exceed `total_count`.
- Avoid a wasted second LLM call on redelivery.

## 3. Non-Goals

- Removing requeue entirely (it is still useful for genuinely un-started messages).
- Changing the queue technology.

## 4. Personas & User Stories

- **As an instructor**, I want each submission graded once, so that cost and counts are accurate.
- **As an operator**, I want redeliveries to be safe, so that transient DB errors do not corrupt run state.

## 5. Functional Requirements

- **FR-1.** Grading MUST be idempotent per `(run_id, submission_id)`: if a non-dry-run result already exists for that pair, the handler MUST short-circuit (no LLM call, no duplicate insert, no extra increment) and ack.
- **FR-2.** `grading_agent_results` MUST enforce uniqueness on `(run_id, submission_id)` for non-dry-run rows (partial unique index), so duplicate inserts fail fast.
- **FR-3.** Progress increment MUST be coupled to the *first* successful processing only (e.g., increment within the same transaction that inserts the result, guarded by the unique constraint).
- **FR-4.** `completed_count` MUST never exceed `total_count`.
- **FR-5.** The final progress update failing MUST NOT cause re-grading; either make insert+increment atomic, or detect the existing result on redelivery and only retry the increment.

## 6. Non-Functional Requirements

- **Reliability** — exactly-once *effect* (at-least-once delivery tolerated).
- **Performance** — idempotency check is one indexed lookup.
- **Observability** — metric: redeliveries detected/short-circuited.
- **Security** — unchanged.
- **Backward compatibility** — existing single result-per-pair runs are already compliant; add the index `NOT VALID`-style/concurrently if needed to handle any historical dupes.

## 7. Acceptance Criteria

- **AC-1.** *Given* a message redelivered after its grade was applied, *when* processed again, *then* no second LLM call, no duplicate result row, and counts are unchanged.
- **AC-2.** *Given* the unique index, *when* a duplicate insert is attempted, *then* it is rejected and handled as "already processed".
- **AC-3.** *Given* an `IncrementRunProgress` failure, *when* the message is redelivered, *then* only the increment is retried (idempotently), not the whole grading.
- **AC-4.** *Given* any run, *when* it completes, *then* `completed_count == total_count` exactly and never exceeds it.

## 8. Data Model

- `CREATE UNIQUE INDEX … ON assessment.grading_agent_results (run_id, submission_id) WHERE run_id IS NOT NULL AND is_dry_run = false;`
- Consider a `processed` semantics or rely on result existence as the idempotency marker.
- Migration: `server/migrations/NNN_grading_agent_results_unique_run_submission.sql` (de-dupe any existing duplicates first).
- Backfill: collapse pre-existing duplicate result rows (keep the latest), then create the index.

## 9. API Surface

- None public. Internal handler + repo changes.

## 10. UI / UX

- None directly; counts become trustworthy (benefits [GA-M7](missing-7-cost-estimate-and-budget.md) and progress display).

## 11. AI / ML Considerations

- Skipping the second LLM call on redelivery is an explicit cost-correctness win.

## 12. Integration Points

- `server/internal/httpserver/grading_agent_queue.go` (idempotency guard at top; atomic insert+increment).
- `server/internal/repos/gradingagent/repo.go` (`InsertResult` conflict handling; combined insert+progress in a tx; or `ResultExists(runID, submissionID)`).
- `server/internal/gradingagentqueue/queue.go` (ack/nack semantics; consider nack-without-requeue for already-applied).

## 13. Dependencies & Sequencing

- Cleanest after [GA-S1](simplify-1-unify-grade-write-paths.md) (one apply path to make idempotent), but can land independently.
- Complements [GA-B1](bug-1-queue-overflow-and-stuck-runs.md) (accurate counts → reachable `done`).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Existing duplicate rows block the unique index | M | M | De-dupe migration before index creation; create concurrently |
| Atomic insert+increment requires a tx refactor | M | M | Wrap both repo calls in one `pgx.Tx`; or use the unique index to gate the increment |
| Auto-grade (`run_id` per single submission) edge cases | L | L | Auto runs have one item; unique index still holds |

## 15. Rollout Plan

- Sequence: de-dupe + unique index → idempotency guard + atomic progress → ack tuning.
- Rollback: drop the guard (index is harmless to keep).

## 16. Test Plan

- **Unit** — idempotency guard short-circuits on existing result; counts capped at total.
- **Integration** — simulate redelivery after apply → no double effects; insert conflict handled.
- **Concurrency** — two workers same message → exactly one effect.

## 17. Documentation & Training

- Runbook: "How redeliveries are handled; results are unique per (run, submission)."

## 18. Open Questions

1. Prefer `ack` (drop) vs `nack(requeue=false)` for an already-applied redelivery?
2. Should the increment live in the same tx as the result insert, or be a separate idempotent step keyed off result existence?

## 19. References

- `server/internal/gradingagentqueue/queue.go` (`rabbitBus.Consume` — `Nack(false, true)` on handler error).
- `server/internal/httpserver/grading_agent_queue.go` (`HandleGradingAgentQueueMessage` returns `IncrementRunProgress` error after writing the grade).
- `server/internal/repos/gradingagent/repo.go` (`InsertResult`, `IncrementRunProgress`).
- Related: [GA-S1](simplify-1-unify-grade-write-paths.md), [GA-B1](bug-1-queue-overflow-and-stuck-runs.md).
