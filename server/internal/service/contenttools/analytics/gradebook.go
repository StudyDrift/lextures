package analytics

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/coursegrades"
)

// LatePolicy values for grade links.
const (
	LateAccept       = "accept"
	LateAcceptMarked = "accept_marked"
	LateReject       = "reject"
)

// GradeLink is the opt-in bridge config for one instance.
type GradeLink struct {
	InstanceID       uuid.UUID  `json:"instanceId"`
	AssignmentItemID *uuid.UUID `json:"assignmentItemId,omitempty"`
	OutcomeID        *uuid.UUID `json:"outcomeId,omitempty"`
	PointsPossible   *float64   `json:"pointsPossible,omitempty"`
	CountsForGrade   bool       `json:"countsForGrade"`
	LatePolicy       string     `json:"latePolicy"`
	EnabledBy        *uuid.UUID `json:"enabledBy,omitempty"`
}

// PointsFromScorePct converts a 0–100 score percent into gradebook points.
func PointsFromScorePct(scorePct *float64, pointsPossible float64) float64 {
	if scorePct == nil || pointsPossible <= 0 {
		return 0
	}
	pts := (*scorePct / 100.0) * pointsPossible
	return math.Round(pts*100) / 100
}

// PushGrade writes/updates a gradebook cell for a bridged tool completion.
func PushGrade(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, studentUserID, assignmentItemID uuid.UUID,
	points float64,
) error {
	if pool == nil {
		return fmt.Errorf("nil pool")
	}
	return coursegrades.UpsertCell(ctx, pool, courseID, studentUserID, assignmentItemID, points, nil, nil, "automatic")
}

// RevertGrade clears a gradebook cell for a bridged instance reset (same transaction preferred).
func RevertGrade(
	ctx context.Context,
	tx pgx.Tx,
	pool *pgxpool.Pool,
	courseID, studentUserID, assignmentItemID uuid.UUID,
) error {
	if tx != nil {
		_, err := tx.Exec(ctx, `
DELETE FROM course.course_grades
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
`, courseID, studentUserID, assignmentItemID)
		return err
	}
	if pool == nil {
		return fmt.Errorf("nil pool")
	}
	return coursegrades.DeleteCell(ctx, pool, courseID, studentUserID, assignmentItemID)
}

// GradeEffectAction describes how a reset affects the gradebook.
const (
	GradeActionUnchanged = "unchanged"
	GradeActionReverted  = "reverted"
	GradeActionRefused   = "refused"
)

// ClassifyBridgedGradeEffect returns the grade effect for a reset when a link exists.
func ClassifyBridgedGradeEffect(enrollmentID uuid.UUID, linked bool, hadScore bool) (action string) {
	if !linked || !hadScore {
		return GradeActionUnchanged
	}
	return GradeActionReverted
}
