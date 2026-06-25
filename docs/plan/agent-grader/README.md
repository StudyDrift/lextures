# Grader Agent — Node Catalog & Expansion Plans

> Plans for new nodes on the **grading agent workflow canvas** (plan 19.17), which sits inside the
> [Auto-Grader Agent](../../completed/auto-grader-agent.md) (19.16). Each node is its own plan file and follows
> [`_TEMPLATE.md`](../_TEMPLATE.md).

## What exists today

The canvas is a [React Flow](https://reactflow.dev) directed-acyclic graph authored in SpeedGrader.
The palette has grown well past the original four nodes; as catalog plans below ship they move to
[`docs/completed/agent-grader/`](../../completed/agent-grader/). Node kinds that exist **today**, by role:

| Role | Node | Outputs / Inputs | Status |
|---|---|---|---|
| Input | **Student Submission** | → `submission` | shipped (original) |
| Input | **Activity** | → `content`, `rubric` | shipped (original) |
| Processing | **AI** | `input` → `output` (rubric- or score-format auto-detected) | shipped (original) |
| Processing | **Criterion Grader** | `content`/`rubric` → `grade`, `comments` | shipped ([19.17.3](../../completed/agent-grader/node-criterion-grader.md)) |
| Processing | **Code Test Runner** | `submission` → `score`, `report` | shipped ([19.17.7](../../completed/agent-grader/node-code-test-runner.md)) |
| Processing/Control | **Conditional Router** | input → `then` / `else` | shipped ([19.17.5](../../completed/agent-grader/node-conditional-router.md)) |
| Control/HITL | **Human Review Gate** | `grade` → `grade` (held) | shipped ([19.17.8](../../completed/agent-grader/node-human-review-gate.md)) |
| Output | **Student Grade** | `grade`, `comments` → (writes provisional grade) | shipped (default sink, auto-added) |
| Output | **Flag for Review** | `flag`/`reason` → review queue | shipped ([19.17.9](../../completed/agent-grader/node-flag-for-review.md)) |

(A legacy **Grader** node type still exists in the schema but is no longer in the palette; the AI node
superseded it. The Criterion Grader revives its scoring semantics as a focused, per-criterion node.)

### How the system fits together (read before editing any node plan)

A node is wired into **five** layers; every plan in this folder specifies all five:

1. **Types & handles** — [`types.ts`](../../../clients/web/src/components/annotation/grader-agent/types.ts)
   declares `GraderNodeType`, `PaletteNodeType`, and the `HANDLE_*` constants; the Go mirror is
   [`workflow.go`](../../../server/internal/service/gradingagent/workflow.go) (`NodeType*`, `Handle*`).
2. **Palette** — [`node-palette.tsx`](../../../clients/web/src/components/annotation/grader-agent/node-palette.tsx)
   exposes draggable items in `groupInput` / `groupProcessing` (a `groupOutput` group is added by the
   output-node plans).
3. **Edge validation** — client [`validation.ts`](../../../clients/web/src/components/annotation/grader-agent/validation.ts)
   (`connectionIsValid`, `validateWorkflowGraph`) and server `validateEdgeTypes` in
   [`workflow.go`](../../../server/internal/service/gradingagent/workflow.go) **must agree**. Both
   enforce typed handles, ≤ 50 nodes / ≤ 100 edges, acyclicity, exactly one Student Grade node, and a
   connected grade slot.
4. **Execution** — the topological walker
   [`workflow_execute.go`](../../../server/internal/service/gradingagent/workflow_execute.go)
   (`ExecuteWorkflowDryRun`) produces a `slotValue{ text, grade, rubric, score }` per `nodeID:handle`. Live
   batch runs use the same engine via
   [`grading_agent_queue.go`](../../../server/internal/httpserver/grading_agent_queue.go)
   (`executeGradingAgentPreview` → `persistGradingAgentPreview`; see
   [GA-S1](../../completed/grading-agent/simplify-1-unify-grade-write-paths.md)).
5. **Inspector & i18n** — the right-hand editor
   [`inspector-panel.tsx`](../../../clients/web/src/components/annotation/grader-agent/inspector-panel.tsx)
   renders per-node settings; all strings live under `gradingAgent.canvas.*` in
   [`common.json`](../../../clients/web/public/locales/en/common.json) (en/es/fr).

Prompts on AI and grader-style nodes support `$NodeName.Property` variables resolved from wired inputs
([`workflow-prompt-variable.ts`](../../../clients/web/src/components/annotation/grader-agent/workflow-prompt-variable.ts)).
Student submission text is always treated as **untrusted data**, never instructions
([`ai-output-system-prompt.ts`](../../../clients/web/src/components/annotation/grader-agent/ai-output-system-prompt.ts)).

## New nodes proposed in this folder

Ordered by recommended build sequence. Severity reflects how often instructors/graders hit the gap.

| # | Plan | Role | Severity | One-liner |
|---|---|---|---|---|
| 19.17.1 | [Rubric](node-rubric.md) | Input | MAJOR | Standalone, reusable rubric source decoupled from a single Activity. |
| 19.17.2 | [Reference Material](node-reference-material.md) | Input | MAJOR | Model answer / answer key / source texts as **trusted** grounding. |
| 19.17.3 | [Criterion Grader](../../completed/agent-grader/node-criterion-grader.md) | Processing | MAJOR | Grade one rubric criterion in isolation; enables fan-out. |
| 19.17.4 | [Score Aggregator](node-score-aggregator.md) | Processing | MAJOR | Combine multiple scores (weighted sum / avg / min / max / rubric merge). |
| 19.17.5 | [Conditional Router](../../completed/agent-grader/node-conditional-router.md) | Processing/Control | MAJOR | Branch on rules (empty, late, threshold, low confidence) — no LLM. |
| 19.17.6 | [Originality Check](../../completed/agent-grader/node-originality-check.md) | Processing | MAJOR | Surface similarity / AI-likelihood signals from the originality service. |
| 19.17.7 | [Code Test Runner](../../completed/agent-grader/node-code-test-runner.md) | Processing | MAJOR (CS) | Autograde code submissions against test cases in the sandbox. |
| 19.17.8 | [Human Review Gate](../../completed/agent-grader/node-human-review-gate.md) | Control/HITL | MAJOR | Hold low-confidence items for human approval before writing. |
| 19.17.9 | [Flag for Review](../../completed/agent-grader/node-flag-for-review.md) | Output | MINOR | Alternate sink that routes an item to a review queue instead of a grade. |

### Shared architectural evolutions these plans introduce

Three changes are needed by more than one node; each plan references this list and the first plan to
land each change owns it:

- **New handle/slot value kinds** — `score` (number), `report`/`reason` (text), `flag` (boolean), and a
  `reference` content variant. Extends `slotValue` in
  [`workflow_execute.go`](../../../server/internal/service/gradingagent/workflow_execute.go) and the
  `HANDLE_*` sets. *(✅ Already shipped: the `score`/`report`/`reason`/`flag` handles and `slotValue.score`
  landed with the criterion-grader, conditional-router, code-test-runner, and flag-for-review nodes.
  Still to add: the `reference` content variant, owned by [Reference Material](node-reference-material.md).)*
- **Fan-in on processing inputs** — today every input handle accepts exactly one inbound edge; aggregation
  requires a node input that accepts **many**. Still open. *(Owned by [Score Aggregator](node-score-aggregator.md).)*
- **Branching / optional paths** — the walker runs every node; conditional routing and multiple terminal
  sinks require "a branch may produce no value" semantics and a validity rule of *"every executable path
  reaches a terminal that satisfies the grade slot or a Flag-for-Review sink."* *(✅ Already shipped with [Conditional Router](../../completed/agent-grader/node-conditional-router.md); consumed by [Flag for Review](../../completed/agent-grader/node-flag-for-review.md).)*

## Conventions for these plans

- File naming: `node-{kebab-slug}.md`; Feature IDs `19.17.N`.
- Every plan fills all 19 template sections and names the concrete files in each of the five layers above.
- Cross-link related node plans with relative links.
