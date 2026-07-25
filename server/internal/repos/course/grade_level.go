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

// NormalizeGradeLevels trims, dedupes (order-preserving), and validates grade tokens.
// Empty input yields nil (unset). Invalid tokens return ok=false.
func NormalizeGradeLevels(levels []string) (out []string, ok bool) {
	if len(levels) == 0 {
		return nil, true
	}
	seen := make(map[string]struct{}, len(levels))
	out = make([]string, 0, len(levels))
	for _, raw := range levels {
		gl := strings.TrimSpace(raw)
		if gl == "" {
			continue
		}
		if !ValidGradeLevel(gl) {
			return nil, false
		}
		if _, exists := seen[gl]; exists {
			continue
		}
		seen[gl] = struct{}{}
		out = append(out, gl)
	}
	if len(out) == 0 {
		return nil, true
	}
	return out, true
}

// SetGradeLevels updates course.courses.grade_levels for the given course_code.
// Pass nil or empty slice to clear the field.
func SetGradeLevels(ctx context.Context, pool *pgxpool.Pool, courseCode string, gradeLevels []string) (*CoursePublic, error) {
	var arg any
	if len(gradeLevels) == 0 {
		arg = nil
	} else {
		arg = gradeLevels
	}
	tag, err := pool.Exec(ctx,
		`UPDATE course.courses SET grade_levels = $1, updated_at = NOW() WHERE course_code = $2`,
		arg, courseCode,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetPublicByCourseCode(ctx, pool, courseCode)
}

// SetGradeLevel updates a single grade level (legacy helper).
// Pass nil to clear the field.
func SetGradeLevel(ctx context.Context, pool *pgxpool.Pool, courseCode string, gradeLevel *string) (*CoursePublic, error) {
	if gradeLevel == nil || strings.TrimSpace(*gradeLevel) == "" {
		return SetGradeLevels(ctx, pool, courseCode, nil)
	}
	gl := strings.TrimSpace(*gradeLevel)
	return SetGradeLevels(ctx, pool, courseCode, []string{gl})
}
