package banners

import (
	"errors"
	"regexp"
	"strings"
)

var emailLike = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// ValidateMessage rejects empty or overlong messages and heuristic PII (email addresses).
func ValidateMessage(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return errors.New("message is required")
	}
	if len(message) > 500 {
		return errors.New("message must be at most 500 characters")
	}
	if emailLike.MatchString(message) {
		return errors.New("message must not contain email addresses")
	}
	return nil
}

// ParseSeverity normalizes a severity string.
func ParseSeverity(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "info", "":
		return "info", nil
	case "warning":
		return "warning", nil
	case "error":
		return "error", nil
	default:
		return "", errors.New("severity must be info, warning, or error")
	}
}

// ParseScope normalizes a scope string.
func ParseScope(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "global":
		return "global", nil
	case "org":
		return "org", nil
	default:
		return "", errors.New("scope must be global or org")
	}
}
