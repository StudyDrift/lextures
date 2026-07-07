package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const financialExportWarning = `WARNING: Financial exports may contain PII and payment records.
Re-run with --yes to confirm you are authorized to export this data.`

type billingEntitlement struct {
	ID              string  `json:"id"`
	EntitlementType string  `json:"entitlementType"`
	CourseID        *string `json:"courseId,omitempty"`
	AmountPaidCents int     `json:"amountPaidCents"`
	Currency        string  `json:"currency"`
	Status          string  `json:"status"`
	ValidFrom       string  `json:"validFrom"`
	ValidUntil      *string `json:"validUntil,omitempty"`
	InvoiceID       *string `json:"invoiceId,omitempty"`
}

type paymentTransaction struct {
	ID            string  `json:"id"`
	CourseID      *string `json:"courseId,omitempty"`
	Provider      string  `json:"provider"`
	ProviderTxnID string  `json:"providerTxnId"`
	AmountCents   int     `json:"amountCents"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"createdAt"`
}

func fetchMyEntitlements(c *client.Client) ([]billingEntitlement, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/entitlements", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Entitlements []billingEntitlement `json:"entitlements"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Entitlements, body, nil
}

func fetchMyTransactions(c *client.Client) ([]paymentTransaction, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/transactions", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Transactions []paymentTransaction `json:"transactions"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Transactions, body, nil
}

func fetchInvoicePDF(c *client.Client, invoiceID string) ([]byte, error) {
	path := "/api/v1/invoices/" + url.PathEscape(invoiceID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func fetchOrgTaxSettings(c *client.Client, orgID string) ([]byte, error) {
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/tax/settings"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func putOrgTaxSettings(c *client.Client, orgID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/tax/settings"
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func fetchOrgTaxReport(c *client.Client, orgID, period, jurisdiction, format string) ([]byte, string, error) {
	q := url.Values{}
	if period != "" {
		q.Set("period", period)
	}
	if jurisdiction != "" {
		q.Set("jurisdiction", jurisdiction)
	}
	if format != "" {
		q.Set("format", format)
	}
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/tax/report"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return body, "", apiErrorBody(resp.StatusCode, body)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func fetchAffiliateCodes(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/creator/affiliate-codes", nil)
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func fetchCreatorEarningsLedger(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/creator/earnings/ledger", nil)
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func createCheckoutLink(c *client.Client, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/billing/checkout", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func filterEntitlementsByMonth(items []billingEntitlement, month string) []billingEntitlement {
	month = strings.TrimSpace(month)
	if month == "" {
		return items
	}
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return items
	}
	out := make([]billingEntitlement, 0, len(items))
	for _, e := range items {
		vf, err := time.Parse(time.RFC3339, e.ValidFrom)
		if err != nil {
			continue
		}
		if vf.Year() == t.Year() && vf.Month() == t.Month() {
			out = append(out, e)
		}
	}
	return out
}

func entitlementsToCSV(items []billingEntitlement) ([]byte, int, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "type", "courseId", "amountCents", "currency", "status", "invoiceId", "validFrom"})
	rows := 1
	for _, e := range items {
		course := ""
		if e.CourseID != nil {
			course = *e.CourseID
		}
		inv := ""
		if e.InvoiceID != nil {
			inv = *e.InvoiceID
		}
		_ = w.Write([]string{
			e.ID, e.EntitlementType, course,
			fmt.Sprintf("%d", e.AmountPaidCents), e.Currency, e.Status, inv, e.ValidFrom,
		})
		rows++
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), rows, nil
}

func transactionsToCSV(items []paymentTransaction) ([]byte, int, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "provider", "providerTxnId", "amountCents", "currency", "status", "createdAt"})
	rows := 1
	for _, tx := range items {
		_ = w.Write([]string{
			tx.ID, tx.Provider, tx.ProviderTxnID,
			fmt.Sprintf("%d", tx.AmountCents), tx.Currency, tx.Status, tx.CreatedAt,
		})
		rows++
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), rows, nil
}