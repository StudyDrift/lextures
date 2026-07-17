// Package transcriptfees provides pure fee/quote math for transcript orders (T05).
package transcriptfees

import (
	"fmt"
	"strings"
)

// PaymentStatus is the order payment gate state.
type PaymentStatus string

const (
	PaymentUnpaid            PaymentStatus = "unpaid"
	PaymentPending           PaymentStatus = "pending"
	PaymentPaid              PaymentStatus = "paid"
	PaymentWaived            PaymentStatus = "waived"
	PaymentRefunded          PaymentStatus = "refunded"
	PaymentPartiallyRefunded PaymentStatus = "partially_refunded"
	PaymentFree              PaymentStatus = "free"
)

// SatisfiesPaymentGate reports whether the status allows fulfillment.
func (s PaymentStatus) SatisfiesPaymentGate() bool {
	switch s {
	case PaymentPaid, PaymentWaived, PaymentFree:
		return true
	default:
		return false
	}
}

// ParsePaymentStatus validates a payment status string.
func ParsePaymentStatus(raw string) (PaymentStatus, error) {
	s := PaymentStatus(strings.ToLower(strings.TrimSpace(raw)))
	switch s {
	case PaymentUnpaid, PaymentPending, PaymentPaid, PaymentWaived,
		PaymentRefunded, PaymentPartiallyRefunded, PaymentFree:
		return s, nil
	default:
		return "", fmt.Errorf("invalid payment status %q", raw)
	}
}

// WaiverKind is how a waiver reduces the total.
type WaiverKind string

const (
	WaiverFull    WaiverKind = "full"
	WaiverPercent WaiverKind = "percent"
	WaiverAmount  WaiverKind = "amount"
	WaiverAdmin   WaiverKind = "admin"
)

// AllotmentPeriod scopes free allotment counting.
type AllotmentPeriod string

const (
	AllotmentLifetime AllotmentPeriod = "lifetime"
	AllotmentYear     AllotmentPeriod = "year"
	AllotmentTerm     AllotmentPeriod = "term"
)

// Schedule is an org fee schedule (minor units).
type Schedule struct {
	Currency         string
	BaseFee          int
	RushFee          int
	PerRecipientFee  int
	MethodSurcharges map[string]int
	FreeAllotment    int
	AllotmentPeriod  AllotmentPeriod
}

// LineItem is one order item contributing to the quote.
type LineItem struct {
	DeliveryMethod string
	Urgency        string // standard | rush
}

// WaiverInput is an optional discount applied to the subtotal.
type WaiverInput struct {
	Kind  WaiverKind
	Value int // percent 0-100 or minor-unit amount; ignored for full/admin
}

// QuoteInput feeds ComputeQuote.
type QuoteInput struct {
	Schedule            Schedule
	Items               []LineItem
	Waiver              *WaiverInput
	FreeAllotmentRemain int  // remaining free official transcripts in period
	ApplyFreeAllotment  bool // when true and remain > 0, zero the order via allotment
}

// QuoteLine is one itemized fee line.
type QuoteLine struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Amount      int    `json:"amount"` // minor units; negative for discounts
	Quantity    int    `json:"quantity,omitempty"`
}

// Quote is an itemized order total.
type Quote struct {
	Currency              string      `json:"currency"`
	Lines                 []QuoteLine `json:"lines"`
	Subtotal              int         `json:"subtotal"`
	WaiverAmount          int         `json:"waiverAmount"`
	FreeAllotmentApplied  bool        `json:"freeAllotmentApplied"`
	Total                 int         `json:"total"`
	RequiresPayment       bool        `json:"requiresPayment"`
	PaymentStatusIfZero   PaymentStatus `json:"paymentStatusIfZero,omitempty"`
}

// ComputeQuote builds an itemized total (pure, no I/O).
//
// Rules (AC-1):
//   - base_fee once per order
//   - per_recipient_fee × item count
//   - rush_fee once if any item has urgency=rush
//   - method surcharge per item from schedule map
//   - free allotment zeros the total when ApplyFreeAllotment && remain > 0
//   - waiver reduces remaining subtotal (full/admin → 0; percent; amount capped)
func ComputeQuote(in QuoteInput) Quote {
	cur := strings.ToLower(strings.TrimSpace(in.Schedule.Currency))
	if cur == "" {
		cur = "usd"
	}
	q := Quote{Currency: cur, Lines: make([]QuoteLine, 0, 8)}

	n := len(in.Items)
	if n == 0 {
		return q
	}

	base := max0(in.Schedule.BaseFee)
	if base > 0 {
		q.Lines = append(q.Lines, QuoteLine{
			Code: "base", Description: "Base transcript fee", Amount: base, Quantity: 1,
		})
	}

	perRec := max0(in.Schedule.PerRecipientFee)
	if perRec > 0 {
		q.Lines = append(q.Lines, QuoteLine{
			Code: "per_recipient", Description: "Per-recipient fee", Amount: perRec * n, Quantity: n,
		})
	}

	hasRush := false
	for _, it := range in.Items {
		if strings.EqualFold(strings.TrimSpace(it.Urgency), "rush") {
			hasRush = true
			break
		}
	}
	rush := max0(in.Schedule.RushFee)
	if hasRush && rush > 0 {
		q.Lines = append(q.Lines, QuoteLine{
			Code: "rush", Description: "Rush surcharge", Amount: rush, Quantity: 1,
		})
	}

	methodTotal := 0
	methodCounts := map[string]int{}
	for _, it := range in.Items {
		m := strings.TrimSpace(it.DeliveryMethod)
		if m == "" || in.Schedule.MethodSurcharges == nil {
			continue
		}
		if fee, ok := in.Schedule.MethodSurcharges[m]; ok && fee > 0 {
			methodTotal += fee
			methodCounts[m]++
		}
	}
	for method, qty := range methodCounts {
		fee := in.Schedule.MethodSurcharges[method]
		q.Lines = append(q.Lines, QuoteLine{
			Code:        "method:" + method,
			Description: "Delivery surcharge (" + method + ")",
			Amount:      fee * qty,
			Quantity:    qty,
		})
	}
	_ = methodTotal

	subtotal := 0
	for _, line := range q.Lines {
		subtotal += line.Amount
	}
	q.Subtotal = subtotal

	if in.ApplyFreeAllotment && in.FreeAllotmentRemain > 0 && subtotal > 0 {
		q.FreeAllotmentApplied = true
		q.WaiverAmount = subtotal
		q.Lines = append(q.Lines, QuoteLine{
			Code: "free_allotment", Description: "Free allotment", Amount: -subtotal, Quantity: 1,
		})
		q.Total = 0
		q.RequiresPayment = false
		q.PaymentStatusIfZero = PaymentFree
		return q
	}

	waiverAmt := 0
	if in.Waiver != nil && subtotal > 0 {
		switch in.Waiver.Kind {
		case WaiverFull, WaiverAdmin:
			waiverAmt = subtotal
		case WaiverPercent:
			p := in.Waiver.Value
			if p < 0 {
				p = 0
			}
			if p > 100 {
				p = 100
			}
			waiverAmt = (subtotal * p) / 100
		case WaiverAmount:
			waiverAmt = max0(in.Waiver.Value)
			if waiverAmt > subtotal {
				waiverAmt = subtotal
			}
		}
	}
	if waiverAmt > 0 {
		q.WaiverAmount = waiverAmt
		q.Lines = append(q.Lines, QuoteLine{
			Code: "waiver", Description: "Fee waiver", Amount: -waiverAmt, Quantity: 1,
		})
	}

	total := subtotal - waiverAmt
	if total < 0 {
		total = 0
	}
	q.Total = total
	q.RequiresPayment = total > 0
	if total == 0 {
		if waiverAmt > 0 {
			q.PaymentStatusIfZero = PaymentWaived
		} else {
			q.PaymentStatusIfZero = PaymentFree
		}
	}
	return q
}

// ApplyWaiver computes the waived amount against a subtotal.
func ApplyWaiver(subtotal int, w WaiverInput) int {
	subtotal = max0(subtotal)
	switch w.Kind {
	case WaiverFull, WaiverAdmin:
		return subtotal
	case WaiverPercent:
		p := w.Value
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		return (subtotal * p) / 100
	case WaiverAmount:
		amt := max0(w.Value)
		if amt > subtotal {
			return subtotal
		}
		return amt
	default:
		return 0
	}
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
