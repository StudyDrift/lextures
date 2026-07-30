package coursegrading

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/course"
)

// ReplaceAssignmentGroupsForImport replaces grading scale + assignment groups from an export bundle.
// Groups are inserted with their export UUIDs so structure rows can reference them.
func ReplaceAssignmentGroupsForImport(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, gradingScale string,
	groups []AssignmentGroupPublic,
) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	courseID, err := course.GetIDByCourseCode(ctx, pool, courseCode)
	if err != nil {
		return err
	}
	if courseID == nil {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
UPDATE course.courses
SET grading_scale = $1, updated_at = NOW()
WHERE course_code = $2
`, gradingScale, courseCode); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM course.assignment_groups WHERE course_id = $1`, *courseID); err != nil {
		return err
	}

	sorted := append([]AssignmentGroupPublic(nil), groups...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].SortOrder != sorted[j].SortOrder {
			return sorted[i].SortOrder < sorted[j].SortOrder
		}
		return sorted[i].Name < sorted[j].Name
	})

	order := 0
	for _, g := range sorted {
		name := strings.TrimSpace(g.Name)
		if name == "" {
			continue
		}
		order++
		w := g.WeightPercent
		if w < 0 {
			w = 0
		}
		if w > 100 {
			w = 100
		}
		dL := g.DropLowest
		if dL < 0 {
			dL = 0
		}
		dH := g.DropHighest
		if dH < 0 {
			dH = 0
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO course.assignment_groups (
	id, course_id, sort_order, name, weight_percent, drop_lowest, drop_highest, replace_lowest_with_final
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, g.ID, *courseID, order, name, w, dL, dH, g.ReplaceLowestWithFinal); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// InsertAssignmentGroupIfMissing inserts a group only when its id is not already present (mergeAdd).
func InsertAssignmentGroupIfMissing(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, id uuid.UUID,
	sortOrder int,
	name string,
	weightPercent float64,
) (bool, error) {
	if pool == nil {
		return false, errors.New("db pool is nil")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	if weightPercent < 0 {
		weightPercent = 0
	}
	if weightPercent > 100 {
		weightPercent = 100
	}
	var out uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO course.assignment_groups (id, course_id, sort_order, name, weight_percent, drop_lowest, drop_highest, replace_lowest_with_final)
VALUES ($1, $2, $3, $4, $5, 0, 0, false)
ON CONFLICT (id) DO NOTHING
RETURNING id
`, id, courseID, sortOrder, name, weightPercent).Scan(&out)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
