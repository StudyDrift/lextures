# Grading Agent — Audit & Remediation Plans

> Findings from a full review of the **grading (grader) agent**: the workflow-canvas auto-grader
> that scores assignment submissions with an instructor-authored node graph. Each finding below is a
> standalone plan that follows [`_TEMPLATE.md`](../_TEMPLATE.md).
>
> Companion folder [`../agent-grader/`](../agent-grader/) plans *new canvas nodes*. This folder is
> scoped to **usability gaps, simplifications, and bugs** in the agent as it exists today.

## What the agent is (as built)

- **Authoring** — a React Flow DAG in SpeedGrader (`clients/web/src/components/annotation/grader-agent/`)
  with a typed palette (Student Submission, Activity, Rubric, Reference, AI, Criterion Grader,
  Code Test Runner, Conditional Router, Score Aggregator, Originality, Human Review Gate,
  Flag for Review, Student Grade output).
- **Validation/compile** — client `validation.ts` + server `workflow.go` (`ValidateWorkflowGraph`,
  `CompileWorkflowGraph`).
- **Execution** — `workflow_execute.go::ExecuteWorkflowDryRun` topologically walks the graph. The
  **same engine runs both dry runs (WebSocket) and live batch/auto grading**
  (`background/grading_agent_consumer.go` → `HandleGradingAgentQueueMessage`).
- **Persistence** — `assessment.grading_agent_configs / _runs / _results` via
  `repos/gradingagent/repo.go`. Batch jobs flow through `gradingagentqueue` (RabbitMQ or in-memory).

## Findings index

### Missing features (HE instructor / TA / grader usability)

| ID | Plan | Severity | One-liner |
|---|---|---|---|
| GA-M1 | [Persistent, actionable review queue & run history](../../completed/grading-agent/missing-1-persistent-review-queue.md) | BLOCKER | **COMPLETE** — Persistent review inbox, flagged actions, run history, and review counts. |
| GA-M2 | [Grade non-file (online text-entry) & image/scanned submissions](../../completed/grading-agent/missing-2-non-file-submission-grading.md) | BLOCKER | **COMPLETE** — Text-entry body grading, vision path for scanned/image submissions, per-submission failure reasons. |
| GA-M3 | [Suggest-only batch + bulk review/apply + posting control](../../completed/grading-agent/missing-3-suggest-only-batch-and-bulk-review.md) | MAJOR | **COMPLETE** — Suggest-only runs, bulk approve/reject, and posting control. |
| GA-M4 | [Agent-level confidence auto-hold threshold](../../completed/grading-agent/missing-4-confidence-auto-hold-threshold.md) | MAJOR | **COMPLETE** — Per-agent confidence floor wired end-to-end; composes with Human Review Gate. |
| GA-M5 | [Section / group / student-scoped runs](../../completed/grading-agent/missing-5-section-scoped-runs.md) | MAJOR | **COMPLETE** — Section, group, and explicit submission filters on batch runs with server-side visibility enforcement. |
| GA-M6 | [Cancel / stop a running batch](missing-6-cancel-running-batch.md) | MAJOR | Once a batch starts it cannot be stopped; a mistake burns the whole class of AI calls. |
| GA-M7 | [Pre-run cost & scope estimate + run cost summary](missing-7-cost-estimate-and-budget.md) | MAJOR | No cost/size preview before running and no aggregate cost after; instructors fear runaway spend. |

### Over-complexities to simplify

| ID | Plan | One-liner |
|---|---|---|
| GA-S1 | [Unify the three duplicated grade-write paths in the consumer](../../completed/grading-agent/simplify-1-unify-grade-write-paths.md) | **Done** — single execute + persist path in the queue consumer. |
| GA-S2 | [Remove the dead HTTP dry-run path; rename the execution engine](simplify-2-remove-dead-dry-run-and-rename-engine.md) | `POST /dry-run` + `Service.Score` legacy path is unused by the client and mis-handles complex graphs. |
| GA-S3 | [Collapse the duplicated per-node update callbacks](simplify-3-generic-node-data-updater.md) | ~12 near-identical `updateXNode` callbacks in the hook can be one generic updater. |
| GA-S4 | [Retire legacy node-type aliases & palette ternaries](../completed/grading-agent/simplify-4-legacy-node-type-aliases.md) | `submission`/`assignmentContext`/`grader` legacy types and giant nested ternaries add accidental complexity. |

### Bugs

| ID | Plan | Size | One-liner |
|---|---|---|---|
| GA-B1 | [In-memory queue overflow & stuck runs](bug-1-queue-overflow-and-stuck-runs.md) | Large | A class > 128 submissions (or a slow consumer) overflows the in-memory queue and leaves the run stuck in `running`. |
| GA-B2 | [Requeue causes double grading & duplicate results](bug-2-requeue-double-grading.md) | Medium | A transient DB error after the grade is written requeues the message → second LLM call, duplicate result, inflated counts. |
| GA-B3 | [Auto-post / confidence_floor dead code](bug-3-auto-post-dead-code.md) | Medium | `post_policy` is never writable, so AI grades never auto-post even when the assignment posts automatically. |
| GA-B4 | [HTTP dry-run mis-previews complex graphs](bug-4-post-dry-run-mispreviews-graphs.md) | Medium | The non-WS dry-run endpoint ignores routers/gates/aggregators/flags and returns a wrong (or failing) preview. |
| GA-B5 | [Run-status polling robustness](bug-5-run-polling-robustness.md) | Small | Polling can over-fire `onApplied` and keep ticking one cycle after a run is already done. |

## Severity legend

- **BLOCKER** — most HE classes cannot adopt the agent without it.
- **MAJOR** — adoption-limiting / RFP-losing gap.
- **MINOR** — parity / nice-to-have.

## Conventions

- File naming: `{category}-{n}-{kebab-slug}.md`; Feature IDs `GA-M*`, `GA-S*`, `GA-B*`.
- Each plan fills every `_TEMPLATE.md` section and names the concrete files it touches.
- Cross-references between plans use relative links.
