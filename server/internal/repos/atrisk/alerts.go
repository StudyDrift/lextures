package atrisk

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AlertStatus is analytics.alert_status.
type AlertStatus string

const (
	AlertActive    AlertStatus = "active"
	AlertDismissed AlertStatus = "dismissed"
	AlertSnoozed   AlertStatus = "snoozed"
	AlertSupported AlertStatus = "supported"
	AlertResolved  AlertStatus = "resolved"
)

// AlertRow is one at-risk alert with student display fields for API responses.
type AlertRow struct {
	ID            uuid.UUID
	EnrollmentID  uuid.UUID
	UserID        uuid.UUID
	DisplayName   *string
	TriggeredDate time.Time
	Score         float32
	Status        AlertStatus
	TopFactor     string
	SnoozeUntil   *time.Time
	Notes         *string
	UpdatedAt     time.Time
	ResolvedAt    *time.Time
	MissingPct    *float32
	QuizAvg       *float32
	DaysInactive  *int
}

// CreateAlert inserts a new alert if none exists for enrollment+date (idempotent).
func CreateAlert(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, triggeredDate time.Time, score float32, topFactor string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO analytics.at_risk_alerts (enrollment_id, triggered_date, score, top_factor)
VALUES ($1, $2::date, $3, $4)
ON CONFLICT (enrollment_id, triggered_date) DO NOTHING
RETURNING id
`, enrollmentID, triggeredDate.Format("2006-01-02"), score, topFactor).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

// HasBlockingAlert is true when an unresolved alert blocks creating a new one.
func HasBlockingAlert(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, today time.Time) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM analytics.at_risk_alerts a
    WHERE a.enrollment_id = $1
      AND (
        a.status IN ('active', 'supported')
        OR (a.status = 'snoozed' AND (a.snooze_until IS NULL OR a.snooze_until > $2::date))
        OR (a.status = 'dismissed' AND a.resolved_at IS NULL)
      )
)
`, enrollmentID, today.Format("2006-01-02")).Scan(&ok)
	return ok, err
}

// LatestDismissedWithoutResolve returns the most recent dismissed alert still in the episode.
func LatestDismissedWithoutResolve(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (*AlertRow, error) {
	var r AlertRow
	err := pool.QueryRow(ctx, `
SELECT id, enrollment_id, triggered_date, score, status, top_factor, snooze_until, notes, updated_at, resolved_at
FROM analytics.at_risk_alerts
WHERE enrollment_id = $1 AND status = 'dismissed' AND resolved_at IS NULL
ORDER BY triggered_date DESC
LIMIT 1
`, enrollmentID).Scan(
		&r.ID, &r.EnrollmentID, &r.TriggeredDate, &r.Score, &r.Status, &r.TopFactor, &r.SnoozeUntil, &r.Notes, &r.UpdatedAt, &r.ResolvedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ResolveActiveAlerts marks active/snoozed/supported alerts resolved for an enrollment.
func ResolveActiveAlerts(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, at time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `
UPDATE analytics.at_risk_alerts
SET status = 'resolved', resolved_at = $2, updated_at = $2
WHERE enrollment_id = $1 AND status IN ('active', 'snoozed', 'supported')
`, enrollmentID, at)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ResolveDismissedEpisode clears dismissed-without-resolve when score drops below threshold.
func ResolveDismissedEpisode(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, at time.Time) error {
	_, err := pool.Exec(ctx, `
UPDATE analytics.at_risk_alerts
SET resolved_at = $2, updated_at = $2
WHERE enrollment_id = $1 AND status = 'dismissed' AND resolved_at IS NULL
`, enrollmentID, at)
	return err
}

// ListActiveForCourse returns alerts for a course sorted by score descending.
func ListActiveForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, includeResolved bool) ([]AlertRow, error) {
	statusFilter := `a.status IN ('active', 'snoozed', 'supported')`
	if includeResolved {
		statusFilter = `a.status IN ('active', 'snoozed', 'supported', 'dismissed', 'resolved')`
	}
	rows, err := pool.Query(ctx, `
SELECT
    a.id,
    a.enrollment_id,
    ce.user_id,
    u.display_name,
    a.triggered_date,
    a.score,
    a.status,
    a.top_factor,
    a.snooze_until,
    a.notes,
    a.updated_at,
    a.resolved_at,
    s.missing_pct,
    s.quiz_avg,
    s.days_inactive
FROM analytics.at_risk_alerts a
INNER JOIN course.course_enrollments ce ON ce.id = a.enrollment_id
INNER JOIN "user".users u ON u.id = ce.user_id
INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
LEFT JOIN LATERAL (
    SELECT missing_pct, quiz_avg, days_inactive
    FROM analytics.at_risk_scores sc
    WHERE sc.enrollment_id = a.enrollment_id
    ORDER BY sc.computed_date DESC
    LIMIT 1
) s ON true
WHERE ce.course_id = $1 AND `+statusFilter+`
ORDER BY a.score DESC, COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlertRows(rows)
}

func scanAlertRows(rows pgx.Rows) ([]AlertRow, error) {
	var out []AlertRow
	for rows.Next() {
		var r AlertRow
		var display sql.NullString
		var snooze sql.NullTime
		var notes sql.NullString
		var resolved sql.NullTime
		var missing, quiz sql.NullFloat64
		var inactive sql.NullInt32
		if err := rows.Scan(
			&r.ID, &r.EnrollmentID, &r.UserID, &display, &r.TriggeredDate, &r.Score, &r.Status, &r.TopFactor,
			&snooze, &notes, &r.UpdatedAt, &resolved, &missing, &quiz, &inactive,
		); err != nil {
			return nil, err
		}
		if display.Valid && display.String != "" {
			s := display.String
			r.DisplayName = &s
		}
		if snooze.Valid {
			t := snooze.Time
			r.SnoozeUntil = &t
		}
		if notes.Valid && notes.String != "" {
			s := notes.String
			r.Notes = &s
		}
		if resolved.Valid {
			t := resolved.Time
			r.ResolvedAt = &t
		}
		if missing.Valid {
			f := float32(missing.Float64)
			r.MissingPct = &f
		}
		if quiz.Valid {
			f := float32(quiz.Float64)
			r.QuizAvg = &f
		}
		if inactive.Valid {
			n := int(inactive.Int32)
			r.DaysInactive = &n
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetAlertByID loads one alert scoped to course.
func GetAlertByID(ctx context.Context, pool *pgxpool.Pool, courseID, alertID uuid.UUID) (*AlertRow, error) {
	var r AlertRow
	var display sql.NullString
	var snooze sql.NullTime
	var notes sql.NullString
	var resolved sql.NullTime
	err := pool.QueryRow(ctx, `
SELECT a.id, a.enrollment_id, ce.user_id, u.display_name, a.triggered_date, a.score, a.status, a.top_factor,
       a.snooze_until, a.notes, a.updated_at, a.resolved_at
FROM analytics.at_risk_alerts a
INNER JOIN course.course_enrollments ce ON ce.id = a.enrollment_id
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE a.id = $1 AND ce.course_id = $2
`, alertID, courseID).Scan(
		&r.ID, &r.EnrollmentID, &r.UserID, &display, &r.TriggeredDate, &r.Score, &r.Status, &r.TopFactor,
		&snooze, &notes, &r.UpdatedAt, &resolved,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if display.Valid && display.String != "" {
		s := display.String
		r.DisplayName = &s
	}
	if snooze.Valid {
		t := snooze.Time
		r.SnoozeUntil = &t
	}
	if notes.Valid && notes.String != "" {
		s := notes.String
		r.Notes = &s
	}
	if resolved.Valid {
		t := resolved.Time
		r.ResolvedAt = &t
	}
	return &r, nil
}

// PatchAlert updates status, snooze, or notes.
func PatchAlert(ctx context.Context, pool *pgxpool.Pool, alertID uuid.UUID, status *AlertStatus, snoozeUntil *time.Time, notes *string, now time.Time) error {
	tag, err := pool.Exec(ctx, `
UPDATE analytics.at_risk_alerts
SET
    status = COALESCE($2::analytics.alert_status, status),
    snooze_until = CASE WHEN $2::text = 'snoozed' OR $2::text = 'supported' THEN $3::date ELSE snooze_until END,
    notes = COALESCE($4, notes),
    updated_at = $5,
    resolved_at = CASE
        WHEN $2::text = 'resolved' THEN $5
        WHEN $2::text = 'dismissed' THEN NULL
        ELSE resolved_at
    END
WHERE id = $1
`, alertID, statusArg(status), dateArg(snoozeUntil), notes, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func statusArg(s *AlertStatus) any {
	if s == nil {
		return nil
	}
	return string(*s)
}

func dateArg(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

// ListInstructorUserIDs returns distinct instructor/teacher user IDs for a course.
func ListInstructorUserIDs(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT ce.user_id
FROM course.course_enrollments ce
INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_staff = true AND er.can_grade = true
WHERE ce.course_id = $1 AND ce.active
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
