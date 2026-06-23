# 19.17.8 — Human Review Gate Node (Hold for Human Approval)

> Implementation plan. Source: [docs/MISSING_FEATURES.md](../../MISSING_FEATURES.md) §19. Extends the grading-agent canvas ([19.17](../auto-grader-agent.md)). See the [node catalog](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | 19.17.8 |
| **Section** | AI-Specific Capabilities → Grading Agent Canvas |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | AI / Grading team |
| **Depends on** | 19.16, 19.17, [19.17.5 Conditional Router](node-conditional-router.md) |
| **Unblocks** | Confidence-gated auto-grading, GDPR Art. 22 oversight on the canvas |

---

## 1. Problem Statement

[19.16](../auto-grader-agent.md) keeps humans in the loop by writing every agent grade as **unposted/provisional** and by holding sub-`confidence_floor` results for review. But on the *canvas*, that policy is implicit and global — there is no node an instructor can place to say "results that reach this point need my sign-off before they are written." Instructors and TAs want an explicit, visible checkpoint: most submissions flow straight through, but borderline or low-confidence ones pause in a review queue until a human approves or edits them. This node makes the human-in-the-loop step a first-class, placeable graph element.

## 2. Goals

- Provide a control node that conditionally **holds** an item for human approval before its grade is written.
- Support hold modes: `always`, `belowConfidence` (with a floor), and `onFlag` (driven by an upstream `flag`).
- Held items appear in a review queue; on approval (optionally edited) the grade proceeds to the Student Grade sink; on rejection it is discarded/sent back.
- Make the existing `confidence_floor` safety net (migration [290](../../../server/migrations/290_grading_agent.sql)) explicit and per-graph.

## 3. Non-Goals

- Building a general task/approvals system — reuse the SpeedGrader review surface and the existing `grading_agent_results` statuses.
- Auto-posting policy — that remains the org-gated `grader_agent_auto_post_allowed` from 19.16; this node governs *writing/holding*, not posting to students.
- Routing logic itself — conditions come from the [Conditional Router](node-conditional-router.md)/upstream `flag`; this node is the *hold*, not the decision tree.

## 4. Personas & User Stories

- **As an instructor enabling auto-grade**, I want only confident grades written automatically and the rest queued for me, so that I trust the automation without abdicating judgement.
- **As a TA**, I want a single queue of held submissions with the agent's suggestion pre-filled, so that I can approve/edit quickly.
- **As a compliance officer**, I want demonstrable human oversight of automated grading decisions (GDPR Art. 22).
- **As a student**, I want assurance a human reviewed borderline AI grades before they counted.

## 5. Functional Requirements

- **FR-1.** The palette MUST offer a **Human Review Gate** node under a Processing/Control group.
- **FR-2.** The node MUST accept `grade` (required) plus optional `comments`, `report`, and `flag` inputs, and expose a `grade` output (pass-through on approval).
- **FR-3.** The inspector MUST offer a hold mode: `always` | `belowConfidence` (numeric floor) | `onFlag`.
- **FR-4.** During a **dry run**, the node MUST pass through and log "would hold for review (mode=…, would-hold=true/false)" without persisting anything.
- **FR-5.** During **live/batch/auto** runs, when the hold condition is met the item MUST be recorded with status `suggested` (held), MUST NOT be applied/posted regardless of post policy, and MUST surface in the review queue; when not met it proceeds to the sink as a provisional grade.
- **FR-6.** Approving a held item (with optional edits) MUST write the grade via the standard grade path ([assignment_submission_grade_http.go](../../../server/internal/httpserver/assignment_submission_grade_http.go)), flagged `graded_by_ai`, and transition the result to `applied`/`overridden`.
- **FR-7.** Rejecting a held item MUST leave the student's grade unchanged and mark the result `skipped` with a reason.
- **FR-8.** Validation MUST treat the gate's `grade` output like any grade source for downstream reachability (a held branch still "reaches" a terminal — the write is deferred, not absent).

## 6. Non-Functional Requirements

- **Performance** — The gate adds negligible compute; it changes *when* a write happens, not throughput. Queue reads paginate.
- **Security** — Only course graders can view/approve the queue; approval reuses existing grade-write authz.
- **Privacy & Compliance** — This is the canvas embodiment of GDPR Art. 22 meaningful human oversight; held items are auditable; ties to the 19.16 disclosure/appeal model.
- **Accessibility** — Queue and approve/edit/reject actions keyboard-navigable; held count announced; focus management on approval.
- **Scalability** — Held items scale with low-confidence volume; queue indexed by config/assignment.
- **Reliability** — Holding is idempotent per `(run_id, submission_id)`; a crash mid-run never silently writes a held grade.
- **Observability** — `grader_agent_gate_held_total{mode}`, `grader_agent_gate_approved_total`, `grader_agent_gate_rejected_total`, time-in-queue.
- **Maintainability** — Reuses `grading_agent_results` statuses + the review surface; gate logic is thin.
- **Internationalization** — Queue/approval strings localized.
- **Backward compatibility** — Additive; without a gate, 19.16's global confidence-floor behavior is unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a gate in `belowConfidence` (floor 0.7) and an AI result at 0.5, *When* a live run executes, *Then* the item is held (`suggested`), not written, and appears in the review queue.
- **AC-2.** *Given* the same gate and a 0.9-confidence result, *When* the run executes, *Then* a provisional grade is written and the item is not queued.
- **AC-3.** *Given* a held item, *When* the instructor edits the score and approves, *Then* the grade is written via the standard path, flagged `graded_by_ai`, and the result becomes `applied`/`overridden`.
- **AC-4.** *Given* a held item, *When* the instructor rejects it, *Then* the student's grade is unchanged and the result is `skipped` with a reason.
- **AC-5.** *Given* a dry run through the gate, *When* it completes, *Then* nothing is persisted and the trace shows the would-hold decision.
- **AC-6.** *Given* `onFlag` mode wired to an originality `flag`, *When* the flag is true, *Then* the item is held regardless of confidence.

## 8. Data Model

Reuses `assessment.grading_agent_results` (migration [290](../../../server/migrations/290_grading_agent.sql)) statuses (`suggested`/`applied`/`skipped`/`overridden`). Node `data`:

```jsonc
{
  "mode": "always" | "belowConfidence" | "onFlag",
  "confidenceFloor": 0.7,   // belowConfidence only
  "queue": "default"        // optional named queue / assignee bucket
}
```

Optional: a `held_reason TEXT` and `held_at TIMESTAMPTZ` column on `grading_agent_results` for queue UX (small additive migration) if not already expressible.

## 9. API Surface

- Reuses the existing run/results endpoints from [grading_agent_http.go](../../../server/internal/httpserver/grading_agent_http.go); the results list already supports filtering by status — add a `held`/queue filter.
- Approve/edit reuses `PUT .../submissions/.../grade`; reject transitions the result status via the results endpoint.
- OpenAPI: node `data` schema + held-status filter.

## 10. UI / UX

- **Palette** — "Human Review Gate" in a Control group (slate, with a pause/gate icon).
- **Node body** — Title; `grade`(+optional `comments`/`report`/`flag`) inputs; `grade` output; a small mode badge and live held-count after a run.
- **Inspector** — Mode selector + floor; queue/assignee selector; helper copy on the oversight model.
- **Review queue** — A held-items list in the SpeedGrader grading panel: each row shows the suggestion (score, rubric, comment, confidence, any flag/report) with **Approve**, **Edit & approve**, **Reject** actions; bulk approve for high-confidence batches.
- **States** — Empty queue, loading, approval success, reject confirm.
- **Mobile** — Queue as a full-screen sheet; actions stack.
- **Copy & i18n** — `gradingAgent.canvas.palette.reviewGate`, `gradingAgent.canvas.nodes.reviewGate.*`, `gradingAgent.review.queue.*`.

## 11. AI / ML Considerations

The gate is the canvas lever for the §19.16 confidence/governance model: pair it with a [Conditional Router](node-conditional-router.md) on `confidence` or an [Originality](node-originality-check.md) `flag`. It writes nothing to the model; it constrains how model outputs become grades. Recommended default for auto-grade graphs: gate everything below the eval-tuned confidence floor.

## 12. Integration Points

- **Client** — `types.ts` (`PaletteNodeType` += `'humanReviewGate'`), `node-palette.tsx`, `workflow-nodes.tsx` (`HumanReviewGateNode`), `workflow-node-types.ts`, `validation.ts` (gate `grade` output satisfies terminal reachability), `inspector-panel.tsx`, plus a review-queue component in the grading panel.
- **Server** — `workflow.go` (`NodeTypeHumanReviewGate`, edge typing), `workflow_execute.go` (dry-run pass-through with would-hold log), [grading_agent_consumer.go](../../../server/internal/background/grading_agent_consumer.go) (live runs honor hold → `suggested`, never apply/post), results status transitions, grade-write reuse.
- **Cross-plan** — [Conditional Router](node-conditional-router.md), [Originality Check](node-originality-check.md), [19.16](../auto-grader-agent.md) governance/disclosure, §3.4 moderated grading.

## 13. Dependencies & Sequencing

- **After**: 19.16, 19.17, and [Conditional Router](node-conditional-router.md) (branching) for `belowConfidence`/`onFlag` routing patterns.
- **Before**: strengthens any auto-grade rollout; recommended prerequisite for enabling auto-grade graphs broadly.
- **Shared infra**: existing results table, review surface, grade-write path, background consumer.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Held items pile up and never get reviewed | M | M | Queue badges, digest notifications, bulk approve, age sorting; optional auto-disable auto-grade when backlog grows |
| Instructor sets floor to 0 and rubber-stamps everything | M | H | Default sensible floor; copy on oversight; override-rate monitoring (§19.13) |
| Race: held item also auto-posted by policy | L | H | Hold strictly precedes post; held = never auto-applied/posted (FR-5); idempotency per item |
| Confusion between "held" and "provisional-but-written" | M | M | Distinct statuses + clear queue vs gradebook labeling |

## 15. Rollout Plan

- Behind `grader_agent_enabled`.
- Sequencing: results held-status + consumer hold logic → node/validation → review-queue UI → i18n.
- Phase 1: manual scopes with `belowConfidence`. Phase 2: required gate for auto-grade graphs in pilot tenants.
- Rollback: remove palette item behind flag; in-flight held items remain reviewable.

## 16. Test Plan

- **Unit** — Hold-decision per mode (always/belowConfidence/onFlag); dry-run never persists; status transitions on approve/reject.
- **Integration** — Live run holds sub-floor item (not written), writes confident item; approve writes via grade path with `graded_by_ai`; reject leaves grade unchanged; idempotency.
- **E2E** — Confidence router → gate → Student Grade; run batch; review queue approve/edit/reject; verify gradebook.
- **Security** — Non-grader cannot view/approve queue; cross-course denied.
- **Accessibility** — axe; keyboard approve/edit/reject; held-count announcement.

## 17. Documentation & Training

- Help center: "Reviewing held AI grades."
- Instructor guide: setting confidence floors; the oversight model; bulk approval safely.
- Admin guide: relationship to auto-post policy and audit.
- Compliance note: GDPR Art. 22 oversight evidence.

## 18. Open Questions

1. Add `held_reason`/`held_at` columns, or derive from existing fields? (Plan: small additive migration for queue UX.)
2. Should rejection optionally re-route to a different branch (e.g., reassign to another grader) vs. plain skip? (Defer; v1 = skip with reason.)
3. Per-assignment backlog threshold that auto-disables auto-grade? (Tie to §19.16 open question 3.)

## 19. References

- Server: [grading_agent_http.go](../../../server/internal/httpserver/grading_agent_http.go), [grading_agent_consumer.go](../../../server/internal/background/grading_agent_consumer.go), [workflow_execute.go](../../../server/internal/service/gradingagent/workflow_execute.go), [assignment_submission_grade_http.go](../../../server/internal/httpserver/assignment_submission_grade_http.go), migration [290](../../../server/migrations/290_grading_agent.sql).
- Related: [node catalog](README.md), [19.16 Auto-Grader Agent](../auto-grader-agent.md), [Conditional Router](node-conditional-router.md), [Originality Check](node-originality-check.md), [Flag for Review](node-flag-for-review.md).
