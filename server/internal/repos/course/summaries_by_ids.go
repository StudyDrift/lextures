package course

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Summary is a minimal course label for access-key allowlists.
type Summary struct {
	ID         uuid.UUID
	CourseCode string
	Title      string
}

// SummariesByIDs returns course code/title for the given ids (skips unknown ids).
func SummariesByIDs(ctx context.Context, pool *pgxpool.Pool, ids []uuid.UUID) ([]Summary, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT id, course_code, title
FROM course.courses
WHERE id = ANY($1::uuid[])
ORDER BY course_code
`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Summary
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.ID, &s.CourseCode, &s.Title); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
