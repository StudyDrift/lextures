package course

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportCardsEnabledForCourseCode returns whether report cards are enabled for a course.
func ReportCardsEnabledForCourseCode(ctx context.Context, pool *pgxpool.Pool, courseCode string) (bool, error) {
	var enabled bool
	err := pool.QueryRow(ctx, `
SELECT report_cards_enabled FROM course.courses WHERE course_code = $1`, courseCode).Scan(&enabled)
	return enabled, err
}

// ReportCardsEnabledForCourseID returns whether report cards are enabled for a course id.
func ReportCardsEnabledForCourseID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (bool, error) {
	var enabled bool
	err := pool.QueryRow(ctx, `
SELECT report_cards_enabled FROM course.courses WHERE id = $1`, courseID).Scan(&enabled)
	return enabled, err
}