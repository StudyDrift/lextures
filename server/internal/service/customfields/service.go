// Package customfields implements validation, visibility filtering, and merge logic (plan 18.7).
package customfields

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	cfrepo "github.com/lextures/lextures/server/internal/repos/customfields"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

const permGlobalRBACManage = "global:app:rbac:manage"

// Audience is the visibility tier of the requesting principal.
type Audience string

const (
	AudienceAdmin      Audience = "admin"
	AudienceInstructor Audience = "instructor"
	AudienceStudent    Audience = "student"
)

// ValidationError describes one invalid custom field value.
type ValidationError struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Key, e.Message)
}

var (
	keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	reservedKeys = map[string]struct{}{
		"email": {}, "first_name": {}, "last_name": {}, "role": {}, "external_id": {},
		"id": {}, "org_id": {}, "created_at": {}, "updated_at": {}, "status": {},
		"course_code": {}, "title": {}, "display_name": {},
	}
)

// Service coordinates custom field business rules.
type Service struct {
	pool *pgxpool.Pool
}

// New constructs a Service.
func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ValidateKey checks a proposed field key slug.
func ValidateKey(key string) error {
	key = strings.TrimSpace(key)
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("key must be snake_case starting with a letter")
	}
	if _, reserved := reservedKeys[key]; reserved {
		return fmt.Errorf("key %q is reserved", key)
	}
	return nil
}

// ValidateDefinitionInput validates create/update definition payloads.
func ValidateDefinitionInput(entityType cfrepo.EntityType, key, label string, fieldType cfrepo.FieldType, selectOptions []string, visibility cfrepo.Visibility) error {
	if entityType != cfrepo.EntityUser && entityType != cfrepo.EntityCourse && entityType != cfrepo.EntityEnrollment {
		return fmt.Errorf("invalid entity type")
	}
	if err := ValidateKey(key); err != nil {
		return err
	}
	if strings.TrimSpace(label) == "" {
		return fmt.Errorf("label is required")
	}
	switch fieldType {
	case cfrepo.FieldText, cfrepo.FieldNumber, cfrepo.FieldBoolean, cfrepo.FieldDate, cfrepo.FieldSelect:
	default:
		return fmt.Errorf("invalid field type")
	}
	if fieldType == cfrepo.FieldSelect {
		if len(selectOptions) == 0 {
			return fmt.Errorf("select fields require at least one option")
		}
	}
	switch visibility {
	case cfrepo.VisibilityAdminOnly, cfrepo.VisibilityInstructor, cfrepo.VisibilityStudent:
	default:
		return fmt.Errorf("invalid visibility")
	}
	return nil
}

// AudienceForUser resolves the visibility audience for a principal.
func (s *Service) AudienceForUser(ctx context.Context, userID, orgID uuid.UUID) (Audience, error) {
	if ok, err := rbac.UserHasPermission(ctx, s.pool, userID, permGlobalRBACManage); err != nil {
		return "", err
	} else if ok {
		return AudienceAdmin, nil
	}
	if ok, err := orgroles.UserHasRole(ctx, s.pool, userID, orgID, orgroles.RoleOrgAdmin); err != nil {
		return "", err
	} else if ok {
		return AudienceAdmin, nil
	}
	if ok, err := orgroles.UserHasRole(ctx, s.pool, userID, orgID, orgroles.RoleOrgViewer); err != nil {
		return "", err
	} else if ok {
		return AudienceAdmin, nil
	}
	var roleName string
	err := s.pool.QueryRow(ctx, `
SELECT ar.name FROM "user".user_app_roles uar
JOIN "user".app_roles ar ON ar.id = uar.role_id
WHERE uar.user_id = $1
ORDER BY CASE ar.name WHEN 'Teacher' THEN 0 WHEN 'Student' THEN 1 ELSE 2 END
LIMIT 1
`, userID).Scan(&roleName)
	if err != nil {
		return AudienceStudent, nil
	}
	if roleName == "Teacher" {
		return AudienceInstructor, nil
	}
	return AudienceStudent, nil
}

// FilterValues returns only values whose definitions are visible to the audience and not soft-deleted.
func FilterValues(values map[string]any, defs []cfrepo.Definition, audience Audience, includeDeleted bool) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	active := make(map[string]cfrepo.Definition, len(defs))
	for _, d := range defs {
		if d.DeletedAt != nil && !includeDeleted {
			continue
		}
		active[d.Key] = d
	}
	out := make(map[string]any)
	for key, val := range values {
		def, ok := active[key]
		if !ok {
			continue
		}
		if def.DeletedAt != nil && !includeDeleted {
			continue
		}
		if visibleTo(def.Visibility, audience) {
			out[key] = val
		}
	}
	return out
}

func visibleTo(v cfrepo.Visibility, audience Audience) bool {
	switch audience {
	case AudienceAdmin:
		return true
	case AudienceInstructor:
		return v == cfrepo.VisibilityInstructor || v == cfrepo.VisibilityStudent
	case AudienceStudent:
		return v == cfrepo.VisibilityStudent
	default:
		return false
	}
}

// ValidateAndMergeValues validates incoming values against active definitions and merges with existing.
func (s *Service) ValidateAndMergeValues(ctx context.Context, orgID uuid.UUID, entityType cfrepo.EntityType, incoming map[string]any, existing map[string]any) (map[string]any, []ValidationError, error) {
	defs, err := cfrepo.ListDefinitions(ctx, s.pool, orgID, entityType, false)
	if err != nil {
		return nil, nil, err
	}
	defByKey := make(map[string]cfrepo.Definition, len(defs))
	for _, d := range defs {
		defByKey[d.Key] = d
	}
	if existing == nil {
		existing = map[string]any{}
	}
	merged := make(map[string]any, len(existing))
	for k, v := range existing {
		merged[k] = v
	}
	var valErrs []ValidationError
	for key, raw := range incoming {
		def, ok := defByKey[key]
		if !ok {
			valErrs = append(valErrs, ValidationError{Key: key, Message: "unknown custom field"})
			continue
		}
		normalized, err := normalizeValue(def, raw)
		if err != nil {
			valErrs = append(valErrs, ValidationError{Key: key, Message: err.Error()})
			continue
		}
		if normalized == nil {
			delete(merged, key)
			continue
		}
		merged[key] = normalized
	}
	for _, def := range defs {
		if !def.IsRequired {
			continue
		}
		val, ok := merged[def.Key]
		if !ok || isEmptyValue(val) {
			valErrs = append(valErrs, ValidationError{Key: def.Key, Message: "required field is missing"})
		}
	}
	if len(valErrs) > 0 {
		return nil, valErrs, nil
	}
	// Drop values for deleted/unknown keys so API does not return stale keys after schema delete.
	cleaned := make(map[string]any)
	for key, val := range merged {
		if _, ok := defByKey[key]; ok {
			cleaned[key] = val
		}
	}
	return cleaned, nil, nil
}

func normalizeValue(def cfrepo.Definition, raw any) (any, error) {
	if raw == nil {
		return nil, nil
	}
	switch def.FieldType {
	case cfrepo.FieldText:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected text value")
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		return s, nil
	case cfrepo.FieldNumber:
		switch n := raw.(type) {
		case float64:
			return n, nil
		case int:
			return float64(n), nil
		case int64:
			return float64(n), nil
		case string:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
			if err != nil {
				return nil, fmt.Errorf("expected number value")
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("expected number value")
		}
	case cfrepo.FieldBoolean:
		switch b := raw.(type) {
		case bool:
			return b, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(b)) {
			case "true", "1", "yes":
				return true, nil
			case "false", "0", "no":
				return false, nil
			default:
				return nil, fmt.Errorf("expected boolean value")
			}
		default:
			return nil, fmt.Errorf("expected boolean value")
		}
	case cfrepo.FieldDate:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected date value (YYYY-MM-DD)")
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return nil, fmt.Errorf("expected date value (YYYY-MM-DD)")
		}
		return s, nil
	case cfrepo.FieldSelect:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected select option string")
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		for _, opt := range def.SelectOptions {
			if opt == s {
				return s, nil
			}
		}
		return nil, fmt.Errorf("value must be one of: %s", strings.Join(def.SelectOptions, ", "))
	default:
		return nil, errors.New("unsupported field type")
	}
}

func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}

// ListDefinitions proxies to repo.
func (s *Service) ListDefinitions(ctx context.Context, orgID uuid.UUID, entityType cfrepo.EntityType, includeDeleted bool) ([]cfrepo.Definition, error) {
	return cfrepo.ListDefinitions(ctx, s.pool, orgID, entityType, includeDeleted)
}

// CreateDefinition validates and creates a definition.
func (s *Service) CreateDefinition(ctx context.Context, orgID uuid.UUID, entityType cfrepo.EntityType, key, label string, fieldType cfrepo.FieldType, selectOptions []string, isRequired bool, visibility cfrepo.Visibility, sortOrder int) (*cfrepo.Definition, error) {
	if err := ValidateDefinitionInput(entityType, key, label, fieldType, selectOptions, visibility); err != nil {
		return nil, err
	}
	return cfrepo.CreateDefinition(ctx, s.pool, cfrepo.Definition{
		OrgID:         orgID,
		EntityType:    entityType,
		Key:           strings.TrimSpace(key),
		Label:         strings.TrimSpace(label),
		FieldType:     fieldType,
		SelectOptions: selectOptions,
		IsRequired:    isRequired,
		Visibility:    visibility,
		SortOrder:     sortOrder,
	})
}

// UpdateDefinition patches a definition.
func (s *Service) UpdateDefinition(ctx context.Context, orgID, id uuid.UUID, label *string, selectOptions []string, isRequired *bool, visibility *cfrepo.Visibility, sortOrder *int) (*cfrepo.Definition, error) {
	if visibility != nil {
		switch *visibility {
		case cfrepo.VisibilityAdminOnly, cfrepo.VisibilityInstructor, cfrepo.VisibilityStudent:
		default:
			return nil, fmt.Errorf("invalid visibility")
		}
	}
	return cfrepo.UpdateDefinition(ctx, s.pool, orgID, id, label, selectOptions, isRequired, visibility, sortOrder)
}

// SoftDeleteDefinition soft-deletes a definition.
func (s *Service) SoftDeleteDefinition(ctx context.Context, orgID, id uuid.UUID) error {
	return cfrepo.SoftDeleteDefinition(ctx, s.pool, orgID, id)
}

// ReorderDefinitions reorders definitions.
func (s *Service) ReorderDefinitions(ctx context.Context, orgID uuid.UUID, entityType cfrepo.EntityType, ids []uuid.UUID) error {
	return cfrepo.ReorderDefinitions(ctx, s.pool, orgID, entityType, ids)
}

// SetUserValues validates and stores user custom fields.
func (s *Service) SetUserValues(ctx context.Context, orgID, userID uuid.UUID, incoming map[string]any) (map[string]any, []ValidationError, error) {
	existing, err := cfrepo.GetUserCustomFields(ctx, s.pool, orgID, userID)
	if err != nil {
		return nil, nil, err
	}
	merged, valErrs, err := s.ValidateAndMergeValues(ctx, orgID, cfrepo.EntityUser, incoming, existing)
	if err != nil || len(valErrs) > 0 {
		return nil, valErrs, err
	}
	if err := cfrepo.SetUserCustomFields(ctx, s.pool, orgID, userID, merged); err != nil {
		return nil, nil, err
	}
	return merged, nil, nil
}

// GetUserValues returns filtered custom fields for a user.
func (s *Service) GetUserValues(ctx context.Context, orgID, userID uuid.UUID, audience Audience, includeDeleted bool) (map[string]any, error) {
	raw, err := cfrepo.GetUserCustomFields(ctx, s.pool, orgID, userID)
	if err != nil {
		return nil, err
	}
	defs, err := cfrepo.ListDefinitions(ctx, s.pool, orgID, cfrepo.EntityUser, includeDeleted)
	if err != nil {
		return nil, err
	}
	return FilterValues(raw, defs, audience, includeDeleted), nil
}

// SetCourseValues validates and stores course custom fields.
func (s *Service) SetCourseValues(ctx context.Context, orgID, courseID uuid.UUID, incoming map[string]any) (map[string]any, []ValidationError, error) {
	existing, err := cfrepo.GetCourseCustomFields(ctx, s.pool, orgID, courseID)
	if err != nil {
		return nil, nil, err
	}
	merged, valErrs, err := s.ValidateAndMergeValues(ctx, orgID, cfrepo.EntityCourse, incoming, existing)
	if err != nil || len(valErrs) > 0 {
		return nil, valErrs, err
	}
	if err := cfrepo.SetCourseCustomFields(ctx, s.pool, orgID, courseID, merged); err != nil {
		return nil, nil, err
	}
	return merged, nil, nil
}

// GetCourseValues returns filtered custom fields for a course.
func (s *Service) GetCourseValues(ctx context.Context, orgID, courseID uuid.UUID, audience Audience, includeDeleted bool) (map[string]any, error) {
	raw, err := cfrepo.GetCourseCustomFields(ctx, s.pool, orgID, courseID)
	if err != nil {
		return nil, err
	}
	defs, err := cfrepo.ListDefinitions(ctx, s.pool, orgID, cfrepo.EntityCourse, includeDeleted)
	if err != nil {
		return nil, err
	}
	return FilterValues(raw, defs, audience, includeDeleted), nil
}

// SetEnrollmentValues validates and stores enrollment custom fields.
func (s *Service) SetEnrollmentValues(ctx context.Context, orgID, enrollmentID uuid.UUID, incoming map[string]any) (map[string]any, []ValidationError, error) {
	existing, err := cfrepo.GetEnrollmentCustomFields(ctx, s.pool, orgID, enrollmentID)
	if err != nil {
		return nil, nil, err
	}
	merged, valErrs, err := s.ValidateAndMergeValues(ctx, orgID, cfrepo.EntityEnrollment, incoming, existing)
	if err != nil || len(valErrs) > 0 {
		return nil, valErrs, err
	}
	if err := cfrepo.SetEnrollmentCustomFields(ctx, s.pool, orgID, enrollmentID, merged); err != nil {
		return nil, nil, err
	}
	return merged, nil, nil
}
