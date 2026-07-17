package background

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
	"github.com/lextures/lextures/server/internal/service/diplomaissue"
)

// JobTypeDiplomaBatchIssue is the durable queue type for T11 cohort issuance.
const JobTypeDiplomaBatchIssue = "diploma.issue.batch"

// DiplomaBatchPayload identifies one batch to process (or resume).
type DiplomaBatchPayload struct {
	BatchID uuid.UUID `json:"batchId"`
}

type diplomaBatchHandler struct {
	pool *pgxpool.Pool
	cfg  config.Config
}

func (h diplomaBatchHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var p DiplomaBatchPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("diploma.issue.batch: bad payload: %w", err)
	}
	if p.BatchID == uuid.Nil {
		return fmt.Errorf("diploma.issue.batch: missing batchId")
	}
	return diplomaissue.ProcessBatch(ctx, h.pool, h.cfg, p.BatchID)
}

// RegisterDiplomaBatchJob registers the diploma.issue.batch handler.
func RegisterDiplomaBatchJob(r *Registry, pool *pgxpool.Pool, cfg config.Config) {
	if r == nil {
		return
	}
	r.Register(JobTypeDiplomaBatchIssue, diplomaBatchHandler{pool: pool, cfg: cfg})
}

// EnqueueDiplomaBatch queues cohort diploma issuance.
func EnqueueDiplomaBatch(ctx context.Context, pool *pgxpool.Pool, batchID uuid.UUID) (uuid.UUID, error) {
	if pool == nil || batchID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("diploma.issue.batch: missing pool or batch")
	}
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:     JobTypeDiplomaBatchIssue,
		Payload:     DiplomaBatchPayload{BatchID: batchID},
		Priority:    5,
		MaxAttempts: 5,
		UniqueKey:   "diploma-batch:" + batchID.String(),
	})
}
