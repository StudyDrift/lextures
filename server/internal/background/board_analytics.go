package background

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/storageobjects"
	"github.com/lextures/lextures/server/internal/scheduler"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// RegisterBoardLifecycleJobs registers analytics rollup and retention handlers (VC.10).
func RegisterBoardLifecycleJobs(r *Registry, pool *pgxpool.Pool, storage filestorage.Driver) {
	if r == nil || pool == nil {
		return
	}
	r.Register(scheduler.JobTypeBoardAnalyticsRollup, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		n, err := board.RefreshAnalyticsDaily(ctx, pool, nil, time.Now().UTC())
		if err != nil {
			telemetry.RecordBusinessEvent("board.analytics.rollup_failed")
			return err
		}
		slog.Info("scheduled.board_analytics_rollup", "rows", n)
		telemetry.RecordBusinessEvent("board.analytics.rollup_ok")
		return nil
	}))

	r.Register(scheduler.JobTypeBoardExportRetention, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		cutoff := time.Now().UTC().AddDate(0, 0, -board.DefaultExportRetentionDays)
		expired, err := board.ListExpiredExports(ctx, pool, cutoff)
		if err != nil {
			return err
		}
		var deleted int
		for _, e := range expired {
			if storage != nil && e.StorageKey != "" {
				_ = storage.DeleteObject(ctx, e.StorageKey)
			}
			_ = storageobjects.SoftDeleteByObjectKey(ctx, pool, e.StorageKey)
			if err := board.ClearExportStorageKey(ctx, pool, e.ID); err != nil {
				return err
			}
			deleted++
		}
		if deleted > 0 {
			slog.Info("scheduled.board_export_retention", "deleted", deleted)
		}
		telemetry.RecordBusinessEvent("board.export.retention_ok")
		return nil
	}))

	r.Register(scheduler.JobTypeBoardContentRetention, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		cutoff := time.Now().UTC().AddDate(0, 0, -board.DefaultArchivedBoardRetentionDays)
		boards, err := board.ListArchivedBoardsForPurge(ctx, pool, cutoff, 50)
		if err != nil {
			return err
		}
		var purged int
		for _, b := range boards {
			for _, key := range b.StorageKeys {
				if storage != nil && key != "" {
					_ = storage.DeleteObject(ctx, key)
				}
				_ = storageobjects.SoftDeleteByObjectKey(ctx, pool, key)
			}
			if err := board.PurgeBoard(ctx, pool, b.ID); err != nil {
				return err
			}
			purged++
		}
		if purged > 0 {
			slog.Info("scheduled.board_content_retention", "purged", purged)
		}
		telemetry.RecordBusinessEvent("board.content.retention_ok")
		return nil
	}))
}
