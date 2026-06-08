package sis

import (
	"context"
	"testing"

	repoSIS "github.com/lextures/lextures/server/internal/repos/sis"
)

func TestAdapterFor_HEVendors(t *testing.T) {
	vendors := []string{
		repoSIS.VendorBanner,
		repoSIS.VendorWorkday,
		repoSIS.VendorColleague,
		repoSIS.VendorJenzabar,
		repoSIS.VendorPeopleSoft,
	}
	for _, v := range vendors {
		a := AdapterFor(v)
		if a == nil {
			t.Fatalf("expected adapter for %q", v)
		}
		if a.Vendor() != v {
			t.Fatalf("vendor mismatch: got %q want %q", a.Vendor(), v)
		}
		summary, errs := a.SyncRoster(context.Background(), ConnectionConfig{Vendor: v, BaseURL: "https://sis.example.edu"})
		if len(errs) != 0 {
			t.Fatalf("stub sync should not error: %v", errs)
		}
		if summary.UsersCreated != 0 || summary.EnrollmentsCreated != 0 {
			t.Fatalf("stub sync should return zero counts: %+v", summary)
		}
		if err := a.TestConnection(context.Background(), ConnectionConfig{}); err != nil {
			t.Fatalf("test connection: %v", err)
		}
	}
}

func TestAdapterFor_K12ReturnsNil(t *testing.T) {
	if AdapterFor(repoSIS.VendorPowerSchool) != nil {
		t.Fatal("expected nil adapter for K-12 vendor")
	}
}

func TestIsHEVendor(t *testing.T) {
	if !IsHEVendor(repoSIS.VendorBanner) {
		t.Fatal("banner should be HE vendor")
	}
	if IsHEVendor(repoSIS.VendorPowerSchool) {
		t.Fatal("powerschool should not be HE vendor")
	}
}
