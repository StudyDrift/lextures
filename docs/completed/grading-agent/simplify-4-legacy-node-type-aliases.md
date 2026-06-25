# GA-S4 — Retire legacy node-type aliases & palette ternaries

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-S4 |
| **Section** | Grading Agent — Over-complexity / Simplification |
| **Severity** | MINOR |
| **Markets** | internal maintainability |
| **Status (today)** | COMPLETE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | — |
| **Unblocks** | cleaner node additions |

## Implementation summary (2026-06-25)

- **Load-time normalizer** — `workflow_normalize.go` / `workflow-normalize.ts` rewrite `submission→studentSubmission`, `assignmentContext→activity`, and expand legacy `context` handles into explicit content/rubric edges (preserving include-flag semantics). Invoked from `UnmarshalWorkflowGraph` and client `normalizeWorkflowGraph`.
- **Alias removal** — `isActivityNodeType`, `isStudentSubmissionNodeType`, `deriveIncludeFlags`, `validateEdgeTypes`, and client validation no longer branch on legacy types/handles after normalization.
- **Node registry** — `NODE_DESCRIPTORS` + `paletteNodeDefaults` replace the ~90-line `addPaletteNode` ternary pyramid; snapshot-tested in `node-descriptors.test.ts`.
- **Migration** — `329_grading_agent_normalize_legacy_nodes.sql` documents lazy persistence (canonical graphs saved on next config/template write).
- **Tests** — Go golden compile test for assignmentContext include flags; client normalizer and descriptor snapshot tests.

## 1. Problem Statement

The graph schema carries **three legacy node types** that the palette no longer offers but every layer
must still special-case:

- `submission` (alias of `studentSubmission`), `assignmentContext` (alias of `activity`), and `grader`
  (superseded by `ai` / `criterionGrader`).

Server code threads these through helpers like `isStudentSubmissionNodeType`, `isActivityNodeType`,
the `HandleContext` legacy handle, and `deriveIncludeFlags`'s `assignmentContext` branch. On the client,
`addPaletteNode` builds node prefixes, fallback positions, and default data via **three giant nested
ternaries** spanning ~90 lines that are easy to mis-edit (the ordering of `originality`/`reference`/
`rubric`/`criterionGrader` branches is already non-obvious). This accidental complexity slows every
node change and is a frequent source of "why did my node get the wrong default position" confusion.

## 2. Goals

- A one-time migration that rewrites stored graphs from legacy types/handles to current ones, then removes the alias branches.
- Replace the `addPaletteNode` ternary pyramids with a single declarative node-descriptor table (prefix, default position, default data factory).
- Shrink the type/handle/validation surface accordingly.

## 3. Non-Goals

- Removing genuinely-current node types.
- Changing node behavior or default values (defaults preserved exactly).

## 4. Personas & User Stories

- **As an engineer**, I want a node registry, so that adding a node is one table entry, not edits across five ternaries.
- **As a maintainer**, I want legacy aliases gone, so that validation/compile logic is half as branchy.

## 5. Functional Requirements

- **FR-1.** A migration MUST rewrite persisted `workflow_graph` JSON: `submission→studentSubmission`, `assignmentContext→activity` (preserving its `includeContent`/`includeRubric` semantics by wiring content/rubric handles), `grader→ai` where safe, and `context` handle → `content`/`rubric` edges.
- **FR-2.** After migration, the alias helpers and `NodeTypeSubmission`/`NodeTypeAssignmentCtx`/legacy `grader` handling MAY be removed (or reduced to a load-time normalizer kept for one release).
- **FR-3.** The client MUST replace the `addPaletteNode` ternaries with a `NODE_DESCRIPTORS` map keyed by `PaletteNodeType` providing `{ idPrefix, fallbackPosition(index), defaultData() }`.
- **FR-4.** Defaults (positions, data) MUST match current output for every node type (snapshot-tested).
- **FR-5.** Validation and compile MUST continue to accept any pre-migration graph during the deprecation window via a normalizer.

## 6. Non-Functional Requirements

- **Reliability** — migration is idempotent and reversible (keep a backup of pre-migration JSON or gate behind a normalizer rather than destructive rewrite initially).
- **Maintainability** — measurable reduction in branch count in `workflow.go` and `use-grader-agent-workflow.ts`.
- **Backward compatibility** — old saved graphs still load (normalizer) until the migration runs; templates migrated too.
- **Observability** — log count of graphs normalized/migrated.

## 7. Acceptance Criteria

- **AC-1.** *Given* a stored graph using `submission`/`assignmentContext`/`grader`, *when* migrated, *then* it loads, validates, and grades identically to before.
- **AC-2.** *Given* the node registry, *when* a node is added via the palette, *then* its prefix, position, and defaults match the previous ternary output (snapshot test).
- **AC-3.** *Given* the removed aliases, *when* the server suite runs, *then* it compiles and passes.
- **AC-4.** *Given* a saved template with legacy types, *when* cloned to an assignment, *then* it works.

## 8. Data Model

- No schema change; a data migration over `assessment.grading_agent_configs.workflow_graph` and `..._templates.workflow_graph`.
- Migration: `server/migrations/NNN_grading_agent_normalize_legacy_nodes.sql` (or a Go one-shot if JSON rewriting is easier in code).

## 9. API Surface

- None. Internal.

## 10. UI / UX

- None visible; palette/canvas behavior preserved.

## 11. AI / ML Considerations

- None (prompt/scoring unchanged; `assignmentContext` include-flag semantics preserved via explicit handles).

## 12. Integration Points

- `server/internal/service/gradingagent/workflow.go` (`NodeTypeSubmission`, `NodeTypeAssignmentCtx`, `HandleContext`, `isActivityNodeType`, `isStudentSubmissionNodeType`, `deriveIncludeFlags`).
- `clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts` (`addPaletteNode`).
- `clients/web/src/components/annotation/grader-agent/types.ts` (legacy type unions).
- Migration script + template normalization.

## 13. Dependencies & Sequencing

- Independent; do after [GA-S3](simplify-3-generic-node-data-updater.md) to keep the hook diff small.
- Coordinate with [GA-S1](simplify-1-unify-grade-write-paths.md) since both touch `workflow.go`/consumer assumptions.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Destructive migration corrupts a graph | L | H | Ship a load-time normalizer first; migrate data only after a release of normalizer soak |
| `assignmentContext` include-flag semantics lost | M | M | Map to explicit content/rubric edges; golden test the include flags |
| Hidden consumer reliance on legacy handles | M | M | Keep normalizer for one release; grep all `Handle*`/`NodeType*` legacy refs |

## 15. Rollout Plan

- Flag/phase: (1) add normalizer + node registry (non-destructive), ship; (2) run data migration; (3) delete alias branches.
- Rollback: keep normalizer until step 3; revert deletion if needed.

## 16. Test Plan

- **Unit** — normalizer maps each legacy type/handle correctly; node-registry defaults snapshot.
- **Golden** — legacy graphs grade identically pre/post.
- **Integration** — templates with legacy types clone and run.

## 17. Documentation & Training

- Contributor note: "Add a node by extending `NODE_DESCRIPTORS` + the registry; legacy aliases are gone."

## 18. Open Questions

1. Do `grader` nodes always map cleanly to `ai`, or do some need `criterionGrader`? (Audit stored graphs first.)
2. Keep the load-time normalizer permanently as a safety net, or remove after migration?

## 19. References

- `server/internal/service/gradingagent/workflow.go` (legacy constants + `isActivityNodeType`/`isStudentSubmissionNodeType`/`deriveIncludeFlags`).
- `clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts` (`addPaletteNode` ternaries).
- Related: [GA-S1](simplify-1-unify-grade-write-paths.md), [GA-S3](simplify-3-generic-node-data-updater.md).
