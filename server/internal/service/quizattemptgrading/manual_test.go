package quizattemptgrading

import "testing"

func TestResponseNeedsManualGrading_essayUngraded(t *testing.T) {
	if !ResponseNeedsManualGrading("essay", nil, 0, 5) {
		t.Fatal("essay with no correctness should need manual grading")
	}
}

func TestResponseNeedsManualGrading_essayGraded(t *testing.T) {
	correct := true
	if ResponseNeedsManualGrading("essay", &correct, 5, 5) {
		t.Fatal("essay marked correct should not need manual grading")
	}
}

func TestResponseNeedsManualGrading_multipleChoiceIgnored(t *testing.T) {
	wrong := false
	if ResponseNeedsManualGrading("multiple_choice", &wrong, 0, 1) {
		t.Fatal("auto-graded multiple choice should not need manual grading")
	}
}

func TestResponseNeedsManualGrading_shortAnswerUngraded(t *testing.T) {
	if !ResponseNeedsManualGrading("short_answer", nil, 0, 3) {
		t.Fatal("short_answer with no correctness should need manual grading")
	}
}

func TestCorrectnessFromManualPoints_partialCredit(t *testing.T) {
	c := CorrectnessFromManualPoints(2, 5)
	if c == nil || *c {
		t.Fatalf("partial credit should mark incorrect, got %v", c)
	}
}