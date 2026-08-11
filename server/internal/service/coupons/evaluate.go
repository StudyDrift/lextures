package coupons

import "strings"

// Evaluate returns eligibility for a coupon against live course/user context.
// Window semantics: inclusive starts_at, exclusive ends_at, both compared in UTC
// (caller should pass Now in UTC). Takes no database handle (FR-13).
func Evaluate(c *Coupon, ec EvalContext) (Reason, Quote) {
	empty := Quote{
		ListCents: ec.CoursePrice,
		Currency:  strings.ToLower(strings.TrimSpace(ec.CourseCurrency)),
	}
	if empty.Currency == "" {
		empty.Currency = "usd"
	}

	if c == nil {
		return ReasonNotFound, empty
	}
	if c.Status != StatusActive {
		return ReasonInactive, empty
	}

	now := ec.Now
	if c.StartsAt != nil && now.Before(c.StartsAt.UTC()) {
		return ReasonNotStarted, empty
	}
	// Exclusive end: now >= ends_at → expired.
	if c.EndsAt != nil && !now.Before(c.EndsAt.UTC()) {
		return ReasonExpired, empty
	}

	if ec.CoursePrice <= 0 {
		return ReasonCourseFree, empty
	}
	if ec.AlreadyOwned {
		return ReasonOwned, empty
	}

	if c.Kind == KindFixed {
		couponCurr := strings.ToLower(strings.TrimSpace(c.Currency))
		courseCurr := strings.ToLower(strings.TrimSpace(ec.CourseCurrency))
		if couponCurr != "" && courseCurr != "" && couponCurr != courseCurr {
			return ReasonCurrencyMismatch, empty
		}
	}

	if c.MaxRedemptions != nil && ec.ConsumedSeats >= *c.MaxRedemptions {
		return ReasonExhausted, empty
	}

	perUser := c.MaxRedemptionsPerUser
	if perUser <= 0 {
		perUser = 1
	}
	if ec.UserSeats >= perUser {
		return ReasonAlreadyUsed, empty
	}

	q := ApplyDiscount(ec.CoursePrice, ec.CourseCurrency, *c)
	return ReasonOK, q
}
