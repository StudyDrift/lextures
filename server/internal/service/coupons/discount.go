package coupons

import (
	"math"
	"strings"

	"github.com/lextures/lextures/server/internal/currency"
)

// ApplyDiscount returns the list / discount / charged amounts for a coupon
// against a catalog price in minor units (FR-11, FR-12).
//
// Percent discounts round half-up to the currency's minor unit. Fixed discounts
// are clamped to listCents. ChargedCents is never negative. When the residual
// charge is positive but below the provider minimum, ChargedCents is clamped to
// 0 and ClampedToFree is set.
func ApplyDiscount(listCents int, curr string, c Coupon) Quote {
	curr = strings.ToLower(strings.TrimSpace(curr))
	if curr == "" {
		curr = "usd"
	}
	q := Quote{
		ListCents: listCents,
		Currency:  curr,
	}
	if listCents <= 0 {
		q.DiscountCents = 0
		q.ChargedCents = 0
		return q
	}

	discount := 0
	switch c.Kind {
	case KindPercent:
		// Half-up: floor(x + 0.5). Works for positive amounts.
		raw := float64(listCents) * c.PercentOff / 100.0
		discount = int(math.Floor(raw + 0.5))
	case KindFixed:
		discount = c.AmountOffCents
		if discount > listCents {
			discount = listCents
		}
	default:
		// Unknown kind: no discount.
		discount = 0
	}
	if discount < 0 {
		discount = 0
	}
	if discount > listCents {
		discount = listCents
	}

	charged := listCents - discount
	if charged < 0 {
		charged = 0
	}

	// Provider floor clamp (FR-12): residual below minimum → free.
	if charged > 0 {
		minCharge := currency.MinimumChargeCents(curr)
		if charged < minCharge {
			q.ClampedToFree = true
			discount = listCents
			charged = 0
		}
	}

	q.DiscountCents = discount
	q.ChargedCents = charged
	return q
}
