package billing

import (
	"context"
	"testing"
	"time"
)

func TestReconcileDecision(t *testing.T) {
	cases := []struct {
		name string
		in   CheckoutSessionState
		want string
	}{
		{"expired", CheckoutSessionState{Status: "expired", PaymentStatus: "unpaid"}, "canceled"},
		{"missing", CheckoutSessionState{Missing: true}, "canceled"},
		{"paid", CheckoutSessionState{Status: "complete", PaymentStatus: "paid"}, "completed"},
		{"no payment required", CheckoutSessionState{Status: "complete", PaymentStatus: "no_payment_required"}, "completed"},
		{"async unpaid", CheckoutSessionState{Status: "complete", PaymentStatus: "unpaid"}, ""},
		{"open", CheckoutSessionState{Status: "open", PaymentStatus: "unpaid"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ReconcileDecision(tc.in); got != tc.want {
				t.Fatalf("ReconcileDecision(%+v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestReconcileStalePendingStripe_NoPool(t *testing.T) {
	if n := ReconcileStalePendingStripe(context.Background(), nil, func(context.Context, string) (CheckoutSessionState, error) {
		return CheckoutSessionState{}, nil
	}, time.Now()); n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}
