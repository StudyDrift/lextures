# 19.17.1 — Rubric Node (Standalone Rubric Source)

> Implementation plan. Source: [docs/MISSING_FEATURES.md](../../MISSING_FEATURES.md) §19. Extends the grading-agent canvas ([19.17](../../completed/auto-grader-agent.md)). See the [node catalog](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | 19.17.1 |
| **Section** | AI-Specific Capabilities → Grading Agent Canvas |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | AI / Grading team |
| **Depends on** | 19.16 (grading agent), 19.17 (canvas) |
| **Unblocks** | [19.17.3 Criterion Grader](../../completed/agent-grader/node-criterion-grader.md), reusable rubric grading across activities |

---

## 1. Problem Statement

Today the only way to feed a rubric into the canvas is the **Activity** node, which couples the rubric to one assignment's stored rubric. Instructors routinely want to grade against a rubric that is *not* the assignment's own — a department-standard writing rubric, a rubric copied from another assignment, or one authored ad-hoc for the agent — and to wire **one** rubric into several downstream graders. There is no standalone rubric source, so these workflows are impossible. This node adds a **Rubric** input node that resolves a rubric from the assignment, a saved/library rubric, or an inline definition, and exposes it on a `rubric` output identical in type to the Activity node's rubric output.

## 2. Goals

- Provide a palette input node whose only job is to emit a `rubric` slot value.
- Let the instructor choose the rubric source: **this assignment**, **another assignment / library rubric** (picker), or **inline** (author criteria in the inspector).
- Make the `rubric` output wire-compatible everywhere the Activity `rubric` output is accepted today (AI input, grader rubric input).
- Allow one Rubric node to fan its output into multiple downstream nodes.

## 3. Non-Goals

- A full rubric-authoring UI — inline mode reuses the existing rubric editor component; this node selects/embeds, it does not reinvent rubric CRUD.
- Cross-org rubric sharing or a global rubric marketplace.
- Changing how rubric scores are validated/clamped at grade time (still `assignmentrubric.ValidateRubricScoresForGrade`).

## 4. Personas & User Stories

- **As an HE instructor**, I want to grade essays against my department's shared rubric rather than the assignment's, so that grading is consistent across sections.
- **As an instructor reusing a recipe**, I want to drop one Rubric node and feed it into three Criterion Graders, so that I don't re-select the rubric three times.
- **As a TA**, I want to author a quick inline rubric for a low-stakes activity that never had one, so that the agent can still produce a rubric breakdown.
- **As a self-learner**, I want to attach a known good rubric to practice work so the AI grades me the way my instructor would.

## 5. Functional Requirements

- **FR-1.** The palette MUST offer a **Rubric** node under the Input group.
- **FR-2.** The node MUST expose exactly one source handle, `rubric` (`HANDLE_RUBRIC`), typed identically to the Activity node's rubric output.
- **FR-3.** The inspector MUST let the instructor pick a source mode: `assignment` (default = grading assignment), `library` (assignment/rubric picker, reusing [`AssignmentPicker`](../../../clients/web/src/components/annotation/grader-agent/assignment-picker.tsx) + a rubric list), or `inline`.
- **FR-4.** In `inline` mode the inspector MUST embed the existing rubric editor and persist the authored `RubricDefinition` in node `data.rubric`.
- **FR-5.** A Rubric `rubric` output MUST be acceptable anywhere an Activity `rubric` output is accepted today, and MUST be rejected anywhere it is not — enforced in both client `connectionIsValid`/`validateWorkflowGraph` and server `validateEdgeTypes`.
- **FR-6.** A single Rubric output MAY feed multiple targets (fan-out is already permitted on source handles).
- **FR-7.** When wired to an AI node input, the AI node's output format MUST switch to `rubric` exactly as it does for an Activity rubric input (`aiOutputFormatForNode`).

## 6. Non-Functional Requirements

- **Performance** — Rubric resolution adds ≤ 1 DB read per dry run; library rubrics cached for the canvas session.
- **Security** — Library/other-assignment rubrics resolvable only within the same course/tenant (`d.requireCourseAccess`); inline rubrics stored in the per-tenant config graph.
- **Privacy & Compliance** — Rubrics contain no student PII; no new obligations.
- **Accessibility** — Source-mode radio group and picker keyboard-navigable, labelled; inline editor inherits rubric-editor a11y (WCAG 2.1 AA).
- **Scalability** — n/a beyond existing graph caps.
- **Reliability** — A deleted/inaccessible referenced rubric surfaces a validation issue, not a crashed run.
- **Observability** — Reuse dry-run node logs (`[Rubric] Loaded N criteria`).
- **Maintainability** — Resolution logic shared with the Activity node's rubric path.
- **Internationalization** — All inspector strings under `gradingAgent.canvas.*`.
- **Backward compatibility** — Purely additive; existing graphs unaffected.

## 7. Acceptance Criteria

- **AC-1.** *Given* a graph with a Rubric node in `assignment` mode wired to an AI input, *When* the instructor dry-runs, *Then* the AI node uses rubric output format and scores against the assignment's rubric criteria.
- **AC-2.** *Given* a Rubric node in `library` mode pointing at another assignment's rubric, *When* dry-run executes, *Then* the prompt and parsed scores use *that* rubric's criterion IDs.
- **AC-3.** *Given* a Rubric node in `inline` mode with two criteria, *When* dry-run executes, *Then* the breakdown contains exactly those two criterion IDs.
- **AC-4.** *Given* a user attempts to wire the Rubric `rubric` output into the Student Grade `grade` slot, *When* they drop the edge, *Then* the connection is rejected (rubric is not a grade source).
- **AC-5.** *Given* a referenced library rubric was deleted, *When* the instructor opens the graph, *Then* a validation issue is shown on the node and the run is blocked with a clear message.

## 8. Data Model

No new tables. Rubric node configuration lives inside `assessment.grading_agent_configs.workflow_graph` (JSONB, migration [311](../../../server/migrations/311_grading_agent_workflow.sql)) as node `data`:

```jsonc
// node.data for a Rubric node
{
  "source": "assignment" | "library" | "inline",
  "rubricAssignmentItemId": "uuid",   // library mode: assignment whose rubric to use
  "rubric": { /* RubricDefinition */ } // inline mode only
}
```

- Inline `RubricDefinition` follows the existing `assignmentrubric` shape (criteria → levels with points).
- No backfill; legacy graphs have no Rubric nodes.

## 9. API Surface

- No new routes. The dry-run WebSocket ([grading_agent_dry_run_ws.go](../../../server/internal/httpserver/grading_agent_dry_run_ws.go)) and config PUT ([grading_agent_http.go](../../../server/internal/httpserver/grading_agent_http.go)) already carry the full graph.
- Library-mode rubric lookup reuses the existing course rubric read path (`assignmentrubricai` / assignment rubric repo); resolution happens inside the dry-run/run `ActivitySource`-style resolver, extended to honor `rubricAssignmentItemId`.
- OpenAPI: document the new `rubric` node `data` shape in the workflow-graph schema.

## 10. UI / UX

- **Palette** — New "Rubric" item in `groupInput`, amber styling (matches other rubric-bearing nodes).
- **Node body** — Title + single `rubric` output slot (orange dot, reusing the Activity rubric slot styling).
- **Inspector** — Source-mode segmented control; `library` shows the assignment/rubric picker; `inline` embeds the rubric editor with a criteria summary.
- **States** — Empty (no rubric chosen → validation hint), loading (picker fetch), error (deleted rubric).
- **Mobile** — Inspector stacks; inline editor scrolls.
- **Copy & i18n** — `gradingAgent.canvas.palette.rubric`, `gradingAgent.canvas.nodes.rubric.*`, `gradingAgent.canvas.inspector.rubric*`.

## 11. AI / ML Considerations

Not AI-touching itself. It shapes downstream AI nodes: a wired rubric flips them to rubric output format and supplies the criterion IDs embedded in the system prompt by [`buildAiSystemPrompt`](../../../clients/web/src/components/annotation/grader-agent/ai-output-system-prompt.ts). No prompt/PII concerns (rubrics are instructor content).

## 12. Integration Points

- **Client** — `types.ts` (`PaletteNodeType` += `'rubric'`, `GraderNodeType`), `node-palette.tsx`, `workflow-nodes.tsx` (new `RubricNode`), `workflow-node-types.ts`, `validation.ts`, `inspector-panel.tsx`, `ai-output-system-prompt.ts` (treat rubric node like activity for format detection).
- **Server** — `workflow.go` (`NodeTypeRubric`, `validateEdgeTypes`, `resolveWiredActivityItemIDs`/include-flag derivation treat Rubric as a rubric source), `workflow_execute.go` (new case loads rubric into `slotValue{rubric}`), the activity resolver.
- **Rubric resolution** — [assignmentrubricai/service.go](../../../server/internal/service/assignmentrubricai/service.go) and the assignment rubric repo.

## 13. Dependencies & Sequencing

- **After**: 19.16, 19.17.
- **Before**: [19.17.3 Criterion Grader](../../completed/agent-grader/node-criterion-grader.md) (which benefits from a shared rubric source).
- **Shared infra**: none new.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Rubric criterion IDs mismatch when applying scores to a different assignment | M | M | Only the wired rubric's IDs are used end-to-end; apply path validates against the *grading* assignment and surfaces a clear error on mismatch |
| Two rubric sources wired into one grader | M | L | Validation: a grader/AI rubric input accepts at most one rubric edge |
| Inline rubric drift from assignment rubric | L | L | Inspector shows source badge; inline is explicitly opt-in |

## 15. Rollout Plan

- Behind the existing `grader_agent_enabled` flag; no separate flag.
- Sequencing: client types/palette/validation → server validation/execution → inspector → i18n.
- Dogfood with HE writing instructors who reuse shared rubrics.
- Rollback: remove palette item; existing inline data is inert if the node type is dropped (validation rejects unknown types, so gate removal behind flag).

## 16. Test Plan

- **Unit** — `validateEdgeTypes`/`connectionIsValid` accept Rubric→AI(rubric)/grader rubric, reject Rubric→grade/comments/submission; include-flag derivation treats Rubric as rubric source.
- **Integration** — Dry run with each source mode resolves the correct criterion set; deleted library rubric → validation error.
- **E2E (Playwright)** — Add Rubric node, pick library rubric, wire to AI, dry run, see rubric breakdown.
- **Security** — Cross-course rubric reference denied.
- **Accessibility** — axe on inspector; keyboard pick + inline edit.

## 17. Documentation & Training

- Help center: "Using a standalone rubric in the grading agent."
- Instructor guide: when to use Activity vs Rubric node; reusing one rubric across graders.
- API reference: workflow-graph node `data` schema update.

## 18. Open Questions

1. Should `library` mode list rubrics by *rubric* (if rubrics become first-class) or by *assignment that owns one* (current model)? (Plan assumes by-assignment until a rubric library lands.)
2. Should inline rubrics be promotable into a saved rubric for reuse? (Fast-follow.)

## 19. References

- [workflow-nodes.tsx](../../../clients/web/src/components/annotation/grader-agent/workflow-nodes.tsx), [types.ts](../../../clients/web/src/components/annotation/grader-agent/types.ts), [validation.ts](../../../clients/web/src/components/annotation/grader-agent/validation.ts), [ai-output-system-prompt.ts](../../../clients/web/src/components/annotation/grader-agent/ai-output-system-prompt.ts), [activity-node-data.ts](../../../clients/web/src/components/annotation/grader-agent/activity-node-data.ts).
- Server: [workflow.go](../../../server/internal/service/gradingagent/workflow.go), [workflow_execute.go](../../../server/internal/service/gradingagent/workflow_execute.go), [assignmentrubricai/service.go](../../../server/internal/service/assignmentrubricai/service.go).
- Related: [node catalog](README.md), [19.16 Auto-Grader Agent](../../completed/auto-grader-agent.md), [Criterion Grader](../../completed/agent-grader/node-criterion-grader.md).
