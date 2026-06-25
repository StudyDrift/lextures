# GA-S3 — Collapse the duplicated per-node update callbacks

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](../../plan/grading-agent/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-S3 |
| **Section** | Grading Agent — Over-complexity / Simplification |
| **Severity** | MINOR |
| **Markets** | internal maintainability |
| **Status (today)** | COMPLETE |
| **Estimated effort** | XS (≤1d) |
| **Owner (proposed)** | Web / Grading squad |
| **Depends on** | — |
| **Unblocks** | faster node-config work |

## Implementation summary (2026-06-25)

- **Generic updater** — `updateNodeData<T>(nodeId, patch)` in `use-grader-agent-workflow.ts` uses `setGraph(prev => …)` with stable empty-deps `useCallback`.
- **Removed duplication** — twelve per-node `updateXNode` callbacks deleted; `inspector-panel.tsx` calls `updateNodeData` directly.
- **Functional graph helpers** — `updateNodeLabel`, `removeNode`, and `removeEdge` also use functional updates (including clearing selection on node removal).
- **Tests** — `use-grader-agent-workflow-graph-mutations.test.ts` covers patch merge, node removal, and edge removal; existing inspector tests updated.

## 1. Problem Statement

`use-grader-agent-workflow.ts` defines **~12 nearly identical callbacks** — `updateGraderNode`,
`updateAiNode`, `updateCriterionGraderNode`, `updateCodeTestRunnerNode`, `updateConditionalRouterNode`,
`updateFlagForReviewNode`, `updateOriginalityNode`, `updateReferenceNode`, `updateRubricNode`,
`updateHumanReviewGateNode`, `updateScoreAggregatorNode`, `updateActivityNode` — each of which is the
same body:

```ts
setGraph({ ...graph, nodes: graph.nodes.map(n => n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n) })
```

Each also closes over `graph` (not the functional `setGraph(prev => …)` form), so they all invalidate
together on every graph change. Every new node type adds yet another copy. This is ~120 lines of pure
duplication and a steady tax on adding nodes.

## 2. Goals

- One generic `updateNodeData(nodeId, patch)` (functional-update form) used by every node inspector.
- Keep per-node **typed** wrappers only where a component wants a narrow `Partial<XNodeData>` signature.
- Remove the closure-over-`graph` churn.

## 3. Non-Goals

- Changing inspector components' external props beyond the update signature.
- Reworking the graph state container.

## 4. Personas & User Stories

- **As a web engineer**, I want one updater, so that adding a node type does not mean adding a callback.
- **As a reviewer**, I want less duplicated code to read, so that diffs are smaller.

## 5. Functional Requirements

- **FR-1.** A single `updateNodeData<T>(nodeId: string, patch: Partial<T>)` MUST exist, implemented with `setGraph(prev => …)`.
- **FR-2.** All existing `updateXNode` call sites MUST route through it (either directly or via thin typed wrappers).
- **FR-3.** Behavior MUST be identical (same patches applied, same re-renders or fewer).
- **FR-4.** `updateNodeLabel`, `removeNode`, `removeEdge` SHOULD also adopt the functional-update form for consistency.

## 6. Non-Functional Requirements

- **Performance** — fewer callback identities recreated per render (stable `useCallback` with empty deps via functional update).
- **Maintainability** — net deletion of ~100 lines.
- **Accessibility / i18n** — unaffected.
- **Backward compatibility** — internal only; no API/UX change.

## 7. Acceptance Criteria

- **AC-1.** *Given* each node inspector, *when* a field changes, *then* the node data updates exactly as before.
- **AC-2.** *Given* the refactor, *when* counting update callbacks, *then* there is one generic updater plus only intentional typed wrappers.
- **AC-3.** *Given* existing tests, *when* run, *then* they pass unchanged.

## 8. Data Model

- None.

## 9. API Surface

- None.

## 10. UI / UX

- None visible. Inspector components receive a single `onChange`/`updateNodeData` style callback.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- `clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts` (the updaters + return object).
- Inspector components that consume the per-node updaters: `inspector-panel.tsx`, `*-inspector.tsx`.

## 13. Dependencies & Sequencing

- Independent; low risk. Good warm-up before [GA-S4](../../plan/grading-agent/simplify-4-legacy-node-type-aliases.md).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A wrapper had subtly different typing a component relied on | L | L | Keep thin typed wrappers where a component imports a specific `Partial<XNodeData>` |
| Functional-update changes a memoization assumption | L | L | Verify inspectors with existing component tests |

## 15. Rollout Plan

- Single PR; no flag needed (internal refactor with full test coverage).
- Rollback: revert PR.

## 16. Test Plan

- **Unit/Component** — existing inspector tests cover each node type's edits.
- **Manual** — edit one field per node type and confirm persistence on save.

## 17. Documentation & Training

- None beyond a short PR description; optionally a contributor note "use `updateNodeData`".

## 18. Open Questions

1. Keep typed wrappers for every node or only those whose inspectors want a narrowed signature?
   - **Resolved:** no per-node wrappers; inspectors call `updateNodeData` directly.

## 19. References

- `clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts` (lines defining `updateGraderNode` … `updateScoreAggregatorNode`).
- Related: [GA-S4](../../plan/grading-agent/simplify-4-legacy-node-type-aliases.md).
