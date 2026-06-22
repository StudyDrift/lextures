package quizautosubmit

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/questionbank"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	"github.com/lextures/lextures/server/internal/service/learnerstate"
	"github.com/lextures/lextures/server/internal/service/learningevents"
)

const defaultSweepBatch = 200

// SweepExpiredAttempts finalizes timed quiz attempts past deadline (Rust `quiz_auto_submit::sweep_expired_attempts`).
func SweepExpiredAttempts(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time, limit int64) (int, error) {
	if pool == nil {
		return 0, nil
	}
	if limit < 1 {
		limit = defaultSweepBatch
	}
	ids, err := quizattempts.ListExpiredInProgressAttemptIDs(ctx, pool, now, limit)
	if err != nil {
		return 0, err
	}
	var n int
	for _, id := range ids {
		att, err := quizattempts.GetAttemptForSweep(ctx, pool, id)
		if err != nil || att == nil {
			continue
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return n, err
		}
		earned, possible, err := quizattempts.SumResponsePointsForAttempt(ctx, tx, id)
		if err != nil {
			_ = tx.Rollback(ctx)
			return n, err
		}

		if !att.IsAdaptive {
			if err := applyMasteryForAutoSubmit(ctx, pool, tx, cfg, att.CourseID, att.StructureItemID, att.StudentUserID, id); err != nil {
				_ = tx.Rollback(ctx)
				return n, err
			}
		}

		var score float32
		if possible > 0 {
			score = float32(math.Min(100, math.Max(0, (earned/possible)*100)))
		}
		ok, err := quizattempts.FinalizeAttemptAutoSubmitted(ctx, tx, id, now, earned, possible, score)
		if err != nil {
			_ = tx.Rollback(ctx)
			return n, err
		}
		if err := tx.Commit(ctx); err != nil {
			return n, err
		}
		if ok {
			n++
			slog.Info("quiz attempt auto-submitted after deadline", "attempt_id", id)
			learningevents.EmitQuizGradedAsync(pool, cfg, id)
		}
	}
	return n, nil
}

func applyMasteryForAutoSubmit(
	ctx context.Context,
	pool *pgxpool.Pool,
	tx pgx.Tx,
	cfg config.Config,
	courseID, structureItemID, studentUserID, attemptID uuid.UUID,
) error {
	meta, err := course.GetCourseQuizMeta(ctx, pool, courseID)
	if err != nil || meta == nil {
		return err
	}
	row, err := coursemodulequizzes.GetForCourseItem(ctx, pool, courseID, structureItemID)
	if err != nil || row == nil {
		return err
	}
	attemptPtr := &attemptID
	questions, _, err := questionbank.ResolveDeliveryQuestionsForGet(
		ctx, pool, courseID, structureItemID, meta.QuestionBankEnabled, row.Questions, attemptPtr, false,
	)
	if err != nil {
		return err
	}
	responses, err := quizattempts.ListResponses(ctx, pool, attemptID)
	if err != nil {
		return err
	}
	return learnerstate.ApplyMasteryFromSavedResponses(ctx, pool, tx, learnerstate.ApplyMasteryParams{
		CourseID:                      courseID,
		UserID:                        studentUserID,
		AttemptID:                     attemptID,
		Questions:                     questions,
		Responses:                     responses,
		HintScaffoldingEnabled:        meta.HintScaffoldingEnabled,
		MisconceptionDetectionEnabled: meta.MisconceptionDetectionEnabled,
		LearnerModelEnabled:           cfg.AdaptiveLearnerModelEnabled,
		EMAAlpha:                      cfg.LearnerModelEMAAlpha,
	})
}