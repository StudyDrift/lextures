# GA-M1 — Persistent, actionable review queue & run history

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](../../plan/grading-agent/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-M1 |
| **Section** | Grading Agent — Missing Features |
| **Severity** | BLOCKER |
| **Markets** | HE / K12 / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | — |
| **Unblocks** | [GA-M3](../../plan/grading-agent/missing-3-suggest-only-batch-and-bulk-review.md), [GA-M6](../../plan/grading-agent/missing-6-cancel-running-batch.md), [GA-M7](../../plan/grading-agent/missing-7-cost-estimate-and-budget.md) |

---

## 1. Problem Statement

Held items (Human Review Gate) and flagged items (Flag for Review) only exist in the UI while the
grading-agent modal is open and a `runId` is held in React state (`use-grader-agent-workflow.ts`).
The moment the instructor closes the modal, the in-memory `runId` is lost and there is **no endpoint
to list an assignment's runs or re-open the latest run** — only `GET …/runs/{run_id}` by id. The
flagged queue (`review-queue-panel.tsx`) is also **read-only**: it lists a reason and priority with
no approve/grade/dismiss action. The result is that a TA who batch-grades 120 essays, sees "18 held
for review," and closes the tab cannot get back to those 18 items. This makes the human-in-the-loop
story — the main reason an instructor trusts the agent — effectively unusable across sessions.

## 2. Goals

- A durable, re-openable review queue scoped to an assignment (and to the course) that survives modal close.
- Make **flagged** items actionable (open submission, grade, dismiss/resolve, re-run) — parity with the held queue.
- Run history per agent: who ran it, when, scope, model, counts, cost, status.
- A single "needs my attention" count surfaced where graders already work (assignment page, gradebook, course agents list).

## 3. Non-Goals

- Cross-tenant or admin-wide review dashboards (course scope is enough for v1).
- Changing how held/flagged decisions are *made* (that is the gate/flag node config; see [GA-M4](../../plan/grading-agent/missing-4-confidence-auto-hold-threshold.md)).
- A new notification channel beyond the existing inbox.

## 4. Personas & User Stories

- **As an instructor**, I want to reopen the list of held/flagged submissions after closing the grader, so that review is not a single-session task.
- **As a TA**, I want one place that shows "N submissions awaiting your review" across runs, so that I can clear the queue between classes.
- **As an instructor**, I want to act on a flagged item (open it, grade it, or dismiss the flag), so that flagging is not a dead end.
- **As an instructor**, I want a run history with model + counts + cost, so that I can audit what the agent did.
- **As an admin**, I want runs and review actions retained for FERPA audit, so that AI grading is accountable.

## 5. Functional Requirements

- **FR-1.** The system MUST expose `GET …/assignments/{item_id}/grader-agent/runs` returning runs ordered by `created_at desc` with status, scope, counts, initiated_by, model, and cost.
- **FR-2.** The system MUST expose a review queue read model: held (`status = suggested` with `held_at`) and flagged (`status = flagged`) results for an assignment across all non-dry-run runs, with submission/student labels.
- **FR-3.** The UI MUST render the review queue from that read model independent of any live `runId`, and on modal open MUST hydrate it for the current assignment.
- **FR-4.** Flagged items MUST support actions: **open submission**, **grade now** (write a grade + mark `applied`/`overridden`), and **dismiss** (mark `skipped` with a reason). These reuse the held-queue handlers in `held-review-queue-panel.tsx`.
- **FR-5.** The review queue count MUST be surfaced on the assignment page action menu and on the course "Grading agents" list row.
- **FR-6.** A held/flagged item MUST be **deduplicated by submission** in the read model (latest non-terminal result per submission wins) so re-runs do not stack duplicates.
- **FR-7.** Review actions MUST be permission-gated identically to existing grading (`course:{code}:item:create`).

## 6. Non-Functional Requirements

- **Performance** — review-queue read model p95 < 300 ms for an assignment with 1k submissions; paginate at 100.
- **Security** — same RBAC as `requireGraderAgentAccess`; no cross-course leakage of labels.
- **Privacy & Compliance** — respect blind/anonymous grading: labels MUST use `blindLabel` when the assignment is anonymized; retain run/result rows per the existing audit retention policy (FERPA).
- **Accessibility** — queue is a list with actionable buttons; `aria-live` count; full keyboard operation; focus returns to the triggering row after an action.
- **Scalability** — indexed reads (see §8); no N+1 per item for labels.
- **Reliability** — actions are idempotent (grading an already-graded item is a safe upsert).
- **Observability** — emit metrics for queue size, time-to-review, and action outcomes.
- **Internationalization** — all new strings under `gradingAgent.review.*`.
- **Backward compatibility** — additive; existing per-run polling continues to work.

## 7. Acceptance Criteria

- **AC-1.** *Given* a completed batch run with held items, *when* I close and reopen the grader for that assignment, *then* the held items still appear and are actionable.
- **AC-2.** *Given* a flagged item, *when* I click "Grade now" and submit a score, *then* the result becomes `applied`/`overridden` and the grade is written to the gradebook.
- **AC-3.** *Given* a flagged item, *when* I click "Dismiss" with a reason, *then* it becomes `skipped` and leaves the queue.
- **AC-4.** *Given* two runs that both touched submission X, *when* I open the queue, *then* X appears at most once.
- **AC-5.** *Given* an anonymized assignment, *when* the queue renders, *then* no student name is shown.
- **AC-6.** *Given* an assignment with 3 items needing review, *when* I view the assignment page, *then* a "3 to review" affordance is shown.

## 8. Data Model

- No new tables required; `grading_agent_runs` and `grading_agent_results` already store everything.
- Add indexes:
  - `CREATE INDEX ON assessment.grading_agent_runs (config_id, created_at DESC);`
  - `CREATE INDEX ON assessment.grading_agent_results (config_id, status) WHERE is_dry_run = false;`
- Optional column on results: `resolved_at TIMESTAMPTZ`, `resolved_by UUID` to record who cleared a held/flagged item (else infer from status transition).
- Migration: `server/migrations/322_grading_agent_review_indexes.sql`.
- Backfill: none (indexes only); resolved_* nullable.

## 9. API Surface

- `GET …/assignments/{item_id}/grader-agent/runs` → `{ runs: [{ id, scope, status, totalCount, completedCount, failedCount, model, costUsd, initiatedBy, createdAt, finishedAt }] }`.
- `GET …/assignments/{item_id}/grader-agent/review-queue` → `{ held: [...], flagged: [...] }` (dedup per submission, includes labels).
- Reuse existing `PATCH …/results/{result_id}` for dismiss/apply; extend `handlePatchGraderAgentResult` to accept `flagged → applied/overridden/skipped` transitions (today only suggested→applied/overridden/skipped is the implicit expectation).
- Rate-limit identical to other grading endpoints; document in OpenAPI.

## 10. UI / UX

- New `ReviewInbox` surface reachable from (a) the grader modal (hydrated independent of live run) and (b) the course "Grading agents" section as a per-agent "Review (N)" link.
- Upgrade `review-queue-panel.tsx` (flagged) to the same action set as `held-review-queue-panel.tsx`.
- States: empty ("Nothing to review"), loading skeleton, error, and per-item busy.
- Mobile: single-column cards; actions wrap.
- Accessibility: `aria-live="polite"` on counts; focus management after each action.
- Copy/i18n: `gradingAgent.review.*`.

## 11. AI / ML Considerations

- None new. Re-grading from the queue may re-invoke the agent (see [GA-M3](../../plan/grading-agent/missing-3-suggest-only-batch-and-bulk-review.md)); cost is attributed to the re-run.

## 12. Integration Points

- `server/internal/httpserver/grading_agent_http.go` (new handlers), `courses_routes.go` (routes).
- `server/internal/repos/gradingagent/repo.go` (`ListRunsByConfig`, `ListReviewQueueByConfig`).
- `clients/web/src/components/annotation/grader-agent/{review-queue-panel,held-review-queue-panel,use-grader-agent-workflow}.tsx/ts`.
- `clients/web/src/pages/lms/course-grading-agents-section.tsx`, assignment page action menu.

## 13. Dependencies & Sequencing

- Must ship before [GA-M6](../../plan/grading-agent/missing-6-cancel-running-batch.md) and [GA-M7](../../plan/grading-agent/missing-7-cost-estimate-and-budget.md) so cancel/cost have a run-history home.
- Shared infra: none beyond Postgres.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Dedup logic hides a genuinely re-run item | M | M | Dedup by (submission, latest createdAt); show run timestamp on each card |
| Label lookups N+1 | M | M | Batch-resolve labels in one query keyed by submission ids |
| Blind-grading identity leak | L | H | Centralize label resolution; cover with a test on anonymized assignments |

## 15. Rollout Plan

- Flag: `graderAgentReviewInbox` (default off → dogfood → on).
- Sequence: migration (indexes) → repo reads → API → UI → flip flag.
- Pilot: a TA-heavy course; GA when queue actions and counts verified.
- Rollback: hide UI via flag; endpoints are read-only/idempotent.

## 16. Test Plan

- **Unit** — dedup + label resolution; PATCH transition matrix (flagged→applied/overridden/skipped).
- **Integration** — list runs / review queue across multiple runs; RBAC denial.
- **E2E** — batch run → close modal → reopen → act on held + flagged items → gradebook reflects change.
- **Security** — cross-course access denied; anonymized labels.
- **Accessibility** — axe + keyboard-only walkthrough.
- **Performance** — 1k-result assignment read p95.

## 17. Documentation & Training

- Help-center: "Reviewing AI-suggested and flagged grades."
- Instructor doc: where the review inbox lives and how counts work.
- API reference + runbook for the new endpoints.

## 18. Open Questions

1. Should the inbox aggregate across *all* assignments in a course (a true grader inbox) in v1, or per-assignment only?
2. Do we keep dry-run results out of the queue entirely (current assumption) or show the latest dry run as a preview?
3. Should "dismiss" on a flagged item leave the submission ungraded or require a grade?

## 19. References

- `clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts` (`runId`, `processRunStatus`, `refreshRunResults`).
- `clients/web/src/components/annotation/grader-agent/{review-queue-panel,held-review-queue-panel}.tsx`.
- `server/internal/httpserver/grading_agent_http.go` (`handleGetGraderAgentRun`, `handlePatchGraderAgentResult`).
- `server/internal/repos/gradingagent/repo.go`.
- Related: [GA-M3](../../plan/grading-agent/missing-3-suggest-only-batch-and-bulk-review.md), [GA-M4](../../plan/grading-agent/missing-4-confidence-auto-hold-threshold.md).