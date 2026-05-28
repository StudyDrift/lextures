package logging

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var uuidInPath = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// pathSegmentLabels maps URL path prefixes to redaction placeholder names (plan 10.14 FR-5).
var pathSegmentLabels = map[string]string{
	"users":       "user_id",
	"user":        "user_id",
	"orgs":        "org_id",
	"org":         "org_id",
	"organizations": "org_id",
	"courses":     "course_id",
	"course":      "course_id",
	"enrollments": "enrollment_id",
	"enrollment":  "enrollment_id",
}

// RedactRequestPath masks UUID path segments (e.g. /users/{uuid} → /users/[REDACTED:user_id]).
func RedactRequestPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "" {
			continue
		}
		if _, err := uuid.Parse(part); err != nil {
			continue
		}
		label := "id"
		if i > 0 {
			if l, ok := pathSegmentLabels[strings.ToLower(parts[i-1])]; ok {
				label = l
			}
		}
		parts[i] = "[REDACTED:" + label + "]"
	}
	// Catch any remaining UUIDs not adjacent to a known prefix.
	out := strings.Join(parts, "/")
	return uuidInPath.ReplaceAllStringFunc(out, func(_ string) string {
		return "[REDACTED:id]"
	})
}
