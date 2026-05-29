// Package accommodationaudit stores append-only IDEA compliance audit events (plan 12.10).
package accommodationaudit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Row struct {
	ID                uuid.UUID
	StudentID         uuid.UUID
	AccommodationType string
	ValueApplied      json.RawMessage
	Context           string
	ContextID         *uuid.UUID
	AppliedAt         time.Time
}

func Insert(
	ctx context.Context,
	pool *pgxpool.Pool,
	studentID uuid.UUID,
	accommodationType string,
	valueApplied json.RawMessage,
	contextName string,
	contextID *uuid.UUID,
) error {
	if valueApplied == nil {
		valueApplied = json.RawMessage(`{}`)
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.accommodation_audit_log (student_id, accommodation_type, value_applied, context, context_id)
VALUES ($1, $2, $3, $4, $5)
`, studentID, accommodationType, valueApplied, contextName, contextID)
	return err
}

type ListFilter struct {
	StudentID *uuid.UUID
	Limit     int
	Offset    int
}

func List(ctx context.Context, pool *pgxpool.Pool, f ListFilter) ([]Row, error) {
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	rows, err := pool.Query(ctx, `
SELECT id, student_id, accommodation_type, value_applied, context, context_id, applied_at
FROM course.accommodation_audit_log
WHERE ($1::uuid IS NULL OR student_id = $1)
ORDER BY applied_at DESC
LIMIT $2 OFFSET $3`, f.StudentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.StudentID, &r.AccommodationType, &r.ValueApplied, &r.Context, &r.ContextID, &r.AppliedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
