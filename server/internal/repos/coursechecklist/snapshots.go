package coursechecklist

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxPayloadBytes is the snapshot payload size ceiling (256 KiB).
const MaxPayloadBytes = 262144

// GetSnapshot returns the cached snapshot for a course, or nil if missing.
func GetSnapshot(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*Snapshot, error) {
	row := pool.QueryRow(ctx, `
SELECT course_id, computed_at, engine_version, catalog_version, payload,
       total_count, done_count, outstanding_essential, outstanding_total, dismissed_count
FROM course.course_checklist_snapshots
WHERE course_id = $1
`, courseID)
	var s Snapshot
	err := row.Scan(
		&s.CourseID, &s.ComputedAt, &s.EngineVersion, &s.CatalogVersion, &s.Payload,
		&s.TotalCount, &s.DoneCount, &s.OutstandingEssential, &s.OutstandingTotal, &s.DismissedCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpsertSnapshotInput is the write payload for a snapshot row.
type UpsertSnapshotInput struct {
	CourseID             uuid.UUID
	ComputedAt           time.Time
	EngineVersion        int
	CatalogVersion       string
	Payload              json.RawMessage
	TotalCount           int
	DoneCount            int
	OutstandingEssential int
	OutstandingTotal     int
	DismissedCount       int
}

// UpsertSnapshot writes/replaces the snapshot. Returns ErrPayloadTooLarge when
// the payload exceeds MaxPayloadBytes.
func UpsertSnapshot(ctx context.Context, pool *pgxpool.Pool, in UpsertSnapshotInput) error {
	if len(in.Payload) > MaxPayloadBytes {
		return ErrPayloadTooLarge
	}
	if in.ComputedAt.IsZero() {
		in.ComputedAt = time.Now().UTC()
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.course_checklist_snapshots (
    course_id, computed_at, engine_version, catalog_version, payload,
    total_count, done_count, outstanding_essential, outstanding_total, dismissed_count
) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10)
ON CONFLICT (course_id) DO UPDATE SET
    computed_at = EXCLUDED.computed_at,
    engine_version = EXCLUDED.engine_version,
    catalog_version = EXCLUDED.catalog_version,
    payload = EXCLUDED.payload,
    total_count = EXCLUDED.total_count,
    done_count = EXCLUDED.done_count,
    outstanding_essential = EXCLUDED.outstanding_essential,
    outstanding_total = EXCLUDED.outstanding_total,
    dismissed_count = EXCLUDED.dismissed_count
`, in.CourseID, in.ComputedAt, in.EngineVersion, in.CatalogVersion, []byte(in.Payload),
		in.TotalCount, in.DoneCount, in.OutstandingEssential, in.OutstandingTotal, in.DismissedCount)
	return err
}

// UpdateSnapshotCounters patches denormalised counters without changing the payload.
func UpdateSnapshotCounters(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, outstandingEssential, outstandingTotal, done, total, dismissed int) error {
	_, err := pool.Exec(ctx, `
UPDATE course.course_checklist_snapshots
SET outstanding_essential = $2,
    outstanding_total = $3,
    done_count = $4,
    total_count = $5,
    dismissed_count = $6
WHERE course_id = $1
`, courseID, outstandingEssential, outstandingTotal, done, total, dismissed)
	return err
}

// MutationFreshnessForCourse returns course.updated_at and max structure updated_at.
func MutationFreshnessForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (MutationFreshness, error) {
	var m MutationFreshness
	err := pool.QueryRow(ctx, `
SELECT c.updated_at,
       (SELECT MAX(si.updated_at) FROM course.course_structure_items si WHERE si.course_id = c.id)
FROM course.courses c
WHERE c.id = $1
`, courseID).Scan(&m.CourseUpdatedAt, &m.StructureMaxAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return MutationFreshness{}, pgx.ErrNoRows
	}
	return m, err
}

// DeleteSnapshotsUntouchedSince deletes snapshots for courses with no enrollment
// activity and no course updates since cutoff (nightly sweeper).
func DeleteSnapshotsUntouchedSince(ctx context.Context, pool *pgxpool.Pool, cutoff time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM course.course_checklist_snapshots s
WHERE s.computed_at < $1
  AND EXISTS (
    SELECT 1 FROM course.courses c
    WHERE c.id = s.course_id AND c.updated_at < $1
  )
  AND NOT EXISTS (
    SELECT 1 FROM course.course_enrollments ce
    WHERE ce.course_id = s.course_id
      AND ce.active
      AND ce.created_at >= $1
  )
`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
