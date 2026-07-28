package explain_it_back

import (
	"encoding/json"
	"strings"
	"time"
	"unicode"
)

// ParseConfig unmarshals instructor config with manifest defaults applied.
func ParseConfig(raw json.RawMessage) Config {
	cfg := DefaultConfig()
	if len(raw) == 0 {
		return cfg
	}
	var overlay struct {
		Prompt                     *string        `json:"prompt"`
		MinWords                   *int           `json:"minWords"`
		MaxWords                   *int           `json:"maxWords"`
		KeyPoints                  []KeyPoint     `json:"keyPoints"`
		RevealKeyPointsAfterSubmit *bool          `json:"revealKeyPointsAfterSubmit"`
		AIFeedback                 *bool          `json:"aiFeedback"`
		FeedbackStyle              *FeedbackStyle `json:"feedbackStyle"`
		Attempts                   *int           `json:"attempts"`
		IncludeProbeQuestion       *bool          `json:"includeProbeQuestion"`
		AllowInstructorNote        *bool          `json:"allowInstructorNote"`
		MaxSubmissionsPerDay       *int           `json:"maxSubmissionsPerDay"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Prompt != nil {
		cfg.Prompt = *overlay.Prompt
	}
	if overlay.MinWords != nil {
		n := *overlay.MinWords
		if n >= 5 && n <= 500 {
			cfg.MinWords = n
		}
	}
	if overlay.MaxWords != nil {
		n := *overlay.MaxWords
		if n >= 10 && n <= 1000 {
			cfg.MaxWords = n
		}
	}
	if overlay.KeyPoints != nil {
		cfg.KeyPoints = sanitizeKeyPoints(overlay.KeyPoints)
	}
	if overlay.RevealKeyPointsAfterSubmit != nil {
		cfg.RevealKeyPointsAfterSubmit = *overlay.RevealKeyPointsAfterSubmit
	}
	if overlay.AIFeedback != nil {
		cfg.AIFeedback = *overlay.AIFeedback
	}
	if overlay.FeedbackStyle != nil {
		switch *overlay.FeedbackStyle {
		case FeedbackEncouraging, FeedbackNeutral, FeedbackSocratic:
			cfg.FeedbackStyle = *overlay.FeedbackStyle
		}
	}
	if overlay.Attempts != nil {
		n := *overlay.Attempts
		if n >= 1 && n <= 10 {
			cfg.Attempts = n
		}
	}
	if overlay.IncludeProbeQuestion != nil {
		cfg.IncludeProbeQuestion = *overlay.IncludeProbeQuestion
	}
	if overlay.AllowInstructorNote != nil {
		cfg.AllowInstructorNote = *overlay.AllowInstructorNote
	}
	if overlay.MaxSubmissionsPerDay != nil {
		n := *overlay.MaxSubmissionsPerDay
		if n >= 1 && n <= 50 {
			cfg.MaxSubmissionsPerDay = n
		}
	}
	if cfg.MaxWords < cfg.MinWords {
		cfg.MaxWords = cfg.MinWords
	}
	return cfg
}

func sanitizeKeyPoints(in []KeyPoint) []KeyPoint {
	out := make([]KeyPoint, 0, len(in))
	seen := map[string]struct{}{}
	for _, kp := range in {
		id := strings.TrimSpace(kp.ID)
		label := strings.TrimSpace(kp.Label)
		desc := strings.TrimSpace(kp.Description)
		if id == "" || label == "" || desc == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, KeyPoint{ID: id, Label: label, Description: desc})
		if len(out) >= 6 {
			break
		}
	}
	return out
}

// ParseState unmarshals learner state with defaults.
func ParseState(raw json.RawMessage) State {
	st := EmptyState()
	if len(raw) == 0 {
		return st
	}
	_ = json.Unmarshal(raw, &st)
	if st.V == 0 {
		st.V = 1
	}
	if st.Attempts == nil {
		st.Attempts = []Attempt{}
	}
	return st
}

// TodayUTC returns YYYY-MM-DD in UTC.
func TodayUTC(now time.Time) string {
	return now.UTC().Format("2006-01-02")
}

// NowRFC3339 returns the current UTC time in RFC3339.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// SubmissionsRemaining returns how many submissions the learner may still make today.
func SubmissionsRemaining(st State, maxPerDay int, now time.Time) int {
	if maxPerDay <= 0 {
		maxPerDay = 10
	}
	today := TodayUTC(now)
	used := 0
	if st.SubmittedToday != nil && st.SubmittedToday.Date == today {
		used = st.SubmittedToday.Count
	}
	left := maxPerDay - used
	if left < 0 {
		return 0
	}
	return left
}

// IncrementSubmittedToday bumps the daily counter.
func IncrementSubmittedToday(st *State, now time.Time) {
	today := TodayUTC(now)
	if st.SubmittedToday == nil || st.SubmittedToday.Date != today {
		st.SubmittedToday = &SubmittedToday{Date: today, Count: 1}
		return
	}
	st.SubmittedToday.Count++
}

// AttemptsRemaining returns how many revisions remain (config.attempts total).
func AttemptsRemaining(cfg Config, st State) int {
	max := cfg.Attempts
	if max <= 0 {
		max = 3
	}
	left := max - len(st.Attempts)
	if left < 0 {
		return 0
	}
	return left
}

// CountWords counts whitespace-separated words (Unicode-aware letters/digits).
func CountWords(text string) int {
	n := 0
	inWord := false
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				n++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	return n
}

// MeetsLengthGuide reports whether text is within the configured word range.
func MeetsLengthGuide(cfg Config, text string) bool {
	n := CountWords(text)
	return n >= cfg.MinWords && n <= cfg.MaxWords
}

// KeyPointLabels returns id→label for configured points.
func KeyPointLabels(cfg Config) map[string]string {
	out := make(map[string]string, len(cfg.KeyPoints))
	for _, kp := range cfg.KeyPoints {
		out[kp.ID] = kp.Label
	}
	return out
}

// ReviewFeedback builds the non-AI acknowledgement feedback object.
func ReviewFeedback() Feedback {
	return Feedback{
		Covered:    []string{},
		Missing:    []string{},
		Strength:   ReviewAcknowledgement,
		Suggestion: "Your instructor can leave a short note after reviewing.",
		Mode:       FeedbackModeReview,
	}
}

// RepresentativeExplanation is an anonymised sample for class insights.
type RepresentativeExplanation struct {
	Text         string   `json:"text"`
	CoveredCount int      `json:"coveredCount"`
	CoveredIDs   []string `json:"coveredIds"`
}

// SelectRepresentatives picks up to max anonymised explanations deterministically
// (longest unique texts first among first attempts).
func SelectRepresentatives(states []State, max int) []RepresentativeExplanation {
	if max <= 0 {
		max = 5
	}
	type cand struct {
		text    string
		covered []string
	}
	seen := map[string]struct{}{}
	var cands []cand
	for _, st := range states {
		if len(st.Attempts) == 0 {
			continue
		}
		a := st.Attempts[0]
		text := strings.TrimSpace(a.Text)
		if text == "" {
			continue
		}
		key := strings.ToLower(text)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		covered := []string{}
		if a.Feedback != nil {
			covered = append([]string{}, a.Feedback.Covered...)
		}
		cands = append(cands, cand{text: summarizeClip(text, 220), covered: covered})
	}
	// Sort by covered count desc, then text length desc (stable insertion).
	for i := 0; i < len(cands); i++ {
		for j := i + 1; j < len(cands); j++ {
			if len(cands[j].covered) > len(cands[i].covered) ||
				(len(cands[j].covered) == len(cands[i].covered) && len(cands[j].text) > len(cands[i].text)) {
				cands[i], cands[j] = cands[j], cands[i]
			}
		}
	}
	if len(cands) > max {
		cands = cands[:max]
	}
	out := make([]RepresentativeExplanation, 0, len(cands))
	for _, c := range cands {
		out = append(out, RepresentativeExplanation{
			Text:         c.text,
			CoveredCount: len(c.covered),
			CoveredIDs:   c.covered,
		})
	}
	return out
}

func summarizeClip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
