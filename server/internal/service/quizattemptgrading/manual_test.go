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