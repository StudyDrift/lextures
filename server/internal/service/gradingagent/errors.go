package gradingagent

import "strings"

// UserFacingScoreError maps internal scoring failures to instructor-safe messages.
func UserFacingScoreError(err error) string {
	if err == nil {
		return "Grading agent failed."
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "grader agent model not configured"):
		return "Grading agent model is not configured. Choose a model under Settings → Intelligence → Models."
	case strings.Contains(msg, "AI provider not configured"), strings.Contains(msg, "missing API key"):
		return "AI provider is not configured. Set an OpenRouter API key under Settings → Intelligence → Models."
	case strings.Contains(msg, "openrouter: status 401"), strings.Contains(msg, "openrouter: status 403"):
		return "OpenRouter rejected the API key. Check Settings → Intelligence → Models."
	case strings.Contains(msg, "openrouter: status 402"):
		return "OpenRouter account has insufficient credits."
	case strings.Contains(msg, "openrouter: status 404"):
		return "The selected AI model was not found on OpenRouter."
	case strings.HasPrefix(msg, "openrouter:"):
		return "OpenRouter request failed: " + truncateErr(msg, 240)
	case strings.Contains(msg, "invalid model JSON"), strings.Contains(msg, "empty model response"):
		return "The AI returned an unreadable grade. Try dry run again or choose a different model."
	case strings.Contains(msg, "invalid rubric scores"):
		return "The AI returned rubric scores that could not be applied. Try again without including the rubric, or adjust your prompt."
	case strings.Contains(msg, "submission text is empty"):
		return "Submission text is empty."
	default:
		return "Grading agent failed: " + truncateErr(msg, 240)
	}
}

func truncateErr(msg string, max int) string {
	msg = strings.TrimSpace(msg)
	if len(msg) <= max {
		return msg
	}
	return msg[:max] + "…"
}