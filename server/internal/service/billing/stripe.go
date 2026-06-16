// Package billing integrates Stripe Checkout, Customer Portal, and webhooks (plan 15.3).
package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v81"
	billingportal "github.com/stripe/stripe-go/v81/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/webhook"

	"github.com/lextures/lextures/server/internal/config"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// CheckoutRequest is the learner checkout payload.
type CheckoutRequest struct {
	UserID     uuid.UUID
	Email      string
	CourseID   *uuid.UUID
	Plan       string // monthly | annual
	PromoCode  string
	SuccessURL string
	CancelURL  string
}

// CheckoutResult is a Stripe Checkout redirect.
type CheckoutResult struct {
	SessionID  string
	CheckoutURL string
}

// Config bundles Stripe credentials and price ids.
type StripeConfig struct {
	SecretKey         string
	WebhookSecret     string
	MonthlyPriceID    string
	AnnualPriceID     string
	PublicWebOrigin   string
}

// ConfigFrom merges process config for Stripe calls.
func ConfigFrom(cfg config.Config) StripeConfig {
	return StripeConfig{
		SecretKey:       strings.TrimSpace(cfg.StripeSecretKey),
		WebhookSecret:   strings.TrimSpace(cfg.StripeWebhookSecret),
		MonthlyPriceID:  strings.TrimSpace(cfg.StripeMonthlyPriceID),
		AnnualPriceID:   strings.TrimSpace(cfg.StripeAnnualPriceID),
		PublicWebOrigin: strings.TrimRight(strings.TrimSpace(cfg.PublicWebOrigin), "/"),
	}
}

func (c StripeConfig) IsConfigured() bool {
	return c.SecretKey != ""
}

// CreateCheckoutSession starts Stripe Checkout for a course purchase or subscription.
func CreateCheckoutSession(ctx context.Context, pool *pgxpool.Pool, cfg StripeConfig, req CheckoutRequest) (*CheckoutResult, error) {
	if !cfg.IsConfigured() {
		return nil, errors.New("stripe not configured")
	}
	stripe.Key = cfg.SecretKey

	customerID, err := ensureCustomer(ctx, pool, cfg, req.UserID, req.Email)
	if err != nil {
		return nil, err
	}

	params := &stripe.CheckoutSessionParams{
		Customer:            stripe.String(customerID),
		Mode:                stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:          stripe.String(req.SuccessURL),
		CancelURL:           stripe.String(req.CancelURL),
		AllowPromotionCodes: stripe.Bool(true),
		Metadata: map[string]string{
			"user_id": req.UserID.String(),
		},
	}

	plan := strings.TrimSpace(req.Plan)
	switch {
	case req.CourseID != nil:
		price, err := repoBilling.CoursePriceByID(ctx, pool, *req.CourseID)
		if err != nil {
			return nil, err
		}
		if price == nil {
			return nil, fmt.Errorf("course not found")
		}
		if price.PriceCents <= 0 {
			return nil, fmt.Errorf("course is free")
		}
		currency := strings.ToLower(price.Currency)
		if currency == "" {
			currency = "usd"
		}
		params.Metadata["course_id"] = req.CourseID.String()
		params.Metadata["entitlement_type"] = repoBilling.TypeCoursePurchase
		params.LineItems = []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(price.Title),
					},
					UnitAmount: stripe.Int64(int64(price.PriceCents)),
				},
				Quantity: stripe.Int64(1),
			},
		}
	case plan == "monthly" || plan == "annual":
		params.Mode = stripe.String(string(stripe.CheckoutSessionModeSubscription))
		priceID := cfg.MonthlyPriceID
		entType := repoBilling.TypeSubscriptionMonthly
		if plan == "annual" {
			priceID = cfg.AnnualPriceID
			entType = repoBilling.TypeSubscriptionAnnual
		}
		if priceID == "" {
			return nil, fmt.Errorf("subscription price not configured")
		}
		params.Metadata["entitlement_type"] = entType
		params.LineItems = []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		}
	default:
		return nil, fmt.Errorf("course_id or plan required")
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return nil, err
	}
	return &CheckoutResult{SessionID: sess.ID, CheckoutURL: sess.URL}, nil
}

// CreatePortalSession returns a Stripe Customer Portal URL.
func CreatePortalSession(ctx context.Context, pool *pgxpool.Pool, cfg StripeConfig, userID uuid.UUID, email, returnURL string) (string, error) {
	if !cfg.IsConfigured() {
		return "", errors.New("stripe not configured")
	}
	stripe.Key = cfg.SecretKey
	customerID, err := ensureCustomer(ctx, pool, cfg, userID, email)
	if err != nil {
		return "", err
	}
	sess, err := billingportal.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	})
	if err != nil {
		return "", err
	}
	return sess.URL, nil
}

// WebhookResult summarizes webhook handling.
type WebhookResult struct {
	EventType string
	Created   bool
}

// HandleWebhook verifies and processes a Stripe webhook payload.
func HandleWebhook(ctx context.Context, pool *pgxpool.Pool, cfg StripeConfig, body []byte, sigHeader string) (*WebhookResult, error) {
	if cfg.WebhookSecret == "" {
		return nil, errors.New("stripe webhook secret not configured")
	}
	event, err := webhook.ConstructEvent(body, sigHeader, cfg.WebhookSecret)
	if err != nil {
		return nil, err
	}
	switch event.Type {
	case "checkout.session.completed":
		return handleCheckoutCompleted(ctx, pool, event)
	case "invoice.payment_succeeded":
		return handleInvoicePaymentSucceeded(ctx, pool, event)
	case "invoice.payment_failed":
		return handleInvoicePaymentFailed(ctx, pool, event)
	case "customer.subscription.deleted":
		return handleSubscriptionDeleted(ctx, pool, event)
	default:
		return &WebhookResult{EventType: string(event.Type)}, nil
	}
}

func handleCheckoutCompleted(ctx context.Context, pool *pgxpool.Pool, event stripe.Event) (*WebhookResult, error) {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(strings.TrimSpace(sess.Metadata["user_id"]))
	if err != nil {
		return nil, fmt.Errorf("missing user_id metadata")
	}
	entType := strings.TrimSpace(sess.Metadata["entitlement_type"])
	if entType == "" {
		entType = repoBilling.TypeCoursePurchase
	}
	var courseID *uuid.UUID
	if raw := strings.TrimSpace(sess.Metadata["course_id"]); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, err
		}
		courseID = &id
	}
	amount := int(sess.AmountTotal)
	currency := string(sess.Currency)
	var validUntil *time.Time
	switch entType {
	case repoBilling.TypeSubscriptionMonthly:
		t := time.Now().UTC().AddDate(0, 1, 0)
		validUntil = &t
	case repoBilling.TypeSubscriptionAnnual:
		t := time.Now().UTC().AddDate(1, 0, 0)
		validUntil = &t
	}
	_, created, err := repoBilling.CreateIdempotent(ctx, pool, repoBilling.CreateInput{
		UserID:          userID,
		EntitlementType: entType,
		CourseID:        courseID,
		StripeEventID:   event.ID,
		AmountPaidCents: amount,
		Currency:        currency,
		ValidUntil:      validUntil,
	})
	if err != nil {
		return nil, err
	}
	if created {
		RecordPayment(amount, entType)
	}
	return &WebhookResult{EventType: string(event.Type), Created: created}, nil
}

func handleInvoicePaymentSucceeded(ctx context.Context, pool *pgxpool.Pool, event stripe.Event) (*WebhookResult, error) {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return nil, err
	}
	if inv.BillingReason != stripe.InvoiceBillingReasonSubscriptionCycle {
		return &WebhookResult{EventType: string(event.Type)}, nil
	}
	userID, entType, err := subscriptionMetaFromInvoice(ctx, pool, &inv)
	if err != nil || userID == uuid.Nil {
		return &WebhookResult{EventType: string(event.Type)}, err
	}
	var validUntil *time.Time
	if inv.PeriodEnd > 0 {
		t := time.Unix(inv.PeriodEnd, 0).UTC()
		validUntil = &t
	}
	invID := inv.ID
	_, created, err := repoBilling.CreateIdempotent(ctx, pool, repoBilling.CreateInput{
		UserID:          userID,
		EntitlementType: entType,
		StripeEventID:   event.ID,
		StripeInvoiceID: &invID,
		AmountPaidCents: int(inv.AmountPaid),
		Currency:        string(inv.Currency),
		ValidUntil:      validUntil,
	})
	if err != nil {
		return nil, err
	}
	if created {
		RecordPayment(int(inv.AmountPaid), entType)
	}
	return &WebhookResult{EventType: string(event.Type), Created: created}, nil
}

func handleInvoicePaymentFailed(ctx context.Context, pool *pgxpool.Pool, event stripe.Event) (*WebhookResult, error) {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return nil, err
	}
	userID, _, err := subscriptionMetaFromInvoice(ctx, pool, &inv)
	if err != nil {
		return nil, err
	}
	if userID == uuid.Nil {
		return &WebhookResult{EventType: string(event.Type)}, nil
	}
	_, err = repoBilling.ExpireActiveSubscriptions(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	return &WebhookResult{EventType: string(event.Type)}, nil
}

func handleSubscriptionDeleted(ctx context.Context, pool *pgxpool.Pool, event stripe.Event) (*WebhookResult, error) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return nil, err
	}
	if sub.Customer == nil || sub.Customer.ID == "" {
		return &WebhookResult{EventType: string(event.Type)}, nil
	}
	userID, err := userIDForStripeCustomer(ctx, pool, sub.Customer.ID)
	if err != nil || userID == uuid.Nil {
		return &WebhookResult{EventType: string(event.Type)}, err
	}
	_, err = repoBilling.ExpireActiveSubscriptions(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	return &WebhookResult{EventType: string(event.Type)}, nil
}

func subscriptionMetaFromInvoice(ctx context.Context, pool *pgxpool.Pool, inv *stripe.Invoice) (uuid.UUID, string, error) {
	if inv.Customer == nil || inv.Customer.ID == "" {
		return uuid.Nil, "", nil
	}
	userID, err := userIDForStripeCustomer(ctx, pool, inv.Customer.ID)
	if err != nil || userID == uuid.Nil {
		return uuid.Nil, "", err
	}
	entType := repoBilling.TypeSubscriptionMonthly
	if inv.Lines != nil && len(inv.Lines.Data) > 0 && inv.Lines.Data[0].Price != nil {
		switch inv.Lines.Data[0].Price.Recurring.Interval {
		case stripe.PriceRecurringIntervalYear:
			entType = repoBilling.TypeSubscriptionAnnual
		}
	}
	return userID, entType, nil
}

func userIDForStripeCustomer(ctx context.Context, pool *pgxpool.Pool, customerID string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id FROM "user".users WHERE stripe_customer_id = $1
`, customerID).Scan(&userID)
	return userID, err
}

func ensureCustomer(ctx context.Context, pool *pgxpool.Pool, cfg StripeConfig, userID uuid.UUID, email string) (string, error) {
	existing, err := repoBilling.StripeCustomerID(ctx, pool, userID)
	if err != nil {
		return "", err
	}
	if existing != "" {
		return existing, nil
	}
	stripe.Key = cfg.SecretKey
	cust, err := customer.New(&stripe.CustomerParams{
		Email: stripe.String(email),
		Metadata: map[string]string{
			"user_id": userID.String(),
		},
	})
	if err != nil {
		return "", err
	}
	if err := repoBilling.SetStripeCustomerID(ctx, pool, userID, cust.ID); err != nil {
		return "", err
	}
	return cust.ID, nil
}

// LookupUserEmail loads the user email for portal/checkout.
func LookupUserEmail(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	row, err := user.FindByID(ctx, pool, userID)
	if err != nil || row == nil {
		return "", fmt.Errorf("user not found")
	}
	return row.Email, nil
}
