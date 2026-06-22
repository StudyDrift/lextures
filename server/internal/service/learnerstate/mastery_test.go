package learnerstate

import (
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

func TestMasteryScaleForHintUses(t *testing.T) {
	if MasteryScaleForHintUses(0) != 1.0 {
		t.Fatal("expected 1.0 with no hints")
	}
	if MasteryScaleForHintUses(3) >= MasteryScaleForHintUses(0) {
		t.Fatal("expected fewer hints to yield higher scale")
	}
}

func TestCollectConceptTouchesFromQuestion_dedupesConceptPerQuestion(t *testing.T) {
	cid := uuid.New().String()
	q := &coursemodulequiz.QuizQuestion{
		ID:         "q1",
		ConceptIDs: []string{cid, cid},
	}
	var touches []ConceptTouch
	CollectConceptTouchesFromQuestion(q, 0, 1, 1, nil, 1, 1, &touches)
	if len(touches) != 1 {
		t.Fatalf("got %d touches want 1", len(touches))
	}
	if touches[0].Score != 1 {
		t.Fatalf("score %v want 1", touches[0].Score)
	}
}