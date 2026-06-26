package gradingagent

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"
)

const maxRegexInputLen = 100_000

// RouterCondition is the persisted predicate on a conditional router node.
type RouterCondition struct {
	Field    string
	Operator string
	Value    any
}

// PredicateEvalContext supplies runtime values for router predicate evaluation.
type PredicateEvalContext struct {
	SubmissionText   string
	IsLate           bool
	OriginalityScore *float64
	InputGrade       *GradeOutput
	InputScore       *float64
}

var allowedRouterFields = map[string]struct{}{
	"submissionLength": {},
	"wordCount":        {},
	"isEmpty":          {},
	"score":            {},
	"confidence":       {},
	"originalityScore": {},
	"isLate":           {},
	"submissionText":   {},
	"matchesRegex":     {},
}

var allowedRouterOperators = map[string]struct{}{
	"<":            {},
	"<=":           {},
	"==":           {},
	">=":           {},
	">":            {},
	"isTrue":       {},
	"contains":     {},
	"matchesRegex": {},
}

func parseRouterCondition(data map[string]any) (RouterCondition, error) {
	if data == nil {
		return RouterCondition{}, fmt.Errorf("condition is required")
	}
	raw, ok := data["condition"].(map[string]any)
	if !ok || raw == nil {
		return RouterCondition{}, fmt.Errorf("condition is required")
	}
	field, _ := raw["field"].(string)
	field = strings.TrimSpace(field)
	op, _ := raw["operator"].(string)
	op = strings.TrimSpace(op)
	if field == "" || op == "" {
		return RouterCondition{}, fmt.Errorf("condition field and operator are required")
	}
	if _, ok := allowedRouterFields[field]; !ok {
		return RouterCondition{}, fmt.Errorf("unknown condition field %q", field)
	}
	if _, ok := allowedRouterOperators[op]; !ok {
		return RouterCondition{}, fmt.Errorf("unknown condition operator %q", op)
	}
	return RouterCondition{Field: field, Operator: op, Value: raw["value"]}, nil
}

func routerConditionFromNode(n WorkflowNode) (RouterCondition, error) {
	return parseRouterCondition(n.Data)
}

// EvaluateRouterCondition evaluates a router predicate deterministically (no LLM).
func EvaluateRouterCondition(cond RouterCondition, ctx PredicateEvalContext) (bool, error) {
	field := cond.Field
	op := cond.Operator

	switch field {
	case "isEmpty", "isLate":
		if op != "isTrue" {
			return false, fmt.Errorf("field %q only supports isTrue operator", field)
		}
		want := conditionBoolValue(cond.Value, true)
		actual := field == "isEmpty" && isSubmissionEmpty(ctx.SubmissionText)
		if field == "isLate" {
			actual = ctx.IsLate
		}
		return actual == want, nil
	case "submissionLength":
		return compareNumbers(float64(len(ctx.SubmissionText)), op, cond.Value)
	case "wordCount":
		return compareNumbers(float64(submissionWordCount(ctx.SubmissionText)), op, cond.Value)
	case "score":
		if ctx.InputGrade == nil {
			return false, fmt.Errorf("score is not available on this path")
		}
		return compareNumbers(ctx.InputGrade.TotalPoints, op, cond.Value)
	case "confidence":
		if ctx.InputGrade == nil {
			return false, fmt.Errorf("confidence is not available on this path")
		}
		return compareNumbers(ctx.InputGrade.Confidence, op, cond.Value)
	case "originalityScore":
		if ctx.OriginalityScore == nil {
			return false, fmt.Errorf("originality score is not available on this path")
		}
		return compareNumbers(*ctx.OriginalityScore, op, cond.Value)
	case "submissionText", "matchesRegex":
		text := ctx.SubmissionText
		if len(text) > maxRegexInputLen {
			text = text[:maxRegexInputLen]
		}
		pattern, err := conditionStringValue(cond.Value)
		if err != nil {
			return false, err
		}
		switch op {
		case "contains":
			return strings.Contains(text, pattern), nil
		case "matchesRegex":
			re, compileErr := regexp.Compile(pattern)
			if compileErr != nil {
				return false, fmt.Errorf("invalid regex pattern: %w", compileErr)
			}
			return re.MatchString(text), nil
		default:
			return false, fmt.Errorf("field %q does not support operator %q", field, op)
		}
	default:
		return false, fmt.Errorf("unknown condition field %q", field)
	}
}

func isSubmissionEmpty(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	for _, r := range trimmed {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func submissionWordCount(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	count := 0
	inWord := false
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				count++
				inWord = true
			}
			continue
		}
		inWord = false
	}
	return count
}

func compareNumbers(actual float64, op string, expected any) (bool, error) {
	target, err := conditionFloatValue(expected)
	if err != nil {
		return false, err
	}
	switch op {
	case "<":
		return actual < target, nil
	case "<=":
		return actual <= target, nil
	case "==":
		return math.Abs(actual-target) < 1e-9, nil
	case ">=":
		return actual >= target, nil
	case ">":
		return actual > target, nil
	default:
		return false, fmt.Errorf("operator %q is not valid for numeric comparison", op)
	}
}

func conditionFloatValue(v any) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case int:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case string:
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(t), "%f", &f); err != nil {
			return 0, fmt.Errorf("expected numeric condition value")
		}
		return f, nil
	default:
		return 0, fmt.Errorf("expected numeric condition value")
	}
}

func conditionBoolValue(v any, defaultVal bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case nil:
		return defaultVal
	default:
		return defaultVal
	}
}

func conditionStringValue(v any) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string condition value")
	}
	return s, nil
}

func formatRouterConditionSentence(cond RouterCondition) string {
	switch cond.Field {
	case "isEmpty", "isLate":
		return fmt.Sprintf("%s is true", cond.Field)
	case "submissionText", "matchesRegex":
		return fmt.Sprintf("%s %s %q", cond.Field, cond.Operator, cond.Value)
	default:
		return fmt.Sprintf("%s %s %v", cond.Field, cond.Operator, cond.Value)
	}
}
