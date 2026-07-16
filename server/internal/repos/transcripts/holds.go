package transcripts

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/transcriptorder"
)

var (
	ErrHoldNotFound      = errors.New("hold not found")
	ErrHoldAlreadyReleased = errors.New("hold already released")
)

// Hold is a blocking transcript hold on a student.
type Hold struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OrgID          *uuid.UUID
	Type           transcriptorder.HoldType
	Reason         *string
	StudentMessage *string
	ExternalID     *string
	PlacedBy       *uuid.UUID
	PlacedAt       time.Time
	ReleasedBy     *uuid.UUID
	ReleasedAt     *time.Time
}

// PlaceHoldInput creates a new hold.
type PlaceHoldInput struct {
	UserID         uuid.UUID
	OrgID          *uuid.UUID
	Type           transcriptorder.HoldType
	Reason         *string
	StudentMessage *string
	ExternalID     *string
	PlacedBy       *uuid.UUID
}

const holdSelectColumns = `
id, user_id, org_id, type, reason, student_message, external_id,
placed_by, placed_at, released_by, released_at`

func scanHold(row pgx.Row, h *Hold) error {
	var typ string
	err := row.Scan(
		&h.ID, &h.UserID, &h.OrgID, &typ, &h.Reason, &h.StudentMessage, &h.ExternalID,
		&h.PlacedBy, &h.PlacedAt, &h.ReleasedBy, &h.ReleasedAt,
	)
	if err != nil {
		return err
	}
	h.Type = transcriptorder.HoldType(typ)
	return nil
}

// Active returns true when the hold has not been released.
func (h Hold) Active() bool {
	return h.ReleasedAt == nil
}

// StudentMessageSafe returns sanitized student-facing copy.
func (h Hold) StudentMessageSafe() string {
	return transcriptorder.StudentFacingMessage(h.Type, h.StudentMessage)
}

// PlaceHold inserts a hold (non-external).
func PlaceHold(ctx context.Context, pool *pgxpool.Pool, in PlaceHoldInput) (*Hold, error) {
	var h Hold
	err := scanHold(pool.QueryRow(ctx, `
INSERT INTO transcripts.holds (
    user_id, org_id, type, reason, student_message, external_id, placed_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING `+holdSelectColumns+`
`, in.UserID, in.OrgID, string(in.Type), in.Reason, in.StudentMessage, in.ExternalID, in.PlacedBy), &h)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// UpsertExternalHold idempotently upserts by (org_id, external_id).
// If the hold was previously released, it is re-activated (released_* cleared).
func UpsertExternalHold(ctx context.Context, pool *pgxpool.Pool, in PlaceHoldInput) (*Hold, error) {
	if in.ExternalID == nil || strings.TrimSpace(*in.ExternalID) == "" {
		return PlaceHold(ctx, pool, in)
	}
	ext := strings.TrimSpace(*in.ExternalID)
	var existing Hold
	err := scanHold(pool.QueryRow(ctx, `
SELECT `+holdSelectColumns+`
FROM transcripts.holds
WHERE external_id = $1
  AND (
    ($2::uuid IS NULL AND org_id IS NULL)
    OR org_id = $2
  )
LIMIT 1
`, ext, in.OrgID), &existing)
	if err == nil {
		var h Hold
		err = scanHold(pool.QueryRow(ctx, `
UPDATE transcripts.holds
SET user_id = $2,
    type = $3,
    reason = $4,
    student_message = $5,
    placed_by = COALESCE($6, placed_by),
    released_by = NULL,
    released_at = NULL
WHERE id = $1
RETURNING `+holdSelectColumns+`
`, existing.ID, in.UserID, string(in.Type), in.Reason, in.StudentMessage, in.PlacedBy), &h)
		if err != nil {
			return nil, err
		}
		return &h, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	in.ExternalID = &ext
	return PlaceHold(ctx, pool, in)
}

// ReleaseHold marks a hold released.
func ReleaseHold(ctx context.Context, pool *pgxpool.Pool, holdID uuid.UUID, releasedBy *uuid.UUID) (*Hold, error) {
	var h Hold
	err := scanHold(pool.QueryRow(ctx, `
UPDATE transcripts.holds
SET released_by = $2, released_at = NOW()
WHERE id = $1 AND released_at IS NULL
RETURNING `+holdSelectColumns+`
`, holdID, releasedBy), &h)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, getErr := GetHold(ctx, pool, holdID)
		if getErr != nil {
			return nil, getErr
		}
		if existing.ReleasedAt != nil {
			return nil, ErrHoldAlreadyReleased
		}
		return nil, ErrHoldNotFound
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// GetHold loads a hold by id.
func GetHold(ctx context.Context, pool *pgxpool.Pool, holdID uuid.UUID) (*Hold, error) {
	var h Hold
	err := scanHold(pool.QueryRow(ctx, `
SELECT `+holdSelectColumns+`
FROM transcripts.holds
WHERE id = $1
`, holdID), &h)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrHoldNotFound
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// ListActiveHoldsForUser returns active holds for a user (optionally scoped to org).
func ListActiveHoldsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, orgID *uuid.UUID) ([]Hold, error) {
	rows, err := pool.Query(ctx, `
SELECT `+holdSelectColumns+`
FROM transcripts.holds
WHERE user_id = $1
  AND released_at IS NULL
  AND ($2::uuid IS NULL OR org_id IS NULL OR org_id = $2)
ORDER BY placed_at DESC
`, userID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Hold
	for rows.Next() {
		var h Hold
		if err := scanHold(rows, &h); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ListHolds filters holds for the registrar console.
func ListHolds(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID, userID *uuid.UUID, activeOnly bool, limit int) ([]Hold, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT `+holdSelectColumns+`
FROM transcripts.holds
WHERE ($1::uuid IS NULL OR org_id IS NULL OR org_id = $1)
  AND ($2::uuid IS NULL OR user_id = $2)
  AND (NOT $3 OR released_at IS NULL)
ORDER BY placed_at DESC
LIMIT $4
`, orgID, userID, activeOnly, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Hold
	for rows.Next() {
		var h Hold
		if err := scanHold(rows, &h); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// HasBlockingHold reports whether the student has any active hold for the org scope.
func HasBlockingHold(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, orgID *uuid.UUID) (bool, error) {
	holds, err := ListActiveHoldsForUser(ctx, pool, userID, orgID)
	if err != nil {
		return false, err
	}
	return len(holds) > 0, nil
}

// FindHoldByExternalID looks up a hold by org + external idempotency key.
func FindHoldByExternalID(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID, externalID string) (*Hold, error) {
	ext := strings.TrimSpace(externalID)
	if ext == "" {
		return nil, ErrHoldNotFound
	}
	var h Hold
	err := scanHold(pool.QueryRow(ctx, `
SELECT `+holdSelectColumns+`
FROM transcripts.holds
WHERE external_id = $1
  AND (
    ($2::uuid IS NULL AND org_id IS NULL)
    OR org_id = $2
  )
ORDER BY placed_at DESC
LIMIT 1
`, ext, orgID), &h)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrHoldNotFound
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}
