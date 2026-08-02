package contenttools

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func scanInstance(row pgx.Row) (*InstanceRow, error) {
	var r InstanceRow
	var cfg []byte
	err := row.Scan(
		&r.ID, &r.CourseID, &r.StructureItemID, &r.HostKind, &r.SectionKey,
		&r.ToolID, &r.ToolVersion, &r.Title, &cfg, &r.ConfigSchemaVersion,
		&r.Status, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(cfg) == 0 {
		cfg = []byte(`{}`)
	}
	r.ConfigJSON = json.RawMessage(cfg)
	return &r, nil
}

const instanceCols = `
id, course_id, structure_item_id, host_kind, section_key,
tool_id, tool_version, title, config_json, config_schema_version,
status, created_by, created_at, updated_at
`

// GetInstance returns one instance by id scoped to course, or nil.
func GetInstance(ctx context.Context, pool *pgxpool.Pool, courseID, instanceID uuid.UUID) (*InstanceRow, error) {
	row := pool.QueryRow(ctx, `
SELECT `+instanceCols+`
FROM course.content_tool_instances
WHERE id = $1 AND course_id = $2
`, instanceID, courseID)
	return scanInstance(row)
}

// GetInstanceByID returns one instance by primary key (CT.6 context assembly).
func GetInstanceByID(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID) (*InstanceRow, error) {
	row := pool.QueryRow(ctx, `
SELECT `+instanceCols+`
FROM course.content_tool_instances
WHERE id = $1
`, instanceID)
	return scanInstance(row)
}

// ListInstances returns instances for a course, optionally filtered by item/host/status.
// A single query — no N+1 (AC-9).
func ListInstances(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	structureItemID *uuid.UUID,
	hostKind string,
	includeArchived bool,
) ([]InstanceRow, error) {
	rows, err := pool.Query(ctx, `
SELECT `+instanceCols+`
FROM course.content_tool_instances
WHERE course_id = $1
  AND ($2::uuid IS NULL OR structure_item_id = $2)
  AND ($3::text = '' OR host_kind = $3)
  AND ($4::bool OR status = 'active')
ORDER BY created_at ASC
`, courseID, structureItemID, hostKind, includeArchived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InstanceRow
	for rows.Next() {
		r, err := scanInstance(rows)
		if err != nil {
			return nil, err
		}
		if r != nil {
			out = append(out, *r)
		}
	}
	return out, rows.Err()
}

// CountActiveForItem returns active instance count for a structure item (or syllabus when item nil).
func CountActiveForItem(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, structureItemID *uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.content_tool_instances
WHERE course_id = $1 AND status = 'active'
  AND (
    ($2::uuid IS NULL AND structure_item_id IS NULL)
    OR structure_item_id = $2
  )
`, courseID, structureItemID).Scan(&n)
	return n, err
}

// UpsertInstanceForImport inserts or updates an instance with an explicit id (course JSON import).
// Does not touch learner state. On conflict, only updates when the existing row belongs to courseID.
func UpsertInstanceForImport(ctx context.Context, pool *pgxpool.Pool, r InstanceRow) error {
	cfg := r.ConfigJSON
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	ver := r.ConfigSchemaVersion
	if ver <= 0 {
		ver = 1
	}
	status := r.Status
	if status == "" {
		status = "active"
	}
	tag, err := pool.Exec(ctx, `
INSERT INTO course.content_tool_instances (
  id, course_id, structure_item_id, host_kind, section_key, tool_id, tool_version,
  title, config_json, config_schema_version, status, created_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,NOW(),NOW())
ON CONFLICT (id) DO UPDATE SET
  structure_item_id = EXCLUDED.structure_item_id,
  host_kind = EXCLUDED.host_kind,
  section_key = EXCLUDED.section_key,
  tool_id = EXCLUDED.tool_id,
  tool_version = EXCLUDED.tool_version,
  title = EXCLUDED.title,
  config_json = EXCLUDED.config_json,
  config_schema_version = EXCLUDED.config_schema_version,
  status = EXCLUDED.status,
  updated_at = NOW()
WHERE course.content_tool_instances.course_id = EXCLUDED.course_id
`, r.ID, r.CourseID, r.StructureItemID, r.HostKind, r.SectionKey, r.ToolID, r.ToolVersion,
		r.Title, cfg, ver, status, r.CreatedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Conflict on another course's primary key (or no-op).
		return errors.New("content tool instance id already exists in another course")
	}
	return nil
}

// InsertInstanceIfMissing inserts an instance with an explicit id; skips when the id already exists.
// Returns true when a row was inserted.
func InsertInstanceIfMissing(ctx context.Context, pool *pgxpool.Pool, r InstanceRow) (bool, error) {
	cfg := r.ConfigJSON
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	ver := r.ConfigSchemaVersion
	if ver <= 0 {
		ver = 1
	}
	status := r.Status
	if status == "" {
		status = "active"
	}
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_instances (
  id, course_id, structure_item_id, host_kind, section_key, tool_id, tool_version,
  title, config_json, config_schema_version, status, created_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,NOW(),NOW())
ON CONFLICT (id) DO NOTHING
RETURNING id
`, r.ID, r.CourseID, r.StructureItemID, r.HostKind, r.SectionKey, r.ToolID, r.ToolVersion,
		r.Title, cfg, ver, status, r.CreatedBy).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// DeleteAllInstancesForCourse removes every content-tool instance for a course
// (including syllabus hosts with null structure_item_id). Cascades learner state.
func DeleteAllInstancesForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
DELETE FROM course.content_tool_instances WHERE course_id = $1
`, courseID)
	return err
}

// DeleteInstancesNotInExport deletes course instances whose ids are not in keep.
// Empty keep deletes all instances for the course.
func DeleteInstancesNotInExport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, keep []uuid.UUID) error {
	if len(keep) == 0 {
		return DeleteAllInstancesForCourse(ctx, pool, courseID)
	}
	_, err := pool.Exec(ctx, `
DELETE FROM course.content_tool_instances
WHERE course_id = $1 AND NOT (id = ANY($2::uuid[]))
`, courseID, keep)
	return err
}

// CreateInstance inserts a new instance row.
func CreateInstance(ctx context.Context, pool *pgxpool.Pool, r InstanceRow) (*InstanceRow, error) {
	cfg := r.ConfigJSON
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	ver := r.ConfigSchemaVersion
	if ver <= 0 {
		ver = 1
	}
	row := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_instances (
  course_id, structure_item_id, host_kind, section_key, tool_id, tool_version,
  title, config_json, config_schema_version, status, created_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,'active',$10)
RETURNING `+instanceCols+`
`, r.CourseID, r.StructureItemID, r.HostKind, r.SectionKey, r.ToolID, r.ToolVersion,
		r.Title, cfg, ver, r.CreatedBy)
	return scanInstance(row)
}

// UpdateInstance patches mutable fields.
func UpdateInstance(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, instanceID uuid.UUID,
	title *string,
	sectionKey *string,
	config json.RawMessage,
	status *string,
) (*InstanceRow, error) {
	row := pool.QueryRow(ctx, `
UPDATE course.content_tool_instances
SET
  title = COALESCE($3, title),
  section_key = COALESCE($4, section_key),
  config_json = COALESCE($5::jsonb, config_json),
  status = COALESCE($6, status),
  updated_at = NOW()
WHERE id = $1 AND course_id = $2
RETURNING `+instanceCols+`
`, instanceID, courseID, title, sectionKey, nullableJSON(config), status)
	return scanInstance(row)
}

func nullableJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return nil
	}
	return []byte(raw)
}

// ArchiveInstance soft-deletes an instance (status=archived).
func ArchiveInstance(ctx context.Context, pool *pgxpool.Pool, courseID, instanceID uuid.UUID) (*InstanceRow, error) {
	st := "archived"
	return UpdateInstance(ctx, pool, courseID, instanceID, nil, nil, nil, &st)
}

// HardDeleteInstance permanently deletes an instance (cascades states/events that reference it).
func HardDeleteInstance(ctx context.Context, pool *pgxpool.Pool, courseID, instanceID uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
DELETE FROM course.content_tool_instances
WHERE id = $1 AND course_id = $2
`, instanceID, courseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ArchiveUnreferencedForItem archives active instances for a structure item (or syllabus when
// structureItemID is nil) whose ids are not in referencedIDs.
func ArchiveUnreferencedForItem(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	structureItemID *uuid.UUID,
	referencedIDs []uuid.UUID,
) error {
	if referencedIDs == nil {
		referencedIDs = []uuid.UUID{}
	}
	_, err := pool.Exec(ctx, `
UPDATE course.content_tool_instances
SET status = 'archived', updated_at = NOW()
WHERE course_id = $1
  AND status = 'active'
  AND (
    ($2::uuid IS NULL AND structure_item_id IS NULL)
    OR structure_item_id = $2
  )
  AND NOT (id = ANY($3::uuid[]))
`, courseID, structureItemID, referencedIDs)
	return err
}

// GetInstancesByIDs returns instances in the course matching the given ids (any status).
func GetInstancesByIDs(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, ids []uuid.UUID) ([]InstanceRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT `+instanceCols+`
FROM course.content_tool_instances
WHERE course_id = $1 AND id = ANY($2::uuid[])
ORDER BY created_at ASC
`, courseID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InstanceRow
	for rows.Next() {
		r, err := scanInstance(rows)
		if err != nil {
			return nil, err
		}
		if r != nil {
			out = append(out, *r)
		}
	}
	return out, rows.Err()
}

// IsConfigSizeViolation reports whether err is the config_json size CHECK.
func IsConfigSizeViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23514" && (pgErr.ConstraintName == "content_tool_instances_config_size" ||
			pgErr.ConstraintName == "content_tool_states_size")
	}
	return false
}
