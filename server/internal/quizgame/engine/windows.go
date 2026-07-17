package engine

import (
	"errors"
	"time"
)

var (
	ErrNotYetOpen   = errors.New("quizgame: assignment not yet open")
	ErrClosed       = errors.New("quizgame: assignment closed")
	ErrOutOfAttempts = errors.New("quizgame: out of attempts")
)

// AssignmentWindow is the base open/due/close triple (IQ.6 FR-6/FR-8).
type AssignmentWindow struct {
	OpensAt  *time.Time
	DueAt    *time.Time
	ClosesAt *time.Time
}

// EffectiveWindow applies a time multiplier to due/close (accommodations).
// opens_at is unchanged; due/close are extended by (multiplier-1)*remaining duration
// from opens (or now if opens is nil) — matching common LMS "extra time" semantics:
// due and close deadlines are pushed later by factoring the window length.
func EffectiveWindow(base AssignmentWindow, timeMultiplier float64, now time.Time) AssignmentWindow {
	if timeMultiplier <= 0 {
		timeMultiplier = 1
	}
	out := base
	if timeMultiplier == 1 {
		return out
	}
	anchor := now
	if base.OpensAt != nil {
		anchor = *base.OpensAt
	}
	if base.DueAt != nil {
		d := extendFromAnchor(anchor, *base.DueAt, timeMultiplier)
		out.DueAt = &d
	}
	if base.ClosesAt != nil {
		c := extendFromAnchor(anchor, *base.ClosesAt, timeMultiplier)
		out.ClosesAt = &c
	}
	return out
}

func extendFromAnchor(anchor, deadline time.Time, mult float64) time.Time {
	if !deadline.After(anchor) {
		return deadline
	}
	dur := deadline.Sub(anchor)
	return anchor.Add(time.Duration(float64(dur) * mult))
}

// CheckPlayWindow enforces opens/closes (AC-5). Late after due is allowed until close.
func CheckPlayWindow(w AssignmentWindow, now time.Time) error {
	if w.OpensAt != nil && now.Before(w.OpensAt.UTC()) {
		return ErrNotYetOpen
	}
	if w.ClosesAt != nil && !now.Before(w.ClosesAt.UTC()) {
		return ErrClosed
	}
	return nil
}

// IsLate reports whether now is after due_at (still may play until closes_at).
func IsLate(w AssignmentWindow, now time.Time) bool {
	if w.DueAt == nil {
		return false
	}
	return !now.Before(w.DueAt.UTC())
}

// EffectiveAttemptsAllowed adds accommodation extra attempts.
func EffectiveAttemptsAllowed(base, extra int) int {
	if base < 1 {
		base = 1
	}
	if extra < 0 {
		extra = 0
	}
	n := base + extra
	if n > 100 {
		return 100
	}
	return n
}

// CheckAttempts returns ErrOutOfAttempts when used >= allowed.
func CheckAttempts(used, allowed int) error {
	if used >= allowed {
		return ErrOutOfAttempts
	}
	return nil
}

// ApplyGradePolicy computes the gradebook score from attempt scores (AC-6).
func ApplyGradePolicy(scores []int, policy GradePolicy) float64 {
	if len(scores) == 0 {
		return 0
	}
	switch NormalizeGradePolicy(string(policy)) {
	case GradePolicyLast:
		return float64(scores[len(scores)-1])
	case GradePolicyAverage:
		sum := 0
		for _, s := range scores {
			sum += s
		}
		return float64(sum) / float64(len(scores))
	default: // best
		best := scores[0]
		for _, s := range scores[1:] {
			if s > best {
				best = s
			}
		}
		return float64(best)
	}
}
