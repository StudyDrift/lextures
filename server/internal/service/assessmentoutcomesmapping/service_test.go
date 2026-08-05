package assessmentoutcomesmapping

import (
	"strings"
	"testing"
)

func TestParseProposalsJSON(t *testing.T) {
	validOutcomes := map[string]struct{}{"o1": {}, "o2": {}}
	validItems := map[string]struct{}{"a1": {}, "a2": {}}
	outcomeTitles := map[string]string{"o1": "Critical thinking", "o2": "Writing"}
	itemTitles := map[string]string{"a1": "Essay", "a2": "Quiz 1"}

	raw := `{"proposals":[
		{"structureItemId":"a1","outcomeId":"o1","measurementLevel":"summative","intensityLevel":"high","confidence":0.9,"rationale":"Essay measures critical thinking."},
		{"structureItemId":"bad","outcomeId":"o1","measurementLevel":"summative","intensityLevel":"high","confidence":0.5,"rationale":"skip"},
		{"structureItemId":"a2","outcomeId":"o2","measurementLevel":"formative","intensityLevel":"medium","confidence":0.7,"rationale":"Quiz checks writing."}
	]}`
	itemKinds := map[string]string{"a1": "assignment", "a2": "quiz"}
	got, err := ParseProposalsJSON(raw, validOutcomes, validItems, outcomeTitles, itemTitles, itemKinds)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 proposals, got %d", len(got))
	}
	if got[0].StructureItemID != "a1" || got[0].OutcomeTitle != "Critical thinking" {
		t.Fatalf("unexpected first proposal: %+v", got[0])
	}
}

func TestLooksLikeLearnerField(t *testing.T) {
	if !looksLikeLearnerField("Student name: Alice") {
		t.Fatal("expected rejection of student name")
	}
	if looksLikeLearnerField("Midterm essay on climate policy") {
		t.Fatal("should allow normal title")
	}
}

func TestSuggestRejectsLearnerCriteria(t *testing.T) {
	// BuildInput path via looksLikeLearnerField used inside Suggest.
	if !looksLikeLearnerField("Submission text for grade for student") {
		t.Fatal("expected learner field detection")
	}
	// Ensure default system prompt forbids learner data.
	if !strings.Contains(DefaultSystemPrompt, "Never include student") {
		t.Fatal("system prompt must forbid student data")
	}
}
