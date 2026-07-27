package contenttools

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

const migrationBatchSize = 200

// RunMigrationJob executes (or dry-runs) an eager migration job with resume support (FR-7 / AC-5).
func RunMigrationJob(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID) error {
	job, err := ctrepo.GetMigrationJob(ctx, pool, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("migration job %s not found", jobID)
	}
	if job.Status == "succeeded" || job.Status == "cancelled" {
		return nil
	}
	table := DefaultMigrations().Get(job.ToolID)
	if table == nil {
		msg := fmt.Sprintf("no migration table for tool %s", job.ToolID)
		job.Status = "failed"
		job.Error = &msg
		now := time.Now().UTC()
		job.FinishedAt = &now
		_ = ctrepo.UpdateMigrationJobProgress(ctx, pool, job)
		return fmt.Errorf("%s", msg)
	}

	job.Status = "running"
	if job.TotalDocs == 0 {
		n, err := ctrepo.CountStatesForMigration(ctx, pool, job.ToolID, job.FromVersion, job.ToVersion)
		if err != nil {
			return err
		}
		job.TotalDocs = n
	}
	if err := ctrepo.UpdateMigrationJobProgress(ctx, pool, job); err != nil {
		return err
	}

	var cursor *uuid.UUID = job.CursorStateID
	for {
		batch, err := ctrepo.ListStatesForMigration(ctx, pool, job.ToolID, job.FromVersion, job.ToVersion, cursor, migrationBatchSize)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}
		for _, row := range batch {
			id := row.ID
			cursor = &id
			res := ApplyStateMigrations(table, row.StateSchemaVersion, row.StateJSON)
			if res.Quarantine {
				job.FailedDocs++
				IncMigrationDocs(job.ToolID, fmt.Sprintf("%d", row.StateSchemaVersion), fmt.Sprintf("%d", job.ToVersion), "fail")
				if !job.DryRun {
					_, _ = ctrepo.InsertQuarantine(ctx, pool, row.ID, job.ToolID, row.StateSchemaVersion, job.ToVersion, res.Error.Error(), row.StateJSON)
				}
				continue
			}
			if res.Unchanged {
				job.MigratedDocs++
				continue
			}
			if !job.DryRun {
				if err := ctrepo.PersistMigratedState(ctx, pool, row.ID, res.Doc, res.ToVersion); err != nil {
					job.FailedDocs++
					IncMigrationDocs(job.ToolID, fmt.Sprintf("%d", row.StateSchemaVersion), fmt.Sprintf("%d", job.ToVersion), "fail")
					continue
				}
			}
			job.MigratedDocs++
			IncMigrationDocs(job.ToolID, fmt.Sprintf("%d", res.FromVersion), fmt.Sprintf("%d", res.ToVersion), "ok")
		}
		job.CursorStateID = cursor
		if err := ctrepo.UpdateMigrationJobProgress(ctx, pool, job); err != nil {
			return err
		}
		if len(batch) < migrationBatchSize {
			break
		}
	}

	job.Status = "succeeded"
	now := time.Now().UTC()
	job.FinishedAt = &now
	if err := ctrepo.UpdateMigrationJobProgress(ctx, pool, job); err != nil {
		return err
	}
	slog.Info("contenttools.migration_job_done",
		"jobId", job.ID, "toolId", job.ToolID, "dryRun", job.DryRun,
		"migrated", job.MigratedDocs, "failed", job.FailedDocs, "total", job.TotalDocs)
	return nil
}
