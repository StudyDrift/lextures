package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/screenshare"
	"github.com/lextures/lextures/server/internal/screenshare/engine"
)

func sweepAbandonedScreenShareSessions(ctx context.Context, pool *pgxpool.Pool) {
	now := time.Now().UTC()
	ids, err := screenshare.ListAbandonedSessions(ctx, pool, now, engine.IdleMaxAge, 50)
	if err != nil {
		slog.Warn("screenshare abandoned list failed", "err", err)
		return
	}
	for _, id := range ids {
		if err := screenshare.FinaliseAbandoned(ctx, pool, id, now); err != nil {
			slog.Warn("screenshare abandon finalise failed", "session_id", id, "err", err)
			continue
		}
		slog.Info("screenshare session abandoned", "session_id", id)
	}
}
