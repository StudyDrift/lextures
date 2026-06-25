# GA-B1 — In-memory queue overflow & stuck runs

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-B1 |
| **Section** | Grading Agent — Bugs |
| **Severity** | BLOCKER |
| **Bug size** | Large |
| **Markets** | HE / K12 (single-node & large-class deployments) |
| **Status (today)** | BUG |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / Grading squad |
| **Depends on** | — |
| **Unblocks** | [GA-M6](missing-6-cancel-running-batch.md), reliable large runs |

## 1. Problem Statement

Two coupled defects make large batch runs unreliable, especially on deployments without RabbitMQ:

1. **Overflow.** The in-memory bus is `make(chan QueueMessage, 128)` and `Publish` is **non-blocking**:
   if the buffer is full it returns `"memory queue full"`. `handlePostGraderAgentRun` publishes one
   message **per submission** in a loop. A class with > 128 submissions (or a consumer slower than the
   publish loop) hits a full channel; `Publish` errors mid-loop.
2. **Stuck run.** On that error the handler returns `500 "Failed to enqueue run"` — but the run was
   already created, `MarkRunRunning` was already called, and some messages were already enqueued and
   will grade. There is no rollback and **no terminal failure state for the run**:
   `IncrementRunProgress` only flips status to `done` when `completed_count >= total_count`, which can
   never happen because the un-enqueued items never complete. The run is stuck in `running` forever, the
   client polls it indefinitely, and the gradebook is left partially graded with no signal.

## 2. Goals

- No silent message loss when enqueuing a batch (back-pressure or durable enqueue).
- Atomic-ish enqueue: either the whole run is queued or it fails cleanly with the run marked failed.
- A terminal `failed` (or `cancelled`, see [GA-M6](missing-6-cancel-running-batch.md)) run state and a stuck-run reconciler so no run polls forever.

## 3. Non-Goals

- Replacing the queue technology; fix the in-memory path and the enqueue/lifecycle logic.
- Cancel UX (separate plan [GA-M6](missing-6-cancel-running-batch.md), but shares terminal-state plumbing).

## 4. Personas & User Stories

- **As an instructor on a single-node install**, I want a 200-student run to fully enqueue, so that everyone gets graded.
- **As an instructor**, I want a run that fails to enqueue to say so, so that I am not stuck watching a frozen progress bar.
- **As an operator**, I want stuck runs auto-reconciled, so that they do not accumulate in `running`.

## 5. Functional Requirements

- **FR-1.** In-memory `Publish` MUST NOT drop messages: use a blocking send with context cancellation (and/or a larger/unbounded-with-backpressure buffer), so enqueuing N > 128 items succeeds.
- **FR-2.** Enqueue MUST be all-or-nothing from the user's perspective: if any publish fails, the run MUST be marked terminally failed and the response MUST report how many items were/weren't queued.
- **FR-3.** A run MUST reach a terminal state even when fewer than `total_count` items complete (reconciler marks runs `failed` after a timeout with no progress, or enqueue accounts for the true queued count).
- **FR-4.** `total_count` MUST reflect the number actually enqueued (set/adjust after enqueue), so `done` is reachable.
- **FR-5.** The client MUST stop polling and show a clear error when a run is terminally failed.

## 6. Non-Functional Requirements

- **Performance** — blocking publish bounded by consumer throughput; publish loop SHOULD run off the request goroutine for large batches (enqueue asynchronously) to avoid long-held HTTP requests.
- **Reliability** — no message loss; terminal state guaranteed; reconciler idempotent.
- **Scalability** — validate against a 500-submission run on the in-memory bus and on RabbitMQ.
- **Observability** — metrics: enqueue failures, stuck-run reconciliations, run terminal states.
- **Security** — unchanged RBAC.
- **Backward compatibility** — RabbitMQ path already blocks on publish; align in-memory semantics.

## 7. Acceptance Criteria

- **AC-1.** *Given* a 300-submission run on the in-memory bus, *when* started, *then* all 300 enqueue and the run reaches `done`.
- **AC-2.** *Given* an induced publish failure, *when* it occurs, *then* the run is marked `failed`, the response states the partial count, and the client stops polling with an error.
- **AC-3.** *Given* a run that loses some workers, *when* the reconciler runs, *then* the run is marked `failed` after the no-progress timeout (not stuck in `running`).
- **AC-4.** *Given* `total_count` adjusted to the enqueued count, *when* all enqueued items finish, *then* `status = done`.

## 8. Data Model

- `grading_agent_runs`: ensure a `failed` terminal status is recognized; add `last_progress_at TIMESTAMPTZ` to support the reconciler.
- Migration: `server/migrations/NNN_grading_agent_run_terminal_states.sql`.
- Backfill: set `last_progress_at = created_at` for existing rows; optionally mark long-stale `running` runs `failed`.

## 9. API Surface

- `POST …/grader-agent/runs` response gains `queuedCount` (and `failed` indication when partial).
- `GET …/runs/{run_id}` returns terminal `failed` with a reason.

## 10. UI / UX

- Run popover/dock shows a terminal error state for failed runs and stops the 1.5 s poll loop.
- Run history ([GA-M1](missing-1-persistent-review-queue.md)) shows failed runs distinctly.
- Copy/i18n under `gradingAgent.run.failed*`.

## 11. AI / ML Considerations

- None directly; preventing stuck/partial runs avoids paying for half-graded classes.

## 12. Integration Points

- `server/internal/gradingagentqueue/queue.go` (`memoryBus.Publish` blocking semantics, buffer).
- `server/internal/httpserver/grading_agent_http.go` (`handlePostGraderAgentRun` enqueue loop + lifecycle).
- `server/internal/repos/gradingagent/repo.go` (`CreateRun`/`MarkRun*`/terminal states, reconciler query).
- A background reconciler (new) near `server/internal/background/`.
- `clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts` (poll stop on terminal).

## 13. Dependencies & Sequencing

- Shares terminal-state plumbing with [GA-M6](missing-6-cancel-running-batch.md); land together or B1 first.
- Benefits from [GA-S1](simplify-1-unify-grade-write-paths.md) (single apply path) for the idempotency in [GA-B2](bug-2-requeue-double-grading.md).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Blocking publish stalls the HTTP request for big runs | M | M | Enqueue asynchronously; return `202` immediately with `queuedCount` pending, or chunk enqueue in a goroutine |
| Reconciler races a slow-but-live run | M | M | Use `last_progress_at` with a generous timeout; only fail runs with no progress |
| Backfill mislabels legitimately-running runs | L | M | Only auto-fail runs stale beyond the timeout at migration time |

## 15. Rollout Plan

- Flag: none (correctness fix); stage the reconciler behind config to tune the timeout.
- Sequence: blocking publish + enqueue lifecycle → terminal states + reconciler → client poll-stop.
- Rollback: revert publish change (note: reverting reintroduces overflow — prefer forward fix).

## 16. Test Plan

- **Unit** — `memoryBus.Publish` blocks and never drops; enqueue failure marks run failed.
- **Integration** — 500-item run completes on memory bus; induced failure → terminal failed; reconciler fails a stalled run.
- **Load** — publish/consume throughput under concurrency.
- **E2E** — large run completes; failed run surfaces error and stops polling.

## 17. Documentation & Training

- Runbook: in-memory vs RabbitMQ queue behavior, run lifecycle, reconciler timeout.

## 18. Open Questions

1. Make the in-memory buffer unbounded-with-backpressure, or just enqueue asynchronously and keep 128?
2. Should the enqueue loop move fully off the request goroutine (return `202` with a pending count)?
3. What no-progress timeout balances responsiveness vs slow models?

## 19. References

- `server/internal/gradingagentqueue/queue.go` (`newMemoryBus`, `memoryBus.Publish` — 128 cap, non-blocking).
- `server/internal/httpserver/grading_agent_http.go` (`handlePostGraderAgentRun`).
- `server/internal/repos/gradingagent/repo.go` (`IncrementRunProgress`, `MarkRunRunning`).
- Related: [GA-M6](missing-6-cancel-running-batch.md), [GA-B2](bug-2-requeue-double-grading.md).
