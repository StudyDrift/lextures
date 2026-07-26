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

const stateCols = `
id, instance_id, enrollment_id, user_id, scope, state_json, state_schema_version,
revision, status, score_raw, score_max, interaction_count,
first_interacted_at, last_interacted_at, completed_at,
reset_count, last_reset_at, last_reset_by, created_at, updated_at
`

type stateScanner interface {
	Scan(dest ...any) error
}

func scanState(row stateScanner) (*StateRow, error) {
	var r StateRow
	var state []byte
	err := row.Scan(
		&r.ID, &r.InstanceID, &r.EnrollmentID, &r.UserID, &r.Scope, &state, &r.StateSchemaVersion,
		&r.Revision, &r.Status, &r.ScoreRaw, &r.ScoreMax, &r.InteractionCount,
		&r.FirstInteractedAt, &r.LastInteractedAt, &r.CompletedAt,
		&r.ResetCount, &r.LastResetAt, &r.LastResetBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(state) == 0 {
		state = []byte(`{}`)
	}
	r.StateJSON = json.RawMessage(state)
	if r.Scope == "" {
		r.Scope = ScopeEnrollment
	}
	return &r, nil
}

// GetState returns real (enrollment-scoped) state for (instance, enrollment), or nil.
func GetState(ctx context.Context, pool *pgxpool.Pool, instanceID, enrollmentID uuid.UUID) (*StateRow, error) {
	return GetStateByScope(ctx, pool, instanceID, enrollmentID, ScopeEnrollment)
}

// GetStateByScope returns state for (instance, enrollment, scope), or nil.
// For preview scope, returns the latest preview row when multiple exist.
func GetStateByScope(ctx context.Context, pool *pgxpool.Pool, instanceID, enrollmentID uuid.UUID, scope string) (*StateRow, error) {
	if scope == "" {
		scope = ScopeEnrollment
	}
	row := pool.QueryRow(ctx, `
SELECT `+stateCols+`
FROM course.content_tool_states
WHERE instance_id = $1 AND enrollment_id = $2 AND scope = $3
ORDER BY created_at DESC
LIMIT 1
`, instanceID, enrollmentID, scope)
	return scanState(row)
}

// UpsertState inserts or updates enrollment-scoped state with optimistic concurrency on revision.
// expectedRevision is the client's known revision; on conflict mismatch returns nil,nil for 409.
// Status advances not_started → in_progress (CT.1 helper; CT.3 writes use UpsertStateWithStatus).
func UpsertState(
	ctx context.Context,
	pool *pgxpool.Pool,
	instanceID, enrollmentID, userID uuid.UUID,
	stateJSON json.RawMessage,
	expectedRevision int64,
) (*StateRow, error) {
	return UpsertStateWithStatus(ctx, pool, instanceID, enrollmentID, userID, stateJSON, expectedRevision, "in_progress")
}

// UpsertPreviewState writes instructor preview-as-student state (scope=preview).
func UpsertPreviewState(
	ctx context.Context,
	pool *pgxpool.Pool,
	instanceID, enrollmentID, userID uuid.UUID,
	stateJSON json.RawMessage,
	expectedRevision int64,
) (*StateRow, error) {
	return UpsertPreviewStateWithStatus(ctx, pool, instanceID, enrollmentID, userID, stateJSON, expectedRevision, "in_progress")
}

// CountInstanceUsage returns distinct enrollments with real state, completed count, and last interaction.
func CountInstanceUsage(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID) (learnersWithState, learnersCompleted int, lastInteraction *time.Time, err error) {
	err = pool.QueryRow(ctx, `
SELECT
  COUNT(DISTINCT enrollment_id)::int,
  COUNT(DISTINCT enrollment_id) FILTER (WHERE status = 'completed')::int,
  MAX(last_interacted_at)
FROM course.content_tool_states
WHERE instance_id = $1 AND scope = 'enrollment'
`, instanceID).Scan(&learnersWithState, &learnersCompleted, &lastInteraction)
	return learnersWithState, learnersCompleted, lastInteraction, err
}

// PurgeStalePreviewStates deletes preview-scoped state rows older than olderThan.
func PurgeStalePreviewStates(ctx context.Context, pool *pgxpool.Pool, olderThan time.Duration) (int64, error) {
	if olderThan <= 0 {
		olderThan = 24 * time.Hour
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	tag, err := pool.Exec(ctx, `
DELETE FROM course.content_tool_states
WHERE scope = 'preview' AND created_at < $1
`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
