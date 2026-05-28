package l10n

import (
	"errors"
	"strings"
	"time"
)

// NormalizeTimezone validates an IANA timezone identifier.
func NormalizeTimezone(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("timezone is required")
	}
	if len(s) > 64 {
		return "", errors.New("timezone identifier is too long")
	}
	if _, err := time.LoadLocation(s); err != nil {
		return "", errors.New("invalid IANA timezone identifier")
	}
	return s, nil
}

// ValidTimezone returns true when raw is a known IANA zone (empty is invalid).
func ValidTimezone(raw string) bool {
	_, err := NormalizeTimezone(raw)
	return err == nil
}
