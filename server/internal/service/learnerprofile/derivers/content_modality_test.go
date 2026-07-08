package derivers

import "testing"

func TestComputeModalityAffinity_VideoOverReading(t *testing.T) {
	items := []itemEngagement{
		{ItemKey: "v1", Modality: modalityVideo, MaxVideoPct: 90},
		{ItemKey: "v2", Modality: modalityVideo, MaxVideoPct: 90},
		{ItemKey: "r1", Modality: modalityReading, MaxScrollDepth: 30, ExpectedDurationSec: 300, TimeOnTaskSec: 60},
	}
	summary, sufficient, _ := computeContentModality(items)
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if summary.ModalityAffinity["video"] <= summary.ModalityAffinity["reading"] {
		t.Fatalf("video=%v reading=%v want video > reading", summary.ModalityAffinity["video"], summary.ModalityAffinity["reading"])
	}
}

func TestComputeModalityAffinity_ExposureSkewNormalised(t *testing.T) {
	items := make([]itemEngagement, 0, 11)
	for i := 0; i < 10; i++ {
		items = append(items, itemEngagement{
			ItemKey:     "v" + string(rune('a'+i)),
			Modality:    modalityVideo,
			MaxVideoPct: 90,
		})
	}
	items = append(items, itemEngagement{
		ItemKey:            "r1",
		Modality:           modalityReading,
		MaxScrollDepth:     30,
		ExpectedDurationSec: 300,
		TimeOnTaskSec:      60,
	})
	summary, sufficient, _ := computeContentModality(items)
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if summary.ModalityAffinity["video"] < 0.85 {
		t.Fatalf("video affinity=%v want ~0.9", summary.ModalityAffinity["video"])
	}
	if summary.ModalityAffinity["reading"] > 0.4 {
		t.Fatalf("reading affinity=%v want ~0.3", summary.ModalityAffinity["reading"])
	}
}

func TestComputeComplexityComfort_GradeBand(t *testing.T) {
	fkgl8, fkgl9, fkgl10, fkgl12 := 8.0, 9.0, 10.0, 12.0
	items := []itemEngagement{
		{ItemKey: "r8", Modality: modalityReading, ReadingLevelFKGL: &fkgl8, MaxScrollDepth: 85, ExpectedDurationSec: 300, TimeOnTaskSec: 240},
		{ItemKey: "r9", Modality: modalityReading, ReadingLevelFKGL: &fkgl9, MaxScrollDepth: 80, ExpectedDurationSec: 300, TimeOnTaskSec: 220},
		{ItemKey: "r10", Modality: modalityReading, ReadingLevelFKGL: &fkgl10, MaxScrollDepth: 75, ExpectedDurationSec: 300, TimeOnTaskSec: 200},
		{ItemKey: "r12", Modality: modalityReading, ReadingLevelFKGL: &fkgl12, MaxScrollDepth: 20, ExpectedDurationSec: 300, TimeOnTaskSec: 30},
		{ItemKey: "v1", Modality: modalityVideo, MaxVideoPct: 90},
		{ItemKey: "v2", Modality: modalityVideo, MaxVideoPct: 85},
	}
	summary, sufficient, _ := computeContentModality(items)
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if summary.ComplexityComfort == nil {
		t.Fatal("expected complexity comfort")
	}
	if summary.ComplexityComfort.Low != "grade8" {
		t.Fatalf("low=%q want grade8", summary.ComplexityComfort.Low)
	}
	if summary.ComplexityComfort.High != "grade10" {
		t.Fatalf("high=%q want grade10", summary.ComplexityComfort.High)
	}
}

func TestModalityDataSufficient_SingleModality(t *testing.T) {
	items := []itemEngagement{
		{ItemKey: "v1", Modality: modalityVideo, MaxVideoPct: 90},
		{ItemKey: "v2", Modality: modalityVideo, MaxVideoPct: 80},
		{ItemKey: "v3", Modality: modalityVideo, MaxVideoPct: 70},
	}
	_, sufficient, _ := computeContentModality(items)
	if sufficient {
		t.Fatal("expected insufficient_data for single modality")
	}
}

func TestComputePacingLabel_ThoroughVideoSkimText(t *testing.T) {
	items := []itemEngagement{
		{ItemKey: "v1", Modality: modalityVideo, MaxVideoPct: 90},
		{ItemKey: "r1", Modality: modalityReading, MaxScrollDepth: 25, ExpectedDurationSec: 300, TimeOnTaskSec: 60},
		{ItemKey: "v2", Modality: modalityVideo, MaxVideoPct: 88},
	}
	pacing := computePacingLabel(items)
	if pacing != "thorough-on-video-skim-on-text" {
		t.Fatalf("pacing=%q", pacing)
	}
}

func TestEstimateReadSeconds_WordCount(t *testing.T) {
	// 400 words at 200 WPM = 2 minutes.
	if got := estimateReadSeconds(400); got != 120 {
		t.Fatalf("seconds=%v want 120", got)
	}
	if got := estimateReadSeconds(0); got != contentModalityMinExpectedReadSec {
		t.Fatalf("seconds=%v want min", got)
	}
}

func TestMapItemTypeToModality(t *testing.T) {
	cases := map[string]modalityKind{
		"video":         modalityVideo,
		"content_page":  modalityReading,
		"quiz":          modalityQuiz,
		"h5p":           modalityInteractive,
		"vibe_activity": modalityInteractive,
		"unknown":       "",
	}
	for in, want := range cases {
		if got := mapItemTypeToModality(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}