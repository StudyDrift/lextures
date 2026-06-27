package paymentprovider

import (
	"context"
)

// MockProvider is a test double for Provider.
type MockProvider struct {
	NameVal            ProviderName
	CheckoutResult     *CheckoutResult
	CheckoutErr        error
	SubscriptionResult *CheckoutResult
	SubscriptionErr    error
	CancelErr          error
	RefundResult       *RefundResult
	RefundErr          error
	TransactionInfo    *TransactionInfo
	TransactionErr     error
	WebhookEvent       *WebhookEvent
	WebhookErr         error
}

func (m *MockProvider) CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error) {
	if m.CheckoutErr != nil {
		return nil, m.CheckoutErr
	}
	return m.CheckoutResult, nil
}
