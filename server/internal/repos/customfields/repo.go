// Package customfields provides persistence for org custom field definitions and entity values (plan 18.7).
package customfields

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

const MaxDefinitionsPerEntity = 50

// EntityType is the parent entity for a custom field definition.
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

// Visibility controls which roles may read a custom field value.
type Visibility string

const (
	VisibilityAdminOnly  Visibility = "admin_only"
	VisibilityInstructor Visibility = "instructor"
	VisibilityStudent    Visibility = "student"
)

var (
	ErrNotFound      = errors.New("customfields: definition not found")
	ErrDuplicateKey  = errors.New("customfields: duplicate key")
	ErrMaxFields     = errors.New("customfields: maximum field count reached")
	ErrReservedKey   = errors.New("customfields: reserved key")
)

// Definition is an org custom field schema row.
type Definition struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	EntityType    EntityType
	Key           string
	Label         string
	FieldType     FieldType
	SelectOptions []string
	IsRequired    bool
	Visibility    Visibility
	SortOrder     int
	DeletedAt     *time.Time
	CreatedAt     time.Time
}

// CreateParams holds input for a new definition.
type CreateParams struct {
	OrgID         uuid.UUID
	EntityType    EntityType
	Key           string
	Label         string
	FieldType     FieldType
	SelectOptions []string
	IsRequired    bool
	Visibility    Visibility
	SortOrder     int
}

// UpdateParams holds mutable definition fields.
type UpdateParams struct {
	Label         *string
	FieldType     *FieldType
	SelectOptions *[]string
	IsRequired    *bool
	Visibility    *Visibility
	SortOrder     *int
}

func scanDefinition(row pgx.Row) (Definition, error) {
	var d Definition
	var selectOpts []string
	var deletedAt *time.Time
	err := row.Scan(
		&d.ID, &d.OrgID, &d.EntityType, &d.Key, &d.Label, &d.FieldType,
		&selectOpts, &d.IsRequired, &d.Visibility, &d.SortOrder, &deletedAt, &d.CreatedAt,
	)
	if err != nil {
		return Definition{}, err
	}
	d.SelectOptions = selectOpts
	d.DeletedAt = deletedAt
	return d, nil
}

const definitionColumns = `
id, org_id, entity_type, key, label, field_type, select_options,
is_required, visibility, sort_order, deleted_at, created_at
`

// ListDefinitions returns active definitions for an org and entity type ordered by sort_order.
func ListDefinitions(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, entityType EntityType) ([]Definition, error) {
	rows, err := pool.Query(ctx, `
SELECT `+definitionColumns+`
FROM tenant.custom_field_definitions
WHERE org_id = $1 AND entity_type = $2 AND deleted_at IS NULL
ORDER BY sort_order ASC, created_at ASC
`, orgID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Definition
	for rows.Next() {
		d, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDefinition loads one active definition by id scoped to org.
func GetDefinition(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) (Definition, error) {
	row := pool.QueryRow(ctx, `
SELECT `+definitionColumns+`
FROM tenant.custom_field_definitions
WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL
`, id, orgID)
	d, err := scanDefinition(row)
	if err == pgx.ErrNoRows {
		return Definition{}, ErrNotFound
	}
	return d, err
}

// CountDefinitions returns the number of active definitions for org+entity.
func CountDefinitions(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, entityType EntityType) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM tenant.custom_field_definitions
WHERE org_id = $1 AND entity_type = $2 AND deleted_at IS NULL
`, orgID, entityType).Scan(&n)
	return n, err
}

// CreateDefinition inserts a new custom field definition.
func CreateDefinition(ctx context.Context, pool *pgxpool.Pool, p CreateParams) (Definition, error) {
	n, err := CountDefinitions(ctx, pool, p.OrgID, p.EntityType)
	if err != nil {
		return Definition{}, err
	}
	if n >= MaxDefinitionsPerEntity {
		return Definition{}, ErrMaxFields
	}
	row := pool.QueryRow(ctx, `
INSERT INTO tenant.custom_field_definitions (
  org_id, entity_type, key, label, field_type, select_options,
  is_required, visibility, sort_order
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING `+definitionColumns,
		p.OrgID, p.EntityType, p.Key, p.Label, p.FieldType, p.SelectOptions,
		p.IsRequired, p.Visibility, p.SortOrder,
	)
	d, err := scanDefinition(row)
	if err != nil {
		if isUniqueViolation(err) {
			return Definition{}, ErrDuplicateKey
		}
		return Definition{}, err
	}
	return d, nil
}

// UpdateDefinition updates mutable fields on an active definition.
func UpdateDefinition(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID, p UpdateParams) (Definition, error) {
	cur, err := GetDefinition(ctx, pool, orgID, id)
	if err != nil {
		return Definition{}, err
	}
	label := cur.Label
	if p.Label != nil {
		label = *p.Label
	}
	fieldType := cur.FieldType
	if p.FieldType != nil {
		fieldType = *p.FieldType
	}
	selectOpts := cur.SelectOptions
	if p.SelectOptions != nil {
		selectOpts = *p.SelectOptions
	}
	isRequired := cur.IsRequired
	if p.IsRequired != nil {
		isRequired = *p.IsRequired
	}
	visibility := cur.Visibility
	if p.Visibility != nil {
		visibility = *p.Visibility
	}
	sortOrder := cur.SortOrder
	if p.SortOrder != nil {
		sortOrder = *p.SortOrder
	}
	row := pool.QueryRow(ctx, `
UPDATE tenant.custom_field_definitions SET
  label = $3,
  field_type = $4,
  select_options = $5,
  is_required = $6,
  visibility = $7,
  sort_order = $8
WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL
RETURNING `+definitionColumns,
		id, orgID, label, fieldType, selectOpts, isRequired, visibility, sortOrder,
	)
	d, err := scanDefinition(row)
	if err == pgx.ErrNoRows {
		return Definition{}, ErrNotFound
	}
	return d, err
}

// SoftDeleteDefinition marks a definition as deleted.
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

// ReorderDefinitions updates sort_order for each id in order.
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
			return fmt.Errorf("customfields: definition %s not found for reorder", id)
		}
	}
	return tx.Commit(ctx)
}

// GetUserCustomFields loads raw JSONB custom fields for a user.
func GetUserCustomFields(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (map[string]any, error) {
	return getEntityCustomFields(ctx, pool, `"user".users`, userID)
}

// SetUserCustomFields writes custom fields JSONB for a user in org.
func SetUserCustomFields(ctx context.Context, pool *pgxpool.Pool, orgID, userID uuid.UUID, fields map[string]any) error {
	return setEntityCustomFields(ctx, pool, `"user".users`, orgID, userID, fields)
}

// GetCourseCustomFields loads raw JSONB custom fields for a course.
func GetCourseCustomFields(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (map[string]any, error) {
	return getEntityCustomFields(ctx, pool, `course.courses`, courseID)
}

// SetCourseCustomFields writes custom fields JSONB for a course in org.
func SetCourseCustomFields(ctx context.Context, pool *pgxpool.Pool, orgID, courseID uuid.UUID, fields map[string]any) error {
	return setEntityCustomFields(ctx, pool, `course.courses`, orgID, courseID, fields)
}

func getEntityCustomFields(ctx context.Context, pool *pgxpool.Pool, table string, id uuid.UUID) (map[string]any, error) {
	var raw []byte
	err := pool.QueryRow(ctx, fmt.Sprintf(`SELECT custom_fields FROM %s WHERE id = $1`, table), id).Scan(&raw)
	if err == pgx.ErrNoRows {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if len(raw) > 0 && string(raw) != "{}" {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func setEntityCustomFields(ctx context.Context, pool *pgxpool.Pool, table string, orgID, id uuid.UUID, fields map[string]any) error {
	raw, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	tag, err := pool.Exec(ctx, fmt.Sprintf(`
UPDATE %s SET custom_fields = $2::jsonb WHERE id = $1 AND org_id = $3
`, table), id, raw, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ Code() string }
	if errors.As(err, &pgErr) {
		return pgErr.Code() == "23505"
	}
	return false
}
