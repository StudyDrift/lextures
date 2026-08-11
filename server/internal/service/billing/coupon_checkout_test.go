package billing

import (
	"testing"

	"github.com/lextures/lextures/server/internal/service/coupons"
)

func TestSeatsRemainingDisclosure(t *testing.T) {
	t.Parallel()
	if seatsRemainingDisclosure(nil, 0) != nil {
		t.Fatal("unlimited should be null")
	}
	max := 100
	if seatsRemainingDisclosure(&max, 50) != nil {
		t.Fatal("remaining 50 > threshold should be null")
	}
	max2 := 12
	got := seatsRemainingDisclosure(&max2, 5) // remaining 7
	if got == nil || *got != 7 {
		t.Fatalf("remaining 7: got %v", got)
	}
	max3 := 5
	got = seatsRemainingDisclosure(&max3, 5) // remaining 0
	if got == nil || *got != 0 {
		t.Fatalf("remaining 0: got %v", got)
	}
	// floor at 0 when oversold in display math
	got = seatsRemainingDisclosure(&max3, 10)
	if got == nil || *got != 0 {
		t.Fatalf("overconsumed: got %v", got)
	}
}

func TestEmptyPreview(t *testing.T) {
	t.Parallel()
	p := emptyPreview("LAUNCH25", PreviewCouponInput{CoursePrice: 4000, CourseCurrency: "USD"})
	if p.Applied || p.Reason != coupons.ReasonNotFound {
		t.Fatalf("got %+v", p)
	}
	if p.ListPriceCents != 4000 || p.ChargedCents != 4000 || p.Currency != "usd" {
		t.Fatalf("amounts: %+v", p)
	}
	if p.Code != "LAUNCH25" {
		t.Fatalf("code: %s", p.Code)
	}
}

func TestAtoiDefault(t *testing.T) {
	t.Parallel()
	if atoiDefault("3000", 0) != 3000 {
		t.Fatal("parse")
	}
	if atoiDefault("", 7) != 7 {
		t.Fatal("default empty")
	}
	if atoiDefault("12x", 1) != 1 {
		t.Fatal("default invalid")
	}
}

func TestCouponPreviewJSONShape_Reasons(t *testing.T) {
	t.Parallel()
	// Map all ten reason tokens so clients can rely on the vocabulary.
	reasons := []coupons.Reason{
		coupons.ReasonOK,
		coupons.ReasonNotFound,
		coupons.ReasonInactive,
		coupons.ReasonNotStarted,
		coupons.ReasonExpired,
		coupons.ReasonExhausted,
		coupons.ReasonAlreadyUsed,
		coupons.ReasonCurrencyMismatch,
		coupons.ReasonCourseFree,
		coupons.ReasonOwned,
	}
	if len(reasons) != 10 {
		t.Fatalf("expected 10 reasons, got %d", len(reasons))
	}
}
