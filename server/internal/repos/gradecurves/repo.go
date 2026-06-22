package gradecurves

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/gradecurve"
)

// Row is one applied (or reverted) grade curve.
type Row struct {
	ID             uuid.UUID
	CourseID       uuid.UUID
	ModuleItemID   *uuid.UUID
	Scope          string
	Method         gradecurve.Method
	ParamsJSON     json.RawMessage
	AllowAboveMax  bool
	AppliedBy      uuid.UUID
	AppliedAt      time.Time
	RevertedAt     *time.Time
}

// AdjustmentRow stores raw vs adjusted for one student.
type AdjustmentRow struct {
	ID            uuid.UUID
	CurveID       uuid.UUID
	StudentID     uuid.UUID
	RawScore      float64
	AdjustedScore float64
}

// CellScore is a grade cell used as curve input.
type CellScore struct {
	StudentID uuid.UUID
	Points    float64
	Excused   bool
}

// ActiveCurveSummary is exposed on the gradebook grid.
type ActiveCurveSummary struct {
	CurveID   uuid.UUID
	Method    string
	AppliedAt time.Time
}

// ListCellsForAssignment returns scored cells for one assignment.
func ListCellsForAssignment(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) ([]CellScore, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	rows, err := pool.Query(ctx, `
SELECT student_user_id, points_earned, excused
FROM course.course_grades
WHERE course_id = $1 AND module_item_id = $2
`, courseID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CellScore
	for rows.Next() {
		var c CellScore
		if err := rows.Scan(&c.StudentID, &c.Points, &c.Excused); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// RawScoresForActiveCurve returns student -> raw score from the active curve, if any.
func RawScoresForActiveCurve(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (uuid.UUID, map[uuid.UUID]float64, error) {
	curve, err := GetActiveForAssignment(ctx, pool, itemID)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if curve == nil {
		return uuid.Nil, nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT student_id, raw_score
FROM course.grade_curve_adjustments
WHERE curve_id = $1
`, curve.ID)
	if err != nil {
		return uuid.Nil, nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]float64)
	for rows.Next() {
		var sid uuid.UUID
		var raw float64
		if err := rows.Scan(&sid, &raw); err != nil {
			return uuid.Nil, nil, err
		}
		out[sid] = raw
	}
	return curve.ID, out, rows.Err()
}

// GetActiveForAssignment returns the non-reverted curve for an assignment, if any.
func GetActiveForAssignment(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (*Row, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r Row
	var moduleItemID uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id, course_id, module_item_id, scope, method, params_json, allow_above_max, applied_by, applied_at, reverted_at
FROM course.grade_curves
WHERE module_item_id = $1 AND scope = 'assessment' AND reverted_at IS NULL
ORDER BY applied_at DESC
LIMIT 1
`, itemID).Scan(
		&r.ID, &r.CourseID, &moduleItemID, &r.Scope, &r.Method, &r.ParamsJSON,
		&r.AllowAboveMax, &r.AppliedBy, &r.AppliedAt, &r.RevertedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.ModuleItemID = &moduleItemID
	return &r, nil
}

// ListActiveForCourse maps module_item_id -> active curve summary for a course.
func ListActiveForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (map[string]ActiveCurveSummary, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	rows, err := pool.Query(ctx, `
SELECT id, module_item_id, method, applied_at
FROM course.grade_curves
WHERE course_id = $1 AND scope = 'assessment' AND reverted_at IS NULL AND module_item_id IS NOT NULL
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]ActiveCurveSummary)
	for rows.Next() {
		var id, itemID uuid.UUID
		var method string
		var appliedAt time.Time
		if err := rows.Scan(&id, &itemID, &method, &appliedAt); err != nil {
			return nil, err
		}
		out[itemID.String()] = ActiveCurveSummary{CurveID: id, Method: method, AppliedAt: appliedAt}
	}
	return out, rows.Err()
}

// GetByID loads a curve by primary key.
func GetByID(ctx context.Context, pool *pgxpool.Pool, curveID uuid.UUID) (*Row, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r Row
	var moduleItemID *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id, course_id, module_item_id, scope, method, params_json, allow_above_max, applied_by, applied_at, reverted_at
FROM course.grade_curves
WHERE id = $1
`, curveID).Scan(
		&r.ID, &r.CourseID, &moduleItemID, &r.Scope, &r.Method, &r.ParamsJSON,
		&r.AllowAboveMax, &r.AppliedBy, &r.AppliedAt, &r.RevertedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.ModuleItemID = moduleItemID
	return &r, nil
}

// ListAdjustmentsForCurve returns all adjustments for a curve.
func ListAdjustmentsForCurve(ctx context.Context, pool *pgxpool.Pool, curveID uuid.UUID) ([]AdjustmentRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	rows, err := pool.Query(ctx, `
SELECT id, curve_id, student_id, raw_score, adjusted_score
FROM course.grade_curve_adjustments
WHERE curve_id = $1
ORDER BY student_id
`, curveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdjustmentRow
	for rows.Next() {
		var a AdjustmentRow
		if err := rows.Scan(&a.ID, &a.CurveID, &a.StudentID, &a.RawScore, &a.AdjustedScore); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ApplyInput is everything needed to persist a new curve.
type ApplyInput struct {
	CourseID      uuid.UUID
	ModuleItemID  uuid.UUID
	Method        gradecurve.Method
	ParamsJSON    json.RawMessage
	AllowAboveMax bool
	AppliedBy     uuid.UUID
	Adjustments   []AdjustmentRow
}

// Apply persists a curve and updates grade cells in one transaction.
func Apply(ctx context.Context, pool *pgxpool.Pool, in ApplyInput, updateGrade func(ctx context.Context, tx pgx.Tx, studentID uuid.UUID, points float64) error) (uuid.UUID, error) {
	if pool == nil {
		return uuid.Nil, errors.New("nil pool")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Revert any existing active curve for this assignment.
	var prevID uuid.UUID
	err = tx.QueryRow(ctx, `
SELECT id FROM course.grade_curves
WHERE module_item_id = $1 AND scope = 'assessment' AND reverted_at IS NULL
FOR UPDATE
`, in.ModuleItemID).Scan(&prevID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	if err == nil {
		if _, err := tx.Exec(ctx, `UPDATE course.grade_curves SET reverted_at = NOW() WHERE id = $1`, prevID); err != nil {
			return uuid.Nil, err
		}
	}

	var curveID uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO course.grade_curves (
	course_id, module_item_id, scope, method, params_json, allow_above_max, applied_by
) VALUES ($1, $2, 'assessment', $3, $4, $5, $6)
RETURNING id
`, in.CourseID, in.ModuleItemID, string(in.Method), in.ParamsJSON, in.AllowAboveMax, in.AppliedBy).Scan(&curveID)
	if err != nil {
		return uuid.Nil, err
	}

	for _, adj := range in.Adjustments {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.grade_curve_adjustments (curve_id, student_id, raw_score, adjusted_score)
VALUES ($1, $2, $3, $4)
`, curveID, adj.StudentID, adj.RawScore, adj.AdjustedScore); err != nil {
			return uuid.Nil, err
		}
		if err := updateGrade(ctx, tx, adj.StudentID, adj.AdjustedScore); err != nil {
			return uuid.Nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return curveID, nil
}

// Revert restores raw scores and marks the curve reverted.
func Revert(ctx context.Context, pool *pgxpool.Pool, curveID uuid.UUID, restoreGrade func(ctx context.Context, tx pgx.Tx, studentID uuid.UUID, points float64) error) error {
	if pool == nil {
		return errors.New("nil pool")
	}
	curve, err := GetByID(ctx, pool, curveID)
	if err != nil {
		return err
	}
	if curve == nil {
		return pgx.ErrNoRows
	}
	if curve.RevertedAt != nil {
		return errors.New("curve already reverted")
	}
	adjs, err := ListAdjustmentsForCurve(ctx, pool, curveID)
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `UPDATE course.grade_curves SET reverted_at = NOW() WHERE id = $1`, curveID); err != nil {
		return err
	}
	for _, adj := range adjs {
		if err := restoreGrade(ctx, tx, adj.StudentID, adj.RawScore); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// BuildScoreInputs merges cell scores with any active curve raw preservation.
func BuildScoreInputs(cells []CellScore, rawByStudent map[uuid.UUID]float64) []gradecurve.ScoreInput {
	out := make([]gradecurve.ScoreInput, 0, len(cells))
	for _, c := range cells {
		raw := c.Points
		if preserved, ok := rawByStudent[c.StudentID]; ok {
			raw = preserved
		}
		out = append(out, gradecurve.ScoreInput{
			StudentID: c.StudentID,
			RawScore:  raw,
			Excused:   c.Excused,
		})
	}
	return out
}
