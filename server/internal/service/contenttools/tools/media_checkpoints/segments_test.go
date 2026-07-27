package media_checkpoints

import "testing"

func TestNormalizeSegmentsCoarseAndMerge(t *testing.T) {
	got := NormalizeSegments([][2]float64{
		{0.2, 3.1},
		{2.5, 6.2},
		{300, 360},
		{5, 5.5},
	}, 5)
	if len(got) != 2 {
		t.Fatalf("want 2 merged segments, got %#v", got)
	}
	if got[0][0] != 0 || got[0][1] != 10 {
		t.Fatalf("first segment: %#v", got[0])
	}
	if got[1][0] != 300 || got[1][1] != 360 {
		t.Fatalf("second segment: %#v", got[1])
	}
}

func TestNormalizeSegmentsAC6(t *testing.T) {
	// AC-6: watch 0:00–3:00 and 5:00–6:00 → coarse segments, furthest 6:00.
	st := EmptyState()
	MergeWatchProgress(&st, [][2]float64{{0, 180}, {300, 360}}, 360)
	if len(st.WatchedSegments) != 2 {
		t.Fatalf("segments: %#v", st.WatchedSegments)
	}
	if st.FurthestSec != 360 {
		t.Fatalf("furthest=%v want 360", st.FurthestSec)
	}
	if st.WatchedSegments[0][0] != 0 || st.WatchedSegments[0][1] != 180 {
		t.Fatalf("seg0=%v", st.WatchedSegments[0])
	}
	if st.WatchedSegments[1][0] != 300 || st.WatchedSegments[1][1] != 360 {
		t.Fatalf("seg1=%v", st.WatchedSegments[1])
	}
}

func TestWatchedBins(t *testing.T) {
	bins := WatchedBins([][2]float64{{0, 12}, {20, 25}}, 5)
	want := []string{"0-5", "10-15", "20-25", "5-10"}
	if len(bins) != len(want) {
		t.Fatalf("bins=%v want %v", bins, want)
	}
	for i := range want {
		if bins[i] != want[i] {
			t.Fatalf("bins=%v want %v", bins, want)
		}
	}
}

func TestAllRequiredComplete(t *testing.T) {
	req := true
	cfg := Config{
		Checkpoints: []Checkpoint{
			{ID: "c1", AtSec: 30, Required: &req, Question: Question{Type: TypeSingle, Prompt: "q"}},
			{ID: "c2", AtSec: 90, Required: &req, Question: Question{Type: TypeSingle, Prompt: "q"}},
		},
	}
	st := EmptyState()
	if AllRequiredComplete(cfg, st) {
		t.Fatal("empty state should not be complete")
	}
	st.Answers["c1"] = CheckpointAnswer{
		Attempts: []Attempt{{Value: "a", Correct: true, At: NowRFC3339()}},
		Done:     true,
	}
	if AllRequiredComplete(cfg, st) {
		t.Fatal("partial answers should not complete")
	}
	st.Answers["c2"] = CheckpointAnswer{
		Attempts: []Attempt{{Value: "a", Correct: true, At: NowRFC3339()}},
		Done:     true,
	}
	if !AllRequiredComplete(cfg, st) {
		t.Fatal("all required done should complete")
	}
}
