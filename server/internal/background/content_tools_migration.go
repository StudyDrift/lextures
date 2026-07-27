package background

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

// JobTypeContentToolMigration is the durable queue type for eager CT.5 migrations.
const JobTypeContentToolMigration = "content_tool.migration"

type contentToolMigrationPayload struct {
	JobID uuid.UUID `json:"jobId"`
}

// RegisterContentToolsMigrationJobs registers the CT.5 eager migration worker.
func RegisterContentToolsMigrationJobs(r *Registry, pool *pgxpool.Pool) {
	if r == nil || pool == nil {
		return
	}
	r.Register(JobTypeContentToolMigration, HandlerFunc(func(ctx context.Context, raw json.RawMessage) error {
		var p contentToolMigrationPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("content_tool.migration: bad payload: %w", err)
		}
		if p.JobID == uuid.Nil {
			return fmt.Errorf("content_tool.migration: missing jobId")
		}
		return ctsvc.RunMigrationJob(ctx, pool, p.JobID)
	}))
}
