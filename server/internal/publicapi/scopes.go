package publicapi

import "slices"

// HasScope reports whether required is granted in scopes.
func HasScope(scopes []string, required string) bool {
	return slices.Contains(scopes, required)
}
