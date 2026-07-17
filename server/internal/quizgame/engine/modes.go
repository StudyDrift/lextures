package engine

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SessionMode is quizgame.session_mode (plan IQ.3 / IQ.6).
type SessionMode string

const (
	ModeLiveClassic  SessionMode = "live_classic"
	ModeTeam         SessionMode = "team"
	ModeStudentPaced SessionMode = "student_paced"
	ModeHomework     SessionMode = "homework"
)

// TeamAggregate is how member scores roll up (IQ.6 FR-2).
type TeamAggregate string

const (
	TeamAggregateAverage TeamAggregate = "average"
	TeamAggregateSum     TeamAggregate = "sum"
)

// TeamAnswerRule controls who submits in team mode (IQ.6 FR-3).
type TeamAnswerRule string

const (
	TeamAnswerEachMember TeamAnswerRule = "each_member_answers"
	TeamAnswerOneDevice  TeamAnswerRule = "one_device_per_team"
)

// GradePolicy is homework attempt aggregation (IQ.6 FR-7).
type GradePolicy string

const (
	GradePolicyBest    GradePolicy = "best"
	GradePolicyLast    GradePolicy = "last"
	GradePolicyAverage GradePolicy = "average"
)

// TeamConfig is stored in sessions.settings for mode=team.
type TeamConfig struct {
	TeamCount   int            `json:"teamCount"`
	Aggregate   TeamAggregate  `json:"aggregate"`
	AnswerRule  TeamAnswerRule `json:"answerRule"`
	AutoBalance bool           `json:"autoBalance"`
}

// PacedConfig is stored in sessions.settings for mode=student_paced.
type PacedConfig struct {
	Shuffle            bool `json:"shuffle"`
	TimeBudgetSeconds  int  `json:"timeBudgetSeconds"`  // 0 = none
	PerQuestionTimers  bool `json:"perQuestionTimers"`  // default true
	LiveLeaderboard    bool `json:"liveLeaderboard"`    // default false (final-only)
}

// ModeSettings is the JSON blob shape in sessions.settings for IQ.6.
type ModeSettings struct {
	Team  *TeamConfig  `json:"team,omitempty"`
	Paced *PacedConfig `json:"paced,omitempty"`
}

// NormalizeMode returns a known session mode or live_classic.
func NormalizeMode(m string) SessionMode {
	switch SessionMode(strings.TrimSpace(m)) {
	case ModeTeam, ModeStudentPaced, ModeHomework, ModeLiveClassic:
		return SessionMode(strings.TrimSpace(m))
	default:
		return ModeLiveClassic
	}
}

// DefaultTeamConfig returns IQ.6 recommended defaults (average aggregate).
func DefaultTeamConfig() TeamConfig {
	return TeamConfig{
		TeamCount:   4,
		Aggregate:   TeamAggregateAverage,
		AnswerRule:  TeamAnswerEachMember,
		AutoBalance: true,
	}
}

// DefaultPacedConfig returns student-paced defaults.
func DefaultPacedConfig() PacedConfig {
	return PacedConfig{
		Shuffle:           true,
		TimeBudgetSeconds: 0,
		PerQuestionTimers: true,
		LiveLeaderboard:   false,
	}
}

// NormalizeTeamConfig fills defaults and clamps invalid values.
func NormalizeTeamConfig(c *TeamConfig) TeamConfig {
	out := DefaultTeamConfig()
	if c == nil {
		return out
	}
	if c.TeamCount >= 2 && c.TeamCount <= 20 {
		out.TeamCount = c.TeamCount
	}
	switch c.Aggregate {
	case TeamAggregateSum, TeamAggregateAverage:
		out.Aggregate = c.Aggregate
	}
	switch c.AnswerRule {
	case TeamAnswerEachMember, TeamAnswerOneDevice:
		out.AnswerRule = c.AnswerRule
	}
	out.AutoBalance = c.AutoBalance
	return out
}

// NormalizePacedConfig fills defaults.
func NormalizePacedConfig(c *PacedConfig) PacedConfig {
	out := DefaultPacedConfig()
	if c == nil {
		return out
	}
	out.Shuffle = c.Shuffle
	if c.TimeBudgetSeconds >= 0 && c.TimeBudgetSeconds <= 24*3600 {
		out.TimeBudgetSeconds = c.TimeBudgetSeconds
	}
	out.PerQuestionTimers = c.PerQuestionTimers
	out.LiveLeaderboard = c.LiveLeaderboard
	return out
}

// NormalizeGradePolicy returns best|last|average.
func NormalizeGradePolicy(p string) GradePolicy {
	switch GradePolicy(strings.TrimSpace(p)) {
	case GradePolicyLast, GradePolicyAverage, GradePolicyBest:
		return GradePolicy(strings.TrimSpace(p))
	default:
		return GradePolicyBest
	}
}

// ParseModeSettings decodes sessions.settings JSON for mode configs.
func ParseModeSettings(raw json.RawMessage) ModeSettings {
	var ms ModeSettings
	if len(raw) == 0 {
		return ms
	}
	_ = json.Unmarshal(raw, &ms)
	return ms
}

// MergeModeSettingsInto builds settings JSON with team/paced config for create.
func MergeModeSettingsInto(base json.RawMessage, mode SessionMode, team *TeamConfig, paced *PacedConfig) (json.RawMessage, error) {
	m := map[string]any{}
	if len(base) > 0 && string(base) != "null" {
		if err := json.Unmarshal(base, &m); err != nil {
			return nil, fmt.Errorf("settings: %w", err)
		}
	}
	switch mode {
	case ModeTeam:
		tc := NormalizeTeamConfig(team)
		m["team"] = tc
	case ModeStudentPaced:
		pc := NormalizePacedConfig(paced)
		m["paced"] = pc
	case ModeHomework:
		// homework windows live on assignments; optional paced-like shuffle on settings
		if paced != nil {
			m["paced"] = NormalizePacedConfig(paced)
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// UsesSharedClock reports whether the session uses the classic host clock.
func UsesSharedClock(mode SessionMode) bool {
	return mode == ModeLiveClassic || mode == ModeTeam
}

// GuestsAllowed reports whether guest join is permitted for the mode (IQ.6 FR-11).
func GuestsAllowed(mode SessionMode) bool {
	return mode != ModeHomework
}
