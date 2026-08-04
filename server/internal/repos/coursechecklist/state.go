package coursechecklist

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ValidDismissReasons are the allowed dismiss_reason values (empty means unset).
var ValidDismissReasons = map[string]struct{}{
	"":               {},
	"not_applicable": {},
	"done_elsewhere": {},
	"disagree":       {},
	"later":          {},
	"other":          {},
}

// MaxDismissNoteLen is the dismiss_note character cap (FR-15 / schema).
const MaxDismissNoteLen = 500

// ListDismissed returns currently dismissed item states for a course.
func ListDismissed(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]ItemState, error) {
	rows, err := pool.Query(ctx, `
SELECT course_id, item_id, dismissed_at, dismissed_by_user_id, dismiss_reason, dismiss_note,
       snoozed_until, restored_at, restored_by_user_id, created_at, updated_at
FROM course.course_checklist_item_state
WHERE course_id = $1 AND dismissed_at IS NOT NULL
ORDER BY item_id
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItemStates(rows)
}

// DismissInput is the payload for an idempotent dismiss.
type DismissInput struct {
	CourseID uuid.UUID
	ItemID   string
	ActorID  uuid.UUID
	Reason   string
	Note     string
}

// Dismiss upserts a dismissal. Idempotent: if already dismissed, returns the
// existing row without changing dismissed_at (FR-5).
func Dismiss(ctx context.Context, pool *pgxpool.Pool, in DismissInput) (ItemState, bool, error) {
	reason := strings.TrimSpace(in.Reason)
	if _, ok := ValidDismissReasons[reason]; !ok {
		return ItemState{}, false, ErrInvalidReason
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > MaxDismissNoteLen {
		return ItemState{}, false, ErrNoteTooLong
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return ItemState{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	existing, err := getStateTx(ctx, tx, in.CourseID, in.ItemID)
	if err != nil {
		return ItemState{}, false, err
	}
	if existing != nil && existing.DismissedAt != nil {
		if err := tx.Commit(ctx); err != nil {
			return ItemState{}, false, err
		}
		return *existing, false, nil
	}

	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
INSERT INTO course.course_checklist_item_state (
    course_id, item_id, dismissed_at, dismissed_by_user_id, dismiss_reason, dismiss_note,
    restored_at, restored_by_user_id, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, NULL, NULL, $3, $3)
ON CONFLICT (course_id, item_id) DO UPDATE SET
    dismissed_at = EXCLUDED.dismissed_at,
    dismissed_by_user_id = EXCLUDED.dismissed_by_user_id,
    dismiss_reason = EXCLUDED.dismiss_reason,
    dismiss_note = EXCLUDED.dismiss_note,
    restored_at = NULL,
    restored_by_user_id = NULL,
    updated_at = EXCLUDED.updated_at
`, in.CourseID, in.ItemID, now, in.ActorID, reason, note)
	if err != nil {
		return ItemState{}, false, err
	}
	if err := insertEventTx(ctx, tx, in.CourseID, in.ItemID, "dismiss", &in.ActorID, reason, now); err != nil {
		return ItemState{}, false, err
	}
	st, err := getStateTx(ctx, tx, in.CourseID, in.ItemID)
	if err != nil {
		return ItemState{}, false, err
	}
	if st == nil {
		return ItemState{}, false, errors.New("coursechecklist: dismiss row missing after upsert")
	}
	if err := tx.Commit(ctx); err != nil {
		return ItemState{}, false, err
	}
	return *st, true, nil
}

// Restore clears a dismissal and stamps restored_* (FR-6). No-op when not dismissed.
func Restore(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, itemID string, actorID uuid.UUID) (ItemState, bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return ItemState{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	existing, err := getStateTx(ctx, tx, courseID, itemID)
	if err != nil {
		return ItemState{}, false, err
	}
	if existing == nil || existing.DismissedAt == nil {
		if existing == nil {
			if err := tx.Commit(ctx); err != nil {
				return ItemState{}, false, err
			}
			return ItemState{CourseID: courseID, ItemID: itemID}, false, nil
		}
		if err := tx.Commit(ctx); err != nil {
			return ItemState{}, false, err
		}
		return *existing, false, nil
	}

	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
UPDATE course.course_checklist_item_state
SET dismissed_at = NULL,
    dismissed_by_user_id = NULL,
    dismiss_reason = '',
    dismiss_note = '',
    restored_at = $3,
    restored_by_user_id = $4,
    updated_at = $3
WHERE course_id = $1 AND item_id = $2
`, courseID, itemID, now, actorID)
	if err != nil {
		return ItemState{}, false, err
	}
	if err := insertEventTx(ctx, tx, courseID, itemID, "restore", &actorID, "", now); err != nil {
		return ItemState{}, false, err
	}
	st, err := getStateTx(ctx, tx, courseID, itemID)
	if err != nil {
		return ItemState{}, false, err
	}
	if st == nil {
		return ItemState{}, false, errors.New("coursechecklist: restore row missing after update")
	}
	if err := tx.Commit(ctx); err != nil {
		return ItemState{}, false, err
	}
	return *st, true, nil
}

// CountDismissed returns the number of currently dismissed items for a course.
func CountDismissed(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM course.course_checklist_item_state
WHERE course_id = $1 AND dismissed_at IS NOT NULL
`, courseID).Scan(&n)
	return n, err
}

func getStateTx(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, itemID string) (*ItemState, error) {
	row := tx.QueryRow(ctx, `
SELECT course_id, item_id, dismissed_at, dismissed_by_user_id, dismiss_reason, dismiss_note,
       snoozed_until, restored_at, restored_by_user_id, created_at, updated_at
FROM course.course_checklist_item_state
WHERE course_id = $1 AND item_id = $2
`, courseID, itemID)
	st, err := scanItemState(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func scanItemStates(rows pgx.Rows) ([]ItemState, error) {
	var out []ItemState
	for rows.Next() {
		st, err := scanItemState(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func scanItemState(row pgx.Row) (ItemState, error) {
	var st ItemState
	err := row.Scan(
		&st.CourseID, &st.ItemID, &st.DismissedAt, &st.DismissedByUserID,
		&st.DismissReason, &st.DismissNote, &st.SnoozedUntil,
		&st.RestoredAt, &st.RestoredByUserID, &st.CreatedAt, &st.UpdatedAt,
	)
	return st, err
}
