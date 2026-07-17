package transcriptfees

import "testing"

func TestComputeQuoteAC1(t *testing.T) {
	// AC-1: base $10 + $5/recipient × 2 + rush $3 = $23
	q := ComputeQuote(QuoteInput{
		Schedule: Schedule{
			Currency:        "usd",
			BaseFee:         1000,
			RushFee:         300,
			PerRecipientFee: 500,
		},
		Items: []LineItem{
			{DeliveryMethod: "secure_link_email", Urgency: "rush"},
			{DeliveryMethod: "secure_link_email", Urgency: "standard"},
		},
	})
	if q.Subtotal != 2300 {
		t.Fatalf("subtotal=%d want 2300", q.Subtotal)
	}
	if q.Total != 2300 {
		t.Fatalf("total=%d want 2300", q.Total)
	}
	if !q.RequiresPayment {
		t.Fatal("expected requiresPayment")
	}
}

func TestComputeQuoteMethodSurcharge(t *testing.T) {
	q := ComputeQuote(QuoteInput{
		Schedule: Schedule{
			Currency: "usd",
			BaseFee:  1000,
			MethodSurcharges: map[string]int{
				"postal_mail": 200,
			},
		},
		Items: []LineItem{
			{DeliveryMethod: "postal_mail", Urgency: "standard"},
			{DeliveryMethod: "secure_link_email", Urgency: "standard"},
		},
	})
	if q.Total != 1200 {
		t.Fatalf("total=%d want 1200", q.Total)
	}
}

func TestComputeQuoteFullWaiver(t *testing.T) {
	q := ComputeQuote(QuoteInput{
		Schedule: Schedule{Currency: "usd", BaseFee: 1000, PerRecipientFee: 500},
		Items:    []LineItem{{DeliveryMethod: "secure_link_email", Urgency: "standard"}},
		Waiver:   &WaiverInput{Kind: WaiverFull},
	})
	if q.Total != 0 || q.RequiresPayment {
		t.Fatalf("total=%d requires=%v", q.Total, q.RequiresPayment)
	}
	if q.PaymentStatusIfZero != PaymentWaived {
		t.Fatalf("status=%s want waived", q.PaymentStatusIfZero)
	}
	if q.WaiverAmount != 1500 {
		t.Fatalf("waiver=%d want 1500", q.WaiverAmount)
	}
}

func TestComputeQuotePercentAndAmount(t *testing.T) {
	pct := ComputeQuote(QuoteInput{
		Schedule: Schedule{Currency: "usd", BaseFee: 1000},
		Items:    []LineItem{{DeliveryMethod: "secure_link_email", Urgency: "standard"}},
		Waiver:   &WaiverInput{Kind: WaiverPercent, Value: 50},
	})
	if pct.Total != 500 {
		t.Fatalf("percent total=%d want 500", pct.Total)
	}
	amt := ComputeQuote(QuoteInput{
		Schedule: Schedule{Currency: "usd", BaseFee: 1000},
		Items:    []LineItem{{DeliveryMethod: "secure_link_email", Urgency: "standard"}},
		Waiver:   &WaiverInput{Kind: WaiverAmount, Value: 300},
	})
	if amt.Total != 700 {
		t.Fatalf("amount total=%d want 700", amt.Total)
	}
}

func TestComputeQuoteFreeAllotment(t *testing.T) {
	q := ComputeQuote(QuoteInput{
		Schedule:            Schedule{Currency: "usd", BaseFee: 1000, FreeAllotment: 2},
		Items:               []LineItem{{DeliveryMethod: "secure_link_email", Urgency: "standard"}},
		FreeAllotmentRemain: 1,
		ApplyFreeAllotment:  true,
	})
	if q.Total != 0 || !q.FreeAllotmentApplied {
		t.Fatalf("total=%d freeApplied=%v", q.Total, q.FreeAllotmentApplied)
	}
	if q.PaymentStatusIfZero != PaymentFree {
		t.Fatalf("status=%s want free", q.PaymentStatusIfZero)
	}
}

func TestComputeQuoteZeroSchedule(t *testing.T) {
	q := ComputeQuote(QuoteInput{
		Schedule: Schedule{Currency: "usd"},
		Items:    []LineItem{{DeliveryMethod: "secure_link_email", Urgency: "standard"}},
	})
	if q.Total != 0 || q.RequiresPayment {
		t.Fatalf("expected free zero schedule, got total=%d", q.Total)
	}
}

func TestPaymentStatusGate(t *testing.T) {
	for _, s := range []PaymentStatus{PaymentPaid, PaymentWaived, PaymentFree} {
		if !s.SatisfiesPaymentGate() {
			t.Fatalf("%s should satisfy gate", s)
		}
	}
	for _, s := range []PaymentStatus{PaymentUnpaid, PaymentPending, PaymentRefunded, PaymentPartiallyRefunded} {
		if s.SatisfiesPaymentGate() {
			t.Fatalf("%s should not satisfy gate", s)
		}
	}
}

func TestApplyWaiver(t *testing.T) {
	if got := ApplyWaiver(1000, WaiverInput{Kind: WaiverFull}); got != 1000 {
		t.Fatalf("full=%d", got)
	}
	if got := ApplyWaiver(1000, WaiverInput{Kind: WaiverPercent, Value: 25}); got != 250 {
		t.Fatalf("pct=%d", got)
	}
	if got := ApplyWaiver(1000, WaiverInput{Kind: WaiverAmount, Value: 9999}); got != 1000 {
		t.Fatalf("cap=%d", got)
	}
}
