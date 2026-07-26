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

func scanState(row pgx.Row) (*StateRow, error) {
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
func UpsertState(
	ctx context.Context,
	pool *pgxpool.Pool,
	instanceID, enrollmentID, userID uuid.UUID,
	stateJSON json.RawMessage,
	expectedRevision int64,
) (*StateRow, error) {
	if len(stateJSON) == 0 {
		stateJSON = json.RawMessage(`{}`)
	}
	row := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_states (
  instance_id, enrollment_id, user_id, scope, state_json, revision, status,
  interaction_count, first_interacted_at, last_interacted_at
) VALUES (
  $1, $2, $3, 'enrollment', $4::jsonb, 1, 'in_progress',
  1, NOW(), NOW()
)
ON CONFLICT (instance_id, enrollment_id) WHERE (scope = 'enrollment') DO UPDATE SET
  state_json = EXCLUDED.state_json,
  revision = course.content_tool_states.revision + 1,
  status = CASE
    WHEN course.content_tool_states.status = 'not_started' THEN 'in_progress'
    ELSE course.content_tool_states.status
  END,
  interaction_count = course.content_tool_states.interaction_count + 1,
  first_interacted_at = COALESCE(course.content_tool_states.first_interacted_at, NOW()),
  last_interacted_at = NOW(),
  updated_at = NOW()
WHERE course.content_tool_states.revision = $5
RETURNING `+stateCols+`
`, instanceID, enrollmentID, userID, stateJSON, expectedRevision)
	return scanState(row)
}

// UpsertPreviewState writes instructor preview-as-student state (scope=preview).
// Preview rows do not participate in enrollment uniqueness; updates the latest
// preview row for (instance, enrollment) or inserts a new one.
func UpsertPreviewState(
	ctx context.Context,
	pool *pgxpool.Pool,
	instanceID, enrollmentID, userID uuid.UUID,
	stateJSON json.RawMessage,
	expectedRevision int64,
) (*StateRow, error) {
	if len(stateJSON) == 0 {
		stateJSON = json.RawMessage(`{}`)
	}
	existing, err := GetStateByScope(ctx, pool, instanceID, enrollmentID, ScopePreview)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		row := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_states (
  instance_id, enrollment_id, user_id, scope, state_json, revision, status,
  interaction_count, first_interacted_at, last_interacted_at
) VALUES (
  $1, $2, $3, 'preview', $4::jsonb, 1, 'in_progress',
  1, NOW(), NOW()
)
RETURNING `+stateCols+`
`, instanceID, enrollmentID, userID, stateJSON)
		return scanState(row)
	}
	if existing.Revision != expectedRevision {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
UPDATE course.content_tool_states
SET
  state_json = $2::jsonb,
  revision = revision + 1,
  status = CASE WHEN status = 'not_started' THEN 'in_progress' ELSE status END,
  interaction_count = interaction_count + 1,
  first_interacted_at = COALESCE(first_interacted_at, NOW()),
  last_interacted_at = NOW(),
  updated_at = NOW()
WHERE id = $1 AND revision = $3
RETURNING `+stateCols+`
`, existing.ID, stateJSON, expectedRevision)
	return scanState(row)
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
