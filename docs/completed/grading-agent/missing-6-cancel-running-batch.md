# GA-M6 — Cancel / stop a running batch

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-M6 |
| **Section** | Grading Agent — Missing Features |
| **Severity** | MAJOR |
| **Markets** | HE / K12 / SL |
| **Status (today)** | DONE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | [GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md) |
| **Unblocks** | safe large-class runs |

## 1. Problem Statement

Once a batch run starts, there is no way to stop it. `handlePostGraderAgentRun` publishes one queue
message per submission and the client only polls progress; there is no cancel endpoint, no run state
the consumer checks, and no UI control. If an instructor notices a bad prompt, the wrong scope, or
runaway cost mid-run on a 300-student class, they can only watch it finish — spending real AI dollars
on grades they will discard. Combined with the lack of a cost estimate ([GA-M7](missing-7-cost-estimate-and-budget.md)),
this makes large runs feel dangerous.

## 2. Goals

- A "Cancel run" control that stops further grading promptly.
- The consumer skips queued items for a cancelled run instead of grading them.
- Clear terminal run state (`cancelled`) and a summary of what was/ wasn't applied.

## 3. Non-Goals

- Rolling back grades already written before cancel (out of scope; surfaced in the summary instead).
- Pausing/resuming (cancel is terminal for v1).

## 4. Personas & User Stories

- **As an instructor**, I want to cancel a run I started by mistake, so that I stop wasting AI budget.
- **As a TA**, I want cancel to take effect quickly, so that few extra submissions are graded after I click it.
- **As an instructor**, I want a clear summary of what was applied before cancel, so that I know the gradebook state.

## 5. Functional Requirements

- **FR-1.** The system MUST expose `POST …/grader-agent/runs/{run_id}/cancel` that sets the run status to `cancelled` (only from `queued`/`running`).
- **FR-2.** The consumer MUST check run status before grading each message and, if `cancelled`, record the item as `skipped` (reason "run cancelled") without an LLM call, then progress the run.
- **FR-3.** The run MUST reach a terminal `cancelled` state once all messages are drained (graded, skipped, or failed).
- **FR-4.** The UI MUST show a Cancel button while `batchRunning`, disable it after click, and surface the final summary.
- **FR-5.** Cancel MUST be permission-gated like other run actions and only by users who can run the agent in that course.

## 6. Non-Functional Requirements

- **Performance** — status check is a cheap indexed read; cache per-run status briefly to avoid hammering Postgres under high concurrency.
- **Security** — RBAC identical to run creation.
- **Privacy & Compliance** — cancelled items leave no partial student-visible state when suggest-only ([GA-M3](missing-3-suggest-only-batch-and-bulk-review.md)).
- **Reliability** — cancel is idempotent; double-cancel is a no-op; works on both RabbitMQ and in-memory buses.
- **Observability** — metric: items skipped due to cancel; time from cancel to drain.
- **Internationalization** — `gradingAgent.run.cancel.*`.
- **Backward compatibility** — additive; runs without cancel behave as today.

## 7. Acceptance Criteria

- **AC-1.** *Given* a running batch, *when* I cancel, *then* the run status becomes `cancelled` and remaining items are skipped.
- **AC-2.** *Given* cancel, *when* the consumer dequeues a remaining message, *then* it records `skipped` without calling the model.
- **AC-3.** *Given* cancel after 40/300 applied, *when* the run drains, *then* the summary reports 40 applied and the rest skipped.
- **AC-4.** *Given* a `done` run, *when* I attempt cancel, *then* it is rejected (not cancellable).
- **AC-5.** *Given* concurrent consumers, *when* I cancel, *then* no item is graded after the cancel is visible.

## 8. Data Model

- `grading_agent_runs.status` already free-text; add `cancelled` as a recognized value (and `finished_at` set on drain).
- Optional `cancelled_at`, `cancelled_by` columns for audit.
- Migration: `server/migrations/NNN_grading_agent_run_cancel.sql` (if audit columns added).

## 9. API Surface

- `POST …/grader-agent/runs/{run_id}/cancel` → `{ status: "cancelled" }`.
- `GET …/runs/{run_id}` already returns status/counts; extend to include cancel metadata.

## 10. UI / UX

- Run popover / dry-run dock: a **Cancel run** button while `batchRunning`; becomes "Cancelling…" then shows the summary.
- Run history ([GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md)) shows `cancelled` runs distinctly.
- Copy/i18n under `gradingAgent.run.cancel.*`.

## 11. AI / ML Considerations

- Cancel's value is precisely to stop further model spend; ensure the status check happens *before* the gateway/LLM call in `HandleGradingAgentQueueMessage`.

## 12. Integration Points

- `server/internal/httpserver/grading_agent_http.go` (cancel handler), `courses_routes.go` (route).
- `server/internal/httpserver/grading_agent_queue.go` (status check at top of `HandleGradingAgentQueueMessage`).
- `server/internal/repos/gradingagent/repo.go` (`CancelRun`, status read).
- `clients/web/src/components/annotation/grader-agent/{run-agent-popover,dry-run-dock}.tsx`, `use-grader-agent-workflow.ts`.

## 13. Dependencies & Sequencing

- Pairs with [GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md) (run history) and [GA-B1](bug-1-queue-overflow-and-stuck-runs.md) (terminal states). Should land alongside the stuck-run fix so `cancelled` and stuck-detection share status handling.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Per-message status read adds DB load | M | M | Short-TTL in-process cache of run status keyed by run id |
| Cancel races with in-flight grading | M | L | Accept that in-flight items finish; only *queued* items are skipped |
| In-memory bus has no per-message control | L | M | Status check is at handler entry, bus-agnostic |

## 15. Rollout Plan

- Flag: `graderAgentCancelRun`.
- Sequence: status read + handler guard → cancel endpoint → UI → flip flag.
- Pilot: a large course.
- Rollback: hide cancel UI; guard is harmless if unused.

## 16. Test Plan

- **Unit** — `CancelRun` transitions; handler guard skips on cancelled.
- **Integration** — cancel mid-run skips remaining; run drains to `cancelled`.
- **E2E** — start large run, cancel, verify summary + gradebook.
- **Security** — RBAC on cancel.

## 17. Documentation & Training

- Help-center: "Cancelling a grading-agent run."

## 18. Open Questions

1. Do we offer "cancel and discard applied grades" as a separate destructive action, or only forward cancel?
2. Should auto-grade-on-submission runs (single item) expose cancel at all?

## 19. References

- `server/internal/httpserver/grading_agent_queue.go` (`HandleGradingAgentQueueMessage`).
- `server/internal/repos/gradingagent/repo.go` (`MarkRunRunning`, `IncrementRunProgress`).
- `clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts` (`batchRunning`, polling).
- Related: [GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md), [GA-B1](bug-1-queue-overflow-and-stuck-runs.md).
