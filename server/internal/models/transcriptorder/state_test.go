package transcriptorder

import (
	"errors"
	"testing"
)

func TestValidateOrderTransition_Matrix(t *testing.T) {
	legal := [][2]OrderStatus{
		{OrderDraft, OrderInReview},
		{OrderDraft, OrderOnHold},
		{OrderDraft, OrderProcessing},
		{OrderInReview, OrderProcessing},
		{OrderInReview, OrderOnHold},
		{OrderInReview, OrderRejected},
		{OrderOnHold, OrderInReview},
		{OrderOnHold, OrderProcessing},
		{OrderProcessing, OrderCompleted},
		{OrderProcessing, OrderOnHold},
	}
	for _, pair := range legal {
		if err := ValidateOrderTransition(pair[0], pair[1]); err != nil {
			t.Fatalf("expected legal %s→%s: %v", pair[0], pair[1], err)
		}
	}
	illegal := [][2]OrderStatus{
		{OrderDraft, OrderCompleted},
		{OrderCompleted, OrderInReview},
		{OrderRejected, OrderProcessing},
		{OrderFailed, OrderInReview},
		{OrderInReview, OrderDraft},
	}
	for _, pair := range illegal {
		err := ValidateOrderTransition(pair[0], pair[1])
		if !errors.Is(err, ErrIllegalTransition) {
			t.Fatalf("expected illegal %s→%s, got %v", pair[0], pair[1], err)
		}
	}
}

func TestResolveSubmitTarget(t *testing.T) {
	if got := ResolveSubmitTarget(GateContext{HasBlockingHold: true, ConsentSatisfied: true, PaymentSatisfied: true}); got != OrderOnHold {
		t.Fatalf("hold: got %s", got)
	}
	if got := ResolveSubmitTarget(GateContext{ConsentSatisfied: false, PaymentSatisfied: true}); got != OrderPendingConsent {
		t.Fatalf("consent: got %s", got)
	}
	if got := ResolveSubmitTarget(GateContext{ConsentSatisfied: true, PaymentSatisfied: false}); got != OrderPendingPayment {
		t.Fatalf("payment: got %s", got)
	}
	if got := ResolveSubmitTarget(GateContext{ConsentSatisfied: true, PaymentSatisfied: true, AutoApproval: true}); got != OrderProcessing {
		t.Fatalf("auto: got %s", got)
	}
	if got := ResolveSubmitTarget(GateContext{ConsentSatisfied: true, PaymentSatisfied: true}); got != OrderInReview {
		t.Fatalf("review: got %s", got)
	}
}

func TestTargetForAction_RejectRequiresReason(t *testing.T) {
	_, err := TargetForAction(OrderInReview, ActionReject, "")
	if !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("want ErrReasonRequired got %v", err)
	}
	to, err := TargetForAction(OrderInReview, ActionReject, "incomplete")
	if err != nil || to != OrderRejected {
		t.Fatalf("got %s %v", to, err)
	}
}

func TestParseOrderStatus_LegacyAliases(t *testing.T) {
	s, err := ParseOrderStatus("submitted")
	if err != nil || s != OrderCompleted {
		t.Fatalf("submitted→completed got %s %v", s, err)
	}
	s, err = ParseOrderStatus("queued")
	if err != nil || s != OrderInReview {
		t.Fatalf("queued→in_review got %s %v", s, err)
	}
}
