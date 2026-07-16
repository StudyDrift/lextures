package aiprovider

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorType classifies provider failures for metrics and Test Connection (AP.8).
type ErrorType string

const (
	ErrorTypeAuth   ErrorType = "auth"
	ErrorTypeQuota  ErrorType = "quota"
	ErrorTypeConfig ErrorType = "config"
	ErrorTypeServer ErrorType = "server"
	ErrorTypeOther  ErrorType = "other"
)

// ProviderError normalizes provider HTTP failures for fallback decisions (FR-7).
type ProviderError struct {
	Provider   ProviderName
	StatusCode int
	Message    string
	Type       ErrorType
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

// IsRetryable reports whether the error qualifies for fallback retry (5xx / server).
func IsRetryable(err error) bool {
	var pe *ProviderError
	if errors.As(err, &pe) {
		if pe.Type == ErrorTypeAuth || pe.Type == ErrorTypeConfig || pe.Type == ErrorTypeQuota {
			return false
		}
		return pe.StatusCode >= 500 && pe.StatusCode < 600
	}
	return false
}

// ClassifyError returns the ErrorType for metrics / HTTP mapping.
func ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeOther
	}
	var pe *ProviderError
	if errors.As(err, &pe) && pe.Type != "" {
		return pe.Type
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not configured"),
		strings.Contains(msg, "requires "),
		strings.Contains(msg, "unsupported auth_mode"),
		strings.Contains(msg, "invalid auth"):
		return ErrorTypeConfig
	case strings.Contains(msg, "unauthorized"),
		strings.Contains(msg, "forbidden"),
		strings.Contains(msg, "access denied"),
		strings.Contains(msg, "authentication"),
		strings.Contains(msg, "invalid_api_key"),
		strings.Contains(msg, "permission"):
		return ErrorTypeAuth
	case strings.Contains(msg, "rate limit"),
		strings.Contains(msg, "quota"),
		strings.Contains(msg, "too many requests"):
		return ErrorTypeQuota
	default:
		return ErrorTypeOther
	}
}

func newProviderError(provider ProviderName, status int, msg string) *ProviderError {
	return &ProviderError{
		Provider:   provider,
		StatusCode: status,
		Message:    msg,
		Type:       classifyHTTPStatus(status),
	}
}

func newConfigError(provider ProviderName, msg string) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Message:  msg,
		Type:     ErrorTypeConfig,
	}
}

func newAuthError(provider ProviderName, status int, msg string) *ProviderError {
	return &ProviderError{
		Provider:   provider,
		StatusCode: status,
		Message:    msg,
		Type:       ErrorTypeAuth,
	}
}

func classifyHTTPStatus(status int) ErrorType {
	switch {
	case status == 401 || status == 403:
		return ErrorTypeAuth
	case status == 429:
		return ErrorTypeQuota
	case status == 400 || status == 404 || status == 422:
		return ErrorTypeConfig
	case status >= 500 && status < 600:
		return ErrorTypeServer
	default:
		return ErrorTypeOther
	}
}
