package billing

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v81"

	repoPayments "github.com/lextures/lextures/server/internal/repos/payments"
	"github.com/lextures/lextures/server/internal/service/paymentprovider"
)

// ProcessStripeWebhookJob runs idempotent Stripe webhook fulfillment for a queued job.
func ProcessStripeWebhookJob(ctx context.Context, pool *pgxpool.Pool, event stripe.Event, opts WebhookOptions) (*WebhookResult, error) {
	result, err := HandleWebhookEvent(ctx, pool, event, opts)
	if err != nil {
		return nil, err
	}
	if err := recordStripePaymentTransaction(ctx, pool, event, result); err != nil {
		return nil, err
	}
	return result, nil
}

func recordStripePaymentTransaction(ctx context.Context, pool *pgxpool.Pool, event stripe.Event, result *WebhookResult) error {
	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return err
		}
		userID, err := uuid.Parse(strings.TrimSpace(sess.Metadata["user_id"]))
		if err != nil {
			return err
		}
		var courseID *uuid.UUID
		if raw := strings.TrimSpace(sess.Metadata["course_id"]); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				return err
			}
			courseID = &id
		}
		var subID *string
		if sess.Subscription != nil && sess.Subscription.ID != "" {
			id := sess.Subscription.ID
			subID = &id
			planID := strings.TrimSpace(sess.Metadata["entitlement_type"])
			if planID == "" {
				planID = "subscription"
			}
			_ = repoPayments.Upsert(ctx, pool, repoPayments.UpsertSubscriptionInput{
				UserID:        userID,
				Provider:      repoPayments.ProviderStripe,
				ProviderSubID: id,
				PlanID:        planID,
				Status:        repoPayments.SubStatusActive,
			})
		}
		_, _, err = repoPayments.CreateIdempotent(ctx, pool, repoPayments.CreateTransactionInput{
			UserID:         userID,
			CourseID:       courseID,
			Provider:       repoPayments.ProviderStripe,
			ProviderTxnID:  sess.ID,
			IdempotencyKey: event.ID,
			AmountCents:    int(sess.AmountTotal),
			Currency:       string(sess.Currency),
			Status:         repoPayments.StatusCompleted,
			SubscriptionID: subID,
		})
		if err != nil {
			return err
		}
		paymentprovider.RecordTransaction(paymentprovider.ProviderStripe, repoPayments.StatusCompleted, string(sess.Currency))
	case "invoice.payment_failed":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			return err
		}
		if inv.Customer == nil {
			return nil
		}
		userID, err := userIDForStripeCustomer(ctx, pool, inv.Customer.ID)
		if err != nil || userID == uuid.Nil {
			return err
		}
		_, _ = repoPayments.MarkPastDueByUser(ctx, pool, userID)
	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return err
		}
		if sub.ID != "" {
			_ = repoPayments.CancelByProviderSubID(ctx, pool, sub.ID)
		}
	case "charge.refunded":
		var ch stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &ch); err != nil {
			return err
		}
		pi := ""
		if ch.PaymentIntent != nil {
			pi = ch.PaymentIntent.ID
		}
		if pi != "" {
			_ = repoPayments.UpdateStatusByProviderTxn(ctx, pool, repoPayments.ProviderStripe, pi, repoPayments.StatusRefunded)
		}
	}
	return nil
}

// ProcessPayPalWebhookJob fulfills PayPal webhook events.
func ProcessPayPalWebhookJob(ctx context.Context, pool *pgxpool.Pool, payload []byte) error {
	return paymentprovider.ProcessPayPalJob(ctx, pool, payload)
}

// SweepPaymentWebhookJobs processes due payment webhook jobs.
func SweepPaymentWebhookJobs(ctx context.Context, pool *pgxpool.Pool, cfg paymentprovider.Config, opts WebhookOptions, now time.Time) {
	jobs, err := repoPayments.ListDueWebhookJobs(ctx, pool, 50, now)
	if err != nil {
		return
	}
	retryDelays := []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute}
	for _, job := range jobs {
		_ = repoPayments.MarkWebhookProcessing(ctx, pool, job.ID)
		var procErr error
		switch job.Provider {
		case repoPayments.ProviderStripe:
			var event stripe.Event
			if err := json.Unmarshal(job.Payload, &event); err != nil {
				procErr = err
			} else {
				_, procErr = ProcessStripeWebhookJob(ctx, pool, event, opts)
			}
		case repoPayments.ProviderPayPal:
			procErr = ProcessPayPalWebhookJob(ctx, pool, job.Payload)
		default:
			procErr = paymentprovider.ErrUnknownProvider(job.Provider)
		}
		if procErr == nil {
			_ = repoPayments.MarkWebhookCompleted(ctx, pool, job.ID, now)
			continue
		}
		attempts := job.Attempts + 1
		dead := attempts >= len(retryDelays)
		var next *time.Time
		if !dead {
			t := now.Add(retryDelays[attempts-1])
			next = &t
		}
		_ = repoPayments.MarkWebhookFailed(ctx, pool, job.ID, attempts, next, procErr.Error(), dead)
	}
}
