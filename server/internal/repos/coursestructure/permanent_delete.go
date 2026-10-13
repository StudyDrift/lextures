package coursestructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrStudentWork means a permanent delete would remove submissions or grades
	// and the caller has not confirmed that.
	ErrStudentWork = errors.New("structure item has student submissions or grades")
	// ErrNotInTrash means the item exists but is not an archived child, so it
	// cannot be permanently deleted from Trash.
	ErrNotInTrash = errors.New("structure item is not in trash")
)

// PermanentlyDeleteArchivedChild removes an archived module child. Dependent
// rows (grades, submissions, attempts) leave with ON DELETE CASCADE, so the
// item no longer appears in structure or gradebook APIs.
//
// When the item has student work and confirmStudentWork is false, the delete
// is rolled back and ErrStudentWork is returned. A missing item is pgx.ErrNoRows.
// A live (not archived) item is ErrNotInTrash.
func PermanentlyDeleteArchivedChild(
	ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, confirmStudentWork bool,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var archived bool
	err = tx.QueryRow(ctx, `
		SELECT archived
		FROM course.course_structure_items
		WHERE id = $1
		  AND course_id = $2
		  AND parent_id IS NOT NULL
		  AND kind `+patchableChildKindsSQL+`
		FOR UPDATE
	`, itemID, courseID).Scan(&archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgx.ErrNoRows
	}
	if err != nil {
		return err
	}
	if !archived {
		return ErrNotInTrash
	}

	if !confirmStudentWork {
		var has bool
		err = tx.QueryRow(ctx, `
			SELECT
				EXISTS (SELECT 1 FROM course.course_grades g WHERE g.module_item_id = $1)
				OR EXISTS (SELECT 1 FROM course.module_assignment_submissions s WHERE s.module_item_id = $1)
				OR EXISTS (SELECT 1 FROM course.quiz_attempts a WHERE a.structure_item_id = $1)
				OR EXISTS (SELECT 1 FROM course.module_survey_responses r WHERE r.structure_item_id = $1)
		`, itemID).Scan(&has)
		if err != nil {
			return err
		}
		if has {
			return ErrStudentWork
		}
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM course.course_structure_items
		WHERE id = $1 AND course_id = $2
	`, itemID, courseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return tx.Commit(ctx)
}
