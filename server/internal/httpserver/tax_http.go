package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/crypto"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
)

func (d Deps) taxFeatureOff(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFStripeBilling || !cfg.FFTaxCollection {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Tax collection is not enabled.")
		return true
	}
	return false
}

func (d Deps) registerTaxRoutes(r chi.Router) {
	r.Post("/api/v1/checkout/quote", d.handleCheckoutQuote())
	r.Post("/api/v1/checkout/tax-id", d.handleCheckoutTaxID())
	r.Get("/api/v1/orgs/{orgId}/tax/settings", d.handleOrgTaxSettings())
	r.Put("/api/v1/orgs/{orgId}/tax/settings", d.handleOrgTaxSettings())
	r.Get("/api/v1/orgs/{orgId}/tax/report", d.handleOrgTaxReport())
	r.Get("/api/v1/invoices/{invoiceId}", d.handleTaxInvoiceDownload())
}

func (d Deps) handleCheckoutQuote() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.billingFeatureOff(w) || d.taxFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		if !d.checkBillingCheckoutRateLimit(userID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Rate limit exceeded.")
			return
		}
		var body struct {
			CourseID *string                `json:"courseId"`
			Plan     string                 `json:"plan"`
			Address  svcBilling.TaxAddress  `json:"address"`
			TaxID    string                 `json:"taxId"`
			TaxIDType string                `json:"taxIdType"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var courseID *uuid.UUID
		var orgID uuid.UUID
		if body.CourseID != nil && strings.TrimSpace(*body.CourseID) != "" {
			id, err := uuid.Parse(strings.TrimSpace(*body.CourseID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
				return
			}
			courseID = &id
			price, err := repoBilling.CoursePriceByID(r.Context(), d.Pool, id)
			if err != nil || price == nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Course not found.")
				return
			}
			orgID = price.OrgID
		} else {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "courseId is required for tax quote.")
			return
		}
		cfg := svcBilling.ConfigFrom(d.effectiveConfig())
		if !cfg.IsConfigured() {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Stripe is not configured.")
			return
		}
		result, err := svcBilling.ComputeTaxQuote(r.Context(), d.Pool, cfg, orgID, d.effectiveConfig().FFTaxCollection, svcBilling.TaxQuoteRequest{
			CourseID:  courseID,
			Plan:      body.Plan,
			Address:   body.Address,
			TaxID:     body.TaxID,
			TaxIDType: body.TaxIDType,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not compute tax quote.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func (d Deps) handleCheckoutTaxID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if d.billingFeatureOff(w) || d.taxFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		var body struct {
			CourseID  *string               `json:"courseId"`
			Address   svcBilling.TaxAddress `json:"address"`
			TaxID     string                `json:"taxId"`
			TaxIDType string                `json:"taxIdType"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var orgID uuid.UUID
		if body.CourseID != nil {
			id, err := uuid.Parse(strings.TrimSpace(*body.CourseID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
				return
			}
			price, err := repoBilling.CoursePriceByID(r.Context(), d.Pool, id)
			if err != nil || price == nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Course not found.")
				return
			}
			orgID = price.OrgID
		}
		cfg := svcBilling.ConfigFrom(d.effectiveConfig())
		result, err := svcBilling.ValidateTaxID(r.Context(), d.Pool, cfg, orgID, d.effectiveConfig().FFTaxCollection, body.Address, body.TaxID, body.TaxIDType)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not validate tax ID.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func (d Deps) handleOrgTaxSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		if d.taxFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		switch r.Method {
		case http.MethodGet:
			if _, ok := d.orgReadAccess(w, r, orgID); !ok {
				return
			}
			settings, err := repoBilling.GetOrgTaxSettings(r.Context(), d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load tax settings.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(orgTaxSettingsJSON(settings))

		case http.MethodPut:
			if _, _, ok := d.adminOrgOrUnitAccess(w, r, orgID); !ok {
				return
			}
			var body struct {
				Enabled                 bool     `json:"enabled"`
				RegisteredJurisdictions []string `json:"registeredJurisdictions"`
				DefaultTaxCategory      string   `json:"defaultTaxCategory"`
				PriceDisplay            string   `json:"priceDisplay"`
				FilingMode              string   `json:"filingMode"`
				RecordRetentionYears    int      `json:"recordRetentionYears"`
				SellerName              string   `json:"sellerName"`
				SellerAddress           string   `json:"sellerAddress"`
				SellerTaxID             string   `json:"sellerTaxId"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			if body.PriceDisplay != "" && body.PriceDisplay != repoBilling.PriceDisplayInc && body.PriceDisplay != repoBilling.PriceDisplayExc {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "priceDisplay must be inclusive or exclusive.")
				return
			}
			settings := repoBilling.OrgTaxSettings{
				OrgID:                   orgID,
				Enabled:                 body.Enabled,
				RegisteredJurisdictions: body.RegisteredJurisdictions,
				DefaultTaxCategory:      body.DefaultTaxCategory,
				PriceDisplay:            body.PriceDisplay,
				FilingMode:              body.FilingMode,
				RecordRetentionYears:    body.RecordRetentionYears,
				SellerName:              body.SellerName,
				SellerAddress:           body.SellerAddress,
				SellerTaxID:             body.SellerTaxID,
			}
			if err := repoBilling.UpsertOrgTaxSettings(r.Context(), d.Pool, settings); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save tax settings.")
				return
			}
			saved, _ := repoBilling.GetOrgTaxSettings(r.Context(), d.Pool, orgID)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(orgTaxSettingsJSON(saved))

		default:
			w.Header().Set("Allow", "GET, PUT")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

func orgTaxSettingsJSON(s *repoBilling.OrgTaxSettings) map[string]any {
	if s == nil {
		return map[string]any{}
	}
	return map[string]any{
		"orgId":                   s.OrgID.String(),
		"enabled":                 s.Enabled,
		"registeredJurisdictions": s.RegisteredJurisdictions,
		"defaultTaxCategory":      s.DefaultTaxCategory,
		"priceDisplay":            s.PriceDisplay,
		"filingMode":              s.FilingMode,
		"recordRetentionYears":    s.RecordRetentionYears,
		"sellerName":              s.SellerName,
		"sellerAddress":           s.SellerAddress,
		"sellerTaxId":             s.SellerTaxID,
	}
}

func (d Deps) handleOrgTaxReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		if d.taxFeatureOff(w) {
			return
		}
		if _, _, ok := d.adminOrgOrUnitAccess(w, r, orgID); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		period := strings.TrimSpace(r.URL.Query().Get("period"))
		jurisdiction := strings.TrimSpace(r.URL.Query().Get("jurisdiction"))
		from, to, err := parseTaxReportPeriod(period)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid period; use YYYY-MM.")
			return
		}
		rows, err := repoBilling.TaxReport(r.Context(), d.Pool, orgID, from, to, jurisdiction)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not generate tax report.")
			return
		}
		format := strings.TrimSpace(r.URL.Query().Get("format"))
		if format == "csv" {
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=tax-report.csv")
			_, _ = w.Write([]byte("jurisdiction,tax_type,transaction_count,tax_collected_cents,subtotal_cents\n"))
			for _, row := range rows {
				line := row.Jurisdiction + "," + row.TaxType + "," +
					itoa(row.TransactionCount) + "," +
					itoa64(row.TaxCollectedCents) + "," +
					itoa64(row.SubtotalCents) + "\n"
				_, _ = w.Write([]byte(line))
			}
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			items = append(items, map[string]any{
				"jurisdiction":      row.Jurisdiction,
				"taxType":           row.TaxType,
				"transactionCount":  row.TransactionCount,
				"taxCollectedCents": row.TaxCollectedCents,
				"subtotalCents":     row.SubtotalCents,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"period":  period,
			"from":    from.UTC().Format(time.RFC3339),
			"to":      to.UTC().Format(time.RFC3339),
			"rows":    items,
		})
	}
}

func (d Deps) handleTaxInvoiceDownload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.billingFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		invID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "invoiceId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid invoice id.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		inv, err := repoBilling.GetTaxInvoiceByID(r.Context(), d.Pool, invID)
		if err != nil || inv == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Invoice not found.")
			return
		}
		ent, err := repoBilling.GetEntitlementWithTax(r.Context(), d.Pool, inv.EntitlementID)
		if err != nil || ent == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Invoice not found.")
			return
		}
		var orgID uuid.UUID
		if ent.CourseID != nil {
			price, _ := repoBilling.CoursePriceByID(r.Context(), d.Pool, *ent.CourseID)
			if price != nil {
				orgID = price.OrgID
			}
		}
		if ent.UserID != userID {
			if orgID == uuid.Nil {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
				return
			}
			if _, _, ok := d.adminOrgOrUnitAccess(w, r, orgID); !ok {
				return
			}
		}
		settings, _ := repoBilling.GetOrgTaxSettings(r.Context(), d.Pool, orgID)
		customerTaxID := ""
		if ent.CustomerTaxIDEnc != "" {
			if plain, err := crypto.DecryptString(ent.CustomerTaxIDEnc); err == nil {
				customerTaxID = plain
			}
		}
		pdfInput := svcBilling.InvoicePDFFromEntitlement(ent, inv, settings, customerTaxID)
		pdfBytes, err := svcBilling.BuildTaxInvoicePDF(pdfInput)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not generate invoice.")
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename="+inv.InvoiceNumber+".pdf")
		_, _ = w.Write(pdfBytes)
	}
}

func parseTaxReportPeriod(period string) (from, to time.Time, err error) {
	if period == "" {
		now := time.Now().UTC()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(0, 1, 0)
		return from, to, nil
	}
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	from = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	to = from.AddDate(0, 1, 0)
	return from, to, nil
}

func itoa(n int) string {
	return strings.TrimSpace(strings.Replace(strings.Replace(
		jsonStringInt(n), "\"", "", -1), " ", "", -1))
}

func itoa64(n int64) string {
	return strings.TrimSpace(strings.Replace(strings.Replace(
		jsonStringInt64(n), "\"", "", -1), " ", "", -1))
}

func jsonStringInt(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func jsonStringInt64(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}