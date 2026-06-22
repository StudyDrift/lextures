# 19.17 — Grader Agent Workflow Canvas (Node-Based SpeedGrader Grading Agent)

> Implementation plan. Source: [docs/MISSING_FEATURES.md](../MISSING_FEATURES.md) §19. Supersedes the side-drawer UI of [19.16 — Auto-Grader Agent](auto-grader-agent.md); reuses its backend scoring service, governance gate, runs/results tables, and grade-write path.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | 19.17 |
| **Section** | AI-Specific Capabilities |
| **Severity** | MAJOR |
| **Markets** | K12, HE, SL |
| **Status (today)** | PARTIAL (19.16 shipped a prompt-and-toggle drawer; this replaces the UX with a visual workflow editor) |
| **Estimated effort** | M (2–4 w) |
| **Owner (proposed)** | AI / Grading team |
| **Depends on** | 19.16 (grading-agent service, runs/results schema, governance + grade-write paths), 19.3 (`graded_by_ai` flag), 19.10 (model governance), 19.11 (PII redaction) |
| **Unblocks** | Multi-step grading recipes (separate score vs. feedback sources); future per-criterion and conditional grading nodes |

---

## 1. Problem Statement

[19.16](auto-grader-agent.md) shipped an instructor-authored grading agent as a SpeedGrader side drawer: one prompt textarea, an "include assignment content/rubric" toggle, a dry run, and scope runs ([grader-agent-drawer.tsx](../../clients/web/src/components/annotation/grader-agent-drawer.tsx)). That UX hides what the agent actually does and makes it hard to express grading as distinct, reusable steps — e.g. "use *this* prompt to derive the score, but *that* source for the comment." Instructors cannot see or compose the data flow that produces a grade.

This plan reworks the **front end** into a full-screen **workflow canvas** built on [React Flow](https://reactflow.dev/) (`@xyflow/react` v12). When the instructor opens the Grader Agent, a full-screen modal presents a node editor. A single, fixed **Output node — "Student Grade"** anchors the graph with exactly two input slots: **grade** and **comments**. The instructor wires source nodes (an LLM grader node, optional assignment-context node, the implicit submission node) into those two slots, then **runs the workflow** — a **dry run** that previews the score and comment without persisting, or **apply grades** across a scope. The backend scoring service, governance gate, runs/results tables, and grade-write path from 19.16 are reused unchanged; the only backend additions are a stored **workflow graph** and a small **compiler/validator** that turns the graph into the existing `ScoreRequest`.

## 2. Goals

- Replace the grader-agent drawer with a **full-screen modal** containing a React Flow node editor, launched from the same SpeedGrader entry point.
- Provide one **fixed, non-deletable Output node** ("Student Grade") with exactly two input slots — **grade** and **comments** — that map to the submission's points/rubric and instructor comment respectively.
- Let instructors add and connect grading nodes (LLM grader, assignment/rubric context, submission source) and edit each node's settings in an inspector panel.
- Run the workflow two ways: **dry run** against the open submission (preview only, no persistence) and **apply grades** across `current` / `ungraded` / `all` scopes — reusing 19.16's dry-run and runs endpoints.
- Persist the graph per assignment, migrate existing 19.16 prompt-based configs into an equivalent default graph, and keep agent-produced grades **unposted, editable, and `graded_by_ai`-flagged**.

## 3. Non-Goals

- Changing the grading *model*, governance, PII redaction, cost logging, batch/auto-grade execution, or grade-write semantics — all reused verbatim from [19.16](auto-grader-agent.md).
- A general-purpose automation builder. The canvas grades one assignment's submissions; it is **not** a Zapier-style cross-feature workflow engine.
- Branching/conditional logic, loops, or fan-out to multiple assignments (v1 is a single-sink DAG; see §18).
- Per-rubric-criterion grader nodes, "static/templated comment" nodes, and multi-grader composition — the data model supports them but they are fast-follow, not v1 (§3 supported node set in §10).
- Real-time multi-instructor co-editing of the canvas (single-author, last-write-wins like the existing config).
- Replacing the [19.3](19-ai-capabilities/19.3-ai-assisted-grading.md) approval queue or §3.4 moderated grading.

## 4. Personas & User Stories

- **As an instructor**, I want to open a full-screen canvas where I can see the grade "flowing" from my prompt into the student's grade, so the agent's behavior is transparent and tweakable.
- **As an instructor**, I want the student-grade Output node to always be present with a slot for the score and a slot for the comment, so I know exactly what the workflow must produce.
- **As an instructor tuning a workflow**, I want a dry run that previews the score and comment without touching the student's grade, so I can iterate safely.
- **As a TA**, I want to apply the workflow only to not-yet-graded submissions so I don't overwrite human grades.
- **As a keyboard / screen-reader user**, I want a form-based fallback view of the same workflow so I can author and run the agent without a pointer.
- **As an org admin**, I want this to remain behind the existing `grader_agent_enabled` flag with the same audit and `graded_by_ai` guarantees.

## 5. Functional Requirements

- **FR-1.** The SpeedGrader "Grader Agent" entry point ([submission-grading-panel.tsx](../../clients/web/src/components/annotation/submission-grading-panel.tsx), gated by `graderAgentEnabled` + staff mode) MUST open a **full-screen modal** (`role="dialog"`, `aria-modal`, focus-trapped, ESC-closable) instead of the side drawer.
- **FR-2.** The modal MUST render a React Flow canvas (pan/zoom, minimap, controls) containing exactly one **Output node** ("Student Grade") that MUST NOT be deletable or duplicable and MUST expose exactly two labelled **target handles**: `grade` and `comments`.
- **FR-3.** The instructor MUST be able to add **Grader (LLM)** nodes and an **Assignment Context** node from a node palette, position them, connect them with edges, and delete non-output nodes.
- **FR-4.** A **Grader node** MUST expose its prompt and model in an inspector panel and MUST provide two **source handles** — `grade` and `comments` — that can be wired to the Output node's matching slots.
- **FR-5.** Edge connections MUST be **type-validated**: a `grade` source may only connect to the `grade` slot, a `comments` source only to the `comments` slot; each Output slot accepts at most one inbound edge; cycles MUST be rejected.
- **FR-6.** A workflow is **runnable** only when the Output node's `grade` slot is connected to a valid source; the `comments` slot is optional (if unconnected, no comment is written). The UI MUST surface validation errors (unconnected required slot, orphan node, unconfigured grader prompt) before run.
- **FR-7.** A **Dry run** action MUST compile the graph, execute it against the currently open submission, and return a previewed score, per-criterion rubric breakdown (when a rubric exists), and comment — and MUST NOT write or modify any grade.
- **FR-8.** The dry-run preview MUST be editable and applyable to the open submission with one click via the existing grade-write path, flagged `graded_by_ai: true`, unposted.
- **FR-9.** An **Apply grades** control MUST offer the three existing scopes — `current`, `ungraded`, `all` — execute via the existing runs endpoint/queue, return a run handle immediately, and report progress (total / completed / failed). The `all` scope MUST require explicit overwrite confirmation.
- **FR-10.** The graph MUST be **persisted** per assignment (saved with the config) and reloaded on reopen; **Accept agent** MUST require at least one successful dry run, mirroring 19.16.
- **FR-11.** Existing 19.16 configs (prompt + include flags, no graph) MUST be **migrated/synthesized** into an equivalent default graph (Submission → Grader[prompt, include flags] → Output{grade,comments}) so accepted agents and auto-grade keep working.
- **FR-12.** Every dry run and live run MUST continue to pass through the AI governance gate, PII redaction, cost logging, untrusted-content handling, and `graded_by_ai` flagging exactly as in 19.16 — the canvas changes *authoring*, not *execution semantics*.
- **FR-13.** A **keyboard/AT-accessible alternative view** ("Form view") MUST present the same workflow (grader prompt, context toggle, slot bindings, run controls) as standard form controls, since a pure drag-canvas cannot meet WCAG 2.1 AA alone.

## 6. Non-Functional Requirements

- **Performance** — Modal opens and renders the canvas in < 300 ms for graphs ≤ 25 nodes. Dry run returns within 30 s (p95) for text submissions ≤ 2 000 words (inherited from 19.16). React Flow lazy-loaded via dynamic import to keep it off the main SpeedGrader bundle; target added gzip ≤ 60 KB on the lazily-loaded chunk.
- **Security** — Canvas, graph, dry-run, and runs accessible only to graders of the course (`requireGraderAgentAccess`). Graph is stored per-tenant; student submissions remain org-private and pass as untrusted data.
- **Privacy & Compliance** — No change to FERPA/GDPR Art. 22/COPPA posture from 19.16: human-in-loop default (unposted grades), governance gate, disclosure, and re-grade route preserved. The graph stores no student PII (only instructor-authored prompts and node config).
- **Accessibility** — Modal is focus-trapped with restore-on-close and a labelled close control. Canvas uses React Flow's `nodesFocusable`/`edgesFocusable`/keyboard pan; **plus** the FR-13 Form view is the conformant path for keyboard/AT users. Run status announced via an `aria-live` region. Target WCAG 2.1 AA via the Form view; canvas is a progressive enhancement.
- **Scalability** — Graph size capped (≤ 50 nodes / 100 edges server-side) to bound compile/validate cost. Batch/auto-grade execution unchanged (shared queue, per-course concurrency cap).
- **Reliability** — Graph validation is deterministic and runs both client- and server-side; an invalid/oversized graph is rejected with a field-level error and never reaches the model. Per-submission failures isolated as in 19.16.
- **Observability** — Reuse 19.16 metrics; add `grader_agent_graph_nodes` (histogram), `grader_agent_graph_invalid_total{reason}`, and `grader_agent_view_used{canvas|form}`.
- **Maintainability** — New FE module `clients/web/src/components/annotation/grader-agent/` (modal, canvas, node components, inspector, form-view, graph types/validation). New BE file `server/internal/service/gradingagent/workflow.go` (graph parse + validate + compile to `ScoreRequest`). Drawer is removed once parity is verified.
- **Internationalization** — All node labels, palette, inspector, and validation strings externalised under `gradingAgent.canvas.*`.
- **Backward compatibility** — Manual SpeedGrader grading unchanged. Legacy prompt-based configs auto-upgraded (FR-11). Dry-run/run request bodies accept the new `graph` field but still honor the legacy `prompt`/`include*` fields when `graph` is absent.

## 7. Acceptance Criteria

- **AC-1.** *Given* a grader opens a submission in SpeedGrader with `graderAgentEnabled`, *When* they click **Grader Agent**, *Then* a full-screen modal opens containing a React Flow canvas with a single non-deletable "Student Grade" output node exposing `grade` and `comments` slots.
- **AC-2.** *Given* the canvas, *When* the instructor drags a Grader node's `grade` handle onto the Output `grade` slot, *Then* the edge is created; *When* they try to drag the same `grade` handle onto the `comments` slot, *Then* the connection is rejected.
- **AC-3.** *Given* the Output `grade` slot is unconnected, *When* the instructor clicks **Dry run** or **Apply**, *Then* the run is blocked with a "connect the grade slot" validation message and nothing is sent to the model.
- **AC-4.** *Given* a valid graph (Grader → Output), *When* the instructor clicks **Dry run**, *Then* a previewed score, rubric breakdown, and comment appear within 30 s and the student's stored grade is unchanged.
- **AC-5.** *Given* a dry-run preview, *When* the instructor edits the score/comment and clicks **Apply to this student**, *Then* the open submission's grade is written via the standard path, flagged `graded_by_ai`, unposted.
- **AC-6.** *Given* a valid graph and a successful dry run, *When* the instructor selects **Submitted, not graded** and applies, *Then* only ungraded submissions receive provisional agent grades and progress (completed/failed) is reported.
- **AC-7.** *Given* an existing 19.16 prompt-based config, *When* the instructor opens the modal, *Then* a default graph (Submission → Grader[its prompt + include flags] → Output) is rendered and is immediately runnable.
- **AC-8.** *Given* the comments slot is unconnected but grade is connected, *When* the workflow runs, *Then* a grade is written and no instructor comment is overwritten/added.
- **AC-9.** *Given* a keyboard-only user, *When* they open the modal and switch to **Form view**, *Then* they can edit the grader prompt, toggle assignment context, see slot bindings, run a dry run, and apply — all without a pointer.
- **AC-10.** *Given* the tenant disabled the feature or a student opted out, *When* the instructor clicks **Dry run**, *Then* the call is blocked with the governance message and nothing is sent to the model (unchanged from 19.16).

## 8. Data Model

Migration `server/migrations/311_grading_agent_workflow.sql` (next free number after `310_scim_groups.sql`).

```sql
-- 311_grading_agent_workflow.sql
-- Plan 19.17 — store the visual workflow graph for the grading agent.

ALTER TABLE assessment.grading_agent_configs
    ADD COLUMN IF NOT EXISTS workflow_graph JSONB;

COMMENT ON COLUMN assessment.grading_agent_configs.workflow_graph IS
    'React Flow node graph authored in the SpeedGrader workflow canvas (plan 19.17). '
    'NULL for legacy prompt-only configs, which are synthesized into a default graph on read.';

-- Optional: record which authoring surface produced the last run, for observability.
ALTER TABLE assessment.grading_agent_runs
    ADD COLUMN IF NOT EXISTS authored_via TEXT;  -- 'canvas' | 'form' | NULL (legacy)
```

- **Graph shape** (stored JSON; versioned for forward-compat):

```jsonc
{
  "version": 1,
  "nodes": [
    { "id": "output", "type": "output", "position": {"x":0,"y":0}, "data": {} },
    { "id": "g1", "type": "grader", "position": {"x":-320,"y":0},
      "data": { "prompt": "Award full marks for a clear thesis…", "modelId": null } },
    { "id": "ctx", "type": "assignmentContext", "position": {"x":-640,"y":120},
      "data": { "includeContent": true, "includeRubric": true } },
    { "id": "sub", "type": "submission", "position": {"x":-640,"y":-80}, "data": {} }
  ],
  "edges": [
    { "id": "e1", "source": "g1", "sourceHandle": "grade",    "target": "output", "targetHandle": "grade" },
    { "id": "e2", "source": "g1", "sourceHandle": "comments", "target": "output", "targetHandle": "comments" },
    { "id": "e3", "source": "ctx", "target": "g1", "targetHandle": "context" },
    { "id": "e4", "source": "sub", "target": "g1", "targetHandle": "submission" }
  ]
}
```

- **Backfill**: none required. Rows with `workflow_graph IS NULL` are synthesized on read into the default graph above from existing `prompt` / `include_assignment_content` / `include_rubric`. On the next save the synthesized graph is persisted, naturally upgrading rows. The legacy scalar columns remain the source of truth for any non-canvas caller.
- **Constraints**: graph validated in application code (size caps, single output node, type-checked edges, acyclic) rather than in SQL; only well-formed graphs are written.

## 9. API Surface

No new routes. The existing 19.16 endpoints ([courses_routes.go](../../server/internal/httpserver/courses_routes.go) lines 117–122) gain an optional `graph` field; when present it is the source of truth and the legacy fields are derived from it for storage/back-compat.

```
GET  /api/v1/courses/{course_code}/assignments/{item_id}/grader-agent
       -> { config: { …existing…, workflowGraph } | null }
       // workflowGraph synthesized from prompt/flags when stored value is null

PUT  /api/v1/courses/{course_code}/assignments/{item_id}/grader-agent
       { prompt?, includeAssignmentContent?, includeRubric?, status,
         autoGradeNew?, model?, workflowGraph }
       -> { config }     // server validates graph, derives prompt/flags, persists both

POST /api/v1/courses/{course_code}/assignments/{item_id}/grader-agent/dry-run
       { workflowGraph, submissionId, model? }      // graph compiled to ScoreRequest
       -> { suggestedPoints, rubricScores, comment, confidence,
            promptTokens, completionTokens }         // does NOT persist a grade

POST /api/v1/courses/{course_code}/assignments/{item_id}/grader-agent/runs
       { scope: 'current'|'ungraded'|'all', submissionId?, overwrite?, authoredVia? }
       -> { runId, totalCount }                      // uses the accepted/persisted graph

GET  /api/v1/courses/{course_code}/assignments/{item_id}/grader-agent/runs/{run_id}
       -> { status, totalCount, completedCount, failedCount, results[] }   // unchanged
```

- **Validation errors** return `400` with `{ error, field }` where `field` identifies the offending node/slot (e.g. `output.grade`, `node:g1.prompt`).
- **Apply path** unchanged: reuses `PUT .../submissions/.../grade` so rubric validation (`assignmentrubric.ValidateRubricScoresForGrade`) and Canvas grade-sync remain intact.
- **OpenAPI**: update the grader-agent request/response schemas to add `workflowGraph` and the graph type.

## 10. UI / UX

### Entry point
The existing "Grader Agent" button in [submission-grading-panel.tsx](../../clients/web/src/components/annotation/submission-grading-panel.tsx) now opens `GraderAgentWorkflowModal` (full-screen) instead of `GraderAgentDrawer`. The drawer file is removed after parity is confirmed.

### Full-screen modal layout
- **Header bar**: title "Grader Agent", assignment name, a **Canvas / Form view** segmented toggle (FR-13), **Dry run** button, **Apply grades** (with scope menu), **Accept agent** (enabled after a successful dry run), and **Close** (✕, ESC).
- **Left palette**: draggable node types — **Grader (LLM)**, **Assignment Context**. (Submission and Output are pre-placed; Output is fixed.)
- **Center canvas** (React Flow): minimap, zoom/pan controls, snap grid, type-validated connections, invalid-slot highlighting.
- **Right inspector**: settings for the selected node — Grader: prompt textarea, model picker (defaults to the resolved tenant model), help text/examples; Assignment Context: include-content + include-rubric toggles; Output: read-only explainer of the two slots.
- **Bottom dock**: dry-run preview card — total score (`x / maxPoints`), editable rubric breakdown via [RubricGradePicker](../../clients/web/src/components/grading/rubric-grade-picker.tsx), editable comment, confidence chip, and **Apply to this student** / **Re-run** actions. Reuses the existing preview/apply logic from the drawer.

### Node set (v1)
- **Output — "Student Grade"** (fixed, single, non-deletable): target handles `grade` (top) and `comments` (bottom), each labelled; rejects a second inbound edge.
- **Grader (LLM)**: input handles `submission`, `context`; source handles `grade`, `comments`; data `{ prompt, modelId }`.
- **Assignment Context**: source handle `context`; data `{ includeContent, includeRubric }`.
- **Submission** (read-only, pre-placed): source handle `submission`; auto-supplies the student submission text. Shown for transparency; not configurable.
- Palette explicitly notes fast-follow nodes (per-criterion grader, static/templated comment, second grader for comments) as "coming soon" placeholders, not enabled in v1.

### Form view (FR-13, accessibility path)
A standard form rendering the same state: grader prompt textarea, model picker, "Include assignment content/rubric" checkboxes, slot-binding summary ("Grade ← Grader node", "Comments ← Grader node / not set"), and the same Dry run / preview / Apply / Accept / scope controls. Authoring in Form view writes back to the canonical graph; the two views are kept in sync.

### States
- **Empty / new** → default graph pre-wired (Submission → Grader → Output) with an empty prompt and a CTA to write one.
- **Validation** → unconnected required slot, orphan grader, or empty prompt disables Dry run/Apply with inline messages and node-level error rings.
- **Loading / running** → run buttons show spinners; `aria-live` announces dry-run/run status and batch progress.
- **Governance block / provider error** → inline governance message / retry affordance (reused from 19.16).
- **Mobile / responsive** → on narrow viewports the modal defaults to **Form view** (canvas drag is impractical on touch); a read-only canvas thumbnail is shown.

### Copy & i18n
New keys under `gradingAgent.canvas.*` (node labels, palette, slots, inspector, validation, view toggle) added to [common.json](../../clients/web/public/locales/en/common.json) (en) and mirrored in es/fr. Existing `gradingAgent.*` run/scope/result keys reused.

## 11. AI / ML Considerations

- **Execution is unchanged from 19.16.** The graph compiles to the existing `gradingagent.ScoreRequest`: the Grader node's `prompt` → `InstructorPrompt`; an Assignment Context node wired into the grader → `IncludeAssignmentContent` / `IncludeRubric`; the Submission node → `SubmissionText`; `modelId` resolved via tenant governance. The same `BuildMessages` system prompt, untrusted-submission delimiters, JSON output schema (`{ total, rubric, comment, confidence }`), server-side rubric clamping (`ParseAndClampModelOutput`), PII redaction (`aitutor.RedactPII`), and cost logging apply.
- **Slot mapping**: the Output node's `grade` slot consumes `{ TotalPoints, RubricScores }`; the `comments` slot consumes `Comment`. In the common single-grader graph, one `Score()` call fills both slots. The compiler is written to allow distinct grade- and comments-source nodes later (fast-follow) without an API change.
- **Compile/validate** (`workflow.go`): parse JSON → enforce caps & single output node → type-check edges → topological order (reject cycles) → emit `ScoreRequest`. Identical model calls regardless of authoring surface, so the §19.13 eval harness, injection suite (AC-8 of 19.16), and bias audit continue to apply without change.
- **Cost** — unchanged per-submission token profile; dry runs and runs logged to `analytics.ai_usage_log`; `all`-scope/auto-grade governed by §19.14 budgets.

## 12. Integration Points

- **FE entry** — [submission-grading-panel.tsx](../../clients/web/src/components/annotation/submission-grading-panel.tsx) swaps drawer → modal; gating via [platform-features-context](../../clients/web/src/context/platform-features-context.tsx) `graderAgentEnabled`.
- **FE library** — add `@xyflow/react` (React Flow v12, [reactflow.dev](https://reactflow.dev/)) to [clients/web/package.json](../../clients/web/package.json); import its stylesheet locally within the lazily-loaded canvas chunk.
- **FE API** — extend `GraderAgentConfigApi`, `putGraderAgentConfig`, `postGraderAgentDryRun` bodies in [courses-api.ts](../../clients/web/src/lib/courses-api.ts) with `workflowGraph`; add a shared `GraderWorkflowGraph` type + client-side validator.
- **BE compile/validate** — new [workflow.go](../../server/internal/service/gradingagent/workflow.go) in the existing `gradingagent` service; called from the dry-run/run/PUT handlers in [grading_agent_http.go](../../server/internal/httpserver/grading_agent_http.go).
- **BE storage** — `workflow_graph` column via `gradingagentrepo` config read/write; `graderAgentConfigToJSON` emits `workflowGraph` (synthesizing the default graph when null).
- **Reused unchanged** — `gradingagent.Score` / `BuildMessages` / `ParseAndClampModelOutput`, governance gate, `aiusage` logging, runs/results tables + background consumer, `RubricGradePicker`, grade-write path.
- **Cross-plan** — [19.16](auto-grader-agent.md) (parent), [19.3](19-ai-capabilities/19.3-ai-assisted-grading.md), §19.10/§19.11/§19.13/§19.14.

## 13. Dependencies & Sequencing

- **Must ship after**: 19.16 (entire backend + entry point already in place).
- **Must ship before**: nothing hard-blocks; enables fast-follow multi-source / per-criterion / conditional nodes.
- **Shared infra**: existing OpenRouter access, governance gate, background queue, `analytics.ai_usage_log`.
- **Can ship in parallel with**: §3.4 moderated grading.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Canvas is inaccessible to keyboard/AT users | H | H | Mandatory Form-view fallback (FR-13/AC-9) is the WCAG-conformant path; canvas is progressive enhancement |
| React Flow bloats the SpeedGrader bundle | M | M | Lazy dynamic import; CSS scoped to the chunk; bundle-size budget check in CI |
| Graph complexity invites invalid/runaway configs | M | M | Size caps + deterministic server-side validation; only well-formed graphs reach the model |
| Visual editor obscures that execution is one LLM call | M | L | Inspector help text + dry-run preview make the actual model output explicit; eval harness unchanged |
| Divergence between canvas state and persisted graph | M | M | Single canonical graph model; Form view and canvas are pure views over it; round-trip tests |
| Regression vs. shipped 19.16 drawer behavior | M | M | Reuse drawer's dry-run/apply/run logic; keep flag; parity E2E before removing drawer |
| Mobile drag-canvas unusable | M | L | Default to Form view on narrow/touch viewports |

(Prompt-injection, cost, bias, GDPR Art. 22, and overwrite risks are inherited and already mitigated in [19.16 §14](auto-grader-agent.md).)

## 15. Rollout Plan

- **Feature flag**: reuse `grader_agent_enabled` (no new flag). The canvas is the new default UI when the flag is on.
- **Sequencing**: migration `311` → BE graph compile/validate + handler changes (accept `graph`, synthesize default) → FE shared graph type/validator + API wiring → modal + canvas + nodes + inspector → Form view → parity E2E → remove drawer.
- **Phase 1**: ship canvas behind a short-lived FE sub-toggle for internal dogfood, drawer still available as fallback.
- **Phase 2**: make canvas the default for all flagged tenants; keep Form view; remove the drawer component.
- **Rollback**: FE — revert the entry point to render the drawer (kept until Phase 2); BE — `workflowGraph` is additive, legacy fields still authoritative, so no schema rollback needed.

## 16. Test Plan

- **Unit (FE)** — graph validator (type-checked edges, single-edge slots, cycle rejection, required-slot rule); default-graph synthesis from legacy config; canvas↔Form-view round-trip; connection-validation handler.
- **Unit (BE)** — `workflow.go` parse/validate (caps, single output, acyclic, type checks) and compile-to-`ScoreRequest` for the default and single-grader graphs; legacy-fallback when `graph` absent; field-level error messages.
- **Integration** — PUT with graph persists graph + derived prompt/flags; GET synthesizes graph for legacy rows; dry-run from graph returns preview without persisting; apply writes unposted `graded_by_ai` grade; run `ungraded` skips graded; governance block path.
- **End-to-end (Playwright)** — open modal → see fixed Output node → wire Grader→grade/comments → dry run → edit & apply → accept → apply to ungraded; reject cross-type connection; blocked run on unconnected grade slot; Form-view keyboard-only authoring + run (AC-9); legacy-config opens as default graph (AC-7).
- **Accessibility** — axe on modal + Form view; focus trap/restore; keyboard-only path through Form view; `aria-live` run announcements; canvas focusability smoke check.
- **Performance** — modal/canvas render budget (≤ 25 nodes); lazy-chunk bundle-size budget; dry-run p95 (inherited).
- **Security** — non-grader blocked; cross-course denied; oversized/cyclic graph rejected server-side; opt-out/COPPA/tenant blocks honored.

## 17. Documentation & Training

- **Help center** — "Building a grading workflow": the canvas, the fixed Student Grade output and its grade/comments slots, adding a Grader node, dry running, applying across a scope, and the Form view.
- **Instructor guide** — when to wire assignment context, reading the dry-run preview, the unposted-grade safety model (carried from 19.16).
- **Admin guide** — unchanged enablement (`grader_agent_enabled`), audit/`graded_by_ai`, cost budgets.
- **Accessibility note** — document Form view as the supported keyboard/AT authoring surface.
- **API reference** — `workflowGraph` schema added to the grader-agent endpoints.
- **Changelog / migration note** — 19.16 drawer replaced by the workflow canvas; existing agents auto-upgrade.

## 18. Open Questions

1. Should v1 ship the **second grader / static-comment** node so the `comments` slot can have a distinct source, or is single-grader-fills-both sufficient for launch? (Leaning single-grader for v1; data model already supports the split.)
2. Should the model be selectable **per Grader node** or stay a single resolved tenant model? (Plan: per-node `modelId`, defaulting to the resolved tenant model.)
3. Do we keep the canvas read-only on mobile/touch and force Form view, or invest in touch-friendly wiring? (Plan: Form view on touch.)
4. Should dry-run previews from the canvas be persisted as `is_dry_run` results (audit/cost) like 19.16, or stay ephemeral? (Plan: persist, consistent with 19.16.)
5. Versioning: when we add conditional/branching nodes, do we bump `graph.version` and write a forward migration of stored graphs, or validate-on-read? (Plan: validate-on-read + `version` gate.)
6. Should `authored_via` influence anything beyond observability (e.g. require canvas for `all`-scope)? (Plan: observability only.)

## 19. References

- Parent plan: [19.16 — Auto-Grader Agent](auto-grader-agent.md).
- Existing implementation: [grader-agent-drawer.tsx](../../clients/web/src/components/annotation/grader-agent-drawer.tsx), [submission-grading-panel.tsx](../../clients/web/src/components/annotation/submission-grading-panel.tsx), [courses-api.ts](../../clients/web/src/lib/courses-api.ts), [gradingagent/service.go](../../server/internal/service/gradingagent/service.go), [gradingagent/prompt.go](../../server/internal/service/gradingagent/prompt.go), [gradingagent/scoring.go](../../server/internal/service/gradingagent/scoring.go), [grading_agent_http.go](../../server/internal/httpserver/grading_agent_http.go), [courses_routes.go](../../server/internal/httpserver/courses_routes.go), [migration 290_grading_agent.sql](../../server/migrations/290_grading_agent.sql), [migration 291_grader_agent_model.sql](../../server/migrations/291_grader_agent_model.sql), [rubric-grade-picker.tsx](../../clients/web/src/components/grading/rubric-grade-picker.tsx).
- External: [React Flow (`@xyflow/react`) docs](https://reactflow.dev/), [React Flow accessibility](https://reactflow.dev/learn/advanced-use/accessibility), WCAG 2.1 AA, OWASP LLM Top 10 (LLM01 Prompt Injection), GDPR Art. 22.
- Related plans: [19.3 — AI-Assisted Grading](19-ai-capabilities/19.3-ai-assisted-grading.md), [19.10 — Model Governance](19-ai-capabilities/19.10-model-governance.md), [19.11 — PII Redaction Proxy](19-ai-capabilities/19.11-pii-redaction-proxy.md), [19.13 — Eval Harness](19-ai-capabilities/19.13-eval-harness.md), [19.14 — Cost & Usage Controls](19-ai-capabilities/19.14-cost-usage-controls.md).
