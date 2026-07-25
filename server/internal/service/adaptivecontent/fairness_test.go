package adaptivecontent

import "testing"

func TestSuppressFairnessMean(t *testing.T) {
	m := 0.9
	if SuppressFairnessMean(SmallCellMinN-1, &m) != nil {
		t.Fatal("small cell should suppress")
	}
	got := SuppressFairnessMean(SmallCellMinN, &m)
	if got == nil || *got != 0.9 {
		t.Fatalf("got %v", got)
	}
}

func TestFlagDisparity(t *testing.T) {
	group := 0.7
	ref := 0.9
	if !FlagDisparity(FairnessMinN, &group, &ref, 0.10) {
		t.Fatal("expected disparity")
	}
	if FlagDisparity(FairnessMinN-1, &group, &ref, 0.10) {
		t.Fatal("n too small")
	}
	close := 0.85
	if FlagDisparity(FairnessMinN, &close, &ref, 0.10) {
		t.Fatal("within threshold")
	}
}

func TestEvaluateFairnessCells_DisparityAndSuppression(t *testing.T) {
	fidA, fidB := 0.95, 0.70
	liftA, liftB := 20.0, 5.0
	cells := []FairnessCellInput{
		{Dimension: "language", GroupLabel: "en", N: 20, MeanFidelity: &fidA, MeanLift: &liftA},
		{Dimension: "language", GroupLabel: "es", N: 20, MeanFidelity: &fidB, MeanLift: &liftB},
		{Dimension: "language", GroupLabel: "tiny", N: 2, MeanFidelity: &fidB, MeanLift: &liftB},
	}
	out := EvaluateFairnessCells(cells)
	if len(out) != 3 {
		t.Fatalf("len %d", len(out))
	}
	var es, tiny *FairnessCellResult
	for i := range out {
		switch out[i].GroupLabel {
		case "es":
			es = &out[i]
		case "tiny":
			tiny = &out[i]
		}
	}
	if es == nil || !es.DisparityFlag {
		t.Fatal("es should be flagged")
	}
	if tiny == nil {
		t.Fatal("missing tiny")
	}
	if tiny.MeanFidelity != nil || tiny.MeanLift != nil {
		t.Fatal("tiny cell means should be suppressed")
	}
	if tiny.DisparityFlag {
		t.Fatal("tiny should not flag (n < FairnessMinN)")
	}
}
