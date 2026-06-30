// Package adminaudit implements the admin audit log service: recording privileged-user actions
// and exposing a queryable compliance audit trail (plan 10.11).
package adminaudit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/lextures/lextures/server/internal/repos/adminaudit"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// ReadPermission gates access to the audit log viewer and export API (separate from admin).
const ReadPermission = "compliance:audit:read:*"

// Valid event types (FR-1).
const (
	EventRoleGrant            = "role_grant"
	EventRoleRevoke           = "role_revoke"
	EventGradeOverride        = "grade_override"
	EventGradeBulkImport      = "grade_bulk_import"
	EventDataExport           = "data_export"
	EventEnrollmentCreate     = "enrollment_create"
	EventEnrollmentDelete     = "enrollment_delete"
	EventCoursePublish        = "course_publish"
	EventCourseDelete         = "course_delete"
	EventAIConfigChange       = "ai_config_change"
	EventSecurityConfigChange = "security_config_change"
	EventUserImpersonation    = "user_impersonation"
	EventPasswordResetAdmin   = "password_reset_admin"
	EventContentDelete        = "content_delete"
	EventIncompleteGranted    = "incomplete_granted"
	EventIncompleteResolved   = "incomplete_resolved"
	EventUserCreate           = "user_create"
	EventUserUpdate           = "user_update"
	EventUserDeactivate       = "user_deactivate"
	EventCourseArchive        = "course_archive"
	EventOrgSettingsChange    = "org_settings_change"
	EventCustomFieldDefinitionChange = "custom_field_definition_change"
)

var ErrNotFound = errors.New("adminaudit: event not found")

// CheckReadAccess returns true when the user holds the compliance:audit:read permission.
func CheckReadAccess(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, ReadPermission)
}

// RecordParams is the input for recording one admin action.
type RecordParams struct {
	OrgID       *uuid.UUID
	EventType   string
	ActorID     uuid.UUID
	ActorIP     *string
	UserAgent   *string
	TargetType  *string
	TargetID    *uuid.UUID
	BeforeValue []byte
	AfterValue  []byte
}

// Record writes an audit event using the pool (own implicit transaction).
// The write is synchronous; callers must propagate any error (NFR Reliability).
func Record(ctx context.Context, pool *pgxpool.Pool, p RecordParams) (uuid.UUID, error) {
	id, _, err := repo.Insert(ctx, pool, repo.InsertParams{
		OrgID:       p.OrgID,
		EventType:   p.EventType,
		ActorID:     p.ActorID,
		ActorIP:     p.ActorIP,
		UserAgent:   p.UserAgent,
		TargetType:  p.TargetType,
		TargetID:    p.TargetID,
		BeforeValue: p.BeforeValue,
		AfterValue:  p.AfterValue,
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("adminaudit: record: %w", err)
	}
	return id, nil
}

// QueryParams holds optional filter parameters for ListEvents.
type QueryParams struct {
	OrgID     *uuid.UUID
	ActorID   *uuid.UUID
	EventType *string
	TargetID  *uuid.UUID
	From      time.Time
	To        time.Time
	Limit     int
}

// ListEvents returns audit events matching the optional filters (FR-6).
func ListEvents(ctx context.Context, pool *pgxpool.Pool, q QueryParams) ([]repo.Event, error) {
	if q.To.IsZero() {
		q.To = time.Now().UTC()
	}
	if q.From.IsZero() {
		q.From = q.To.AddDate(0, -1, 0)
	}
	events, err := repo.List(ctx, pool, repo.Query{
		OrgID:     q.OrgID,
		ActorID:   q.ActorID,
		EventType: q.EventType,
		TargetID:  q.TargetID,
		From:      q.From,
		To:        q.To,
		Limit:     q.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("adminaudit: list: %w", err)
	}
	return events, nil
}

// GetEvent returns a single audit event by ID, or ErrNotFound.
func GetEvent(ctx context.Context, pool *pgxpool.Pool, eventID uuid.UUID) (*repo.Event, error) {
	e, err := repo.GetByID(ctx, pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("adminaudit: get: %w", err)
	}
	if e == nil {
		return nil, ErrNotFound
	}
	return e, nil
}
