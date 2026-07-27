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

// ListStatesForEnrollment returns enrollment-scoped states for the given instances.
func ListStatesForEnrollment(
	ctx context.Context,
	pool *pgxpool.Pool,
	enrollmentID uuid.UUID,
	instanceIDs []uuid.UUID,
) (map[uuid.UUID]*StateRow, error) {
	out := make(map[uuid.UUID]*StateRow, len(instanceIDs))
	if len(instanceIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
SELECT `+stateCols+`
FROM course.content_tool_states
WHERE enrollment_id = $1
  AND scope = 'enrollment'
  AND instance_id = ANY($2::uuid[])
`, enrollmentID, instanceIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		st, err := scanState(rows)
		if err != nil {
			return nil, err
		}
		if st != nil {
			out[st.InstanceID] = st
		}
	}
	return out, rows.Err()
}

// UpsertStateWithStatus is like UpsertState but also applies a target status when provided.
// expectedRevision is the client's known revision; mismatch returns nil,nil for 409.
// stateSchemaVersion: when > 0, persisted on insert/update (CT.5 lazy migration); 0 keeps default/existing.
func UpsertStateWithStatus(
	ctx context.Context,
	pool *pgxpool.Pool,
	instanceID, enrollmentID, userID uuid.UUID,
	stateJSON json.RawMessage,
	expectedRevision int64,
	status string,
	stateSchemaVersion int,
) (*StateRow, error) {
	if len(stateJSON) == 0 {
		stateJSON = json.RawMessage(`{}`)
	}
	if status == "" {
		status = "in_progress"
	}
	row := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_states (
  instance_id, enrollment_id, user_id, scope, state_json, state_schema_version, revision, status,
  interaction_count, first_interacted_at, last_interacted_at,
  completed_at
) VALUES (
  $1, $2, $3, 'enrollment', $4::jsonb, CASE WHEN $7 > 0 THEN $7 ELSE 1 END, 1, $6,
  1, NOW(), NOW(),
  CASE WHEN $6 IN ('submitted', 'completed') THEN NOW() ELSE NULL END
)
ON CONFLICT (instance_id, enrollment_id) WHERE (scope = 'enrollment') DO UPDATE SET
  state_json = EXCLUDED.state_json,
  state_schema_version = CASE
    WHEN $7 > 0 THEN $7
    ELSE course.content_tool_states.state_schema_version
  END,
  revision = course.content_tool_states.revision + 1,
  status = EXCLUDED.status,
  interaction_count = course.content_tool_states.interaction_count + 1,
  first_interacted_at = COALESCE(course.content_tool_states.first_interacted_at, NOW()),
  last_interacted_at = NOW(),
  completed_at = CASE
    WHEN EXCLUDED.status IN ('submitted', 'completed')
      THEN COALESCE(course.content_tool_states.completed_at, NOW())
    ELSE course.content_tool_states.completed_at
  END,
  updated_at = NOW()
WHERE course.content_tool_states.revision = $5
RETURNING `+stateCols+`
`, instanceID, enrollmentID, userID, stateJSON, expectedRevision, status, stateSchemaVersion)
	return scanState(row)
}

// UpsertPreviewStateWithStatus mirrors UpsertPreviewState with status control.
func UpsertPreviewStateWithStatus(
	ctx context.Context,
	pool *pgxpool.Pool,
	instanceID, enrollmentID, userID uuid.UUID,
	stateJSON json.RawMessage,
	expectedRevision int64,
	status string,
	stateSchemaVersion int,
) (*StateRow, error) {
	if len(stateJSON) == 0 {
		stateJSON = json.RawMessage(`{}`)
	}
	if status == "" {
		status = "in_progress"
	}
	existing, err := GetStateByScope(ctx, pool, instanceID, enrollmentID, ScopePreview)
	if err != nil {
		return nil, err
	}
	schemaVer := 1
	if stateSchemaVersion > 0 {
		schemaVer = stateSchemaVersion
	}
	if existing == nil {
		row := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_states (
  instance_id, enrollment_id, user_id, scope, state_json, state_schema_version, revision, status,
  interaction_count, first_interacted_at, last_interacted_at, completed_at
) VALUES (
  $1, $2, $3, 'preview', $4::jsonb, $6, 1, $5,
  1, NOW(), NOW(),
  CASE WHEN $5 IN ('submitted', 'completed') THEN NOW() ELSE NULL END
)
RETURNING `+stateCols+`
`, instanceID, enrollmentID, userID, stateJSON, status, schemaVer)
		return scanState(row)
	}
	if existing.Revision != expectedRevision {
		return nil, nil
	}
	keepSchema := existing.StateSchemaVersion
	if stateSchemaVersion > 0 {
		keepSchema = stateSchemaVersion
	}
	row := pool.QueryRow(ctx, `
UPDATE course.content_tool_states
SET
  state_json = $2::jsonb,
  state_schema_version = $5,
  revision = revision + 1,
  status = $4,
  interaction_count = interaction_count + 1,
  first_interacted_at = COALESCE(first_interacted_at, NOW()),
  last_interacted_at = NOW(),
  completed_at = CASE
    WHEN $4 IN ('submitted', 'completed') THEN COALESCE(completed_at, NOW())
    ELSE completed_at
  END,
  updated_at = NOW()
WHERE id = $1 AND revision = $3
RETURNING `+stateCols+`
`, existing.ID, stateJSON, expectedRevision, status, keepSchema)
	return scanState(row)
}

// ApplyActionState writes state/status/score atomically after a server action.
// expectedRevision is the revision the action read; mismatch returns nil,nil.
func ApplyActionState(
	ctx context.Context,
	pool *pgxpool.Pool,
	instanceID, enrollmentID, userID uuid.UUID,
	stateJSON json.RawMessage,
	expectedRevision int64,
	status string,
	scoreRaw, scoreMax *float64,
	scope string,
) (*StateRow, error) {
	if scope == "" {
		scope = ScopeEnrollment
	}
	if len(stateJSON) == 0 {
		stateJSON = json.RawMessage(`{}`)
	}
	if status == "" {
		status = "in_progress"
	}
	if scope == ScopePreview {
		existing, err := GetStateByScope(ctx, pool, instanceID, enrollmentID, ScopePreview)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			row := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_states (
  instance_id, enrollment_id, user_id, scope, state_json, revision, status,
  score_raw, score_max, interaction_count, first_interacted_at, last_interacted_at, completed_at
) VALUES (
  $1, $2, $3, 'preview', $4::jsonb, 1, $5,
  $6, $7, 1, NOW(), NOW(),
  CASE WHEN $5 IN ('submitted', 'completed') THEN NOW() ELSE NULL END
)
RETURNING `+stateCols+`
`, instanceID, enrollmentID, userID, stateJSON, status, scoreRaw, scoreMax)
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
  status = $4,
  score_raw = $5,
  score_max = $6,
  interaction_count = interaction_count + 1,
  first_interacted_at = COALESCE(first_interacted_at, NOW()),
  last_interacted_at = NOW(),
  completed_at = CASE
    WHEN $4 IN ('submitted', 'completed') THEN COALESCE(completed_at, NOW())
    ELSE completed_at
  END,
  updated_at = NOW()
WHERE id = $1 AND revision = $3
RETURNING `+stateCols+`
`, existing.ID, stateJSON, expectedRevision, status, scoreRaw, scoreMax)
		return scanState(row)
	}

	row := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_states (
  instance_id, enrollment_id, user_id, scope, state_json, revision, status,
  score_raw, score_max, interaction_count, first_interacted_at, last_interacted_at, completed_at
) VALUES (
  $1, $2, $3, 'enrollment', $4::jsonb, 1, $5,
  $6, $7, 1, NOW(), NOW(),
  CASE WHEN $5 IN ('submitted', 'completed') THEN NOW() ELSE NULL END
)
ON CONFLICT (instance_id, enrollment_id) WHERE (scope = 'enrollment') DO UPDATE SET
  state_json = EXCLUDED.state_json,
  revision = course.content_tool_states.revision + 1,
  status = EXCLUDED.status,
  score_raw = EXCLUDED.score_raw,
  score_max = EXCLUDED.score_max,
  interaction_count = course.content_tool_states.interaction_count + 1,
  first_interacted_at = COALESCE(course.content_tool_states.first_interacted_at, NOW()),
  last_interacted_at = NOW(),
  completed_at = CASE
    WHEN EXCLUDED.status IN ('submitted', 'completed')
      THEN COALESCE(course.content_tool_states.completed_at, NOW())
    ELSE course.content_tool_states.completed_at
  END,
  updated_at = NOW()
WHERE course.content_tool_states.revision = $8
RETURNING `+stateCols+`
`, instanceID, enrollmentID, userID, stateJSON, status, scoreRaw, scoreMax, expectedRevision)
	return scanState(row)
}

// ActionIdempotencyRow is a cached action result.
type ActionIdempotencyRow struct {
	Key          string
	InstanceID   uuid.UUID
	EnrollmentID uuid.UUID
	Action       string
	ResultJSON   json.RawMessage
	CreatedAt    time.Time
}

// GetActionIdempotency returns a prior result for the key, or nil.
func GetActionIdempotency(ctx context.Context, pool *pgxpool.Pool, key string) (*ActionIdempotencyRow, error) {
	var r ActionIdempotencyRow
	var raw []byte
	err := pool.QueryRow(ctx, `
SELECT idempotency_key, instance_id, enrollment_id, action, result_json, created_at
FROM course.content_tool_action_idempotency
WHERE idempotency_key = $1
`, key).Scan(&r.Key, &r.InstanceID, &r.EnrollmentID, &r.Action, &raw, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.ResultJSON = json.RawMessage(raw)
	return &r, nil
}

// PutActionIdempotency stores an action result. Conflicts are ignored (first wins).
func PutActionIdempotency(
	ctx context.Context,
	pool *pgxpool.Pool,
	key string,
	instanceID, enrollmentID uuid.UUID,
	action string,
	resultJSON json.RawMessage,
) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.content_tool_action_idempotency (
  idempotency_key, instance_id, enrollment_id, action, result_json
) VALUES ($1, $2, $3, $4, $5::jsonb)
ON CONFLICT (idempotency_key) DO NOTHING
`, key, instanceID, enrollmentID, action, resultJSON)
	return err
}

// PurgeStaleActionIdempotency deletes idempotency rows older than olderThan (default 24h).
func PurgeStaleActionIdempotency(ctx context.Context, pool *pgxpool.Pool, olderThan time.Duration) (int64, error) {
	if olderThan <= 0 {
		olderThan = 24 * time.Hour
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	tag, err := pool.Exec(ctx, `
DELETE FROM course.content_tool_action_idempotency
WHERE created_at < $1
`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
