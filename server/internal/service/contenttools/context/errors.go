package context

import "errors"

// Typed errors surfaced to tool actions / HTTP (FR-12, FR-14).
var (
	ErrSSRFBlocked          = errors.New("contenttools/context: blocked private network")
	ErrRobotsDisallowed     = errors.New("contenttools/context: robots.txt disallows fetch")
	ErrTooLarge             = errors.New("contenttools/context: response exceeds size cap")
	ErrUnsupportedType      = errors.New("contenttools/context: content type not ingestible")
	ErrHostBreakerOpen      = errors.New("contenttools/context: host circuit breaker open")
	ErrIngestDisabled       = errors.New("contenttools/context: link ingestion disabled")
	ErrHostNotAllowlisted   = errors.New("contenttools/context: host not on allowlist")
	ErrKillSwitch           = errors.New("contenttools/context: link ingest kill-switch active")
	ErrBudgetRequestTokens  = errors.New("contenttools/context: per-request token budget exceeded")
	ErrBudgetDailyUserCalls = errors.New("contenttools/context: per-user daily call budget exceeded")
	ErrBudgetCourseMonthly  = errors.New("contenttools/context: per-course monthly spend budget exceeded")
	ErrAIGatewayDenied      = errors.New("contenttools/context: AI gateway denied")
	ErrProviderUnavailable  = errors.New("contenttools/context: provider unavailable")
	ErrInstanceNotFound     = errors.New("contenttools/context: instance not found")
)

// BudgetError names which limit was hit (FR-14, AC-7).
type BudgetError struct {
	Level   string // request | daily_user | monthly_course
	Message string
}

func (e *BudgetError) Error() string {
	if e == nil {
		return "contenttools/context: budget exceeded"
	}
	return e.Message
}

func (e *BudgetError) Unwrap() error {
	switch e.Level {
	case "request":
		return ErrBudgetRequestTokens
	case "daily_user":
		return ErrBudgetDailyUserCalls
	case "monthly_course":
		return ErrBudgetCourseMonthly
	default:
		return ErrBudgetRequestTokens
	}
}

// GatewayError wraps an aigateway denial with a user-friendly message (FR-12, AC-6).
type GatewayError struct {
	Reason  string
	Message string
}

func (e *GatewayError) Error() string {
	if e == nil || e.Message == "" {
		return ErrAIGatewayDenied.Error()
	}
	return e.Message
}

func (e *GatewayError) Unwrap() error { return ErrAIGatewayDenied }
