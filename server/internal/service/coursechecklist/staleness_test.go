package coursechecklist

import (
	"testing"
	"time"

	"github.com/google/uuid"
	ccrepo "github.com/lextures/lextures/server/internal/repos/coursechecklist"
)

func TestIsSnapshotStale_TruthTable(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	base := &ccrepo.Snapshot{
		CourseID:       uuid.New(),
		ComputedAt:     now.Add(-5 * time.Minute),
		EngineVersion:  1,
		CatalogVersion: "abc",
	}
	fresh := ccrepo.MutationFreshness{CourseUpdatedAt: now.Add(-1 * time.Hour)}

	cases := []struct {
		name    string
		snap    *ccrepo.Snapshot
		engine  int
		catalog string
		ttl     time.Duration
		mut     ccrepo.MutationFreshness
		want    bool
	}{
		{"nil", nil, 1, "abc", SnapshotTTLDefault, fresh, true},
		{"engine bump", base, 2, "abc", SnapshotTTLDefault, fresh, true},
		{"catalog bump", base, 1, "xyz", SnapshotTTLDefault, fresh, true},
		{"ttl expired", base, 1, "abc", 2 * time.Minute, fresh, true},
		{"course mutated", base, 1, "abc", SnapshotTTLDefault, ccrepo.MutationFreshness{CourseUpdatedAt: now.Add(-1 * time.Minute)}, true},
		{"warm hit", base, 1, "abc", SnapshotTTLDefault, fresh, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsSnapshotStale(tc.snap, tc.engine, tc.catalog, tc.ttl, tc.mut, now)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
