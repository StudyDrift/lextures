package billing

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/lextures/lextures/server/internal/currency"
	"github.com/lextures/lextures/server/internal/models/transcriptfees"
)

// TranscriptReceiptInput is the data for a transcript payment/refund receipt.
type TranscriptReceiptInput struct {
	OrderID        string
	IssuedAt       time.Time
	StudentEmail   string
	Currency       string
	PaymentStatus  string
	PaymentRef     string
	AmountPaid     int
	AmountRefunded int
	Lines          []transcriptfees.QuoteLine
	IsRefund       bool
}

// TranscriptReceiptJSON is the JSON receipt payload (AC-6).
type TranscriptReceiptJSON struct {
	OrderID        string                      `json:"orderId"`
	IssuedAt       string                      `json:"issuedAt"`
	StudentEmail   string                      `json:"studentEmail,omitempty"`
	Currency       string                      `json:"currency"`
	PaymentStatus  string                      `json:"paymentStatus"`
	PaymentRef     string                      `json:"paymentRef,omitempty"`
	AmountPaid     int                         `json:"amountPaid"`
	AmountPaidFmt  string                      `json:"amountPaidFormatted"`
	AmountRefunded int                         `json:"amountRefunded"`
	Lines          []transcriptfees.QuoteLine  `json:"lines"`
	IsRefund       bool                        `json:"isRefund"`
}

// BuildTranscriptReceiptJSON builds a JSON receipt view model.
func BuildTranscriptReceiptJSON(in TranscriptReceiptInput) TranscriptReceiptJSON {
	cur := in.Currency
	if cur == "" {
		cur = "usd"
	}
	return TranscriptReceiptJSON{
		OrderID:        in.OrderID,
		IssuedAt:       in.IssuedAt.UTC().Format(time.RFC3339),
		StudentEmail:   in.StudentEmail,
		Currency:       cur,
		PaymentStatus:  in.PaymentStatus,
		PaymentRef:     in.PaymentRef,
		AmountPaid:     in.AmountPaid,
		AmountPaidFmt:  currency.FormatAmount(in.AmountPaid, cur),
		AmountRefunded: in.AmountRefunded,
		Lines:          in.Lines,
		IsRefund:       in.IsRefund,
	}
}

// BuildTranscriptReceiptPDF renders a simple itemized receipt PDF.
func BuildTranscriptReceiptPDF(in TranscriptReceiptInput) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	title := "Transcript Payment Receipt"
	if in.IsRefund {
		title = "Transcript Refund Receipt"
	}
	pdf.Cell(0, 10, title)
	pdf.Ln(12)
	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(0, 6, fmt.Sprintf("Order: %s", in.OrderID))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Date: %s", in.IssuedAt.UTC().Format("2006-01-02")))
	pdf.Ln(6)
	if in.StudentEmail != "" {
		pdf.Cell(0, 6, fmt.Sprintf("Student: %s", in.StudentEmail))
		pdf.Ln(6)
	}
	pdf.Cell(0, 6, fmt.Sprintf("Payment status: %s", in.PaymentStatus))
	pdf.Ln(6)
	if in.PaymentRef != "" {
		pdf.Cell(0, 6, fmt.Sprintf("Payment ref: %s", in.PaymentRef))
		pdf.Ln(6)
	}
	pdf.Ln(4)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 8, "Itemized charges")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 10)
	cur := in.Currency
	if cur == "" {
		cur = "usd"
	}
	for _, line := range in.Lines {
		pdf.Cell(0, 6, fmt.Sprintf("%s: %s", line.Description, currency.FormatAmount(line.Amount, cur)))
		pdf.Ln(6)
	}
	pdf.Ln(4)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 8, fmt.Sprintf("Total paid: %s", currency.FormatAmount(in.AmountPaid, cur)))
	pdf.Ln(8)
	if in.AmountRefunded > 0 {
		pdf.Cell(0, 8, fmt.Sprintf("Refunded: %s", currency.FormatAmount(in.AmountRefunded, cur)))
		pdf.Ln(8)
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
