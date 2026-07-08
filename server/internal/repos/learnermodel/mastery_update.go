package learnermodel

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MasteryUpdateInput is one concept mastery touch from quiz grading (Rust LearnerModelUpdateInput).
type MasteryUpdateInput struct {
	UserID             uuid.UUID
	AttemptID          uuid.UUID
	CourseID           uuid.UUID
	ConceptID          uuid.UUID
	Score              float64
	QuestionIndex      int32
	EMAAlpha           float64
	EMAAlphaMultiplier float64
}

// DecayAdjustedMastery applies time decay to stored mastery (Rust effective_mastery_engine).
func DecayAdjustedMastery(stored float64, lastSeenAt *time.Time, decayLambda float64) float64 {
	return DecayAdjustedMasteryAt(stored, lastSeenAt, decayLambda, time.Now().UTC())
}

// DecayAdjustedMasteryAt applies decay relative to an explicit reference time (for tests and batch derive).
func DecayAdjustedMasteryAt(stored float64, lastSeenAt *time.Time, decayLambda float64, now time.Time) float64 {
	if lastSeenAt == nil {
		return clamp01(stored)
	}
	days := now.Sub(lastSeenAt.UTC()).Seconds() / 86400.0
	if days <= 0 {
		return clamp01(stored)
	}
	return clamp01(stored * math.Exp(-decayLambda*days))
}

func clamp01(v float64) float64 {
	return math.Min(1, math.Max(0, v))
}

func needsReviewAt(now time.Time, mastery float64) time.Time {
	switch {
	case mastery < 0.5:
		return now.Add(3 * 24 * time.Hour)
	case mastery < 0.8:
		return now.Add(14 * 24 * time.Hour)
	default:
		return now.Add(30 * 24 * time.Hour)
	}
}

// ApplyMasteryUpdateInTx records one quiz-grade mastery event and upserts learner state
// (Rust apply_mastery_update_in_tx).
func ApplyMasteryUpdateInTx(ctx context.Context, tx pgx.Tx, input MasteryUpdateInput) error {
	idempotencyKey := fmt.Sprintf("quiz_grade:%s:%s:%d", input.AttemptID, input.ConceptID, input.QuestionIndex)

	var decayLambda float64
	err := tx.QueryRow(ctx, `
SELECT (c.decay_lambda)::float8
FROM course.concepts c
WHERE c.id = $1
FOR UPDATE
`, input.ConceptID).Scan(&decayLambda)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("concept decay: %w", err)
	}

	var storedMastery float64
	var lastSeenAt *time.Time
	err = tx.QueryRow(ctx, `
SELECT COALESCE((mastery)::float8, 0.0), last_seen_at
FROM course.learner_concept_states
WHERE user_id = $1 AND concept_id = $2
FOR UPDATE
`, input.UserID, input.ConceptID).Scan(&storedMastery, &lastSeenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		storedMastery = 0
		lastSeenAt = nil
	} else if err != nil {
		return fmt.Errorf("learner state: %w", err)
	}

	mOldEff := DecayAdjustedMastery(storedMastery, lastSeenAt, decayLambda)
	score := clamp01(input.Score)
	alpha := clamp01(input.EMAAlpha * clamp(input.EMAAlphaMultiplier, 0.01, 2.0))
	if alpha < 0.01 {
		alpha = 0.01
	}
	mNew := clamp01(mOldEff*(1-alpha) + score*alpha)
	delta := mNew - mOldEff

	now := time.Now().UTC()
	reviewAt := needsReviewAt(now, mNew)

	var eventID uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO course.learner_concept_events (
  user_id, concept_id, attempt_id, delta, mastery_after, source, idempotency_key
)
VALUES ($1, $2, $3, $4::numeric, $5::numeric, 'quiz_grade', $6)
ON CONFLICT (idempotency_key) DO NOTHING
RETURNING id
`, input.UserID, input.ConceptID, input.AttemptID, delta, mNew, idempotencyKey).Scan(&eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("learner concept event: %w", err)
	}

	_, err = tx.Exec(ctx, `
INSERT INTO course.learner_concept_states (
  user_id, concept_id, mastery, attempt_count, last_seen_at, needs_review_at, updated_at
)
VALUES ($1, $2, $3::numeric, 1, $4, $5, NOW())
ON CONFLICT (user_id, concept_id) DO UPDATE SET
  mastery = EXCLUDED.mastery,
  attempt_count = course.learner_concept_states.attempt_count + 1,
  last_seen_at = EXCLUDED.last_seen_at,
  needs_review_at = EXCLUDED.needs_review_at,
  updated_at = NOW()
`, input.UserID, input.ConceptID, mNew, now, reviewAt)
	if err != nil {
		return fmt.Errorf("learner concept state: %w", err)
	}
	return nil
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(hi, math.Max(lo, v))
}