package currency

import "testing"

func TestIsZeroDecimal(t *testing.T) {
	if !IsZeroDecimal("jpy") {
		t.Fatal("expected jpy to be zero-decimal")
	}
	if IsZeroDecimal("usd") {
		t.Fatal("expected usd to be two-decimal")
	}
}

func TestMinorUnitFactor(t *testing.T) {
	if got := MinorUnitFactor("jpy"); got != 1 {
		t.Fatalf("jpy factor: got %d want 1", got)
	}
	if got := MinorUnitFactor("usd"); got != 100 {
		t.Fatalf("usd factor: got %d want 100", got)
	}
}

func TestMinorUnitsToMajorUnits(t *testing.T) {
	if got := MinorUnitsToMajorUnits(1999, "usd"); got != 19.99 {
		t.Fatalf("usd: got %v want 19.99", got)
	}
	if got := MinorUnitsToMajorUnits(1000, "jpy"); got != 1000 {
		t.Fatalf("jpy: got %v want 1000", got)
	}
}

func TestValidateCatalogPrice(t *testing.T) {
	if err := ValidateCatalogPrice(1999, "usd"); err != nil {
		t.Fatalf("valid usd price: %v", err)
	}
	if err := ValidateCatalogPrice(1000, "jpy"); err != nil {
		t.Fatalf("valid jpy price: %v", err)
	}
	if err := ValidateCatalogPrice(25, "usd"); err == nil {
		t.Fatal("expected sub-minimum usd price to fail")
	}
	if err := ValidateCatalogPrice(100_000, "jpy"); err == nil {
		t.Fatal("expected over-max jpy price to fail")
	}
}

func TestFormatAmount(t *testing.T) {
	if got := FormatAmount(1999, "usd"); got != "USD 19.99" {
		t.Fatalf("usd format: got %q", got)
	}
	if got := FormatAmount(1000, "jpy"); got != "JPY 1000" {
		t.Fatalf("jpy format: got %q", got)
	}
}