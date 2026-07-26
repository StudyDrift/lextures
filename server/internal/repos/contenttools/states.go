package contenttools

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func scanState(row pgx.Row) (*StateRow, error) {
	var r StateRow
	var state []byte
	err := row.Scan(
		&r.ID, &r.InstanceID, &r.EnrollmentID, &r.UserID, &state, &r.StateSchemaVersion,
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
	return &r, nil
}

// GetState returns state for (instance, enrollment), or nil.
func GetState(ctx context.Context, pool *pgxpool.Pool, instanceID, enrollmentID uuid.UUID) (*StateRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, instance_id, enrollment_id, user_id, state_json, state_schema_version,
       revision, status, score_raw, score_max, interaction_count,
       first_interacted_at, last_interacted_at, completed_at,
       reset_count, last_reset_at, last_reset_by, created_at, updated_at
FROM course.content_tool_states
WHERE instance_id = $1 AND enrollment_id = $2
`, instanceID, enrollmentID)
	return scanState(row)
}

// UpsertState inserts or updates state with optimistic concurrency on revision.
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
  instance_id, enrollment_id, user_id, state_json, revision, status,
  interaction_count, first_interacted_at, last_interacted_at
) VALUES (
  $1, $2, $3, $4::jsonb, 1, 'in_progress',
  1, NOW(), NOW()
)
ON CONFLICT (instance_id, enrollment_id) DO UPDATE SET
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
RETURNING id, instance_id, enrollment_id, user_id, state_json, state_schema_version,
          revision, status, score_raw, score_max, interaction_count,
          first_interacted_at, last_interacted_at, completed_at,
          reset_count, last_reset_at, last_reset_by, created_at, updated_at
`, instanceID, enrollmentID, userID, stateJSON, expectedRevision)
	return scanState(row)
}
