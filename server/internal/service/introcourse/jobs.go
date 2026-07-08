package introcourse

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
)

const (
	// JobTypeEnrollRetry is the durable queue job for a failed inline enrollment (IC02 FR-3).
	JobTypeEnrollRetry = "intro_course.enroll_retry"
	// JobTypeBackfill is the durable queue job for the one-time user backfill (IC02 FR-5).
	JobTypeBackfill = "intro_course.backfill"
)

// EnrollRetryPayload is the JSON payload for intro_course.enroll_retry jobs.
type EnrollRetryPayload struct {
	UserID uuid.UUID  `json:"userId"`
	Path   EnrollPath `json:"path"`
}

// EnqueueEnrollmentRetry queues a best-effort retry for a failed inline enrollment.
func EnqueueEnrollmentRetry(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, path EnrollPath) (uuid.UUID, error) {
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:   JobTypeEnrollRetry,
		Payload:   EnrollRetryPayload{UserID: userID, Path: path},
		Priority:  6,
		UniqueKey: fmt.Sprintf("intro_course:enroll:%s", userID.String()),
	})
}

// EnqueueBackfillIfNeeded queues the backfill when the flag is on and it has not completed.
func EnqueueBackfillIfNeeded(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) (uuid.UUID, error) {
	if !Enabled(cfg) || pool == nil {
		return uuid.Nil, nil
	}
	st, err := icrepo.LoadBackfillState(ctx, pool)
	if err != nil {
		return uuid.Nil, err
	}
	if st.CompletedAt != nil {
		return uuid.Nil, nil
	}
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:   JobTypeBackfill,
		Payload:   map[string]any{},
		Priority:  4,
		UniqueKey: "intro_course:backfill",
	})
}

// JobHandler executes one intro course queue job.
type JobHandler interface {
	Execute(ctx context.Context, payload json.RawMessage) error
}

// RegisterJobHandlers wires intro course jobs into a background job registry.
func RegisterJobHandlers(register func(string, JobHandler), svc *Service, cfg config.Config) {
	if register == nil || svc == nil {
		return
	}
	register(JobTypeEnrollRetry, enrollRetryHandler{svc: svc, cfg: cfg})
	register(JobTypeBackfill, backfillHandler{svc: svc, cfg: cfg})
}

type enrollRetryHandler struct {
	svc *Service
	cfg config.Config
}

func (h enrollRetryHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var p EnrollRetryPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("intro_course.enroll_retry: bad payload: %w", err)
	}
	if p.UserID == uuid.Nil {
		return fmt.Errorf("intro_course.enroll_retry: missing userId")
	}
	path := p.Path
	if path == "" {
		path = PathSignup
	}
	return h.svc.EnsureEnrollment(ctx, h.cfg, h.svc.Pool, p.UserID, path)
}

type backfillHandler struct {
	svc *Service
	cfg config.Config
}

func (h backfillHandler) Execute(ctx context.Context, _ json.RawMessage) error {
	return h.svc.RunBackfill(ctx, h.cfg)
}