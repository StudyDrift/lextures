// Package customfields implements validation and visibility filtering for org metadata fields (plan 18.7).
package customfields

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	cfrepo "github.com/lextures/lextures/server/internal/repos/customfields"
)

var (
	keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

	// reservedKeys blocks custom field keys that collide with core CSV/API columns.
	reservedKeys = map[string]struct{}{
		"email": {}, "first_name": {}, "last_name": {}, "role": {}, "external_id": {},
		"display_name": {}, "id": {}, "org_id": {}, "active": {}, "created_at": {},
		"course_code": {}, "title": {}, "status": {}, "term_id": {},
	}
)

// ValidationError describes one invalid custom field value.
type ValidationError struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Key, e.Message)
}

// ValidationErrors is a collection of field validation failures (HTTP 422).
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return "validation failed"
	}
	return v[0].Error()
}

// ViewerLevel is the caller's read access tier for custom field visibility.
type ViewerLevel int

const (
	ViewerStudent ViewerLevel = iota
	ViewerInstructor
	ViewerAdmin
)

// Service coordinates custom field schema and value operations.
type Service struct {
	Pool *pgxpool.Pool
}

// ValidateKey checks a proposed field key slug.
func ValidateKey(key string) error {
	key = strings.TrimSpace(key)
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("key must be snake_case starting with a letter")
	}
	if _, ok := reservedKeys[key]; ok {
		return cfrepo.ErrReservedKey
	}
	return nil
}

// ValidateDefinitionInput validates create/update definition input.
func ValidateDefinitionInput(key, label string, fieldType cfrepo.FieldType, selectOptions []string) ValidationErrors {
	var errs ValidationErrors
	if err := ValidateKey(key); err != nil {
		errs = append(errs, ValidationError{Key: "key", Message: err.Error()})
	}
	if strings.TrimSpace(label) == "" {
		errs = append(errs, ValidationError{Key: "label", Message: "label is required"})
	}
	switch fieldType {
	case cfrepo.FieldText, cfrepo.FieldNumber, cfrepo.FieldBoolean, cfrepo.FieldDate:
	case cfrepo.FieldSelect:
		if len(selectOptions) == 0 {
			errs = append(errs, ValidationError{Key: "selectOptions", Message: "select options are required for select fields"})
		}
	default:
		errs = append(errs, ValidationError{Key: "fieldType", Message: "invalid field type"})
	}
	return errs
}

// ValidateValues checks incoming values against active definitions.
func ValidateValues(defs []cfrepo.Definition, values map[string]any) ValidationErrors {
	var errs ValidationErrors
	activeKeys := make(map[string]cfrepo.Definition, len(defs))
	for _, d := range defs {
		activeKeys[d.Key] = d
	}
	for key, val := range values {
		def, ok := activeKeys[key]
		if !ok {
			errs = append(errs, ValidationError{Key: key, Message: "unknown custom field"})
			continue
		}
		if err := validateOneValue(def, val); err != nil {
			errs = append(errs, ValidationError{Key: key, Message: err.Error()})
		}
	}
	for _, def := range defs {
		if !def.IsRequired {
			continue
		}
		val, ok := values[def.Key]
		if !ok || val == nil || val == "" {
			errs = append(errs, ValidationError{Key: def.Key, Message: "required field is missing"})
		}
	}
	return errs
}

func validateOneValue(def cfrepo.Definition, val any) error {
	if val == nil {
		return nil
	}
	switch def.FieldType {
	case cfrepo.FieldText:
		if _, ok := val.(string); !ok {
			return errors.New("must be a string")
		}
	case cfrepo.FieldNumber:
		switch val.(type) {
		case float64, int, int64:
		default:
			return errors.New("must be a number")
		}
	case cfrepo.FieldBoolean:
		if _, ok := val.(bool); !ok {
			return errors.New("must be a boolean")
		}
	case cfrepo.FieldDate:
		s, ok := val.(string)
		if !ok {
			return errors.New("must be a date string (YYYY-MM-DD)")
		}
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return errors.New("must be a date string (YYYY-MM-DD)")
		}
	case cfrepo.FieldSelect:
		s, ok := val.(string)
		if !ok {
			return errors.New("must be a string")
		}
		found := false
		for _, opt := range def.SelectOptions {
			if opt == s {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("value must be one of: %s", strings.Join(def.SelectOptions, ", "))
		}
	default:
		return errors.New("unsupported field type")
	}
	return nil
}

// FilterByVisibility returns only fields visible to the viewer level.
func FilterByVisibility(defs []cfrepo.Definition, values map[string]any, level ViewerLevel) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	defByKey := make(map[string]cfrepo.Definition, len(defs))
	for _, d := range defs {
		defByKey[d.Key] = d
	}
	out := make(map[string]any)
	for key, val := range values {
		def, ok := defByKey[key]
		if !ok {
			continue
		}
		if visibleTo(def.Visibility, level) {
			out[key] = val
		}
	}
	return out
}

func visibleTo(v cfrepo.Visibility, level ViewerLevel) bool {
	switch v {
	case cfrepo.VisibilityAdminOnly:
		return level >= ViewerAdmin
	case cfrepo.VisibilityInstructor:
		return level >= ViewerInstructor
	case cfrepo.VisibilityStudent:
		return true
	default:
		return false
	}
}

// MergePatch merges patch values into existing custom fields, removing keys set to null.
func MergePatch(existing, patch map[string]any) map[string]any {
	out := make(map[string]any, len(existing)+len(patch))
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range patch {
		if v == nil {
			delete(out, k)
		} else {
			out[k] = normalizeJSONValue(v)
		}
	}
	return out
}

func normalizeJSONValue(v any) any {
	switch n := v.(type) {
	case float64:
		if n == float64(int64(n)) {
			return int64(n)
		}
		return n
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return i
		}
		if f, err := n.Float64(); err == nil {
			return f
		}
		return n.String()
	default:
		return v
	}
}

// ListDefinitions loads active definitions for org+entity.
func (s Service) ListDefinitions(ctx context.Context, orgID uuid.UUID, entityType cfrepo.EntityType) ([]cfrepo.Definition, error) {
	return cfrepo.ListDefinitions(ctx, s.Pool, orgID, entityType)
}

// PatchUserCustomFields validates and merges custom field values on a user.
func (s Service) PatchUserCustomFields(ctx context.Context, orgID, userID uuid.UUID, patch map[string]any) (map[string]any, ValidationErrors, error) {
	defs, err := cfrepo.ListDefinitions(ctx, s.Pool, orgID, cfrepo.EntityUser)
	if err != nil {
		return nil, nil, err
	}
	existing, err := cfrepo.GetUserCustomFields(ctx, s.Pool, userID)
	if err != nil {
		return nil, nil, err
	}
	merged := MergePatch(existing, patch)
	if errs := ValidateValues(defs, merged); len(errs) > 0 {
		return nil, errs, nil
	}
	if err := cfrepo.SetUserCustomFields(ctx, s.Pool, orgID, userID, merged); err != nil {
		return nil, nil, err
	}
	return merged, nil, nil
}

// UserCustomFieldsForViewer returns filtered custom fields for a user.
func (s Service) UserCustomFieldsForViewer(ctx context.Context, orgID, userID uuid.UUID, level ViewerLevel) (map[string]any, error) {
	defs, err := cfrepo.ListDefinitions(ctx, s.Pool, orgID, cfrepo.EntityUser)
	if err != nil {
		return nil, err
	}
	raw, err := cfrepo.GetUserCustomFields(ctx, s.Pool, userID)
	if err != nil {
		return nil, err
	}
	return FilterByVisibility(defs, raw, level), nil
}

// ParseEntityType normalizes entity_type query param.
func ParseEntityType(s string) (cfrepo.EntityType, error) {
	switch cfrepo.EntityType(strings.TrimSpace(s)) {
	case cfrepo.EntityUser, cfrepo.EntityCourse, cfrepo.EntityEnrollment:
		return cfrepo.EntityType(strings.TrimSpace(s)), nil
	default:
		return "", fmt.Errorf("invalid entity_type %q", s)
	}
}

// NormalizeSelectValue coerces CSV string values for validation.
func NormalizeSelectValue(s string) any {
	return strings.TrimSpace(s)
}

// CoerceCSVValue converts a CSV string to the typed value for validation/storage.
func CoerceCSVValue(def cfrepo.Definition, raw string) (any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	switch def.FieldType {
	case cfrepo.FieldText, cfrepo.FieldSelect, cfrepo.FieldDate:
		return raw, nil
	case cfrepo.FieldNumber:
		if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return i, nil
		}
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, errors.New("must be a number")
		}
		return f, nil
	case cfrepo.FieldBoolean:
		switch strings.ToLower(raw) {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no":
			return false, nil
		default:
			return nil, errors.New("must be true or false")
		}
	default:
		return raw, nil
	}
}
