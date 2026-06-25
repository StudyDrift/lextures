package paymentprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PayPalProvider implements Provider using PayPal Orders API v2.
type PayPalProvider struct {
	cfg        Config
	httpClient *http.Client
	tokenMu    sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewPayPalProvider returns a PayPal-backed provider.
func NewPayPalProvider(cfg Config) *PayPalProvider {
	return &PayPalProvider{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *PayPalProvider) Name() ProviderName { return ProviderPayPal }

func (p *PayPalProvider) baseURL() string {
	if p.cfg.PayPalSandbox {
		return "https://api-m.sandbox.paypal.com"
	}
	return "https://api-m.paypal.com"
}

func (p *PayPalProvider) CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error) {
	token, err := p.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	currency := strings.ToUpper(req.Currency)
	if currency == "" {
		currency = "USD"
	}
	idempotencyKey := fmt.Sprintf("paypal:checkout:%s:%s", req.UserID, strings.TrimSpace(req.Metadata["checkout_key"]))
	customID := paypalCustomID(req.UserID, req.CourseID, idempotencyKey)
	body := map[string]any{
		"intent": "CAPTURE",
		"purchase_units": []map[string]any{{
			"reference_id": req.UserID.String(),
			"description":  req.CourseTitle,
			"custom_id":    customID,
			"amount": map[string]any{
				"currency_code": currency,
				"value":         formatPayPalAmount(req.PriceCents),
			},
		}},
		"payment_source": map[string]any{
			"paypal": map[string]any{
				"experience_context": map[string]any{
					"return_url": req.SuccessURL,
					"cancel_url": req.CancelURL,
				},
			},
		},
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL()+"/v2/checkout/orders", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paymentprovider: paypal create order: %s", strings.TrimSpace(string(respBody)))
	}
	var parsed struct {
		ID    string `json:"id"`
		Links []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	checkoutURL := ""
	for _, link := range parsed.Links {
		if link.Rel == "payer-action" || link.Rel == "approve" {
			checkoutURL = link.Href
			break
		}
	}
	if checkoutURL == "" {
		return nil, fmt.Errorf("paymentprovider: paypal approval url missing")
	}
	return &CheckoutResult{
		SessionID:      parsed.ID,
		CheckoutURL:    checkoutURL,
		Provider:       ProviderPayPal,
		IdempotencyKey: idempotencyKey,
		AmountCents:    req.PriceCents,
		Currency:       strings.ToLower(currency),
	}, nil
}

func (p *PayPalProvider) CreateSubscription(ctx context.Context, req SubscriptionRequest) (*CheckoutResult, error) {
	return nil, fmt.Errorf("paymentprovider: paypal subscriptions not implemented")
}

func (p *PayPalProvider) CancelSubscription(ctx context.Context, providerSubID string) error {
	return fmt.Errorf("paymentprovider: paypal subscriptions not implemented")
}

func (p *PayPalProvider) IssueRefund(ctx context.Context, providerTxnID string, amountCents *int) (*RefundResult, error) {
	token, err := p.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	body := map[string]any{}
	if amountCents != nil && *amountCents > 0 {
		body["amount"] = map[string]any{
			"value":         formatPayPalAmount(*amountCents),
			"currency_code": "USD",
		}
	}
	raw, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/v2/payments/captures/%s/refund", p.baseURL(), providerTxnID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paymentprovider: paypal refund: %s", strings.TrimSpace(string(respBody)))
	}
	var parsed struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount struct {
			Value        string `json:"value"`
			CurrencyCode string `json:"currency_code"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	return &RefundResult{
		RefundID: parsed.ID,
		Status:   parsed.Status,
		Currency: strings.ToLower(parsed.Amount.CurrencyCode),
	}, nil
}

func (p *PayPalProvider) GetTransaction(ctx context.Context, providerTxnID string) (*TransactionInfo, error) {
	token, err := p.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL()+"/v2/checkout/orders/"+providerTxnID, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paymentprovider: paypal get order: %s", strings.TrimSpace(string(respBody)))
	}
	var parsed struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		PurchaseUnits []struct {
			CustomID string `json:"custom_id"`
			Amount   struct {
				Value        string `json:"value"`
				CurrencyCode string `json:"currency_code"`
			} `json:"amount"`
			Payments struct {
				Captures []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
					Amount struct {
						Value        string `json:"value"`
						CurrencyCode string `json:"currency_code"`
					} `json:"amount"`
				} `json:"captures"`
			} `json:"payments"`
		} `json:"purchase_units"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	info := &TransactionInfo{
		ProviderTxnID: parsed.ID,
		Status:        parsed.Status,
		Metadata:      map[string]string{},
	}
	if len(parsed.PurchaseUnits) > 0 {
		unit := parsed.PurchaseUnits[0]
		info.Metadata["custom_id"] = unit.CustomID
		if len(unit.Payments.Captures) > 0 {
			cap := unit.Payments.Captures[0]
			info.ProviderTxnID = cap.ID
			info.Status = cap.Status
		}
	}
	return info, nil
}

func (p *PayPalProvider) VerifyWebhook(body []byte, headers http.Header) (*WebhookEvent, error) {
	var payload struct {
		ID         string `json:"id"`
		EventType  string `json:"event_type"`
		CreateTime string `json:"create_time"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.ID == "" {
		return nil, fmt.Errorf("paymentprovider: paypal webhook missing event id")
	}
	if p.cfg.PayPalWebhookID != "" {
		if err := p.verifyWebhookSignature(ctxBackground(), body, headers); err != nil {
			return nil, err
		}
	}
	return &WebhookEvent{
		ID:       payload.ID,
		Type:     payload.EventType,
		Provider: ProviderPayPal,
		Raw:      body,
		Payload:  payload,
	}, nil
}

func (p *PayPalProvider) verifyWebhookSignature(ctx context.Context, body []byte, headers http.Header) error {
	token, err := p.ensureAccessToken(ctx)
	if err != nil {
		return err
	}
	reqBody := map[string]any{
		"auth_algo":         headers.Get("PAYPAL-AUTH-ALGO"),
		"cert_url":          headers.Get("PAYPAL-CERT-URL"),
		"transmission_id":   headers.Get("PAYPAL-TRANSMISSION-ID"),
		"transmission_sig":  headers.Get("PAYPAL-TRANSMISSION-SIG"),
		"transmission_time": headers.Get("PAYPAL-TRANSMISSION-TIME"),
		"webhook_id":        p.cfg.PayPalWebhookID,
		"webhook_event":     json.RawMessage(body),
	}
	raw, _ := json.Marshal(reqBody)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL()+"/v1/notifications/verify-webhook-signature", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	var parsed struct {
		VerificationStatus string `json:"verification_status"`
	}
	_ = json.Unmarshal(respBody, &parsed)
	if parsed.VerificationStatus != "SUCCESS" {
		return fmt.Errorf("paymentprovider: invalid paypal webhook signature")
	}
	return nil
}

func (p *PayPalProvider) ensureAccessToken(ctx context.Context) (string, error) {
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()
	if p.accessToken != "" && time.Now().Before(p.tokenExpiry) {
		return p.accessToken, nil
	}
	body := "grant_type=client_credentials"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL()+"/v1/oauth2/token", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.SetBasicAuth(p.cfg.PayPalClientID, p.cfg.PayPalClientSecret)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("paymentprovider: paypal oauth: %s", strings.TrimSpace(string(respBody)))
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	p.accessToken = parsed.AccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(parsed.ExpiresIn-60) * time.Second)
	return p.accessToken, nil
}

func formatPayPalAmount(cents int) string {
	return fmt.Sprintf("%.2f", float64(cents)/100.0)
}

func ctxBackground() context.Context {
	return context.Background()
}

func paypalCustomID(userID uuid.UUID, courseID *uuid.UUID, key string) string {
	course := ""
	if courseID != nil {
		course = courseID.String()
	}
	return fmt.Sprintf("user:%s|course:%s|key:%s", userID, course, key)
}

func parsePayPalCustomID(customID string) (uuid.UUID, *uuid.UUID, string) {
	parts := strings.Split(customID, "|")
	var userID uuid.UUID
	var courseID *uuid.UUID
	key := customID
	for _, part := range parts {
		if strings.HasPrefix(part, "user:") {
			userID, _ = uuid.Parse(strings.TrimPrefix(part, "user:"))
		}
		if strings.HasPrefix(part, "course:") {
			raw := strings.TrimPrefix(part, "course:")
			if raw != "" {
				if id, err := uuid.Parse(raw); err == nil {
					courseID = &id
				}
			}
		}
		if strings.HasPrefix(part, "key:") {
			key = strings.TrimPrefix(part, "key:")
		}
	}
	return userID, courseID, key
}
