package course

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ValidGradeLevels is the allowed set of grade-level values (K, 1-12, and band ranges).
var ValidGradeLevels = []string{
	"K", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12",
	"K-2", "3-5", "6-8", "9-12", "K-12",
}

// ValidGradeLevel returns true when v is one of the accepted grade-level tokens.
func ValidGradeLevel(v string) bool {
	v = strings.TrimSpace(v)
	for _, g := range ValidGradeLevels {
		if g == v {
			return true
		}
	}
	return false
}

// SetGradeLevel updates course.courses.grade_level for the given course_code.
// Pass nil to clear the field.
func SetGradeLevel(ctx context.Context, pool *pgxpool.Pool, courseCode string, gradeLevel *string) (*CoursePublic, error) {
	tag, err := pool.Exec(ctx,
		`UPDATE course.courses SET grade_level = $1, updated_at = NOW() WHERE course_code = $2`,
		gradeLevel, courseCode,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetPublicByCourseCode(ctx, pool, courseCode)
}
