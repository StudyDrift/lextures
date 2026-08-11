package paymentprovider

import (
	"time"

	"github.com/google/uuid"
)

// DefaultDigitalTaxCode is Stripe's "Software as a service (SaaS) - personal use".
// Eligible for Managed Payments; replaces the legacy catch-all txcd_99999999.
const DefaultDigitalTaxCode = "txcd_10103000"

// NormalizeTaxCode returns a Managed Payments–eligible product tax code.
func NormalizeTaxCode(code string) string {
	if code == "" || code == "txcd_99999999" {
		return DefaultDigitalTaxCode
	}
	return code
}

// CheckoutRequest starts a hosted checkout flow.
type CheckoutRequest struct {
	UserID             uuid.UUID
	Email              string
	CourseID           *uuid.UUID
	CourseTitle        string
	PriceCents         int
	Currency           string
	OrgID              uuid.UUID
	Plan               string
	PromoCode          string
	AffiliateCode      string
	SuccessURL         string
	CancelURL          string
	Country            string
	PlatformTaxEnabled bool
	// TaxCode is a Stripe product tax code (e.g. txcd_10103000 for SaaS personal).
	// Empty or the legacy catch-all txcd_99999999 is replaced with DefaultDigitalTaxCode
	// so Checkout works with Managed Payments accounts.
	TaxCode  string
	Metadata map[string]string
	// First-party course coupon (plan MKTC.3). When HasFirstPartyCoupon is true,
	// UnitAmount is ChargedCents and Stripe promotion codes are disabled.
	HasFirstPartyCoupon bool
	DiscountCents       int
	ChargedCents        int // discounted unit amount (pre-tax)
}

// UnitAmount is the line-item amount sent to the payment provider.
func (r CheckoutRequest) UnitAmount() int {
	if r.HasFirstPartyCoupon {
		return r.ChargedCents
	}
	return r.PriceCents
}

// SubscriptionRequest starts a recurring checkout.
type SubscriptionRequest struct {
	UserID     uuid.UUID
	Email      string
	Plan       string
	PriceID    string
	SuccessURL string
	CancelURL  string
	Metadata   map[string]string
}

// CheckoutResult is a redirect to a hosted provider checkout page.
type CheckoutResult struct {
	SessionID      string
	CheckoutURL    string
	Provider       ProviderName
	IdempotencyKey string
	AmountCents    int
	Currency       string
}

// RefundResult summarizes a refund operation.
type RefundResult struct {
	RefundID    string
	AmountCents int
	Currency    string
	Status      string
}

// TransactionInfo is provider-side transaction metadata.
type TransactionInfo struct {
	ProviderTxnID  string
	AmountCents    int
	Currency       string
	Status         string
	SubscriptionID *string
	CustomerID     string
	Metadata       map[string]string
}

// WebhookEvent is a verified inbound provider event.
type WebhookEvent struct {
	ID        string
	Type      string
	Provider  ProviderName
	Raw       []byte
	Payload   any
	CreatedAt time.Time
}
