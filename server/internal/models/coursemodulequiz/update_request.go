package coursemodulequiz

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// UnmarshalJSON supports sparse PATCH semantics for nullable fields:
// omit → leave unchanged; null → clear; value → set.
// Standard encoding/json cannot express this with **T alone.
func (r *UpdateModuleQuizRequest) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var out UpdateModuleQuizRequest

	if err := decodeOpt(raw, "title", &out.Title); err != nil {
		return err
	}
	if err := decodeOpt(raw, "markdown", &out.Markdown); err != nil {
		return err
	}
	if err := decodeOpt(raw, "questions", &out.Questions); err != nil {
		return err
	}
	if err := decodeNullable(raw, "dueAt", &out.DueAt); err != nil {
		return err
	}
	if err := decodeNullable(raw, "availableFrom", &out.AvailableFrom); err != nil {
		return err
	}
	if err := decodeNullable(raw, "availableUntil", &out.AvailableUntil); err != nil {
		return err
	}
	if err := decodeOpt(raw, "unlimitedAttempts", &out.UnlimitedAttempts); err != nil {
		return err
	}
	if err := decodeOpt(raw, "oneQuestionAtATime", &out.OneQuestionAtATime); err != nil {
		return err
	}
	if err := decodeOpt(raw, "maxAttempts", &out.MaxAttempts); err != nil {
		return err
	}
	if err := decodeOpt(raw, "gradeAttemptPolicy", &out.GradeAttemptPolicy); err != nil {
		return err
	}
	if err := decodeNullable(raw, "passingScorePercent", &out.PassingScorePercent); err != nil {
		return err
	}
	if err := decodeNullable(raw, "pointsWorth", &out.PointsWorth); err != nil {
		return err
	}
	if err := decodeOpt(raw, "lateSubmissionPolicy", &out.LateSubmissionPolicy); err != nil {
		return err
	}
	if err := decodeNullable(raw, "latePenaltyPercent", &out.LatePenaltyPercent); err != nil {
		return err
	}
	if err := decodeNullable(raw, "timeLimitMinutes", &out.TimeLimitMinutes); err != nil {
		return err
	}
	if err := decodeOpt(raw, "timerPauseWhenTabHidden", &out.TimerPauseWhenTabHidden); err != nil {
		return err
	}
	if err := decodeNullable(raw, "perQuestionTimeLimitSeconds", &out.PerQuestionTimeLimitSeconds); err != nil {
		return err
	}
	if err := decodeOpt(raw, "showScoreTiming", &out.ShowScoreTiming); err != nil {
		return err
	}
	if err := decodeOpt(raw, "reviewVisibility", &out.ReviewVisibility); err != nil {
		return err
	}
	if err := decodeOpt(raw, "reviewWhen", &out.ReviewWhen); err != nil {
		return err
	}
	if err := decodeOpt(raw, "shuffleQuestions", &out.ShuffleQuestions); err != nil {
		return err
	}
	if err := decodeOpt(raw, "shuffleChoices", &out.ShuffleChoices); err != nil {
		return err
	}
	if err := decodeOpt(raw, "allowBackNavigation", &out.AllowBackNavigation); err != nil {
		return err
	}
	if err := decodeNullable(raw, "quizAccessCode", &out.QuizAccessCode); err != nil {
		return err
	}
	if err := decodeOpt(raw, "adaptiveDifficulty", &out.AdaptiveDifficulty); err != nil {
		return err
	}
	if err := decodeOpt(raw, "adaptiveTopicBalance", &out.AdaptiveTopicBalance); err != nil {
		return err
	}
	if err := decodeOpt(raw, "adaptiveStopRule", &out.AdaptiveStopRule); err != nil {
		return err
	}
	if err := decodeNullable(raw, "randomQuestionPoolCount", &out.RandomQuestionPoolCount); err != nil {
		return err
	}
	if err := decodeOpt(raw, "lockdownMode", &out.LockdownMode); err != nil {
		return err
	}
	if err := decodeNullable(raw, "focusLossThreshold", &out.FocusLossThreshold); err != nil {
		return err
	}
	if err := decodeOpt(raw, "isAdaptive", &out.IsAdaptive); err != nil {
		return err
	}
	if err := decodeOpt(raw, "adaptiveSystemPrompt", &out.AdaptiveSystemPrompt); err != nil {
		return err
	}
	if err := decodeOpt(raw, "adaptiveSourceItemIds", &out.AdaptiveSourceItemIDs); err != nil {
		return err
	}
	if err := decodeOpt(raw, "adaptiveQuestionCount", &out.AdaptiveQuestionCount); err != nil {
		return err
	}
	if err := decodeOpt(raw, "adaptiveDeliveryMode", &out.AdaptiveDeliveryMode); err != nil {
		return err
	}
	if err := decodeOpt(raw, "neverDrop", &out.NeverDrop); err != nil {
		return err
	}
	if err := decodeOpt(raw, "replaceWithFinal", &out.ReplaceWithFinal); err != nil {
		return err
	}

	*r = out
	return nil
}

// HasUpdates reports whether any patch field was present in the JSON body.
func (r *UpdateModuleQuizRequest) HasUpdates() bool {
	if r == nil {
		return false
	}
	return r.Title != nil ||
		r.Markdown != nil ||
		r.Questions != nil ||
		r.DueAt != nil ||
		r.AvailableFrom != nil ||
		r.AvailableUntil != nil ||
		r.UnlimitedAttempts != nil ||
		r.OneQuestionAtATime != nil ||
		r.MaxAttempts != nil ||
		r.GradeAttemptPolicy != nil ||
		r.PassingScorePercent != nil ||
		r.PointsWorth != nil ||
		r.LateSubmissionPolicy != nil ||
		r.LatePenaltyPercent != nil ||
		r.TimeLimitMinutes != nil ||
		r.TimerPauseWhenTabHidden != nil ||
		r.PerQuestionTimeLimitSeconds != nil ||
		r.ShowScoreTiming != nil ||
		r.ReviewVisibility != nil ||
		r.ReviewWhen != nil ||
		r.ShuffleQuestions != nil ||
		r.ShuffleChoices != nil ||
		r.AllowBackNavigation != nil ||
		r.QuizAccessCode != nil ||
		r.AdaptiveDifficulty != nil ||
		r.AdaptiveTopicBalance != nil ||
		r.AdaptiveStopRule != nil ||
		r.RandomQuestionPoolCount != nil ||
		r.LockdownMode != nil ||
		r.FocusLossThreshold != nil ||
		r.IsAdaptive != nil ||
		r.AdaptiveSystemPrompt != nil ||
		r.AdaptiveSourceItemIDs != nil ||
		r.AdaptiveQuestionCount != nil ||
		r.AdaptiveDeliveryMode != nil ||
		r.NeverDrop != nil ||
		r.ReplaceWithFinal != nil
}

// ValidatePatch checks field values when present. Returns a client-facing message.
func (r *UpdateModuleQuizRequest) ValidatePatch() error {
	if r.Title != nil && strings.TrimSpace(*r.Title) == "" {
		return fmt.Errorf("Title is required.")
	}
	if r.MaxAttempts != nil && (*r.MaxAttempts < 1 || *r.MaxAttempts > 100) {
		return fmt.Errorf("maxAttempts must be between 1 and 100.")
	}
	if r.GradeAttemptPolicy != nil && !slices.Contains(GradeAttemptPolicies, strings.TrimSpace(*r.GradeAttemptPolicy)) {
		return fmt.Errorf("Invalid gradeAttemptPolicy.")
	}
	if r.PassingScorePercent != nil && *r.PassingScorePercent != nil {
		v := **r.PassingScorePercent
		if v < 0 || v > 100 {
			return fmt.Errorf("passingScorePercent must be between 0 and 100.")
		}
	}
	if r.PointsWorth != nil && *r.PointsWorth != nil {
		v := **r.PointsWorth
		if v < 0 || v > MaxItemPointsWorth {
			return fmt.Errorf("pointsWorth is out of range.")
		}
	}
	if r.LateSubmissionPolicy != nil {
		p := strings.TrimSpace(*r.LateSubmissionPolicy)
		if p != "allow" && p != "penalty" && p != "block" {
			return fmt.Errorf("Invalid lateSubmissionPolicy.")
		}
	}
	if r.LatePenaltyPercent != nil && *r.LatePenaltyPercent != nil {
		v := **r.LatePenaltyPercent
		if v < 0 || v > 100 {
			return fmt.Errorf("latePenaltyPercent must be between 0 and 100.")
		}
	}
	if r.TimeLimitMinutes != nil && *r.TimeLimitMinutes != nil {
		v := **r.TimeLimitMinutes
		if v < 1 || v > 10080 {
			return fmt.Errorf("timeLimitMinutes must be between 1 and 10080.")
		}
	}
	if r.PerQuestionTimeLimitSeconds != nil && *r.PerQuestionTimeLimitSeconds != nil {
		v := **r.PerQuestionTimeLimitSeconds
		if v < 10 || v > 86400 {
			return fmt.Errorf("perQuestionTimeLimitSeconds must be between 10 and 86400.")
		}
	}
	if r.ShowScoreTiming != nil && !slices.Contains(ShowScoreTimings, strings.TrimSpace(*r.ShowScoreTiming)) {
		return fmt.Errorf("Invalid showScoreTiming.")
	}
	if r.ReviewVisibility != nil && !slices.Contains(ReviewVisibilities, strings.TrimSpace(*r.ReviewVisibility)) {
		return fmt.Errorf("Invalid reviewVisibility.")
	}
	if r.ReviewWhen != nil && !slices.Contains(ReviewWhens, strings.TrimSpace(*r.ReviewWhen)) {
		return fmt.Errorf("Invalid reviewWhen.")
	}
	if r.QuizAccessCode != nil && *r.QuizAccessCode != nil {
		if len(strings.TrimSpace(**r.QuizAccessCode)) > MaxQuizAccessCodeLen {
			return fmt.Errorf("quizAccessCode is too long.")
		}
	}
	if r.AdaptiveDifficulty != nil && !slices.Contains(AdaptiveDifficulties, strings.TrimSpace(*r.AdaptiveDifficulty)) {
		return fmt.Errorf("Invalid adaptiveDifficulty.")
	}
	if r.AdaptiveStopRule != nil && !slices.Contains(AdaptiveStopRules, strings.TrimSpace(*r.AdaptiveStopRule)) {
		return fmt.Errorf("Invalid adaptiveStopRule.")
	}
	if r.RandomQuestionPoolCount != nil && *r.RandomQuestionPoolCount != nil {
		v := **r.RandomQuestionPoolCount
		if v < 1 || v > MaxQuizQuestions {
			return fmt.Errorf("randomQuestionPoolCount must be between 1 and %d.", MaxQuizQuestions)
		}
	}
	if r.LockdownMode != nil {
		m := strings.TrimSpace(*r.LockdownMode)
		if m != "standard" && m != "one_at_a_time" && m != "kiosk" {
			return fmt.Errorf("Invalid lockdownMode.")
		}
	}
	if r.FocusLossThreshold != nil && *r.FocusLossThreshold != nil {
		v := **r.FocusLossThreshold
		if v < 1 || v > 1000 {
			return fmt.Errorf("focusLossThreshold must be between 1 and 1000.")
		}
	}
	if r.Questions != nil && len(*r.Questions) > MaxQuizQuestions {
		return fmt.Errorf("A quiz may have at most %d questions.", MaxQuizQuestions)
	}
	if r.AdaptiveQuestionCount != nil {
		v := *r.AdaptiveQuestionCount
		if v < MinAdaptiveQuestionCount || v > MaxAdaptiveQuestionCount {
			return fmt.Errorf("adaptiveQuestionCount must be between %d and %d.", MinAdaptiveQuestionCount, MaxAdaptiveQuestionCount)
		}
	}
	if r.AdaptiveDeliveryMode != nil {
		m := strings.TrimSpace(*r.AdaptiveDeliveryMode)
		if m != "ai" && m != "cat" {
			return fmt.Errorf("Invalid adaptiveDeliveryMode.")
		}
	}
	return nil
}

func decodeOpt[T any](raw map[string]json.RawMessage, key string, dst **T) error {
	v, ok := raw[key]
	if !ok {
		return nil
	}
	if string(v) == "null" {
		return nil
	}
	var val T
	if err := json.Unmarshal(v, &val); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	*dst = &val
	return nil
}

func decodeNullable[T any](raw map[string]json.RawMessage, key string, dst ***T) error {
	v, ok := raw[key]
	if !ok {
		return nil
	}
	if string(v) == "null" {
		var nilT *T
		*dst = &nilT
		return nil
	}
	var val T
	if err := json.Unmarshal(v, &val); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	p := &val
	*dst = &p
	return nil
}
