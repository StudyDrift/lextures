package coursechecklist

import (
	"time"

	ccrepo "github.com/lextures/lextures/server/internal/repos/coursechecklist"
)

// SnapshotTTLDefault is the default CHECKLIST_SNAPSHOT_TTL (FR-9).
const SnapshotTTLDefault = 15 * time.Minute

// IsSnapshotStale reports whether a stored snapshot must be recomputed (FR-9).
func IsSnapshotStale(
	snap *ccrepo.Snapshot,
	engineVersion int,
	catalogVersion string,
	ttl time.Duration,
	freshness ccrepo.MutationFreshness,
	now time.Time,
) bool {
	if snap == nil {
		return true
	}
	if snap.EngineVersion != engineVersion {
		return true
	}
	if snap.CatalogVersion != catalogVersion {
		return true
	}
	if ttl <= 0 {
		ttl = SnapshotTTLDefault
	}
	if now.Sub(snap.ComputedAt) > ttl {
		return true
	}
	if freshness.LatestMutation().After(snap.ComputedAt) {
		return true
	}
	return false
}
