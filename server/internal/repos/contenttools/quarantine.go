package contenttools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QuarantineRow is a course.content_tool_state_quarantine row.
type QuarantineRow struct {
	ID           uuid.UUID
	StateID      uuid.UUID
	ToolID       string
	FromVersion  int
	ToVersion    int
	Error        string
	OriginalJSON json.RawMessage
	CreatedAt    time.Time
	ResolvedAt   *time.Time
}

// InsertQuarantine records a failed migration; original is preserved (FR-8).
func InsertQuarantine(
	ctx context.Context,
	pool *pgxpool.Pool,
	stateID uuid.UUID,
	toolID string,
	fromVersion, toVersion int,
	errMsg string,
	original json.RawMessage,
) (*QuarantineRow, error) {
	if len(original) == 0 {
		original = json.RawMessage(`{}`)
	}
	var r QuarantineRow
	var resolved *time.Time
	err := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_state_quarantine (
  state_id, tool_id, from_version, to_version, error, original_json
) VALUES ($1, $2, $3, $4, $5, $6::jsonb)
RETURNING id, state_id, tool_id, from_version, to_version, error, original_json, created_at, resolved_at
`, stateID, toolID, fromVersion, toVersion, errMsg, original).Scan(
		&r.ID, &r.StateID, &r.ToolID, &r.FromVersion, &r.ToVersion, &r.Error, &r.OriginalJSON, &r.CreatedAt, &resolved,
	)
	if err != nil {
		return nil, err
	}
	r.ResolvedAt = resolved
	return &r, nil
}

// ListQuarantine returns unresolved quarantine rows for a tool (empty tool = all).
func ListQuarantine(ctx context.Context, pool *pgxpool.Pool, toolID string, limit int) ([]QuarantineRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var (
		rows interface {
			Next() bool
			Scan(dest ...any) error
			Close()
			Err() error
		}
		err error
	)
	if toolID == "" {
		q, e := pool.Query(ctx, `
SELECT id, state_id, tool_id, from_version, to_version, error, original_json, created_at, resolved_at
FROM course.content_tool_state_quarantine
WHERE resolved_at IS NULL
ORDER BY created_at DESC
LIMIT $1
`, limit)
		rows, err = q, e
	} else {
		q, e := pool.Query(ctx, `
SELECT id, state_id, tool_id, from_version, to_version, error, original_json, created_at, resolved_at
FROM course.content_tool_state_quarantine
WHERE tool_id = $1 AND resolved_at IS NULL
ORDER BY created_at DESC
LIMIT $2
`, toolID, limit)
		rows, err = q, e
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]QuarantineRow, 0)
	for rows.Next() {
		var r QuarantineRow
		var resolved *time.Time
		if err := rows.Scan(
			&r.ID, &r.StateID, &r.ToolID, &r.FromVersion, &r.ToVersion, &r.Error, &r.OriginalJSON, &r.CreatedAt, &resolved,
		); err != nil {
			return nil, err
		}
		r.ResolvedAt = resolved
		out = append(out, r)
	}
	return out, rows.Err()
}

// HasOpenQuarantine reports whether stateID has an unresolved quarantine row.
func HasOpenQuarantine(ctx context.Context, pool *pgxpool.Pool, stateID uuid.UUID) (bool, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM course.content_tool_state_quarantine
WHERE state_id = $1 AND resolved_at IS NULL
`, stateID).Scan(&n)
	return n > 0, err
}
