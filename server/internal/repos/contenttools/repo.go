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

// SettingsRow is a course.content_tool_settings row.
type SettingsRow struct {
	CourseID            uuid.UUID
	AllowedToolIDs      []string
	StudentResetAllowed bool
	MaxInstancesPerItem int16
	UpdatedBy           *uuid.UUID
	UpdatedAt           time.Time
}

// InstanceRow is a course.content_tool_instances row.
type InstanceRow struct {
	ID                  uuid.UUID
	CourseID            uuid.UUID
	StructureItemID     *uuid.UUID
	HostKind            string
	SectionKey          *string
	ToolID              string
	ToolVersion         string
	Title               *string
	ConfigJSON          json.RawMessage
	ConfigSchemaVersion int
	Status              string
	CreatedBy           *uuid.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// StateRow is a course.content_tool_states row.
type StateRow struct {
	ID                 uuid.UUID
	InstanceID         uuid.UUID
	EnrollmentID       uuid.UUID
	UserID             uuid.UUID
	Scope              string
	StateJSON          json.RawMessage
	StateSchemaVersion int
	Revision           int64
	Status             string
	ScoreRaw           *float64
	ScoreMax           *float64
	InteractionCount   int
	FirstInteractedAt  *time.Time
	LastInteractedAt   *time.Time
	CompletedAt        *time.Time
	ResetCount         int
	LastResetAt        *time.Time
	LastResetBy        *uuid.UUID
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

const (
	ScopeEnrollment = "enrollment"
	ScopePreview    = "preview"
)

// DefaultSettings returns in-memory defaults matching the migration.
func DefaultSettings(courseID uuid.UUID) SettingsRow {
	return SettingsRow{
		CourseID:            courseID,
		AllowedToolIDs:      []string{},
		StudentResetAllowed: false,
		MaxInstancesPerItem: 50,
		UpdatedAt:           time.Now().UTC(),
	}
}

// GetSettings returns settings for a course, or nil if no row exists.
func GetSettings(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*SettingsRow, error) {
	var r SettingsRow
	var updatedBy *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT course_id, allowed_tool_ids, student_reset_allowed, max_instances_per_item, updated_by, updated_at
FROM course.content_tool_settings
WHERE course_id = $1
`, courseID).Scan(
		&r.CourseID, &r.AllowedToolIDs, &r.StudentResetAllowed, &r.MaxInstancesPerItem, &updatedBy, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.UpdatedBy = updatedBy
	if r.AllowedToolIDs == nil {
		r.AllowedToolIDs = []string{}
	}
	return &r, nil
}

// UpsertSettings inserts or updates settings for a course.
func UpsertSettings(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, s SettingsRow, actorUserID uuid.UUID) (*SettingsRow, error) {
	maxInst := s.MaxInstancesPerItem
	if maxInst <= 0 {
		maxInst = 50
	}
	allowed := s.AllowedToolIDs
	if allowed == nil {
		allowed = []string{}
	}
	var r SettingsRow
	var updatedBy *uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_settings (
  course_id, allowed_tool_ids, student_reset_allowed, max_instances_per_item, updated_by, updated_at
) VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (course_id) DO UPDATE SET
  allowed_tool_ids = EXCLUDED.allowed_tool_ids,
  student_reset_allowed = EXCLUDED.student_reset_allowed,
  max_instances_per_item = EXCLUDED.max_instances_per_item,
  updated_by = EXCLUDED.updated_by,
  updated_at = NOW()
RETURNING course_id, allowed_tool_ids, student_reset_allowed, max_instances_per_item, updated_by, updated_at
`, courseID, allowed, s.StudentResetAllowed, maxInst, actorUserID).Scan(
		&r.CourseID, &r.AllowedToolIDs, &r.StudentResetAllowed, &r.MaxInstancesPerItem, &updatedBy, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.UpdatedBy = updatedBy
	if r.AllowedToolIDs == nil {
		r.AllowedToolIDs = []string{}
	}
	return &r, nil
}

// StructureItemInCourse reports whether the structure item belongs to the course.
func StructureItemInCourse(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM course.course_structure_items WHERE id = $1 AND course_id = $2
)`, itemID, courseID).Scan(&ok)
	return ok, err
}

// InsertEvent appends an event row.
func InsertEvent(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	instanceID *uuid.UUID,
	enrollmentID *uuid.UUID,
	actorUserID *uuid.UUID,
	toolID string,
	eventType string,
	payload map[string]any,
) error {
	if payload == nil {
		payload = map[string]any{}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO course.content_tool_events (instance_id, course_id, enrollment_id, actor_user_id, tool_id, event_type, payload_json)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
`, instanceID, courseID, enrollmentID, actorUserID, toolID, eventType, b)
	return err
}
