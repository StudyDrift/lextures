package contenttools

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationJobRow is a course.content_tool_migration_jobs row.
type MigrationJobRow struct {
	ID            uuid.UUID
	ToolID        string
	FromVersion   int
	ToVersion     int
	DryRun        bool
	Status        string
	TotalDocs     int
	MigratedDocs  int
	FailedDocs    int
	CursorStateID *uuid.UUID
	Error         *string
	CreatedAt     time.Time
	FinishedAt    *time.Time
}

// CreateMigrationJob inserts a queued job.
func CreateMigrationJob(
	ctx context.Context,
	pool *pgxpool.Pool,
	toolID string,
	fromVersion, toVersion int,
	dryRun bool,
) (*MigrationJobRow, error) {
	var r MigrationJobRow
	var cursor *uuid.UUID
	var errMsg *string
	var finished *time.Time
	err := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_migration_jobs (
  tool_id, from_version, to_version, dry_run, status
) VALUES ($1, $2, $3, $4, 'queued')
RETURNING id, tool_id, from_version, to_version, dry_run, status,
  total_docs, migrated_docs, failed_docs, cursor_state_id, error, created_at, finished_at
`, toolID, fromVersion, toVersion, dryRun).Scan(
		&r.ID, &r.ToolID, &r.FromVersion, &r.ToVersion, &r.DryRun, &r.Status,
		&r.TotalDocs, &r.MigratedDocs, &r.FailedDocs, &cursor, &errMsg, &r.CreatedAt, &finished,
	)
	if err != nil {
		return nil, err
	}
	r.CursorStateID = cursor
	r.Error = errMsg
	r.FinishedAt = finished
	return &r, nil
}

// GetMigrationJob returns a job by id.
func GetMigrationJob(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*MigrationJobRow, error) {
	var r MigrationJobRow
	var cursor *uuid.UUID
	var errMsg *string
	var finished *time.Time
	err := pool.QueryRow(ctx, `
SELECT id, tool_id, from_version, to_version, dry_run, status,
  total_docs, migrated_docs, failed_docs, cursor_state_id, error, created_at, finished_at
FROM course.content_tool_migration_jobs
WHERE id = $1
`, id).Scan(
		&r.ID, &r.ToolID, &r.FromVersion, &r.ToVersion, &r.DryRun, &r.Status,
		&r.TotalDocs, &r.MigratedDocs, &r.FailedDocs, &cursor, &errMsg, &r.CreatedAt, &finished,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.CursorStateID = cursor
	r.Error = errMsg
	r.FinishedAt = finished
	return &r, nil
}

// UpdateMigrationJobProgress persists counters/cursor/status.
func UpdateMigrationJobProgress(ctx context.Context, pool *pgxpool.Pool, r *MigrationJobRow) error {
	_, err := pool.Exec(ctx, `
UPDATE course.content_tool_migration_jobs
SET status = $2, total_docs = $3, migrated_docs = $4, failed_docs = $5,
    cursor_state_id = $6, error = $7, finished_at = $8
WHERE id = $1
`, r.ID, r.Status, r.TotalDocs, r.MigratedDocs, r.FailedDocs, r.CursorStateID, r.Error, r.FinishedAt)
	return err
}

// StateMigrationBatchRow is one state document eligible for eager migration.
type StateMigrationBatchRow struct {
	ID                 uuid.UUID
	InstanceID         uuid.UUID
	ToolID             string
	StateJSON          json.RawMessage
	StateSchemaVersion int
}

// ListStatesForMigration returns a batch of states for toolID with schema version in [from, to).
func ListStatesForMigration(
	ctx context.Context,
	pool *pgxpool.Pool,
	toolID string,
	fromVersion, toVersion int,
	afterStateID *uuid.UUID,
	limit int,
) ([]StateMigrationBatchRow, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	var (
		rows pgx.Rows
		err  error
	)
	if afterStateID == nil {
		rows, err = pool.Query(ctx, `
SELECT s.id, s.instance_id, i.tool_id, s.state_json, s.state_schema_version
FROM course.content_tool_states s
JOIN course.content_tool_instances i ON i.id = s.instance_id
WHERE i.tool_id = $1
  AND s.state_schema_version >= $2
  AND s.state_schema_version < $3
ORDER BY s.id
LIMIT $4
`, toolID, fromVersion, toVersion, limit)
	} else {
		rows, err = pool.Query(ctx, `
SELECT s.id, s.instance_id, i.tool_id, s.state_json, s.state_schema_version
FROM course.content_tool_states s
JOIN course.content_tool_instances i ON i.id = s.instance_id
WHERE i.tool_id = $1
  AND s.state_schema_version >= $2
  AND s.state_schema_version < $3
  AND s.id > $4
ORDER BY s.id
LIMIT $5
`, toolID, fromVersion, toVersion, *afterStateID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]StateMigrationBatchRow, 0, limit)
	for rows.Next() {
		var r StateMigrationBatchRow
		var state []byte
		if err := rows.Scan(&r.ID, &r.InstanceID, &r.ToolID, &state, &r.StateSchemaVersion); err != nil {
			return nil, err
		}
		r.StateJSON = json.RawMessage(state)
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountStatesForMigration counts documents needing migration.
func CountStatesForMigration(ctx context.Context, pool *pgxpool.Pool, toolID string, fromVersion, toVersion int) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM course.content_tool_states s
JOIN course.content_tool_instances i ON i.id = s.instance_id
WHERE i.tool_id = $1
  AND s.state_schema_version >= $2
  AND s.state_schema_version < $3
`, toolID, fromVersion, toVersion).Scan(&n)
	return n, err
}

// PersistMigratedState writes migrated JSON + schema version without bumping revision/interaction.
func PersistMigratedState(ctx context.Context, pool *pgxpool.Pool, stateID uuid.UUID, stateJSON json.RawMessage, schemaVersion int) error {
	if len(stateJSON) == 0 {
		stateJSON = json.RawMessage(`{}`)
	}
	_, err := pool.Exec(ctx, `
UPDATE course.content_tool_states
SET state_json = $2::jsonb, state_schema_version = $3, updated_at = NOW()
WHERE id = $1
`, stateID, stateJSON, schemaVersion)
	return err
}
