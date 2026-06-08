package sis

import "testing"

func TestValidVendor(t *testing.T) {
	cases := []struct {
		vendor string
		ok     bool
		market string
	}{
		{VendorPowerSchool, true, "k12"},
		{VendorBanner, true, "he"},
		{VendorWorkday, true, "he"},
		{VendorPeopleSoft, true, "he"},
		{"canvas", false, "k12"},
	}
	for _, c := range cases {
		if ValidVendor(c.vendor) != c.ok {
			t.Fatalf("ValidVendor(%q) = %v, want %v", c.vendor, !c.ok, c.ok)
		}
		if c.ok && VendorMarket(c.vendor) != c.market {
			t.Fatalf("VendorMarket(%q) = %q, want %q", c.vendor, VendorMarket(c.vendor), c.market)
		}
	}
}
