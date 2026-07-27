package media_checkpoints

import "testing"

func TestGradeCheckpointSingle(t *testing.T) {
	cp := Checkpoint{
		ID: "c1",
		Question: Question{
			Type:   TypeSingle,
			Prompt: "Cap?",
			Options: []Option{
				{ID: "a", Text: "A", Correct: false, Feedback: "no"},
				{ID: "b", Text: "B", Correct: true, Feedback: "yes"},
			},
		},
	}
	g := GradeCheckpoint(cp, "b")
	if !g.Correct || g.Feedback != "yes" {
		t.Fatalf("got %#v", g)
	}
	g2 := GradeCheckpoint(cp, "a")
	if g2.Correct || g2.Feedback != "no" {
		t.Fatalf("got %#v", g2)
	}
}

func TestGradeCheckpointNumeric(t *testing.T) {
	cv := 3.14
	cp := Checkpoint{
		ID: "c2",
		Question: Question{
			Type:         TypeNumeric,
			Prompt:       "pi?",
			CorrectValue: &cv,
			Tolerance:    &Tolerance{Kind: ToleranceAbsolute, Value: 0.05},
		},
	}
	if !GradeCheckpoint(cp, "3,14").Correct {
		t.Fatal("locale numeric should pass")
	}
}

func TestComputeScoreLastAttempt(t *testing.T) {
	req := true
	attempts := 2
	cfg := Config{
		Checkpoints: []Checkpoint{
			{ID: "c1", AtSec: 10, Required: &req, Attempts: &attempts, Question: Question{Type: TypeTrueFalse, Prompt: "t"}},
			{ID: "c2", AtSec: 20, Required: &req, Attempts: &attempts, Question: Question{Type: TypeTrueFalse, Prompt: "t"}},
		},
	}
	st := EmptyState()
	st.Answers["c1"] = CheckpointAnswer{
		Attempts: []Attempt{
			{Value: "false", Correct: false, At: NowRFC3339()},
			{Value: "true", Correct: true, At: NowRFC3339()},
		},
		Done: true,
	}
	raw, max := ComputeScore(cfg, st)
	if raw != 1 || max != 2 {
		t.Fatalf("score=%v/%v", raw, max)
	}
}
