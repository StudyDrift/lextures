# Adding a new background job type

The background job queue (plan 17.3) runs durable, retried work off the HTTP
request path. Adding a job type is two small steps: write a handler, register it.

## 1. Define the payload and handler

Create a file in `server/internal/background/`, e.g. `jobqueue_webhook.go`:

```go
const JobTypeWebhookDelivery = "webhook.delivery"

type WebhookDeliveryPayload struct {
    EndpointID uuid.UUID `json:"endpointId"`
    EventID    uuid.UUID `json:"eventId"`
}

type webhookDeliveryHandler struct{ pool *pgxpool.Pool }

func (h webhookDeliveryHandler) Execute(ctx context.Context, payload json.RawMessage) error {
    var p WebhookDeliveryPayload
    if err := json.Unmarshal(payload, &p); err != nil {
        return err
    }
    // ... do the work. Return an error to trigger retry/backoff; return nil on success.
    return nil
}
```

**Handlers must be idempotent.** Delivery is at-least-once: a worker crash
between the side effect and the completion write means the job can run again.
Guard with a `unique_key` at enqueue time and/or an idempotency check in the
handler (e.g. AI grading keys on `{submissionId}-{rubricVersion}`).

## 2. Register it

In `RegisterBuiltinJobs` (`jobqueue_email.go`):

```go
r.Register(JobTypeWebhookDelivery, webhookDeliveryHandler{pool: pool})
```

That is the only central change — the worker, retry, dead-letter, and admin UI
pick the type up automatically.

## 3. Enqueue

```go
jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
    JobType:     JobTypeWebhookDelivery,
    Payload:     WebhookDeliveryPayload{EndpointID: id, EventID: ev},
    Priority:    3,            // 1 = highest .. 10 = lowest; default 5
    MaxAttempts: 5,            // default 5
    UniqueKey:   "webhook:"+ev.String(), // optional dedup within the in-flight window
    ScheduledAt: time.Now().Add(time.Minute), // optional delayed run
})
```

## Operating

- Enable with `BACKGROUND_JOBS_ENABLED=true` (default off). Set
  `BACKGROUND_JOBS_CONCURRENCY` (default 4) per instance.
- Inspect via `GET /api/v1/admin/jobs` and `…/admin/jobs/dead-letters`.
- Re-drive a dead-letter: `POST /api/v1/admin/jobs/dead-letters/{id}/redrive`.
- Backoff schedule: 1m, 5m, 30m, 2h, 8h; after `MaxAttempts` the job
  dead-letters.
