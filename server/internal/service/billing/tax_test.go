package billing

import (
	"testing"

	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
)

func TestMapStripeTaxType(t *testing.T) {
	cases := []struct {
		jurisdiction string
		want         string
	}{
		{"GB", repoBilling.TaxTypeVAT},
		{"DE", repoBilling.TaxTypeVAT},
		{"AU", repoBilling.TaxTypeGST},
		{"CA", repoBilling.TaxTypeGST},
		{"US", repoBilling.TaxTypeSalesTax},
		{"", repoBilling.TaxTypeNone},
	}
	for _, tc := range cases {
		got := mapStripeTaxType(tc.jurisdiction)
		if got != tc.want {
			t.Errorf("mapStripeTaxType(%q) = %q, want %q", tc.jurisdiction, got, tc.want)
		}
	}
}

func TestInferTaxIDType(t *testing.T) {
	if got := inferTaxIDType("", "GB"); got != "gb_vat" {
		t.Fatalf("inferTaxIDType GB: got %q", got)
	}
	if got := inferTaxIDType("eu_vat", "DE"); got != "eu_vat" {
		t.Fatalf("explicit type: got %q", got)
	}
}

func TestTaxLineLabel(t *testing.T) {
	if taxLineLabel(repoBilling.TaxTypeVAT, true) != "VAT (reverse charge)" {
		t.Fatal("reverse charge label")
	}
	if taxLineLabel(repoBilling.TaxTypeGST, false) != "GST" {
		t.Fatal("gst label")
	}
}

func TestBuildTaxInvoicePDF(t *testing.T) {
	rate := 20.0
	pdf, err := BuildTaxInvoicePDF(InvoicePDFInput{
		InvoiceNumber:  "INV-TEST-001",
		SellerName:     "Lextures Inc",
		SellerAddress:  "123 Main St\nLondon",
		SellerTaxID:    "GB123456789",
		CustomerEmail:  "buyer@example.com",
		CustomerCountry: "GB",
		SubtotalCents:  10000,
		TaxAmountCents: 2000,
		TotalCents:     12000,
		Currency:       "gbp",
		TaxType:        repoBilling.TaxTypeVAT,
		TaxRate:        &rate,
		Description:    "Course purchase",
	})
	if err != nil {
		t.Fatalf("BuildTaxInvoicePDF: %v", err)
	}
	if len(pdf) < 100 {
		t.Fatalf("PDF too small: %d bytes", len(pdf))
	}
	if pdf[0] != '%' {
		t.Fatal("expected PDF header")
	}
}

func TestReverseChargeMessage(t *testing.T) {
	if reverseChargeMessage(true) == "" {
		t.Fatal("expected message for reverse charge")
	}
	if reverseChargeMessage(false) == "" {
		t.Fatal("expected message for valid ID")
	}
}