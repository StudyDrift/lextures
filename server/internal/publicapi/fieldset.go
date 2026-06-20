package publicapi

import (
	"encoding/json"
	"reflect"
	"strings"
)

// ParseFieldsets reads ?fields[resource]=a,b query params into a map.
func ParseFieldsets(q map[string][]string) map[string]map[string]struct{} {
	out := make(map[string]map[string]struct{})
	for key, vals := range q {
		if !strings.HasPrefix(key, "fields[") || !strings.HasSuffix(key, "]") {
			continue
		}
		resource := strings.TrimSuffix(strings.TrimPrefix(key, "fields["), "]")
		if resource == "" || len(vals) == 0 {
			continue
		}
		fields := make(map[string]struct{})
		for _, part := range strings.Split(vals[0], ",") {
			f := strings.TrimSpace(part)
			if f != "" {
				fields[f] = struct{}{}
			}
		}
		if len(fields) > 0 {
			out[resource] = fields
		}
	}
	return out
}

// ApplyFieldset filters a JSON object map to only the requested fields.
// When fields is nil or empty, the value is returned unchanged.
func ApplyFieldset(v any, fields map[string]struct{}) any {
	if len(fields) == 0 {
		return v
	}
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return v
	}
	out := make(map[string]any, len(fields))
	for k, val := range m {
		if _, ok := fields[k]; ok {
			out[k] = val
		}
	}
	// Always keep id when present so clients can correlate sparse responses.
	if id, ok := m["id"]; ok {
		out["id"] = id
	}
	return out
}

// ApplyFieldsetsToCollection applies per-resource fieldsets to a typed slice.
func ApplyFieldsetsToCollection[T any](items []T, resource string, fieldsets map[string]map[string]struct{}) []any {
	fields, ok := fieldsets[resource]
	if !ok || len(fields) == 0 {
		return ToAnySlice(items)
	}
	out := make([]any, len(items))
	rv := reflect.ValueOf(items)
	for i := 0; i < rv.Len(); i++ {
		out[i] = ApplyFieldset(rv.Index(i).Interface(), fields)
	}
	return out
}
