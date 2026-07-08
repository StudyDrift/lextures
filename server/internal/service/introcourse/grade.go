package introcourse

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
)

// Grade policy values persisted on settings.intro_course_items.grade_policy.
const (
	GradePolicyQuizAutoscore    = "quiz_autoscore"
	GradePolicyCompletionFull   = "completion_full"
	GradePolicyGraderAgent      = "grader_agent"
)

// GraderAgentRequest asks the HTTP layer to enqueue feedback-only grading for a submission.
type GraderAgentRequest struct {
	CourseID     uuid.UUID
	ItemID       uuid.UUID
	SubmissionID uuid.UUID
	CourseCode   string
}

// OnQuizAttempt upserts the student's best quiz score into course_grades (keep-highest policy).
func OnQuizAttempt(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID, studentID, itemID uuid.UUID) error {
	if pool == nil || !cfg.IntroCourseEnabled {
		return nil
	}
	ok, err := icrepo.IsIntroCourseID(ctx, pool, courseID)
	if err != nil || !ok {
		return err
	}
	quizRow, err := coursemodulequizzes.GetForCourseItem(ctx, pool, courseID, itemID)
	if err != nil || quizRow == nil {
		return err
	}
	policy := quizRow.GradeAttemptPolicy
	if policy == "" {
		policy = "highest"
	}
	points, ready, err := quizattempts.PolicyPointsForStudent(ctx, pool, courseID, itemID, studentID, policy)
	if err != nil {
		recordAutograde("quiz", "error")
		return err
	}
	if !ready {
		recordAutograde("quiz", "pending")
		return nil
	}
	if err := coursegrades.UpsertCellWithFlags(
		ctx, pool, courseID, studentID, itemID, points, nil, nil, nil, "automatic", false,
	); err != nil {
		recordAutograde("quiz", "error")
		return err
	}
	recordAutograde("quiz", "success")
	recordGradeWrite()
	logGradeWrite(studentID, itemID, points)
	_, _ = RecheckCompletion(ctx, pool, cfg, courseID, studentID)
	return nil
}

// OnAssignmentSubmit awards completion credit and optionally requests grader-agent feedback.
func OnAssignmentSubmit(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg config.Config,
	courseID, studentID, itemID uuid.UUID,
	courseCode string,
) (GraderAgentRequest, error) {
	var empty GraderAgentRequest
	if pool == nil || !cfg.IntroCourseEnabled {
		return empty, nil
	}
	ok, err := icrepo.IsIntroCourseID(ctx, pool, courseID)
	if err != nil || !ok {
		return empty, err
	}

	assignRow, err := coursemoduleassignments.GetForCourseItem(ctx, pool, courseID, itemID)
	if err != nil || assignRow == nil {
		return empty, err
	}
	points := completionPoints(assignRow.PointsWorth)
	if err := coursegrades.UpsertCellWithFlags(
		ctx, pool, courseID, studentID, itemID, points, nil, nil, nil, "automatic", false,
	); err != nil {
		recordAutograde("assignment", "error")
		return empty, err
	}
	recordAutograde("assignment", "success")
	recordGradeWrite()
	logGradeWrite(studentID, itemID, points)
	_, _ = RecheckCompletion(ctx, pool, cfg, courseID, studentID)

	policy, err := lookupItemGradePolicy(ctx, pool, itemID)
	if err != nil {
		return empty, err
	}
	if policy != GradePolicyGraderAgent || !graderAgentFeedbackEnabled(cfg) {
		return empty, nil
	}
	var submissionID uuid.UUID
	err = pool.QueryRow(ctx, `
SELECT id FROM course.module_assignment_submissions
WHERE course_id = $1 AND module_item_id = $2 AND submitted_by = $3
ORDER BY submitted_at DESC LIMIT 1
`, courseID, itemID, studentID).Scan(&submissionID)
	if err != nil {
		return empty, nil
	}
	return GraderAgentRequest{
		CourseID:     courseID,
		ItemID:       itemID,
		SubmissionID: submissionID,
		CourseCode:   courseCode,
	}, nil
}

func lookupItemGradePolicy(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (string, error) {
	var policy *string
	err := pool.QueryRow(ctx, `
SELECT grade_policy FROM settings.intro_course_items WHERE structure_item_id = $1
`, itemID).Scan(&policy)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if policy == nil {
		return "", nil
	}
	return *policy, nil
}

func completionPoints(pointsWorth *int) float64 {
	if pointsWorth == nil || *pointsWorth <= 0 {
		return 1
	}
	return float64(*pointsWorth)
}

func graderAgentFeedbackEnabled(cfg config.Config) bool {
	return cfg.GraderAgentEnabled && cfg.GraderAgentTextEntryGradingEnabled
}

// SyncGradingConfig reconciles assignment-group weights and grader-agent config for graded items.
func SyncGradingConfig(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, cfg config.Config) error {
	if err := icrepo.EnsureAssignmentGroups(ctx, tx, courseID); err != nil {
		return err
	}
	if !graderAgentFeedbackEnabled(cfg) {
		return nil
	}
	rows, err := icrepo.ListContentItems(ctx, tx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.GradePolicy == nil || *row.GradePolicy != GradePolicyGraderAgent {
			continue
		}
		if err := upsertGraderAgentConfig(ctx, tx, courseID, row.StructureItemID); err != nil {
			return fmt.Errorf("grader agent config %s: %w", row.Slug, err)
		}
	}
	return nil
}

func upsertGraderAgentConfig(ctx context.Context, tx pgx.Tx, courseID, itemID uuid.UUID) error {
	prompt := "Provide brief, encouraging feedback on this intro-course reflection. Do not change the student's grade."
	_, err := tx.Exec(ctx, `
INSERT INTO assessment.grading_agent_configs (
    course_id, module_item_id, status, prompt,
    include_assignment_content, include_rubric, auto_grade_new, post_policy,
    created_by, updated_at
) VALUES ($1, $2, 'accepted'::assessment.grading_agent_status, $3, TRUE, FALSE, TRUE, 'automatic', $4, NOW())
ON CONFLICT (module_item_id) DO UPDATE SET
    status = EXCLUDED.status,
    prompt = EXCLUDED.prompt,
    include_assignment_content = EXCLUDED.include_assignment_content,
    auto_grade_new = EXCLUDED.auto_grade_new,
    post_policy = EXCLUDED.post_policy,
    updated_at = NOW()
`, courseID, itemID, prompt, SystemUserID)
	return err
}

func logGradeWrite(studentID, itemID uuid.UUID, points float64) {
	sum := sha256.Sum256([]byte(studentID.String()))
	slog.Debug("intro course grade write",
		"student_hash", hex.EncodeToString(sum[:8]),
		"item_id", itemID,
		"points", points,
	)
}