package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/lextures/lextures/clients/cli/internal/config"
)

func TestFilterEntitlementsByMonth(t *testing.T) {
	items := []billingEntitlement{
		{ID: "1", ValidFrom: "2026-06-15T00:00:00Z"},
		{ID: "2", ValidFrom: "2026-05-15T00:00:00Z"},
	}
	got := filterEntitlementsByMonth(items, "2026-06")
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("filter = %+v", got)
	}
}

func TestEntitlementsToCSV(t *testing.T) {
	inv := "inv-1"
	data, rows, err := entitlementsToCSV([]billingEntitlement{
		{ID: "e1", EntitlementType: "course", AmountPaidCents: 1000, Currency: "usd", Status: "active", InvoiceID: &inv, ValidFrom: "2026-06-01T00:00:00Z"},
	})
	if err != nil || rows != 2 {
		t.Fatalf("rows=%d err=%v", rows, err)
	}
	if !strings.Contains(string(data), "inv-1") {
		t.Fatalf("csv = %q", string(data))
	}
}

func TestInvoicesExport_RequiresYes(t *testing.T) {
	invoicesExportFlags.yes = false
	defer func() { invoicesExportFlags.yes = false }()
	err := invoicesExportCmd.RunE(invoicesExportCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
}

func TestTaxGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/orgs/org-1/tax/settings" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"orgId": "org-1", "enabled": true, "sellerTaxId": "VAT123",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	taxGetFlags.org = "org-1"
	defer func() { taxGetFlags.org = "" }()

	globalFlags.jsonOut = true
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	taxGetCmd.SetOut(&out)
	if err := taxGetCmd.RunE(taxGetCmd, nil); err != nil {
		t.Fatalf("tax get: %v", err)
	}
	if !strings.Contains(out.String(), "VAT123") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestTaxSet_FromFile(t *testing.T) {
	var saved map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/api/v1/orgs/org-1/tax/settings" {
			_ = json.NewDecoder(r.Body).Decode(&saved)
			_ = json.NewEncoder(w).Encode(saved)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := dir + "/tax.json"
	if err := os.WriteFile(path, []byte(`{"enabled":true,"sellerTaxId":"VAT999"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	taxSetFlags.org = "org-1"
	taxSetFlags.file = path
	defer func() {
		taxSetFlags.org = ""
		taxSetFlags.file = ""
	}()

	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	if err := taxSetCmd.RunE(taxSetCmd, nil); err != nil {
		t.Fatalf("tax set: %v", err)
	}
	if saved["sellerTaxId"] != "VAT999" {
		t.Fatalf("saved = %+v", saved)
	}
}

func TestPaymentsTransactionsList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me/transactions" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"transactions": []any{map[string]any{
					"id": "tx1", "provider": "stripe", "providerTxnId": "pi_1",
					"amountCents": 2500, "currency": "usd", "status": "succeeded",
					"createdAt": "2026-06-01T00:00:00Z",
				}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	paymentsTransactionsListCmd.SetOut(&out)
	if err := paymentsTransactionsListCmd.RunE(paymentsTransactionsListCmd, nil); err != nil {
		t.Fatalf("transactions list: %v", err)
	}
	if !strings.Contains(out.String(), "tx1") {
		t.Fatalf("output = %q", out.String())
	}
}