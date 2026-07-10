// Package currency provides ISO 4217 minor-unit helpers for Stripe amounts.
package currency

import (
	"fmt"
	"math"
	"strings"
)

// zeroDecimalCurrencies are ISO 4217 codes where Stripe's smallest unit is the major unit.
var zeroDecimalCurrencies = map[string]struct{}{
	"jpy": {},
}

// IsZeroDecimal reports whether the currency has no fractional minor unit.
func IsZeroDecimal(code string) bool {
	_, ok := zeroDecimalCurrencies[strings.ToLower(strings.TrimSpace(code))]
	return ok
}

// MinorUnitFactor returns the multiplier from major units to Stripe smallest units.
func MinorUnitFactor(code string) int {
	if IsZeroDecimal(code) {
		return 1
	}
	return 100
}

// MinorUnitsToMajorUnits converts Stripe smallest units to major units for display.
func MinorUnitsToMajorUnits(minor int, code string) float64 {
	factor := MinorUnitFactor(code)
	if factor == 0 {
		return 0
	}
	return float64(minor) / float64(factor)
}

// StripeMinimumMinorUnits is the minimum non-zero charge Stripe accepts per currency.
func StripeMinimumMinorUnits(code string) int {
	// JPY minimum is ¥50; USD/EUR-style currencies use 50 minor units ($0.50).
	return 50
}

// MaxCatalogMinorUnits caps marketplace course fees in Stripe smallest units.
func MaxCatalogMinorUnits(code string) int {
	if IsZeroDecimal(code) {
		return 99_999
	}
	return 9_999_999
}

// ValidateCatalogPrice checks stored price_cents against currency rules.
func ValidateCatalogPrice(minor int, code string) error {
	if minor < 0 {
		return fmt.Errorf("price cannot be negative")
	}
	if minor > MaxCatalogMinorUnits(code) {
		return fmt.Errorf("price exceeds the maximum allowed amount")
	}
	if minor > 0 && minor < StripeMinimumMinorUnits(code) {
		return fmt.Errorf("paid courses must be at least $0.50 (or equivalent)")
	}
	return nil
}

// FormatAmount formats Stripe smallest units for receipts and invoices.
func FormatAmount(minor int, code string) string {
	sym := strings.ToUpper(strings.TrimSpace(code))
	major := MinorUnitsToMajorUnits(minor, code)
	if IsZeroDecimal(code) {
		return fmt.Sprintf("%s %d", sym, int(math.Round(major)))
	}
	return fmt.Sprintf("%s %.2f", sym, major)
}