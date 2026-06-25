# GA-M7 — Pre-run cost & scope estimate + run cost summary

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-M7 |
| **Section** | Grading Agent — Missing Features |
| **Severity** | MAJOR |
| **Markets** | HE / SL |
| **Status (today)** | THIN |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | [GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md) |
| **Unblocks** | confident large-class runs |

## 1. Problem Statement

The system records `prompt_tokens`, `completion_tokens`, and `cost_usd` per result row, but nowhere does
the instructor see **how much a run will cost before starting it**, or **the total cost after it finishes**.
The run popover shows only a scope radio and a progress count. For a department footing the AI bill, an
unbounded "grade all 300 submissions" button with no estimate and no cap is a hard blocker to approval.
A single dry run already produces real token counts, so a per-submission estimate is readily available.

## 2. Goals

- Before running: show "≈ N submissions, ≈ $X" derived from a sample dry run and the chosen scope/model.
- After running: show the run's actual total cost and token usage in the run summary and history.
- Optional per-run or per-agent budget cap that stops the run (ties to [GA-M6](missing-6-cancel-running-batch.md)).

## 3. Non-Goals

- Department-wide budgeting/quotas (belongs to the AI gateway / billing surfaces).
- Exact pricing guarantees — estimates are clearly labeled approximate.

## 4. Personas & User Stories

- **As an instructor**, I want a cost estimate before I run, so that I do not surprise my department.
- **As an admin**, I want a per-run budget cap, so that a misconfigured agent cannot spend without bound.
- **As an instructor**, I want the actual run cost afterward, so that I can report spend.

## 5. Functional Requirements

- **FR-1.** The run popover MUST display the resolved submission count and an estimated cost range for the selected scope/filter and model before the run is created.
- **FR-2.** The estimate SHOULD reuse the most recent dry-run token usage for the agent (per-submission tokens × count × model price), and clearly label it approximate.
- **FR-3.** After a run, the system MUST expose the aggregate `cost_usd`, `prompt_tokens`, `completion_tokens` for the run.
- **FR-4.** An optional budget cap (per run) MUST stop further grading (skip remaining as `skipped`, reason "budget exceeded") when projected/observed spend exceeds the cap.
- **FR-5.** Cost numbers MUST reconcile with the AI-gateway usage already recorded (`recordAIUsage`).

## 6. Non-Functional Requirements

- **Performance** — estimate is a cheap computation; the optional sample dry run is one model call, run on demand.
- **Security** — cost surfaced only to users who can run the agent.
- **Privacy & Compliance** — no submission content in cost telemetry.
- **Accessibility** — estimate text is associated with the run control via `aria-describedby`.
- **Reliability** — budget enforcement uses the same per-message guard as cancel ([GA-M6](missing-6-cancel-running-batch.md)).
- **Observability** — run-level cost metric; estimate-vs-actual delta metric.
- **Internationalization** — currency/number formatting via locale; `gradingAgent.run.cost.*`.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a prior dry run and scope "ungraded (24)", *when* I open the run popover, *then* I see "≈ 24 submissions, ≈ $X–$Y".
- **AC-2.** *Given* no prior dry run, *when* I open the popover, *then* I see the count and a prompt to dry-run for a cost estimate.
- **AC-3.** *Given* a finished run, *when* I view its summary/history, *then* the actual total cost and tokens are shown.
- **AC-4.** *Given* a per-run budget cap, *when* observed spend crosses it, *then* remaining items are skipped with reason "budget exceeded" and the run terminates.
- **AC-5.** *Given* the run cost, *when* compared to AI-gateway usage records, *then* they reconcile.

## 8. Data Model

- Add run-level aggregates (or compute on read): `cost_usd`, `prompt_tokens`, `completion_tokens` summed from results; optional materialized columns on `grading_agent_runs` for fast history.
- Optional `budget_usd NUMERIC NULL` on `grading_agent_runs`.
- Migration: `server/migrations/NNN_grading_agent_run_cost.sql` (if materialized).

## 9. API Surface

- `GET …/grader-agent/runs/{run_id}` includes aggregate cost/tokens (sum from results if not materialized).
- `POST …/grader-agent/runs` body gains optional `budgetUsd`.
- Optional `GET …/grader-agent/estimate?scope=…&filter=…` returning count + estimated cost (or compute client-side from the last dry run + count endpoint).

## 10. UI / UX

- Run popover: a cost line under the scope picker; "Dry run for an estimate" CTA when none exists.
- Run summary + history ([GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md)): actual cost and tokens.
- Optional budget cap input with helper text.
- Copy/i18n under `gradingAgent.run.cost.*`.

## 11. AI / ML Considerations

- Estimate accuracy depends on token variance across submissions; present a range, not a point.
- Use the model's price (per the provider/BYOK config) to convert tokens → USD.

## 12. Integration Points

- `server/internal/repos/gradingagent/repo.go` (run cost aggregation; `budget_usd`).
- `server/internal/httpserver/grading_agent_{http,queue}.go` (estimate endpoint, budget guard, reconcile with `recordAIUsage`).
- Pricing/model metadata from the AI provider settings (per-tenant BYOK, plan 16.7).
- `clients/web/src/components/annotation/grader-agent/run-agent-popover.tsx`, `use-grader-agent-workflow.ts`.

## 13. Dependencies & Sequencing

- Needs [GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md) for the run-history home and [GA-M6](missing-6-cancel-running-batch.md) for the budget stop mechanism.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Estimate badly off → eroded trust | M | M | Show a range; label approximate; refine from rolling per-agent averages |
| Model price data missing for some models | M | M | Fall back to tokens-only display when price unknown |

## 15. Rollout Plan

- Flag: `graderAgentCostEstimate`.
- Sequence: run cost aggregation → estimate display → budget cap → flip flag.
- Pilot: a cost-sensitive department.
- Rollback: hide cost UI; data still recorded.

## 16. Test Plan

- **Unit** — estimate math; budget-stop boundary; reconcile with usage.
- **Integration** — run aggregates match summed results.
- **E2E** — dry run → estimate shown → run → actual cost shown; budget cap stops a run.

## 17. Documentation & Training

- Help-center: "Understanding grading-agent costs and budgets."
- Admin doc: setting per-run budgets.

## 18. Open Questions

1. Per-agent rolling average vs last dry run as the estimate basis?
2. Should the budget cap live on the agent config (default) as well as per run?

## 19. References

- `server/internal/repos/gradingagent/repo.go` (`ResultRow.CostUSD/PromptTokens/CompletionTokens`).
- `server/internal/httpserver/grading_agent_http.go` (`recordAIUsage`, `openrouterUsageFromScore`).
- `clients/web/src/components/annotation/grader-agent/run-agent-popover.tsx`.
- Related: [GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md), [GA-M6](missing-6-cancel-running-batch.md); per-tenant BYOK (plan 16.7).
