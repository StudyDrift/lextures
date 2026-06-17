package billing

import (
	"testing"

	"github.com/google/uuid"
)

func TestComputeCreatorShare(t *testing.T) {
	tests := []struct {
		amount    int
		feePct    float64
		wantCents int
	}{
		{4900, 0.30, 3430},
		{1000, 0.30, 700},
		{0, 0.30, 0},
		{100, 0.0, 100},
	}
	for _, tc := range tests {
		got := ComputeCreatorShare(tc.amount, tc.feePct)
		if got != tc.wantCents {
			t.Errorf("ComputeCreatorShare(%d, %v) = %d, want %d", tc.amount, tc.feePct, got, tc.wantCents)
		}
	}
}

func TestComputeAffiliateCommission(t *testing.T) {
	got := ComputeAffiliateCommission(4900, 0.10)
	if got != 490 {
		t.Fatalf("want 490 got %d", got)
	}
}

func TestIsSelfReferral(t *testing.T) {
	buyer := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	affiliate := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	creator := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	if !IsSelfReferral(buyer, buyer, creator) {
		t.Fatal("buyer == affiliate should be self-referral")
	}
	if !IsSelfReferral(creator, creator, creator) {
		t.Fatal("creator buying via own link should be self-referral")
	}
	if IsSelfReferral(buyer, affiliate, creator) {
		t.Fatal("unrelated buyer should not be self-referral")
	}
}
