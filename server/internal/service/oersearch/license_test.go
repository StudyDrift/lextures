package oersearch

import "testing"

func TestMatchesLicenseFilter_CC_BY(t *testing.T) {
	if !MatchesLicenseFilter("CC-BY-4.0", "CC-BY") {
		t.Fatal("CC-BY-4.0 should match CC-BY filter")
	}
	if MatchesLicenseFilter("CC-BY-NC-4.0", "CC-BY") {
		t.Fatal("NC license should not match CC-BY filter")
	}
	if MatchesLicenseFilter("CC-BY-ND-4.0", "CC-BY") {
		t.Fatal("ND license should not match CC-BY filter")
	}
	if MatchesLicenseFilter("CC-BY-SA-4.0", "CC-BY") {
		t.Fatal("SA-only style should not match strict CC-BY filter")
	}
}

func TestAllowsImportCopy(t *testing.T) {
	if !AllowsImportCopy("CC-BY-4.0") {
		t.Fatal("CC-BY should allow import")
	}
	if AllowsImportCopy("CC-BY-NC-4.0") {
		t.Fatal("NC should block import")
	}
}

func TestFilterStubResults_photosynthesis(t *testing.T) {
	all := stubCatalog("oer_commons")
	out := filterStubResults(all, SearchParams{Query: "photosynthesis"})
	if len(out) < 1 {
		t.Fatalf("expected photosynthesis hits, got %d", len(out))
	}
	out = filterStubResults(all, SearchParams{Query: "photosynthesis", License: "CC-BY"})
	for _, r := range out {
		if !MatchesLicenseFilter(r.LicenseSPDX, "CC-BY") {
			t.Fatalf("filtered result has license %s", r.LicenseSPDX)
		}
	}
}
