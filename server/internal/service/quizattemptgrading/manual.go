package quizattemptgrading

// ManualGradingQuestionTypes are quiz question types that require instructor scoring.
var ManualGradingQuestionTypes = []string{
	"essay",
	"file_upload",
	"audio_response",
	"video_response",
	"code",
	"hotspot",
	"formula",
}

func isManualGradingQuestionType(qtype string) bool {
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