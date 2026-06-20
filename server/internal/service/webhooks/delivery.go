package webhooksvc

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/mail"
	webhooksrepo "github.com/lextures/lextures/server/internal/repos/webhooks"
	"github.com/lextures/lextures/server/internal/webhooks"
)

var retryDelays = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	8 * time.Hour,
	24 * time.Hour,
}

const maxAttempts = 6

const deliveryTimeout = 10 * time.Second

// Emitter enqueues webhook deliveries for org events.
type Emitter struct {
	Pool *pgxpool.Pool
	Cfg  config.Config
}

// EmitAsync enqueues webhook deliveries after the triggering transaction commits.
func EmitAsync(pool *pgxpool.Pool, cfg config.Config, orgID uuid.UUID, eventType webhooks.EventType, data any) {
	if !cfg.FFWebhooks || pool == nil {
		return
	}
	go func() {
		ctx := context.Background()
		em := Emitter{Pool: pool, Cfg: cfg}
		if err := em.Emit(ctx, orgID, eventType, data, false); err != nil {
			slog.Warn("webhooks.emit", "event_type", eventType, "org_id", orgID, "err", err)
		}
	}()
}

// Emit enqueues deliveries for all matching active subscriptions.
func (e Emitter) Emit(ctx context.Context, orgID uuid.UUID, eventType webhooks.EventType, data any, test bool) error {
	if !e.Cfg.FFWebhooks || e.Pool == nil {
		return nil
	}
	env, body, err := webhooks.NewEnvelope(eventType, data, test)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(env.EventID)
	if err != nil {
		return err
	}
	hash := webhooks.PayloadHash(body)
	subs, err := webhooksrepo.ListActiveForEvent(ctx, e.Pool, orgID, string(eventType))
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if _, err := webhooksrepo.InsertDelivery(ctx, e.Pool, sub.ID, string(eventType), eventID, hash, body); err != nil {
			slog.Warn("webhooks.enqueue", "subscription_id", sub.ID, "err", err)
		}
	}
	return nil
}

// SweepDueDeliveries processes pending webhook deliveries.
func SweepDueDeliveries(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.FFWebhooks || pool == nil {
		return
	}
	jobs, err := webhooksrepo.ListDueDeliveries(ctx, pool, 50, now)
	if err != nil {
		slog.Warn("webhooks.list_due", "err", err)
		return
	}
	for _, job := range jobs {
		if err := deliverJob(ctx, pool, cfg, job, now); err != nil {
			slog.Warn("webhooks.deliver", "delivery_id", job.ID, "err", err)
		}
	}
}

func deliverJob(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, job webhooksrepo.Delivery, now time.Time) error {
	sub, err := webhooksrepo.GetByIDAnyOrg(ctx, pool, job.SubscriptionID)
	if err != nil || sub == nil || !sub.Active || sub.PausedAt != nil {
		return err
	}
	if len(cfg.PlatformSecretsKey) != 32 {
		return fmt.Errorf("platform secrets key not configured")
	}
	signingKey, err := webhooks.DecryptSigningKey(sub.SigningKeyEnc, cfg.PlatformSecretsKey)
	if err != nil {
		return err
	}
	payload := []byte("")
	if job.LastResponse != nil {
		payload = []byte(*job.LastResponse)
	}
	if len(payload) == 0 {
		payload, err = webhooksrepo.GetDeliveryPayload(ctx, pool, job.ID)
		if err != nil || len(payload) == 0 {
			return fmt.Errorf("missing delivery payload")
		}
	}
	client := outboundClient(sub.TLSSkipVerify)
	start := time.Now()
	status, snippet, derr := postWebhook(ctx, client, sub.EndpointURL, payload, signingKey)
	latencyMS := int(time.Since(start).Milliseconds())
	if derr == nil && status >= 200 && status < 300 {
		return webhooksrepo.MarkDelivered(ctx, pool, job.ID, now, status, latencyMS, snippet)
	}
	msg := snippet
	if derr != nil {
		msg = derr.Error()
	}
	attempts := job.AttemptCount + 1
	dead := attempts >= maxAttempts
	var next time.Time
	if !dead {
		idx := attempts - 1
		if idx >= len(retryDelays) {
			idx = len(retryDelays) - 1
		}
		next = now.Add(retryDelays[idx])
	}
	if err := webhooksrepo.MarkFailed(ctx, pool, job.ID, attempts, next, dead, status, msg, payload); err != nil {
		return err
	}
	if dead {
		_ = webhooksrepo.Pause(ctx, pool, sub.ID)
		notifyPausedSubscription(ctx, pool, cfg, sub)
	}
	return fmt.Errorf("%s", msg)
}

func postWebhook(ctx context.Context, client *http.Client, endpoint string, body, signingKey []byte) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(endpoint), bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Lextures-Webhooks/1.0")
	req.Header.Set(webhooks.SignatureHeaderName(), webhooks.SignPayload(body, signingKey))
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	snip, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return resp.StatusCode, string(snip), nil
}

func outboundClient(tlsSkipVerify bool) *http.Client {
	transport := &http.Transport{
		DialContext: safeDialContext,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: tlsSkipVerify, //nolint:gosec // admin opt-in only
		},
	}
	return &http.Client{
		Timeout:   deliveryTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if err := webhooks.ValidateEndpointURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
}

func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if webhooks.BlockedIP(ip.IP) {
			return nil, webhooks.ErrSSRFPolicy
		}
	}
	d := net.Dialer{Timeout: deliveryTimeout}
	return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
}

func notifyPausedSubscription(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, sub *webhooksrepo.Subscription) {
	if !cfg.EmailNotificationsEnabled || sub.CreatedBy == nil {
		return
	}
	var email string
	err := pool.QueryRow(ctx, `SELECT email FROM "user".users WHERE id = $1`, *sub.CreatedBy).Scan(&email)
	if err != nil || strings.TrimSpace(email) == "" {
		return
	}
	subject := "Webhook subscription paused"
	body := fmt.Sprintf("Your webhook subscription %q (%s) was paused after repeated delivery failures.", sub.Label, sub.EndpointURL)
	_ = mail.SendMultipart(cfg, email, subject, body, "", nil)
}

// DeliverTest sends a synthetic test event immediately (synchronous).
func DeliverTest(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, sub *webhooksrepo.Subscription, eventType webhooks.EventType) (*webhooksrepo.Delivery, error) {
	sample := SampleDataForEvent(eventType)
	env, body, err := webhooks.NewEnvelope(eventType, sample, true)
	if err != nil {
		return nil, err
	}
	eventID, err := uuid.Parse(env.EventID)
	if err != nil {
		return nil, err
	}
	hash := webhooks.PayloadHash(body)
	id, err := webhooksrepo.InsertDelivery(ctx, pool, sub.ID, string(eventType), eventID, hash, body)
	if err != nil {
		return nil, err
	}
	job := webhooksrepo.Delivery{
		ID:             id,
		SubscriptionID: sub.ID,
		EventType:      string(eventType),
		EventID:        eventID,
		PayloadHash:    hash,
		LastResponse:   strPtr(string(body)),
	}
	if err := deliverJob(ctx, pool, cfg, job, time.Now().UTC()); err != nil {
		// still return delivery row for log
	}
	deliveries, err := webhooksrepo.ListDeliveries(ctx, pool, sub.ID, 1)
	if err != nil || len(deliveries) == 0 {
		return nil, err
	}
	return &deliveries[0], nil
}

func strPtr(s string) *string { return &s }
