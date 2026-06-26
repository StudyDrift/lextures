// Package peerreview provides SQL access for plan 3.15 peer review.
package peerreview

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/peerreview"
)

type ConfigRow struct {
	ID                 uuid.UUID
	AssignmentID       uuid.UUID
	CourseID           uuid.UUID
	ReviewsPerReviewer int
	Anonymity          peerreview.AnonymityMode
	OpensAt            *time.Time
	ClosesAt           *time.Time
	GradeMode          peerreview.GradeMode
	BlendWeight        float64
	Aggregation        peerreview.Aggregation
	ExcludeSameGroup   bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type AllocationRow struct {
	ID                   uuid.UUID
	ConfigID             uuid.UUID
	ReviewerEnrollmentID uuid.UUID
	TargetSubmissionID   uuid.UUID
	Status               peerreview.AllocationStatus
	AssignedAt           time.Time
	CourseID             uuid.UUID
	CourseCode           string
	AssignmentID         uuid.UUID
	TargetUserID         uuid.UUID
	ReviewerUserID       uuid.UUID
}

type ReviewRow struct {
	ID               uuid.UUID
	AllocationID     uuid.UUID
	Score            *float64
	RubricScoresJSON []byte
	Comments         *string
	SubmittedAt      time.Time
}

type SubmitterRow struct {
	EnrollmentID uuid.UUID
	UserID       uuid.UUID
	SubmissionID uuid.UUID
}

type SubmissionReviewCount struct {
	SubmissionID uuid.UUID
	ReviewCount  int
}

func scanConfig(scanner interface{ Scan(...any) error }) (*ConfigRow, error) {
	var r ConfigRow
	var anonymity, gradeMode, aggregation string
	err := scanner.Scan(
		&r.ID, &r.AssignmentID, &r.CourseID, &r.ReviewsPerReviewer,
		&anonymity, &r.OpensAt, &r.ClosesAt, &gradeMode, &r.BlendWeight,
		&aggregation, &r.ExcludeSameGroup, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Anonymity = peerreview.AnonymityMode(anonymity)
	r.GradeMode = peerreview.GradeMode(gradeMode)
	r.Aggregation = peerreview.Aggregation(aggregation)
	return &r, nil
}

const configSelect = `
SELECT c.id, c.assignment_id, si.course_id, c.reviews_per_reviewer,
       c.anonymity::text, c.opens_at, c.closes_at, c.grade_mode::text, c.blend_weight,
       c.aggregation::text, c.exclude_same_group, c.created_at, c.updated_at
FROM course.peer_review_configs c
JOIN course.course_structure_items si ON si.id = c.assignment_id
`

func GetConfigByAssignment(ctx context.Context, pool *pgxpool.Pool, courseID, assignmentID uuid.UUID) (*ConfigRow, error) {
	q := configSelect + `WHERE c.assignment_id = $1`
	args := []any{assignmentID}
	if courseID != uuid.Nil {
		q += ` AND si.course_id = $2`
		args = append(args, courseID)
	}
	row, err := scanConfig(pool.QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return row, err
}

type UpsertConfigInput struct {
	AssignmentID       uuid.UUID
	ReviewsPerReviewer int
	Anonymity          peerreview.AnonymityMode
	OpensAt            *time.Time
	ClosesAt           *time.Time
	GradeMode          peerreview.GradeMode
	BlendWeight        float64
	Aggregation        peerreview.Aggregation
	ExcludeSameGroup   bool
}

func UpsertConfig(ctx context.Context, pool *pgxpool.Pool, in UpsertConfigInput) (*ConfigRow, error) {
	var configID uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO course.peer_review_configs (
    assignment_id, reviews_per_reviewer, anonymity, opens_at, closes_at,
    grade_mode, blend_weight, aggregation, exclude_same_group, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
ON CONFLICT (assignment_id) DO UPDATE SET
    reviews_per_reviewer = EXCLUDED.reviews_per_reviewer,
    anonymity = EXCLUDED.anonymity,
    opens_at = EXCLUDED.opens_at,
    closes_at = EXCLUDED.closes_at,
    grade_mode = EXCLUDED.grade_mode,
    blend_weight = EXCLUDED.blend_weight,
    aggregation = EXCLUDED.aggregation,
    exclude_same_group = EXCLUDED.exclude_same_group,
    updated_at = NOW()
RETURNING id
`, in.AssignmentID, in.ReviewsPerReviewer, string(in.Anonymity), in.OpensAt, in.ClosesAt,
		string(in.GradeMode), in.BlendWeight, string(in.Aggregation), in.ExcludeSameGroup).Scan(&configID)
	if err != nil {
		return nil, err
	}
	_ = configID
	return GetConfigByAssignment(ctx, pool, uuid.Nil, in.AssignmentID)
}

func ListSubmittersForAssignment(ctx context.Context, pool *pgxpool.Pool, courseID, assignmentID uuid.UUID) ([]SubmitterRow, error) {
	rows, err := pool.Query(ctx, `
SELECT ce.id, ce.user_id, s.id
FROM course.module_assignment_submissions s
JOIN course.course_enrollments ce
    ON ce.user_id = s.submitted_by AND ce.course_id = s.course_id AND ce.active
JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE s.course_id = $1 AND s.module_item_id = $2
ORDER BY s.submitted_at ASC, s.id ASC
`, courseID, assignmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SubmitterRow, 0)
	for rows.Next() {
		var r SubmitterRow
		if err := rows.Scan(&r.EnrollmentID, &r.UserID, &r.SubmissionID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func ShareGroup(ctx context.Context, pool *pgxpool.Pool, enrollmentA, enrollmentB uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM course.enrollment_group_memberships m1
    JOIN course.enrollment_group_memberships m2
        ON m1.group_id = m2.group_id AND m1.group_set_id = m2.group_set_id
    WHERE m1.enrollment_id = $1 AND m2.enrollment_id = $2
)
`, enrollmentA, enrollmentB).Scan(&ok)
	return ok, err
}

func CountReviewsPerSubmission(ctx context.Context, pool *pgxpool.Pool, configID uuid.UUID) (map[uuid.UUID]int, error) {
	rows, err := pool.Query(ctx, `
SELECT target_submission_id, COUNT(*)::int
FROM course.peer_review_allocations
WHERE config_id = $1
GROUP BY target_submission_id
`, configID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]int)
	for rows.Next() {
		var sid uuid.UUID
		var n int
		if err := rows.Scan(&sid, &n); err != nil {
			return nil, err
		}
		out[sid] = n
	}
	return out, rows.Err()
}

func InsertAllocation(ctx context.Context, tx pgx.Tx, configID, reviewerEnrollmentID, targetSubmissionID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
INSERT INTO course.peer_review_allocations (config_id, reviewer_enrollment_id, target_submission_id)
VALUES ($1, $2, $3)
ON CONFLICT (config_id, reviewer_enrollment_id, target_submission_id) DO NOTHING
`, configID, reviewerEnrollmentID, targetSubmissionID)
	return err
}

func CountReviewerAllocations(ctx context.Context, pool *pgxpool.Pool, configID, reviewerEnrollmentID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM course.peer_review_allocations
WHERE config_id = $1 AND reviewer_enrollment_id = $2
`, configID, reviewerEnrollmentID).Scan(&n)
	return n, err
}

func ListAllocationsForReviewer(ctx context.Context, pool *pgxpool.Pool, reviewerEnrollmentID uuid.UUID) ([]AllocationRow, error) {
	rows, err := pool.Query(ctx, `
SELECT a.id, a.config_id, a.reviewer_enrollment_id, a.target_submission_id,
       a.status::text, a.assigned_at,
       si.course_id, co.course_code, c.assignment_id,
       s.submitted_by, rev_ce.user_id
FROM course.peer_review_allocations a
JOIN course.peer_review_configs c ON c.id = a.config_id
JOIN course.course_structure_items si ON si.id = c.assignment_id
JOIN course.courses co ON co.id = si.course_id
JOIN course.module_assignment_submissions s ON s.id = a.target_submission_id
JOIN course.course_enrollments rev_ce ON rev_ce.id = a.reviewer_enrollment_id
WHERE a.reviewer_enrollment_id = $1
ORDER BY a.assigned_at ASC
`, reviewerEnrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAllocationRows(rows)
}

func GetAllocationByID(ctx context.Context, pool *pgxpool.Pool, allocationID uuid.UUID) (*AllocationRow, error) {
	row, err := scanAllocation(pool.QueryRow(ctx, `
SELECT a.id, a.config_id, a.reviewer_enrollment_id, a.target_submission_id,
       a.status::text, a.assigned_at,
       si.course_id, co.course_code, c.assignment_id,
       s.submitted_by, rev_ce.user_id
FROM course.peer_review_allocations a
JOIN course.peer_review_configs c ON c.id = a.config_id
JOIN course.course_structure_items si ON si.id = c.assignment_id
JOIN course.courses co ON co.id = si.course_id
JOIN course.module_assignment_submissions s ON s.id = a.target_submission_id
JOIN course.course_enrollments rev_ce ON rev_ce.id = a.reviewer_enrollment_id
WHERE a.id = $1
`, allocationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return row, err
}

func scanAllocation(scanner interface{ Scan(...any) error }) (*AllocationRow, error) {
	var r AllocationRow
	var status string
	err := scanner.Scan(
		&r.ID, &r.ConfigID, &r.ReviewerEnrollmentID, &r.TargetSubmissionID,
		&status, &r.AssignedAt, &r.CourseID, &r.CourseCode, &r.AssignmentID,
		&r.TargetUserID, &r.ReviewerUserID,
	)
	if err != nil {
		return nil, err
	}
	r.Status = peerreview.AllocationStatus(status)
	return &r, nil
}

type pgxRows interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func scanAllocationRows(rows pgxRows) ([]AllocationRow, error) {
	out := make([]AllocationRow, 0)
	for rows.Next() {
		r, err := scanAllocation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func ListAllocationsForAssignment(ctx context.Context, pool *pgxpool.Pool, configID uuid.UUID) ([]AllocationRow, error) {
	rows, err := pool.Query(ctx, `
SELECT a.id, a.config_id, a.reviewer_enrollment_id, a.target_submission_id,
       a.status::text, a.assigned_at,
       si.course_id, co.course_code, c.assignment_id,
       s.submitted_by, rev_ce.user_id
FROM course.peer_review_allocations a
JOIN course.peer_review_configs c ON c.id = a.config_id
JOIN course.course_structure_items si ON si.id = c.assignment_id
JOIN course.courses co ON co.id = si.course_id
JOIN course.module_assignment_submissions s ON s.id = a.target_submission_id
JOIN course.course_enrollments rev_ce ON rev_ce.id = a.reviewer_enrollment_id
WHERE a.config_id = $1
ORDER BY a.assigned_at ASC
`, configID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAllocationRows(rows)
}

func UpdateAllocationStatus(ctx context.Context, pool *pgxpool.Pool, allocationID uuid.UUID, status peerreview.AllocationStatus) error {
	_, err := pool.Exec(ctx, `
UPDATE course.peer_review_allocations SET status = $2 WHERE id = $1
`, allocationID, string(status))
	return err
}

func UpsertReview(ctx context.Context, pool *pgxpool.Pool, allocationID uuid.UUID, score *float64, rubricScores map[string]float64, comments *string) (*ReviewRow, error) {
	var rubricJSON []byte
	if len(rubricScores) > 0 {
		var err error
		rubricJSON, err = json.Marshal(rubricScores)
		if err != nil {
			return nil, err
		}
	}
	var r ReviewRow
	err := pool.QueryRow(ctx, `
INSERT INTO course.peer_reviews (allocation_id, score, rubric_scores_json, comments, submitted_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (allocation_id) DO UPDATE SET
    score = EXCLUDED.score,
    rubric_scores_json = EXCLUDED.rubric_scores_json,
    comments = EXCLUDED.comments,
    submitted_at = NOW()
RETURNING id, allocation_id, score, rubric_scores_json, comments, submitted_at
`, allocationID, score, rubricJSON, comments).Scan(
		&r.ID, &r.AllocationID, &r.Score, &r.RubricScoresJSON, &r.Comments, &r.SubmittedAt,
	)
	return &r, err
}

func ListReviewsForConfig(ctx context.Context, pool *pgxpool.Pool, configID uuid.UUID) ([]ReviewRow, error) {
	rows, err := pool.Query(ctx, `
SELECT pr.id, pr.allocation_id, pr.score, pr.rubric_scores_json, pr.comments, pr.submitted_at
FROM course.peer_reviews pr
JOIN course.peer_review_allocations a ON a.id = pr.allocation_id
WHERE a.config_id = $1
ORDER BY pr.submitted_at ASC
`, configID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ReviewRow, 0)
	for rows.Next() {
		var r ReviewRow
		if err := rows.Scan(&r.ID, &r.AllocationID, &r.Score, &r.RubricScoresJSON, &r.Comments, &r.SubmittedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func ListReceivedReviewsForUser(ctx context.Context, pool *pgxpool.Pool, configID uuid.UUID, revieweeUserID uuid.UUID) ([]ReviewRow, error) {
	rows, err := pool.Query(ctx, `
SELECT pr.id, pr.allocation_id, pr.score, pr.rubric_scores_json, pr.comments, pr.submitted_at
FROM course.peer_reviews pr
JOIN course.peer_review_allocations a ON a.id = pr.allocation_id
JOIN course.module_assignment_submissions s ON s.id = a.target_submission_id
WHERE a.config_id = $1 AND s.submitted_by = $2
ORDER BY pr.submitted_at ASC
`, configID, revieweeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ReviewRow, 0)
	for rows.Next() {
		var r ReviewRow
		if err := rows.Scan(&r.ID, &r.AllocationID, &r.Score, &r.RubricScoresJSON, &r.Comments, &r.SubmittedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func UpsertTeamEvaluation(ctx context.Context, pool *pgxpool.Pool, groupID, raterEnrollmentID, rateeEnrollmentID uuid.UUID, score int, comment *string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.team_peer_evaluations (group_id, rater_enrollment_id, ratee_enrollment_id, contribution_score, comment, submitted_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (group_id, rater_enrollment_id, ratee_enrollment_id) DO UPDATE SET
    contribution_score = EXCLUDED.contribution_score,
    comment = EXCLUDED.comment,
    submitted_at = NOW()
`, groupID, raterEnrollmentID, rateeEnrollmentID, score, comment)
	return err
}

type TeamEvalSummaryRow struct {
	RateeEnrollmentID uuid.UUID
	RateeUserID       uuid.UUID
	AvgScore          float64
	EvalCount         int
	Comments          []string
}

func GetEnrollmentIDForUser(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT ce.id
FROM course.course_enrollments ce
JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE ce.course_id = $1 AND ce.user_id = $2 AND ce.active
LIMIT 1
`, courseID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}
