# 19.17.5 — Conditional Router Node (Rule-Based Branching)

> Implementation plan. Source: [docs/MISSING_FEATURES.md](../../MISSING_FEATURES.md) §19. Extends the grading-agent canvas ([19.17](../../plan/auto-grader-agent.md)). See the [node catalog](../../plan/agent-grader/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | 19.17.5 |
| **Section** | AI-Specific Capabilities → Grading Agent Canvas |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | AI / Grading team |
| **Depends on** | 19.16, 19.17 |
| **Unblocks** | [19.17.8 Human Review Gate](../../plan/agent-grader/node-human-review-gate.md), [19.17.9 Flag for Review](../../plan/agent-grader/node-flag-for-review.md) |
| **Owns shared change** | **Branching / optional paths** (see [catalog](../../plan/agent-grader/README.md)) |

---

## 1. Problem Statement

Every submission today runs the exact same linear path: load → AI grade → output. But real grading policy is conditional. A blank submission should get a zero without burning an LLM call; a late submission may need a cap; a very short submission should be flagged; a low-confidence AI result should be routed to a human instead of written automatically; a high-originality-similarity score should divert to an integrity path. There is no way to express "if X then this branch, else that branch." This node evaluates a **deterministic rule** (no LLM) against the submission and upstream signals and forwards its input down one of two branches — `then` or `else`.

## 2. Goals

- Evaluate a structured predicate over the submission and upstream values (score, confidence, originality, length, lateness, regex match) — no LLM, fully deterministic.
- Forward the incoming value to a `then` branch when true and an `else` branch when false; the untaken branch produces no value.
- Compose with the other nodes: route to a cheaper path, a [Human Review Gate](node-human-review-gate.md), or a [Flag for Review](node-flag-for-review.md) sink.
- Define and document the **branching execution semantics** the canvas needs (optional slot values, terminal-reachability validation) — this node owns that change.

## 3. Non-Goals

- LLM-based routing/classification (a future "Classifier" node can emit a label that this node routes on).
- Loops — the graph stays acyclic; routing only forwards.
- Arbitrary scripting — conditions are a fixed, safe predicate vocabulary, not free code.

## 4. Personas & User Stories

- **As an instructor**, I want blank/near-empty submissions to auto-score zero without an AI call, so that I save cost and avoid hallucinated grades.
- **As a TA**, I want AI grades below 0.6 confidence routed to a human review queue, so that only confident grades are written automatically.
- **As an integrity-minded instructor**, I want submissions over 40% similarity diverted to an integrity review branch.
- **As an instructor with a late policy**, I want late submissions routed through a cap/penalty branch.

## 5. Functional Requirements

- **FR-1.** The palette MUST offer a **Conditional Router** node under the Processing group.
- **FR-2.** The node MUST accept one `input` edge (any single-valued kind: `grade`, `score`, `submission`, text) and expose two source handles: `then` and `else`.
- **FR-3.** The inspector MUST build a predicate from a safe vocabulary: **field** (`submissionLength`, `wordCount`, `isEmpty`, `score`, `confidence`, `originalityScore`, `isLate`, `matchesRegex`) × **operator** (`<`, `<=`, `==`, `>=`, `>`, `isTrue`, `contains`) × **value**.
- **FR-4.** At execution the predicate MUST be evaluated server-side; the input value MUST be forwarded to `then` if true, else `else`; the **untaken branch MUST yield no `slotValue`**.
- **FR-5.** Downstream nodes wired only to an untaken branch MUST be skipped (no LLM call, no cost) and reported as `skipped` in the dry-run trace.
- **FR-6.** The graph MUST remain valid only if **every executable path** reaches a terminal that satisfies the Student Grade `grade` slot **or** a [Flag for Review](node-flag-for-review.md) sink — validated client and server.
- **FR-7.** Multiple routers MAY be chained; the engine MUST evaluate reachability per path without cycles.
- **FR-8.** Routable fields not available on a given path (e.g., `confidence` with no upstream grade) MUST produce a validation issue at author time, not a runtime surprise.

## 6. Non-Functional Requirements

- **Performance** — Predicate evaluation is sub-millisecond; the chief benefit is *avoiding* LLM calls on short-circuited branches.
- **Security** — Predicate vocabulary is a closed allow-list; `matchesRegex` runs with a length/step bound to prevent ReDoS; no eval of arbitrary expressions.
- **Privacy & Compliance** — No content leaves the system for routing.
- **Accessibility** — Field/operator/value builder fully keyboard operable; conditions readable as a sentence for screen readers ("if confidence < 0.6").
- **Scalability** — Bounded by graph caps; branch skipping reduces load.
- **Reliability** — Deterministic; a missing field on a path is caught at validation (FR-8).
- **Observability** — Dry-run trace shows the evaluated condition, the chosen branch, and skipped nodes; `grader_agent_router_total{branch}`.
- **Maintainability** — Predicate evaluator is a small pure module; reachability analysis shared with validation.
- **Internationalization** — Condition sentence localized; regex is locale-agnostic.
- **Backward compatibility** — Additive; linear graphs behave identically (no router = single path).

## 7. Acceptance Criteria

- **AC-1.** *Given* a router `isEmpty == true` with `then` → a fixed-zero path and `else` → the AI grader, *When* an empty submission runs, *Then* the AI node is skipped (no call) and the grade is zero.
- **AC-2.** *Given* a router `confidence < 0.6` on the AI output with `then` → [Human Review Gate](node-human-review-gate.md), *When* a 0.4-confidence result runs, *Then* it routes to the gate branch and the auto-write branch is skipped.
- **AC-3.** *Given* a graph where the `else` branch dead-ends without reaching the grade slot or a flag sink, *When* validated, *Then* a path-reachability issue blocks saving/running.
- **AC-4.** *Given* a router using `confidence` but no grade exists upstream on that path, *When* validated, *Then* an unavailable-field issue is shown.
- **AC-5.** *Given* a dry run through a router, *When* it completes, *Then* the trace shows the condition result, chosen branch, and any skipped nodes.

## 8. Data Model

No new tables. Node `data`:

```jsonc
{
  "condition": {
    "field": "confidence",
    "operator": "<",
    "value": 0.6
  }
  // matchesRegex uses { field: "submissionText", operator: "contains"|"matchesRegex", value: "..." }
}
```

Branch taken is an execution-time fact, not persisted (dry-run trace only). No backfill.

## 9. API Surface

- No new routes.
- The dry-run `DryRunEvent` stream gains a `branch`/`skipped` semantics: routers emit a `log` with the decision and `node_complete{status:"skipped"}` for short-circuited nodes (extend the existing status enum used by [`dry-run-console.tsx`](../../../clients/web/src/components/annotation/grader-agent/dry-run-console.tsx)).
- OpenAPI: node `data` schema; document the `skipped` node status.

## 10. UI / UX

- **Palette** — "Conditional Router" in `groupProcessing` (slate/neutral, control-flow styling).
- **Node body** — Title; one `input` slot; two clearly labelled output slots **Then** (top) and **Else** (bottom); the condition rendered as a one-line sentence.
- **Inspector** — Condition builder (field dropdown → operator dropdown → value input with type-aware control), with available fields filtered to those reachable on this node's path; a plain-language preview.
- **Canvas affordance** — Skipped nodes render dimmed during/after a dry run.
- **States** — Unavailable field (error), unreachable branch (warning on the dangling branch), running/decided.
- **Mobile** — Builder stacks.
- **Copy & i18n** — `gradingAgent.canvas.palette.router`, `gradingAgent.canvas.nodes.router.*`, `gradingAgent.canvas.inspector.condition*`, branch labels `gradingAgent.canvas.slots.then`/`.else`.

## 11. AI / ML Considerations

No model call. It *governs* AI usage: short-circuiting branches avoids unnecessary LLM calls (cost) and prevents the model from grading degenerate inputs (blank/non-submissions) where it tends to hallucinate. Confidence-based routing is a primary lever for the human-in-the-loop posture in [19.16 §11](../auto-grader-agent.md).

## 12. Integration Points

- **Client** — `types.ts` (`PaletteNodeType` += `'conditionalRouter'`, `HANDLE_THEN`/`HANDLE_ELSE`, `HANDLE_INPUT` reuse), `node-palette.tsx`, `workflow-nodes.tsx` (`ConditionalRouterNode`), `workflow-node-types.ts`, `validation.ts` (**new path-reachability + field-availability analysis**; routers create branch sets), `inspector-panel.tsx`, `dry-run-console.tsx` (render skipped), [`use-grader-agent-workflow.ts`](../../../clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts) (`NodeExecutionStatus` += `'skipped'`).
- **Server** — `workflow.go` (`NodeTypeConditionalRouter`, edge typing for `input`/`then`/`else`, **reachability validation**), `workflow_execute.go` (**branch-aware walk**: a node executes only if at least one of its inputs received a value; routers set only the taken branch), a pure `predicate.go` evaluator.
- **Owns** the branching change consumed by [Human Review Gate](node-human-review-gate.md) and [Flag for Review](node-flag-for-review.md).

## 13. Dependencies & Sequencing

- **After**: 19.16, 19.17.
- **Before**: [Human Review Gate](node-human-review-gate.md), [Flag for Review](node-flag-for-review.md) (both rely on routing + optional terminals).
- **Shared infra**: none new; introduces the branching execution model.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Branching breaks the "grade slot must be connected" invariant | H | H | Replace with per-path reachability validation (FR-6); extensive validation tests; the linear case is a trivial single path |
| Author builds a graph where no path reaches a terminal | M | H | Author-time reachability check blocks save/run with a precise issue |
| ReDoS via `matchesRegex` | L | H | Bounded regex engine / step limit; input length cap |
| Field referenced that isn't produced on the path | M | M | Field-availability analysis at author time (FR-8) |
| Confusing "skipped vs failed" in the trace | M | L | Distinct status + dimmed rendering + copy |

## 15. Rollout Plan

- Behind `grader_agent_enabled`.
- Sequencing: validation reachability model → branch-aware execution → predicate evaluator → node/inspector → i18n. **Land before** the gate/flag nodes since they depend on branching.
- Dogfood: empty-submission short-circuit and confidence routing.
- Rollback: remove palette item behind flag; without routers the engine runs the single linear path as before.

## 16. Test Plan

- **Unit** — Predicate evaluator (every field/operator); reachability analysis (valid/invalid graphs); field-availability per path; regex bounds.
- **Integration** — Dry run short-circuits a branch (no LLM call); confidence routing; skipped-node trace; multi-router chains.
- **E2E** — Build empty→zero / else→AI graph; run a blank and a normal submission; verify branch + cost.
- **Security** — ReDoS attempt bounded; predicate allow-list enforced.
- **Accessibility** — axe; keyboard condition builder; SR-readable condition sentence.

## 17. Documentation & Training

- Help center: "Branching your grading workflow with conditions."
- Instructor guide: cost-saving short-circuits, confidence routing, integrity diversion, late policy.
- API reference: node `data` schema + `skipped` status.

## 18. Open Questions

1. Support an N-way `switch` (label → branch) in addition to binary then/else? (Defer; binary covers the top cases.)
2. Should `isLate`/`wordCount` be computed here or supplied by upstream signal nodes? (Plan: a small set of intrinsic submission fields computed inline; richer signals come from dedicated nodes.)
3. How are skipped branches represented in persisted live-run results for audit? (Plan: per-item result records the taken path id.)

## 19. References

- [validation.ts](../../../clients/web/src/components/annotation/grader-agent/validation.ts), [workflow-nodes.tsx](../../../clients/web/src/components/annotation/grader-agent/workflow-nodes.tsx), [dry-run-console.tsx](../../../clients/web/src/components/annotation/grader-agent/dry-run-console.tsx), [use-grader-agent-workflow.ts](../../../clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts).
- Server: [workflow.go](../../../server/internal/service/gradingagent/workflow.go), [workflow_execute.go](../../../server/internal/service/gradingagent/workflow_execute.go) (`TopologicalNodeOrder`).
- Related: [node catalog](README.md), [Human Review Gate](node-human-review-gate.md), [Flag for Review](node-flag-for-review.md), [Originality Check](node-originality-check.md).
