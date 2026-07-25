package background

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/notifevents"
	"github.com/lextures/lextures/server/internal/scheduler"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

// RegisterAdaptiveContentEffectivenessJobs registers the daily ACE effectiveness refresh (AC.7).
func RegisterAdaptiveContentEffectivenessJobs(r *Registry, pool *pgxpool.Pool, cfgSrc ConfigSource, hub *notifevents.Hub) {
	if r == nil || pool == nil {
		return
	}
	r.Register(scheduler.JobTypeAdaptiveContentEffectiveness, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		cfg := config.Config{}
		if cfgSrc != nil {
			cfg = cfgSrc.Config()
		}
		notify := &acsvc.EffectivenessNotifyDeps{Pool: pool, Config: cfg, SSEHub: hub}
		n, err := acsvc.RefreshAll(ctx, pool, notify)
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.adaptive_content_effectiveness", "units", n)
		}
		return nil
	}))
}
