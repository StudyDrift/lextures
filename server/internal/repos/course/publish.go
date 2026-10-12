package course

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SetPublished sets course.courses.published for a course code.
func SetPublished(ctx context.Context, pool *pgxpool.Pool, courseCode string, published bool) error {
	code := strings.TrimSpace(courseCode)
	if code == "" {
		return errors.New("course code is required")
	}
	tag, err := pool.Exec(ctx, `
UPDATE course.courses
SET published = $1, updated_at = NOW()
WHERE course_code = $2
`, published, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("course not found")
	}
	return nil
}

// DeleteByCode removes a course and its cascaded rows. Used when a starter copy fails mid-way.
func DeleteByCode(ctx context.Context, pool *pgxpool.Pool, courseCode string) error {
	code := strings.TrimSpace(courseCode)
	if code == "" {
		return errors.New("course code is required")
	}
	_, err := pool.Exec(ctx, `DELETE FROM course.courses WHERE course_code = $1`, code)
	return err
}
