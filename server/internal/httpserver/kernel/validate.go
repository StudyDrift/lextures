package kernel

import "github.com/lextures/lextures/server/internal/apierr"

// ValidationError is a field-addressable 422 (UX.6 envelope).
type ValidationError struct {
	Message string
	Fields  []apierr.FieldViolation
}

func (e *ValidationError) Error() string {
	if e == nil || e.Message == "" {
		return "Validation failed"
	}
	return e.Message
}

// Field is apierr.FieldViolation for callers that do not want to import apierr.
type Field = apierr.FieldViolation

// FieldError builds one field violation. Path uses the same dot/bracket
// notation the web client already announces.
func FieldError(path, code, message string) Field {
	if code == "" {
		code = "custom"
	}
	if message == "" {
		message = "Enter a valid value for this field."
	}
	return Field{Path: path, Code: code, Message: message}
}

// Validation returns a 422 error with the given summary and fields.
func Validation(message string, fields ...Field) error {
	if message == "" {
		message = "Validation failed"
	}
	if fields == nil {
		fields = []Field{}
	}
	return &ValidationError{Message: message, Fields: fields}
}

// CollectValidation returns nil when fields is empty, otherwise Validation.
func CollectValidation(message string, fields []Field) error {
	if len(fields) == 0 {
		return nil
	}
	return Validation(message, fields...)
}
