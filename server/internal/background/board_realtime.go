package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/board"
)

func sweepBoardCompaction(ctx context.Context, pool *pgxpool.Pool) {
	ids, err := board.ListBoardIDsNeedingCompaction(ctx, pool, board.CompactUpdateThreshold, 20)
	if err != nil {
		slog.Warn("board compaction list failed", "err", err)
		return
	}
	for _, id := range ids {
		ok, err := board.CompactBoard(ctx, pool, id)
		if err != nil {
			slog.Warn("board compaction failed", "board_id", id, "err", err)
			continue
		}
		if ok {
			slog.Info("board compacted", "board_id", id)
		}
	}
}

func sweepBoardReconcile(ctx context.Context, pool *pgxpool.Pool) {
	since := time.Now().UTC().Add(-10 * time.Minute)
	ids, err := board.ListBoardIDsForReconcile(ctx, pool, since, 50)
	if err != nil {
		slog.Warn("board reconcile list failed", "err", err)
		return
	}
	for _, id := range ids {
		n, err := board.ReconcileBoard(ctx, pool, id)
		if err != nil {
			slog.Warn("board reconcile failed", "board_id", id, "err", err)
			continue
		}
		if n > 0 {
			slog.Info("board reconciled", "board_id", id, "posts_updated", n)
		}
	}
}
