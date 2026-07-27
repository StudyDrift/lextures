package worked_example

import (
	"encoding/json"
	"math"
	"strings"
	"unicode"

	"github.com/lextures/lextures/server/internal/service/mathnorm"
)

// GradeOutcome is the result of checking one step value.
type GradeOutcome struct {
	Result   AttemptResult
	Feedback string
}

// GradeStep checks a learner value against a blanked step.
func GradeStep(cfg Config, step Step, value string) GradeOutcome {
	if step.Blank == nil {
		return GradeOutcome{Result: ResultCorrect}
	}
	b := *step.Blank
	value = strings.TrimSpace(value)
	switch b.Type {
	case BlankNumeric:
		return gradeNumeric(b, value)
	case BlankExpression:
		return gradeExpression(cfg, b, value)
	case BlankChoice:
		return gradeChoice(b, value)
	case BlankText:
		return gradeText(b, value)
	default:
		return GradeOutcome{Result: ResultNeedsReview, Feedback: "Unsupported step type."}
	}
}

func gradeNumeric(b Blank, value string) GradeOutcome {
	got, ok := ParseNumericValue(value)
	if !ok {
		return GradeOutcome{Result: ResultIncorrect, Feedback: "Enter a number."}
	}
	target, ok := coerceExpectedNumber(b.Expected)
	if !ok {
		// Fall back to accepted answers as numeric strings.
		for _, a := range b.AcceptedAnswers {
			if t, ok2 := ParseNumericValue(a); ok2 {
				if numericClose(got, t, b.Tolerance) {
					return GradeOutcome{Result: ResultCorrect}
				}
			}
		}
		return GradeOutcome{Result: ResultNeedsReview, Feedback: "This step needs review."}
	}
	if numericClose(got, target, b.Tolerance) {
		return GradeOutcome{Result: ResultCorrect}
	}
	for _, a := range b.AcceptedAnswers {
		if t, ok2 := ParseNumericValue(a); ok2 && numericClose(got, t, b.Tolerance) {
			return GradeOutcome{Result: ResultCorrect}
		}
	}
	return GradeOutcome{Result: ResultIncorrect}
}

func gradeExpression(cfg Config, b Blank, value string) GradeOutcome {
	expected := stringify(b.Expected)
	if expected == "" && len(b.AcceptedAnswers) > 0 {
		expected = b.AcceptedAnswers[0]
	}
	if expected == "" {
		return GradeOutcome{Result: ResultNeedsReview, Feedback: "This step needs review."}
	}
	outcome := mathnorm.Compare(value, expected, cfg.Variables)
	switch outcome {
	case mathnorm.OutcomeEquivalent:
		return GradeOutcome{Result: ResultCorrect}
	case mathnorm.OutcomeDifferent:
		// Still try accepted answers list.
		for _, a := range b.AcceptedAnswers {
			if mathnorm.Compare(value, a, cfg.Variables) == mathnorm.OutcomeEquivalent {
				return GradeOutcome{Result: ResultCorrect}
			}
			if normalizeText(value, false) == normalizeText(a, false) {
				return GradeOutcome{Result: ResultCorrect}
			}
		}
		return GradeOutcome{Result: ResultIncorrect}
	default:
		for _, a := range b.AcceptedAnswers {
			if normalizeText(value, false) == normalizeText(a, false) {
				return GradeOutcome{Result: ResultCorrect}
			}
			if mathnorm.Compare(value, a, cfg.Variables) == mathnorm.OutcomeEquivalent {
				return GradeOutcome{Result: ResultCorrect}
			}
		}
		if normalizeText(value, false) == normalizeText(expected, false) {
			return GradeOutcome{Result: ResultCorrect}
		}
		return GradeOutcome{Result: ResultNeedsReview, Feedback: "Could not decide — marked for review."}
	}
}

func gradeChoice(b Blank, value string) GradeOutcome {
	if b.CorrectOptionID == "" {
		return GradeOutcome{Result: ResultNeedsReview, Feedback: "This step needs review."}
	}
	if value == b.CorrectOptionID {
		return GradeOutcome{Result: ResultCorrect}
	}
	return GradeOutcome{Result: ResultIncorrect}
}

func gradeText(b Blank, value string) GradeOutcome {
	got := normalizeText(value, false)
	if got == "" {
		return GradeOutcome{Result: ResultIncorrect}
	}
	if b.Expected != nil {
		if normalizeText(stringify(b.Expected), false) == got {
			return GradeOutcome{Result: ResultCorrect}
		}
	}
	for _, a := range b.AcceptedAnswers {
		if normalizeText(a, false) == got {
			return GradeOutcome{Result: ResultCorrect}
		}
	}
	if b.Expected == nil && len(b.AcceptedAnswers) == 0 {
		return GradeOutcome{Result: ResultNeedsReview, Feedback: "This step needs review."}
	}
	return GradeOutcome{Result: ResultIncorrect}
}

func numericClose(got, target float64, tol *Tolerance) bool {
	window := 0.0
	if tol != nil {
		switch tol.Kind {
		case ToleranceRelative:
			window = math.Abs(target) * tol.Value
		default:
			window = tol.Value
		}
	}
	return math.Abs(got-target) <= window+1e-9
}

func coerceExpectedNumber(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		return ParseNumericValue(t)
	default:
		return 0, false
	}
}

// ParseNumericValue accepts locale-aware decimal forms (comma or dot).
func ParseNumericValue(raw string) (float64, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false
	}
	s = normalizeLocaleNumber(s)
	n, err := json.Number(s).Float64()
	if err != nil {
		return 0, false
	}
	return n, true
}

func normalizeLocaleNumber(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\u00a0", "")
	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")
	switch {
	case hasDot && hasComma:
		lastDot := strings.LastIndex(s, ".")
		lastComma := strings.LastIndex(s, ",")
		if lastComma > lastDot {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	case hasComma && !hasDot:
		s = strings.Replace(s, ",", ".", 1)
	}
	return s
}

func normalizeText(s string, caseSensitive bool) string {
	s = strings.TrimSpace(s)
	if !caseSensitive {
		s = strings.ToLower(s)
	}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			if b.Len() > 0 && b.String()[b.Len()-1] != ' ' {
				b.WriteByte(' ')
			}
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// VerifyExpected runs the author's expected answer through the checker (authoring verify).
func VerifyExpected(cfg Config, step Step) GradeOutcome {
	if step.Blank == nil {
		return GradeOutcome{Result: ResultCorrect}
	}
	expected := ExpectedDisplay(step)
	if step.Blank.Type == BlankChoice {
		expected = step.Blank.CorrectOptionID
	} else if step.Blank.Expected != nil {
		expected = stringify(step.Blank.Expected)
	}
	return GradeStep(cfg, step, expected)
}
