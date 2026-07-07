package cli

import (
	"encoding/json"
	"fmt"
)

// APIError is a structured server error envelope.
type APIError struct {
	Message   string `json:"message"`
	ErrorText string `json:"error"`
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
	Status    int    `json:"-"`
}

func (e APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.ErrorText
	}
	if msg == "" {
		msg = "server error"
	}
	if e.RequestID != "" {
		return fmt.Sprintf("server error (%d): %s (request_id=%s)", e.Status, msg, e.RequestID)
	}
	return fmt.Sprintf("server error (%d): %s", e.Status, msg)
}

// ParseAPIError parses an HTTP error body into APIError.
func ParseAPIError(status int, body []byte) APIError {
	var e APIError
	e.Status = status
	_ = json.Unmarshal(body, &e)
	return e
}

// WriteJSONError encodes err for --json stderr output.
func WriteJSONError(enc *json.Encoder, err error, code int) error {
	if api, ok := err.(APIError); ok {
		return enc.Encode(map[string]any{
			"error":      api.Error(),
			"code":       code,
			"request_id": api.RequestID,
		})
	}
	return enc.Encode(map[string]any{
		"error": err.Error(),
		"code":  code,
	})
}