package contenttools

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VersionRow is a course.content_tool_versions row.
type VersionRow struct {
	ToolID              string
	Version             string
	ManifestJSON        json.RawMessage
	ConfigSchemaVersion int
	StateSchemaVersion  int
	SandboxMode         string
	Status              string
	BreakerOpenAt       *time.Time
	SunsetAt            *time.Time
	FirstSeenAt         time.Time
}

// UpsertToolVersion inserts or updates a registry mirror row.
func UpsertToolVersion(ctx context.Context, pool *pgxpool.Pool, row VersionRow) (*VersionRow, error) {
	if row.SandboxMode == "" {
		row.SandboxMode = "inprocess"
	}
	if row.Status == "" {
		row.Status = "active"
	}
	if row.ConfigSchemaVersion <= 0 {
		row.ConfigSchemaVersion = 1
	}
	if row.StateSchemaVersion <= 0 {
		row.StateSchemaVersion = 1
	}
	if len(row.ManifestJSON) == 0 {
		row.ManifestJSON = json.RawMessage(`{}`)
	}
	var out VersionRow
	var breaker, sunset *time.Time
	err := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_versions (
  tool_id, version, manifest_json, config_schema_version, state_schema_version,
  sandbox_mode, status, breaker_open_at, sunset_at, first_seen_at
) VALUES (
  $1, $2, $3::jsonb, $4, $5, $6, $7, $8, $9, NOW()
)
ON CONFLICT (tool_id, version) DO UPDATE SET
  manifest_json = EXCLUDED.manifest_json,
  config_schema_version = EXCLUDED.config_schema_version,
  state_schema_version = EXCLUDED.state_schema_version,
  sandbox_mode = EXCLUDED.sandbox_mode,
  -- Preserve admin status / breaker unless still default active with nil breaker.
  status = course.content_tool_versions.status,
  breaker_open_at = course.content_tool_versions.breaker_open_at,
  sunset_at = COALESCE(EXCLUDED.sunset_at, course.content_tool_versions.sunset_at)
RETURNING tool_id, version, manifest_json, config_schema_version, state_schema_version,
  sandbox_mode, status, breaker_open_at, sunset_at, first_seen_at
`, row.ToolID, row.Version, row.ManifestJSON, row.ConfigSchemaVersion, row.StateSchemaVersion,
		row.SandboxMode, row.Status, row.BreakerOpenAt, row.SunsetAt,
	).Scan(
		&out.ToolID, &out.Version, &out.ManifestJSON, &out.ConfigSchemaVersion, &out.StateSchemaVersion,
		&out.SandboxMode, &out.Status, &breaker, &sunset, &out.FirstSeenAt,
	)
	if err != nil {
		return nil, err
	}
	out.BreakerOpenAt = breaker
	out.SunsetAt = sunset
	return &out, nil
}

// ListToolVersions returns all mirrored versions, optionally filtered by tool_id.
func ListToolVersions(ctx context.Context, pool *pgxpool.Pool, toolID string) ([]VersionRow, error) {
	var rows pgx.Rows
	var err error
	if toolID == "" {
		rows, err = pool.Query(ctx, `
SELECT tool_id, version, manifest_json, config_schema_version, state_schema_version,
  sandbox_mode, status, breaker_open_at, sunset_at, first_seen_at
FROM course.content_tool_versions
ORDER BY tool_id, version
`)
	} else {
		rows, err = pool.Query(ctx, `
SELECT tool_id, version, manifest_json, config_schema_version, state_schema_version,
  sandbox_mode, status, breaker_open_at, sunset_at, first_seen_at
FROM course.content_tool_versions
WHERE tool_id = $1
ORDER BY version
`, toolID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]VersionRow, 0)
	for rows.Next() {
		var r VersionRow
		var breaker, sunset *time.Time
		if err := rows.Scan(
			&r.ToolID, &r.Version, &r.ManifestJSON, &r.ConfigSchemaVersion, &r.StateSchemaVersion,
			&r.SandboxMode, &r.Status, &breaker, &sunset, &r.FirstSeenAt,
		); err != nil {
			return nil, err
		}
		r.BreakerOpenAt = breaker
		r.SunsetAt = sunset
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetToolVersion returns one mirror row or nil.
func GetToolVersion(ctx context.Context, pool *pgxpool.Pool, toolID, version string) (*VersionRow, error) {
	var r VersionRow
	var breaker, sunset *time.Time
	err := pool.QueryRow(ctx, `
SELECT tool_id, version, manifest_json, config_schema_version, state_schema_version,
  sandbox_mode, status, breaker_open_at, sunset_at, first_seen_at
FROM course.content_tool_versions
WHERE tool_id = $1 AND version = $2
`, toolID, version).Scan(
		&r.ToolID, &r.Version, &r.ManifestJSON, &r.ConfigSchemaVersion, &r.StateSchemaVersion,
		&r.SandboxMode, &r.Status, &breaker, &sunset, &r.FirstSeenAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.BreakerOpenAt = breaker
	r.SunsetAt = sunset
	return &r, nil
}

// PatchToolVersionStatus updates status and optionally resets/opens the breaker.
type VersionPatch struct {
	Status          *string
	ResetBreaker    bool
	OpenBreaker     bool
	SunsetAt        *time.Time
}

// PatchToolVersion applies admin status / breaker changes.
func PatchToolVersion(ctx context.Context, pool *pgxpool.Pool, toolID, version string, patch VersionPatch) (*VersionRow, error) {
	cur, err := GetToolVersion(ctx, pool, toolID, version)
	if err != nil || cur == nil {
		return cur, err
	}
	status := cur.Status
	if patch.Status != nil && *patch.Status != "" {
		status = *patch.Status
	}
	breaker := cur.BreakerOpenAt
	if patch.ResetBreaker {
		breaker = nil
	}
	if patch.OpenBreaker {
		now := time.Now().UTC()
		breaker = &now
	}
	sunset := cur.SunsetAt
	if patch.SunsetAt != nil {
		sunset = patch.SunsetAt
	}
	var out VersionRow
	var b, s *time.Time
	err = pool.QueryRow(ctx, `
UPDATE course.content_tool_versions
SET status = $3, breaker_open_at = $4, sunset_at = $5
WHERE tool_id = $1 AND version = $2
RETURNING tool_id, version, manifest_json, config_schema_version, state_schema_version,
  sandbox_mode, status, breaker_open_at, sunset_at, first_seen_at
`, toolID, version, status, breaker, sunset).Scan(
		&out.ToolID, &out.Version, &out.ManifestJSON, &out.ConfigSchemaVersion, &out.StateSchemaVersion,
		&out.SandboxMode, &out.Status, &b, &s, &out.FirstSeenAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out.BreakerOpenAt = b
	out.SunsetAt = s
	return &out, nil
}

// ListPublishedVersionsForTool returns version strings for a tool (any status except disabled optional).
func ListPublishedVersionsForTool(ctx context.Context, pool *pgxpool.Pool, toolID string) ([]string, error) {
	rows, err := pool.Query(ctx, `
SELECT version FROM course.content_tool_versions
WHERE tool_id = $1 AND status IN ('active','deprecated','sunset')
`, toolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
