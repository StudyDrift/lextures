package paymentprovider

import (
	"testing"

	"github.com/google/uuid"
)

func TestCheckoutRequest_UnitAmount_NoCoupon(t *testing.T) {
	t.Parallel()
	req := CheckoutRequest{PriceCents: 4000, ChargedCents: 3000, DiscountCents: 1000}
	if req.UnitAmount() != 4000 {
		t.Fatalf("without first-party coupon, unit amount should be list price, got %d", req.UnitAmount())
	}
}

func TestCheckoutRequest_UnitAmount_WithCoupon(t *testing.T) {
	t.Parallel()
	req := CheckoutRequest{
		PriceCents:          4000,
		ChargedCents:        3000,
		DiscountCents:       1000,
		HasFirstPartyCoupon: true,
	}
	if req.UnitAmount() != 3000 {
		t.Fatalf("got %d want 3000", req.UnitAmount())
	}
}

func TestCheckoutRequest_UnitAmount_ZeroAfterDiscount(t *testing.T) {
	t.Parallel()
	// Free path should not call the provider; if it did, unit amount is 0.
	req := CheckoutRequest{PriceCents: 4000, ChargedCents: 0, HasFirstPartyCoupon: true}
	if req.UnitAmount() != 0 {
		t.Fatalf("got %d", req.UnitAmount())
	}
}

func TestCopyMetadata_CouponFields(t *testing.T) {
	t.Parallel()
	uid := uuid.New()
	meta := map[string]string{
		"checkout_key":          "k1",
		"coupon_id":             uid.String(),
		"coupon_code":           "LAUNCH25",
		"coupon_discount_cents": "1000",
		"list_price_cents":      "4000",
	}
	out := copyMetadata(meta)
	if out["coupon_code"] != "LAUNCH25" {
		t.Fatalf("%v", out)
	}
	// Mutation isolation.
	out["coupon_code"] = "X"
	if meta["coupon_code"] != "LAUNCH25" {
		t.Fatal("copy not isolated")
	}
}
