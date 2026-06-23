# 19.17.9 — Flag for Review Node (Alternate Review-Queue Sink)

> Implementation plan. Source: [docs/MISSING_FEATURES.md](../../MISSING_FEATURES.md) §19. Extends the grading-agent canvas ([19.17](../auto-grader-agent.md)). See the [node catalog](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | 19.17.9 |
| **Section** | AI-Specific Capabilities → Grading Agent Canvas |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | AI / Grading team |
| **Depends on** | 19.16, 19.17, [19.17.5 Conditional Router](node-conditional-router.md) |
| **Unblocks** | Triage-instead-of-grade workflows, integrity holds |
| **Consumes shared change** | **Branching / optional terminals** (owned by [Conditional Router](node-conditional-router.md)) |

---

## 1. Problem Statement

Today the canvas has exactly one terminal: **Student Grade**, which writes a (provisional) grade. But the right outcome for some submissions is *not a grade at all* — it is "send this to a human to look at." A blank or off-topic submission, a suspected integrity case, or an edge case the agent shouldn't grade should be **triaged**, not scored. With only one sink, instructors are forced to fabricate a grade or drop the branch (which the new branching rules reject). This node is a second **output/sink** kind: it routes an item — with a reason — into a review/triage queue instead of writing a grade, satisfying the "every path reaches a terminal" rule on branches that intentionally don't grade.

## 2. Goals

- Provide an **output-category** node that terminates a branch by enqueuing the item for human triage rather than writing a grade.
- Capture a `reason` (from upstream `report`/`comments`/`flag` or a template) and a priority/queue.
- Satisfy branch-terminal reachability (a routed-away "don't grade" path is valid because it ends in this sink).
- Coexist with the single Student Grade node — a graph may have one Student Grade plus one or more Flag-for-Review sinks.

## 3. Non-Goals

- Writing or modifying a grade (that is the Student Grade sink; pair with the [Human Review Gate](node-human-review-gate.md) if you want "hold a *grade*").
- A general ticketing system — it enqueues into the grading-agent review surface.
- Notifications/escalation policy beyond a queue + priority (future enhancement).

## 4. Personas & User Stories

- **As an instructor**, I want off-topic/blank submissions sent to a "needs human" list instead of getting a hallucinated grade.
- **As an integrity officer**, I want high-similarity submissions diverted to an integrity triage queue with the originality report attached.
- **As a TA**, I want a single place to see everything the agent declined to grade, with the reason.
- **As a student**, I want to know my submission is awaiting human review rather than silently ungraded.

## 5. Functional Requirements

- **FR-1.** The palette MUST offer a **Flag for Review** node under a new Output group (alongside the conceptual Student Grade sink).
- **FR-2.** The node MUST accept `reason`/`comments`/`report` (text), `flag` (boolean), and optionally `grade` (carried context, *not* written) inputs, and MUST have **no** outputs (it is terminal).
- **FR-3.** The inspector MUST let the instructor choose a queue/assignee and priority, and define a reason template that can interpolate `$Node.Property` values (e.g., similarity score).
- **FR-4.** During a **dry run**, the node MUST log "would flag for review: <reason>" and persist nothing.
- **FR-5.** During **live/batch/auto** runs, reaching this sink MUST create a review-queue entry (a `grading_agent_results` row marked `flagged`) with the reason/priority and MUST NOT write a grade for that submission.
- **FR-6.** A graph MUST be valid when some branches reach Student Grade and others reach Flag for Review; reachability requires **every executable path** to reach *some* terminal (grade or flag).
- **FR-7.** A graph MUST remain limited to exactly **one** Student Grade node, but MAY contain multiple Flag-for-Review nodes.
- **FR-8.** Flagged items MUST appear in the same review surface as held items from the [Human Review Gate](node-human-review-gate.md), distinguishable by origin.

## 6. Non-Functional Requirements

- **Performance** — Negligible; a single insert per flagged item.
- **Security** — Only course graders see/triage flagged items; standard course authz.
- **Privacy & Compliance** — Flagged items are FERPA records; reasons may reference integrity signals — same disclosure/appeal posture as 19.16; no grade is written, so no automated-decision *grade* is made on these.
- **Accessibility** — Queue/priority controls and reason editor keyboard-navigable; flagged count announced.
- **Scalability** — Indexed by config/assignment/status like other results.
- **Reliability** — Idempotent per `(run_id, submission_id)`; never both flags and grades the same item on one path.
- **Observability** — `grader_agent_flagged_total{queue,priority}`; reason captured.
- **Maintainability** — Reuses the results table + review surface from the gate node.
- **Internationalization** — Reason template + queue labels localized.
- **Backward compatibility** — Additive; introduces the Output palette group but doesn't change the existing Student Grade node.

## 7. Acceptance Criteria

- **AC-1.** *Given* a router `isEmpty == true` whose `then` reaches a Flag-for-Review sink and `else` reaches Student Grade, *When* validated, *Then* the graph is valid (both paths reach a terminal).
- **AC-2.** *Given* a blank submission on that graph, *When* a live run executes, *Then* a `flagged` review entry is created with the reason and **no grade is written**.
- **AC-3.** *Given* an originality `report` wired as the reason with a similarity template, *When* an item is flagged, *Then* the queue entry shows the interpolated similarity in the reason.
- **AC-4.** *Given* a dry run reaching the sink, *When* it completes, *Then* nothing is persisted and the trace logs the would-flag reason.
- **AC-5.** *Given* a graph with two Student Grade nodes, *When* validated, *Then* it is rejected (still exactly one grade sink); two Flag-for-Review nodes is allowed.
- **AC-6.** *Given* a flagged item, *When* a grader opens the review surface, *Then* it appears alongside gate-held items, labelled as "flagged (not graded)".

## 8. Data Model

Reuses `assessment.grading_agent_results` (migration [290](../../../server/migrations/290_grading_agent.sql)). Adds a `flagged` value to `assessment.grading_agent_item_status` (small additive migration) and an optional `flag_reason TEXT` / `flag_priority TEXT`. Node `data`:

```jsonc
{
  "queue": "integrity" | "default" | "...",
  "priority": "low" | "normal" | "high",
  "reasonTemplate": "Similarity $Originality.Score — needs human review"
}
```

No backfill.

## 9. API Surface

- Reuses run/results endpoints from [grading_agent_http.go](../../../server/internal/httpserver/grading_agent_http.go); the results list filter extends to include `flagged`.
- Triage actions (resolve / send to grade / dismiss) transition the result status; resolving "to grade" can hand off to the standard grade-write path.
- OpenAPI: node `data` schema + `flagged` status.

## 10. UI / UX

- **Palette** — New **Output** group containing "Flag for Review" (and documenting the fixed Student Grade sink). Emerald/rose accent.
- **Node body** — Title; `reason`/`comments`/`report`/`flag`/(optional `grade`) input slots; **no** output slot; queue + priority badges.
- **Inspector** — Queue/assignee selector, priority, reason-template editor with `$Node.Property` autocomplete (reuse [`WorkflowPromptEditor`](../../../clients/web/src/components/annotation/grader-agent/workflow-prompt-editor.tsx)).
- **Review surface** — Flagged items listed with held items (from the gate), tagged "flagged — not graded," each with reason, priority, and triage actions.
- **States** — Empty group hint, validation error if a path reaches no terminal, dry-run would-flag preview.
- **Mobile** — Sheet; inputs stack.
- **Copy & i18n** — `gradingAgent.canvas.palette.groupOutput`, `gradingAgent.canvas.palette.flagForReview`, `gradingAgent.canvas.nodes.flagForReview.*`, `gradingAgent.review.flagged.*`.

## 11. AI / ML Considerations

No model call. It is the "escape hatch" that keeps the agent from grading what it shouldn't — pairs naturally with the [Conditional Router](node-conditional-router.md) (route degenerate/suspicious inputs here) and [Originality Check](node-originality-check.md) (integrity reason). Reduces hallucinated grades on out-of-distribution submissions, improving trust metrics.

## 12. Integration Points

- **Client** — `types.ts` (`GraderNodeType` += `'flagForReview'`; this is the first palette **output** node, so add a `groupOutput`; `HANDLE_REASON` reuse of text), `node-palette.tsx` (new group), `workflow-nodes.tsx` (`FlagForReviewNode`, terminal/no source handle), `workflow-node-types.ts`, `validation.ts` (multi-terminal reachability; exactly one Student Grade; ≥1 terminal per path), `inspector-panel.tsx`.
- **Server** — `workflow.go` (`NodeTypeFlagForReview`, terminal edge typing, validation: one output/grade node + N flag sinks, per-path terminal reachability), `workflow_execute.go` (sink case creates the flagged entry; dry-run logs only), `grading_agent_item_status` enum + result write, [grading_agent_consumer.go](../../../server/internal/background/grading_agent_consumer.go).
- **Cross-plan** — [Conditional Router](node-conditional-router.md) (branching/terminals), [Human Review Gate](node-human-review-gate.md) (shared review surface), [Originality Check](node-originality-check.md).

## 13. Dependencies & Sequencing

- **After**: 19.16, 19.17, **and** [Conditional Router](node-conditional-router.md) (multi-terminal/branch reachability must exist first).
- **Before**: nothing hard; completes the triage story with the [Human Review Gate](node-human-review-gate.md).
- **Shared infra**: results table + review surface.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Flagged items become a dumping ground that's never triaged | M | M | Shared queue with badges/age sorting; reuse gate-queue notifications |
| Author builds a graph where everything flags and nothing grades | L | M | Allowed but warned; dry-run/preview shows expected terminal distribution |
| Confusion between "flagged" and "held" | M | L | Distinct status + clear "not graded" labeling vs gate's "suggested" |
| Multi-terminal validation regresses the single-grade invariant | M | M | Exactly one Student Grade enforced; flag sinks are separate; covered by validation tests |

## 15. Rollout Plan

- Behind `grader_agent_enabled`; ship after the Conditional Router.
- Sequencing: status enum + result write → multi-terminal validation → node + output palette group → review-surface integration → i18n.
- Dogfood: blank/off-topic triage and integrity diversion.
- Rollback: remove palette item behind flag; existing flagged rows remain triageable.

## 16. Test Plan

- **Unit** — Multi-terminal reachability (valid mixed-terminal graphs; invalid dead-ends; one-grade-node enforcement); reason-template interpolation; dry-run persists nothing.
- **Integration** — Live run flags a blank submission (no grade written); reason shows interpolated signal; triage "send to grade" hands to grade path.
- **E2E** — Router → Flag-for-Review (then) / Student Grade (else); run blank + normal; verify queue entry vs gradebook grade.
- **Security** — Non-grader cannot view/triage; cross-course denied.
- **Accessibility** — axe; keyboard triage; flagged-count announcement.

## 17. Documentation & Training

- Help center: "Sending submissions to human review instead of grading."
- Instructor guide: triage vs hold (Flag-for-Review vs Human Review Gate); integrity diversion recipe.
- API reference: node `data` schema + `flagged` status + results filter.

## 18. Open Questions

1. Should resolving a flagged item be able to *re-enter* the workflow (re-grade) or only hand to manual grading? (Plan: manual grade or dismiss in v1; re-run is a fast-follow.)
2. Per-queue assignee/notification routing now or later? (Later; v1 = named queue + priority.)
3. Should a flagged item also surface to the student as "awaiting review"? (Tie to 19.16 student disclosure; leaning yes.)

## 19. References

- Server: [workflow.go](../../../server/internal/service/gradingagent/workflow.go), [workflow_execute.go](../../../server/internal/service/gradingagent/workflow_execute.go), [grading_agent_http.go](../../../server/internal/httpserver/grading_agent_http.go), [grading_agent_consumer.go](../../../server/internal/background/grading_agent_consumer.go), migration [290](../../../server/migrations/290_grading_agent.sql).
- Client: [types.ts](../../../clients/web/src/components/annotation/grader-agent/types.ts), [validation.ts](../../../clients/web/src/components/annotation/grader-agent/validation.ts), [node-palette.tsx](../../../clients/web/src/components/annotation/grader-agent/node-palette.tsx), [workflow-prompt-editor.tsx](../../../clients/web/src/components/annotation/grader-agent/workflow-prompt-editor.tsx).
- Related: [node catalog](README.md), [Conditional Router](node-conditional-router.md), [Human Review Gate](node-human-review-gate.md), [Originality Check](node-originality-check.md), [19.16 Auto-Grader Agent](../auto-grader-agent.md).
