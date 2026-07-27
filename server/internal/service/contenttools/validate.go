package contenttools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// FieldError is one JSON Pointer path failure.
type FieldError struct {
	Path    string
	Message string
}

// ConfigValidationError is returned when config_json fails schema validation (FR-6).
type ConfigValidationError struct {
	Errors []FieldError
}

func (e *ConfigValidationError) Error() string {
	if e == nil || len(e.Errors) == 0 {
		return "config validation failed"
	}
	return fmt.Sprintf("config validation failed: %s (%s)", e.Errors[0].Path, e.Errors[0].Message)
}

// ValidateConfigJSON validates raw config against the manifest configSchema.
func ValidateConfigJSON(m *CompiledManifest, raw json.RawMessage) error {
	if m == nil || m.ConfigCompiled == nil {
		return fmt.Errorf("missing compiled config schema")
	}
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if len(raw) > DefaultMaxConfigBytes {
		return ErrConfigTooLarge
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return &ConfigValidationError{Errors: []FieldError{{Path: "", Message: "config must be valid JSON object"}}}
	}
	if err := m.ConfigCompiled.Validate(doc); err != nil {
		return &ConfigValidationError{Errors: schemaErrors(err)}
	}
	return nil
}

// ValidateStateJSON validates raw state against the manifest stateSchema and size budget.
func ValidateStateJSON(m *CompiledManifest, raw json.RawMessage) error {
	if m == nil || m.StateCompiled == nil {
		return fmt.Errorf("missing compiled state schema")
	}
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	maxBytes := EffectiveMaxStateBytes(m)
	if len(raw) > maxBytes {
		return ErrStateTooLarge
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return &ConfigValidationError{Errors: []FieldError{{Path: "", Message: "state must be valid JSON object"}}}
	}
	if err := m.StateCompiled.Validate(doc); err != nil {
		return &ConfigValidationError{Errors: schemaErrors(err)}
	}
	return nil
}

func schemaErrors(err error) []FieldError {
	if err == nil {
		return nil
	}
	var out []FieldError
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		flattenVE(ve, &out)
		if len(out) > 0 {
			return out
		}
	}
	return []FieldError{{Path: "", Message: err.Error()}}
}

func flattenVE(ve *jsonschema.ValidationError, out *[]FieldError) {
	if ve == nil {
		return
	}
	if len(ve.Causes) == 0 {
		appendLeafErrors(ve, out)
		return
	}
	for _, c := range ve.Causes {
		flattenVE(c, out)
	}
}

func appendLeafErrors(ve *jsonschema.ValidationError, out *[]FieldError) {
	base := instancePath(ve)
	msg := ve.Error()
	switch k := ve.ErrorKind.(type) {
	case *kind.Required:
		for _, prop := range k.Missing {
			path := base + "/" + escapeJSONPointer(prop)
			*out = append(*out, FieldError{Path: path, Message: msg})
		}
		if len(k.Missing) > 0 {
			return
		}
	}
	*out = append(*out, FieldError{Path: base, Message: msg})
}

func instancePath(ve *jsonschema.ValidationError) string {
	if ve == nil {
		return ""
	}
	// jsonschema v6 exposes InstanceLocation as []string tokens.
	loc := ve.InstanceLocation
	if len(loc) == 0 {
		return ""
	}
	var b strings.Builder
	for _, tok := range loc {
		b.WriteByte('/')
		b.WriteString(escapeJSONPointer(tok))
	}
	return b.String()
}

func escapeJSONPointer(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

// ValidateHostKindShape checks FR-5 syllabus vs structure_item_id shape.
func ValidateHostKindShape(hostKind string, structureItemIDPresent bool) error {
	if _, ok := AllowedHostKinds[hostKind]; !ok {
		return ErrInvalidHostKind
	}
	if hostKind == "syllabus" {
		if structureItemIDPresent {
			return ErrStructureItemForbidden
		}
		return nil
	}
	if !structureItemIDPresent {
		return ErrStructureItemRequired
	}
	return nil
}

// ToolAllowedByAllowlist returns true when allowlist is empty (all) or contains toolID.
func ToolAllowedByAllowlist(allowed []string, toolID string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, id := range allowed {
		if id == toolID {
			return true
		}
	}
	return false
}
