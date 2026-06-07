// Package notebooktasks persists learner tasks from notebook editors.
package notebooktasks

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Task is one notebook checkbox item for a learner.
type Task struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	CourseCode     string
	NotebookPageID string
	TaskText       string
	Completed      bool
	DueAt          *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Upsert inserts or updates a task row owned by userID.
func Upsert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, t Task) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.student_notebook_tasks (
    id, user_id, course_code, notebook_page_id, task_text, completed, due_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (id) DO UPDATE SET
    task_text = EXCLUDED.task_text,
    completed = EXCLUDED.completed,
    due_at = EXCLUDED.due_at,
    updated_at = now()
WHERE analytics.student_notebook_tasks.user_id = $2
`, t.ID, userID, t.CourseCode, t.NotebookPageID, t.TaskText, t.Completed, t.DueAt)
	return err
}

// Get returns a task when it belongs to userID.
func Get(ctx context.Context, pool *pgxpool.Pool, userID, taskID uuid.UUID) (*Task, error) {
	var t Task
	err := pool.QueryRow(ctx, `
SELECT id, user_id, course_code, notebook_page_id, task_text, completed, due_at, created_at, updated_at
FROM analytics.student_notebook_tasks
WHERE id = $1 AND user_id = $2
`, taskID, userID).Scan(
		&t.ID, &t.UserID, &t.CourseCode, &t.NotebookPageID, &t.TaskText, &t.Completed, &t.DueAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListOpen returns incomplete tasks for a user, ordered by due date then recency.
func ListOpen(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT id, user_id, course_code, notebook_page_id, task_text, completed, due_at, created_at, updated_at
FROM analytics.student_notebook_tasks
WHERE user_id = $1 AND completed = false
ORDER BY due_at NULLS LAST, created_at DESC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.CourseCode, &t.NotebookPageID, &t.TaskText, &t.Completed, &t.DueAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Patch updates mutable fields on a task owned by userID.
func Patch(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID, taskID uuid.UUID,
	taskText *string,
	completed *bool,
	dueAt *time.Time,
	clearDueAt bool,
) (*Task, error) {
	cur, err := Get(ctx, pool, userID, taskID)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, nil
	}
	text := cur.TaskText
	done := cur.Completed
	due := cur.DueAt
	if taskText != nil {
		text = *taskText
	}
	if completed != nil {
		done = *completed
	}
	if clearDueAt {
		due = nil
	} else if dueAt != nil {
		due = dueAt
	}
	_, err = pool.Exec(ctx, `
UPDATE analytics.student_notebook_tasks
SET task_text = $3, completed = $4, due_at = $5, updated_at = now()
WHERE id = $1 AND user_id = $2
`, taskID, userID, text, done, due)
	if err != nil {
		return nil, err
	}
	return Get(ctx, pool, userID, taskID)
}
