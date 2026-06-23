# 19.17.4 — Score Aggregator Node (Combine Multiple Scores)

> Implementation plan. Source: [docs/MISSING_FEATURES.md](../../MISSING_FEATURES.md) §19. Extends the grading-agent canvas ([19.17](../auto-grader-agent.md)). See the [node catalog](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | 19.17.4 |
| **Section** | AI-Specific Capabilities → Grading Agent Canvas |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | AI / Grading team |
| **Depends on** | 19.16, 19.17, [19.17.3 Criterion Grader](node-criterion-grader.md) |
| **Unblocks** | Fan-out grading, blended human/AI/code scores |
| **Owns shared change** | **Fan-in on processing inputs** (see [catalog](README.md)) |

---

## 1. Problem Statement

The canvas is strictly one-grade-in, one-grade-out: the Student Grade `grade` slot accepts exactly one inbound edge. The moment an instructor splits grading across multiple [Criterion Graders](node-criterion-grader.md), blends an AI judgement with a [Code Test Runner](node-code-test-runner.md) result, or wants to apply an originality penalty, there is no node that can **combine** several scores into one total. This node consumes many `grade` inputs and folds them into a single grade via a chosen strategy — weighted sum, average, min, max, or rubric merge — and is the piece that makes fan-out actually usable.

## 2. Goals

- Accept **multiple** `grade` inputs on a single fan-in handle.
- Combine them deterministically (no LLM) by: `sum`, `weightedSum`, `average`, `min`, `max`, or `rubricMerge` (union of per-criterion scores).
- Produce one `grade` output (total + merged rubric map + combined confidence) and optionally a merged `comments` output.
- Clamp the result to `maxPoints` and rubric bounds; define behavior for missing/errored inputs.

## 3. Non-Goals

- LLM reasoning over scores (use an AI node for that).
- Cross-submission analytics or curving (that is gradebook-level, e.g. [3.17 grade curving](../../completed/03-submissions-grading-integrity/3.17-grade-curving-scaling.md)).
- Defining how scores are produced — only how they combine.

## 4. Personas & User Stories

- **As an instructor using per-criterion graders**, I want the criterion scores summed into the total automatically.
- **As a CS instructor**, I want `0.7 × autograder + 0.3 × AI code-style score` as the final grade.
- **As an integrity-conscious instructor**, I want to subtract an originality penalty from the AI score (min with a cap).
- **As a TA**, I want the combined confidence to be the *minimum* of inputs so a single low-confidence criterion drags the item into review.

## 5. Functional Requirements

- **FR-1.** The palette MUST offer a **Score Aggregator** node under the Processing group.
- **FR-2.** The node MUST accept **many** inbound edges on a `grade` input handle (the first node to relax the single-edge rule for a non-output handle).
- **FR-3.** The inspector MUST offer a combine mode (`sum` | `weightedSum` | `average` | `min` | `max` | `rubricMerge`) and, for `weightedSum`, a per-source weight editor keyed by upstream node.
- **FR-4.** The node MUST emit one `grade` output with: combined `TotalPoints` (clamped to `[0, maxPoints]`), a merged `RubricScores` map (for `rubricMerge`/criterion fan-in), and a combined `Confidence`.
- **FR-5.** Combined confidence MUST be configurable: `min` (default), `mean`, or `weighted`.
- **FR-6.** Missing-input policy MUST be configurable: `treatAsZero`, `skipAndRenormalize`, or `failItem`.
- **FR-7.** The node MUST optionally emit `comments` by concatenating upstream comments with source labels (toggle).
- **FR-8.** Validation MUST require ≥ 1 `grade` input and reject non-grade sources; `rubricMerge` MUST reject overlapping criterion IDs from different sources (each criterion scored once).

## 6. Non-Functional Requirements

- **Performance** — Pure arithmetic, sub-millisecond; no model call, no cost.
- **Security** — No external calls; operates on in-memory `slotValue`s.
- **Privacy & Compliance** — No new data leaves the system.
- **Accessibility** — Mode/weight controls labelled; weight inputs numeric-validated.
- **Scalability** — Bounded by graph caps.
- **Reliability** — Deterministic; well-defined behavior for empty/partial inputs (FR-6).
- **Observability** — Dry-run log lists each input, weight, and the resulting total; `grader_agent_aggregator_total{mode}`.
- **Maintainability** — Combine logic is a small pure module with table-driven tests.
- **Internationalization** — Concatenated comment separators localized.
- **Backward compatibility** — Additive; the single-edge rule stays for the Student Grade output node.

## 7. Acceptance Criteria

- **AC-1.** *Given* three Criterion Graders (scores 4, 3, 5) wired into a `sum` aggregator, *When* dry-run executes, *Then* the total is 12 (clamped to maxPoints) with a merged 3-criterion rubric map.
- **AC-2.** *Given* a `weightedSum` with weights {auto: 0.7, ai: 0.3} over scores {auto: 80, ai: 100}, *When* executed, *Then* the total is 86.
- **AC-3.** *Given* `min` confidence and inputs with confidences {0.9, 0.4}, *When* executed, *Then* combined confidence is 0.4.
- **AC-4.** *Given* one input errored and policy `skipAndRenormalize`, *When* executed, *Then* the remaining inputs are renormalized and the item still produces a grade.
- **AC-5.** *Given* a `rubricMerge` where two inputs score the same criterion, *When* the graph validates, *Then* a node-level conflict issue blocks the run.
- **AC-6.** *Given* a non-grade source wired to the aggregator, *When* dropped, *Then* the connection is rejected.

## 8. Data Model

No new tables. Node `data`:

```jsonc
{
  "mode": "sum" | "weightedSum" | "average" | "min" | "max" | "rubricMerge",
  "weights": { "<sourceNodeId>": 1.0 },     // weightedSum only
  "confidence": "min" | "mean" | "weighted", // default "min"
  "onMissing": "treatAsZero" | "skipAndRenormalize" | "failItem",
  "mergeComments": true
}
```

The fan-in changes only validation/execution, not persistence; the final grade is the existing result row.

## 9. API Surface

- No new routes.
- Dry-run emits an aggregation log line per input and the combined result; the assembled preview flows to the output node unchanged.
- OpenAPI: node `data` schema; document that the aggregator `grade` input accepts multiple edges.

## 10. UI / UX

- **Palette** — "Score Aggregator" in `groupProcessing` (emerald, grade-family styling).
- **Node body** — Title; a single `grade` input slot rendered to accept multiple connections (no per-edge slots); one `grade` output and optional `comments` output.
- **Inspector** — Mode selector; weight table (rows = wired sources by display label) for `weightedSum`; confidence-combine selector; missing-input policy; merge-comments toggle; live "preview math" line after a dry run.
- **States** — No inputs (hint), conflicting criteria (error), weights not summing to 1 in `weightedSum` (warn, auto-normalize option).
- **Mobile** — Weight table scrolls.
- **Copy & i18n** — `gradingAgent.canvas.palette.aggregator`, `gradingAgent.canvas.nodes.aggregator.*`, `gradingAgent.canvas.inspector.aggregator*`.

## 11. AI / ML Considerations

None — deterministic arithmetic. Its value to AI quality is indirect: enabling [Criterion Grader](node-criterion-grader.md) fan-out (better rationales) and confidence floors that route low-confidence items to the [Human Review Gate](node-human-review-gate.md).

## 12. Integration Points

- **Client** — `types.ts` (`PaletteNodeType` += `'scoreAggregator'`, new handle reuse of `grade`), `node-palette.tsx`, `workflow-nodes.tsx` (`ScoreAggregatorNode` with multi-connection target handle), `workflow-node-types.ts`, `validation.ts` (allow multiple inbound `grade` edges to aggregator; reject elsewhere), `inspector-panel.tsx`.
- **Server** — `workflow.go` (`NodeTypeScoreAggregator`, edge typing: `grade` source from grader/criterionGrader/ai/codeRunner; multi-edge allowed for aggregator only), `workflow_execute.go` (new case folds incoming `slotValue.grade`s per mode; extend output assembly to accept the aggregator's `grade`), a new pure `aggregate.go` combine module.
- **Rubric clamp** — `assignmentrubric.ValidateRubricScoresForGrade` on the merged map.

## 13. Dependencies & Sequencing

- **After**: 19.16, 19.17; build with [Criterion Grader](node-criterion-grader.md).
- **Before**: makes [Code Test Runner](node-code-test-runner.md) blending and [Originality Check](node-originality-check.md) penalties usable.
- **Shared infra**: none new — but **owns the fan-in relaxation** other nodes rely on.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Relaxing single-edge rule leaks into output node | M | M | Multi-edge allowed *only* for the aggregator's `grade` input; output node keeps single-edge; covered by validation tests |
| Weights misconfigured (don't sum to 1) | M | M | Warn + one-click normalize; `weightedSum` documented |
| Double-counting a criterion across inputs | M | M | `rubricMerge` rejects overlapping criterion IDs (FR-8) |
| Silent zero from `treatAsZero` masks an upstream failure | M | M | Dry-run log flags substituted zeros; default policy is documented per use case |

## 15. Rollout Plan

- Behind `grader_agent_enabled`; ship with Criterion Grader.
- Sequencing: validation fan-in relaxation → combine module + execution → inspector → i18n.
- Dogfood: rubric fan-out and CS blended grading.
- Rollback: remove palette item behind flag; fan-in relaxation is inert without the node.

## 16. Test Plan

- **Unit** — Combine module: each mode, weights, confidence modes, missing-input policies, clamping, rubric-merge conflict detection.
- **Integration** — Multi-edge validation (allowed for aggregator, rejected for output); dry-run fold; renormalize on error.
- **E2E** — Criterion fan-out → aggregator → Student Grade; verify total and breakdown.
- **Accessibility** — axe; keyboard weight entry.

## 17. Documentation & Training

- Help center: "Combining scores (sum, weighted, min/max, rubric merge)."
- Instructor guide: blending AI + autograder; using min-confidence to trigger review.
- API reference: node `data` schema + fan-in note.

## 18. Open Questions

1. Should weights be percentages (auto-normalized) or raw multipliers? (Plan: raw multipliers with a normalize helper.)
2. Should the aggregator expose its own `confidence` as a routable `score` output for the [Conditional Router](node-conditional-router.md)? (Leaning yes.)

## 19. References

- [workflow-nodes.tsx](../../../clients/web/src/components/annotation/grader-agent/workflow-nodes.tsx), [validation.ts](../../../clients/web/src/components/annotation/grader-agent/validation.ts).
- Server: [workflow.go](../../../server/internal/service/gradingagent/workflow.go), [workflow_execute.go](../../../server/internal/service/gradingagent/workflow_execute.go), [scoring.go](../../../server/internal/service/gradingagent/scoring.go), [assignmentrubric](../../../server/internal/models/assignmentrubric).
- Related: [node catalog](README.md), [Criterion Grader](node-criterion-grader.md), [Code Test Runner](node-code-test-runner.md), [Conditional Router](node-conditional-router.md).
