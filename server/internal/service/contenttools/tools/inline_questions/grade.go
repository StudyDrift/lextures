package inline_questions

import (
	"encoding/json"
	"math"
	"strings"
	"unicode"
)

// GradeResult is the outcome of grading one response.
type GradeResult struct {
	Correct        bool
	PointsAwarded  float64
	Feedback       string
	Explanation    string
	CorrectAnswer  any
	MatchedOptionIDs []string
}

// GradeQuestion grades a learner value against a question definition.
func GradeQuestion(q Question, value any) GradeResult {
	maxPts := PointsFor(q)
	out := GradeResult{}
	switch q.Type {
	case TypeSingle, TypeTrueFalse:
		got, ok := coerceString(value)
		if !ok {
			return out
		}
		var selected *Option
		for i := range q.Options {
			if q.Options[i].ID == got {
				selected = &q.Options[i]
				break
			}
		}
		if selected == nil {
			return out
		}
		out.MatchedOptionIDs = []string{selected.ID}
		out.Feedback = selected.Feedback
		out.Correct = selected.Correct
		if out.Correct {
			out.PointsAwarded = maxPts
		}
		out.CorrectAnswer = correctOptionIDs(q)
		out.Explanation = q.Explanation
		return out

	case TypeMulti:
		got, ok := coerceStringSlice(value)
		if !ok {
			return out
		}
		out.MatchedOptionIDs = append([]string{}, got...)
		correctIDs := correctOptionIDs(q)
		out.CorrectAnswer = correctIDs
		out.Explanation = q.Explanation
		// Collect feedback from selected options (first non-empty).
		for _, id := range got {
			for i := range q.Options {
				if q.Options[i].ID == id && q.Options[i].Feedback != "" {
					if out.Feedback != "" {
						out.Feedback += " "
					}
					out.Feedback += q.Options[i].Feedback
				}
			}
		}
		if q.PartialCredit {
			out.PointsAwarded, out.Correct = gradeMultiPartial(got, correctIDs, maxPts)
		} else {
			out.Correct = sameStringSet(got, correctIDs)
			if out.Correct {
				out.PointsAwarded = maxPts
			}
		}
		return out

	case TypeShortText:
		got, ok := coerceString(value)
		if !ok {
			return out
		}
		out.Correct = shortTextCorrect(q, got)
		if out.Correct {
			out.PointsAwarded = maxPts
		}
		out.CorrectAnswer = append([]string{}, q.AcceptedAnswers...)
		out.Explanation = q.Explanation
		return out

	case TypeNumeric:
		got, ok := coerceNumber(value)
		if !ok {
			return out
		}
		out.Correct = numericCorrect(q, got)
		if out.Correct {
			out.PointsAwarded = maxPts
		}
		if q.CorrectValue != nil {
			out.CorrectAnswer = *q.CorrectValue
		}
		out.Explanation = q.Explanation
		return out
	}
	return out
}

func correctOptionIDs(q Question) []string {
	var out []string
	for _, o := range q.Options {
		if o.Correct {
			out = append(out, o.ID)
		}
	}
	return out
}

func gradeMultiPartial(got, expected []string, maxPts float64) (float64, bool) {
	if len(expected) == 0 {
		return 0, false
	}
	expSet := map[string]struct{}{}
	for _, id := range expected {
		expSet[id] = struct{}{}
	}
	gotSet := map[string]struct{}{}
	for _, id := range got {
		gotSet[id] = struct{}{}
	}
	correctSelected := 0
	incorrectSelected := 0
	for id := range gotSet {
		if _, ok := expSet[id]; ok {
			correctSelected++
		} else {
			incorrectSelected++
		}
	}
	missed := len(expSet) - correctSelected
	if incorrectSelected == 0 && missed == 0 {
		return maxPts, true
	}
	// Simple partial: proportion of correct options selected, minus wrong selections.
	raw := float64(correctSelected) / float64(len(expSet))
	penalty := float64(incorrectSelected) / float64(len(expSet))
	pts := math.Max(0, raw-penalty) * maxPts
	pts = math.Round(pts*100) / 100
	return pts, false
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[string]struct{}{}
	for _, x := range a {
		set[x] = struct{}{}
	}
	for _, x := range b {
		if _, ok := set[x]; !ok {
			return false
		}
	}
	return true
}

func shortTextCorrect(q Question, answer string) bool {
	got := normalizeShortText(answer, q.CaseSensitive, q.NormalizePunctuation)
	if got == "" {
		return false
	}
	for _, a := range q.AcceptedAnswers {
		if normalizeShortText(a, q.CaseSensitive, q.NormalizePunctuation) == got {
			return true
		}
	}
	return false
}

func normalizeShortText(s string, caseSensitive, normalizePunct bool) string {
	s = strings.TrimSpace(s)
	if !caseSensitive {
		s = strings.ToLower(s)
	}
	if normalizePunct {
		var b strings.Builder
		for _, r := range s {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
				b.WriteRune(r)
			}
		}
		s = strings.Join(strings.Fields(b.String()), " ")
	}
	return s
}

func numericCorrect(q Question, value float64) bool {
	if q.CorrectValue == nil {
		return false
	}
	target := *q.CorrectValue
	tol := 0.0
	if q.Tolerance != nil {
		switch q.Tolerance.Kind {
		case ToleranceRelative:
			tol = math.Abs(target) * q.Tolerance.Value
		default:
			tol = q.Tolerance.Value
		}
	}
	return math.Abs(value-target) <= tol+1e-9
}

func coerceString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case json.Number:
		return t.String(), true
	case float64:
		return "", false
	default:
		return "", false
	}
}

func coerceStringSlice(v any) ([]string, bool) {
	switch t := v.(type) {
	case []string:
		return append([]string{}, t...), true
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, ok := coerceString(item)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	case string:
		return []string{t}, true
	default:
		return nil, false
	}
}

// ParseNumericValue accepts locale-aware decimal forms (comma or dot).
func ParseNumericValue(raw string) (float64, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false
	}
	// Prefer last comma/dot as decimal separator when both present.
	s = normalizeLocaleNumber(s)
	n, err := json.Number(s).Float64()
	if err != nil {
		f, err2 := parseFloatLoose(s)
		if err2 != nil {
			return 0, false
		}
		return f, true
	}
	return n, true
}

func coerceNumber(v any) (float64, bool) {
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

func normalizeLocaleNumber(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\u00a0", "")
	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")
	switch {
	case hasDot && hasComma:
		// European: 1.234,56 or US: 1,234.56 — use the rightmost separator as decimal.
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

func parseFloatLoose(s string) (float64, error) {
	return json.Number(s).Float64()
}
