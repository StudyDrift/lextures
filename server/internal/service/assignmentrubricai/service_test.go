package assignmentrubricai

import (
	"context"
	"testing"

	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

func TestServiceHealth(t *testing.T) {
	s := New()
	if s.Name != "assignmentrubricai" {
		t.Fatalf("name: %q", s.Name)
	}
	got, err := s.Health(context.Background())
	if err != nil || got != "assignmentrubricai:ok" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestParseModelJSON(t *testing.T) {
	raw := "```json\n" + `{
  "title": "Essay rubric",
  "criteria": [
    {
      "title": "Thesis",
      "description": "Clear argument",
      "levels": [
        {"label": "Needs work", "points": 0, "description": "Unclear"},
        {"label": "Excellent", "points": 5, "description": "Sharp"}
      ]
    },
    {
      "title": "Evidence",
      "levels": [
        {"label": "Weak", "points": 1},
        {"label": "Strong", "points": 5}
      ]
    }
  ]
}` + "\n```"
	rubric, err := parseModelJSON(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rubric.Title == nil || *rubric.Title != "Essay rubric" {
		t.Fatalf("title = %v", rubric.Title)
	}
	if len(rubric.Criteria) != 2 {
		t.Fatalf("criteria len = %d", len(rubric.Criteria))
	}
	if rubric.Criteria[0].ID.String() == "" {
		t.Fatal("expected criterion id")
	}
	// Labels from first row synced onto second.
	if rubric.Criteria[1].Levels[0].Label != "Needs work" || rubric.Criteria[1].Levels[1].Label != "Excellent" {
		t.Fatalf("synced labels = %#v", rubric.Criteria[1].Levels)
	}
	if err := assignmentrubric.ValidateRubricDefinition(rubric); err != nil {
		t.Fatalf("validate: %v", err)
	}
	pts := 10
	if err := validateAgainstPoints(rubric, &pts); err != nil {
		t.Fatalf("points: %v", err)
	}
}

func TestNormalizeRubricGridPadsLevels(t *testing.T) {
	r := normalizeRubricGrid(assignmentrubric.RubricDefinition{
		Criteria: []assignmentrubric.RubricCriterion{
			{Title: "A", Levels: []assignmentrubric.RubricLevel{{Label: "Low", Points: 0}, {Label: "High", Points: 2}}},
			{Title: "B", Levels: []assignmentrubric.RubricLevel{{Label: "x", Points: 1}}},
		},
	})
	if len(r.Criteria[1].Levels) != 2 {
		t.Fatalf("padded levels = %d", len(r.Criteria[1].Levels))
	}
	if r.Criteria[1].Levels[0].Label != "Low" || r.Criteria[1].Levels[1].Label != "High" {
		t.Fatalf("labels = %#v", r.Criteria[1].Levels)
	}
}

func TestGenerate_rejectsEmptyPrompt(t *testing.T) {
	_, _, err := Generate(context.Background(), nil, "model", "", GenerateInput{})
	if err == nil || err.Error() != "instructions are required" {
		t.Fatalf("err = %v", err)
	}
}
