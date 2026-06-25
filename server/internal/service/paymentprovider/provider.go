// Package paymentprovider defines a provider-agnostic payment abstraction (plan 16.8).
package paymentprovider

import (
	"context"
	"net/http"
)

// ProviderName identifies a payment backend.
type ProviderName string

const (
	ProviderStripe ProviderName = "stripe"
	ProviderPayPal ProviderName = "paypal"
)

// Provider is the payment backend abstraction (plan 16.8 FR-1).
type Provider interface {
	Name() ProviderName
	CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error)
	CreateSubscription(ctx context.Context, req SubscriptionRequest) (*CheckoutResult, error)
	CancelSubscription(ctx context.Context, providerSubID string) error
	IssueRefund(ctx context.Context, providerTxnID string, amountCents *int) (*RefundResult, error)
	GetTransaction(ctx context.Context, providerTxnID string) (*TransactionInfo, error)
	VerifyWebhook(body []byte, headers http.Header) (*WebhookEvent, error)
}
