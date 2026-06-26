package publicapi

// HasScope reports whether granted scopes include the required scope id.
func HasScope(granted []string, required string) bool {
	for _, s := range granted {
		if s == required {
			return true
		}
	}
	return false
}
