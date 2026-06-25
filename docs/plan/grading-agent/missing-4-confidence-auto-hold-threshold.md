# GA-M4 — Agent-level confidence auto-hold threshold

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-M4 |
| **Section** | Grading Agent — Missing Features |
| **Severity** | MAJOR |
| **Markets** | HE / K12 / SL |
| **Status (today)** | THIN |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | [GA-M1](missing-1-persistent-review-queue.md) |
| **Unblocks** | confident auto-apply |

## 1. Problem Statement

To hold low-confidence grades for human review, an instructor must drop a **Human Review Gate** node
onto the canvas and wire it on every grade path. The data model already anticipated a simpler control —
`grading_agent_configs.confidence_floor NUMERIC` exists and is read into `ConfigRow.ConfidenceFloor` —
but it is **never written and never used** anywhere in the codebase. The common ask ("auto-apply grades
the model is confident about; send everything below 80% to me") therefore requires graph surgery that
most instructors will not do. This both under-delivers the obvious feature and leaves a dead column.

## 2. Goals

- A single per-agent setting: "Hold for review when confidence is below X%."
- Wire the existing `confidence_floor` column end-to-end (config API → consumer → review queue).
- Make it composable with suggest-only mode and the Human Review Gate (gate still wins when stricter).

## 3. Non-Goals

- Replacing the Human Review Gate node (it stays for per-branch logic).
- Inventing a new confidence metric — reuse the model confidence already returned in `GradeOutput.Confidence`.

## 4. Personas & User Stories

- **As an instructor**, I want to auto-apply confident grades and review the rest, so that I save time without losing oversight.
- **As a TA**, I want a course-wide default floor, so that every agent behaves consistently.

## 5. Functional Requirements

- **FR-1.** The agent config MUST expose `confidenceFloor` (0–1, nullable) via GET/PUT, persisted by `UpsertConfig`.
- **FR-2.** At write time, the consumer MUST hold (record `suggested` + `held_reason`) any item whose `Confidence < confidenceFloor` instead of applying it.
- **FR-3.** The floor MUST apply in both apply-mode batch runs and auto-grade-on-submission.
- **FR-4.** When a Human Review Gate also holds, the stricter outcome wins (gate hold OR floor hold ⇒ held); reasons are combined.
- **FR-5.** A null/zero floor MUST preserve today's behavior (no extra holding).
- **FR-6.** The held reason MUST state the threshold and the observed confidence (e.g., "Confidence 0.62 < floor 0.80").

## 6. Non-Functional Requirements

- **Performance** — pure comparison; negligible.
- **Security** — same RBAC as config edits.
- **Privacy & Compliance** — held items are not written to the gradebook, so low-confidence AI grades never reach students unreviewed.
- **Accessibility** — slider/number input with label, min/max, and step; value echoed as %.
- **Reliability** — deterministic; covered by unit tests on the boundary.
- **Observability** — metric: % of items auto-held by floor per run.
- **Internationalization** — `gradingAgent.settings.confidenceFloor.*`.
- **Backward compatibility** — additive; default null.

## 7. Acceptance Criteria

- **AC-1.** *Given* floor 0.8 and an item at 0.62, *when* the agent runs, *then* the item is held with a threshold reason and no grade is written.
- **AC-2.** *Given* floor 0.8 and an item at 0.91, *when* the agent runs, *then* the grade is applied.
- **AC-3.** *Given* floor null, *when* the agent runs, *then* behavior is unchanged from today.
- **AC-4.** *Given* a gate that holds at 0.9 and a floor at 0.8, *when* an item is at 0.85, *then* it is held (gate stricter).
- **AC-5.** *Given* I set the floor in the UI and reload, *then* the value persists.

## 8. Data Model

- Reuse `grading_agent_configs.confidence_floor` (exists; `CHECK (confidence_floor BETWEEN 0 AND 1)`).
- Add `confidence_floor` to `UpsertConfig` INSERT/UPDATE column list (today it is omitted, so it stays default/null).
- No migration needed unless a course-wide default is added (`courses` or course settings).

## 9. API Surface

- `GET …/grader-agent` config response gains `confidenceFloor`.
- `PUT …/grader-agent` body gains `confidenceFloor` (nullable number); validated 0–1.
- `graderAgentConfigToJSON` includes `confidenceFloor` (currently omitted).

## 10. UI / UX

- Inspector / agent settings: a "Hold for review below __%" control with helper text and a "never hold" (off) state.
- Surfaced near the run popover so the instructor sees the active floor before running.
- Copy/i18n under `gradingAgent.settings.confidenceFloor.*`.

## 11. AI / ML Considerations

- Confidence comes from `ParseAndClampModelOutput` / `GradeOutput.Confidence`. Document that confidence is model-reported and should be treated as a heuristic, not a calibrated probability.

## 12. Integration Points

- `server/internal/repos/gradingagent/repo.go` (`UpsertConfigInput.ConfidenceFloor`, INSERT/UPDATE).
- `server/internal/httpserver/grading_agent_http.go` (`putGraderAgentConfigBody`, `graderAgentConfigToJSON`).
- `server/internal/httpserver/grading_agent_queue.go` (hold decision before `UpsertCellWithFlags`).
- `server/internal/service/gradingagent/gate_hold.go` (compose with gate hold).
- `clients/web/src/components/annotation/grader-agent/inspector-panel.tsx` (or a config panel).

## 13. Dependencies & Sequencing

- Needs [GA-M1](missing-1-persistent-review-queue.md) so auto-held items are reviewable.
- Complements [GA-M3](missing-3-suggest-only-batch-and-bulk-review.md) (suggest-only is the floor=1 extreme).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Users over-trust model confidence | M | M | Helper text framing it as a heuristic; recommend conservative defaults |
| Floor + gate interaction confusing | L | M | Combine reasons explicitly; document "stricter wins" |

## 15. Rollout Plan

- Flag: none required (additive, default null). Optional `graderAgentConfidenceFloor` for staged exposure.
- Sequence: repo write → API → consumer hold → UI.
- Rollback: set floor null.

## 16. Test Plan

- **Unit** — boundary tests at, just below, and just above the floor; compose-with-gate matrix; null floor no-op.
- **Integration** — config persists; held vs applied write paths.
- **E2E** — set floor, run, verify split into applied vs review queue.

## 17. Documentation & Training

- Help-center: "Auto-applying confident grades and reviewing the rest."

## 18. Open Questions

1. Add a course-wide default floor (inherited by new agents)?
2. Should confidence be surfaced per item in the review queue sort (highest-uncertainty first)?

## 19. References

- `server/internal/repos/gradingagent/repo.go` (`ConfigRow.ConfidenceFloor`, `UpsertConfig`).
- `server/internal/service/gradingagent/gate_hold.go` (`EvaluateHoldDecision`, `gateConfidenceFloorFromNode`).
- `server/internal/httpserver/grading_agent_queue.go` (apply vs hold).
- Related: [GA-M1](missing-1-persistent-review-queue.md), [GA-M3](missing-3-suggest-only-batch-and-bulk-review.md).
