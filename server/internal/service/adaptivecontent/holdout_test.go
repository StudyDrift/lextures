package adaptivecontent

import (
	"testing"

	"github.com/google/uuid"
)

func TestIsHoldout_ZeroPercent(t *testing.T) {
	e := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	u := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	if IsHoldout(e, u, 0) {
		t.Fatal("holdout 0 must never assign holdout")
	}
	if IsHoldout(e, u, -1) {
		t.Fatal("negative holdout must never assign holdout")
	}
}

func TestIsHoldout_Stable(t *testing.T) {
	e := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	u := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	a := IsHoldout(e, u, 50)
	b := IsHoldout(e, u, 50)
	if a != b {
		t.Fatal("holdout assignment must be stable across calls")
	}
	// Different enrollment can differ
	e2 := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	_ = IsHoldout(e2, u, 50) // smoke
}

func TestIsHoldout_HundredPercentClampedTo50(t *testing.T) {
	// holdout > 50 is clamped to 50, so roughly half — not all.
	// With a fixed pair we only assert clamp doesn't panic and is deterministic.
	e := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	u := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	a := IsHoldout(e, u, 100)
	b := IsHoldout(e, u, 50)
	if a != b {
		t.Fatalf("holdout 100 should clamp to 50: got %v vs %v", a, b)
	}
}

func TestIsHoldout_ProportionApprox(t *testing.T) {
	// 1000 synthetic enrollments at 20% should land near 200 (±15%).
	unit := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	const n = 1000
	const pct int16 = 20
	holdouts := 0
	for i := 0; i < n; i++ {
		// Deterministic UUIDs from counter.
		enr := uuid.UUID{}
		enr[0] = byte(i >> 24)
		enr[1] = byte(i >> 16)
		enr[2] = byte(i >> 8)
		enr[3] = byte(i)
		enr[6] = 0x40 // version nibble
		enr[8] = 0x80 // variant
		if IsHoldout(enr, unit, pct) {
			holdouts++
		}
	}
	// Allow generous band for hash distribution.
	if holdouts < 120 || holdouts > 280 {
		t.Fatalf("holdout proportion off: got %d/1000 for 20%%", holdouts)
	}
}

func TestHoldoutBucket_Range(t *testing.T) {
	e := uuid.New()
	u := uuid.New()
	b := HoldoutBucket(e, u)
	if b < 0 || b > 99 {
		t.Fatalf("bucket out of range: %d", b)
	}
}

func TestAdaptationReasonLabel(t *testing.T) {
	if got := AdaptationReasonLabel(EmphasisReinforce, nil); got == "" {
		t.Fatal("expected non-empty reason")
	}
	if got := AdaptationReasonLabel("", []string{"scaffolding"}); got == "" {
		t.Fatal("expected axis-based reason")
	}
	if got := AdaptationReasonLabel("", nil); got != "matched to your progress" {
		t.Fatalf("default: %q", got)
	}
}
