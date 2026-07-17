package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v81"

	"github.com/lextures/lextures/server/internal/currency"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/service/paymentprovider"
)

const EntitlementTranscriptOrder = "transcript_order"

// TranscriptCheckoutRequest starts Stripe Checkout for a transcript order.
type TranscriptCheckoutRequest struct {
	UserID     uuid.UUID
	Email      string
	OrderID    uuid.UUID
	SuccessURL string
	CancelURL  string
	WaiverCode string
}

// StartTranscriptCheckout creates a Stripe Checkout session for an unpaid transcript order.
func StartTranscriptCheckout(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg StripeConfig,
	tcfg *transcriptsrepo.Config,
	req TranscriptCheckoutRequest,
) (*CheckoutResult, error) {
	if !cfg.IsConfigured() {
		return nil, errors.New("stripe not configured")
	}
	if tcfg == nil || !tcfg.FeesEnabled {
		return nil, transcriptsrepo.ErrPaymentNotRequired
	}
	o, err := transcriptsrepo.GetOrderForUser(ctx, pool, req.OrderID, req.UserID)
	if err != nil {
		return nil, err
	}
	switch o.Status {
	case transcriptsrepo.OrderPendingPayment, transcriptsrepo.OrderPendingConsent, transcriptsrepo.OrderDraft,
		transcriptsrepo.OrderInReview, transcriptsrepo.OrderOnHold:
		// checkout allowed while unpaid
	default:
		return nil, transcriptsrepo.ErrCheckoutNotReady
	}
	ok, err := transcriptsrepo.PaymentSatisfiedForOrder(ctx, pool, tcfg, o)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, transcriptsrepo.ErrPaymentAlreadyDone
	}
	q, _, err := transcriptsrepo.QuoteOrder(ctx, pool, tcfg, o, transcriptsrepo.QuoteOptions{
		WaiverCode: req.WaiverCode,
	})
	if err != nil {
		return nil, err
	}
	if q == nil || !q.RequiresPayment {
		var waiverID *uuid.UUID
		_ = transcriptsrepo.PersistOrderQuote(ctx, pool, o.ID, q, waiverID, true)
		return nil, transcriptsrepo.ErrPaymentNotRequired
	}
	if err := currency.ValidateCatalogPrice(q.Total, q.Currency); err != nil {
		return nil, err
	}

	pcfg := paymentprovider.Config{
		StripeSecretKey:     cfg.SecretKey,
		StripeWebhookSecret: cfg.WebhookSecret,
		PublicWebOrigin:     cfg.PublicWebOrigin,
	}
	customerID, err := ensureCustomer(ctx, pool, cfg, req.UserID, req.Email)
	if err != nil {
		return nil, err
	}
	checkoutKey := paymentprovider.NewCheckoutKey()
	orgID := uuid.Nil
	if o.OrgID != nil {
		orgID = *o.OrgID
	}
	meta := map[string]string{
		"checkout_key":         checkoutKey,
		"entitlement_type":     EntitlementTranscriptOrder,
		"transcript_order_id":  o.ID.String(),
		"stripe_customer_id":   customerID,
		"user_id":              req.UserID.String(),
	}
	if orgID != uuid.Nil {
		meta["org_id"] = orgID.String()
	}
	provider := paymentprovider.NewStripeProvider(pcfg)
	success := strings.TrimSpace(req.SuccessURL)
	cancel := strings.TrimSpace(req.CancelURL)
	if success == "" {
		success = cfg.PublicWebOrigin + "/transcripts?checkout=success&orderId=" + o.ID.String()
	}
	if cancel == "" {
		cancel = cfg.PublicWebOrigin + "/transcripts?checkout=cancel&orderId=" + o.ID.String()
	}
	result, err := provider.CreateCheckoutSession(ctx, paymentprovider.CheckoutRequest{
		UserID:      req.UserID,
		Email:       req.Email,
		CourseTitle: "Official transcript order",
		PriceCents:  q.Total,
		Currency:    q.Currency,
		OrgID:       orgID,
		SuccessURL:  success,
		CancelURL:   cancel,
		TaxCode:     "txcd_99999999",
		Metadata:    meta,
	})
	if err != nil {
		return nil, err
	}
	if err := transcriptsrepo.MarkOrderPaymentPending(ctx, pool, o.ID, result.SessionID, q.Total, q.Currency); err != nil {
		return nil, err
	}
	_ = paymentprovider.PendingTransaction(ctx, pool, req.UserID, nil, result)
	return &CheckoutResult{SessionID: result.SessionID, CheckoutURL: result.CheckoutURL}, nil
}

// RefundTranscriptOrder issues a Stripe refund for a paid transcript order.
func RefundTranscriptOrder(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg StripeConfig,
	orderID uuid.UUID,
	amountCents *int,
) (*paymentprovider.RefundResult, *transcriptsrepo.Order, error) {
	if !cfg.IsConfigured() {
		return nil, nil, errors.New("stripe not configured")
	}
	o, err := transcriptsrepo.GetOrderByID(ctx, pool, orderID)
	if err != nil {
		return nil, nil, err
	}
	if o.PaymentStatus != transcriptsrepo.OrderPaymentPaid &&
		o.PaymentStatus != transcriptsrepo.OrderPaymentPartiallyRefunded {
		return nil, nil, transcriptsrepo.ErrRefundNotAllowed
	}
	ref := ""
	if o.PaymentRef != nil {
		ref = strings.TrimSpace(*o.PaymentRef)
	}
	if ref == "" {
		return nil, nil, fmt.Errorf("missing payment reference")
	}
	pi := ref
	// If we stored a Checkout Session id, resolve the PaymentIntent.
	if strings.HasPrefix(ref, "cs_") {
		stripe.Key = cfg.SecretKey
		sess, err := paymentprovider.GetStripeCheckoutSession(cfg.SecretKey, ref)
		if err != nil {
			return nil, nil, err
		}
		if sess.PaymentIntent == nil || sess.PaymentIntent.ID == "" {
			return nil, nil, fmt.Errorf("checkout session has no payment intent")
		}
		pi = sess.PaymentIntent.ID
	}
	provider := paymentprovider.NewStripeProvider(paymentprovider.Config{StripeSecretKey: cfg.SecretKey})
	refund, err := provider.IssueRefund(ctx, pi, amountCents)
	if err != nil {
		return nil, nil, err
	}
	amt := refund.AmountCents
	updated, err := transcriptsrepo.ApplyAdminRefund(ctx, pool, orderID, amt)
	if err != nil {
		return refund, nil, err
	}
	return refund, updated, nil
}

// ensureCustomer is already in stripe.go — reuse via same package.

// ResolveTranscriptOrderFromSessionMetadata extracts order id from checkout metadata.
func ResolveTranscriptOrderFromSessionMetadata(meta map[string]string) (uuid.UUID, bool) {
	raw := strings.TrimSpace(meta["transcript_order_id"])
	if raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
