package paymentprovider

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/refund"
	"github.com/stripe/stripe-go/v81/subscription"
	"github.com/stripe/stripe-go/v81/webhook"
)

// StripeProvider implements Provider using Stripe Checkout and Billing.
type StripeProvider struct {
	cfg Config
}

// NewStripeProvider returns a Stripe-backed provider.
func NewStripeProvider(cfg Config) *StripeProvider {
	return &StripeProvider{cfg: cfg}
}

func (p *StripeProvider) Name() ProviderName { return ProviderStripe }

func (p *StripeProvider) CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error) {
	stripe.Key = p.cfg.StripeSecretKey
	customerID := strings.TrimSpace(req.Metadata["stripe_customer_id"])
	if customerID == "" {
		return nil, fmt.Errorf("paymentprovider: stripe customer required")
	}
	idempotencyKey := fmt.Sprintf("checkout:%s:%s", req.UserID, strings.TrimSpace(req.Metadata["checkout_key"]))
	params := &stripe.CheckoutSessionParams{
		Customer:            stripe.String(customerID),
		Mode:                stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:          stripe.String(req.SuccessURL),
		CancelURL:           stripe.String(req.CancelURL),
		AllowPromotionCodes: stripe.Bool(true),
		Metadata:            copyMetadata(req.Metadata),
	}
	params.Metadata["user_id"] = req.UserID.String()
	if req.CourseID != nil {
		params.Metadata["course_id"] = req.CourseID.String()
	}
	if req.OrgID != uuid.Nil {
		params.Metadata["org_id"] = req.OrgID.String()
	}
	if code := strings.TrimSpace(req.AffiliateCode); code != "" {
		params.Metadata["affiliate_code"] = code
	}
	currency := strings.ToLower(req.Currency)
	if currency == "" {
		currency = "usd"
	}
	taxCode := req.TaxCode
	if taxCode == "" {
		taxCode = "txcd_99999999"
	}
	params.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{
		Metadata: params.Metadata,
	}
	params.LineItems = []*stripe.CheckoutSessionLineItemParams{{
		PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
			Currency: stripe.String(currency),
			ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
				Name:    stripe.String(req.CourseTitle),
				TaxCode: stripe.String(taxCode),
			},
			UnitAmount: stripe.Int64(int64(req.PriceCents)),
		},
		Quantity: stripe.Int64(1),
	}}
	if req.PlatformTaxEnabled {
		params.AutomaticTax = &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(true)}
	}
	paymentMethods := stripeCheckoutPaymentMethods(req.Country)
	if len(paymentMethods) > 0 {
		params.PaymentMethodTypes = stripe.StringSlice(paymentMethods)
	}
	slog.Info(
		"stripe checkout session amount",
		"currency", currency,
		"unit_amount", req.PriceCents,
		"course_id", req.CourseID,
	)
	sess, err := checkoutsession.New(params)
	if err != nil {
		return nil, err
	}
	return &CheckoutResult{
		SessionID:      sess.ID,
		CheckoutURL:    sess.URL,
		Provider:       ProviderStripe,
		IdempotencyKey: idempotencyKey,
		AmountCents:    req.PriceCents,
		Currency:       currency,
	}, nil
}

func (p *StripeProvider) CreateSubscription(ctx context.Context, req SubscriptionRequest) (*CheckoutResult, error) {
	stripe.Key = p.cfg.StripeSecretKey
	customerID := strings.TrimSpace(req.Metadata["stripe_customer_id"])
	if customerID == "" {
		return nil, fmt.Errorf("paymentprovider: stripe customer required")
	}
	priceID := req.PriceID
	if priceID == "" {
		switch strings.TrimSpace(req.Plan) {
		case "monthly":
			priceID = p.cfg.StripeMonthlyPriceID
		case "annual":
			priceID = p.cfg.StripeAnnualPriceID
		default:
			return nil, fmt.Errorf("paymentprovider: subscription plan required")
		}
	}
	if priceID == "" {
		return nil, fmt.Errorf("paymentprovider: subscription price not configured")
	}
	idempotencyKey := fmt.Sprintf("sub:%s:%s", req.UserID, strings.TrimSpace(req.Plan))
	params := &stripe.CheckoutSessionParams{
		Customer:   stripe.String(customerID),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(req.SuccessURL),
		CancelURL:  stripe.String(req.CancelURL),
		Metadata:   copyMetadata(req.Metadata),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Price:    stripe.String(priceID),
			Quantity: stripe.Int64(1),
		}},
	}
	params.Metadata["user_id"] = req.UserID.String()
	sess, err := checkoutsession.New(params)
	if err != nil {
		return nil, err
	}
	return &CheckoutResult{
		SessionID:      sess.ID,
		CheckoutURL:    sess.URL,
		Provider:       ProviderStripe,
		IdempotencyKey: idempotencyKey,
	}, nil
}

func (p *StripeProvider) CancelSubscription(ctx context.Context, providerSubID string) error {
	stripe.Key = p.cfg.StripeSecretKey
	_, err := subscription.Cancel(providerSubID, nil)
	return err
}

func (p *StripeProvider) IssueRefund(ctx context.Context, providerTxnID string, amountCents *int) (*RefundResult, error) {
	stripe.Key = p.cfg.StripeSecretKey
	params := &stripe.RefundParams{PaymentIntent: stripe.String(providerTxnID)}
	if amountCents != nil && *amountCents > 0 {
		params.Amount = stripe.Int64(int64(*amountCents))
	}
	r, err := refund.New(params)
	if err != nil {
		return nil, err
	}
	return &RefundResult{
		RefundID:    r.ID,
		AmountCents: int(r.Amount),
		Currency:    string(r.Currency),
		Status:      string(r.Status),
	}, nil
}

func (p *StripeProvider) GetTransaction(ctx context.Context, providerTxnID string) (*TransactionInfo, error) {
	stripe.Key = p.cfg.StripeSecretKey
	sess, err := checkoutsession.Get(providerTxnID, nil)
	if err != nil {
		return nil, err
	}
	info := &TransactionInfo{
		ProviderTxnID: sess.ID,
		AmountCents:   int(sess.AmountTotal),
		Currency:      string(sess.Currency),
		Status:        string(sess.PaymentStatus),
		Metadata:      sess.Metadata,
	}
	if sess.Subscription != nil {
		id := sess.Subscription.ID
		info.SubscriptionID = &id
	}
	if sess.Customer != nil {
		info.CustomerID = sess.Customer.ID
	}
	return info, nil
}

func (p *StripeProvider) VerifyWebhook(body []byte, headers http.Header) (*WebhookEvent, error) {
	if p.cfg.StripeWebhookSecret == "" {
		return nil, fmt.Errorf("paymentprovider: stripe webhook secret not configured")
	}
	sig := headers.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(body, sig, p.cfg.StripeWebhookSecret)
	if err != nil {
		return nil, err
	}
	return &WebhookEvent{
		ID:       event.ID,
		Type:     string(event.Type),
		Provider: ProviderStripe,
		Raw:      body,
		Payload:  event,
	}, nil
}

// GetStripeCheckoutSession loads a Checkout Session by id (for PI resolution on refund).
func GetStripeCheckoutSession(secretKey, sessionID string) (*stripe.CheckoutSession, error) {
	stripe.Key = secretKey
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("payment_intent")
	return checkoutsession.Get(sessionID, params)
}

func stripeCheckoutPaymentMethods(country string) []string {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "NL":
		return []string{"card", "ideal"}
	default:
		return nil
	}
}

func copyMetadata(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
