package background

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
	"github.com/lextures/lextures/server/internal/repos/userimport"
	"github.com/lextures/lextures/server/internal/service/csvimport"
)

// JobTypeUserImport is the registered type for bulk user CSV imports (plan 18.2).
const JobTypeUserImport = "users.import"

// UserImportPayload is the JSON payload for a users.import job.
type UserImportPayload struct {
	ImportJobID uuid.UUID `json:"importJobId"`
}

type userImportHandler struct {
	pool *pgxpool.Pool
	cfg  config.Config
}

// UserImportHandler returns the users.import handler (for inline fallback when the job queue is off).
func UserImportHandler(pool *pgxpool.Pool, cfg config.Config) userImportHandler {
	return userImportHandler{pool: pool, cfg: cfg}
}

func (h userImportHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var p UserImportPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("users.import: bad payload: %w", err)
	}
	if p.ImportJobID == uuid.Nil {
		return fmt.Errorf("users.import: missing importJobId")
	}

	job, err := userimport.GetByID(ctx, h.pool, p.ImportJobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("users.import: job %s not found", p.ImportJobID)
	}
	if job.Status == userimport.StatusComplete {
		return nil
	}
	if job.InputFilePath == nil || *job.InputFilePath == "" {
		return fmt.Errorf("users.import: job %s has no input file", p.ImportJobID)
	}

	if err := userimport.MarkRunning(ctx, h.pool, job.ID); err != nil {
		return err
	}

	f, err := os.Open(*job.InputFilePath)
	if err != nil {
		_ = userimport.Complete(ctx, h.pool, job.ID, userimport.CompleteParams{Status: userimport.StatusFailed})
		return err
	}
	defer func() { _ = f.Close() }()

	parsed, err := csvimport.ParseCSV(f, job.ImportProfile)
	if err != nil {
		_ = userimport.Complete(ctx, h.pool, job.ID, userimport.CompleteParams{Status: userimport.StatusFailed})
		return err
	}

	cursor := job.CursorRow
	result, err := csvimport.Process(ctx, h.pool, csvimport.ProcessParams{
		JobID:         job.ID,
		OrgID:         job.OrgID,
		ActorID:       job.ActorID,
		MergeStrategy: job.MergeStrategy,
		DryRun:        job.DryRun,
		Rows:          parsed.Rows,
		CursorRow:     cursor,
		OnProgress: func(processed, errors int) {
			_ = userimport.UpdateProgress(ctx, h.pool, job.ID, userimport.ProgressUpdate{
				ProcessedRows: cursor + processed,
				ErrorRows:     errors,
				CursorRow:     cursor + processed,
			})
		},
	})
	if err != nil {
		_ = userimport.Complete(ctx, h.pool, job.ID, userimport.CompleteParams{Status: userimport.StatusFailed})
		return err
	}

	var resultPath *string
	if !job.DryRun && len(result.Outcomes) > 0 {
		dir := filepath.Dir(*job.InputFilePath)
		rp := filepath.Join(dir, "result.csv")
		if err := csvimport.WriteResultCSV(rp, result.Outcomes); err != nil {
			slog.Warn("users.import: failed to write result csv", "job_id", job.ID, "error", err)
		} else {
			resultPath = &rp
		}
	}

	_ = os.Remove(*job.InputFilePath)

	status := userimport.StatusComplete
	if err := userimport.Complete(ctx, h.pool, job.ID, userimport.CompleteParams{
		Status:         status,
		ResultFilePath: resultPath,
		Errors:         result.Errors,
		ProcessedRows:  result.ProcessedRows,
		ErrorRows:      result.ErrorRows,
		CreatedCount:   result.CreatedCount,
		UpdatedCount:   result.UpdatedCount,
		Deactivated:    result.DeactivatedCount,
		Skipped:        result.SkippedCount,
	}); err != nil {
		return err
	}

	slog.Info("users.import completed",
		"job_id", job.ID,
		"org_id", job.OrgID,
		"rows_processed", result.ProcessedRows,
		"errors_count", result.ErrorRows,
		"dry_run", job.DryRun,
	)
	return nil
}

// EnqueueUserImport queues a bulk user CSV import job.
func EnqueueUserImport(ctx context.Context, pool *pgxpool.Pool, importJobID uuid.UUID) (uuid.UUID, error) {
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:   JobTypeUserImport,
		Payload:   UserImportPayload{ImportJobID: importJobID},
		Priority:  4,
		UniqueKey: "users.import:" + importJobID.String(),
	})
}

// RegisterUserImportJob registers the users.import handler.
func RegisterUserImportJob(r *Registry, pool *pgxpool.Pool, cfg config.Config) {
	r.Register(JobTypeUserImport, userImportHandler{pool: pool, cfg: cfg})
}
