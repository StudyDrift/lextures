package learnerprofile

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/jobqueue"
)

const (
	// JobTypeIncremental is the durable queue job for one-user facet recompute.
	JobTypeIncremental = "learner_profile.recompute_incremental"
	// JobTypeFull is the durable queue job for nightly full recompute.
	JobTypeFull = "learner_profile.recompute_all"
)

// IncrementalPayload is the JSON payload for incremental recompute jobs.
type IncrementalPayload struct {
	UserID uuid.UUID `json:"userId"`
	Facets []string  `json:"facets,omitempty"`
}

// EnqueueIncremental queues a debounced incremental recompute for one user.
func EnqueueIncremental(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, facets ...string) (uuid.UUID, error) {
	key := fmt.Sprintf("learner_profile:%s", userID.String())
	if len(facets) > 0 {
		key = fmt.Sprintf("%s:%s", key, facets[0])
	}
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:   JobTypeIncremental,
		Payload:   IncrementalPayload{UserID: userID, Facets: facets},
		Priority:  6,
		UniqueKey: key,
	})
}

// EnqueueFull queues a nightly full recompute job.
func EnqueueFull(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, error) {
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:   JobTypeFull,
		Payload:   map[string]any{},
		Priority:  3,
		UniqueKey: "learner_profile:full",
	})
}

// JobHandler executes one learner profile queue job.
type JobHandler interface {
	Execute(ctx context.Context, payload json.RawMessage) error
}

// RegisterJobHandlers wires learner profile jobs into a background job registry.
func RegisterJobHandlers(register func(string, JobHandler), svc *Service) {
	if register == nil || svc == nil {
		return
	}
	register(JobTypeIncremental, incrementalHandler{svc: svc})
	register(JobTypeFull, fullHandler{svc: svc})
}

type incrementalHandler struct{ svc *Service }

func (h incrementalHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var p IncrementalPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("learner_profile.incremental: bad payload: %w", err)
	}
	if p.UserID == uuid.Nil {
		return fmt.Errorf("learner_profile.incremental: missing userId")
	}
	return h.svc.RecomputeIncremental(ctx, p.UserID, p.Facets...)
}

type fullHandler struct{ svc *Service }

func (h fullHandler) Execute(ctx context.Context, _ json.RawMessage) error {
	return h.svc.RecomputeAll(ctx)
}