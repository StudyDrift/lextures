package paymentprovider

import (
	"time"

	"github.com/google/uuid"
)

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
	TaxCode            string
	Metadata           map[string]string
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
