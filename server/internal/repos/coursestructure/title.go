package coursestructure

import (
	"encoding/json"
	"errors"
	"strings"
)

// ErrAPIErrorPayloadTitle is returned when a structure title is an API error envelope.
// A failed reorder (or any other API error) must be shown to the caller and must not
// become the stored name of a module or item.
var ErrAPIErrorPayloadTitle = errors.New("coursestructure: title is an API error payload")

// IsAPIErrorPayloadTitle reports whether title is a JSON API error envelope
// ({"error":{"code":"...","message":"..."}}), including the raw body of a failed
// course-structure reorder.
func IsAPIErrorPayloadTitle(title string) bool {
	t := strings.TrimSpace(title)
	if !strings.HasPrefix(t, "{") {
		return false
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(t), &body); err != nil {
		return false
	}
	return strings.TrimSpace(body.Error.Code) != "" && strings.TrimSpace(body.Error.Message) != ""
}

// ValidateItemTitle rejects titles that are API error payloads.
func ValidateItemTitle(title string) error {
	if IsAPIErrorPayloadTitle(title) {
		return ErrAPIErrorPayloadTitle
	}
	return nil
}
