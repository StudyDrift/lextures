package jobqueue

import "time"

// retryDelays is the exponential backoff schedule applied between attempts
// (plan 17.3 FR-4). The Nth failed attempt waits retryDelays[N-1] before the
// row becomes eligible again. Attempts beyond the schedule reuse the final
// delay until max_attempts is reached.
var retryDelays = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	8 * time.Hour,
}

// BackoffDelay returns how long to wait before the given attempt number is
// retried. attempt is the number of attempts already made (1 after the first
// failure). Values <= 0 are treated as 1.
func BackoffDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	idx := attempt - 1
	if idx >= len(retryDelays) {
		idx = len(retryDelays) - 1
	}
	return retryDelays[idx]
}

// NextRetryAt returns the absolute time a job should next become eligible after
// failing for the attempt-th time.
func NextRetryAt(now time.Time, attempt int) time.Time {
	return now.Add(BackoffDelay(attempt))
}
