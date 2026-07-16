package aiusage

import (
	"strings"
	"testing"
	"time"
)

func TestFilters_whereSQL_provider(t *testing.T) {
	from := time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	where, args := (Filters{From: from, To: to, Provider: "anthropic"}).whereSQL("")
	if !strings.Contains(where, "provider = $3") {
		t.Fatalf("expected provider clause: %q", where)
	}
	if len(args) != 3 || args[2] != "anthropic" {
		t.Fatalf("args: %#v", args)
	}
}

func TestFilters_whereSQL_qualifiedForJoin(t *testing.T) {
	from := time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	where, args := (Filters{From: from, To: to}).whereSQL("l")
	if !strings.Contains(where, "l.created_at >= $1") {
		t.Fatalf("expected qualified created_at: %q", where)
	}
	if !strings.Contains(where, "l.succeeded = TRUE") {
		t.Fatalf("expected qualified succeeded: %q", where)
	}
	if len(args) != 2 {
		t.Fatalf("args: %d", len(args))
	}
}

func TestFilters_whereSQL_unqualifiedForSingleTable(t *testing.T) {
	from := time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	where, _ := (Filters{From: from, To: to}).whereSQL("")
	if strings.Contains(where, "l.") {
		t.Fatalf("unexpected alias in single-table where: %q", where)
	}
	if !strings.Contains(where, "created_at >= $1") {
		t.Fatalf("expected bare created_at: %q", where)
	}
}