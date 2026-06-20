// Package assignmentoverrides implements plan 2.15 — differentiated assignments. It generalizes the
// former per-section due-date overrides and per-enrollment quiz overrides into one "assign to" model:
// an assignment/quiz item can have zero or more targets (everyone/section/group/student), each with
// optional due/availability overrides and (for quizzes) extra_attempts/time_multiplier. The most
// specific target wins: student > group > section > everyone.
package assignmentoverrides

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TargetEveryone = "everyone"
	TargetSection  = "section"
	TargetGroup    = "group"
	TargetStudent  = "student"
)

// OverrideRow is one assign-to target row for an item.
type OverrideRow struct {
	ID              uuid.UUID
	StructureItemID uuid.UUID
	TargetType      string
	TargetID        *uuid.UUID
	DueAt           *time.Time
	AvailableFrom   *time.Time
	AvailableUntil  *time.Time
	ExtraAttempts   *int32
	TimeMultiplier  *float64
	CreatedBy       *uuid.UUID
	CreatedAt       time.Time
}

// OverrideWrite is the instructor-supplied payload for one target when replacing an item's targets.
type OverrideWrite struct {
	TargetType     string
	TargetID       *uuid.UUID
	DueAt          *time.Time
	AvailableFrom  *time.Time
	AvailableUntil *time.Time
	ExtraAttempts  *int32
	TimeMultiplier *float64
}

// Effective is the resolved due/availability dates and quiz limits for one student on one item.
// A nil field means "use the item's own default" (no override applies for that field).
type Effective struct {
	DueAt          *time.Time
	AvailableFrom  *time.Time
	AvailableUntil *time.Time
	ExtraAttempts  *int32
	TimeMultiplier *float64
}

func validTargetType(t string) bool {
	switch t {
	case TargetEveryone, TargetSection, TargetGroup, TargetStudent:
		return true
	default:
		return false
	}
}

// ListForItem returns all assign-to targets for an item, oldest first.
func ListForItem(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) ([]OverrideRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, structure_item_id, target_type, target_id, due_at, available_from, available_until,
       extra_attempts, time_multiplier, created_by, created_at
FROM course.assignment_overrides
WHERE structure_item_id = $1
ORDER BY created_at ASC
`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OverrideRow{}
	for rows.Next() {
		var r OverrideRow
		if err := rows.Scan(
			&r.ID, &r.StructureItemID, &r.TargetType, &r.TargetID, &r.DueAt, &r.AvailableFrom, &r.AvailableUntil,
			&r.ExtraAttempts, &r.TimeMultiplier, &r.CreatedBy, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListForItems batch-fetches assign-to targets for many items in one query, keyed by structure_item_id.
// Used by list/dashboard endpoints to avoid N+1 queries (plan 2.15 NFR: resolution does not N+1 across
// students/items).
func ListForItems(ctx context.Context, pool *pgxpool.Pool, itemIDs []uuid.UUID) (map[uuid.UUID][]OverrideRow, error) {
	out := make(map[uuid.UUID][]OverrideRow)
	if len(itemIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
SELECT id, structure_item_id, target_type, target_id, due_at, available_from, available_until,
       extra_attempts, time_multiplier, created_by, created_at
FROM course.assignment_overrides
WHERE structure_item_id = ANY($1::uuid[])
ORDER BY created_at ASC
`, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r OverrideRow
		if err := rows.Scan(
			&r.ID, &r.StructureItemID, &r.TargetType, &r.TargetID, &r.DueAt, &r.AvailableFrom, &r.AvailableUntil,
			&r.ExtraAttempts, &r.TimeMultiplier, &r.CreatedBy, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out[r.StructureItemID] = append(out[r.StructureItemID], r)
	}
	return out, rows.Err()
}

// ReplaceForItem atomically replaces all assign-to targets for an item.
func ReplaceForItem(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, writes []OverrideWrite, createdBy uuid.UUID) error {
	for _, w := range writes {
		if !validTargetType(w.TargetType) {
			return fmt.Errorf("invalid target type %q", w.TargetType)
		}
		if w.TargetType == TargetEveryone && w.TargetID != nil {
			return errors.New("everyone target must not have a target id")
		}
		if w.TargetType != TargetEveryone && w.TargetID == nil {
			return fmt.Errorf("%s target requires a target id", w.TargetType)
		}
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
INSERT INTO course.assignment_overrides
  (structure_item_id, target_type, target_id, due_at, available_from, available_until, extra_attempts, time_multiplier, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
`, itemID, w.TargetType, w.TargetID, w.DueAt, w.AvailableFrom, w.AvailableUntil, w.ExtraAttempts, w.TimeMultiplier, createdBy); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// resolveBestMatch picks the most-specific target row that applies to this student: student > group >
// section > everyone. When multiple group targets match (the student is in several targeted groups),
// the earliest-created one wins for determinism.
func resolveBestMatch(rows []OverrideRow, enrollmentID uuid.UUID, sectionID *uuid.UUID, groupIDs []uuid.UUID) (*OverrideRow, bool) {
	var studentMatch, groupMatch, sectionMatch, everyoneMatch *OverrideRow
	for i := range rows {
		r := &rows[i]
		switch r.TargetType {
		case TargetStudent:
			if r.TargetID != nil && *r.TargetID == enrollmentID {
				studentMatch = r
			}
		case TargetGroup:
			if r.TargetID == nil {
				continue
			}
			for _, g := range groupIDs {
				if g == *r.TargetID && (groupMatch == nil || r.CreatedAt.Before(groupMatch.CreatedAt)) {
					groupMatch = r
				}
			}
		case TargetSection:
			if sectionID != nil && r.TargetID != nil && *r.TargetID == *sectionID {
				sectionMatch = r
			}
		case TargetEveryone:
			everyoneMatch = r
		}
	}
	switch {
	case studentMatch != nil:
		return studentMatch, true
	case groupMatch != nil:
		return groupMatch, true
	case sectionMatch != nil:
		return sectionMatch, true
	case everyoneMatch != nil:
		return everyoneMatch, true
	default:
		return nil, false
	}
}

// ResolveFromRows reports whether the item is visible to this student and, if so, the field-level
// overrides to apply on top of the item's own defaults (nil fields mean "keep the default"), given an
// already-fetched set of target rows for the item. An item with zero target rows is implicitly visible
// to everyone with no overrides (backward compatibility, plan 2.15 NFR "Backward compatibility").
func ResolveFromRows(rows []OverrideRow, enrollmentID uuid.UUID, sectionID *uuid.UUID, groupIDs []uuid.UUID) (visible bool, eff Effective) {
	if len(rows) == 0 {
		return true, Effective{}
	}
	match, ok := resolveBestMatch(rows, enrollmentID, sectionID, groupIDs)
	if !ok {
		return false, Effective{}
	}
	return true, Effective{
		DueAt:          match.DueAt,
		AvailableFrom:  match.AvailableFrom,
		AvailableUntil: match.AvailableUntil,
		ExtraAttempts:  match.ExtraAttempts,
		TimeMultiplier: match.TimeMultiplier,
	}
}

// Resolve is the single-item convenience form of ResolveFromRows; it fetches the item's target rows
// itself. Prefer ListForItems + ResolveFromRows when resolving many items at once.
func Resolve(ctx context.Context, pool *pgxpool.Pool, itemID, enrollmentID uuid.UUID, sectionID *uuid.UUID, groupIDs []uuid.UUID) (visible bool, eff Effective, err error) {
	rows, err := ListForItem(ctx, pool, itemID)
	if err != nil {
		return false, Effective{}, err
	}
	visible, eff = ResolveFromRows(rows, enrollmentID, sectionID, groupIDs)
	return visible, eff, nil
}

// StudentGroupIDs returns the group ids the enrollment belongs to (any group set, any course — callers
// pass an enrollment id already scoped to the right course).
func StudentGroupIDs(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT group_id FROM course.enrollment_group_memberships WHERE enrollment_id = $1
`, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []uuid.UUID{}
	for rows.Next() {
		var g uuid.UUID
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// HasOrphanedTargeting reports whether the item has explicit (non-"everyone") assign-to targets that,
// taken together, currently resolve to zero active enrolled students (e.g. targeted only at an empty
// section). Items with no override rows, or with an "everyone" target, are never orphaned.
func HasOrphanedTargeting(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (bool, error) {
	rows, err := ListForItem(ctx, pool, itemID)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	var sectionIDs, groupIDs []uuid.UUID
	for _, r := range rows {
		switch r.TargetType {
		case TargetEveryone:
			return false, nil
		case TargetStudent:
			if r.TargetID != nil {
				return false, nil
			}
		case TargetSection:
			if r.TargetID != nil {
				sectionIDs = append(sectionIDs, *r.TargetID)
			}
		case TargetGroup:
			if r.TargetID != nil {
				groupIDs = append(groupIDs, *r.TargetID)
			}
		}
	}
	if len(sectionIDs) == 0 && len(groupIDs) == 0 {
		return true, nil
	}
	var count int
	err = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_enrollments ce
WHERE ce.active = true AND (
  ce.section_id = ANY($1::uuid[])
  OR EXISTS (
    SELECT 1 FROM course.enrollment_group_memberships m
    WHERE m.enrollment_id = ce.id AND m.group_id = ANY($2::uuid[])
  )
)
`, sectionIDs, groupIDs).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}
