package learnerprofile

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestParseModalityAffinity(t *testing.T) {
	summary, _ := json.Marshal(map[string]any{
		"modalityAffinity": map[string]float64{"video": 0.82, "reading": 0.34},
	})
	aff, pref := parseModalityAffinity(summary)
	if pref != "video" {
		t.Fatalf("pref=%q", pref)
	}
	if aff["video"] != 0.82 {
		t.Fatalf("affinity=%v", aff)
	}
}

func TestParseStrengthsGrowth(t *testing.T) {
	summary, _ := json.Marshal(map[string]any{
		"growth":      []map[string]string{{"concept": "fractions"}},
		"needsReview": []map[string]string{{"concept": "decimals"}},
	})
	growth, needs := parseStrengthsGrowth(summary)
	if len(growth) != 1 || growth[0] != "fractions" {
		t.Fatalf("growth=%v", growth)
	}
	if len(needs) != 1 || needs[0] != "decimals" {
		t.Fatalf("needs=%v", needs)
	}
}

func TestEvalCohortStable(t *testing.T) {
	id := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	a := evalCohortForUser(id)
	b := evalCohortForUser(id)
	if a != b {
		t.Fatalf("cohort not stable: %q vs %q", a, b)
	}
}

func TestMatchConcept(t *testing.T) {
	if got := MatchConcept("Review photosynthesis basics", []string{"Photosynthesis"}); got != "Photosynthesis" {
		t.Fatalf("got %q", got)
	}
}