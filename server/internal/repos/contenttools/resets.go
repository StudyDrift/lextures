package contenttools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ResetScopeInstanceEnrollment = "instance_enrollment"
	ResetScopeInstanceAll        = "instance_all"
	ResetScopeItemEnrollment     = "item_enrollment"
	ResetScopeItemAll            = "item_all"
	ResetScopeCourseEnrollment   = "course_enrollment"
	ResetScopeSelf               = "self"

	DefaultRetentionDays = 90
	MinRetentionDays     = 7
)

// StateResetRow is a course.content_tool_state_resets snapshot.
type StateResetRow struct {
	ID             uuid.UUID
	InstanceID     uuid.UUID
	EnrollmentID   uuid.UUID
	CourseID       uuid.UUID
	ToolID         string
	Scope          string
	Reason         *string
	PriorStateJSON json.RawMessage
	PriorStatus    string
	PriorScoreRaw  *float64
	PriorScoreMax  *float64
	PriorRevision  int64
	BatchID        *uuid.UUID
	ResetBy        *uuid.UUID
	ResetAt        time.Time
	RestoredAt     *time.Time
	RestoredBy     *uuid.UUID
	PurgeAfter     time.Time
}

// AffectedState is a state row targeted by a reset (with display name).
type AffectedState struct {
	State        StateRow
	DisplayName  string
	UserID       uuid.UUID
	InstanceID   uuid.UUID
	ToolID       string
	EnrollmentID uuid.UUID
}

// OrgRetentionDays returns content_tool_state_retention_days for the course's org (floor 7).
func OrgRetentionDays(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (int, error) {
	var days int
	err := pool.QueryRow(ctx, `
SELECT COALESCE(o.content_tool_state_retention_days, $2)
FROM course.courses c
JOIN tenant.organizations o ON o.id = c.org_id
WHERE c.id = $1
`, courseID, DefaultRetentionDays).Scan(&days)
	if err != nil {
		return DefaultRetentionDays, err
	}
	if days < MinRetentionDays {
		days = MinRetentionDays
	}
	return days, nil
}

// ResolveAffectedStates returns enrollment-scoped states matching the reset scope.
// sectionIDs, when non-empty, narrows to those sections (TA scoping).
func ResolveAffectedStates(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	scope string,
	instanceID, itemID, enrollmentID *uuid.UUID,
	sectionIDs []uuid.UUID,
) ([]AffectedState, error) {
	var q string
	var args []any

	base := `
SELECT
  s.id, s.instance_id, s.enrollment_id, s.user_id, s.scope, s.state_json, s.state_schema_version,
  s.revision, s.status, s.score_raw, s.score_max, s.interaction_count,
  s.first_interacted_at, s.last_interacted_at, s.completed_at,
  s.reset_count, s.last_reset_at, s.last_reset_by, s.created_at, s.updated_at,
  COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) AS display_name,
  i.tool_id
FROM course.content_tool_states s
JOIN course.content_tool_instances i ON i.id = s.instance_id
JOIN course.course_enrollments ce ON ce.id = s.enrollment_id
JOIN "user".users u ON u.id = s.user_id
WHERE i.course_id = $1
  AND s.scope = 'enrollment'
  AND ce.active
  AND ce.role = 'student'
`
	args = []any{courseID}
	n := 2

	switch scope {
	case ResetScopeInstanceEnrollment, ResetScopeSelf:
		if instanceID == nil || enrollmentID == nil {
			return nil, fmt.Errorf("instanceId and enrollmentId required for scope %s", scope)
		}
		q = base + fmt.Sprintf(` AND s.instance_id = $%d AND s.enrollment_id = $%d`, n, n+1)
		args = append(args, *instanceID, *enrollmentID)
		n += 2
	case ResetScopeInstanceAll:
		if instanceID == nil {
			return nil, fmt.Errorf("instanceId required for scope %s", scope)
		}
		q = base + fmt.Sprintf(` AND s.instance_id = $%d`, n)
		args = append(args, *instanceID)
		n++
	case ResetScopeItemEnrollment:
		if itemID == nil || enrollmentID == nil {
			return nil, fmt.Errorf("itemId and enrollmentId required for scope %s", scope)
		}
		q = base + fmt.Sprintf(` AND i.structure_item_id = $%d AND s.enrollment_id = $%d AND i.status = 'active'`, n, n+1)
		args = append(args, *itemID, *enrollmentID)
		n += 2
	case ResetScopeItemAll:
		if itemID == nil {
			return nil, fmt.Errorf("itemId required for scope %s", scope)
		}
		q = base + fmt.Sprintf(` AND i.structure_item_id = $%d AND i.status = 'active'`, n)
		args = append(args, *itemID)
		n++
	case ResetScopeCourseEnrollment:
		if enrollmentID == nil {
			return nil, fmt.Errorf("enrollmentId required for scope %s", scope)
		}
		q = base + fmt.Sprintf(` AND s.enrollment_id = $%d AND i.status = 'active'`, n)
		args = append(args, *enrollmentID)
		n++
	default:
		return nil, fmt.Errorf("unknown reset scope %q", scope)
	}

	if len(sectionIDs) > 0 {
		q += fmt.Sprintf(` AND ce.section_id = ANY($%d::uuid[])`, n)
		args = append(args, sectionIDs)
	}
	q += ` ORDER BY display_name ASC, s.id ASC`

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AffectedState, 0)
	for rows.Next() {
		var st StateRow
		var state []byte
		var display string
		var toolID string
		if err := rows.Scan(
			&st.ID, &st.InstanceID, &st.EnrollmentID, &st.UserID, &st.Scope, &state, &st.StateSchemaVersion,
			&st.Revision, &st.Status, &st.ScoreRaw, &st.ScoreMax, &st.InteractionCount,
			&st.FirstInteractedAt, &st.LastInteractedAt, &st.CompletedAt,
			&st.ResetCount, &st.LastResetAt, &st.LastResetBy, &st.CreatedAt, &st.UpdatedAt,
			&display, &toolID,
		); err != nil {
			return nil, err
		}
		if len(state) == 0 {
			state = []byte(`{}`)
		}
		st.StateJSON = json.RawMessage(state)
		if st.Scope == "" {
			st.Scope = ScopeEnrollment
		}
		out = append(out, AffectedState{
			State:        st,
			DisplayName:  display,
			UserID:       st.UserID,
			InstanceID:   st.InstanceID,
			ToolID:       toolID,
			EnrollmentID: st.EnrollmentID,
		})
	}
	return out, rows.Err()
}

// InsertStateReset inserts one snapshot row.
func InsertStateReset(ctx context.Context, tx pgx.Tx, row StateResetRow) (*StateResetRow, error) {
	prior := row.PriorStateJSON
	if len(prior) == 0 {
		prior = json.RawMessage(`{}`)
	}
	var out StateResetRow
	var priorBytes []byte
	err := tx.QueryRow(ctx, `
INSERT INTO course.content_tool_state_resets (
  instance_id, enrollment_id, course_id, tool_id, scope, reason,
  prior_state_json, prior_status, prior_score_raw, prior_score_max, prior_revision,
  batch_id, reset_by, reset_at, purge_after
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7::jsonb,$8,$9,$10,$11,
  $12,$13,NOW(),$14
)
RETURNING id, instance_id, enrollment_id, course_id, tool_id, scope, reason,
  prior_state_json, prior_status, prior_score_raw, prior_score_max, prior_revision,
  batch_id, reset_by, reset_at, restored_at, restored_by, purge_after
`, row.InstanceID, row.EnrollmentID, row.CourseID, row.ToolID, row.Scope, row.Reason,
		[]byte(prior), row.PriorStatus, row.PriorScoreRaw, row.PriorScoreMax, row.PriorRevision,
		row.BatchID, row.ResetBy, row.PurgeAfter,
	).Scan(
		&out.ID, &out.InstanceID, &out.EnrollmentID, &out.CourseID, &out.ToolID, &out.Scope, &out.Reason,
		&priorBytes, &out.PriorStatus, &out.PriorScoreRaw, &out.PriorScoreMax, &out.PriorRevision,
		&out.BatchID, &out.ResetBy, &out.ResetAt, &out.RestoredAt, &out.RestoredBy, &out.PurgeAfter,
	)
	if err != nil {
		return nil, err
	}
	out.PriorStateJSON = json.RawMessage(priorBytes)
	return &out, nil
}

// ClearStateForReset sets state to initial, status not_started, nulls scores/timestamps, bumps revision/reset_count.
func ClearStateForReset(
	ctx context.Context,
	tx pgx.Tx,
	stateID uuid.UUID,
	initialJSON json.RawMessage,
	actor uuid.UUID,
) (*StateRow, error) {
	if len(initialJSON) == 0 {
		initialJSON = json.RawMessage(`{}`)
	}
	row := tx.QueryRow(ctx, `
UPDATE course.content_tool_states
SET state_json = $2::jsonb,
    status = 'not_started',
    score_raw = NULL,
    score_max = NULL,
    interaction_count = 0,
    first_interacted_at = NULL,
    last_interacted_at = NULL,
    completed_at = NULL,
    revision = revision + 1,
    reset_count = reset_count + 1,
    last_reset_at = NOW(),
    last_reset_by = $3,
    updated_at = NOW()
WHERE id = $1 AND scope = 'enrollment'
RETURNING `+stateCols+`
`, stateID, []byte(initialJSON), actor)
	return scanState(row)
}

// GetStateReset returns a snapshot by id.
func GetStateReset(ctx context.Context, pool *pgxpool.Pool, resetID uuid.UUID) (*StateResetRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, instance_id, enrollment_id, course_id, tool_id, scope, reason,
  prior_state_json, prior_status, prior_score_raw, prior_score_max, prior_revision,
  batch_id, reset_by, reset_at, restored_at, restored_by, purge_after
FROM course.content_tool_state_resets
WHERE id = $1
`, resetID)
	return scanStateReset(row)
}

func scanStateReset(row pgx.Row) (*StateResetRow, error) {
	var r StateResetRow
	var prior []byte
	err := row.Scan(
		&r.ID, &r.InstanceID, &r.EnrollmentID, &r.CourseID, &r.ToolID, &r.Scope, &r.Reason,
		&prior, &r.PriorStatus, &r.PriorScoreRaw, &r.PriorScoreMax, &r.PriorRevision,
		&r.BatchID, &r.ResetBy, &r.ResetAt, &r.RestoredAt, &r.RestoredBy, &r.PurgeAfter,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(prior) == 0 {
		prior = []byte(`{}`)
	}
	r.PriorStateJSON = json.RawMessage(prior)
	return &r, nil
}

// ListStateResets lists snapshots for an instance and/or enrollment.
func ListStateResets(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, instanceID, enrollmentID *uuid.UUID, limit int) ([]StateResetRow, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	args := []any{courseID}
	where := `course_id = $1`
	n := 2
	if instanceID != nil {
		where += fmt.Sprintf(` AND instance_id = $%d`, n)
		args = append(args, *instanceID)
		n++
	}
	if enrollmentID != nil {
		where += fmt.Sprintf(` AND enrollment_id = $%d`, n)
		args = append(args, *enrollmentID)
		n++
	}
	args = append(args, limit)
	rows, err := pool.Query(ctx, fmt.Sprintf(`
SELECT id, instance_id, enrollment_id, course_id, tool_id, scope, reason,
  prior_state_json, prior_status, prior_score_raw, prior_score_max, prior_revision,
  batch_id, reset_by, reset_at, restored_at, restored_by, purge_after
FROM course.content_tool_state_resets
WHERE %s
ORDER BY reset_at DESC
LIMIT $%d
`, where, n), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]StateResetRow, 0)
	for rows.Next() {
		var r StateResetRow
		var prior []byte
		if err := rows.Scan(
			&r.ID, &r.InstanceID, &r.EnrollmentID, &r.CourseID, &r.ToolID, &r.Scope, &r.Reason,
			&prior, &r.PriorStatus, &r.PriorScoreRaw, &r.PriorScoreMax, &r.PriorRevision,
			&r.BatchID, &r.ResetBy, &r.ResetAt, &r.RestoredAt, &r.RestoredBy, &r.PurgeAfter,
		); err != nil {
			return nil, err
		}
		if len(prior) == 0 {
			prior = []byte(`{}`)
		}
		r.PriorStateJSON = json.RawMessage(prior)
		out = append(out, r)
	}
	return out, rows.Err()
}

// RestoreStateFromReset reinstates snapshotted document/status/score and marks the snapshot restored.
func RestoreStateFromReset(ctx context.Context, pool *pgxpool.Pool, resetID, actor uuid.UUID) (*StateRow, *StateResetRow, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
SELECT id, instance_id, enrollment_id, course_id, tool_id, scope, reason,
  prior_state_json, prior_status, prior_score_raw, prior_score_max, prior_revision,
  batch_id, reset_by, reset_at, restored_at, restored_by, purge_after
FROM course.content_tool_state_resets
WHERE id = $1
FOR UPDATE
`, resetID)
	snap, err := scanStateReset(row)
	if err != nil {
		return nil, nil, err
	}
	if snap == nil {
		return nil, nil, nil
	}
	if snap.RestoredAt != nil {
		return nil, nil, fmt.Errorf("already_restored")
	}
	if time.Now().UTC().After(snap.PurgeAfter) {
		return nil, nil, fmt.Errorf("expired")
	}

	prior := snap.PriorStateJSON
	if len(prior) == 0 {
		prior = json.RawMessage(`{}`)
	}
	stRow := tx.QueryRow(ctx, `
UPDATE course.content_tool_states
SET state_json = $3::jsonb,
    status = $4,
    score_raw = $5,
    score_max = $6,
    revision = revision + 1,
    updated_at = NOW()
WHERE instance_id = $1 AND enrollment_id = $2 AND scope = 'enrollment'
RETURNING `+stateCols+`
`, snap.InstanceID, snap.EnrollmentID, []byte(prior), snap.PriorStatus, snap.PriorScoreRaw, snap.PriorScoreMax)
	st, err := scanState(stRow)
	if err != nil {
		return nil, nil, err
	}
	if st == nil {
		// Recreate state row if it was deleted.
		stRow = tx.QueryRow(ctx, `
INSERT INTO course.content_tool_states (
  instance_id, enrollment_id, user_id, scope, state_json, revision, status,
  score_raw, score_max, interaction_count
)
SELECT $1, $2, ce.user_id, 'enrollment', $3::jsonb, $4, $5, $6, $7, 0
FROM course.course_enrollments ce WHERE ce.id = $2
RETURNING `+stateCols+`
`, snap.InstanceID, snap.EnrollmentID, []byte(prior), snap.PriorRevision+1, snap.PriorStatus, snap.PriorScoreRaw, snap.PriorScoreMax)
		st, err = scanState(stRow)
		if err != nil {
			return nil, nil, err
		}
	}

	var priorBytes []byte
	err = tx.QueryRow(ctx, `
UPDATE course.content_tool_state_resets
SET restored_at = NOW(), restored_by = $2
WHERE id = $1
RETURNING id, instance_id, enrollment_id, course_id, tool_id, scope, reason,
  prior_state_json, prior_status, prior_score_raw, prior_score_max, prior_revision,
  batch_id, reset_by, reset_at, restored_at, restored_by, purge_after
`, resetID, actor).Scan(
		&snap.ID, &snap.InstanceID, &snap.EnrollmentID, &snap.CourseID, &snap.ToolID, &snap.Scope, &snap.Reason,
		&priorBytes, &snap.PriorStatus, &snap.PriorScoreRaw, &snap.PriorScoreMax, &snap.PriorRevision,
		&snap.BatchID, &snap.ResetBy, &snap.ResetAt, &snap.RestoredAt, &snap.RestoredBy, &snap.PurgeAfter,
	)
	if err != nil {
		return nil, nil, err
	}
	snap.PriorStateJSON = json.RawMessage(priorBytes)
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return st, snap, nil
}

// PurgeExpiredStateResets deletes snapshots past purge_after.
func PurgeExpiredStateResets(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM course.content_tool_state_resets
WHERE purge_after < NOW()
`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
