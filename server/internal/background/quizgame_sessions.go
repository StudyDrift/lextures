package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/quizgame/engine"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
)

const quizLobbyMaxAge = 2 * time.Hour

func sweepAbandonedQuizGames(ctx context.Context, pool *pgxpool.Pool) {
	now := time.Now().UTC()
	ids, err := quizgame.ListAbandonedSessions(ctx, pool, now, engine.HostGraceDefault, quizLobbyMaxAge, 50)
	if err != nil {
		slog.Warn("quizgame abandoned list failed", "err", err)
		return
	}
	for _, id := range ids {
		if err := quizgame.FinaliseAbandoned(ctx, pool, id, now); err != nil {
			slog.Warn("quizgame abandon finalise failed", "session_id", id, "err", err)
			continue
		}
		slog.Info("quizgame session abandoned", "session_id", id)
	}
}
