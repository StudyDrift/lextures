// Package classroomsignals provides data access for the hall pass and
// anonymous question queue features (plan 13.9).
package classroomsignals

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HallPassStatus values.
const (
	StatusRequested = "requested"
	StatusApproved  = "approved"
	StatusReturned  = "returned"
	StatusDenied    = "denied"
)

// AllowedDestinations lists the canonical destinations for hall passes.
// Frontends and i18n layers should localize these labels; the backend
// stores the raw token verbatim.
var AllowedDestinations = []string{"bathroom", "office", "library", "nurse", "other"}

// IsAllowedDestination reports whether v matches an AllowedDestinations entry.
func IsAllowedDestination(v string) bool {
	for _, d := range AllowedDestinations {
		if d == v {
			return true
		}
	}
	return false
}

// HallPass is a single hall-pass record.
type HallPass struct {
	ID            uuid.UUID
	StudentID     uuid.UUID
	SectionID     uuid.UUID
	Destination   string
	EstimatedMins *int
	Status        string
	RequestedAt   time.Time
	ApprovedAt    *time.Time
	ReturnedAt    *time.Time
	ApprovedBy    *uuid.UUID
}

// AnonymousQuestion is a question submitted by a student in a course.
type AnonymousQuestion struct {
	ID        uuid.UUID
	CourseID  uuid.UUID
	AuthorID  *uuid.UUID // NEVER returned to non-teacher requesters
	Question  string
	Addressed bool
	CreatedAt time.Time
}

// CreateHallPass inserts a new pending pass request.
func CreateHallPass(ctx context.Context, pool *pgxpool.Pool, studentID, sectionID uuid.UUID, destination string, estimatedMins *int) (*HallPass, error) {
	var p HallPass
	err := pool.QueryRow(ctx, `
INSERT INTO classroom.hall_passes (student_id, section_id, destination, estimated_mins)
VALUES ($1, $2, $3, $4)
RETURNING id, student_id, section_id, destination, estimated_mins, status,
          requested_at, approved_at, returned_at, approved_by
`, studentID, sectionID, destination, estimatedMins).Scan(
		&p.ID, &p.StudentID, &p.SectionID, &p.Destination, &p.EstimatedMins,
		&p.Status, &p.RequestedAt, &p.ApprovedAt, &p.ReturnedAt, &p.ApprovedBy,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetHallPass returns a hall pass by ID, or nil if it does not exist.
func GetHallPass(ctx context.Context, pool *pgxpool.Pool, passID uuid.UUID) (*HallPass, error) {
	var p HallPass
	err := pool.QueryRow(ctx, `
SELECT id, student_id, section_id, destination, estimated_mins, status,
       requested_at, approved_at, returned_at, approved_by
FROM classroom.hall_passes
WHERE id = $1
`, passID).Scan(
		&p.ID, &p.StudentID, &p.SectionID, &p.Destination, &p.EstimatedMins,
		&p.Status, &p.RequestedAt, &p.ApprovedAt, &p.ReturnedAt, &p.ApprovedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ErrInvalidTransition is returned when attempting a status change not allowed
// by the hall-pass state machine.
var ErrInvalidTransition = errors.New("classroom_signals: invalid status transition")

// CanTransition reports whether `to` is reachable from `from` per FR-1..FR-3.
func CanTransition(from, to string) bool {
	switch from {
	case StatusRequested:
		return to == StatusApproved || to == StatusDenied
	case StatusApproved:
		return to == StatusReturned
	}
	return false
}

// UpdateHallPassStatus applies an allowed transition and stamps timestamps.
// approver may be nil for student "I'm back" returns.
func UpdateHallPassStatus(ctx context.Context, pool *pgxpool.Pool, passID uuid.UUID, newStatus string, approver *uuid.UUID) (*HallPass, error) {
	switch newStatus {
	case StatusApproved:
		var p HallPass
		err := pool.QueryRow(ctx, `
UPDATE classroom.hall_passes
SET status = 'approved', approved_at = now(), approved_by = $2
WHERE id = $1 AND status = 'requested'
RETURNING id, student_id, section_id, destination, estimated_mins, status,
          requested_at, approved_at, returned_at, approved_by
`, passID, approver).Scan(
			&p.ID, &p.StudentID, &p.SectionID, &p.Destination, &p.EstimatedMins,
			&p.Status, &p.RequestedAt, &p.ApprovedAt, &p.ReturnedAt, &p.ApprovedBy,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidTransition
		}
		return &p, err
	case StatusDenied:
		var p HallPass
		err := pool.QueryRow(ctx, `
UPDATE classroom.hall_passes
SET status = 'denied'
WHERE id = $1 AND status = 'requested'
RETURNING id, student_id, section_id, destination, estimated_mins, status,
          requested_at, approved_at, returned_at, approved_by
`, passID).Scan(
			&p.ID, &p.StudentID, &p.SectionID, &p.Destination, &p.EstimatedMins,
			&p.Status, &p.RequestedAt, &p.ApprovedAt, &p.ReturnedAt, &p.ApprovedBy,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidTransition
		}
		return &p, err
	case StatusReturned:
		var p HallPass
		err := pool.QueryRow(ctx, `
UPDATE classroom.hall_passes
SET status = 'returned', returned_at = now()
WHERE id = $1 AND status = 'approved'
RETURNING id, student_id, section_id, destination, estimated_mins, status,
          requested_at, approved_at, returned_at, approved_by
`, passID).Scan(
			&p.ID, &p.StudentID, &p.SectionID, &p.Destination, &p.EstimatedMins,
			&p.Status, &p.RequestedAt, &p.ApprovedAt, &p.ReturnedAt, &p.ApprovedBy,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidTransition
		}
		return &p, err
	}
	return nil, ErrInvalidTransition
}

// ListActiveSectionPasses returns the currently-out passes (requested or approved)
// for a section, oldest-first so teachers see the longest-out students at top.
func ListActiveSectionPasses(ctx context.Context, pool *pgxpool.Pool, sectionID uuid.UUID) ([]HallPass, error) {
	rows, err := pool.Query(ctx, `
SELECT id, student_id, section_id, destination, estimated_mins, status,
       requested_at, approved_at, returned_at, approved_by
FROM classroom.hall_passes
WHERE section_id = $1 AND status IN ('requested', 'approved')
ORDER BY requested_at ASC
`, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPasses(rows)
}

func scanPasses(rows pgx.Rows) ([]HallPass, error) {
	var out []HallPass
	for rows.Next() {
		var p HallPass
		if err := rows.Scan(
			&p.ID, &p.StudentID, &p.SectionID, &p.Destination, &p.EstimatedMins,
			&p.Status, &p.RequestedAt, &p.ApprovedAt, &p.ReturnedAt, &p.ApprovedBy,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreateAnonymousQuestion inserts a student question into the course queue.
func CreateAnonymousQuestion(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, authorID uuid.UUID, question string) (*AnonymousQuestion, error) {
	var q AnonymousQuestion
	err := pool.QueryRow(ctx, `
INSERT INTO classroom.anonymous_questions (course_id, author_id, question)
VALUES ($1, $2, $3)
RETURNING id, course_id, author_id, question, addressed, created_at
`, courseID, authorID, question).Scan(
		&q.ID, &q.CourseID, &q.AuthorID, &q.Question, &q.Addressed, &q.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// ListCourseQuestions returns the question queue for a course, oldest-first.
// includeAuthor controls whether AuthorID is populated in the returned rows.
func ListCourseQuestions(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, includeAuthor bool, includeAddressed bool) ([]AnonymousQuestion, error) {
	whereAddr := ""
	if !includeAddressed {
		whereAddr = " AND NOT addressed"
	}
	rows, err := pool.Query(ctx, `
SELECT id, course_id, author_id, question, addressed, created_at
FROM classroom.anonymous_questions
WHERE course_id = $1`+whereAddr+`
ORDER BY created_at ASC
LIMIT 500
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AnonymousQuestion
	for rows.Next() {
		var q AnonymousQuestion
		if err := rows.Scan(&q.ID, &q.CourseID, &q.AuthorID, &q.Question, &q.Addressed, &q.CreatedAt); err != nil {
			return nil, err
		}
		if !includeAuthor {
			q.AuthorID = nil
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// MarkQuestionAddressed flips the addressed flag for a question scoped to a course.
func MarkQuestionAddressed(ctx context.Context, pool *pgxpool.Pool, courseID, questionID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE classroom.anonymous_questions
SET addressed = TRUE
WHERE id = $1 AND course_id = $2
`, questionID, courseID)
	return err
}
