// Package kernel is the TD.7 handler toolkit: typed JSON I/O, composable
// authorisation guards, structured validation, and a central error mapper.
//
// Adoption is incremental and opt-in. Hand-rolled handlers that call
// apierr.WriteJSON continue to work; new handlers should use GET/POST/PUT/
// PATCH/DELETE so decode, guards, and error mapping stay in one place.
//
// # Error envelope
//
// Mapped errors write the legacy {error:{code,message}} body via apierr
// except validation failures, which use the UX.6 validation_failed envelope
// (HTTP 422) the web client already parses.
//
// # Error-mapping table
//
//	pgx.ErrNoRows                         → 404 NOT_FOUND
//	PostgreSQL integrity constraint (23*) → 409 CONFLICT
//	context.Canceled                      → 400 INVALID_INPUT
//	context.DeadlineExceeded              → 503 SERVICE_UNAVAILABLE
//	*http.MaxBytesError                   → 413 INVALID_INPUT
//	*ValidationError                      → 422 validation_failed
//	*Error (explicit)                     → as specified
//	ErrAINotConfigured                    → 503 AI_NOT_CONFIGURED
//	ErrAIRateLimited                      → 503 SERVICE_UNAVAILABLE + Retry-After
//	ErrAIBudgetExceeded                   → 402 PAYMENT_REQUIRED
//	ErrAIContentFiltered                  → 422 INVALID_INPUT
//	anything else                         → 500 INTERNAL (safe message; cause logged)
//
// Unmapped errors are never silently downgraded to 4xx. Internal messages,
// SQL text, and stack traces are not written to the client.
//
// # Guards
//
// Guards wrap the existing requireCourseAccess / meUserID family so converted
// handlers keep those exact status codes and messages. A nil guard fails
// closed (requires authentication) and is counted as unguarded (FR-6, FR-11).
// Public routes must pass Public() explicitly.
//
// Request/response type parameters on GET/POST/… are the hook for future
// OpenAPI generation (TD.3 / FR-12); this package does not emit a spec.
package kernel
