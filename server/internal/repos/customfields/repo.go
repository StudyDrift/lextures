// Package customfields persists org-scoped custom field definitions and entity values (plan 18.7).
package customfields

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EntityType identifies which parent entity a definition applies to.
type EntityType string

const (
	EntityUser       EntityType = "user"
	EntityCourse     EntityType = "course"
	EntityEnrollment EntityType = "enrollment"
)

// FieldType is the value type for a custom field.
type FieldType string

const (
	FieldText    FieldType = "text"
	FieldNumber  FieldType = "number"
	FieldBoolean FieldType = "boolean"
	FieldDate    FieldType = "date"
	FieldSelect  FieldType = "select"
)

// Visibility controls which roles may see a field value.
type Visibility string

const (
	VisibilityAdminOnly  Visibility = "admin_only"
	VisibilityInstructor Visibility = "instructor"
	VisibilityStudent    Visibility = "student"
)

// Definition is a custom field schema row.
type Definition struct {
	ID            uuid.UUID  `json:"id"`
	OrgID         uuid.UUID  `json:"orgId"`
	EntityType    EntityType `json:"entityType"`
	Key           string     `json:"key"`
	Label         string     `json:"label"`
	FieldType     FieldType  `json:"fieldType"`
	SelectOptions []string   `json:"selectOptions,omitempty"`
	IsRequired    bool       `json:"isRequired"`
	Visibility    Visibility `json:"visibility"`
	SortOrder     int        `json:"sortOrder"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

const maxDefinitionsPerEntity = 50

var (
	ErrNotFound      = errors.New("custom field definition not found")
	ErrMaxFields     = errors.New("maximum custom fields per entity type reached")
	ErrDuplicateKey  = errors.New("custom field key already exists")
)

// ListDefinitions returns active definitions for an org and entity type, ordered by sort_order.
func ListDefinitions(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, entityType EntityType, includeDeleted bool) ([]Definition, error) {
	where := `org_id = $1 AND entity_type = $2`
	if !includeDeleted {
		where += ` AND deleted_at IS NULL`
	}
	rows, err := pool.Query(ctx, fmt.Sprintf(`
SELECT id, org_id, entity_type::text, key, label, field_type::text, select_options,
       is_required, visibility::text, sort_order, deleted_at, created_at
FROM tenant.custom_field_definitions
WHERE %s
ORDER BY sort_order ASC, created_at ASC
`, where), orgID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDefinitions(rows)
}

// GetDefinition loads one definition by id scoped to org.
func GetDefinition(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) (*Definition, error) {
	row := pool.QueryRow(ctx, `
SELECT id, org_id, entity_type::text, key, label, field_type::text, select_options,
       is_required, visibility::text, sort_order, deleted_at, created_at
FROM tenant.custom_field_definitions
WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL
`, id, orgID)
	def, err := scanDefinition(row)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return def, err
}

// CreateDefinition inserts a new field definition.
func CreateDefinition(ctx context.Context, pool *pgxpool.Pool, d Definition) (*Definition, error) {
	var count int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM tenant.custom_field_definitions
WHERE org_id = $1 AND entity_type = $2 AND deleted_at IS NULL
`, d.OrgID, d.EntityType).Scan(&count); err != nil {
		return nil, err
	}
	if count >= maxDefinitionsPerEntity {
		return nil, ErrMaxFields
	}
	var selectOpts any
	if len(d.SelectOptions) > 0 {
		selectOpts = d.SelectOptions
	}
	row := pool.QueryRow(ctx, `
INSERT INTO tenant.custom_field_definitions (
  org_id, entity_type, key, label, field_type, select_options, is_required, visibility, sort_order
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, org_id, entity_type::text, key, label, field_type::text, select_options,
          is_required, visibility::text, sort_order, deleted_at, created_at
`, d.OrgID, d.EntityType, d.Key, d.Label, d.FieldType, selectOpts, d.IsRequired, d.Visibility, d.SortOrder)
	def, err := scanDefinition(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateKey
		}
		return nil, err
	}
	return def, nil
}

// UpdateDefinition patches mutable definition fields.
func UpdateDefinition(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID, label *string, selectOptions []string, isRequired *bool, visibility *Visibility, sortOrder *int) (*Definition, error) {
	sets := []string{}
	args := []any{orgID, id}
	argIdx := 3
	if label != nil {
		sets = append(sets, fmt.Sprintf("label = $%d", argIdx))
		args = append(args, strings.TrimSpace(*label))
		argIdx++
	}
	if selectOptions != nil {
		var opts any
		if len(selectOptions) > 0 {
			opts = selectOptions
		}
		sets = append(sets, fmt.Sprintf("select_options = $%d", argIdx))
		args = append(args, opts)
		argIdx++
	}
	if isRequired != nil {
		sets = append(sets, fmt.Sprintf("is_required = $%d", argIdx))
		args = append(args, *isRequired)
		argIdx++
	}
	if visibility != nil {
		sets = append(sets, fmt.Sprintf("visibility = $%d", argIdx))
		args = append(args, *visibility)
		argIdx++
	}
	if sortOrder != nil {
		sets = append(sets, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *sortOrder)
		argIdx++
	}
	if len(sets) == 0 {
		return GetDefinition(ctx, pool, orgID, id)
	}
	q := fmt.Sprintf(`
UPDATE tenant.custom_field_definitions SET %s
WHERE id = $2 AND org_id = $1 AND deleted_at IS NULL
RETURNING id, org_id, entity_type::text, key, label, field_type::text, select_options,
          is_required, visibility::text, sort_order, deleted_at, created_at
`, strings.Join(sets, ", "))
	row := pool.QueryRow(ctx, q, args...)
	def, err := scanDefinition(row)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return def, err
}

// SoftDeleteDefinition marks a definition deleted without removing stored values.
func SoftDeleteDefinition(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
UPDATE tenant.custom_field_definitions SET deleted_at = NOW()
WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL
`, id, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReorderDefinitions updates sort_order for the provided ids in order.
func ReorderDefinitions(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, entityType EntityType, ids []uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for i, id := range ids {
		tag, err := tx.Exec(ctx, `
UPDATE tenant.custom_field_definitions SET sort_order = $4
WHERE id = $1 AND org_id = $2 AND entity_type = $3 AND deleted_at IS NULL
`, id, orgID, entityType, i)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
	}
	return tx.Commit(ctx)
}

// GetUserCustomFields loads raw JSONB custom fields for a user in org.
func GetUserCustomFields(ctx context.Context, pool *pgxpool.Pool, orgID, userID uuid.UUID) (map[string]any, error) {
	return getEntityCustomFields(ctx, pool, `
SELECT custom_fields FROM "user".users WHERE id = $1 AND org_id = $2
`, userID, orgID)
}

// SetUserCustomFields replaces custom field values for a user.
func SetUserCustomFields(ctx context.Context, pool *pgxpool.Pool, orgID, userID uuid.UUID, values map[string]any) error {
	return setEntityCustomFields(ctx, pool, `
UPDATE "user".users SET custom_fields = $3::jsonb WHERE id = $1 AND org_id = $2
`, userID, orgID, values)
}

// GetCourseCustomFields loads raw JSONB custom fields for a course in org.
func GetCourseCustomFields(ctx context.Context, pool *pgxpool.Pool, orgID, courseID uuid.UUID) (map[string]any, error) {
	return getEntityCustomFields(ctx, pool, `
SELECT custom_fields FROM course.courses WHERE id = $1 AND org_id = $2
`, courseID, orgID)
}

// SetCourseCustomFields replaces custom field values for a course.
func SetCourseCustomFields(ctx context.Context, pool *pgxpool.Pool, orgID, courseID uuid.UUID, values map[string]any) error {
	return setEntityCustomFields(ctx, pool, `
UPDATE course.courses SET custom_fields = $3::jsonb WHERE id = $1 AND org_id = $2
`, courseID, orgID, values)
}

// GetEnrollmentCustomFields loads raw JSONB custom fields for an enrollment in org.
func GetEnrollmentCustomFields(ctx context.Context, pool *pgxpool.Pool, orgID, enrollmentID uuid.UUID) (map[string]any, error) {
	return getEntityCustomFields(ctx, pool, `
SELECT ce.custom_fields
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
WHERE ce.id = $1 AND c.org_id = $2
`, enrollmentID, orgID)
}

// SetEnrollmentCustomFields replaces custom field values for an enrollment.
func SetEnrollmentCustomFields(ctx context.Context, pool *pgxpool.Pool, orgID, enrollmentID uuid.UUID, values map[string]any) error {
	return setEntityCustomFields(ctx, pool, `
UPDATE course.course_enrollments ce
SET custom_fields = $3::jsonb
FROM course.courses c
WHERE ce.id = $1 AND ce.course_id = c.id AND c.org_id = $2
`, enrollmentID, orgID, values)
}

func getEntityCustomFields(ctx context.Context, pool *pgxpool.Pool, query string, id, orgID uuid.UUID) (map[string]any, error) {
	var raw []byte
	err := pool.QueryRow(ctx, query, id, orgID).Scan(&raw)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return unmarshalFields(raw)
}

func setEntityCustomFields(ctx context.Context, pool *pgxpool.Pool, query string, id, orgID uuid.UUID, values map[string]any) error {
	if values == nil {
		values = map[string]any{}
	}
	b, err := json.Marshal(values)
	if err != nil {
		return err
	}
	tag, err := pool.Exec(ctx, query, id, orgID, string(b))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func unmarshalFields(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDefinition(row scannable) (*Definition, error) {
	var d Definition
	var entityType, fieldType, visibility string
	var selectOpts []string
	err := row.Scan(&d.ID, &d.OrgID, &entityType, &d.Key, &d.Label, &fieldType, &selectOpts,
		&d.IsRequired, &visibility, &d.SortOrder, &d.DeletedAt, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	d.EntityType = EntityType(entityType)
	d.FieldType = FieldType(fieldType)
	d.Visibility = Visibility(visibility)
	if len(selectOpts) > 0 {
		d.SelectOptions = selectOpts
	}
	return &d, nil
}

func scanDefinitions(rows pgx.Rows) ([]Definition, error) {
	var out []Definition
	for rows.Next() {
		var d Definition
		var entityType, fieldType, visibility string
		var selectOpts []string
		if err := rows.Scan(&d.ID, &d.OrgID, &entityType, &d.Key, &d.Label, &fieldType, &selectOpts,
			&d.IsRequired, &visibility, &d.SortOrder, &d.DeletedAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		d.EntityType = EntityType(entityType)
		d.FieldType = FieldType(fieldType)
		d.Visibility = Visibility(visibility)
		if len(selectOpts) > 0 {
			d.SelectOptions = selectOpts
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unique")
}
