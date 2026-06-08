package enrollment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	modelenrollment "github.com/lextures/lextures/server/internal/models/enrollment"
)

// StateRow is the current enrollment state fields.
type StateRow struct {
	ID             uuid.UUID
	CourseID       uuid.UUID
	UserID         uuid.UUID
	Role           string
	State          modelenrollment.State
	StateChangedAt *time.Time
	StateReason    *string
	Active         bool
}

// HistoryRow is one enrollment_state_history record.
type HistoryRow struct {
	ID            uuid.UUID
	EnrollmentID  uuid.UUID
	ActorID       *uuid.UUID
	PreviousState modelenrollment.State
	NewState      modelenrollment.State
	Reason        *string
	Source        string
	CreatedAt     time.Time
}

// TermDeadlines holds add/drop and withdrawal dates for a course term.
type TermDeadlines struct {
	AddDropDeadline    *time.Time
	WithdrawalDeadline *time.Time
}

// GetStateByID loads enrollment state for a row in a course.
func GetStateByID(ctx context.Context, pool *pgxpool.Pool, courseID, enrollmentID uuid.UUID) (*StateRow, error) {
	var row StateRow
	var stateStr string
	var reason *string
	err := pool.QueryRow(ctx, `
SELECT ce.id, ce.course_id, ce.user_id, ce.role, ce.state::text, ce.state_changed_at, ce.state_reason, ce.active
FROM course.course_enrollments ce
WHERE ce.id = $1 AND ce.course_id = $2
`, enrollmentID, courseID).Scan(
		&row.ID, &row.CourseID, &row.UserID, &row.Role, &stateStr, &row.StateChangedAt, &reason, &row.Active,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	st, err := modelenrollment.ParseState(stateStr)
	if err != nil {
		return nil, err
	}
	row.State = st
	row.StateReason = reason
	return &row, nil
}

// TermDeadlinesForCourse returns deadline dates from the course term, if any.
func TermDeadlinesForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (TermDeadlines, error) {
	var addDrop, withdraw *time.Time
	err := pool.QueryRow(ctx, `
SELECT t.add_drop_deadline, t.withdrawal_deadline
FROM course.courses c
LEFT JOIN tenant.terms t ON t.id = c.term_id
WHERE c.id = $1
`, courseID).Scan(&addDrop, &withdraw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TermDeadlines{}, nil
		}
		return TermDeadlines{}, err
	}
	return TermDeadlines{AddDropDeadline: addDrop, WithdrawalDeadline: withdraw}, nil
}

// TransitionState applies a state change atomically with history.
func TransitionState(
	ctx context.Context,
	pool *pgxpool.Pool,
	enrollmentID uuid.UUID,
	courseID uuid.UUID,
	actorID *uuid.UUID,
	newState modelenrollment.State,
	reason *string,
	source string,
	dc modelenrollment.DeadlineContext,
) (*StateRow, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var prevStr string
	var row StateRow
	var reasonCur *string
	err = tx.QueryRow(ctx, `
SELECT ce.id, ce.course_id, ce.user_id, ce.role, ce.state::text, ce.state_changed_at, ce.state_reason, ce.active
FROM course.course_enrollments ce
WHERE ce.id = $1 AND ce.course_id = $2
FOR UPDATE
`, enrollmentID, courseID).Scan(
		&row.ID, &row.CourseID, &row.UserID, &row.Role, &prevStr, &row.StateChangedAt, &reasonCur, &row.Active,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	prev, err := modelenrollment.ParseState(prevStr)
	if err != nil {
		return nil, err
	}
	row.State = prev
	row.StateReason = reasonCur

	if err := modelenrollment.ValidateTransition(prev, newState, dc); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if dc.Now.IsZero() {
		dc.Now = now
	}
	active := newState.SetsActiveEnrollment()

	_, err = tx.Exec(ctx, `
UPDATE course.course_enrollments
SET state = $1::course.enrollment_state,
    state_changed_at = $2,
    state_reason = $3,
    active = $4
WHERE id = $5
`, string(newState), now, reason, active, enrollmentID)
	if err != nil {
		return nil, err
	}

	src := source
	if src == "" {
		src = "manual"
	}
	_, err = tx.Exec(ctx, `
INSERT INTO course.enrollment_state_history
    (enrollment_id, actor_id, previous_state, new_state, reason, source)
VALUES ($1, $2, $3::course.enrollment_state, $4::course.enrollment_state, $5, $6)
`, enrollmentID, actorID, string(prev), string(newState), reason, src)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	row.State = newState
	row.StateChangedAt = &now
	row.StateReason = reason
	row.Active = active
	return &row, nil
}

// ListStateHistory returns state transitions for an enrollment, newest first.
func ListStateHistory(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, limit int) ([]HistoryRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT id, enrollment_id, actor_id, previous_state::text, new_state::text, reason, source, created_at
FROM course.enrollment_state_history
WHERE enrollment_id = $1
ORDER BY created_at DESC
LIMIT $2
`, enrollmentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HistoryRow
	for rows.Next() {
		var h HistoryRow
		var prevStr, newStr string
		if err := rows.Scan(&h.ID, &h.EnrollmentID, &h.ActorID, &prevStr, &newStr, &h.Reason, &h.Source, &h.CreatedAt); err != nil {
			return nil, err
		}
		h.PreviousState, err = modelenrollment.ParseState(prevStr)
		if err != nil {
			return nil, err
		}
		h.NewState, err = modelenrollment.ParseState(newStr)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// GradebookStudentRow is a student row for the gradebook grid with lifecycle state.
type GradebookStudentRow struct {
	UserID       uuid.UUID
	EnrollmentID uuid.UUID
	DisplayName  string
	State        modelenrollment.State
}

// ListGradebookStudents returns active and optionally former students for the gradebook.
func ListGradebookStudents(ctx context.Context, pool *pgxpool.Pool, courseCode string, sectionIDs []uuid.UUID, includeFormer bool) ([]GradebookStudentRow, error) {
	stateFilter := `ce.state = 'active' AND ce.active`
	if includeFormer {
		stateFilter = `(ce.state = 'active' AND ce.active) OR ce.state IN ('dropped', 'withdrawn', 'no_credit')`
	}
	var rows pgx.Rows
	var err error
	if len(sectionIDs) == 0 {
		rows, err = pool.Query(ctx, fmt.Sprintf(`
SELECT ce.user_id, ce.id,
       COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) AS display_label,
       ce.state::text
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE c.course_code = $1 AND ce.role = 'student' AND (%s)
ORDER BY display_label ASC, ce.user_id ASC
`, stateFilter), courseCode)
	} else {
		rows, err = pool.Query(ctx, fmt.Sprintf(`
SELECT ce.user_id, ce.id,
       COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) AS display_label,
       ce.state::text
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE c.course_code = $1 AND ce.role = 'student' AND (%s)
  AND ce.section_id = ANY($2::uuid[])
ORDER BY display_label ASC, ce.user_id ASC
`, stateFilter), courseCode, sectionIDs)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GradebookStudentRow
	for rows.Next() {
		var r GradebookStudentRow
		var stateStr string
		if err := rows.Scan(&r.UserID, &r.EnrollmentID, &r.DisplayName, &stateStr); err != nil {
			return nil, err
		}
		r.State, err = modelenrollment.ParseState(stateStr)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ViewerStudentState returns the viewer's student enrollment state for a course.
func ViewerStudentState(ctx context.Context, pool *pgxpool.Pool, courseCode string, userID uuid.UUID) (modelenrollment.State, *time.Time, error) {
	var stateStr string
	var changedAt *time.Time
	err := pool.QueryRow(ctx, `
SELECT ce.state::text, ce.state_changed_at
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
WHERE c.course_code = $1 AND ce.user_id = $2 AND ce.role = 'student'
ORDER BY ce.created_at ASC
LIMIT 1
`, courseCode, userID).Scan(&stateStr, &changedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return modelenrollment.StateActive, nil, nil
		}
		return "", nil, err
	}
	st, err := modelenrollment.ParseState(stateStr)
	return st, changedAt, err
}
