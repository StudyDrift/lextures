// Package enrollmentgroups provides CRUD for instructor-managed enrollment group sets.
package enrollmentgroups

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/enrollmentgroup"
)

// IsEnabled returns whether enrollment groups are enabled for the course.
func IsEnabled(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (bool, error) {
	var enabled bool
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(enrollment_groups_enabled, false)
		FROM course.courses
		WHERE id = $1
	`, courseID).Scan(&enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return enabled, nil
}

// SetEnabled toggles enrollment_groups_enabled on the course.
func SetEnabled(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, enabled bool) error {
	tag, err := pool.Exec(ctx, `
		UPDATE course.courses
		SET enrollment_groups_enabled = $1, updated_at = NOW()
		WHERE id = $2
	`, enabled, courseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Tree returns group sets with nested groups and enrollment IDs.
func Tree(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (enrollmentgroup.EnrollmentGroupsTreeResponse, error) {
	out := enrollmentgroup.EnrollmentGroupsTreeResponse{GroupSets: []enrollmentgroup.EnrollmentGroupSetPublic{}}

	setRows, err := pool.Query(ctx, `
		SELECT id, name, sort_order
		FROM course.enrollment_group_sets
		WHERE course_id = $1
		ORDER BY sort_order ASC, name ASC
	`, courseID)
	if err != nil {
		return out, err
	}
	defer setRows.Close()

	type setRow struct {
		id        uuid.UUID
		name      string
		sortOrder int32
	}
	var sets []setRow
	for setRows.Next() {
		var s setRow
		if err := setRows.Scan(&s.id, &s.name, &s.sortOrder); err != nil {
			return out, err
		}
		sets = append(sets, s)
	}
	if err := setRows.Err(); err != nil {
		return out, err
	}
	if len(sets) == 0 {
		return out, nil
	}

	setIDs := make([]uuid.UUID, len(sets))
	for i, s := range sets {
		setIDs[i] = s.id
	}

	groupRows, err := pool.Query(ctx, `
		SELECT id, group_set_id, name, sort_order
		FROM course.enrollment_groups
		WHERE group_set_id = ANY($1)
		ORDER BY sort_order ASC, name ASC
	`, setIDs)
	if err != nil {
		return out, err
	}
	defer groupRows.Close()

	type groupRow struct {
		id        uuid.UUID
		setID     uuid.UUID
		name      string
		sortOrder int32
	}
	var groups []groupRow
	groupIDs := make([]uuid.UUID, 0)
	for groupRows.Next() {
		var g groupRow
		if err := groupRows.Scan(&g.id, &g.setID, &g.name, &g.sortOrder); err != nil {
			return out, err
		}
		groups = append(groups, g)
		groupIDs = append(groupIDs, g.id)
	}
	if err := groupRows.Err(); err != nil {
		return out, err
	}

	membersByGroup := map[uuid.UUID][]uuid.UUID{}
	if len(groupIDs) > 0 {
		memRows, err := pool.Query(ctx, `
			SELECT group_id, enrollment_id
			FROM course.enrollment_group_memberships
			WHERE group_id = ANY($1)
		`, groupIDs)
		if err != nil {
			return out, err
		}
		defer memRows.Close()
		for memRows.Next() {
			var gid, eid uuid.UUID
			if err := memRows.Scan(&gid, &eid); err != nil {
				return out, err
			}
			membersByGroup[gid] = append(membersByGroup[gid], eid)
		}
		if err := memRows.Err(); err != nil {
			return out, err
		}
	}

	groupsBySet := map[uuid.UUID][]enrollmentgroup.EnrollmentGroupPublic{}
	for _, g := range groups {
		ids := membersByGroup[g.id]
		if ids == nil {
			ids = []uuid.UUID{}
		}
		groupsBySet[g.setID] = append(groupsBySet[g.setID], enrollmentgroup.EnrollmentGroupPublic{
			ID:            g.id,
			Name:          g.name,
			SortOrder:     g.sortOrder,
			EnrollmentIDs: ids,
		})
	}

	out.GroupSets = make([]enrollmentgroup.EnrollmentGroupSetPublic, 0, len(sets))
	for _, s := range sets {
		gs := groupsBySet[s.id]
		if gs == nil {
			gs = []enrollmentgroup.EnrollmentGroupPublic{}
		}
		out.GroupSets = append(out.GroupSets, enrollmentgroup.EnrollmentGroupSetPublic{
			ID:        s.id,
			Name:      s.name,
			SortOrder: s.sortOrder,
			Groups:    gs,
		})
	}
	return out, nil
}

// CreateSet inserts a group set and returns its id.
func CreateSet(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, name string) (uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return uuid.Nil, errors.New("name required")
	}
	var sortOrder int32
	_ = pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(sort_order), -1) + 1
		FROM course.enrollment_group_sets
		WHERE course_id = $1
	`, courseID).Scan(&sortOrder)

	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO course.enrollment_group_sets (course_id, name, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id
	`, courseID, name, sortOrder).Scan(&id)
	return id, err
}

// CreateGroup inserts a group in a set belonging to the course.
func CreateGroup(ctx context.Context, pool *pgxpool.Pool, courseID, setID uuid.UUID, name string) (uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return uuid.Nil, errors.New("name required")
	}
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM course.enrollment_group_sets
			WHERE id = $1 AND course_id = $2
		)
	`, setID, courseID).Scan(&ok)
	if err != nil {
		return uuid.Nil, err
	}
	if !ok {
		return uuid.Nil, pgx.ErrNoRows
	}

	var sortOrder int32
	_ = pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(sort_order), -1) + 1
		FROM course.enrollment_groups
		WHERE group_set_id = $1
	`, setID).Scan(&sortOrder)

	var id uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO course.enrollment_groups (group_set_id, name, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id
	`, setID, name, sortOrder).Scan(&id)
	return id, err
}

// PatchSetName renames a set in the course.
func PatchSetName(ctx context.Context, pool *pgxpool.Pool, courseID, setID uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name required")
	}
	tag, err := pool.Exec(ctx, `
		UPDATE course.enrollment_group_sets
		SET name = $1
		WHERE id = $2 AND course_id = $3
	`, name, setID, courseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// PatchGroupName renames a group in the course.
func PatchGroupName(ctx context.Context, pool *pgxpool.Pool, courseID, groupID uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name required")
	}
	tag, err := pool.Exec(ctx, `
		UPDATE course.enrollment_groups g
		SET name = $1
		FROM course.enrollment_group_sets s
		WHERE g.id = $2
		  AND g.group_set_id = s.id
		  AND s.course_id = $3
	`, name, groupID, courseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteSet removes a set (cascade groups/memberships).
func DeleteSet(ctx context.Context, pool *pgxpool.Pool, courseID, setID uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
		DELETE FROM course.enrollment_group_sets
		WHERE id = $1 AND course_id = $2
	`, setID, courseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteGroup removes a group.
func DeleteGroup(ctx context.Context, pool *pgxpool.Pool, courseID, groupID uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
		DELETE FROM course.enrollment_groups g
		USING course.enrollment_group_sets s
		WHERE g.id = $1
		  AND g.group_set_id = s.id
		  AND s.course_id = $2
	`, groupID, courseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// PutMembership assigns or clears membership for one enrollment in a set.
// groupID nil clears membership for that set.
func PutMembership(ctx context.Context, pool *pgxpool.Pool, courseID, enrollmentID, groupSetID uuid.UUID, groupID *uuid.UUID) error {
	// Ensure enrollment and set belong to course.
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM course.course_enrollments e
			WHERE e.id = $1 AND e.course_id = $2
		) AND EXISTS (
			SELECT 1
			FROM course.enrollment_group_sets s
			WHERE s.id = $3 AND s.course_id = $2
		)
	`, enrollmentID, courseID, groupSetID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return pgx.ErrNoRows
	}

	if groupID == nil {
		_, err = pool.Exec(ctx, `
			DELETE FROM course.enrollment_group_memberships
			WHERE enrollment_id = $1 AND group_set_id = $2
		`, enrollmentID, groupSetID)
		return err
	}

	// Group must belong to the set.
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM course.enrollment_groups
			WHERE id = $1 AND group_set_id = $2
		)
	`, *groupID, groupSetID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return pgx.ErrNoRows
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO course.enrollment_group_memberships (enrollment_id, group_set_id, group_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (enrollment_id, group_set_id)
		DO UPDATE SET group_id = EXCLUDED.group_id
	`, enrollmentID, groupSetID, *groupID)
	return err
}
