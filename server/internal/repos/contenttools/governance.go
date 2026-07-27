package contenttools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PolicyRow is tenant.content_tool_policies.
type PolicyRow struct {
	OrgID                   uuid.UUID
	DeniedCapabilities      []string
	DeniedToolIDs           []string
	AllowedToolIDs          []string
	AIDisclosureMode        string
	FreeTextFilterAction    string
	CrisisEscalationEnabled bool
	AILogRetentionDays      int
	UpdatedBy               *uuid.UUID
	UpdatedAt               time.Time
}

// DefaultPolicy returns permissive-but-safe defaults (CT.8 rollout).
func DefaultPolicy(orgID uuid.UUID) PolicyRow {
	return PolicyRow{
		OrgID:                   orgID,
		DeniedCapabilities:      []string{},
		DeniedToolIDs:           []string{},
		AllowedToolIDs:          []string{},
		AIDisclosureMode:        "banner",
		FreeTextFilterAction:    "flag",
		CrisisEscalationEnabled: true,
		AILogRetentionDays:      30,
		UpdatedAt:               time.Now().UTC(),
	}
}

// GetPolicy loads org policy or nil when unset.
func GetPolicy(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*PolicyRow, error) {
	if pool == nil {
		return nil, nil
	}
	row := PolicyRow{OrgID: orgID}
	var deniedCaps, deniedTools, allowedTools []string
	var updatedBy *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT org_id, denied_capabilities, denied_tool_ids, allowed_tool_ids,
       ai_disclosure_mode, free_text_filter_action, crisis_escalation_enabled,
       ai_log_retention_days, updated_by, updated_at
FROM tenant.content_tool_policies WHERE org_id = $1
`, orgID).Scan(
		&row.OrgID, &deniedCaps, &deniedTools, &allowedTools,
		&row.AIDisclosureMode, &row.FreeTextFilterAction, &row.CrisisEscalationEnabled,
		&row.AILogRetentionDays, &updatedBy, &row.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.DeniedCapabilities = nonNilStrings(deniedCaps)
	row.DeniedToolIDs = nonNilStrings(deniedTools)
	row.AllowedToolIDs = nonNilStrings(allowedTools)
	row.UpdatedBy = updatedBy
	return &row, nil
}

// UpsertPolicy writes org policy.
func UpsertPolicy(ctx context.Context, pool *pgxpool.Pool, p PolicyRow) (*PolicyRow, error) {
	if pool == nil {
		return nil, nil
	}
	if p.DeniedCapabilities == nil {
		p.DeniedCapabilities = []string{}
	}
	if p.DeniedToolIDs == nil {
		p.DeniedToolIDs = []string{}
	}
	if p.AllowedToolIDs == nil {
		p.AllowedToolIDs = []string{}
	}
	if p.AIDisclosureMode == "" {
		p.AIDisclosureMode = "banner"
	}
	if p.FreeTextFilterAction == "" {
		p.FreeTextFilterAction = "flag"
	}
	if p.AILogRetentionDays <= 0 {
		p.AILogRetentionDays = 30
	}
	err := pool.QueryRow(ctx, `
INSERT INTO tenant.content_tool_policies (
  org_id, denied_capabilities, denied_tool_ids, allowed_tool_ids,
  ai_disclosure_mode, free_text_filter_action, crisis_escalation_enabled,
  ai_log_retention_days, updated_by, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
ON CONFLICT (org_id) DO UPDATE SET
  denied_capabilities = EXCLUDED.denied_capabilities,
  denied_tool_ids = EXCLUDED.denied_tool_ids,
  allowed_tool_ids = EXCLUDED.allowed_tool_ids,
  ai_disclosure_mode = EXCLUDED.ai_disclosure_mode,
  free_text_filter_action = EXCLUDED.free_text_filter_action,
  crisis_escalation_enabled = EXCLUDED.crisis_escalation_enabled,
  ai_log_retention_days = EXCLUDED.ai_log_retention_days,
  updated_by = EXCLUDED.updated_by,
  updated_at = NOW()
RETURNING org_id, denied_capabilities, denied_tool_ids, allowed_tool_ids,
          ai_disclosure_mode, free_text_filter_action, crisis_escalation_enabled,
          ai_log_retention_days, updated_by, updated_at
`, p.OrgID, p.DeniedCapabilities, p.DeniedToolIDs, p.AllowedToolIDs,
		p.AIDisclosureMode, p.FreeTextFilterAction, p.CrisisEscalationEnabled,
		p.AILogRetentionDays, p.UpdatedBy,
	).Scan(
		&p.OrgID, &p.DeniedCapabilities, &p.DeniedToolIDs, &p.AllowedToolIDs,
		&p.AIDisclosureMode, &p.FreeTextFilterAction, &p.CrisisEscalationEnabled,
		&p.AILogRetentionDays, &p.UpdatedBy, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.DeniedCapabilities = nonNilStrings(p.DeniedCapabilities)
	p.DeniedToolIDs = nonNilStrings(p.DeniedToolIDs)
	p.AllowedToolIDs = nonNilStrings(p.AllowedToolIDs)
	return &p, nil
}

// ModerationRow is course.content_tool_moderation.
type ModerationRow struct {
	ID            uuid.UUID
	InstanceID    uuid.UUID
	StateID       *uuid.UUID
	ContentPath   *string
	Action        string
	Category      *string
	Reason        *string
	ActorUserID   *uuid.UUID
	SubjectUserID *uuid.UUID
	CreatedAt     time.Time
}

// InsertModeration appends a moderation action.
func InsertModeration(ctx context.Context, pool *pgxpool.Pool, m ModerationRow) (*ModerationRow, error) {
	if pool == nil {
		return nil, nil
	}
	err := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_moderation (
  instance_id, state_id, content_path, action, category, reason, actor_user_id, subject_user_id
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING id, instance_id, state_id, content_path, action, category, reason, actor_user_id, subject_user_id, created_at
`, m.InstanceID, m.StateID, m.ContentPath, m.Action, m.Category, m.Reason, m.ActorUserID, m.SubjectUserID).Scan(
		&m.ID, &m.InstanceID, &m.StateID, &m.ContentPath, &m.Action, &m.Category, &m.Reason,
		&m.ActorUserID, &m.SubjectUserID, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ListModeration returns moderation history for an instance (newest first).
func ListModeration(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID, limit int) ([]ModerationRow, error) {
	if pool == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT id, instance_id, state_id, content_path, action, category, reason, actor_user_id, subject_user_id, created_at
FROM course.content_tool_moderation
WHERE instance_id = $1
ORDER BY created_at DESC
LIMIT $2
`, instanceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ModerationRow, 0)
	for rows.Next() {
		var m ModerationRow
		if err := rows.Scan(&m.ID, &m.InstanceID, &m.StateID, &m.ContentPath, &m.Action, &m.Category,
			&m.Reason, &m.ActorUserID, &m.SubjectUserID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// LatestContentAction returns the latest hide/remove/restore action for a content path.
func LatestContentAction(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID, contentPath string) (string, error) {
	if pool == nil {
		return "", nil
	}
	var action string
	err := pool.QueryRow(ctx, `
SELECT action FROM course.content_tool_moderation
WHERE instance_id = $1 AND COALESCE(content_path, '') = $2
  AND action IN ('hidden','removed','restored')
ORDER BY created_at DESC
LIMIT 1
`, instanceID, contentPath).Scan(&action)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return action, err
}

// AIConsentRow is course.content_tool_ai_consents.
type AIConsentRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CourseID  *uuid.UUID
	ToolID    *string
	Decision  string
	DecidedAt time.Time
}

// UpsertAIConsent records acknowledgement or opt-out.
func UpsertAIConsent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseID *uuid.UUID, toolID *string, decision string) (*AIConsentRow, error) {
	if pool == nil {
		return nil, nil
	}
	var row AIConsentRow
	err := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_ai_consents (user_id, course_id, tool_id, decision, decided_at)
VALUES ($1,$2,$3,$4,NOW())
ON CONFLICT (user_id, course_id, tool_id) DO UPDATE SET
  decision = EXCLUDED.decision,
  decided_at = NOW()
RETURNING id, user_id, course_id, tool_id, decision, decided_at
`, userID, courseID, toolID, decision).Scan(
		&row.ID, &row.UserID, &row.CourseID, &row.ToolID, &row.Decision, &row.DecidedAt,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetAIConsent loads the latest consent for user/course/tool.
func GetAIConsent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseID *uuid.UUID, toolID *string) (*AIConsentRow, error) {
	if pool == nil {
		return nil, nil
	}
	var row AIConsentRow
	err := pool.QueryRow(ctx, `
SELECT id, user_id, course_id, tool_id, decision, decided_at
FROM course.content_tool_ai_consents
WHERE user_id = $1
  AND course_id IS NOT DISTINCT FROM $2
  AND tool_id IS NOT DISTINCT FROM $3
`, userID, courseID, toolID).Scan(
		&row.ID, &row.UserID, &row.CourseID, &row.ToolID, &row.Decision, &row.DecidedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// DataSheetRow is course.content_tool_data_sheets.
type DataSheetRow struct {
	ToolID             string
	Version            string
	CollectsJSON       json.RawMessage
	LeavesPlatform     bool
	Processors         []string
	Visibility         string
	WCAGLevel          string
	A11yLimitations    *string
	AITransparencyJSON json.RawMessage
	UpdatedAt          time.Time
}

// UpsertDataSheet mirrors a registry data sheet.
func UpsertDataSheet(ctx context.Context, pool *pgxpool.Pool, row DataSheetRow) error {
	if pool == nil {
		return nil
	}
	if row.Processors == nil {
		row.Processors = []string{}
	}
	if len(row.CollectsJSON) == 0 {
		row.CollectsJSON = []byte("{}")
	}
	if len(row.AITransparencyJSON) == 0 {
		row.AITransparencyJSON = []byte("{}")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.content_tool_data_sheets (
  tool_id, version, collects_json, leaves_platform, processors, visibility,
  wcag_level, a11y_limitations, ai_transparency_json, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
ON CONFLICT (tool_id) DO UPDATE SET
  version = EXCLUDED.version,
  collects_json = EXCLUDED.collects_json,
  leaves_platform = EXCLUDED.leaves_platform,
  processors = EXCLUDED.processors,
  visibility = EXCLUDED.visibility,
  wcag_level = EXCLUDED.wcag_level,
  a11y_limitations = EXCLUDED.a11y_limitations,
  ai_transparency_json = EXCLUDED.ai_transparency_json,
  updated_at = NOW()
`, row.ToolID, row.Version, row.CollectsJSON, row.LeavesPlatform, row.Processors,
		row.Visibility, row.WCAGLevel, row.A11yLimitations, row.AITransparencyJSON)
	return err
}

// ListDataSheets returns all mirrored data sheets.
func ListDataSheets(ctx context.Context, pool *pgxpool.Pool) ([]DataSheetRow, error) {
	if pool == nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT tool_id, version, collects_json, leaves_platform, processors, visibility,
       wcag_level, a11y_limitations, ai_transparency_json, updated_at
FROM course.content_tool_data_sheets
ORDER BY tool_id
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DataSheetRow, 0)
	for rows.Next() {
		var r DataSheetRow
		if err := rows.Scan(&r.ToolID, &r.Version, &r.CollectsJSON, &r.LeavesPlatform, &r.Processors,
			&r.Visibility, &r.WCAGLevel, &r.A11yLimitations, &r.AITransparencyJSON, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.Processors = nonNilStrings(r.Processors)
		out = append(out, r)
	}
	return out, rows.Err()
}

// KillRow is settings.content_tool_kills.
type KillRow struct {
	ID        uuid.UUID
	Scope     string
	Target    string
	Engaged   bool
	Reason    *string
	UpdatedBy *uuid.UUID
	UpdatedAt time.Time
}

// UpsertKill engages or clears a kill path entry for scope+target.
func UpsertKill(ctx context.Context, pool *pgxpool.Pool, scope, target string, engaged bool, reason *string, updatedBy *uuid.UUID) (*KillRow, error) {
	if pool == nil {
		return nil, nil
	}
	// Clear any currently engaged row for this scope/target first (partial unique index).
	_, err := pool.Exec(ctx, `
UPDATE settings.content_tool_kills
SET engaged = FALSE, updated_at = NOW(), updated_by = $3
WHERE scope = $1 AND target = $2 AND engaged = TRUE
`, scope, target, updatedBy)
	if err != nil {
		return nil, err
	}
	var row KillRow
	err = pool.QueryRow(ctx, `
INSERT INTO settings.content_tool_kills (scope, target, engaged, reason, updated_by, updated_at)
VALUES ($1,$2,$3,$4,$5,NOW())
RETURNING id, scope, target, engaged, reason, updated_by, updated_at
`, scope, target, engaged, reason, updatedBy).Scan(
		&row.ID, &row.Scope, &row.Target, &row.Engaged, &row.Reason, &row.UpdatedBy, &row.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListActiveKills returns currently engaged kill paths.
func ListActiveKills(ctx context.Context, pool *pgxpool.Pool) ([]KillRow, error) {
	if pool == nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT id, scope, target, engaged, reason, updated_by, updated_at
FROM settings.content_tool_kills
WHERE engaged = TRUE
ORDER BY updated_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]KillRow, 0)
	for rows.Next() {
		var r KillRow
		if err := rows.Scan(&r.ID, &r.Scope, &r.Target, &r.Engaged, &r.Reason, &r.UpdatedBy, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// InsertFilterFlag stores an aggregate content-filter hit (no raw text).
func InsertFilterFlag(ctx context.Context, pool *pgxpool.Pool, instanceID, courseID uuid.UUID, userID *uuid.UUID, category, action string) error {
	if pool == nil {
		return nil
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.content_tool_filter_flags (instance_id, course_id, user_id, category, action)
VALUES ($1,$2,$3,$4,$5)
`, instanceID, courseID, userID, category, action)
	return err
}

// ListFilterFlags returns recent aggregate flags for an instance.
func ListFilterFlags(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID, limit int) ([]map[string]any, error) {
	if pool == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT id, category, action, user_id, created_at
FROM course.content_tool_filter_flags
WHERE instance_id = $1
ORDER BY created_at DESC
LIMIT $2
`, instanceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var id uuid.UUID
		var category, action string
		var userID *uuid.UUID
		var createdAt time.Time
		if err := rows.Scan(&id, &category, &action, &userID, &createdAt); err != nil {
			return nil, err
		}
		item := map[string]any{
			"id":        id.String(),
			"category":  category,
			"action":    action,
			"createdAt": createdAt.UTC().Format(time.RFC3339),
		}
		if userID != nil {
			item["userId"] = userID.String()
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
