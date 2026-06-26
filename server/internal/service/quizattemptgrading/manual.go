package quizattemptgrading

import "strings"

// ManualGradingQuestionTypes are quiz question types that require instructor scoring.
var ManualGradingQuestionTypes = []string{
	"essay",
	"short_answer",
	"fill_in_blank",
	"file_upload",
	"audio_response",
	"video_response",
	"code",
	"hotspot",
	"formula",
}

// ManualGradingQuestionTypesSQL returns a comma-separated quoted list for SQL IN clauses.
func ManualGradingQuestionTypesSQL() string {
	parts := make([]string, len(ManualGradingQuestionTypes))
	for i, t := range ManualGradingQuestionTypes {
		parts[i] = "'" + t + "'"
	}
	return strings.Join(parts, ", ")
}

func isManualGradingQuestionType(qtype string) bool {
	return IsManualGradingQuestionType(qtype)
}

// IsManualGradingQuestionType reports whether instructors must score this question type.
func IsManualGradingQuestionType(qtype string) bool {
	for _, t := range ManualGradingQuestionTypes {
		if t == qtype {
			return true
		}
	}
	return false
}

// ResponseNeedsManualGrading reports whether a stored response still needs instructor scoring.
func ResponseNeedsManualGrading(questionType string, isCorrect *bool, pointsAwarded, maxPoints float64) bool {
	if !isManualGradingQuestionType(questionType) {
		return false
	}
	if isCorrect != nil {
		return false
	}
	if maxPoints > 0 && pointsAwarded >= maxPoints-0.0001 {
		return false
	}
	return true
}

// CorrectnessFromManualPoints derives is_correct after an instructor sets points on a manual item.
func CorrectnessFromManualPoints(pointsAwarded, maxPoints float64) *bool {
	if maxPoints <= 0 {
		c := pointsAwarded > 0.0001
		return &c
	}
	if pointsAwarded >= maxPoints-0.0001 {
		t := true
		return &t
	}
	f := false
	return &f
}