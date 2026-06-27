// Package learnerprogress persists per-enrollment learner item progress for
// self-paced courses (plan 15.2).
package learnerprogress

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ItemProgress is one learner_item_progress row.
type ItemProgress struct {
	ItemID        uuid.UUID
	Status        string
	LastVisitedAt *time.Time
	CompletedAt   *time.Time
}

// ModuleProgress aggregates leaf-item completion for one module, ordered by sort_order.
type ModuleProgress struct {
	ModuleID       uuid.UUID
	Title          string
	SortOrder      int
	TotalItems     int
	CompletedItems int
}

// CourseTotals is course-wide completion for one enrollment.
type CourseTotals struct {
	TotalItems     int
	CompletedItems int
}

// MarkVisited records that the learner opened an item; it moves not_started → in_progress
// but never downgrades a completed item.
func MarkVisited(ctx context.Context, pool *pgxpool.Pool, enrollmentID, itemID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.learner_item_progress (enrollment_id, item_id, status, last_visited_at, updated_at)
VALUES ($1, $2, 'in_progress', NOW(), NOW())
ON CONFLICT (enrollment_id, item_id) DO UPDATE SET
    status = CASE WHEN course.learner_item_progress.status = 'completed' THEN 'completed' ELSE 'in_progress' END,
    last_visited_at = NOW(),
    updated_at = NOW()
`, enrollmentID, itemID)
	return err
}

// MarkCompleted records that the learner completed an item. Returns true when the row
// transitioned to completed (i.e. it was not already completed).
func MarkCompleted(ctx context.Context, pool *pgxpool.Pool, enrollmentID, itemID uuid.UUID) (bool, error) {
	var alreadyCompleted bool
	err := pool.QueryRow(ctx, `
WITH existing AS (
    SELECT status FROM course.learner_item_progress
    WHERE enrollment_id = $1 AND item_id = $2
)
INSERT INTO course.learner_item_progress (enrollment_id, item_id, status, last_visited_at, completed_at, updated_at)
VALUES ($1, $2, 'completed', NOW(), NOW(), NOW())
ON CONFLICT (enrollment_id, item_id) DO UPDATE SET
    status = 'completed',
    completed_at = COALESCE(course.learner_item_progress.completed_at, NOW()),
    last_visited_at = NOW(),
    updated_at = NOW()
RETURNING COALESCE((SELECT status = 'completed' FROM existing), false)
`, enrollmentID, itemID).Scan(&alreadyCompleted)
	if err != nil {
		return false, err
	}
	return !alreadyCompleted, nil
}

// CourseProgress returns total published leaf items and completed items for an enrollment.
func CourseProgress(ctx context.Context, pool *pgxpool.Pool, courseID, enrollmentID uuid.UUID) (CourseTotals, error) {
	var t CourseTotals
	err := pool.QueryRow(ctx, `
SELECT
    COUNT(i.id) AS total,
    COUNT(lip.id) FILTER (WHERE lip.status = 'completed') AS completed
FROM course.course_structure_items i
LEFT JOIN course.learner_item_progress lip
    ON lip.item_id = i.id AND lip.enrollment_id = $2
WHERE i.course_id = $1
  AND i.kind NOT IN ('module', 'heading')
  AND i.published
  AND NOT i.archived
`, courseID, enrollmentID).Scan(&t.TotalItems, &t.CompletedItems)
	if err != nil {
		return CourseTotals{}, err
	}
	return t, nil
}

// ModuleProgressForEnrollment returns per-module completion ordered by sort_order, resolving
// each published leaf item to its owning module via the parent chain.
func ModuleProgressForEnrollment(ctx context.Context, pool *pgxpool.Pool, courseID, enrollmentID uuid.UUID) ([]ModuleProgress, error) {
	rows, err := pool.Query(ctx, `
WITH RECURSIVE leaf AS (
    SELECT id AS leaf_id, id AS cur, parent_id, kind
    FROM course.course_structure_items
    WHERE course_id = $1 AND kind NOT IN ('module', 'heading') AND published AND NOT archived
  UNION ALL
    SELECT l.leaf_id, p.id, p.parent_id, p.kind
    FROM leaf l
    JOIN course.course_structure_items p ON p.id = l.parent_id
    WHERE l.kind <> 'module'
),
leaf_module AS (
    SELECT leaf_id, cur AS module_id FROM leaf WHERE kind = 'module'
)
SELECT m.id, m.title, m.sort_order,
    COUNT(lm.leaf_id) AS total_items,
    COUNT(lip.id) FILTER (WHERE lip.status = 'completed') AS completed_items
FROM course.course_structure_items m
LEFT JOIN leaf_module lm ON lm.module_id = m.id
LEFT JOIN course.learner_item_progress lip
    ON lip.item_id = lm.leaf_id AND lip.enrollment_id = $2
WHERE m.course_id = $1 AND m.kind = 'module' AND NOT m.archived
GROUP BY m.id, m.title, m.sort_order
ORDER BY m.sort_order
`, courseID, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModuleProgress
	for rows.Next() {
		var m ModuleProgress
		if err := rows.Scan(&m.ModuleID, &m.Title, &m.SortOrder, &m.TotalItems, &m.CompletedItems); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// LastVisitedItem returns the item id the learner most recently visited, or nil if none.
func LastVisitedItem(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT item_id FROM course.learner_item_progress
WHERE enrollment_id = $1 AND last_visited_at IS NOT NULL
ORDER BY last_visited_at DESC
LIMIT 1
`, enrollmentID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}

// ItemBelongsToCourse reports whether the structure item is a published leaf item in the course.
func ItemBelongsToCourse(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.course_structure_items
    WHERE id = $2 AND course_id = $1 AND kind NOT IN ('module', 'heading') AND NOT archived
)
`, courseID, itemID).Scan(&ok)
	return ok, err
}

// ModuleForItem resolves the owning module of a leaf item via its parent chain.
func ModuleForItem(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
WITH RECURSIVE walk AS (
    SELECT id, parent_id, kind
    FROM course.course_structure_items
    WHERE course_id = $1 AND id = $2
  UNION ALL
    SELECT p.id, p.parent_id, p.kind
    FROM walk w
    JOIN course.course_structure_items p ON p.id = w.parent_id
    WHERE w.kind <> 'module'
)
SELECT id FROM walk WHERE kind = 'module' LIMIT 1
`, courseID, itemID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}

// FirstItem returns the first published leaf item in the course (by sort order), or nil.
func FirstItem(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id FROM course.course_structure_items
WHERE course_id = $1 AND kind NOT IN ('module', 'heading') AND published AND NOT archived
ORDER BY sort_order
LIMIT 1
`, courseID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}

// FirstIncompleteItem returns the first published leaf item (by sort order) the learner has
// not yet completed, falling back to the first item when all are complete or none visited.
func FirstIncompleteItem(ctx context.Context, pool *pgxpool.Pool, courseID, enrollmentID uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT i.id
FROM course.course_structure_items i
LEFT JOIN course.learner_item_progress lip
    ON lip.item_id = i.id AND lip.enrollment_id = $2
WHERE i.course_id = $1 AND i.kind NOT IN ('module', 'heading') AND i.published AND NOT i.archived
  AND (lip.status IS NULL OR lip.status <> 'completed')
ORDER BY i.sort_order
LIMIT 1
`, courseID, enrollmentID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FirstItem(ctx, pool, courseID)
		}
		return nil, err
	}
	return &id, nil
}
