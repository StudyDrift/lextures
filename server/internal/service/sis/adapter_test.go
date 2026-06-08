package sis

import (
	"context"
	"testing"

	"github.com/google/uuid"

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
		sections, errs, err := SyncCatalog(context.Background(), a, ConnectionConfig{Vendor: v, BaseURL: "https://sis.example.edu"}, uuid.Nil, "Spring 2027")
		if err != nil {
			t.Fatalf("catalog sync: %v", err)
		}
		if len(errs) != 0 {
			t.Fatalf("catalog sync errors: %v", errs)
		}
		if len(sections) == 0 {
			t.Fatal("expected stub catalog sections")
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
