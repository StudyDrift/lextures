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
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/webhook"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/courseroles"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/paymentprovider"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// CheckoutRequest is the learner checkout payload.
type CheckoutRequest struct {
	UserID              uuid.UUID
	Email               string
	CourseID            *uuid.UUID
	Plan                string // monthly | annual
	PromoCode           string
	AffiliateCode       string
	SuccessURL          string
	CancelURL           string
	PlatformTaxEnabled  bool
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
	taxCode := "txcd_99999999"
	if req.CourseID != nil {
		price, err := repoBilling.CoursePriceByID(ctx, pool, *req.CourseID)
		if err != nil {
			return nil, err
		}
		if price != nil {
			if taxEnabled, settings, _ := TaxEnabledForOrg(ctx, pool, price.OrgID, req.PlatformTaxEnabled); taxEnabled && settings != nil {
				taxCode = settings.DefaultTaxCategory
			}
		}
	}
	pcfg := paymentprovider.Config{
		StripeSecretKey:      cfg.SecretKey,
		StripeWebhookSecret:  cfg.WebhookSecret,
		StripeMonthlyPriceID: cfg.MonthlyPriceID,
		StripeAnnualPriceID:  cfg.AnnualPriceID,
		PublicWebOrigin:      cfg.PublicWebOrigin,
	}
	result, err := paymentprovider.StartCheckout(ctx, pool, pcfg, paymentprovider.StartCheckoutRequest{
		UserID:             req.UserID,
		Email:              req.Email,
		CourseID:           req.CourseID,
		Plan:               req.Plan,
		PromoCode:          req.PromoCode,
		AffiliateCode:      req.AffiliateCode,
		SuccessURL:         req.SuccessURL,
		CancelURL:          req.CancelURL,
		PlatformTaxEnabled: req.PlatformTaxEnabled,
		TaxCode:            taxCode,
	})
	if err != nil {
		return nil, err
	}
	return &CheckoutResult{SessionID: result.SessionID, CheckoutURL: result.CheckoutURL}, nil
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

// WebhookOptions configures optional post-payment side effects.
type WebhookOptions struct {
	RevenueShareEnabled bool
	TaxCollectionEnabled bool
}

// HandleWebhook verifies and processes a Stripe webhook payload.
func HandleWebhook(ctx context.Context, pool *pgxpool.Pool, cfg StripeConfig, body []byte, sigHeader string, opts WebhookOptions) (*WebhookResult, error) {
	if cfg.WebhookSecret == "" {
		return nil, errors.New("stripe webhook secret not configured")
	}
	event, err := webhook.ConstructEvent(body, sigHeader, cfg.WebhookSecret)
	if err != nil {
		return nil, err
	}
	return HandleWebhookEvent(ctx, pool, event, opts)
}

// HandleWebhookEvent processes a verified Stripe event (sync or queued worker).
func HandleWebhookEvent(ctx context.Context, pool *pgxpool.Pool, event stripe.Event, opts WebhookOptions) (*WebhookResult, error) {
	switch event.Type {
	case "checkout.session.completed":
		return handleCheckoutCompleted(ctx, pool, event, opts)
	case "invoice.payment_succeeded":
		return handleInvoicePaymentSucceeded(ctx, pool, event)
	case "invoice.payment_failed":
		return handleInvoicePaymentFailed(ctx, pool, event)
	case "customer.subscription.deleted":
		return handleSubscriptionDeleted(ctx, pool, event)
	case "charge.refunded":
		return handleChargeRefunded(ctx, pool, event, opts)
	default:
		return &WebhookResult{EventType: string(event.Type)}, nil
	}
}

func handleCheckoutCompleted(ctx context.Context, pool *pgxpool.Pool, event stripe.Event, opts WebhookOptions) (*WebhookResult, error) {
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
	ent, created, err := repoBilling.CreateIdempotent(ctx, pool, repoBilling.CreateInput{
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
	if created && opts.TaxCollectionEnabled {
		orgID := orgIDFromMetadata(sess.Metadata)
		if orgID != uuid.Nil {
			_ = PersistCheckoutTax(ctx, pool, event.ID, &sess, orgID, opts.TaxCollectionEnabled)
			if ent != nil {
				_, _ = IssueTaxInvoice(ctx, pool, ent.ID, orgID)
			}
		}
	}
	if created {
		RecordPayment(amount, entType)
		if opts.RevenueShareEnabled && courseID != nil && entType == repoBilling.TypeCoursePurchase {
			if err := RecordSaleEarnings(ctx, pool, SaleEarningsInput{
				BuyerUserID:   userID,
				CourseID:      *courseID,
				AmountCents:   amount,
				Currency:      currency,
				StripeEventID: event.ID,
				AffiliateCode: strings.TrimSpace(sess.Metadata["affiliate_code"]),
			}); err != nil {
				return nil, err
			}
		}
	}
	// Marketplace paid path: enroll + refresh grants after entitlement (plan MKT4 FR-4).
	// Idempotent — safe on webhook retries even when entitlement was already created.
	if courseID != nil && entType == repoBilling.TypeCoursePurchase {
		if err := enrollCoursePurchase(ctx, pool, userID, *courseID); err != nil {
			return nil, err
		}
		if created && amount > 0 {
			telemetry.RecordMarketplacePurchaseCompleted()
		}
	}
	return &WebhookResult{EventType: string(event.Type), Created: created}, nil
}

func enrollCoursePurchase(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) error {
	code, err := repoCourse.GetCourseCodeByID(ctx, pool, courseID)
	if err != nil {
		return err
	}
	if code == nil || strings.TrimSpace(*code) == "" {
		return nil
	}
	_, err = courseroles.EnrollStudentWithGrants(ctx, pool, courseID, userID, *code)
	return err
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

func handleChargeRefunded(ctx context.Context, pool *pgxpool.Pool, event stripe.Event, opts WebhookOptions) (*WebhookResult, error) {
	var ch stripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &ch); err != nil {
		return nil, err
	}
	rawCourse := strings.TrimSpace(ch.Metadata["course_id"])
	rawUser := strings.TrimSpace(ch.Metadata["user_id"])
	var courseID uuid.UUID
	var userID uuid.UUID
	if rawCourse != "" {
		if id, err := uuid.Parse(rawCourse); err == nil {
			courseID = id
		}
	}
	if rawUser != "" {
		if id, err := uuid.Parse(rawUser); err == nil {
			userID = id
		}
	}
	// Prefer user+course refund (plan MKT4 FR-8); fall back to tax reverse by course.
	if userID != uuid.Nil && courseID != uuid.Nil {
		refunded, err := repoBilling.RefundCourseEntitlement(ctx, pool, userID, courseID)
		if err != nil {
			return nil, err
		}
		if refunded {
			telemetry.RecordMarketplaceRefund()
		}
	} else if opts.TaxCollectionEnabled && courseID != uuid.Nil {
		_, _ = repoBilling.ReverseEntitlementTaxByCourse(ctx, pool, courseID)
	}
	if !opts.RevenueShareEnabled || courseID == uuid.Nil {
		return &WebhookResult{EventType: string(event.Type)}, nil
	}
	refundAmount := int(ch.AmountRefunded)
	if refundAmount <= 0 {
		refundAmount = int(ch.Amount)
	}
	currency := string(ch.Currency)
	if err := RecordRefundEarnings(ctx, pool, RefundEarningsInput{
		CourseID:      courseID,
		AmountCents:   refundAmount,
		Currency:      currency,
		StripeEventID: event.ID,
		AffiliateCode: strings.TrimSpace(ch.Metadata["affiliate_code"]),
	}); err != nil {
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
	return EnsureStripeCustomer(ctx, pool, cfg, userID, email)
}

// EnsureStripeCustomer returns or creates the Stripe customer id for a user.
func EnsureStripeCustomer(ctx context.Context, pool *pgxpool.Pool, cfg StripeConfig, userID uuid.UUID, email string) (string, error) {
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

func orgIDFromMetadata(meta map[string]string) uuid.UUID {
	raw := strings.TrimSpace(meta["org_id"])
	if raw == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// LookupUserEmail loads the user email for portal/checkout.
func LookupUserEmail(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	row, err := user.FindByID(ctx, pool, userID)
	if err != nil || row == nil {
		return "", fmt.Errorf("user not found")
	}
	return row.Email, nil
}
