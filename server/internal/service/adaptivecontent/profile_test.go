package adaptivecontent

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestComputeProfile_Compress(t *testing.T) {
	c1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	c2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	r := ComputeProfile(ProfileInput{
		UnitID:     uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ConceptIDs: []uuid.UUID{c1, c2},
		ConceptMastery: map[uuid.UUID]float64{
			c1: 0.95,
			c2: 0.9,
		},
		AxisSet: []string{"emphasis"},
	})
	if r.IsNeutral {
		t.Fatal("expected non-neutral")
	}
	if r.EmphasisMode != EmphasisCompress {
		t.Fatalf("emphasis: got %q want compress", r.EmphasisMode)
	}
	if r.ProfileSignature == NeutralSignature || r.ProfileSignature == "" {
		t.Fatalf("signature: %q", r.ProfileSignature)
	}
	if r.TargetBloom != "analyze" {
		t.Fatalf("bloom: %q", r.TargetBloom)
	}
}

func TestComputeProfile_Remediate(t *testing.T) {
	c1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	mis := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	r := ComputeProfile(ProfileInput{
		UnitID:     uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ConceptIDs: []uuid.UUID{c1},
		// High mastery would compress without misconception — misconception wins.
		ConceptMastery: map[uuid.UUID]float64{
			c1: 0.95,
		},
		MisconceptionIDs: []uuid.UUID{mis},
		AxisSet:          []string{"misconception"},
	})
	if r.EmphasisMode != EmphasisRemediate {
		t.Fatalf("emphasis: got %q want remediate", r.EmphasisMode)
	}
	if len(r.Payload.Misconceptions) != 1 || r.Payload.Misconceptions[0] != mis.String() {
		t.Fatalf("misconceptions: %+v", r.Payload.Misconceptions)
	}
}

func TestComputeProfile_Introduce_NoPrior(t *testing.T) {
	c1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	r := ComputeProfile(ProfileInput{
		UnitID:         uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ConceptIDs:     []uuid.UUID{c1},
		ConceptMastery: map[uuid.UUID]float64{}, // no prior
		AxisSet:        []string{"emphasis"},
	})
	if r.EmphasisMode != EmphasisIntroduce {
		t.Fatalf("emphasis: got %q want introduce", r.EmphasisMode)
	}
	if r.Payload.PriorRecord {
		t.Fatal("expected PriorRecord false")
	}
	if r.Payload.MeanGap != 1.0 {
		t.Fatalf("meanGap: %v", r.Payload.MeanGap)
	}
}

func TestComputeProfile_Introduce_HighMeanGap(t *testing.T) {
	c1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	c2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	r := ComputeProfile(ProfileInput{
		UnitID:     uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ConceptIDs: []uuid.UUID{c1, c2},
		ConceptMastery: map[uuid.UUID]float64{
			c1: 0.1,
			c2: 0.2, // gaps 0.9, 0.8 → mean 0.85 ≥ 0.6
		},
	})
	if r.EmphasisMode != EmphasisIntroduce {
		t.Fatalf("emphasis: got %q want introduce", r.EmphasisMode)
	}
}

func TestComputeProfile_Reinforce(t *testing.T) {
	c1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	c2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	r := ComputeProfile(ProfileInput{
		UnitID:     uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ConceptIDs: []uuid.UUID{c1, c2},
		ConceptMastery: map[uuid.UUID]float64{
			c1: 0.6, // gap 0.4
			c2: 0.5, // gap 0.5 → mean 0.45 ∈ (0.2, 0.6)
		},
	})
	if r.EmphasisMode != EmphasisReinforce {
		t.Fatalf("emphasis: got %q want reinforce", r.EmphasisMode)
	}
}

func TestComputeProfile_NeutralEmpty(t *testing.T) {
	r := ComputeProfile(ProfileInput{
		UnitID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
	})
	if !r.IsNeutral || r.ProfileSignature != NeutralSignature {
		t.Fatalf("expected neutral base: %+v", r)
	}
	if r.EmphasisMode != EmphasisIntroduce {
		t.Fatalf("emphasis: %q", r.EmphasisMode)
	}
}

func TestComputeProfile_SignatureStableAcrossLearners(t *testing.T) {
	unit := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	c1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	c2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	mis := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	// Gaps that land in the same 0.1 buckets after quantization:
	// mastery 0.91 / 0.94 → gaps 0.09 / 0.06 → both 0.1
	// mastery 0.55 / 0.52 → gaps 0.45 / 0.48 → both 0.5
	inA := ProfileInput{
		UnitID:     unit,
		ConceptIDs: []uuid.UUID{c1, c2},
		ConceptMastery: map[uuid.UUID]float64{
			c1: 0.91,
			c2: 0.55,
		},
		MisconceptionIDs: []uuid.UUID{mis},
		AxisSet:          []string{"emphasis", "misconception"},
		ReadingLevelPref: "default",
		ModalityPref:     "default",
	}
	inB := ProfileInput{
		UnitID:     unit,
		ConceptIDs: []uuid.UUID{c2, c1}, // order differs
		ConceptMastery: map[uuid.UUID]float64{
			c1: 0.94,
			c2: 0.52,
		},
		MisconceptionIDs: []uuid.UUID{mis, mis}, // dupe
		AxisSet:          []string{"misconception", "emphasis"},
		ReadingLevelPref: "default",
		ModalityPref:     "default",
	}
	a := ComputeProfile(inA)
	b := ComputeProfile(inB)
	if a.ProfileSignature != b.ProfileSignature {
		t.Fatalf("signatures differ:\n  a=%s\n  b=%s\n  gapsA=%+v\n  gapsB=%+v",
			a.ProfileSignature, b.ProfileSignature, a.Payload.ConceptGaps, b.Payload.ConceptGaps)
	}
	if a.EmphasisMode != EmphasisRemediate {
		t.Fatalf("expected remediate, got %s", a.EmphasisMode)
	}
}

func TestBucketGap(t *testing.T) {
	cases := []struct {
		gap, bucket, want float64
	}{
		{0.0, 0.1, 0.0},
		{0.04, 0.1, 0.0},
		{0.05, 0.1, 0.1}, // round half away? math.Round half away from zero for positive: 0.5 → 1
		{0.14, 0.1, 0.1},
		{0.15, 0.1, 0.2},
		{1.0, 0.1, 1.0},
		{-0.5, 0.1, 0.0},
		{1.5, 0.1, 1.0},
	}
	for _, tc := range cases {
		got := BucketGap(tc.gap, tc.bucket)
		if got != tc.want {
			t.Errorf("BucketGap(%v,%v)=%v want %v", tc.gap, tc.bucket, got, tc.want)
		}
	}
}

func TestMasteryIsFresh(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	if MasteryIsFresh(nil, 30, now) {
		t.Fatal("nil not fresh")
	}
	recent := now.Add(-10 * 24 * time.Hour)
	if !MasteryIsFresh(&recent, 30, now) {
		t.Fatal("10d within 30d should be fresh")
	}
	old := now.Add(-40 * 24 * time.Hour)
	if MasteryIsFresh(&old, 30, now) {
		t.Fatal("40d not within 30d")
	}
}

func TestValidateTriggerMode(t *testing.T) {
	if err := ValidateTriggerMode("pre_quiz"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTriggerMode("nope"); err == nil {
		t.Fatal("expected error")
	}
	if NormalizeTriggerMode("") != TriggerPreQuiz {
		t.Fatal("default")
	}
}

func TestDefaultBloomForEmphasis(t *testing.T) {
	if DefaultBloomForEmphasis(EmphasisCompress) != "analyze" {
		t.Fatal()
	}
	if DefaultBloomForEmphasis(EmphasisIntroduce) != "remember" {
		t.Fatal()
	}
}
