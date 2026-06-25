# GA-B4 — HTTP dry-run mis-previews complex graphs

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-B4 |
| **Section** | Grading Agent — Bugs |
| **Severity** | MINOR |
| **Bug size** | Medium |
| **Markets** | internal correctness |
| **Status (today)** | BUG (latent — endpoint currently unused by the client) |
| **Estimated effort** | XS (≤1d) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | — |
| **Unblocks** | — |

## 1. Problem Statement

`handlePostGraderAgentDryRun` (`POST …/grader-agent/dry-run`) compiles the graph with
`CompileWorkflowGraph` and then calls `svc.Score(scoreReq)` — a **single** LLM scoring call. It does not
use the graph engine (`ExecuteWorkflowDryRun`). For any graph beyond "submission → AI/grader → output",
this is wrong:

- Routers, Human Review Gates, Flag-for-Review sinks, Score Aggregators, Originality, Reference, and
  multi-criterion fan-out are **ignored** — the preview reflects only the single compiled grade source.
- For a Code Test Runner or Score Aggregator grade source, `CompileWorkflowGraph` returns an **empty**
  `ScoreRequest` (no submission text/prompt), so `svc.Score` fails with "submission text is empty" or
  produces a meaningless preview.

The client currently dodges this by only using the WebSocket dry run (`handleGraderAgentDryRunWS`, which
*does* run the engine), so the POST handler is effectively dead (`postGraderAgentDryRun` is unused in
`courses-api.ts`). But the divergence is a live trap: any future caller, test, or integration that hits
the POST endpoint gets a silently incorrect preview that disagrees with how the submission is actually
graded.

## 2. Goals

- Eliminate the divergence so there is a single source of truth for previews and live grading.
- Either delete the POST endpoint ([GA-S2](simplify-2-remove-dead-dry-run-and-rename-engine.md)) or make it delegate to the same engine as the WS path and the consumer.

## 3. Non-Goals

- Changing the WebSocket dry-run protocol or behavior (it is correct).
- Adding new preview features.

## 4. Personas & User Stories

- **As an engineer**, I want one preview engine, so that a dry run can never disagree with live grading.
- **As an integrator**, I want the POST endpoint (if it exists) to return the true preview, so that I am not misled.

## 5. Functional Requirements

- **FR-1.** The POST dry-run endpoint MUST either be removed, or reimplemented to call `ExecuteWorkflowDryRun` (the engine) and return the same preview shape as the WS path for the same graph + submission.
- **FR-2.** If retained, a graph whose grade source is a Code Test Runner or Score Aggregator MUST produce a correct preview (not an empty-`ScoreRequest` error).
- **FR-3.** A graph with routers/gates/flags/aggregators MUST preview those effects (held/flagged/branch) identically to the WS path.
- **FR-4.** Removal MUST also drop the unused client `postGraderAgentDryRun`.

## 6. Non-Functional Requirements

- **Reliability** — preview parity between POST (if kept), WS, and live grading, asserted by a shared test.
- **Security** — reduce surface if removed.
- **Backward compatibility** — no client impact (endpoint unused today).
- **Maintainability** — single engine path.

## 7. Acceptance Criteria

- **AC-1.** *Given* a complex graph (router + gate + aggregator), *when* previewed via POST (if kept) and WS, *then* the previews match.
- **AC-2.** *Given* a Code-Test-Runner grade source, *when* previewed via POST (if kept), *then* it returns a correct preview, not "submission text is empty".
- **AC-3.** *Given* removal, *when* grepping, *then* `handlePostGraderAgentDryRun` and `postGraderAgentDryRun` are gone and the route is unregistered.

## 8. Data Model

- None.

## 9. API Surface

- Remove `POST …/grader-agent/dry-run` (preferred), or change its implementation to delegate to the engine. Update OpenAPI either way.

## 10. UI / UX

- None (endpoint not surfaced).

## 11. AI / ML Considerations

- Ensures previews and live grading use one engine, so token/cost and outputs are consistent.

## 12. Integration Points

- `server/internal/httpserver/grading_agent_http.go` (`handlePostGraderAgentDryRun`, `dryRunGraderAgentBody`).
- `server/internal/httpserver/grading_agent_dry_run_ws.go` (the correct engine call to mirror).
- `server/internal/httpserver/courses_routes.go` (route).
- `clients/web/src/lib/courses-api.ts` (`postGraderAgentDryRun`).

## 13. Dependencies & Sequencing

- This is the bug; [GA-S2](simplify-2-remove-dead-dry-run-and-rename-engine.md) is the matching cleanup. Resolve together (removal satisfies both).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A hidden caller relies on the POST endpoint | L | M | API analytics + grep before removal; otherwise delegate to engine instead of deleting |

## 15. Rollout Plan

- Sequence: confirm no callers → remove (or delegate) → ship.
- Rollback: revert (pure code change).

## 16. Test Plan

- **Integration** — if kept, POST and WS previews match for representative graphs; if removed, route 404s and build is green.
- **Static** — no dangling references.

## 17. Documentation & Training

- OpenAPI updated; note that dry run is WS-only (or engine-backed POST).

## 18. Open Questions

1. Keep a non-WS preview endpoint (engine-backed) for non-WebSocket clients, or remove outright?

## 19. References

- `server/internal/httpserver/grading_agent_http.go` (`handlePostGraderAgentDryRun` → `svc.Score`).
- `server/internal/service/gradingagent/workflow.go` (`CompileWorkflowGraph` returns empty `ScoreRequest` for code-test/aggregator sources).
- `server/internal/httpserver/grading_agent_dry_run_ws.go` (`ExecuteWorkflowDryRun`).
- Related: [GA-S2](simplify-2-remove-dead-dry-run-and-rename-engine.md).
