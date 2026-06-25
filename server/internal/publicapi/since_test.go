package publicapi

import (
	"testing"
	"time"
)

func TestParseSinceTime(t *testing.T) {
	t.Parallel()
	ts, err := ParseSinceTime(map[string][]string{"since": {"2026-01-15T12:00:00Z"}})
	if err != nil || ts == nil {
		t.Fatalf("err %v ts %v", err, ts)
	}
	if ts.UTC().Format(time.RFC3339) != "2026-01-15T12:00:00Z" {
		t.Fatalf("got %s", ts)
	}
}

func TestFilterBySince(t *testing.T) {
	t.Parallel()
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	items := []GradeResource{
		{CourseID: "a", UpdatedAt: &t1},
		{CourseID: "b", UpdatedAt: nil},
	}
	out := FilterBySince(items, &since, func(g GradeResource) *time.Time { return g.UpdatedAt })
	if len(out) != 1 || out[0].CourseID != "a" {
		t.Fatalf("got %+v", out)
	}
}
