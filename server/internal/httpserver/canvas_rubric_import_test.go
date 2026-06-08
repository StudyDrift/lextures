package httpserver

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestCanvasOptionalRubricJSONFromAssignment(t *testing.T) {
	obj := map[string]any{
		"name": "Essay 1",
		"rubric": []any{
			map[string]any{
				"id":          "crit1",
				"description": "Thesis",
				"ratings": []any{
					map[string]any{"description": "Excellent", "points": float64(10)},
					map[string]any{"description": "Weak", "points": float64(0)},
				},
			},
			map[string]any{
				"id":                 "crit2",
				"description":        "Bonus",
				"ignore_for_scoring": true,
				"ratings": []any{
					map[string]any{"description": "Extra", "points": float64(1)},
				},
			},
		},
	}
	raw, err := canvasOptionalRubricJSONFromAssignment(obj)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("expected rubric json")
	}
	var parsed struct {
		Title    *string `json:"title"`
		Criteria []struct {
			ID     uuid.UUID `json:"id"`
			Title  string    `json:"title"`
			Levels []struct {
				Label  string  `json:"label"`
				Points float64 `json:"points"`
			} `json:"levels"`
		} `json:"criteria"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Title == nil || *parsed.Title != "Essay 1" {
		t.Fatalf("title: %#v", parsed.Title)
	}
	if len(parsed.Criteria) != 1 {
		t.Fatalf("criteria count: %d", len(parsed.Criteria))
	}
	if parsed.Criteria[0].Title != "Thesis" {
		t.Fatalf("criterion title: %q", parsed.Criteria[0].Title)
	}
	if len(parsed.Criteria[0].Levels) != 2 {
		t.Fatalf("levels: %d", len(parsed.Criteria[0].Levels))
	}
	if parsed.Criteria[0].Levels[0].Label != "Excellent" || parsed.Criteria[0].Levels[0].Points != 10 {
		t.Fatalf("first level: %+v", parsed.Criteria[0].Levels[0])
	}
}

func TestCanvasRubricIDFromAssignment(t *testing.T) {
	obj := map[string]any{
		"rubric_settings": map[string]any{"id": float64(42)},
	}
	if got := canvasRubricIDFromAssignment(obj); got != 42 {
		t.Fatalf("got %d", got)
	}
}

func TestCanvasRubricLevelsFromRatingsSortsDescending(t *testing.T) {
	levels := canvasRubricLevelsFromRatings([]map[string]any{
		{"description": "Low", "points": float64(1)},
		{"description": "High", "points": float64(5)},
	})
	if len(levels) != 2 || levels[0].Points != 5 || levels[1].Points != 1 {
		t.Fatalf("levels: %+v", levels)
	}
}
