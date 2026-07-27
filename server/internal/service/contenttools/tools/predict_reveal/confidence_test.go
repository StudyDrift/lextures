package predict_reveal

import (
	"encoding/json"
	"testing"
)

func TestNormalizeConfidence(t *testing.T) {
	cases := []struct {
		scale   ConfidenceScale
		raw     float64
		norm    float64
		bucket  string
		wantErr bool
	}{
		{ScaleThree, 1, 0, "guessing", false},
		{ScaleThree, 2, 0.5, "fairly_sure", false},
		{ScaleThree, 3, 1, "certain", false},
		{ScaleThree, 0, 0, "", true},
		{ScaleFive, 5, 1, "5", false},
		{ScaleFive, 1, 0, "1", false},
		{ScalePercent, 85, 0.85, "80_100", false},
		{ScalePercent, 10, 0.1, "0_20", false},
		{ScalePercent, 101, 0, "", true},
		{ScaleNone, 99, 0, "none", false},
	}
	for _, tc := range cases {
		n, b, err := NormalizeConfidence(tc.scale, tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("scale=%s raw=%v: want error", tc.scale, tc.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("scale=%s raw=%v: %v", tc.scale, tc.raw, err)
		}
		if n != tc.norm || b != tc.bucket {
			t.Fatalf("scale=%s raw=%v: got (%v,%q) want (%v,%q)", tc.scale, tc.raw, n, b, tc.norm, tc.bucket)
		}
	}
}

func TestTagCorrectness(t *testing.T) {
	cfg := Config{
		Mode: ModeChoice,
		Outcomes: []Outcome{
			{ID: "a", Text: "A", Correct: false},
			{ID: "b", Text: "B", Correct: true},
		},
	}
	c := TagCorrectness(cfg, "b")
	if c == nil || !*c {
		t.Fatal("want correct=true")
	}
	c = TagCorrectness(cfg, "a")
	if c == nil || *c {
		t.Fatal("want correct=false")
	}
	open := Config{Mode: ModeOpen}
	if TagCorrectness(open, "x") != nil {
		t.Fatal("open mode should not tag")
	}
	unmarked := Config{
		Mode: ModeChoice,
		Outcomes: []Outcome{
			{ID: "a", Text: "A"},
			{ID: "b", Text: "B"},
		},
	}
	if TagCorrectness(unmarked, "a") != nil {
		t.Fatal("no correct markers → nil")
	}
}

func TestBuildCalibrationMatrix_HighlightsConfidentlyWrong(t *testing.T) {
	rows := []struct {
		Bucket  string
		Correct bool
	}{
		{"certain", false},
		{"certain", false},
		{"certain", true},
		{"guessing", false},
	}
	cells := BuildCalibrationMatrix(rows)
	var found bool
	for _, c := range cells {
		if c.ConfidenceBucket == "certain" && !c.Correct {
			if !c.Highlight || c.Count != 2 {
				t.Fatalf("confidently-wrong cell: %#v", c)
			}
			found = true
		}
		if c.ConfidenceBucket == "guessing" && c.Highlight {
			t.Fatal("guessing wrong should not highlight")
		}
	}
	if !found {
		t.Fatal("missing highlighted cell")
	}
}

func TestBuildPeerResults_SmallN(t *testing.T) {
	blobs := []json.RawMessage{
		json.RawMessage(`{"v":1,"committedAt":"2026-01-01T00:00:00Z","prediction":{"outcomeId":"a"},"confidenceBucket":"certain"}`),
		json.RawMessage(`{"v":1,"committedAt":"2026-01-01T00:00:00Z","prediction":{"outcomeId":"b"},"confidenceBucket":"guessing"}`),
		json.RawMessage(`{"v":1,"committedAt":"2026-01-01T00:00:00Z","prediction":{"outcomeId":"a"},"confidenceBucket":"certain"}`),
	}
	pr := BuildPeerResults(blobs, ModeChoice)
	if !pr.Suppressed || pr.Reason != "small_n" || pr.Learners != 3 {
		t.Fatalf("want suppressed small_n n=3, got %#v", pr)
	}
}

func TestBuildPeerResults_Ok(t *testing.T) {
	var blobs []json.RawMessage
	for i := 0; i < 5; i++ {
		blobs = append(blobs, json.RawMessage(`{"v":1,"committedAt":"2026-01-01T00:00:00Z","prediction":{"outcomeId":"a"},"confidenceBucket":"certain"}`))
	}
	pr := BuildPeerResults(blobs, ModeChoice)
	if pr.Suppressed || pr.Learners != 5 || len(pr.Outcomes) == 0 {
		t.Fatalf("want peer results, got %#v", pr)
	}
}

func TestGuardStatePut(t *testing.T) {
	blocked, _ := GuardStatePut([]byte(`{}`), []byte(`{"draft":{"text":"x"}}`))
	if blocked {
		t.Fatal("uncommitted should allow")
	}
	blocked, msg := GuardStatePut(
		[]byte(`{"v":1,"committedAt":"2026-01-01T00:00:00Z","prediction":{"outcomeId":"a"}}`),
		[]byte(`{"prediction":{"outcomeId":"b"}}`),
	)
	if !blocked || msg == "" {
		t.Fatal("committed should refuse")
	}
}

func TestRedactRevealViaSchema(t *testing.T) {
	// Smoke: ParseConfig keeps reveal for server-side use.
	cfg := ParseConfig(json.RawMessage(`{
		"question":"What happens?",
		"mode":"choice",
		"outcomes":[{"id":"a","text":"A"},{"id":"b","text":"B","correct":true}],
		"reveal":{"markdown":"It expands."}
	}`))
	if cfg.Reveal.Markdown != "It expands." {
		t.Fatalf("reveal: %#v", cfg.Reveal)
	}
	if cfg.Mode != ModeChoice || len(cfg.Outcomes) != 2 {
		t.Fatalf("config: %#v", cfg)
	}
}
