package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
)

const stuckRunNoProgressTimeout = 30 * time.Minute

func sweepStuckGradingRuns(ctx context.Context, pool *pgxpool.Pool) {
	n, err := gradingagentrepo.ReconcileStuckRuns(ctx, pool, stuckRunNoProgressTimeout)
	if err != nil {
		slog.Warn("grading_agent_reconciler: sweep failed", "err", err)
		return
	}
	if n > 0 {
		slog.Info("grading_agent_reconciler: marked stuck runs failed", "count", n)
	}
}
