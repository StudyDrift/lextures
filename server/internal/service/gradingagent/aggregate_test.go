package gradingagent

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

func gradeInput(sourceID string, pts float64, conf float64, rubric map[string]float64) AggregatorInput {
	g := GradeOutput{TotalPoints: pts, Confidence: conf, RubricScores: rubric}
	return AggregatorInput{SourceID: sourceID, Label: sourceID, Grade: &g, Weight: 1}
}

func TestCombineGrades_sum(t *testing.T) {
	inputs := []AggregatorInput{
		gradeInput("a", 4, 0.9, map[string]float64{"c1": 4}),
		gradeInput("b", 3, 0.8, map[string]float64{"c2": 3}),
		gradeInput("c", 5, 0.7, map[string]float64{"c3": 5}),
	}
	cfg := AggregatorConfig{Mode: AggregatorModeSum, Confidence: AggregatorConfidenceMin}
	out, _, err := CombineGrades(inputs, cfg, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalPoints != 12 {
		t.Fatalf("total = %v, want 12", out.TotalPoints)
	}
	if len(out.RubricScores) != 3 {
		t.Fatalf("rubric map len = %d, want 3", len(out.RubricScores))
	}
	if out.Confidence != 0.7 {
		t.Fatalf("confidence = %v, want 0.7", out.Confidence)
	}
}

func TestCombineGrades_weightedSum(t *testing.T) {
	auto := gradeInput("auto", 80, 0.9, nil)
	auto.Weight = 0.7
	ai := gradeInput("ai", 100, 0.8, nil)
	ai.Weight = 0.3
	cfg := AggregatorConfig{Mode: AggregatorModeWeightedSum, Confidence: AggregatorConfidenceMin}
	out, _, err := CombineGrades([]AggregatorInput{auto, ai}, cfg, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalPoints != 86 {
		t.Fatalf("total = %v, want 86", out.TotalPoints)
	}
}

func TestCombineGrades_minConfidence(t *testing.T) {
	inputs := []AggregatorInput{
		gradeInput("a", 10, 0.9, nil),
		gradeInput("b", 10, 0.4, nil),
	}
	cfg := AggregatorConfig{Mode: AggregatorModeSum, Confidence: AggregatorConfidenceMin}
	out, _, err := CombineGrades(inputs, cfg, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Confidence != 0.4 {
		t.Fatalf("confidence = %v, want 0.4", out.Confidence)
	}
}

func TestCombineGrades_skipAndRenormalize(t *testing.T) {
	auto := gradeInput("auto", 80, 0.9, nil)
	auto.Weight = 0.7
	ai := gradeInput("ai", 100, 0.8, nil)
	ai.Weight = 0.3
	ai.Missing = true
	cfg := AggregatorConfig{
		Mode:       AggregatorModeWeightedSum,
		Confidence: AggregatorConfidenceMin,
		OnMissing:  AggregatorOnMissingSkipAndRenormalize,
	}
	out, _, err := CombineGrades([]AggregatorInput{auto, ai}, cfg, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalPoints != 80 {
		t.Fatalf("total = %v, want 80", out.TotalPoints)
	}
}

func TestCombineGrades_rubricMergeConflict(t *testing.T) {
	cid := uuid.New().String()
	inputs := []AggregatorInput{
		gradeInput("a", 4, 0.9, map[string]float64{cid: 4}),
		gradeInput("b", 3, 0.8, map[string]float64{cid: 3}),
	}
	cfg := AggregatorConfig{Mode: AggregatorModeRubricMerge, Confidence: AggregatorConfidenceMin}
	_, _, err := CombineGrades(inputs, cfg, 100, nil)
	if err == nil {
		t.Fatal("expected rubric merge conflict error")
	}
}

func TestCombineGrades_clampMaxPoints(t *testing.T) {
	inputs := []AggregatorInput{gradeInput("a", 150, 1, nil)}
	cfg := AggregatorConfig{Mode: AggregatorModeSum, Confidence: AggregatorConfidenceMin}
	out, _, err := CombineGrades(inputs, cfg, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalPoints != 100 {
		t.Fatalf("total = %v, want 100", out.TotalPoints)
	}
}

func TestCombineGrades_rubricMergeWithDefinition(t *testing.T) {
	c1 := uuid.New()
	c2 := uuid.New()
	rubric := &assignmentrubric.RubricDefinition{
		Criteria: []assignmentrubric.RubricCriterion{
			{ID: c1, Title: "A", Levels: []assignmentrubric.RubricLevel{{Points: 4}, {Points: 3}}},
			{ID: c2, Title: "B", Levels: []assignmentrubric.RubricLevel{{Points: 5}, {Points: 2}}},
		},
	}
	inputs := []AggregatorInput{
		gradeInput("a", 4, 0.9, map[string]float64{c1.String(): 4}),
		gradeInput("b", 5, 0.8, map[string]float64{c2.String(): 5}),
	}
	cfg := AggregatorConfig{Mode: AggregatorModeRubricMerge, Confidence: AggregatorConfidenceMin}
	out, _, err := CombineGrades(inputs, cfg, 100, rubric)
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalPoints != 9 {
		t.Fatalf("total = %v, want 9", out.TotalPoints)
	}
}

func TestDetectRubricMergeCriterionConflicts(t *testing.T) {
	dupes := DetectRubricMergeCriterionConflicts([]string{"a", "b", "a"})
	if len(dupes) != 1 || dupes[0] != "a" {
		t.Fatalf("dupes = %v, want [a]", dupes)
	}
}