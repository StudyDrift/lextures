package paymentprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"

	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	repoPayments "github.com/lextures/lextures/server/internal/repos/payments"
)

// StartCheckoutRequest is the HTTP checkout payload.
type StartCheckoutRequest struct {
	UserID             uuid.UUID
	Email              string
	CourseID           *uuid.UUID
	Plan               string
	Provider           ProviderName
	Country            string
	PromoCode          string
	AffiliateCode      string
	SuccessURL         string
	CancelURL          string
	PlatformTaxEnabled bool
	TaxCode            string
}

// StartCheckout creates a hosted checkout session via the selected provider.
func StartCheckout(ctx context.Context, pool *pgxpool.Pool, cfg Config, req StartCheckoutRequest) (*CheckoutResult, error) {
	providerName, err := ResolveProvider(req.Provider, cfg)
	if err != nil {
		return nil, err
	}
	factory := Factory{}
	provider, err := factory.Build(providerName, cfg)
	if err != nil {
		return nil, err
	}
	checkoutKey := NewCheckoutKey()
	meta := map[string]string{
		"checkout_key": checkoutKey,
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
		meta["entitlement_type"] = repoBilling.TypeCoursePurchase
		if code := strings.TrimSpace(req.AffiliateCode); code != "" {
			meta["affiliate_code"] = code
		}
		taxCode := req.TaxCode
		if taxCode == "" {
			taxCode = "txcd_99999999"
		}
		if providerName == ProviderStripe {
			customerID, err := ensureStripeCustomer(ctx, pool, cfg, req.UserID, req.Email)
			if err != nil {
				return nil, err
			}
			meta["stripe_customer_id"] = customerID
		}
		result, err := provider.CreateCheckoutSession(ctx, CheckoutRequest{
			UserID:             req.UserID,
			Email:              req.Email,
			CourseID:           req.CourseID,
			CourseTitle:        price.Title,
			PriceCents:         price.PriceCents,
			Currency:           price.Currency,
			OrgID:              price.OrgID,
			AffiliateCode:      req.AffiliateCode,
			SuccessURL:         req.SuccessURL,
			CancelURL:          req.CancelURL,
			Country:            req.Country,
			PlatformTaxEnabled: req.PlatformTaxEnabled,
			TaxCode:            taxCode,
			Metadata:           meta,
		})
		if err != nil {
			return nil, err
		}
		_ = PendingTransaction(ctx, pool, req.UserID, req.CourseID, result)
		return result, nil
	case plan == "monthly" || plan == "annual":
		if providerName != ProviderStripe {
			return nil, fmt.Errorf("subscriptions require stripe")
		}
		customerID, err := ensureStripeCustomer(ctx, pool, cfg, req.UserID, req.Email)
		if err != nil {
			return nil, err
		}
		meta["stripe_customer_id"] = customerID
		entType := repoBilling.TypeSubscriptionMonthly
		priceID := cfg.StripeMonthlyPriceID
		if plan == "annual" {
			entType = repoBilling.TypeSubscriptionAnnual
			priceID = cfg.StripeAnnualPriceID
		}
		meta["entitlement_type"] = entType
		result, err := provider.CreateSubscription(ctx, SubscriptionRequest{
			UserID:     req.UserID,
			Email:      req.Email,
			Plan:       plan,
			PriceID:    priceID,
			SuccessURL: req.SuccessURL,
			CancelURL:  req.CancelURL,
			Metadata:   meta,
		})
		if err != nil {
			return nil, err
		}
		_ = PendingTransaction(ctx, pool, req.UserID, nil, result)
		return result, nil
	default:
		return nil, fmt.Errorf("course_id or plan required")
	}
}

func ensureStripeCustomer(ctx context.Context, pool *pgxpool.Pool, cfg Config, userID uuid.UUID, email string) (string, error) {
	existing, err := repoBilling.StripeCustomerID(ctx, pool, userID)
	if err != nil {
		return "", err
	}
	if existing != "" {
		return existing, nil
	}
	stripe.Key = cfg.StripeSecretKey
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

// PendingTransaction records a checkout session before redirect.
func PendingTransaction(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseID *uuid.UUID, result *CheckoutResult) error {
	if result == nil {
		return nil
	}
	_, _, err := repoPayments.CreateIdempotent(ctx, pool, repoPayments.CreateTransactionInput{
		UserID:         userID,
		CourseID:       courseID,
		Provider:       string(result.Provider),
		ProviderTxnID:  result.SessionID,
		IdempotencyKey: result.IdempotencyKey,
		AmountCents:    result.AmountCents,
		Currency:       result.Currency,
		Status:         repoPayments.StatusPending,
	})
	return err
}

// IssueProviderRefund calls the provider and updates local transaction status.
func IssueProviderRefund(ctx context.Context, pool *pgxpool.Pool, cfg Config, tx *repoPayments.Transaction, amountCents *int) (*RefundResult, error) {
	factory := Factory{}
	provider, err := factory.Build(ProviderName(tx.Provider), cfg)
	if err != nil {
		return nil, err
	}
	refund, err := provider.IssueRefund(ctx, tx.ProviderTxnID, amountCents)
	if err != nil {
		return nil, err
	}
	_ = repoPayments.UpdateStatus(ctx, pool, tx.ID, repoPayments.StatusRefunded)
	RecordTransaction(ProviderName(tx.Provider), repoPayments.StatusRefunded, tx.Currency)
	return refund, nil
}

// NewCheckoutKey returns a stable per-request checkout key.
func NewCheckoutKey() string {
	return uuid.New().String()
}

// ProcessPayPalJob fulfills PayPal webhook events.
func ProcessPayPalJob(ctx context.Context, pool *pgxpool.Pool, payload []byte) error {
	var event struct {
		ID        string `json:"id"`
		EventType string `json:"event_type"`
		Resource  struct {
			ID       string `json:"id"`
			CustomID string `json:"custom_id"`
			Amount   struct {
				Value        string `json:"value"`
				CurrencyCode string `json:"currency_code"`
			} `json:"amount"`
			SupplementaryData struct {
				RelatedIDs struct {
					OrderID string `json:"order_id"`
				} `json:"related_ids"`
			} `json:"supplementary_data"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}
	if event.EventType != "PAYMENT.CAPTURE.COMPLETED" {
		return nil
	}
	orderID := event.Resource.SupplementaryData.RelatedIDs.OrderID
	if orderID == "" {
		orderID = event.Resource.ID
	}
	idempotencyKey := event.ID
	if idempotencyKey == "" {
		idempotencyKey = event.Resource.ID
	}
	userID, courseID, customKey := parsePayPalCustomID(event.Resource.CustomID)
	if customKey != "" {
		idempotencyKey = customKey
	}
	amountCents := parsePayPalCents(event.Resource.Amount.Value)
	currency := strings.ToLower(event.Resource.Amount.CurrencyCode)
	_, _, err := repoPayments.CreateIdempotent(ctx, pool, repoPayments.CreateTransactionInput{
		UserID:         userID,
		CourseID:       courseID,
		Provider:       repoPayments.ProviderPayPal,
		ProviderTxnID:  orderID,
		IdempotencyKey: idempotencyKey,
		AmountCents:    amountCents,
		Currency:       currency,
		Status:         repoPayments.StatusCompleted,
	})
	if err != nil {
		return err
	}
	if userID != uuid.Nil {
		_, _, _ = repoBilling.CreateIdempotent(ctx, pool, repoBilling.CreateInput{
			UserID:          userID,
			EntitlementType: repoBilling.TypeCoursePurchase,
			CourseID:        courseID,
			StripeEventID:   idempotencyKey,
			AmountPaidCents: amountCents,
			Currency:        currency,
		})
	}
	RecordTransaction(ProviderPayPal, repoPayments.StatusCompleted, currency)
	return nil
}

func parsePayPalCents(value string) int {
	var dollars float64
	_, _ = fmt.Sscanf(value, "%f", &dollars)
	return int(dollars * 100)
}
