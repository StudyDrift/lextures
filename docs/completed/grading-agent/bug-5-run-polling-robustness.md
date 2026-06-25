# GA-B5 — Run-status polling robustness

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-B5 |
| **Section** | Grading Agent — Bugs |
| **Severity** | MINOR |
| **Bug size** | Small |
| **Markets** | HE / K12 / SL |
| **Status (today)** | BUG |
| **Estimated effort** | XS (≤1d) |
| **Owner (proposed)** | Web / Grading squad |
| **Depends on** | — |
| **Unblocks** | — |

## 1. Problem Statement

The batch-run poller in `use-grader-agent-workflow.ts` has two minor robustness issues:

1. **Late interval clear.** `poll()` is invoked immediately (`void poll()`) **before** `timer` is
   assigned, and the "finished" branch only clears the interval when `timer !== undefined`. If a run is
   already `done` on the very first poll, that first invocation cannot clear the (not-yet-created)
   interval, so the interval is still created and fires at least one extra time before the next branch
   clears it. Harmless but sloppy, and it makes the "done" path do one redundant network round trip.
2. **Repeated `onApplied`.** `processRunStatus` calls `onApplied?.()` on every terminal poll, and
   `syncAppliedResultToCanvas` is invoked for every `applied` result on every poll (guarded by a ref
   set, so mostly idempotent). `onApplied` (which typically refreshes the gradebook) can fire more than
   necessary, causing extra refetches.

Neither corrupts data, but both add avoidable network chatter and make the polling logic harder to reason about — and they sit right next to the more serious lifecycle gaps in [GA-B1](bug-1-queue-overflow-and-stuck-runs.md).

## 2. Goals

- Stop polling promptly and exactly once when a run is terminal (including terminal-on-first-poll).
- Fire `onApplied` once on terminal, not repeatedly.
- Make the poller resilient to terminal `failed`/`cancelled` states (ties to [GA-B1](bug-1-queue-overflow-and-stuck-runs.md), [GA-M6](missing-6-cancel-running-batch.md)).

## 3. Non-Goals

- Switching from polling to WebSocket/SSE for batch progress (possible future, not here).
- Changing the 1.5 s cadence.

## 4. Personas & User Stories

- **As an instructor**, I want progress to settle immediately when a run finishes, so that the UI does not flicker or refetch needlessly.
- **As a web engineer**, I want clear poll lifecycle, so that terminal states are handled in one place.

## 5. Functional Requirements

- **FR-1.** When a run is terminal on the first poll, the poller MUST NOT start (or MUST immediately stop) the interval — no extra request.
- **FR-2.** The interval MUST be cleared exactly once on terminal, regardless of which poll observed it.
- **FR-3.** `onApplied` MUST fire once per run completion, not on every terminal poll.
- **FR-4.** The poller MUST treat `failed` and `cancelled` as terminal and stop (forward-compatible with [GA-B1](bug-1-queue-overflow-and-stuck-runs.md)/[GA-M6](missing-6-cancel-running-batch.md)).
- **FR-5.** Use a ref/flag to guard against double-finalization across overlapping polls.

## 6. Non-Functional Requirements

- **Performance** — fewer redundant requests and gradebook refetches.
- **Reliability** — deterministic single finalization.
- **Accessibility** — `statusMessage`/`aria-live` settles to one final value.
- **Backward compatibility** — internal behavior only.

## 7. Acceptance Criteria

- **AC-1.** *Given* a run that is `done` on first poll, *when* polling starts, *then* at most one status request is made and no interval lingers.
- **AC-2.** *Given* a run that completes after several polls, *when* it finishes, *then* `onApplied` fires once and polling stops.
- **AC-3.** *Given* a `failed`/`cancelled` run, *when* observed, *then* polling stops and the terminal state is shown.

## 8. Data Model

- None.

## 9. API Surface

- None.

## 10. UI / UX

- No visible change beyond fewer flickers/refetches; terminal status settles cleanly.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- `clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts` (the poll `useEffect`, `processRunStatus`, `finished` handling, `onApplied`).

## 13. Dependencies & Sequencing

- Small; best landed with [GA-B1](bug-1-queue-overflow-and-stuck-runs.md) so terminal `failed`/`cancelled` are handled together.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Over-eager finalization stops a still-running poll | L | M | Finalize only on explicit terminal statuses; guard with a `finishedRef` |

## 15. Rollout Plan

- Single PR; no flag. Covered by hook/component tests.
- Rollback: revert PR.

## 16. Test Plan

- **Unit/Component** — terminal-on-first-poll stops cleanly; `onApplied` fires once; failed/cancelled stop polling.
- **Manual** — observe network panel: no polling after terminal.

## 17. Documentation & Training

- None beyond PR notes.

## 18. Open Questions

1. Move batch progress to the existing WS channel instead of polling (larger change, separate plan)?

## 19. References

- `clients/web/src/components/annotation/grader-agent/use-grader-agent-workflow.ts` (poll `useEffect` ~L379–407; `processRunStatus` ~L327–377).
- Related: [GA-B1](bug-1-queue-overflow-and-stuck-runs.md), [GA-M6](missing-6-cancel-running-batch.md).
