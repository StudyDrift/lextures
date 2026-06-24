# 19.17.3 — Criterion Grader Node (Per-Criterion Scoring)

> Implementation plan. Source: [docs/MISSING_FEATURES.md](../../MISSING_FEATURES.md) §19. Extends the grading-agent canvas ([19.17](../auto-grader-agent.md)). See the [node catalog](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | 19.17.3 |
| **Section** | AI-Specific Capabilities → Grading Agent Canvas |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | AI / Grading team |
| **Depends on** | 19.16, 19.17, [19.17.1 Rubric](node-rubric.md) |
| **Unblocks** | [19.17.4 Score Aggregator](node-score-aggregator.md), high-accuracy rubric grading |

---

## 1. Problem Statement

The single AI node grades a whole submission against the whole rubric in one call. For multi-criterion rubrics this hurts accuracy (the model spreads attention thinly, "anchors" the total, and rationales get terse) and gives the instructor no way to tune one criterion without disturbing the rest. Graders want to score **one criterion at a time** — sometimes with a different prompt or model per criterion (e.g., a cheap model for "formatting," a strong model for "argument quality") — and then recombine. This node grades exactly one rubric criterion and emits a partial grade, designed to fan out (one node per criterion) into a [Score Aggregator](node-score-aggregator.md).

## 2. Goals

- Grade a single rubric criterion in isolation, with a focused prompt and per-node model selection.
- Emit a partial `grade` whose `RubricScores` contains exactly the one criterion's score, plus a per-criterion `comments`.
- Compose cleanly: many Criterion Graders → one Score Aggregator → Student Grade.
- Reuse the existing scoring/parse/clamp pipeline; constrain it to one criterion.

## 3. Non-Goals

- Replacing the whole-rubric AI node — both coexist; this is for instructors who want granular control.
- Inventing new score math (aggregation is the [Score Aggregator](node-score-aggregator.md)'s job).
- Automatic criterion discovery — the instructor binds each node to a criterion explicitly.

## 4. Personas & User Stories

- **As an HE instructor with a 6-criterion essay rubric**, I want one grader per criterion so each score has a focused rationale and I can fix a misbehaving criterion in isolation.
- **As a cost-conscious instructor**, I want a cheap model for objective criteria (citations present?) and a strong model for subjective ones (argument quality).
- **As a TA**, I want to see per-criterion confidence so I know which criteria to spot-check.

## 5. Functional Requirements

- **FR-1.** The palette MUST offer a **Criterion Grader** node under the Processing group.
- **FR-2.** The node MUST accept `submission`, `content`, and `rubric` inputs (same typing as the legacy grader) plus optional `reference`.
- **FR-3.** The inspector MUST require selecting **one criterion** from the wired rubric (`data.criterionId`); the dropdown is populated from the rubric on the wired Rubric/Activity node.
- **FR-4.** The node MUST emit a `grade` output whose `RubricScores` map contains only the bound criterion and a `comments` output scoped to that criterion.
- **FR-5.** The model call MUST request a single-criterion score constrained to that criterion's allowed level points, parsed/clamped via the existing pipeline (`snapScoreToRubricLevel`, `ValidateRubricScoresForGrade` for the single criterion).
- **FR-6.** The node MUST support per-node `prompt` (with `$Node.Property` variables) and `modelId`, independent of other graders.
- **FR-7.** A Criterion Grader `grade` output MUST be wireable into a [Score Aggregator](node-score-aggregator.md) input and (for a single-criterion rubric) directly into the Student Grade `grade` slot.
- **FR-8.** Validation MUST flag a node whose `criterionId` is absent from the wired rubric.

## 6. Non-Functional Requirements

- **Performance** — One LLM call per criterion; fan-out runs concurrently in dry run and batch (bounded by a per-run concurrency cap) so wall-clock stays near a single call for typical rubrics.
- **Security** — Submission still untrusted; same gateway/PII path as the AI node.
- **Privacy & Compliance** — Same FERPA/automated-decision posture as 19.16; per-criterion outputs are intermediate, not separately persisted as grades.
- **Accessibility** — Criterion dropdown labelled; per-criterion result announced.
- **Scalability** — Fan-out multiplies LLM calls; cost meter warns when criterion-grader count × roster is large.
- **Reliability** — A failed single criterion isolates to that node; the aggregator decides how to treat a missing criterion (configurable).
- **Observability** — `grader_agent_criterion_calls_total{criterion}`; per-criterion tokens in `ai_usage_log`.
- **Maintainability** — Shares `scoring.go` / `ai_prompt.go`; criterion constraint is a thin wrapper over the existing system prompt.
- **Internationalization** — Per-criterion comments in instructor language.
- **Backward compatibility** — Additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a rubric with criteria A/B/C and three Criterion Graders bound to each, *When* dry-run executes, *Then* each node's output contains exactly its own criterion score and rationale.
- **AC-2.** *Given* a Criterion Grader bound to criterion A, *When* the model returns a score off the allowed levels, *Then* it is snapped to the nearest allowed level for A.
- **AC-3.** *Given* a single-criterion rubric and one Criterion Grader wired directly to Student Grade, *When* dry-run executes, *Then* the total equals that criterion's score.
- **AC-4.** *Given* a node whose `criterionId` is not in the wired rubric, *When* the graph validates, *Then* a node-level issue blocks the run.
- **AC-5.** *Given* two Criterion Graders with different `modelId`s, *When* dry-run executes, *Then* each call uses its node's model (verified via per-node logs/usage).

## 8. Data Model

No new tables. Node `data` in `workflow_graph`:

```jsonc
{
  "criterionId": "uuid",   // required: which rubric criterion this node scores
  "prompt": "string",      // criterion-focused instructions, supports $Node.Property
  "modelId": "string|null" // optional per-node model override
}
```

Per-criterion results are intermediate `slotValue`s; only the final assembled grade is persisted as a grading-agent result (existing `assessment.grading_agent_results`, migration [290](../../../server/migrations/290_grading_agent.sql)).

## 9. API Surface

- No new routes; carried by the dry-run WS and config PUT.
- Dry-run streams `node_start`/`node_complete` per criterion grader with compiled prompt/output (reusing the AI node's compiled-prompt fields in `DryRunEvent`).
- OpenAPI: node `data` schema.

## 10. UI / UX

- **Palette** — "Criterion Grader" in `groupProcessing` (indigo, grader-family styling).
- **Node body** — Title showing the bound criterion name; `submission`/`content`/`rubric` input slots; `grade`/`comments` output slots; execution badge.
- **Inspector** — Criterion dropdown (from wired rubric), prompt editor with variables, model picker, compiled-prompt preview after dry run (reuse [`AiNodeCompiledPrompt`](../../../clients/web/src/components/annotation/grader-agent/ai-node-compiled-prompt.tsx)).
- **States** — No rubric wired (disable criterion picker with hint), unbound criterion (error), running/success/error badges.
- **Mobile** — Inspector stacks.
- **Copy & i18n** — `gradingAgent.canvas.palette.criterionGrader`, `gradingAgent.canvas.nodes.criterionGrader.*`, `gradingAgent.canvas.inspector.criterion*`.

## 11. AI / ML Considerations

- **Prompt** — A single-criterion variant of [`buildAiSystemPrompt`](../../../clients/web/src/components/annotation/grader-agent/ai-output-system-prompt.ts): schema reduced to `{ score, rationale, confidence }` for one criterion; the criterion title, description, and allowed level points injected; submission still wrapped as untrusted.
- **Eval** — Compare per-criterion fan-out vs single-call grading on the §19.13 golden set; target equal-or-better Cohen's κ per criterion and lower variance in rationales.
- **Cost** — N calls per submission; surfaced in the cost meter and `ai_usage_log`; recommend pairing cheap/strong models per criterion.

## 12. Integration Points

- **Client** — `types.ts` (`PaletteNodeType` += `'criterionGrader'`), `node-palette.tsx`, `workflow-nodes.tsx` (`CriterionGraderNode`), `workflow-node-types.ts`, `validation.ts`, `inspector-panel.tsx` (criterion dropdown sourced from wired rubric), `ai-node-output-format.tsx`.
- **Server** — `workflow.go` (`NodeTypeCriterionGrader`, edge typing like grader; criterion-binding validation), `workflow_execute.go` (new case builds a single-criterion `ScoreRequest`, stores `slotValue{grade}` with one-entry `RubricScores`), `scoring.go`/`ai_prompt.go` (single-criterion system prompt).
- **Rubric** — criterion metadata from [assignmentrubric](../../../server/internal/models/assignmentrubric).

## 13. Dependencies & Sequencing

- **After**: 19.16, 19.17, and ideally [Rubric](node-rubric.md) (to supply criteria without an Activity).
- **Before**: [Score Aggregator](node-score-aggregator.md) is its natural consumer (build them together).
- **Shared infra**: existing scoring service, gateway, usage logging.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Fan-out cost balloons on large rosters | M | H | Cost meter, per-run concurrency + item caps (§19.14), recommend whole-rubric AI node when criteria are few |
| Instructor forgets to aggregate, wiring one criterion to the grade slot | M | M | Validation hint: "grade slot has only 1 of N criteria — add a Score Aggregator?" |
| Criterion IDs drift if rubric changes | M | M | Re-validate `criterionId` against the live rubric on open; surface stale bindings |
| Inconsistent scales across criteria misread by aggregator | L | M | Aggregator normalizes per criterion; documented |

## 15. Rollout Plan

- Behind `grader_agent_enabled`; build alongside Score Aggregator.
- Sequencing: types/palette/validation → server single-criterion scoring → inspector → i18n → eval gate.
- Phase 1: HE rubric-heavy courses. GA after per-criterion eval ≥ whole-rubric baseline.
- Rollback: remove palette items behind flag.

## 16. Test Plan

- **Unit** — Single-criterion `ScoreRequest` build; parse/clamp to one criterion's levels; validation of `criterionId` membership; edge typing.
- **Integration** — Three-criterion fan-out dry run; per-node model selection honored; failed-criterion isolation.
- **E2E** — Wire Rubric → three Criterion Graders → Aggregator → Student Grade; dry run; verify per-criterion breakdown.
- **AI eval** — Per-criterion κ vs baseline; injection suite.
- **Accessibility** — axe; keyboard criterion selection.

## 17. Documentation & Training

- Help center: "Grading criterion by criterion."
- Instructor guide: when fan-out helps; mixing models per criterion; cost trade-offs.
- API reference: node `data` schema.

## 18. Open Questions

1. Should a Criterion Grader auto-create from "explode this rubric into per-criterion graders" one-click action? (Strong UX win — fast-follow.)
2. How should the aggregator treat a criterion that errored — zero, skip-and-renormalize, or fail the item? (Default configurable on the aggregator; see [Score Aggregator](node-score-aggregator.md).)

## 19. References

- [workflow-nodes.tsx](../../../clients/web/src/components/annotation/grader-agent/workflow-nodes.tsx), [ai-output-system-prompt.ts](../../../clients/web/src/components/annotation/grader-agent/ai-output-system-prompt.ts), [ai-node-compiled-prompt.tsx](../../../clients/web/src/components/annotation/grader-agent/ai-node-compiled-prompt.tsx), [validation.ts](../../../clients/web/src/components/annotation/grader-agent/validation.ts).
- Server: [workflow.go](../../../server/internal/service/gradingagent/workflow.go), [workflow_execute.go](../../../server/internal/service/gradingagent/workflow_execute.go), [scoring.go](../../../server/internal/service/gradingagent/scoring.go), [ai_prompt.go](../../../server/internal/service/gradingagent/ai_prompt.go).
- Related: [node catalog](README.md), [Rubric](node-rubric.md), [Score Aggregator](node-score-aggregator.md).
