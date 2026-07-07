package cli

import (
	"fmt"
	"strings"
)

// RowError describes a single failed row in a bulk operation.
type RowError struct {
	Index   int
	ID      string
	Message string
}

// BulkSummary aggregates bulk operation results.
type BulkSummary struct {
	Created int
	Updated int
	Skipped int
	Failed  int
	Errors  []RowError
}

// ExitCode returns 0 if no failures, 2 if any failed (unless continueOnError).
func (s BulkSummary) ExitCode(continueOnError bool) int {
	if s.Failed > 0 && !continueOnError {
		return 2
	}
	return 0
}

// RenderHuman prints a human-readable bulk summary.
func (s BulkSummary) RenderHuman() string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "created=%d updated=%d skipped=%d failed=%d\n", s.Created, s.Updated, s.Skipped, s.Failed)
	for _, e := range s.Errors {
		_, _ = fmt.Fprintf(&b, "  row %d", e.Index)
		if e.ID != "" {
			_, _ = fmt.Fprintf(&b, " (%s)", e.ID)
		}
		_, _ = fmt.Fprintf(&b, ": %s\n", e.Message)
	}
	return b.String()
}