# GA-S2 — Remove the dead HTTP dry-run path; rename the execution engine

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](../../plan/grading-agent/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-S2 |
| **Section** | Grading Agent — Over-complexity / Simplification |
| **Severity** | MINOR |
| **Markets** | internal maintainability |
| **Status (today)** | COMPLETE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | — |
| **Unblocks** | clarity for all engine work |

## 1. Problem Statement

There are **two** dry-run code paths that do different things:

- `handlePostGraderAgentDryRun` (`POST …/grader-agent/dry-run`, ≈ 185 lines) compiles the graph to a
  single `ScoreRequest` and calls `Service.Score` — it cannot execute routers, gates, aggregators,
  flag sinks, code-test, originality, or reference nodes. The client never calls it
  (`postGraderAgentDryRun` is dead in `courses-api.ts`).
- `handleGraderAgentDryRunWS` (WebSocket) calls `ExecuteWorkflowDryRun`, the real engine, and is the
  only dry run the canvas uses.

So the POST endpoint is **dead and divergent** — it produces wrong previews for any non-trivial graph
(see [GA-B4](bug-4-post-dry-run-mispreviews-graphs.md)). Separately, the engine is named
`ExecuteWorkflowDryRun` but is the **live grading engine** too (the consumer calls it for real grades).
The "DryRun" name actively misleads readers into thinking the consumer only previews.

## 2. Goals

- Delete the dead POST dry-run handler, route, and client function (or make it delegate to the engine if a non-WS path is genuinely needed).
- Rename `ExecuteWorkflowDryRun` → `ExecuteWorkflow` (and `DryRunExecutionInput` → `ExecutionInput`, `DryRunEvent` → `ExecutionEvent`) to reflect dual use, keeping the dry-run *flag* explicit.
- Reduce the engine's API surface and the reader's confusion.

## 3. Non-Goals

- Changing what the WS dry run does or its event protocol on the wire (only Go-side type names).
- Removing `Service.Score` if still used by other features (audit usages first).

## 4. Personas & User Stories

- **As an engineer**, I want one dry-run path, so that I do not accidentally fix a bug in the dead one.
- **As a new contributor**, I want the engine name to tell me it runs live grading, so that I do not assume it is preview-only.

## 5. Functional Requirements

- **FR-1.** The dead `POST …/grader-agent/dry-run` route and `handlePostGraderAgentDryRun` MUST be removed, **or** reimplemented to call the shared engine (`ExecuteWorkflow`) so its output matches the WS path.
- **FR-2.** The unused client `postGraderAgentDryRun` MUST be removed from `courses-api.ts`.
- **FR-3.** The engine and its public types MUST be renamed to drop the misleading "DryRun" prefix where they cover live execution; the dry-run vs live distinction remains an explicit input field.
- **FR-4.** `Service.Score` usages MUST be audited; if only the dead handler used it, it MAY be removed or retained for the unified consumer ([GA-S1](simplify-1-unify-grade-write-paths.md)).
- **FR-5.** No behavior change to the WS dry run or live grading.

## 6. Non-Functional Requirements

- **Maintainability** — net code reduction; one dry-run path.
- **Backward compatibility** — wire protocol for the WS dry run unchanged; only internal Go symbols and a dead HTTP route change.
- **Observability** — no change.
- **Security** — removing an unused authenticated route reduces surface.

## 7. Acceptance Criteria

- **AC-1.** *Given* the change, *when* the client dry-runs, *then* it still works via WS with identical behavior.
- **AC-2.** *Given* a grep for `postGraderAgentDryRun` / `handlePostGraderAgentDryRun`, *then* there are no remaining references (or the handler now delegates to the engine).
- **AC-3.** *Given* the rename, *when* the suite runs, *then* it compiles and passes with the new symbol names.
- **AC-4.** *Given* live grading, *when* a batch runs, *then* outcomes are unchanged.

## 8. Data Model

- None.

## 9. API Surface

- Remove `POST …/assignments/{item_id}/grader-agent/dry-run` (keep the `…/dry-run/ws` route). Update OpenAPI.

## 10. UI / UX

- None (the dead POST path is not surfaced).

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- `server/internal/httpserver/grading_agent_http.go` (`handlePostGraderAgentDryRun`, `dryRunGraderAgentBody`).
- `server/internal/httpserver/courses_routes.go` (route).
- `server/internal/service/gradingagent/workflow_execute.go` (renames).
- `server/internal/background/grading_agent_consumer.go`, `grading_agent_queue.go`, `grading_agent_dry_run_ws.go` (call sites).
- `clients/web/src/lib/courses-api.ts` (`postGraderAgentDryRun`).

## 13. Dependencies & Sequencing

- Do alongside or just after [GA-S1](simplify-1-unify-grade-write-paths.md); together they leave one execution engine and one apply path.
- Fixes the divergence reported in [GA-B4](bug-4-post-dry-run-mispreviews-graphs.md).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A non-web caller depends on the POST route | L | M | Grep + API analytics before removal; 1 release deprecation if uncertain |
| Rename churn touches many files | M | L | Mechanical rename in one commit; rely on the compiler |

## 15. Rollout Plan

- Sequence: confirm no callers → remove route + client fn → rename symbols → ship.
- Rollback: revert; pure code change.

## 16. Test Plan

- **Unit/Integration** — WS dry run unchanged; live grading unchanged.
- **Static** — grep proves no dangling references; build green.

## 17. Documentation & Training

- OpenAPI updated; internal note that `ExecuteWorkflow` is the single engine (dry-run is a flag).

## 18. Open Questions

1. Keep a non-WS dry-run endpoint (delegating to the engine) for clients that cannot use WebSockets, or drop entirely?
2. Is `Service.Score` still wanted as the LLM call primitive inside the unified consumer?

## 19. References

- `server/internal/httpserver/grading_agent_http.go` (`handlePostGraderAgentDryRun`).
- `server/internal/httpserver/grading_agent_dry_run_ws.go` (`ExecuteWorkflow` call).
- `server/internal/service/gradingagent/workflow_execute.go` (engine + `Execution*` types).
- `clients/web/src/lib/courses-api.ts` (`streamGraderAgentDryRun`).
- Related: [GA-S1](simplify-1-unify-grade-write-paths.md), [GA-B4](bug-4-post-dry-run-mispreviews-graphs.md).

## 20. Implementation notes

- The dead `POST …/grader-agent/dry-run` route, handler, and `postGraderAgentDryRun` client function were already removed prior to this change (via GA-S1 / GA-B4 work). Confirmed by grep: no live code references remain.
- Renamed the shared workflow engine: `ExecuteWorkflowDryRun` → `ExecuteWorkflow`, `DryRunExecutionInput` → `ExecutionInput`, `DryRunEvent` → `ExecutionEvent` across `workflow_execute.go` and all call sites (WS dry run, queue consumer execute path, node helpers, tests).
- `Service.Score` is retained for legacy configs without a compilable workflow graph (`executeGradingAgentLegacyScore` in `grading_agent_execute.go`).
- WS dry-run wire protocol unchanged; only internal Go symbol names updated.
- OpenAPI never documented the removed POST route; no spec change required.
