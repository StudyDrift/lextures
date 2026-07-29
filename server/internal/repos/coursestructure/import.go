package coursestructure

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DeleteAllItemsForCourse removes every structure row for a course (children first).
func DeleteAllItemsForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM course.course_structure_items
WHERE course_id = $1 AND parent_id IS NOT NULL
`, courseID); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
DELETE FROM course.course_structure_items
WHERE course_id = $1
`, courseID)
	return err
}

// DeleteStructureNotInExport deletes structure items whose ids are not in keep.
// Children are deleted before parents. Empty keep wipes the whole outline.
func DeleteStructureNotInExport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, keep []uuid.UUID) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	if len(keep) == 0 {
		return DeleteAllItemsForCourse(ctx, pool, courseID)
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM course.course_structure_items
WHERE course_id = $1
  AND parent_id IS NOT NULL
  AND NOT (id = ANY($2::uuid[]))
`, courseID, keep); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
DELETE FROM course.course_structure_items
WHERE course_id = $1
  AND parent_id IS NULL
  AND NOT (id = ANY($2::uuid[]))
`, courseID, keep)
	return err
}

// ImportUpsertStructureItem inserts or updates a structure row for JSON/Canvas-style import.
// When onlyInsert is true, existing ids are left unchanged and the return value is true only on insert.
func ImportUpsertStructureItem(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	item ItemResponse,
	onlyInsert bool,
) (insertedOrUpdated bool, err error) {
	if pool == nil {
		return false, errors.New("db pool is nil")
	}
	itemID, err := uuid.Parse(item.ID)
	if err != nil {
		return false, err
	}
	var parentID *uuid.UUID
	if item.ParentID != nil && *item.ParentID != "" {
		p, perr := uuid.Parse(*item.ParentID)
		if perr != nil {
			return false, perr
		}
		parentID = &p
	}
	var groupID *uuid.UUID
	if item.AssignmentGroupID != nil && *item.AssignmentGroupID != "" {
		g, gerr := uuid.Parse(*item.AssignmentGroupID)
		if gerr != nil {
			return false, gerr
		}
		groupID = &g
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := item.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	if onlyInsert {
		var id uuid.UUID
		err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (
	id, course_id, sort_order, kind, title, parent_id,
	published, visible_from, archived, due_at, assignment_group_id,
	created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (id) DO NOTHING
RETURNING id
`, itemID, courseID, item.SortOrder, item.Kind, item.Title, parentID,
			item.Published, item.VisibleFrom, item.Archived, item.DueAt, groupID,
			createdAt, updatedAt).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}

	_, err = pool.Exec(ctx, `
INSERT INTO course.course_structure_items (
	id, course_id, sort_order, kind, title, parent_id,
	published, visible_from, archived, due_at, assignment_group_id,
	created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (id) DO UPDATE SET
	course_id = EXCLUDED.course_id,
	sort_order = EXCLUDED.sort_order,
	kind = EXCLUDED.kind,
	title = EXCLUDED.title,
	parent_id = EXCLUDED.parent_id,
	published = EXCLUDED.published,
	visible_from = EXCLUDED.visible_from,
	archived = EXCLUDED.archived,
	due_at = EXCLUDED.due_at,
	assignment_group_id = EXCLUDED.assignment_group_id,
	updated_at = EXCLUDED.updated_at
`, itemID, courseID, item.SortOrder, item.Kind, item.Title, parentID,
		item.Published, item.VisibleFrom, item.Archived, item.DueAt, groupID,
		createdAt, updatedAt)
	if err != nil {
		return false, err
	}
	return true, nil
}

// SetItemDueAt updates due_at on a structure item of the given kind.
func SetItemDueAt(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, kind string, dueAt *time.Time) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	tag, err := pool.Exec(ctx, `
UPDATE course.course_structure_items
SET due_at = $3, updated_at = NOW()
WHERE id = $1 AND course_id = $2 AND kind = $4
`, itemID, courseID, dueAt, kind)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("structure item due date update failed")
	}
	return nil
}
