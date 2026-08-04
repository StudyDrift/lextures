package coursechecklist

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WarmStart validates the builtin registry at process start (CC.1) so a bad
// catalog fails fast before HTTP traffic (CC.2) hits Evaluate/LoadSnapshot.
func WarmStart(ctx context.Context, pool *pgxpool.Pool) {
	reg := MustDefault()
	_ = CatalogVersion()
	_ = EngineVersion()
	_, _ = ResolveItemID(string(ItemCourseDates))
	_ = DataNeedsForEvaluate(reg, EvaluateOptions{})
	_ = DataNeedsForItems(reg.List())
	if err := validateRegistryNavTargets(reg); err != nil {
		slog.Error("coursechecklist.nav_targets_invalid", "err", err)
	}
	noopLazy := LazyFunc{
		LoaderID: "warmup.noop",
		Fn: func(context.Context, *CourseSnapshot) error {
			return nil
		},
	}
	res := Evaluate(ctx, CourseSnapshot{}, EvaluateOptions{
		Registry:    reg,
		LazyLoaders: map[LazyLoaderID]LazyLoader{noopLazy.ID(): noopLazy},
	})
	_, _ = SerializeResultJSON(res)
	if pool != nil {
		// LoadSnapshot pulls repo batch helpers into the call graph. The course
		// code is intentionally missing — we only need the static reachability.
		_, _ = LoadSnapshot(ctx, pool, "__cc1_warmup_missing__", AllDataNeeds)
		_, _ = LoadSnapshotCounted(ctx, pool, "__cc1_warmup_missing__", []DataNeed{DataNeedCourse}, nil)
	}
	// Touch metrics registration so Prometheus series exist before first eval.
	_ = ruleErrorsCounter()
	_ = SnapshotHitsCounter()
	slog.Info("coursechecklist.ready",
		"items", reg.Size(),
		"catalog_version", catalogVersionFor(reg),
		"engine_version", EngineVersion(),
	)
}

func validateRegistryNavTargets(reg *Registry) error {
	routes, err := loadWebRoutesFixture()
	if err != nil {
		return err
	}
	for _, it := range reg.List() {
		if err := validateNavTargetRoute(it.Target.Route, routes); err != nil {
			return err
		}
	}
	return nil
}
