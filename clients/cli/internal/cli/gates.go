package cli

import (
	"fmt"
	"net/http"
	"strings"
)

const sensitiveExportWarning = `WARNING: This operation may export sensitive personal or academic data (FERPA); re-run with --yes to confirm`

// RequireYes refuses destructive or sensitive operations without confirmation.
func RequireYes(confirmed bool, context string) error {
	if confirmed {
		return nil
	}
	msg := sensitiveExportWarning
	if context != "" {
		msg = fmt.Sprintf("WARNING: %s; re-run with --yes to confirm", context)
	}
	return fmt.Errorf("%s", msg)
}

// RequireForce is an alias for destructive ops that accept --force.
func RequireForce(confirmed bool, message string) error {
	if confirmed {
		return nil
	}
	if message == "" {
		message = "re-run with --force to confirm this destructive operation"
	}
	return fmt.Errorf("%s", message)
}

// SetIdempotencyKey sets the Idempotency-Key header when non-empty.
func SetIdempotencyKey(req *http.Request, key string) {
	key = strings.TrimSpace(key)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
}