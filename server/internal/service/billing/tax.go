package billing

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v81"
	taxcalc "github.com/stripe/stripe-go/v81/tax/calculation"

	"github.com/lextures/lextures/server/internal/crypto"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
)

// TaxAddress is the customer billing location for tax calculation.
type TaxAddress struct {
	Country string `json:"country"`
	Region  string `json:"region,omitempty"`
	Line1   string `json:"line1,omitempty"`
	City    string `json:"city,omitempty"`
	Postal  string `json:"postalCode,omitempty"`
}

// TaxQuoteRequest is the checkout quote payload.
type TaxQuoteRequest struct {
	CourseID *uuid.UUID
	Plan     string
	Address  TaxAddress
	TaxID    string
	TaxIDType string
}

// TaxQuoteLine is one line in a tax quote response.
type TaxQuoteLine struct {
	Label       string `json:"label"`
	AmountCents int    `json:"amountCents"`
}

// TaxQuoteResult is the pre-checkout tax breakdown.
type TaxQuoteResult struct {
	SubtotalCents   int            `json:"subtotalCents"`
	TaxAmountCents  int            `json:"taxAmountCents"`
	TotalCents      int            `json:"totalCents"`
	Currency        string         `json:"currency"`
	TaxRate         *float64       `json:"taxRate,omitempty"`
	TaxJurisdiction string         `json:"taxJurisdiction,omitempty"`
	TaxType         string         `json:"taxType"`
	TaxInclusive    bool           `json:"taxInclusive"`
	ReverseCharge   bool           `json:"reverseCharge"`
	Lines           []TaxQuoteLine `json:"lines"`
	CalculationID   string         `json:"calculationId,omitempty"`
}

// TaxIDValidationResult reports reverse-charge applicability.
type TaxIDValidationResult struct {
	Valid         bool   `json:"valid"`
	ReverseCharge bool   `json:"reverseCharge"`
	TaxIDType     string `json:"taxIdType,omitempty"`
	Message       string `json:"message,omitempty"`
}

// TaxEnabledForOrg reports whether tax collection is active for an org.
func TaxEnabledForOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, platformTaxEnabled bool) (bool, *repoBilling.OrgTaxSettings, error) {
	if !platformTaxEnabled {
		return false, nil, nil
	}
	settings, err := repoBilling.GetOrgTaxSettings(ctx, pool, orgID)
	if err != nil {
		return false, nil, err
	}
	return settings.Enabled, settings, nil
}

// ComputeTaxQuote returns a Stripe Tax calculation for checkout preview.
func ComputeTaxQuote(ctx context.Context, pool *pgxpool.Pool, cfg StripeConfig, orgID uuid.UUID, platformTaxEnabled bool, req TaxQuoteRequest) (*TaxQuoteResult, error) {
	if !cfg.IsConfigured() {
		return nil, errors.New("stripe not configured")
	}
	enabled, settings, err := TaxEnabledForOrg(ctx, pool, orgID, platformTaxEnabled)
	if err != nil {
		return nil, err
	}

	subtotal, currency, label, err := quoteSubtotal(ctx, pool, cfg, req)
	if err != nil {
		return nil, err
	}

	result := &TaxQuoteResult{
		SubtotalCents:  subtotal,
		TaxAmountCents: 0,
		TotalCents:     subtotal,
		Currency:       currency,
		TaxType:        repoBilling.TaxTypeNone,
		TaxInclusive:   settings != nil && settings.PriceDisplay == repoBilling.PriceDisplayInc,
		Lines: []TaxQuoteLine{
			{Label: label, AmountCents: subtotal},
		},
	}

	if !enabled || strings.TrimSpace(req.Address.Country) == "" {
		result.Lines = append(result.Lines, TaxQuoteLine{Label: "Tax", AmountCents: 0})
		return result, nil
	}

	stripe.Key = cfg.SecretKey
	taxBehavior := "exclusive"
	if settings.PriceDisplay == repoBilling.PriceDisplayInc {
		taxBehavior = "inclusive"
	}
	lineItem := &stripe.TaxCalculationLineItemParams{
		Amount:       stripe.Int64(int64(subtotal)),
		Reference:    stripe.String("item_1"),
		TaxCode:      stripe.String(settings.DefaultTaxCategory),
		TaxBehavior:  stripe.String(taxBehavior),
		Quantity:     stripe.Int64(1),
	}
	custDetails := &stripe.TaxCalculationCustomerDetailsParams{
		Address: &stripe.AddressParams{
			Country:    stripe.String(strings.ToUpper(req.Address.Country)),
			State:      stripe.String(req.Address.Region),
			Line1:      stripe.String(req.Address.Line1),
			City:       stripe.String(req.Address.City),
			PostalCode: stripe.String(req.Address.Postal),
		},
		AddressSource: stripe.String("billing"),
	}
	if taxID := strings.TrimSpace(req.TaxID); taxID != "" {
		idType := inferTaxIDType(req.TaxIDType, req.Address.Country)
		custDetails.TaxIDs = []*stripe.TaxCalculationCustomerDetailsTaxIDParams{
			{Type: stripe.String(idType), Value: stripe.String(taxID)},
		}
	}

	calc, err := taxcalc.New(&stripe.TaxCalculationParams{
		Currency:        stripe.String(currency),
		LineItems:       []*stripe.TaxCalculationLineItemParams{lineItem},
		CustomerDetails: custDetails,
	})
	if err != nil {
		RecordTaxCalcFailure()
		return nil, fmt.Errorf("tax calculation failed: %w", err)
	}

	taxAmount := int(calc.TaxAmountExclusive)
	if settings.PriceDisplay == repoBilling.PriceDisplayInc {
		taxAmount = int(calc.TaxAmountInclusive)
	}
	total := subtotal + taxAmount
	if settings.PriceDisplay == repoBilling.PriceDisplayInc {
		total = subtotal
		taxAmount = int(calc.TaxAmountInclusive)
	}

	jurisdiction, taxType, rate, reverseCharge := parseTaxBreakdown(calc, req.Address.Country)
	result.TaxAmountCents = taxAmount
	result.TotalCents = total
	result.TaxJurisdiction = jurisdiction
	result.TaxType = taxType
	result.TaxRate = rate
	result.ReverseCharge = reverseCharge
	result.CalculationID = calc.ID
	result.Lines = append(result.Lines, TaxQuoteLine{Label: taxLineLabel(taxType, reverseCharge), AmountCents: taxAmount})
	RecordTaxCollected(jurisdiction, taxAmount)
	if reverseCharge {
		RecordReverseCharge()
	}
	return result, nil
}

// ValidateTaxID checks a VAT/GST ID via Stripe Tax calculation.
func ValidateTaxID(ctx context.Context, pool *pgxpool.Pool, cfg StripeConfig, orgID uuid.UUID, platformTaxEnabled bool, address TaxAddress, taxID, taxIDType string) (*TaxIDValidationResult, error) {
	taxID = strings.TrimSpace(taxID)
	if taxID == "" {
		return &TaxIDValidationResult{Valid: false, Message: "Tax ID is required."}, nil
	}
	if strings.TrimSpace(address.Country) == "" {
		return &TaxIDValidationResult{Valid: false, Message: "Country is required."}, nil
	}
	quote, err := ComputeTaxQuote(ctx, pool, cfg, orgID, platformTaxEnabled, TaxQuoteRequest{
		Plan:      "monthly",
		Address:   address,
		TaxID:     taxID,
		TaxIDType: taxIDType,
	})
	if err != nil {
		return &TaxIDValidationResult{Valid: false, Message: "Could not validate tax ID."}, nil
	}
	return &TaxIDValidationResult{
		Valid:         true,
		ReverseCharge: quote.ReverseCharge,
		TaxIDType:     inferTaxIDType(taxIDType, address.Country),
		Message:       reverseChargeMessage(quote.ReverseCharge),
	}, nil
}

// PersistCheckoutTax stores tax metadata from a completed Stripe checkout session.
func PersistCheckoutTax(ctx context.Context, pool *pgxpool.Pool, stripeEventID string, sess *stripe.CheckoutSession, orgID uuid.UUID, platformTaxEnabled bool) error {
	enabled, settings, err := TaxEnabledForOrg(ctx, pool, orgID, platformTaxEnabled)
	if err != nil || !enabled {
		return err
	}

	taxAmount := 0
	subtotal := int(sess.AmountSubtotal)
	if sess.TotalDetails != nil {
		taxAmount = int(sess.TotalDetails.AmountTax)
	}
	if subtotal == 0 {
		subtotal = int(sess.AmountTotal) - taxAmount
	}

	jurisdiction := ""
	taxType := repoBilling.TaxTypeNone
	var rate *float64
	reverseCharge := false
	if sess.TotalDetails != nil && sess.TotalDetails.Breakdown != nil {
		for _, t := range sess.TotalDetails.Breakdown.Taxes {
			jurisdiction = string(t.Rate.Jurisdiction)
			taxType = mapStripeTaxType(jurisdiction)
			if t.Rate != nil && t.Rate.Percentage > 0 {
				pct := t.Rate.Percentage
				rate = &pct
			}
			if t.TaxabilityReason == stripe.CheckoutSessionTotalDetailsBreakdownTaxTaxabilityReasonReverseCharge {
				reverseCharge = true
			}
		}
	}
	if taxAmount > 0 && taxType == repoBilling.TaxTypeNone {
		taxType = repoBilling.TaxTypeSalesTax
	}

	country := ""
	region := ""
	if sess.CustomerDetails != nil && sess.CustomerDetails.Address != nil {
		country = sess.CustomerDetails.Address.Country
		region = sess.CustomerDetails.Address.State
	}

	var taxIDEnc string
	if sess.CustomerDetails != nil && len(sess.CustomerDetails.TaxIDs) > 0 {
		if enc, err := crypto.EncryptString(sess.CustomerDetails.TaxIDs[0].Value); err == nil {
			taxIDEnc = enc
		}
	}

	tax := repoBilling.TaxFields{
		SubtotalCents:          subtotal,
		TaxAmountCents:         taxAmount,
		TaxRate:                rate,
		TaxJurisdiction:        jurisdiction,
		TaxType:                taxType,
		TaxInclusive:           settings.PriceDisplay == repoBilling.PriceDisplayInc,
		CustomerCountry:        country,
		CustomerRegion:         region,
		CustomerTaxIDEnc:       taxIDEnc,
		ReverseCharge:          reverseCharge,
		StripeTaxCalculationID: "",
	}
	return repoBilling.UpdateEntitlementTax(ctx, pool, stripeEventID, tax)
}

// IssueTaxInvoice creates an invoice record for a completed entitlement.
func IssueTaxInvoice(ctx context.Context, pool *pgxpool.Pool, entitlementID uuid.UUID, orgID uuid.UUID) (*repoBilling.TaxInvoice, error) {
	ent, err := repoBilling.GetEntitlementWithTax(ctx, pool, entitlementID)
	if err != nil || ent == nil {
		return nil, err
	}
	if ent.InvoiceID != nil {
		return repoBilling.GetTaxInvoiceByID(ctx, pool, *ent.InvoiceID)
	}
	invoiceNumber := fmt.Sprintf("INV-%s-%s", time.Now().UTC().Format("20060102"), entitlementID.String()[:8])
	return repoBilling.CreateTaxInvoice(ctx, pool, entitlementID, invoiceNumber, "")
}

func quoteSubtotal(ctx context.Context, pool *pgxpool.Pool, cfg StripeConfig, req TaxQuoteRequest) (int, string, string, error) {
	switch {
	case req.CourseID != nil:
		price, err := repoBilling.CoursePriceByID(ctx, pool, *req.CourseID)
		if err != nil {
			return 0, "", "", err
		}
		if price == nil {
			return 0, "", "", fmt.Errorf("course not found")
		}
		if price.PriceCents <= 0 {
			return 0, "", "", fmt.Errorf("course is free")
		}
		currency := strings.ToLower(price.Currency)
		if currency == "" {
			currency = "usd"
		}
		return price.PriceCents, currency, price.Title, nil
	case req.Plan == "monthly" || req.Plan == "annual":
		// Subscription quote uses configured price; amount resolved at Stripe checkout.
		return 1000, "usd", "Subscription", nil
	default:
		return 0, "", "", fmt.Errorf("course_id or plan required")
	}
}

func parseTaxBreakdown(calc *stripe.TaxCalculation, country string) (jurisdiction, taxType string, rate *float64, reverseCharge bool) {
	jurisdiction = strings.ToUpper(strings.TrimSpace(country))
	if calc == nil || len(calc.TaxBreakdown) == 0 {
		return jurisdiction, repoBilling.TaxTypeNone, nil, false
	}
	bd := calc.TaxBreakdown[0]
	taxType = mapStripeTaxType(jurisdiction)
	if bd.TaxRateDetails != nil && bd.TaxRateDetails.PercentageDecimal != "" {
		if pct, err := strconv.ParseFloat(bd.TaxRateDetails.PercentageDecimal, 64); err == nil {
			rate = &pct
		}
	}
	if bd.TaxabilityReason == stripe.TaxCalculationTaxBreakdownTaxabilityReasonReverseCharge {
		reverseCharge = true
	}
	return jurisdiction, taxType, rate, reverseCharge
}

func mapStripeTaxType(jurisdiction string) string {
	j := strings.ToUpper(strings.TrimSpace(jurisdiction))
	if j == "" {
		return repoBilling.TaxTypeNone
	}
	// EU/UK VAT jurisdictions
	euVAT := map[string]bool{
		"GB": true, "DE": true, "FR": true, "IT": true, "ES": true, "NL": true,
		"BE": true, "AT": true, "IE": true, "PT": true, "PL": true, "SE": true,
	}
	if euVAT[j] || strings.HasPrefix(j, "EU") {
		return repoBilling.TaxTypeVAT
	}
	if j == "AU" || j == "NZ" || j == "CA" || j == "IN" {
		return repoBilling.TaxTypeGST
	}
	if len(j) == 2 && j[0] >= 'A' && j[0] <= 'Z' {
		return repoBilling.TaxTypeSalesTax
	}
	return repoBilling.TaxTypeSalesTax
}

func inferTaxIDType(explicit, country string) string {
	if t := strings.TrimSpace(explicit); t != "" {
		return t
	}
	switch strings.ToUpper(country) {
	case "GB":
		return "gb_vat"
	case "DE", "FR", "IT", "ES", "NL", "BE", "AT", "IE", "PT", "PL", "SE":
		return "eu_vat"
	case "AU":
		return "au_abn"
	case "CA":
		return "ca_gst_hst"
	case "NZ":
		return "nz_gst"
	default:
		return "eu_vat"
	}
}

func taxLineLabel(taxType string, reverseCharge bool) string {
	if reverseCharge {
		return "VAT (reverse charge)"
	}
	switch taxType {
	case repoBilling.TaxTypeVAT:
		return "VAT"
	case repoBilling.TaxTypeGST:
		return "GST"
	case repoBilling.TaxTypeSalesTax:
		return "Sales tax"
	default:
		return "Tax"
	}
}

func reverseChargeMessage(reverse bool) string {
	if reverse {
		return "Reverse charge applies — no VAT will be charged."
	}
	return "Tax ID accepted."
}