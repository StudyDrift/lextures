package aiprovider

import (
	"errors"
	"fmt"
)

// ProviderError normalizes provider HTTP failures for fallback decisions (FR-7).
type ProviderError struct {
	Provider   ProviderName
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "aiprovider: provider error"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s: status %d: %s", e.Provider, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Provider, e.Message)
}

// IsRetryable reports whether the error qualifies for fallback retry (5xx).
func IsRetryable(err error) bool {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.StatusCode >= 500 && pe.StatusCode < 600
	}
	return false
}

func newProviderError(provider ProviderName, status int, msg string) *ProviderError {
	return &ProviderError{Provider: provider, StatusCode: status, Message: msg}
}