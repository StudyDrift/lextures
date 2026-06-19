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

// HasAnyScope reports whether any of the required scopes is granted.
func HasAnyScope(granted, required []string) bool {
	for _, req := range required {
		if HasScope(granted, req) {
			return true
		}
	}
	return false
}
