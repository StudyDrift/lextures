package publicapi

import (
	"strings"
)

// ParseSparseFields reads ?fields[resource]=a,b from the query string.
func ParseSparseFields(q map[string][]string, resource string) map[string]struct{} {
	key := "fields[" + resource + "]"
	raw := strings.TrimSpace(firstVal(q[key]))
	if raw == "" {
		return nil
	}
	out := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		f := strings.TrimSpace(part)
		if f != "" {
			out[f] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// FilterObject keeps only allowed keys when fields is non-nil.
func FilterObject(fields map[string]struct{}, obj map[string]any) map[string]any {
	if fields == nil {
		return obj
	}
	out := make(map[string]any, len(fields))
	for k, v := range obj {
		if _, ok := fields[k]; ok {
			out[k] = v
		}
	}
	// Always keep id when present.
	if id, ok := obj["id"]; ok {
		out["id"] = id
	}
	return out
}

func firstVal(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}
