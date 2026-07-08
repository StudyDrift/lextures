package background

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/mail"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
	learnerprofilesvc "github.com/lextures/lextures/server/internal/service/learnerprofile"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

// JobTypeEmailDelivery is the registered type for transactional email delivery
// (plan 17.3 Phase 1 / 6.2 EmailDeliveryJob).
const JobTypeEmailDelivery = "email.delivery"

// EmailDeliveryPayload is the JSON payload for an email.delivery job. It carries
// only identifiers and template data — never a rendered body with PII beyond
// what the template needs (plan 17.3 NFR security/privacy).
type EmailDeliveryPayload struct {
	RecipientID  uuid.UUID         `json:"recipientId"`
	EventType    string            `json:"eventType"`
	Subject      string            `json:"subject"`
	Template     string            `json:"template"`
	TemplateVars map[string]string `json:"templateVars"`
}

// emailDeliveryHandler renders and sends one transactional email. It is
// idempotent at the SMTP level only insofar as the mail server allows; the
// queue's at-least-once delivery means a duplicate send is possible on retry
// after a crash between send and ack (plan 17.3 FR-1 idempotency note).
type emailDeliveryHandler struct {
	pool *pgxpool.Pool
	cfg  config.Config
}

func (h emailDeliveryHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var p EmailDeliveryPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("email.delivery: bad payload: %w", err)
	}
	if p.RecipientID == uuid.Nil {
		return fmt.Errorf("email.delivery: missing recipientId")
	}
	to, err := notifications.RecipientEmail(ctx, h.pool, p.RecipientID)
	if err != nil {
		return err
	}
	branding := brandingForRecipient(ctx, h.pool, p.RecipientID)
	mailBranding := mailBrandingFromRow(branding)
	rendered, err := mail.RenderTemplate(p.Template, p.TemplateVars, mailBranding)
	if err != nil {
		return err
	}
	subject := p.Subject
	if rendered.Subject != "" {
		subject = rendered.Subject
	}
	icsContent := p.TemplateVars["icsContent"]
	icsFilename := p.TemplateVars["icsFilename"]
	if strings.TrimSpace(icsContent) != "" {
		return mail.SendMultipartWithICS(h.cfg, to, subject, rendered.BodyText, rendered.HTMLBody, mailBranding, icsContent, icsFilename)
	}
	return mail.SendMultipart(h.cfg, to, subject, rendered.BodyText, rendered.HTMLBody, mailBranding)
}

// RegisterBuiltinJobs registers the job types shipped with the platform. New
// job types are added here alongside their handler implementation
// (plan 17.3 NFR maintainability).
func RegisterBuiltinJobs(r *Registry, pool *pgxpool.Pool, cfg config.Config) {
	r.Register(JobTypeEmailDelivery, emailDeliveryHandler{pool: pool, cfg: cfg})
	RegisterUserImportJob(r, pool, cfg)
	registerScheduledJobs(r, pool, cfg)
}

// RegisterLearnerProfileJobs adds learner profile queue handlers when the service is wired.
func RegisterLearnerProfileJobs(r *Registry, svc *learnerprofilesvc.Service) {
	if r == nil || svc == nil {
		return
	}
	learnerprofilesvc.RegisterJobHandlers(func(jobType string, h learnerprofilesvc.JobHandler) {
		r.Register(jobType, HandlerFunc(h.Execute))
	}, svc)
}

// EnqueueEmail queues a transactional email for asynchronous delivery on the
// generic background queue. unique_key dedups identical sends within the
// in-flight window (plan 17.3 FR-8).
func EnqueueEmail(ctx context.Context, pool *pgxpool.Pool, p EmailDeliveryPayload, uniqueKey string) (uuid.UUID, error) {
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:   JobTypeEmailDelivery,
		Payload:   p,
		Priority:  5,
		UniqueKey: uniqueKey,
	})
}
