// Package publicapi implements the versioned public REST API helpers (plan 16.1).
package publicapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type cursorPayload struct {
	Offset int `json:"o"`
}

// EncodeCursor returns an opaque pagination cursor for the given row offset.
func EncodeCursor(offset int) string {
	if offset <= 0 {
		return ""
	}
	b, _ := json.Marshal(cursorPayload{Offset: offset})
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor parses a cursor query parameter into a row offset.
func DecodeCursor(cursor string) (int, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor")
	}
	var p cursorPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.Offset < 0 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return p.Offset, nil
}

// ParseLimit returns limit clamped to [1, maxDefault].
func ParseLimit(raw string, defaultLimit, maxLimit int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if defaultLimit <= 0 {
			defaultLimit = 25
		}
		return defaultLimit, nil
	}
	var n int
	if _, err := fmt.Sscanf(raw, "%d", &n); err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid limit")
	}
	if maxLimit > 0 && n > maxLimit {
		n = maxLimit
	}
	return n, nil
}
