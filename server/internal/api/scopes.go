// Package api defines public API scope strings for personal access keys.
package api

import (
	"sort"
	"strings"
)

// Scope describes a grantable API access scope.
type Scope struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Group       string `json:"group"`
}

// AllScopes returns the fixed enumeration of personal access key scopes.
func AllScopes() []Scope {
	scopes := []Scope{
		{ID: "courses:read", Label: "Read courses", Description: "List and view courses you can access.", Group: "Courses"},
		{ID: "courses:write", Label: "Write courses", Description: "Create and update courses and course content.", Group: "Courses"},
		{ID: "enrollments:read", Label: "Read enrollments", Description: "View course rosters and enrollment data.", Group: "Enrollments"},
		{ID: "enrollments:write", Label: "Write enrollments", Description: "Add, update, or remove enrollments.", Group: "Enrollments"},
		{ID: "assignments:read", Label: "Read assignments", Description: "View assignments and submissions.", Group: "Assignments"},
		{ID: "assignments:write", Label: "Write assignments", Description: "Create and update assignments.", Group: "Assignments"},
		{ID: "grades:read", Label: "Read grades", Description: "View gradebook data and posted grades.", Group: "Grades"},
		{ID: "grades:write", Label: "Write grades", Description: "Post and update grades.", Group: "Grades"},
		{ID: "users:read", Label: "Read users", Description: "View basic user directory data.", Group: "Users"},
		{ID: "feed:read", Label: "Read activity feed", Description: "Read course activity feed events.", Group: "Feed"},
		{ID: "files:read", Label: "Read files", Description: "Download course files you can access.", Group: "Files"},
		{ID: "mcp:connect", Label: "Connect MCP agents", Description: "Allow AI agents to connect via MCP using this key.", Group: "Integrations"},
		{ID: "webhooks:manage", Label: "Manage webhooks", Description: "Create and manage outbound webhook subscriptions.", Group: "Integrations"},
	}
	sort.Slice(scopes, func(i, j int) bool { return scopes[i].ID < scopes[j].ID })
	return scopes
}

// ValidScopeIDs returns a set of known scope ids.
func ValidScopeIDs() map[string]struct{} {
	out := make(map[string]struct{}, len(AllScopes()))
	for _, s := range AllScopes() {
		out[s.ID] = struct{}{}
	}
	return out
}

// NormalizeScopes deduplicates and validates scope ids; returns false if any id is unknown.
func NormalizeScopes(ids []string) ([]string, bool) {
	valid := ValidScopeIDs()
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := valid[id]; !ok {
			return nil, false
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out, true
}
