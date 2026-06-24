package billing

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
)

// InvoicePDFInput describes a tax-compliant invoice.
type InvoicePDFInput struct {
	InvoiceNumber   string
	IssuedAt        time.Time
	SellerName      string
	SellerAddress   string
	SellerTaxID     string
	CustomerEmail   string
	CustomerCountry string
	CustomerRegion  string
	CustomerTaxID   string
	SubtotalCents   int
	TaxAmountCents  int
	TotalCents      int
	Currency        string
	TaxType         string
	TaxRate         *float64
	TaxJurisdiction string
	ReverseCharge   bool
	TaxInclusive    bool
	Description     string
	IsCredit        bool
}

// BuildTaxInvoicePDF renders a tax-compliant invoice as PDF bytes.
func BuildTaxInvoicePDF(in InvoicePDFInput) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)

	title := "Tax Invoice"
	if in.IsCredit {
		title = "Credit Note"
	}
	pdf.Cell(0, 10, title)
	pdf.Ln(12)

	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(0, 6, fmt.Sprintf("Invoice number: %s", in.InvoiceNumber))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Date: %s", in.IssuedAt.UTC().Format("2006-01-02")))
	pdf.Ln(10)

	// Seller
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 6, "Seller")
	pdf.Ln(7)
	pdf.SetFont("Helvetica", "", 10)
	if in.SellerName != "" {
		pdf.Cell(0, 5, in.SellerName)
		pdf.Ln(5)
	}
	for _, line := range strings.Split(in.SellerAddress, "\n") {
		if strings.TrimSpace(line) != "" {
			pdf.Cell(0, 5, line)
			pdf.Ln(5)
		}
	}
	if in.SellerTaxID != "" {
		pdf.Cell(0, 5, "Tax ID: "+in.SellerTaxID)
		pdf.Ln(5)
	}
	pdf.Ln(6)

	// Customer
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 6, "Customer")
	pdf.Ln(7)
	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(0, 5, in.CustomerEmail)
	pdf.Ln(5)
	if in.CustomerCountry != "" {
		loc := in.CustomerCountry
		if in.CustomerRegion != "" {
			loc += ", " + in.CustomerRegion
		}
		pdf.Cell(0, 5, loc)
		pdf.Ln(5)
	}
	if in.CustomerTaxID != "" {
		pdf.Cell(0, 5, "Tax ID: "+in.CustomerTaxID)
		pdf.Ln(5)
	}
	pdf.Ln(8)

	// Line items
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(120, 7, "Description")
	pdf.Cell(40, 7, "Amount")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(120, 6, in.Description)
	pdf.Cell(40, 6, formatCurrency(in.SubtotalCents, in.Currency))
	pdf.Ln(8)

	taxLabel := taxLineLabel(in.TaxType, in.ReverseCharge)
	if in.TaxRate != nil && !in.ReverseCharge {
		taxLabel = fmt.Sprintf("%s (%.1f%%)", taxLabel, *in.TaxRate)
	}
	pdf.Cell(120, 6, taxLabel)
	if in.ReverseCharge {
		pdf.Cell(40, 6, "Reverse charge")
	} else {
		pdf.Cell(40, 6, formatCurrency(in.TaxAmountCents, in.Currency))
	}
	pdf.Ln(8)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(120, 7, "Total")
	pdf.Cell(40, 7, formatCurrency(in.TotalCents, in.Currency))
	pdf.Ln(10)

	if in.ReverseCharge {
		pdf.SetFont("Helvetica", "I", 9)
		pdf.MultiCell(0, 5, "VAT reverse charge: customer is responsible for VAT reporting.", "", "L", false)
	}
	if in.TaxInclusive {
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "I", 9)
		pdf.MultiCell(0, 5, "Prices shown are tax-inclusive.", "", "L", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// InvoicePDFFromEntitlement builds invoice PDF input from stored entitlement data.
func InvoicePDFFromEntitlement(ent *repoBilling.EntitlementWithTax, inv *repoBilling.TaxInvoice, settings *repoBilling.OrgTaxSettings, customerTaxID string) InvoicePDFInput {
	subtotal := ent.SubtotalCents
	if subtotal == 0 {
		subtotal = ent.AmountPaidCents - ent.TaxAmountCents
	}
	desc := ent.EntitlementType
	if ent.CourseID != nil {
		desc = "Course purchase"
	}
	sellerName := ""
	sellerAddr := ""
	sellerTaxID := ""
	if settings != nil {
		sellerName = settings.SellerName
		sellerAddr = settings.SellerAddress
		sellerTaxID = settings.SellerTaxID
	}
	issuedAt := ent.CreatedAt
	invNum := ""
	isCredit := false
	if inv != nil {
		invNum = inv.InvoiceNumber
		issuedAt = inv.IssuedAt
		isCredit = inv.CreditedBy != nil
	}
	return InvoicePDFInput{
		InvoiceNumber:   invNum,
		IssuedAt:        issuedAt,
		SellerName:      sellerName,
		SellerAddress:   sellerAddr,
		SellerTaxID:     sellerTaxID,
		CustomerEmail:   ent.UserEmail,
		CustomerCountry: ent.CustomerCountry,
		CustomerRegion:  ent.CustomerRegion,
		CustomerTaxID:   customerTaxID,
		SubtotalCents:   subtotal,
		TaxAmountCents:  ent.TaxAmountCents,
		TotalCents:      ent.AmountPaidCents,
		Currency:        ent.Currency,
		TaxType:         ent.TaxType,
		TaxRate:         ent.TaxRate,
		TaxJurisdiction: ent.TaxJurisdiction,
		ReverseCharge:   ent.ReverseCharge,
		TaxInclusive:    ent.TaxInclusive,
		Description:     desc,
		IsCredit:        isCredit,
	}
}

func formatCurrency(cents int, currency string) string {
	sym := strings.ToUpper(currency)
	return fmt.Sprintf("%s %.2f", sym, float64(cents)/100)
}