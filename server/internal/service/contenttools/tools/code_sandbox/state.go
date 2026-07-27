package code_sandbox

import (
	"encoding/json"
	"strings"
	"time"
)

// ParseConfig unmarshals instructor config with defaults applied.
func ParseConfig(raw json.RawMessage) Config {
	cfg := DefaultConfig()
	if len(raw) == 0 {
		return cfg
	}
	var overlay struct {
		Language          *string     `json:"language"`
		Prompt            *string     `json:"prompt"`
		StarterCode       *string     `json:"starterCode"`
		PrefixCode        *string     `json:"prefixCode"`
		SuffixCode        *string     `json:"suffixCode"`
		SampleInput       *string     `json:"sampleInput"`
		Tests             []TestCase  `json:"tests"`
		RunLimitPerHour   *int        `json:"runLimitPerHour"`
		CheckLimitPerHour *int        `json:"checkLimitPerHour"`
		EditorMode        *EditorMode `json:"editorMode"`
		ScoringMode       *ScoringMode `json:"scoringMode"`
		ErrorHints        []ErrorHint `json:"errorHints"`
		OutputLimitBytes  *int        `json:"outputLimitBytes"`
		MaxRunHistory     *int        `json:"maxRunHistory"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Language != nil {
		cfg.Language = NormalizeLanguage(*overlay.Language)
	}
	if overlay.Prompt != nil {
		cfg.Prompt = *overlay.Prompt
	}
	if overlay.StarterCode != nil {
		cfg.StarterCode = *overlay.StarterCode
	}
	if overlay.PrefixCode != nil {
		cfg.PrefixCode = *overlay.PrefixCode
	}
	if overlay.SuffixCode != nil {
		cfg.SuffixCode = *overlay.SuffixCode
	}
	if overlay.SampleInput != nil {
		cfg.SampleInput = *overlay.SampleInput
	}
	if overlay.Tests != nil {
		cfg.Tests = overlay.Tests
		if len(cfg.Tests) > MaxTests {
			cfg.Tests = cfg.Tests[:MaxTests]
		}
	}
	if overlay.RunLimitPerHour != nil && *overlay.RunLimitPerHour > 0 {
		cfg.RunLimitPerHour = *overlay.RunLimitPerHour
	}
	if overlay.CheckLimitPerHour != nil && *overlay.CheckLimitPerHour > 0 {
		cfg.CheckLimitPerHour = *overlay.CheckLimitPerHour
	}
	if overlay.EditorMode != nil {
		switch *overlay.EditorMode {
		case EditorRich, EditorPlain, EditorUserChoice:
			cfg.EditorMode = *overlay.EditorMode
		}
	}
	if overlay.ScoringMode != nil {
		switch *overlay.ScoringMode {
		case ScoringAuto, ScoringNone:
			cfg.ScoringMode = *overlay.ScoringMode
		}
	}
	if overlay.ErrorHints != nil {
		cfg.ErrorHints = overlay.ErrorHints
	}
	if overlay.OutputLimitBytes != nil && *overlay.OutputLimitBytes > 0 {
		cfg.OutputLimitBytes = *overlay.OutputLimitBytes
	}
	if overlay.MaxRunHistory != nil && *overlay.MaxRunHistory > 0 {
		cfg.MaxRunHistory = *overlay.MaxRunHistory
	}
	return cfg
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
	if st.Runs == nil {
		st.Runs = []RunRecord{}
	}
	return st
}

// NormalizeLanguage maps aliases to runner-supported runtimes.
func NormalizeLanguage(lang string) string {
	r := strings.TrimSpace(strings.ToLower(lang))
	switch {
	case r == "" || strings.HasPrefix(r, "python"):
		return "python"
	case strings.HasPrefix(r, "javascript") || strings.HasPrefix(r, "node") || r == "js":
		return "javascript"
	default:
		return r
	}
}

// SupportedLanguage reports whether the language is runner-supported.
func SupportedLanguage(lang string) bool {
	switch NormalizeLanguage(lang) {
	case "python", "javascript":
		return true
	default:
		return false
	}
}

// ComposeCode prepends/appends read-only prefix/suffix around learner code.
func ComposeCode(cfg Config, learnerCode string) string {
	var b strings.Builder
	if cfg.PrefixCode != "" {
		b.WriteString(cfg.PrefixCode)
		if !strings.HasSuffix(cfg.PrefixCode, "\n") {
			b.WriteByte('\n')
		}
	}
	b.WriteString(learnerCode)
	if cfg.SuffixCode != "" {
		if !strings.HasSuffix(learnerCode, "\n") && learnerCode != "" {
			b.WriteByte('\n')
		}
		b.WriteString(cfg.SuffixCode)
	}
	return b.String()
}

// TruncateOutput caps output and appends an explicit truncation marker (FR-7).
func TruncateOutput(s string, maxBytes int) string {
	if maxBytes <= 0 {
		maxBytes = DefaultOutputLimitBytes
	}
	if len(s) <= maxBytes {
		return s
	}
	// Keep room for the marker itself.
	cut := maxBytes
	if cut > len(TruncationMarker) {
		cut = maxBytes - len(TruncationMarker)
	}
	if cut < 0 {
		cut = 0
	}
	return s[:cut] + TruncationMarker
}

// AppendRun adds a history entry and caps length.
func AppendRun(st State, rec RunRecord, maxHistory int) State {
	if maxHistory <= 0 {
		maxHistory = DefaultMaxRunHistory
	}
	st.Runs = append(st.Runs, rec)
	if len(st.Runs) > maxHistory {
		st.Runs = st.Runs[len(st.Runs)-maxHistory:]
	}
	return st
}

// UpdateBest updates best check result when improved.
func UpdateBest(st State, passed, total int, at string) State {
	if total <= 0 {
		return st
	}
	if st.Best == nil || passed > st.Best.Passed || (passed == st.Best.Passed && total >= st.Best.Total) {
		st.Best = &BestResult{Passed: passed, Total: total, At: at}
	}
	return st
}

// HourKey returns the UTC hour bucket key for rate accounting.
func HourKey(t time.Time) string {
	return t.UTC().Format("2006-01-02T15")
}

// HourResetAt returns the unix seconds when the current hour window resets.
func HourResetAt(t time.Time) int64 {
	utc := t.UTC()
	next := time.Date(utc.Year(), utc.Month(), utc.Day(), utc.Hour()+1, 0, 0, 0, time.UTC)
	return next.Unix()
}

// EnsureRateWindow resets counters when the hour rolls.
func EnsureRateWindow(st State, now time.Time) State {
	key := HourKey(now)
	if st.Rate == nil || st.Rate.HourKey != key {
		st.Rate = &RateWindow{HourKey: key}
	}
	return st
}

// RateLimitError is returned when run/check hourly limits are hit.
type RateLimitError struct {
	Action  RunAction
	Limit   int
	ResetAt int64
	Message string
}

// CheckRateLimit returns a RateLimitError when the learner is over the hourly cap.
func CheckRateLimit(cfg Config, st State, action RunAction, now time.Time) *RateLimitError {
	st = EnsureRateWindow(st, now)
	reset := HourResetAt(now)
	switch action {
	case ActionRun:
		limit := cfg.RunLimitPerHour
		if limit <= 0 {
			limit = 30
		}
		if st.Rate != nil && st.Rate.Runs >= limit {
			return &RateLimitError{
				Action:  ActionRun,
				Limit:   limit,
				ResetAt: reset,
				Message: "Run rate limit reached. Try again after the hourly reset.",
			}
		}
	case ActionCheck:
		limit := cfg.CheckLimitPerHour
		if limit <= 0 {
			limit = 20
		}
		if st.Rate != nil && st.Rate.Checks >= limit {
			return &RateLimitError{
				Action:  ActionCheck,
				Limit:   limit,
				ResetAt: reset,
				Message: "Check rate limit reached. Try again after the hourly reset.",
			}
		}
	}
	return nil
}

// RecordRateUsage increments the hourly counter for an action.
func RecordRateUsage(st State, action RunAction, now time.Time) State {
	st = EnsureRateWindow(st, now)
	switch action {
	case ActionRun:
		st.Rate.Runs++
	case ActionCheck:
		st.Rate.Checks++
	}
	return st
}

// MatchErrorHint returns the first author hint whose match appears in stderr.
func MatchErrorHint(cfg Config, stderr string) string {
	if stderr == "" || len(cfg.ErrorHints) == 0 {
		return ""
	}
	lower := strings.ToLower(stderr)
	for _, h := range cfg.ErrorHints {
		m := strings.TrimSpace(h.Match)
		if m == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(m)) {
			return h.Hint
		}
	}
	return ""
}

// EffectiveScoringMode returns auto only when tests exist and author allows scoring.
func EffectiveScoringMode(cfg Config) ScoringMode {
	if len(cfg.Tests) == 0 {
		return ScoringNone
	}
	if cfg.ScoringMode == ScoringNone {
		return ScoringNone
	}
	return ScoringAuto
}

// NowRFC3339 returns the current UTC time as RFC3339.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// InitialCode returns starter code when state code is empty.
func InitialCode(cfg Config, st State) string {
	if strings.TrimSpace(st.Code) != "" {
		return st.Code
	}
	return cfg.StarterCode
}
