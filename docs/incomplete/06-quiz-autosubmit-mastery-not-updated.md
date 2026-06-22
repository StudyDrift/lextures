# 06 — Quiz auto-submit does not update non-adaptive mastery

- **Category:** Feature not fully implemented (partial port)
- **Severity:** P2
- **Area:** Assessment / time limits & auto-submit (plan 2.7) × learner model (plan 1.1)

## Summary

The background sweep that finalizes timed quiz attempts past their deadline computes and
saves the **score** but does **not** apply the non-adaptive **mastery / learner-state**
update that a normal (student-initiated) submit performs. Auto-submitted attempts therefore
score correctly but do not move the learner model, creating a divergence between attempts
that were submitted manually vs. those swept by the timer.

## Evidence

`server/internal/service/quizautosubmit/sweep.go`:

```go
// SweepExpiredAttempts finalizes timed quiz attempts past deadline ...
// Non-adaptive mastery updates from `learner_state::apply_mastery_from_saved_responses`
// are not yet ported in Go; scores are still finalized so learners receive credit.
func SweepExpiredAttempts(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time, limit int64) (int, error) {
    ...
    // sums response points, finalizes the attempt — but no mastery application
}
```

The corresponding learner-state port boundary is effectively empty
(`server/internal/service/learnerstate/service.go` only exposes a `Health()` heartbeat),
and `quizautosubmit/service.go` is likewise a `Health()`-only stub.

## Impact

- Learners who let a timed quiz expire instead of clicking submit get credit but **no
  mastery/IRT theta update** from those responses — adaptive recommendations, spaced
  repetition scheduling, and mastery reporting under-count their evidence.
- Subtle, data-dependent inconsistency that is hard to notice in QA but compounds over a
  term.

## Suggested fix

- Port `learner_state::apply_mastery_from_saved_responses` and invoke it inside
  `SweepExpiredAttempts` for non-adaptive quizzes (mirroring the manual-submit path), within
  the same transaction that finalizes the score.
- Add a test asserting that an auto-swept attempt and an equivalent manual submit produce
  the same mastery/learner-state deltas.

## Acceptance criteria

- An expired, auto-submitted timed attempt updates the learner's mastery identically to a
  manual submit of the same responses.
