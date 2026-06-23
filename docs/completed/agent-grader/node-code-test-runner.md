# 19.17.7 — Code Test Runner Node (Autograde Code Submissions)

> Implementation plan. Source: [docs/MISSING_FEATURES.md](../../MISSING_FEATURES.md) §19. Extends the grading-agent canvas ([19.17](../auto-grader-agent.md)). See the [node catalog](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | 19.17.7 |
| **Section** | AI-Specific Capabilities → Grading Agent Canvas |
| **Severity** | MAJOR (CS / programming courses) |
| **Markets** | HE / K12 (CS) / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | AI / Grading team + Code Execution team |
| **Depends on** | 19.16, 19.17, [2.4 code-execution questions](../../completed/02-assessment-and-authoring/2.4-code-execution-questions.md) |
| **Unblocks** | Programming autograding, AI + tests blended scoring |

---

## 1. Problem Statement

[19.16 §3](../auto-grader-agent.md) explicitly excludes code-execution submissions from the LLM grader's v1 — and rightly so: code should be graded by **running it against test cases**, not by asking a model to guess whether it works. Lextures already has a sandboxed [code-execution service](../../../server/internal/service/codeexecution/service.go) ([2.4](../../completed/02-assessment-and-authoring/2.4-code-execution-questions.md)). This node brings deterministic autograding onto the canvas: it executes a code submission against an instructor test suite in the sandbox and emits a `grade` derived from the pass rate plus a `report` of test results — which can stand alone or be blended with an AI "code style/quality" judgement via the [Score Aggregator](node-score-aggregator.md).

## 2. Goals

- Run a code submission against an instructor-defined test suite in the existing sandbox.
- Map results to a `grade` (configurable pass-rate → points) and a `report`/`comments` of per-test outcomes.
- Compose with AI grading: e.g., `0.7 × tests + 0.3 × AI style review` via the aggregator.
- Inherit the sandbox's safety guarantees (isolation, resource caps, no network).

## 3. Non-Goals

- Building a new execution sandbox — reuse [codeexecution](../../../server/internal/service/codeexecution/service.go).
- Authoring test suites here — reuse the 2.4 test-case authoring; this node *selects/references* a suite.
- Grading non-code submissions (those stay on the AI/grader path).
- Partial-credit static analysis beyond what the test suite expresses (a future "Static Analysis" node).

## 4. Personas & User Stories

- **As a CS instructor**, I want students' functions run against my unit tests and scored by pass rate, so that grading is objective and instant.
- **As a CS instructor**, I want test pass-rate combined with an AI review of code style, so that the grade reflects correctness *and* quality.
- **As a TA**, I want failing-test output attached as feedback so students see exactly what broke.
- **As a self-learner**, I want immediate autograded feedback on practice problems.

## 5. Functional Requirements

- **FR-1.** The palette MUST offer a **Code Test Runner** node under the Processing group (visible where code-execution is enabled for the tenant).
- **FR-2.** The node MUST accept a `submission` input and expose `grade` and `report` outputs (and optional `score` = raw pass-rate).
- **FR-3.** The inspector MUST let the instructor select a test suite (referencing the assignment's 2.4 suite or a chosen one), language/runtime, and a pass-rate → points mapping (e.g., linear, all-or-nothing, weighted per test).
- **FR-4.** Execution MUST run the submission in the sandbox with enforced time/memory/output caps and **no network**, collecting per-test pass/fail, stdout/stderr (truncated), and timing.
- **FR-5.** The node MUST map results to a `GradeOutput` (total points, confidence = 1.0 for deterministic tests) and a `report` summarizing passed/failed tests.
- **FR-6.** Sandbox failures (compile error, timeout, crash) MUST be represented as a graded outcome per policy (e.g., compile error → 0 with the compiler message in the report), never a fabricated score.
- **FR-7.** The `grade` output MUST be wireable to the Student Grade slot or a [Score Aggregator](node-score-aggregator.md); `report` to comments / AI input / [Flag for Review](node-flag-for-review.md).
- **FR-8.** Student code MUST never be executed with elevated privileges or access to other students' work.

## 6. Non-Functional Requirements

- **Performance** — Per-submission execution bounded by the sandbox's wall-clock/resource caps; batch runs respect a per-course concurrency cap; dry run executes the open submission inline within the sandbox timeout.
- **Security** — Untrusted code in an isolated sandbox: no network, read-only FS except a scratch dir, CPU/mem/time/output limits, killed on overrun. Inherits [2.4](../../completed/02-assessment-and-authoring/2.4-code-execution-questions.md) threat model.
- **Privacy & Compliance** — Submission is a FERPA record; execution logs are org-private and truncated; no code leaves the tenant.
- **Accessibility** — Suite picker, mapping editor, and result report all keyboard-navigable; report uses semantic pass/fail markup.
- **Scalability** — Sandbox capacity is the bottleneck; queue + concurrency cap; large suites time-boxed.
- **Reliability** — Deterministic given the same code + suite; idempotent per `(run_id, submission_id)`; sandbox errors isolated per item.
- **Observability** — `grader_agent_codetests_total{result}`, `grader_agent_codetest_latency_ms`, sandbox resource metrics.
- **Maintainability** — Thin adapter over the codeexecution service; mapping logic is a pure module.
- **Internationalization** — Report labels localized; compiler output passed through verbatim.
- **Backward compatibility** — Additive; gated by the code-execution tenant capability.

## 7. Acceptance Criteria

- **AC-1.** *Given* a Python submission passing 8/10 tests with a linear mapping over 10 points, *When* dry-run executes, *Then* the grade is 8 and the report lists the 2 failures with output.
- **AC-2.** *Given* a submission that fails to compile, *When* executed, *Then* the grade is 0 (per policy) and the report contains the compiler error — no fabricated score.
- **AC-3.** *Given* code that loops forever, *When* executed, *Then* it is killed at the timeout and scored per the timeout policy with a clear report.
- **AC-4.** *Given* tests pass-rate wired into a `0.7/0.3` aggregator with an AI style score, *When* executed, *Then* the blended total matches the weighted formula.
- **AC-5.** *Given* code attempting a network call, *When* executed, *Then* the call fails (no network) and the run completes safely.
- **AC-6.** *Given* the tenant lacks code-execution, *When* opening the palette, *Then* the node is absent/disabled with a tooltip.

## 8. Data Model

No new grading-agent tables. Node `data`:

```jsonc
{
  "testSuiteId": "uuid",          // references a 2.4 test suite
  "runtime": "python3.12" | "...",
  "mapping": {
    "type": "linear" | "allOrNothing" | "weighted",
    "maxPoints": 10,
    "weights": { "<testId>": 1.0 } // weighted only
  },
  "onCompileError": "zero" | "failItem",
  "onTimeout": "zero" | "partial" | "failItem"
}
```

Execution artifacts (per-test results) are intermediate; the final grade persists via existing `assessment.grading_agent_results`. Sandbox logs follow the 2.4 retention policy.

## 9. API Surface

- No new grading-agent routes; invokes the codeexecution service interface.
- Reuses the sandbox execution entry point from [codeexecution/service.go](../../../server/internal/service/codeexecution/service.go).
- Dry-run streams per-test progress as `log` events and a final `node_complete`.
- OpenAPI: node `data` schema.

## 10. UI / UX

- **Palette** — "Code Test Runner" in `groupProcessing` (cyan/teal, autograder styling), shown only when code-execution is enabled.
- **Node body** — Title; `submission` input; `grade`/`report` outputs; execution badge.
- **Inspector** — Test-suite picker, runtime selector, pass-rate→points mapping editor, compile/timeout policy selectors, and (after dry run) a per-test results table.
- **States** — No suite selected (hint), running (live test progress), error (sandbox unavailable), compile/timeout outcomes.
- **Mobile** — Results table scrolls.
- **Copy & i18n** — `gradingAgent.canvas.palette.codeTests`, `gradingAgent.canvas.nodes.codeTests.*`, `gradingAgent.canvas.inspector.codeTests*`.

## 11. AI / ML Considerations

No LLM in this node — it exists precisely to keep correctness grading deterministic. It *complements* AI: pair it with an AI node that reviews **style/quality/readability** (fed the code as untrusted content) and blend via the aggregator. The AI side inherits the standard untrusted-content framing; the test results are trusted, system-generated facts.

## 12. Integration Points

- **Client** — `types.ts` (`PaletteNodeType` += `'codeTestRunner'`, reuse `grade`/`report` handles), `node-palette.tsx` (capability-gated), `workflow-nodes.tsx` (`CodeTestRunnerNode`), `workflow-node-types.ts`, `validation.ts`, `inspector-panel.tsx`.
- **Server** — `workflow.go` (`NodeTypeCodeTestRunner`, edge typing), `workflow_execute.go` (new case calls the sandbox, maps results to `slotValue{grade, report}`), adapter to [codeexecution/service.go](../../../server/internal/service/codeexecution/service.go), a pure `passrate_mapping.go` module.
- **Cross-plan** — [2.4 code-execution](../../completed/02-assessment-and-authoring/2.4-code-execution-questions.md) (suites + sandbox), [Score Aggregator](node-score-aggregator.md).

## 13. Dependencies & Sequencing

- **After**: 19.16, 19.17, 2.4 (sandbox + test suites).
- **Before**: nothing hard; best paired with [Score Aggregator](node-score-aggregator.md) for blended grading.
- **Shared infra**: code-execution sandbox + queue concurrency caps.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Sandbox escape / abuse via student code | L | H | Reuse 2.4 isolation: no network, resource/time caps, dropped privileges, killed on overrun |
| Flaky tests (timing/nondeterminism) misgrade | M | M | Recommend deterministic tests; per-test timeout; optional re-run-on-flaky policy |
| Sandbox capacity starves batch grading | M | M | Per-course concurrency cap; queue backpressure; time-boxed suites |
| Compile/timeout policy surprises students | M | M | Explicit policy selectors + report transparency + provisional/editable grade |

## 15. Rollout Plan

- Behind `grader_agent_enabled` **and** the code-execution tenant capability.
- Sequencing: capability gate + types/palette → sandbox adapter + mapping → execution + dry-run progress → inspector → i18n.
- Phase 1: CS pilot courses, tests-only grading. Phase 2: tests + AI style blend.
- Rollback: hide palette item behind flag/capability; queued jobs drain.

## 16. Test Plan

- **Unit** — Pass-rate→points mappings (linear/all-or-nothing/weighted); compile/timeout policies; report formatting.
- **Integration** — Run a known suite vs known submissions (full/partial/zero); compile error; infinite loop timeout; network-denied.
- **E2E** — Code Test Runner → aggregator (+AI style) → Student Grade; verify blended grade and report feedback.
- **Security** — Sandbox isolation suite (network/FS/privilege/resource); cross-student access denied.
- **Performance** — Concurrency cap under batch load; latency targets.
- **Accessibility** — axe; keyboard suite/mapping; SR-readable results table.

## 17. Documentation & Training

- Help center: "Autograding code with test cases."
- Instructor guide: writing deterministic tests; pass-rate mappings; blending with AI style review.
- Runbook: sandbox capacity/queue monitoring, flaky-test triage.

## 18. Open Questions

1. Reference an assignment's existing 2.4 suite only, or allow a node-local inline suite? (Plan: reference first; inline as fast-follow.)
2. Expose per-test partial credit directly to the rubric, or only an aggregate pass-rate? (Plan: aggregate in v1; per-test→criterion mapping later.)
3. Should style-review AI see test results as context? (Optional wiring; default off to avoid anchoring.)

## 19. References

- Server: [codeexecution/service.go](../../../server/internal/service/codeexecution/service.go), [workflow.go](../../../server/internal/service/gradingagent/workflow.go), [workflow_execute.go](../../../server/internal/service/gradingagent/workflow_execute.go).
- Client: [node-palette.tsx](../../../clients/web/src/components/annotation/grader-agent/node-palette.tsx), [validation.ts](../../../clients/web/src/components/annotation/grader-agent/validation.ts).
- Related: [node catalog](README.md), [2.4 code-execution questions](../../completed/02-assessment-and-authoring/2.4-code-execution-questions.md), [Score Aggregator](node-score-aggregator.md), [Flag for Review](node-flag-for-review.md).
