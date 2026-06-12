// Package studentnotebooks persists learner notebook documents synced from clients.
package studentnotebooks

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Notebook is one learner notebook document for a course (or the global key).
type Notebook struct {
	UserID     uuid.UUID
	CourseCode string
	Data       []byte // CourseNotebookStore JSON (format v2)
	UpdatedAt  time.Time
}

// List returns all notebooks owned by userID.
func List(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Notebook, error) {
	rows, err := pool.Query(ctx, `
SELECT user_id, course_code, data, updated_at
FROM analytics.student_notebooks
WHERE user_id = $1
ORDER BY updated_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Notebook
	for rows.Next() {
		var n Notebook
		if err := rows.Scan(&n.UserID, &n.CourseCode, &n.Data, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// Get returns one notebook, or nil when the learner has none for courseCode.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseCode string) (*Notebook, error) {
	var n Notebook
	err := pool.QueryRow(ctx, `
SELECT user_id, course_code, data, updated_at
FROM analytics.student_notebooks
WHERE user_id = $1 AND course_code = $2
`, userID, courseCode).Scan(&n.UserID, &n.CourseCode, &n.Data, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// Upsert writes a notebook unless the stored copy is newer (last-write-wins by updatedAt).
// It returns the row that is current after the call, so callers can hand back the winner.
func Upsert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseCode string, data []byte, updatedAt time.Time) (*Notebook, error) {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.student_notebooks (user_id, course_code, data, updated_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, course_code) DO UPDATE SET
    data = EXCLUDED.data,
    updated_at = EXCLUDED.updated_at
WHERE analytics.student_notebooks.updated_at <= EXCLUDED.updated_at
`, userID, courseCode, data, updatedAt)
	if err != nil {
		return nil, err
	}
	return Get(ctx, pool, userID, courseCode)
}
