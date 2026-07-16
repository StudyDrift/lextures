package gradingagent

import "strings"

// UserFacingScoreError maps internal scoring failures to instructor-safe messages.
// Copy is provider-agnostic (AP.4 FR-8): it must not assume OpenRouter is the
// backend, since AI.Complete/CompleteVision may be served by any configured
// aiprovider backend.
func UserFacingScoreError(err error) string {
	if err == nil {
		return "Grading agent failed."
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "grader agent model not configured"):
		return "Grading agent model is not configured. Choose a model under Settings → Intelligence → Models."
	case strings.Contains(msg, "AI provider not configured"),
		strings.Contains(msg, "missing API key"),
		strings.Contains(msg, "AI not configured"):
		return "AI provider is not configured. Configure AI under Settings → Intelligence."
	case strings.Contains(msg, "status 401"), strings.Contains(msg, "status 403"):
		return "AI provider rejected the API key. Check Settings → Intelligence."
	case strings.Contains(msg, "status 402"):
		return "AI provider account has insufficient credits."
	case strings.Contains(msg, "status 404"):
		return "The selected AI model was not found."
	case strings.Contains(msg, "invalid model JSON"), strings.Contains(msg, "empty model response"):
		return "The AI returned an unreadable grade. Try dry run again or choose a different model."
	case strings.Contains(msg, "invalid rubric scores"):
		return "The AI returned rubric scores that could not be applied. Try again without including the rubric, or adjust your prompt."
	case strings.Contains(msg, "submission text is empty"):
		return "Submission text is empty."
	default:
		return "AI request failed: " + truncateErr(msg, 240)
	}
}

func truncateErr(msg string, max int) string {
	msg = strings.TrimSpace(msg)
	if len(msg) <= max {
		return msg
	}
	return msg[:max] + "…"
}
