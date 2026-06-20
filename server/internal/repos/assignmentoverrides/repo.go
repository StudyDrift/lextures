// Package assignmentoverrides persists plan 2.15 "assign to" targets (everyone/section/group/student)
// and resolves the single effective due/availability window for a given student + item, generalizing
// the section-only overrides introduced in migration 131.
package assignmentoverrides

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Target is one assign-to row for an item.
type Target struct {
	ID              uuid.UUID
	StructureItemID uuid.UUID
	ItemType        string
	TargetType      string // everyone|section|group|student
	TargetID        *uuid.UUID
	DueAt           *time.Time
	AvailableFrom   *time.Time
	AvailableUntil  *time.Time
	CreatedBy       *uuid.UUID
	CreatedAt       time.Time
}

// TargetWrite is the instructor-supplied shape for replacing an item's targets.
type TargetWrite struct {
	TargetType     string
	TargetID       *uuid.UUID // nil only for "everyone"
	DueAt          *time.Time
	AvailableFrom  *time.Time
	AvailableUntil *time.Time
}

// BaseDates are the item's own due/availability fields, used as the per-field fallback
// when a matched target leaves a field unset.
type BaseDates struct {
	DueAt          *time.Time
	AvailableFrom  *time.Time
	AvailableUntil *time.Time
}

// Effective is one student's resolved view of an item.
type Effective struct {
	Visible        bool
	DueAt          *time.Time
	AvailableFrom  *time.Time
	AvailableUntil *time.Time
	// MatchedTarget is "everyone"|"section"|"group"|"student", or "" when the item has
	// no override rows at all (implicit everyone, base dates apply).
	MatchedTarget string
}

var validTargetTypes = map[string]bool{"everyone": true, "section": true, "group": true, "student": true}

// ErrInvalidTargetType is returned when a write specifies an unrecognized target type.
var ErrInvalidTargetType = errors.New("invalid target type")

// ErrTargetIDRequired is returned when a non-everyone target omits target id.
var ErrTargetIDRequired = errors.New("target id is required for non-everyone targets")

func scanTarget(row pgx.Row) (*Target, error) {
	var t Target
	if err := row.Scan(&t.ID, &t.StructureItemID, &t.ItemType, &t.TargetType, &t.TargetID, &t.DueAt, &t.AvailableFrom, &t.AvailableUntil, &t.CreatedBy, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

const selectTarget = `SELECT id, structure_item_id, item_type, target_type, target_id, due_at, available_from, available_until, created_by, created_at FROM course.assignment_overrides`

// ListForItem returns all assign-to targets for one item.
func ListForItem(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) ([]Target, error) {
	rows, err := pool.Query(ctx, selectTarget+` WHERE structure_item_id = $1 ORDER BY target_type, created_at`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Target
	for rows.Next() {
		t, err := scanTarget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// validateWrites checks target-type validity, required target ids, and uniqueness
// before any rows are touched.
func validateWrites(writes []TargetWrite) error {
	seen := map[string]bool{}
	for _, w := range writes {
		if !validTargetTypes[w.TargetType] {
			return ErrInvalidTargetType
		}
		if w.TargetType == "everyone" {
			if w.TargetID != nil {
				return ErrTargetIDRequired
			}
		} else if w.TargetID == nil {
			return ErrTargetIDRequired
		}
		key := w.TargetType
		if w.TargetID != nil {
			key += ":" + w.TargetID.String()
		}
		if seen[key] {
			return fmt.Errorf("duplicate target %s", key)
		}
		seen[key] = true
	}
	return nil
}

// ReplaceForItem atomically replaces all targets for an item with the given set.
func ReplaceForItem(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType string, writes []TargetWrite, createdBy uuid.UUID) error {
	if err := validateWrites(writes); err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM course.assignment_overrides WHERE structure_item_id = $1`, itemID); err != nil {
		return err
	}
	for _, w := range writes {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.assignment_overrides (structure_item_id, item_type, target_type, target_id, due_at, available_from, available_until, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, itemID, itemType, w.TargetType, w.TargetID, w.DueAt, w.AvailableFrom, w.AvailableUntil, createdBy); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// BulkExtendDueDate sets (or replaces) a student-level due date override for each enrollment,
// leaving any existing availability window on that student's row untouched.
func BulkExtendDueDate(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType string, enrollmentIDs []uuid.UUID, dueAt time.Time, createdBy uuid.UUID) error {
	if len(enrollmentIDs) == 0 {
		return nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, eid := range enrollmentIDs {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.assignment_overrides (structure_item_id, item_type, target_type, target_id, due_at, created_by)
VALUES ($1, $2, 'student', $3, $4, $5)
ON CONFLICT (structure_item_id, target_type, target_id) WHERE target_type <> 'everyone' DO UPDATE SET
  due_at = EXCLUDED.due_at,
  created_by = EXCLUDED.created_by,
  updated_at = NOW()
`, itemID, itemType, eid, dueAt, createdBy); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

type studentContext struct {
	EnrollmentID uuid.UUID
	SectionID    *uuid.UUID
	GroupIDs     []uuid.UUID
}

func loadStudentContext(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (*studentContext, error) {
	sc := &studentContext{EnrollmentID: enrollmentID}
	if err := pool.QueryRow(ctx, `SELECT section_id FROM course.course_enrollments WHERE id = $1`, enrollmentID).Scan(&sc.SectionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sc, nil
		}
		return nil, err
	}
	rows, err := pool.Query(ctx, `SELECT group_id FROM course.enrollment_group_memberships WHERE enrollment_id = $1`, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var gid uuid.UUID
		if err := rows.Scan(&gid); err != nil {
			return nil, err
		}
		sc.GroupIDs = append(sc.GroupIDs, gid)
	}
	return sc, rows.Err()
}

func containsUUID(list []uuid.UUID, id uuid.UUID) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

func coalesceTime(override, base *time.Time) *time.Time {
	if override != nil {
		return override
	}
	return base
}

func resolve(targets []Target, stu *studentContext, base BaseDates) Effective {
	if len(targets) == 0 {
		return Effective{Visible: true, DueAt: base.DueAt, AvailableFrom: base.AvailableFrom, AvailableUntil: base.AvailableUntil}
	}
	var student, group, section, everyone *Target
	for i := range targets {
		t := &targets[i]
		switch t.TargetType {
		case "student":
			if stu != nil && t.TargetID != nil && *t.TargetID == stu.EnrollmentID {
				student = t
			}
		case "group":
			if stu != nil && t.TargetID != nil && containsUUID(stu.GroupIDs, *t.TargetID) {
				group = t
			}
		case "section":
			if stu != nil && stu.SectionID != nil && t.TargetID != nil && *t.TargetID == *stu.SectionID {
				section = t
			}
		case "everyone":
			everyone = t
		}
	}
	match, matchedType := student, "student"
	if match == nil {
		match, matchedType = group, "group"
	}
	if match == nil {
		match, matchedType = section, "section"
	}
	if match == nil {
		match, matchedType = everyone, "everyone"
	}
	if match == nil {
		return Effective{Visible: false}
	}
	return Effective{
		Visible:        true,
		MatchedTarget:  matchedType,
		DueAt:          coalesceTime(match.DueAt, base.DueAt),
		AvailableFrom:  coalesceTime(match.AvailableFrom, base.AvailableFrom),
		AvailableUntil: coalesceTime(match.AvailableUntil, base.AvailableUntil),
	}
}

// EffectiveForStudent resolves the most-specific-wins effective dates/visibility for one item.
func EffectiveForStudent(ctx context.Context, pool *pgxpool.Pool, itemID, enrollmentID uuid.UUID, base BaseDates) (Effective, error) {
	targets, err := ListForItem(ctx, pool, itemID)
	if err != nil {
		return Effective{}, err
	}
	if len(targets) == 0 {
		return Effective{Visible: true, DueAt: base.DueAt, AvailableFrom: base.AvailableFrom, AvailableUntil: base.AvailableUntil}, nil
	}
	stu, err := loadStudentContext(ctx, pool, enrollmentID)
	if err != nil {
		return Effective{}, err
	}
	return resolve(targets, stu, base), nil
}

// EffectiveForStudentBatch resolves effective dates/visibility for many items in one round trip
// (avoids N+1 across a student's dashboard/calendar/gradebook listing).
func EffectiveForStudentBatch(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, itemIDs []uuid.UUID, bases map[uuid.UUID]BaseDates) (map[uuid.UUID]Effective, error) {
	out := make(map[uuid.UUID]Effective, len(itemIDs))
	if len(itemIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, selectTarget+` WHERE structure_item_id = ANY($1)`, itemIDs)
	if err != nil {
		return nil, err
	}
	grouped := map[uuid.UUID][]Target{}
	for rows.Next() {
		t, err := scanTarget(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		grouped[t.StructureItemID] = append(grouped[t.StructureItemID], *t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	var stu *studentContext
	hasAnyTargets := len(grouped) > 0
	if hasAnyTargets {
		stu, err = loadStudentContext(ctx, pool, enrollmentID)
		if err != nil {
			return nil, err
		}
	}
	for _, itemID := range itemIDs {
		out[itemID] = resolve(grouped[itemID], stu, bases[itemID])
	}
	return out, nil
}

// IsOrphaned reports whether an item's explicit targets (if any) match zero active students
// in the course — i.e. it was assigned to nobody. Items with no targets at all (implicit
// everyone) or an explicit "everyone" target are never orphaned.
func IsOrphaned(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (bool, error) {
	targets, err := ListForItem(ctx, pool, itemID)
	if err != nil {
		return false, err
	}
	if len(targets) == 0 {
		return false, nil
	}
	var sectionIDs, groupIDs, studentIDs []uuid.UUID
	for _, t := range targets {
		switch t.TargetType {
		case "everyone":
			return false, nil
		case "section":
			sectionIDs = append(sectionIDs, *t.TargetID)
		case "group":
			groupIDs = append(groupIDs, *t.TargetID)
		case "student":
			studentIDs = append(studentIDs, *t.TargetID)
		}
	}
	var exists bool
	err = pool.QueryRow(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM course.course_enrollments ce
	INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
	LEFT JOIN course.enrollment_group_memberships egm ON egm.enrollment_id = ce.id
	WHERE ce.course_id = $1 AND ce.active
	  AND (ce.section_id = ANY($2) OR egm.group_id = ANY($3) OR ce.id = ANY($4))
)
`, courseID, sectionIDs, groupIDs, studentIDs).Scan(&exists)
	if err != nil {
		return false, err
	}
	return !exists, nil
}
