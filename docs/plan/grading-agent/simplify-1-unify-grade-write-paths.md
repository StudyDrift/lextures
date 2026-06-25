# GA-S1 — Unify the three duplicated grade-write paths in the consumer

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-S1 |
| **Section** | Grading Agent — Over-complexity / Simplification |
| **Severity** | MAJOR |
| **Markets** | HE / K12 / SL (internal maintainability) |
| **Status (today)** | THIN |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | — |
| **Unblocks** | [GA-M3](missing-3-suggest-only-batch-and-bulk-review.md), [GA-M4](missing-4-confidence-auto-hold-threshold.md), [GA-M6](missing-6-cancel-running-batch.md), [GA-B2](bug-2-requeue-double-grading.md) |

## 1. Problem Statement

`HandleGradingAgentQueueMessage` (≈ 300 lines) grades one submission through **three overlapping
branches**:

1. **Path A** — `WorkflowRequiresGraphExecution` (router/flag/gate/aggregator present): runs
   `ExecuteWorkflowDryRun`, handles flagged/held/applied.
2. **Path B** — single grade source of type AI / Criterion Grader / Code Test Runner: runs
   `ExecuteWorkflowDryRun` *again* with a near-identical input block and applies — but ignores
   held/flagged (≈ 70 lines duplicated from Path A).
3. **Path C** — legacy fallthrough: builds a `ScoreRequest`, calls `Service.Score`, applies.

All three end in the same "resolve model → AI-gateway check → execute → build comment/points/rubric →
`UpsertCellWithFlags` → `InsertResult` → `IncrementRunProgress`" shape. The model-resolution,
gateway-evaluation, posting-derivation, and `gradecomment.Append` logic are copy-pasted three times.
Any new behavior (suggest-only mode, confidence floor, cancel check, idempotency) must be added in
three places, and Path B silently can't hold/flag because its branch predates that capability. This is
the single biggest maintainability liability in the agent.

## 2. Goals

- One execution path for live grading: always walk the graph via the shared engine, then apply the resulting preview.
- Delete Paths B and C as distinct branches; collapse to a single helper sequence.
- Make held/flagged handling uniform regardless of graph shape.
- Preserve current outcomes for every existing graph (AI-only, criterion, code-test, router, gate, flag, aggregator, legacy).

## 3. Non-Goals

- Changing scoring semantics or prompt construction.
- Removing the legacy `Service.Score` API used by other callers (handled in [GA-S2](simplify-2-remove-dead-dry-run-and-rename-engine.md)).
- Rewriting the queue/bus.

## 4. Personas & User Stories

- **As an engineer**, I want one apply path, so that adding suggest-only/cancel/idempotency is a one-place change.
- **As a maintainer**, I want held/flagged handled uniformly, so that a "simple" AI-only graph can still hold low-confidence items.
- **As a QA engineer**, I want one code path to test, so that coverage is tractable.

## 5. Functional Requirements

- **FR-1.** The consumer MUST execute every accepted graph through the single shared engine and obtain a uniform preview (points, comment, rubric, confidence, held, flagged, tokens, cost).
- **FR-2.** Held, flagged, skipped (already-graded), applied, and failed outcomes MUST be produced by one shared "persist preview" helper.
- **FR-3.** Model resolution, AI-gateway evaluation, posting derivation, and comment assembly MUST each exist once.
- **FR-4.** Behavior MUST be byte-for-byte equivalent for existing graphs (verified by golden tests across all node shapes).
- **FR-5.** The legacy non-graph path (config with no workflow graph) MUST still work via the synthesized default graph (`EffectiveWorkflowGraph`) routed through the same engine.

## 6. Non-Functional Requirements

- **Performance** — no extra model calls vs today (the engine already runs once); avoid the double-execute that Path B/A could imply.
- **Security** — gateway checks preserved and centralized.
- **Reliability** — single path makes [GA-B2](bug-2-requeue-double-grading.md) idempotency a one-place fix.
- **Maintainability** — target ≤ ~120 lines for `HandleGradingAgentQueueMessage` with extracted helpers.
- **Observability** — one structured "graded" log/metric with outcome label.
- **Backward compatibility** — outcomes unchanged; covered by golden tests.

## 7. Acceptance Criteria

- **AC-1.** *Given* each existing graph shape (AI-only, criterion, code-test, router, gate, flag, aggregator, legacy-no-graph), *when* graded before and after the refactor, *then* points/comment/rubric/status match.
- **AC-2.** *Given* an AI-only graph with a confidence floor (post-[GA-M4](missing-4-confidence-auto-hold-threshold.md)), *when* a low-confidence item is graded, *then* it is held — proving Path B's gap is closed.
- **AC-3.** *Given* a flagged graph, *when* graded, *then* the flagged result is recorded exactly as today.
- **AC-4.** *Given* the refactor, *when* measured, *then* no graph triggers two model executions.

## 8. Data Model

- None. Pure refactor.

## 9. API Surface

- None. Internal only.

## 10. UI / UX

- None.

## 11. AI / ML Considerations

- Ensure exactly one execution per submission; the consolidation removes the risk of Path A/B double execution.

## 12. Integration Points

- `server/internal/httpserver/grading_agent_queue.go` (the consolidation target).
- `server/internal/service/gradingagent/{workflow_execute,flag_sink,code_test_runner}.go` (engine + capability helpers).
- `server/internal/repos/gradingagent/repo.go` (`InsertResult`, `IncrementRunProgress`).

## 13. Dependencies & Sequencing

- Best done **before** [GA-M3](missing-3-suggest-only-batch-and-bulk-review.md), [GA-M4](missing-4-confidence-auto-hold-threshold.md), [GA-M6](missing-6-cancel-running-batch.md), and [GA-B2](bug-2-requeue-double-grading.md) so each lands once.
- Pairs with [GA-S2](simplify-2-remove-dead-dry-run-and-rename-engine.md).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Subtle behavior drift for an edge graph | M | H | Golden tests per node shape captured **before** refactor |
| Path B "skip model on code-test-only" nuance lost | M | M | Preserve the no-LLM detection (`WorkflowUsesLLM`) in the unified path |
| Hidden coupling to legacy `Service.Score` outputs | L | M | Keep `Score` for [GA-S2]; route default-graph through engine |

## 15. Rollout Plan

- Flag: `graderAgentUnifiedConsumer` to switch old↔new path during bake.
- Sequence: capture golden tests → implement unified path behind flag → shadow-compare in staging → flip → delete old branches.
- Rollback: flag back to legacy branches until deletion.

## 16. Test Plan

- **Unit** — extracted helpers (model resolve, gateway, persist preview, posting).
- **Golden/Integration** — every node-shape graph graded pre/post produces identical persisted rows.
- **Regression** — already-graded skip (ungraded scope), failed item, held, flagged.
- **Performance** — assert single execution per submission.

## 17. Documentation & Training

- Internal runbook: "How a queued submission is graded" updated to one path.
- Architecture note in `../agent-grader/README.md` cross-link.

## 18. Open Questions

1. Keep a flag long-term or delete legacy branches immediately after bake?
2. Should the default-graph synthesis be removed in favor of always persisting an explicit graph on accept?

## 19. References

- `server/internal/httpserver/grading_agent_queue.go` (`HandleGradingAgentQueueMessage`, Paths A/B/C).
- `server/internal/service/gradingagent/flag_sink.go` (`WorkflowRequiresGraphExecution`).
- `server/internal/service/gradingagent/code_test_runner.go` (`WorkflowUsesLLM`).
- Related: [GA-S2](simplify-2-remove-dead-dry-run-and-rename-engine.md), [GA-B2](bug-2-requeue-double-grading.md).
