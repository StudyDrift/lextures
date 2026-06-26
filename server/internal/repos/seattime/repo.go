// Package seattime persists seat-time sessions, CEU configuration, and awards (plan 14.17).
package seattime

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Session is one seat-time session row.
type Session struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ContentItemID uuid.UUID
	CourseID      uuid.UUID
	SessionToken  string
	SessionStart  time.Time
	SessionEnd    *time.Time
	MinutesActive int
	AnomalyFlag   bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CEUConfig is per-course CEU settings.
type CEUConfig struct {
	ID                  uuid.UUID
	CourseID            uuid.UUID
	RequiredHours       float64
	CEUCredit           float64
	CertificateTemplate *string
	Enabled             bool
}

// CEUAward is an issued CEU certificate record.
type CEUAward struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	CourseID     uuid.UUID
	CEUCredit    float64
	ContactHours float64
	IssuedAt     time.Time
}

// LearnerCourseProgress aggregates seat time for one learner in one course.
type LearnerCourseProgress struct {
	UserID       uuid.UUID
	DisplayName  string
	TotalMinutes int
	CEUEarned    bool
}

// ContentItemCourse resolves a structure item to its course.
type ContentItemCourse struct {
	ContentItemID uuid.UUID
	CourseID      uuid.UUID
	CourseCode    string
	ItemTitle     string
}

// UpsertSession writes or updates a session row.
func UpsertSession(ctx context.Context, pool *pgxpool.Pool, s Session) error {
	_, err := pool.Exec(ctx, `
INSERT INTO seattime.sessions
    (id, user_id, content_item_id, course_id, session_token, session_start, session_end,
     minutes_active, anomaly_flag, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
ON CONFLICT (user_id, content_item_id, session_token) DO UPDATE SET
    session_end = EXCLUDED.session_end,
    minutes_active = EXCLUDED.minutes_active,
    anomaly_flag = seattime.sessions.anomaly_flag OR EXCLUDED.anomaly_flag,
    updated_at = NOW()
`, s.ID, s.UserID, s.ContentItemID, s.CourseID, s.SessionToken, s.SessionStart, s.SessionEnd,
		s.MinutesActive, s.AnomalyFlag)
	return err
}

// GetSession loads an existing session by composite key.
func GetSession(ctx context.Context, pool *pgxpool.Pool, userID, contentItemID uuid.UUID, sessionToken string) (*Session, error) {
	var s Session
	var sessionEnd *time.Time
	err := pool.QueryRow(ctx, `
SELECT id, user_id, content_item_id, course_id, session_token, session_start, session_end,
       minutes_active, anomaly_flag, created_at, updated_at
FROM seattime.sessions
WHERE user_id = $1 AND content_item_id = $2 AND session_token = $3
`, userID, contentItemID, sessionToken).Scan(
		&s.ID, &s.UserID, &s.ContentItemID, &s.CourseID, &s.SessionToken, &s.SessionStart, &sessionEnd,
		&s.MinutesActive, &s.AnomalyFlag, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.SessionEnd = sessionEnd
	return &s, nil
}

// TotalMinutesForCourse sums verified seat time for a user in a course.
func TotalMinutesForCourse(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (int, error) {
	var total int
	err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(minutes_active), 0)::int
FROM seattime.sessions
WHERE user_id = $1 AND course_id = $2
`, userID, courseID).Scan(&total)
	return total, err
}

// DailyMinutesForCourse sums seat time for a user in a course on the current UTC day.
func DailyMinutesForCourse(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, day time.Time) (int, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	var total int
	err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(minutes_active), 0)::int
FROM seattime.sessions
WHERE user_id = $1 AND course_id = $2
  AND session_start >= $3 AND session_start < $4
`, userID, courseID, start, end).Scan(&total)
	return total, err
}

// ResolveContentItemCourse returns course metadata for a structure item.
func ResolveContentItemCourse(ctx context.Context, pool *pgxpool.Pool, contentItemID uuid.UUID) (*ContentItemCourse, error) {
	var out ContentItemCourse
	err := pool.QueryRow(ctx, `
SELECT i.id, c.id, c.course_code, COALESCE(i.title, '')
FROM course.course_structure_items i
JOIN course.courses c ON c.id = i.course_id
WHERE i.id = $1
`, contentItemID).Scan(&out.ContentItemID, &out.CourseID, &out.CourseCode, &out.ItemTitle)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UserEnrolledInCourse returns true when the user has an active enrollment.
func UserEnrolledInCourse(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
    SELECT 1 FROM course.course_enrollments
    WHERE user_id = $1 AND course_id = $2 AND active = true
)
`, userID, courseID).Scan(&exists)
	return exists, err
}

// GetCEUConfig loads CEU settings for a course.
func GetCEUConfig(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*CEUConfig, error) {
	var c CEUConfig
	var template *string
	err := pool.QueryRow(ctx, `
SELECT id, course_id, required_hours::float8, ceu_credit::float8, certificate_template, enabled
FROM seattime.ceu_configurations
WHERE course_id = $1
`, courseID).Scan(&c.ID, &c.CourseID, &c.RequiredHours, &c.CEUCredit, &template, &c.Enabled)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.CertificateTemplate = template
	return &c, nil
}

// UpsertCEUConfig creates or updates CEU configuration for a course.
func UpsertCEUConfig(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, requiredHours, ceuCredit float64, template *string, enabled bool) (*CEUConfig, error) {
	var c CEUConfig
	var tpl *string
	err := pool.QueryRow(ctx, `
INSERT INTO seattime.ceu_configurations (course_id, required_hours, ceu_credit, certificate_template, enabled, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (course_id) DO UPDATE SET
    required_hours = EXCLUDED.required_hours,
    ceu_credit = EXCLUDED.ceu_credit,
    certificate_template = EXCLUDED.certificate_template,
    enabled = EXCLUDED.enabled,
    updated_at = NOW()
RETURNING id, course_id, required_hours::float8, ceu_credit::float8, certificate_template, enabled
`, courseID, requiredHours, ceuCredit, template, enabled).Scan(
		&c.ID, &c.CourseID, &c.RequiredHours, &c.CEUCredit, &tpl, &c.Enabled,
	)
	if err != nil {
		return nil, err
	}
	c.CertificateTemplate = tpl
	return &c, nil
}

// GetCEUAward returns an existing award if present.
func GetCEUAward(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (*CEUAward, error) {
	var a CEUAward
	err := pool.QueryRow(ctx, `
SELECT id, user_id, course_id, ceu_credit::float8, contact_hours::float8, issued_at
FROM seattime.ceu_awards
WHERE user_id = $1 AND course_id = $2
`, userID, courseID).Scan(&a.ID, &a.UserID, &a.CourseID, &a.CEUCredit, &a.ContactHours, &a.IssuedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// CreateCEUAward inserts a new CEU award.
func CreateCEUAward(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, ceuCredit, contactHours float64, issuedAt time.Time) (*CEUAward, error) {
	var a CEUAward
	err := pool.QueryRow(ctx, `
INSERT INTO seattime.ceu_awards (user_id, course_id, ceu_credit, contact_hours, issued_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, course_id, ceu_credit::float8, contact_hours::float8, issued_at
`, userID, courseID, ceuCredit, contactHours, issuedAt).Scan(
		&a.ID, &a.UserID, &a.CourseID, &a.CEUCredit, &a.ContactHours, &a.IssuedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListCEUAwardsForUser returns all CEU awards for transcript generation.
func ListCEUAwardsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]CEUAward, error) {
	rows, err := pool.Query(ctx, `
SELECT a.id, a.user_id, a.course_id, a.ceu_credit::float8, a.contact_hours::float8, a.issued_at
FROM seattime.ceu_awards a
WHERE a.user_id = $1
ORDER BY a.issued_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CEUAward
	for rows.Next() {
		var a CEUAward
		if err := rows.Scan(&a.ID, &a.UserID, &a.CourseID, &a.CEUCredit, &a.ContactHours, &a.IssuedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CourseSeatTimeReport lists per-learner totals for a CE course.
func CourseSeatTimeReport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]LearnerCourseProgress, error) {
	rows, err := pool.Query(ctx, `
SELECT e.user_id,
       COALESCE(u.display_name, u.email) AS display_name,
       COALESCE(SUM(s.minutes_active), 0)::int AS total_minutes,
       EXISTS (
           SELECT 1 FROM seattime.ceu_awards aw
           WHERE aw.user_id = e.user_id AND aw.course_id = e.course_id
       ) AS ceu_earned
FROM course.course_enrollments e
JOIN "user".users u ON u.id = e.user_id
LEFT JOIN seattime.sessions s ON s.user_id = e.user_id AND s.course_id = e.course_id
WHERE e.course_id = $1 AND e.active = true
GROUP BY e.user_id, display_name
ORDER BY display_name ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LearnerCourseProgress
	for rows.Next() {
		var row LearnerCourseProgress
		if err := rows.Scan(&row.UserID, &row.DisplayName, &row.TotalMinutes, &row.CEUEarned); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CourseTitle returns the course title for display.
func CourseTitle(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (string, error) {
	var title string
	err := pool.QueryRow(ctx, `SELECT COALESCE(title, course_code) FROM course.courses WHERE id = $1`, courseID).Scan(&title)
	return title, err
}
