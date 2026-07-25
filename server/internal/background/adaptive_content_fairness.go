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

// RegisterAdaptiveContentFairnessJobs registers the daily ACE fairness audit (AC.8).
func RegisterAdaptiveContentFairnessJobs(r *Registry, pool *pgxpool.Pool, cfgSrc ConfigSource, hub *notifevents.Hub) {
	if r == nil || pool == nil {
		return
	}
	r.Register(scheduler.JobTypeAdaptiveContentFairness, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		cfg := config.Config{}
		if cfgSrc != nil {
			cfg = cfgSrc.Config()
		}
		notify := &acsvc.FairnessNotifyDeps{Pool: pool, Config: cfg, SSEHub: hub}
		n, err := acsvc.RefreshFairnessAll(ctx, pool, notify)
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.adaptive_content_fairness", "cells", n)
		}
		return nil
	}))
}
