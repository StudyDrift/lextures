# GA-M3 — Suggest-only batch + bulk review/apply + posting control

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](../../plan/grading-agent/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-M3 |
| **Section** | Grading Agent — Missing Features |
| **Severity** | MAJOR |
| **Markets** | HE / K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | [GA-M1](missing-1-persistent-review-queue.md) |
| **Unblocks** | trust-driven adoption |

## Implementation summary (2026-06-24)

- **Migration** `324_grading_agent_run_mode.sql` — `grading_agent_runs.mode` (`suggest`|`apply`, backfilled to `apply`), platform flag `grader_agent_suggest_mode_enabled`.
- **Consumer** — `finishGradingAgentSuccess` in `grading_agent_apply.go` skips grade writes when `run.mode = suggest`; records held suggestions with `held_at`.
- **Posting** — `gradingAgentCellPosting` respects agent `postPolicy` (`draft`/`auto_post`) on apply and bulk approve; config GET/PUT exposes `postPolicy`.
- **API** — `POST …/runs` accepts `mode`; `POST …/review/bulk` supports `approve`, `approve_all`, `reject` with `resultIds`, `minConfidence`, and per-item overrides.
- **UI** — Run popover mode toggle + posting notes; held queue bulk toolbar (select-all, threshold approve, approve/reject selected, approve all); i18n `gradingAgent.run.mode.*`, `gradingAgent.review.bulk.*`.
- **Flag** — `graderAgentSuggestModeEnabled` (Settings → Global platform). When off, runs default to `apply` (backward compatible).

## 1. Problem Statement

A batch run **writes grades immediately**. In `HandleGradingAgentQueueMessage`, every non-held,
non-flagged item calls `coursegrades.UpsertCellWithFlags(...)` as it is processed. The only way to get
"AI suggests, human approves" is to hand-build a graph with a Human Review Gate on every path. There
is no first-class **suggest-only** batch mode and no **bulk review/apply** of suggestions. New adopters
almost universally want to start in suggest-only mode (AI drafts, instructor spot-checks and approves
in bulk) before trusting auto-apply. Additionally, applied grades are always written with
`posting = "manual"` because the auto-post branch is unreachable (see [GA-B3](../../plan/grading-agent/bug-3-auto-post-dead-code.md)),
so there is no coherent control over whether students see AI grades.

## 2. Goals

- A run mode toggle: **Suggest only** (nothing is written to the gradebook; all items land in the review queue) vs **Auto-apply**.
- Bulk actions over suggestions: approve all, approve above a confidence threshold, approve selected, edit-then-approve, reject selected.
- One clear posting control: keep AI grades as draft (unposted) or post on apply, per agent.

## 3. Non-Goals

- Replacing the Human Review Gate node (it remains for per-path control); this is the simple, whole-run switch.
- Changing the gradebook's own posting/hide mechanics — only choosing which posting state the agent writes.

## 4. Personas & User Stories

- **As a cautious instructor**, I want a suggest-only run so I can review every AI grade before any student sees it.
- **As a TA with 200 submissions**, I want "approve all above 85% confidence" so I only hand-check the uncertain ones.
- **As an instructor**, I want to keep AI grades unposted until I post them, so students never see an unreviewed grade.

## 5. Functional Requirements

- **FR-1.** The run request MUST accept a `mode` of `suggest` or `apply` (default `suggest` for a newly accepted agent).
- **FR-2.** In `suggest` mode the consumer MUST record results as `suggested` (held) and MUST NOT call `UpsertCellWithFlags`.
- **FR-3.** The review queue ([GA-M1](missing-1-persistent-review-queue.md)) MUST support bulk **approve all**, **approve ≥ confidence X**, **approve selected**, **reject selected**, and **edit + approve** for a single item.
- **FR-4.** Approving a suggestion MUST write the grade and mark the result `applied` (or `overridden` if edited).
- **FR-5.** The agent config MUST carry a posting choice (`draft` vs `auto_post`) that is **persisted and read** at write time — fixing the dead `post_policy` path ([GA-B3](../../plan/grading-agent/bug-3-auto-post-dead-code.md)).
- **FR-6.** Bulk apply MUST be transactional per batch chunk and idempotent (re-approving an applied item is a no-op).

## 6. Non-Functional Requirements

- **Performance** — bulk approve of 500 items completes < 10 s; chunked writes.
- **Security** — same RBAC as grading; posting choice respects course posting policy.
- **Privacy & Compliance** — suggest-only guarantees no student visibility until approval (FERPA-friendly default).
- **Accessibility** — bulk controls keyboard-operable; selection has clear focus + `aria-selected`.
- **Reliability** — partial bulk failures report per-item outcomes; no half-written state.
- **Observability** — metrics: suggest vs apply runs, bulk-approve sizes, edit rate.
- **Internationalization** — `gradingAgent.run.mode.*`, `gradingAgent.review.bulk.*`.
- **Backward compatibility** — existing runs default to today's behavior under a flag until GA.

## 7. Acceptance Criteria

- **AC-1.** *Given* a suggest-only run, *when* it completes, *then* no gradebook cells changed and all items are in the review queue.
- **AC-2.** *Given* 50 suggestions, *when* I "approve all ≥ 80%", *then* only those write grades and the rest remain.
- **AC-3.** *Given* a suggestion, *when* I edit the score and approve, *then* the result is `overridden` with my values.
- **AC-4.** *Given* an agent set to `draft`, *when* grades are applied, *then* they are written unposted.
- **AC-5.** *Given* an agent set to `auto_post` and an assignment that posts automatically, *when* grades are applied, *then* they are posted.

## 8. Data Model

- `grading_agent_runs`: add `mode TEXT NOT NULL DEFAULT 'suggest'` (`suggest`|`apply`).
- `grading_agent_configs`: make `post_policy` writable (already exists; `'unposted'` default → allow `'auto_post'`).
- Migration: `server/migrations/324_grading_agent_run_mode.sql`.
- Backfill: existing runs are historical; default `apply` for backfilled rows to preserve meaning, `suggest` for new.

## 9. API Surface

- `POST …/grader-agent/runs` body gains `mode`.
- `PUT …/grader-agent` config body gains `postPolicy` (`draft`|`auto_post`); persisted by `UpsertConfig` (extend `UpsertConfigInput`).
- New bulk endpoint `POST …/grader-agent/review/bulk` `{ action, resultIds?|filter }` → per-item outcomes. (Or extend PATCH with batch semantics.)

## 10. UI / UX

- Run popover (`run-agent-popover.tsx`) gains a **Suggest only / Auto-apply** segmented control and a posting note.
- Review queue gains a bulk toolbar: select-all, "approve ≥ [confidence]", approve/reject selected.
- Empty/loading/error states; confirmation on "approve all".
- Copy/i18n under `gradingAgent.run.mode.*` and `gradingAgent.review.bulk.*`.

## 11. AI / ML Considerations

- None new; suggest-only still runs the model once per submission. Confidence threshold for bulk approve reuses model confidence already returned.

## 12. Integration Points

- `server/internal/httpserver/grading_agent_queue.go` (mode-aware write), `grading_agent_http.go` (run body, config body, bulk endpoint).
- `server/internal/repos/gradingagent/repo.go` (`UpsertConfigInput.PostPolicy`, run `mode`).
- `clients/web/src/components/annotation/grader-agent/{run-agent-popover,review-queue-panel,held-review-queue-panel}.tsx`, `use-grader-agent-workflow.ts`.

## 13. Dependencies & Sequencing

- Requires [GA-M1](missing-1-persistent-review-queue.md) (the durable queue) to host suggestions and bulk actions.
- Pairs with [GA-B3](../../plan/grading-agent/bug-3-auto-post-dead-code.md) (posting) and [GA-M4](../../plan/grading-agent/missing-4-confidence-auto-hold-threshold.md) (confidence).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Suggest-only confuses users expecting grades written | M | M | Clear mode copy + post-run summary "0 written, N to review" |
| Bulk approve writes wrong grades at scale | L | H | Dry-run-style confirmation count + per-item outcome report + idempotency |
| `mode` backfill changes historical meaning | L | M | Backfill historical runs to `apply` |

## 15. Rollout Plan

- Flag: `graderAgentSuggestModeEnabled`.
- Sequence: migration → consumer mode branch + posting write → run/config API → review bulk UI → flip flag (default suggest for new agents).
- Pilot: a course onboarding the agent for the first time.
- Rollback: flag off → revert to immediate apply.

## 16. Test Plan

- **Unit** — consumer write/no-write by mode; posting decision matrix; bulk filter selection.
- **Integration** — suggest run writes nothing; approve writes grade; auto_post posts.
- **E2E** — suggest run → bulk approve ≥ threshold → gradebook reflects only approved.
- **Security** — bulk action RBAC; idempotent re-approve.

## 17. Documentation & Training

- Help-center: "Suggest-only vs auto-apply, and approving AI grades in bulk."
- Instructor onboarding checklist recommends suggest-only first.

## 18. Open Questions

1. Should the default for a brand-new accepted agent be `suggest` (recommended) and switch to `apply` only after the instructor opts in?
2. Should "approve ≥ confidence" use model confidence or also factor originality/flag signals?
3. Do we expose posting choice at run time as well as config time?

## 19. References

- `server/internal/httpserver/grading_agent_queue.go` (`UpsertCellWithFlags`, `posting` derivation).
- `server/internal/repos/gradingagent/repo.go` (`UpsertConfig`, `CreateRun`).
- `clients/web/src/components/annotation/grader-agent/run-agent-popover.tsx`, `held-review-queue-panel.tsx`.
- Related: [GA-M1](missing-1-persistent-review-queue.md), [GA-M4](../../plan/grading-agent/missing-4-confidence-auto-hold-threshold.md), [GA-B3](../../plan/grading-agent/bug-3-auto-post-dead-code.md).