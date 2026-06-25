package paymentprovider

import (
	"context"
	"net/http"
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

func (m *MockProvider) Name() ProviderName {
	if m.NameVal != "" {
		return m.NameVal
	}
	return ProviderStripe
}

func (m *MockProvider) CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error) {
	if m.CheckoutErr != nil {
		return nil, m.CheckoutErr
	}
	return m.CheckoutResult, nil
}

func (m *MockProvider) CreateSubscription(ctx context.Context, req SubscriptionRequest) (*CheckoutResult, error) {
	if m.SubscriptionErr != nil {
		return nil, m.SubscriptionErr
	}
	return m.SubscriptionResult, nil
}

func (m *MockProvider) CancelSubscription(ctx context.Context, providerSubID string) error {
	return m.CancelErr
}

func (m *MockProvider) IssueRefund(ctx context.Context, providerTxnID string, amountCents *int) (*RefundResult, error) {
	if m.RefundErr != nil {
		return nil, m.RefundErr
	}
	return m.RefundResult, nil
}

func (m *MockProvider) GetTransaction(ctx context.Context, providerTxnID string) (*TransactionInfo, error) {
	if m.TransactionErr != nil {
		return nil, m.TransactionErr
	}
	return m.TransactionInfo, nil
}

func (m *MockProvider) VerifyWebhook(body []byte, headers http.Header) (*WebhookEvent, error) {
	if m.WebhookErr != nil {
		return nil, m.WebhookErr
	}
	return m.WebhookEvent, nil
}
