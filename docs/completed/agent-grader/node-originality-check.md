# 19.17.6 — Originality Check Node (Similarity & AI-Likelihood Signal)

> Implementation plan. Source: [docs/MISSING_FEATURES.md](../../MISSING_FEATURES.md) §19. Extends the grading-agent canvas ([19.17](../auto-grader-agent.md)). See the [node catalog](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | 19.17.6 |
| **Section** | AI-Specific Capabilities → Grading Agent Canvas |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | AI / Grading team + Integrity team |
| **Depends on** | 19.16, 19.17, [3.5 plagiarism/AI detection](../../completed/03-submissions-grading-integrity/3.5-plagiarism-ai-detection.md), [3.14 originality reports](../../completed/03-submissions-grading-integrity/3.14-originality-reports-stored-with-submission.md) |
| **Unblocks** | Integrity-aware grading, similarity-driven routing |
| **Reuses shared change** | `score` / `report` / `flag` handle kinds (already shipped with the conditional-router / criterion-grader / flag-for-review nodes; see [catalog](README.md)) |

---

## 1. Problem Statement

Lextures already produces originality/similarity and AI-likelihood signals ([originality service](../../../server/internal/service/originality/), [plagiarism service](../../../server/internal/service/plagiarism/), reports stored per submission via [3.14](../../completed/03-submissions-grading-integrity/3.14-originality-reports-stored-with-submission.md)), but the grading agent is blind to them. An instructor cannot say "use the similarity score as context for the grade," "subtract a penalty when similarity is high," or "route flagged submissions to a human." This node surfaces the existing integrity signals as graph values — a numeric `score`, a human-readable `report`, and a boolean `flag` — for use as AI context, a [Conditional Router](../../completed/agent-grader/node-conditional-router.md) condition, a [Score Aggregator](node-score-aggregator.md) penalty, or a [Flag for Review](../../completed/agent-grader/node-flag-for-review.md) reason. It reuses the `score`/`report`/`flag` slot kinds already shipped by the routing, criterion-grader, and flag nodes.

## 2. Goals

- Expose the existing originality / AI-likelihood signal for the open submission as graph outputs.
- Provide three outputs: `score` (0–1 normalized similarity or AI-likelihood), `report` (summary text + link), and `flag` (boolean against a threshold).
- Let those outputs feed AI context (as **trusted signal**), router conditions, aggregator penalties, and review sinks.
- Reuse the existing `score`/`report`/`flag` slot-value kinds and `HANDLE_*` constants (already on the canvas); add only the originality-specific wiring (and a `slotValue.flag` field if not yet present).

## 3. Non-Goals

- Running a *new* plagiarism scan synchronously inside a dry run — v1 consumes the **stored** report ([3.14](../../completed/03-submissions-grading-integrity/3.14-originality-reports-stored-with-submission.md)); on-demand scans are an enhancement gated on provider latency.
- Making an integrity *decision* — this node surfaces a signal; routing/penalizing/flagging is the instructor's explicit wiring.
- Replacing the dedicated plagiarism workflow ([14.8](../../completed/14-higher-ed-specific/14.8-plagiarism-workflow.md)); this is a grading-time read of its output.

## 4. Personas & User Stories

- **As an instructor**, I want the AI grader to *see* the similarity score so its feedback can mention unoriginal passages.
- **As a TA**, I want submissions over 40% similarity routed to a human instead of auto-graded.
- **As an integrity officer**, I want a similarity-based penalty blended into the score via the aggregator, with the report attached to the feedback.
- **As a student**, I want any integrity-related grade impact disclosed and appealable (inherits 19.16 disclosure + re-grade).

## 5. Functional Requirements

- **FR-1.** The palette MUST offer an **Originality Check** node under the Processing group.
- **FR-2.** The node MUST accept a `submission` input and expose `score`, `report`, and `flag` outputs.
- **FR-3.** The inspector MUST let the instructor pick the metric (`similarity` | `aiLikelihood`) and a `flag` threshold.
- **FR-4.** Execution MUST read the **stored** originality report for the submission; if none exists, the node MUST emit a clearly-marked "no report available" state (not a fabricated score) and downstream consumers MUST handle absence gracefully.
- **FR-5.** `score` MUST be wireable into a router condition, an aggregator (as a weighted/penalty input — see interop note), and an AI input (as labelled trusted signal); `report` into AI input / flag-for-review reason / comments; `flag` into a router condition.
- **FR-6.** When `score`/`report` is fed to an AI node, it MUST be labelled a **trusted integrity signal**, never wrapped as untrusted submission content.
- **FR-7.** Any grade impact derived from this node MUST remain provisional/unposted and disclosed per [19.16](../auto-grader-agent.md) (no silent integrity penalties).
- **FR-8.** The node MUST emit on the existing `HANDLE_SCORE`, `HANDLE_REPORT`, and `HANDLE_FLAG` handles (already declared in `types.ts`/`workflow.go`). `slotValue.score` already exists; the node MUST add a `slotValue.flag` boolean field only if one is not yet carried by the execution layer.

## 6. Non-Functional Requirements

- **Performance** — Reading a stored report is a single DB/object read; no provider round-trip in the v1 path.
- **Security** — Reports accessible only to course graders; integrity vendor data stays org-private; signal feeds the model as trusted context (no injection surface from the report, which is system-generated).
- **Privacy & Compliance** — Originality reports are sensitive; FERPA education-record handling; integrity-driven grade effects are automated decisions and inherit 19.16 human-oversight + appeal. Some regions restrict AI-likelihood detectors — gate `aiLikelihood` behind tenant policy.
- **Accessibility** — Metric/threshold controls labelled; report summary readable; numeric signal announced.
- **Scalability** — Stored-report read scales with submissions; no extra provider load in v1.
- **Reliability** — Missing/expired report → explicit absence, never a guessed value.
- **Observability** — `grader_agent_originality_reads_total{metric,present}`; dry-run log `[Originality] similarity 0.32 (report 2026-06-20)`.
- **Maintainability** — Thin adapter over the originality/plagiarism services.
- **Internationalization** — Report summary localized where the provider supports it.
- **Backward compatibility** — Additive; new slot kinds are opt-in.

## 7. Acceptance Criteria

- **AC-1.** *Given* a submission with a stored 32% similarity report and an Originality node wired to an AI input, *When* dry-run executes, *Then* the compiled input contains a labelled "Integrity signal: similarity 0.32" block (trusted, not delimited as untrusted).
- **AC-2.** *Given* `flag` threshold 0.4 and a 0.55 similarity, *When* the node runs, *Then* `flag` is true and a wired [Conditional Router](../../completed/agent-grader/node-conditional-router.md) takes the integrity branch.
- **AC-3.** *Given* no stored report, *When* the node runs, *Then* it emits "no report available" and the run does not fabricate a score; downstream router treats it per its missing-field policy.
- **AC-4.** *Given* `aiLikelihood` disabled by tenant policy, *When* the instructor selects it, *Then* the option is disabled with an explanatory tooltip.
- **AC-5.** *Given* an integrity penalty applied via the aggregator, *When* the grade is produced, *Then* it is provisional/unposted and the student-facing disclosure notes integrity input.

## 8. Data Model

No new tables — reads existing originality storage ([3.14](../../completed/03-submissions-grading-integrity/3.14-originality-reports-stored-with-submission.md)). Node `data`:

```jsonc
{
  "metric": "similarity" | "aiLikelihood",
  "flagThreshold": 0.4
}
```

The `HANDLE_SCORE`/`HANDLE_REPORT`/`HANDLE_FLAG` constants already exist in both `types.ts` and `workflow.go`, and `slotValue` in [workflow_execute.go](../../../server/internal/service/gradingagent/workflow_execute.go) already carries `score *float64`. This node only adds a `flag *bool` field to `slotValue` if the execution layer does not already carry one.

## 9. API Surface

- No new grading-agent routes; reads via the originality/plagiarism service interfaces.
- Reuses the stored-report read used by [webhooks_originality.go](../../../server/internal/httpserver/webhooks_originality.go) / the originality repo.
- OpenAPI: node `data` schema; document new handle kinds.

## 10. UI / UX

- **Palette** — "Originality Check" in `groupProcessing` (rose/amber, integrity styling).
- **Node body** — Title; `submission` input; `score` / `report` / `flag` output slots with distinct dots.
- **Inspector** — Metric selector (with policy-gated `aiLikelihood`), threshold slider, and a preview of the most recent stored report (score + link) for the selected submission.
- **States** — No report (explicit empty state), policy-disabled metric, loading.
- **Mobile** — Stacked.
- **Copy & i18n** — `gradingAgent.canvas.palette.originality`, `gradingAgent.canvas.nodes.originality.*`, `gradingAgent.canvas.inspector.originality*`.

## 11. AI / ML Considerations

- **As context** — The signal is injected as a trusted, labelled block; the system prompt instructs the model to *consider* it as evidence, not to mechanically convert it to a score (scoring stays rubric/instructor driven unless the instructor wires an explicit penalty).
- **AI-likelihood caveats** — Detectors are error-prone and biased against non-native writers; copy and the model card MUST warn against using `aiLikelihood` as sole grounds for a grade; recommend routing to human review rather than auto-penalizing.
- **Cost** — No LLM cost from the node itself; only added context tokens when wired to AI.

## 12. Integration Points

- **Client** — `types.ts` (new handles + `PaletteNodeType` += `'originality'`), `node-palette.tsx`, `workflow-nodes.tsx` (`OriginalityNode`), `workflow-node-types.ts`, `validation.ts` (typing for `score`/`report`/`flag` consumers), `workflow-prompt-variable.ts` (`score`→`Score`, `report`→`Report` properties), `inspector-panel.tsx`.
- **Server** — `workflow.go` (`NodeTypeOriginality`, new handles, edge typing), `workflow_execute.go` (new case + `slotValue` extension; trusted-label in `gatherAIInput`), adapters to [originality](../../../server/internal/service/originality/) / [plagiarism](../../../server/internal/service/plagiarism/).
- **Cross-plan** — [3.5](../../completed/03-submissions-grading-integrity/3.5-plagiarism-ai-detection.md), [3.14](../../completed/03-submissions-grading-integrity/3.14-originality-reports-stored-with-submission.md), [14.8](../../completed/14-higher-ed-specific/14.8-plagiarism-workflow.md).

## 13. Dependencies & Sequencing

- **After**: 19.16, 19.17, and the originality/plagiarism services (already present).
- **Before**: makes integrity routing/penalties possible; pairs with [Conditional Router](../../completed/agent-grader/node-conditional-router.md) and [Flag for Review](../../completed/agent-grader/node-flag-for-review.md).
- **Shared infra**: reuses the `score`/`report`/`flag` slot kinds already on the canvas; no new shared change to land first.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| AI-likelihood false positives unfairly penalize students (esp. ESL) | H | H | Default to routing-to-human, not auto-penalty; policy gate; strong warning copy + model card |
| Fabricated score when no report exists | M | H | Explicit "no report" state (FR-4); never default to 0 or 1 |
| Integrity penalty applied silently | M | H | Provisional/unposted + disclosure + appeal (FR-7) |
| Stale report graded against | M | M | Show report timestamp; option to require a recent report |

## 15. Rollout Plan

- Behind `grader_agent_enabled`; `aiLikelihood` behind a tenant policy sub-flag.
- Sequencing: shared slot kinds → service adapter + execution → node/inspector → i18n.
- Phase 1: `similarity` as AI context only (no auto-penalty). Phase 2: routing + opt-in penalty with disclosure.
- Rollback: remove palette item behind flag; slot kinds inert without consumers.

## 16. Test Plan

- **Unit** — Stored-report read adapter; score normalization; flag threshold; missing-report state; new slot-kind typing in validation.
- **Integration** — Dry run injects labelled trusted signal; flag routes correctly; policy-gated metric disabled.
- **E2E** — Originality → router → Flag-for-Review path; verify integrity branch and provisional grade.
- **Security/Privacy** — Cross-course report access denied; AI-likelihood policy enforced.
- **Accessibility** — axe; keyboard metric/threshold.

## 17. Documentation & Training

- Help center: "Using originality signals in grading."
- Instructor guide: prefer routing over auto-penalty; AI-likelihood limitations; disclosure obligations.
- Model card update: integrity-signal usage and bias caveats.

## 18. Open Questions

1. Allow an on-demand scan inside dry run when no stored report exists, accepting added latency? (Defer; v1 stored-report only.)
2. Should `score` carry provider metadata (which sources matched) for richer feedback? (Leaning yes in `report`, not `score`.)
3. Standard mapping from similarity to a penalty, or leave entirely to the aggregator weights? (Plan: leave to instructor wiring; offer a documented recipe.)

## 19. References

- Server: [originality/](../../../server/internal/service/originality/), [plagiarism/](../../../server/internal/service/plagiarism/), [webhooks_originality.go](../../../server/internal/httpserver/webhooks_originality.go), [workflow.go](../../../server/internal/service/gradingagent/workflow.go), [workflow_execute.go](../../../server/internal/service/gradingagent/workflow_execute.go).
- Client: [types.ts](../../../clients/web/src/components/annotation/grader-agent/types.ts), [validation.ts](../../../clients/web/src/components/annotation/grader-agent/validation.ts), [workflow-prompt-variable.ts](../../../clients/web/src/components/annotation/grader-agent/workflow-prompt-variable.ts).
- Related: [node catalog](README.md), [Conditional Router](../../completed/agent-grader/node-conditional-router.md), [Score Aggregator](node-score-aggregator.md), [Flag for Review](../../completed/agent-grader/node-flag-for-review.md); [3.5](../../completed/03-submissions-grading-integrity/3.5-plagiarism-ai-detection.md), [3.14](../../completed/03-submissions-grading-integrity/3.14-originality-reports-stored-with-submission.md), [14.8](../../completed/14-higher-ed-specific/14.8-plagiarism-workflow.md).
