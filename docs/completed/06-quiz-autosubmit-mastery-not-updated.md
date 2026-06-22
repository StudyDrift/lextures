# 06 — Quiz auto-submit does not update non-adaptive mastery

- **Category:** Feature not fully implemented (partial port)
- **Severity:** P2
- **Area:** Assessment / time limits & auto-submit (plan 2.7) × learner model (plan 1.1)
- **Status:** Fixed (2026-06-22)

## Summary

The background sweep that finalizes timed quiz attempts past their deadline computes and
saves the **score** but did **not** apply the non-adaptive **mastery / learner-state**
update that a normal (student-initiated) submit performs. Auto-submitted attempts therefore
scored correctly but did not move the learner model, creating a divergence between attempts
that were submitted manually vs. those swept by the timer.

## Fix

- Ported `learner_state::apply_mastery_from_saved_responses` to Go in
  `server/internal/service/learnerstate/mastery.go`, with persistence in
  `server/internal/repos/learnermodel/mastery_update.go`.
- `SweepExpiredAttempts` (`server/internal/service/quizautosubmit/sweep.go`) now resolves
  delivery questions, loads saved responses, and applies mastery inside the same transaction
  that finalizes the score (for non-adaptive quizzes when `adaptiveLearnerModelEnabled` is on).
- Added unit tests for EMA/decay helpers and a Postgres integration test asserting auto-sweep
  and manual mastery application produce identical `learner_concept_states` deltas
  (`sweep_db_test.go`).

## Acceptance criteria

- An expired, auto-submitted timed attempt updates the learner's mastery identically to a
  manual submit of the same responses. **Met** — verified by `TestSweepExpiredAttempts_MasteryMatchesManualSubmit_Pg`.