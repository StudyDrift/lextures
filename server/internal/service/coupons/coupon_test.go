package coupons

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNormalizeCode(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"launch25", "LAUNCH25"},
		{"  launch25  ", "LAUNCH25"},
		{"launch 25", "LAUNCH25"},
		{"Launch_25", "LAUNCH_25"},
		{"a-b_c", "A-B_C"},
		{"", ""},
		{"  \t\n  ", ""},
	}
	for _, c := range cases {
		if got := NormalizeCode(c.in); got != c.want {
			t.Errorf("NormalizeCode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// Cap large inputs.
	huge := strings.Repeat("a", 20_000)
	got := NormalizeCode(huge)
	if len(got) > 10_000 {
		t.Fatalf("NormalizeCode did not cap input: len=%d", len(got))
	}
}

func TestValidateCode(t *testing.T) {
	ok := []string{"LAUNCH25", "AB12", "A1234567890123456789012345678901"} // 32
	for _, c := range ok {
		if err := ValidateCode(c); err != nil {
			t.Errorf("ValidateCode(%q) unexpected: %v", c, err)
		}
	}
	bad := []string{"ABC", "ab12", " launch25", "LAUNCH 25", "LAUNCH@25", "", "A" + strings.Repeat("B", 32)}
	for _, c := range bad {
		if err := ValidateCode(c); err == nil {
			t.Errorf("ValidateCode(%q) expected error", c)
		}
	}
}

func TestApplyDiscount_Percent(t *testing.T) {
	// AC-1: 25% of 4000¢ = 1000, charged 3000.
	q := ApplyDiscount(4000, "usd", Coupon{Kind: KindPercent, PercentOff: 25})
	if q.DiscountCents != 1000 || q.ChargedCents != 3000 || q.ClampedToFree {
		t.Fatalf("AC-1: got discount=%d charged=%d clamp=%v", q.DiscountCents, q.ChargedCents, q.ClampedToFree)
	}
}

func TestApplyDiscount_HalfUp(t *testing.T) {
	// AC-2: 33% of 999¢ → 329.67 → 330 half-up, charged 669.
	q := ApplyDiscount(999, "usd", Coupon{Kind: KindPercent, PercentOff: 33})
	if q.DiscountCents != 330 || q.ChargedCents != 669 {
		t.Fatalf("AC-2 half-up: got discount=%d charged=%d", q.DiscountCents, q.ChargedCents)
	}
	// JPY: whole yen, same arithmetic on whole units.
	qJPY := ApplyDiscount(999, "jpy", Coupon{Kind: KindPercent, PercentOff: 33})
	if qJPY.DiscountCents != 330 || qJPY.ChargedCents != 669 {
		t.Fatalf("AC-2 jpy: got discount=%d charged=%d", qJPY.DiscountCents, qJPY.ChargedCents)
	}
}

func TestApplyDiscount_FixedClamp(t *testing.T) {
	// AC-3: fixed 5000 on 3000 → discount 3000, charged 0, not clamped-to-free.
	q := ApplyDiscount(3000, "usd", Coupon{Kind: KindFixed, AmountOffCents: 5000, Currency: "usd"})
	if q.DiscountCents != 3000 || q.ChargedCents != 0 || q.ClampedToFree {
		t.Fatalf("AC-3: got discount=%d charged=%d clamp=%v", q.DiscountCents, q.ChargedCents, q.ClampedToFree)
	}
}

func TestApplyDiscount_ProviderFloor(t *testing.T) {
	// AC-4: 99% of 4000 = 40¢ residual < 50¢ floor → free.
	q := ApplyDiscount(4000, "usd", Coupon{Kind: KindPercent, PercentOff: 99})
	if q.ChargedCents != 0 || q.DiscountCents != 4000 || !q.ClampedToFree {
		t.Fatalf("AC-4: got discount=%d charged=%d clamp=%v", q.DiscountCents, q.ChargedCents, q.ClampedToFree)
	}
}

func TestApplyDiscount_HundredPercent(t *testing.T) {
	q := ApplyDiscount(4000, "usd", Coupon{Kind: KindPercent, PercentOff: 100})
	if q.ChargedCents != 0 || q.DiscountCents != 4000 || q.ClampedToFree {
		t.Fatalf("100%%: got discount=%d charged=%d clamp=%v", q.DiscountCents, q.ChargedCents, q.ClampedToFree)
	}
}

func TestApplyDiscount_ZeroAndNegativeList(t *testing.T) {
	q := ApplyDiscount(0, "usd", Coupon{Kind: KindPercent, PercentOff: 50})
	if q.DiscountCents != 0 || q.ChargedCents != 0 {
		t.Fatalf("zero list: %+v", q)
	}
	q = ApplyDiscount(-100, "usd", Coupon{Kind: KindPercent, PercentOff: 50})
	if q.DiscountCents != 0 || q.ChargedCents != 0 {
		t.Fatalf("negative list: %+v", q)
	}
}

func TestApplyDiscount_HalfUpBoundary(t *testing.T) {
	// 50% of 1¢ → 0.5 → 1 half-up.
	q := ApplyDiscount(1, "usd", Coupon{Kind: KindPercent, PercentOff: 50})
	// Residual 0 after half-up discount of 1.
	if q.DiscountCents != 1 || q.ChargedCents != 0 {
		t.Fatalf("0.5 boundary: discount=%d charged=%d", q.DiscountCents, q.ChargedCents)
	}
	// 10% of 5¢ → 0.5 → 1 half-up.
	q = ApplyDiscount(5, "usd", Coupon{Kind: KindPercent, PercentOff: 10})
	// Would charge 4 which is below 50 floor → clamp free.
	if !q.ClampedToFree || q.ChargedCents != 0 {
		t.Fatalf("expected floor clamp after half-up: %+v", q)
	}
}

func TestEvaluate_Reasons(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	base := Coupon{
		ID:                    uuid.New(),
		CourseID:              uuid.New(),
		Code:                  "LAUNCH25",
		Kind:                  KindPercent,
		PercentOff:            25,
		MaxRedemptionsPerUser: 1,
		Status:                StatusActive,
	}
	ec := EvalContext{
		Now:            now,
		CoursePrice:    4000,
		CourseCurrency: "usd",
		ConsumedSeats:  0,
		UserSeats:      0,
		AlreadyOwned:   false,
	}

	// ok
	r, q := Evaluate(&base, ec)
	if r != ReasonOK || q.ChargedCents != 3000 {
		t.Fatalf("ok: reason=%s quote=%+v", r, q)
	}

	// not_found
	r, _ = Evaluate(nil, ec)
	if r != ReasonNotFound {
		t.Fatalf("nil coupon: %s", r)
	}

	// inactive
	c := base
	c.Status = StatusDisabled
	r, _ = Evaluate(&c, ec)
	if r != ReasonInactive {
		t.Fatalf("disabled: %s", r)
	}
	c.Status = StatusArchived
	r, _ = Evaluate(&c, ec)
	if r != ReasonInactive {
		t.Fatalf("archived: %s", r)
	}

	// not_started (AC-8)
	c = base
	start := now.Add(time.Hour)
	c.StartsAt = &start
	r, _ = Evaluate(&c, ec)
	if r != ReasonNotStarted {
		t.Fatalf("not_started: %s", r)
	}
	// inclusive start: exactly at starts_at → ok
	c.StartsAt = &now
	r, _ = Evaluate(&c, ec)
	if r != ReasonOK {
		t.Fatalf("inclusive start: %s", r)
	}

	// expired (AC-8) — exclusive end
	c = base
	end := now
	c.EndsAt = &end
	r, _ = Evaluate(&c, ec)
	if r != ReasonExpired {
		t.Fatalf("exclusive end expired: %s", r)
	}
	// one nanosecond before end → ok
	endLater := now.Add(time.Hour)
	c.EndsAt = &endLater
	r, _ = Evaluate(&c, ec)
	if r != ReasonOK {
		t.Fatalf("before end: %s", r)
	}

	// course_free
	ecFree := ec
	ecFree.CoursePrice = 0
	r, _ = Evaluate(&base, ecFree)
	if r != ReasonCourseFree {
		t.Fatalf("course_free: %s", r)
	}

	// owned
	ecOwned := ec
	ecOwned.AlreadyOwned = true
	r, _ = Evaluate(&base, ecOwned)
	if r != ReasonOwned {
		t.Fatalf("owned: %s", r)
	}

	// currency_mismatch (AC-10)
	c = base
	c.Kind = KindFixed
	c.AmountOffCents = 500
	c.PercentOff = 0
	c.Currency = "usd"
	ecEUR := ec
	ecEUR.CourseCurrency = "eur"
	r, _ = Evaluate(&c, ecEUR)
	if r != ReasonCurrencyMismatch {
		t.Fatalf("currency_mismatch: %s", r)
	}

	// exhausted
	c = base
	max := 5
	c.MaxRedemptions = &max
	ecEx := ec
	ecEx.ConsumedSeats = 5
	r, _ = Evaluate(&c, ecEx)
	if r != ReasonExhausted {
		t.Fatalf("exhausted: %s", r)
	}

	// already_used (AC-9)
	ecUsed := ec
	ecUsed.UserSeats = 1
	r, _ = Evaluate(&base, ecUsed)
	if r != ReasonAlreadyUsed {
		t.Fatalf("already_used: %s", r)
	}
	// another learner (UserSeats=0) still ok
	r, _ = Evaluate(&base, ec)
	if r != ReasonOK {
		t.Fatalf("other learner ok: %s", r)
	}
}

func TestEvaluate_FuzzControlChars(t *testing.T) {
	// NormalizeCode should not panic on control / RTL marks.
	inputs := []string{
		"\x00LAUNCH",
		"\u200fLAUNCH25", // RTL mark
		"\u202eLAUNCH25",
		strings.Repeat("\x01", 100) + "AB12",
	}
	for _, in := range inputs {
		_ = NormalizeCode(in)
	}
}
