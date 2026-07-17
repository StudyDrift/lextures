package background

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/jobqueue"
	"github.com/lextures/lextures/server/internal/service/transcriptinbound"
)

// JobTypeTranscriptInboundProcess is the durable queue type for T07 parse/match.
const JobTypeTranscriptInboundProcess = "transcript.inbound.process"

// TranscriptInboundProcessPayload identifies one inbound document to process.
type TranscriptInboundProcessPayload struct {
	InboundID uuid.UUID `json:"inboundId"`
}

type transcriptInboundProcessHandler struct {
	pool *pgxpool.Pool
}

func (h transcriptInboundProcessHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var p TranscriptInboundProcessPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("transcript.inbound.process: bad payload: %w", err)
	}
	if p.InboundID == uuid.Nil {
		return fmt.Errorf("transcript.inbound.process: missing inboundId")
	}
	_, err := transcriptinbound.Process(ctx, h.pool, p.InboundID)
	return err
}

// RegisterTranscriptInboundJob registers the inbound process handler and hooks.
func RegisterTranscriptInboundJob(r *Registry, pool *pgxpool.Pool) {
	if r == nil {
		return
	}
	r.Register(JobTypeTranscriptInboundProcess, transcriptInboundProcessHandler{pool: pool})
	transcriptinbound.AfterReceived = func(ctx context.Context, p *pgxpool.Pool, inboundID uuid.UUID) {
		_, _ = EnqueueTranscriptInboundProcess(ctx, p, inboundID)
	}
	transcriptinbound.NotifyFn = func(ctx context.Context, p *pgxpool.Pool, userID uuid.UUID, eventType, title, message, uniqueKey string) {
		_, _ = EnqueueEmail(ctx, p, EmailDeliveryPayload{
			RecipientID: userID,
			EventType:   eventType,
			Subject:     title,
			Template:    "generic_notice",
			TemplateVars: map[string]string{
				"title":   title,
				"message": message,
			},
		}, uniqueKey)
	}
}

// EnqueueTranscriptInboundProcess queues parse/match for one inbound document.
func EnqueueTranscriptInboundProcess(ctx context.Context, pool *pgxpool.Pool, inboundID uuid.UUID) (uuid.UUID, error) {
	if pool == nil || inboundID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("transcript.inbound.process: missing pool or id")
	}
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:     JobTypeTranscriptInboundProcess,
		Payload:     TranscriptInboundProcessPayload{InboundID: inboundID},
		Priority:    4,
		MaxAttempts: 5,
		UniqueKey:   "transcript-inbound:" + inboundID.String(),
	})
}
