package httpserver

import "testing"

func TestCanvasTimestampToDate(t *testing.T) {
	if got := canvasTimestampToDate("2025-01-15T08:00:00Z"); got != "2025-01-15" {
		t.Fatalf("RFC3339: got %q", got)
	}
	if got := canvasTimestampToDate("2025-05-01"); got != "2025-05-01" {
		t.Fatalf("date-only: got %q", got)
	}
	if got := canvasTimestampToDate(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
	if got := canvasTimestampToDate("not-a-date"); got != "" {
		t.Fatalf("invalid: got %q", got)
	}
}
