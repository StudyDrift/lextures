// Package scoring implements the versioned, pure IQ.5 scoring function and profile registry.
package scoring

import (
	"encoding/json"
	"math"
)

// Version is the scoring-function version stored on sessions for historical reproducibility (FR-10).
const Version = 1

// Profile identifiers (FR-7).
const (
	ProfileCompetitive = "competitive"
	ProfileFormative   = "formative"
	ProfileCustom      = "custom"
)

// Points styles from kit questions (FR-3).
const (
	StyleStandard = "standard"
	StyleDouble   = "double"
	StyleNoPoints = "no_points"
)

// Power-up kinds (FR-8).
const (
	PowerUpDoubleOrNothing = "double_or_nothing"
	PowerUpShield          = "shield"
)

// Leaderboard privacy modes (FR-11).
const (
	PrivacyNames      = "names"
	PrivacyNicknames  = "nicknames"
	PrivacyHidden     = "hidden"
)

// Config is the persisted scoring_config JSONB (plus resolved profile defaults).
type Config struct {
	Base                int     `json:"base"`
	SpeedWeight         float64 `json:"speedWeight"`
	StreakStep          int     `json:"streakStep"`
	StreakCap           int     `json:"streakCap"`
	PowerUpsEnabled     bool    `json:"powerUpsEnabled"`
	ParticipationPoints int     `json:"participationPoints"` // formative optional: award for any answer
}

// Breakdown is the explainable per-response award (FR-9).
type Breakdown struct {
	Base            int     `json:"base"`
	SpeedBonus      int     `json:"speedBonus"`
	StreakBonus     int     `json:"streakBonus"`
	StyleMultiplier float64 `json:"styleMultiplier"`
	PowerUp         string  `json:"powerUp,omitempty"`
	PowerUpFactor   float64 `json:"powerUpFactor"`
	Total           int     `json:"total"`
}

// Input is everything the pure Score function needs (FR-1).
type Input struct {
	IsCorrect     bool
	ResponseMs    int
	TimeLimitMs   int // deadline window; 0 → no speed bonus
	PointsStyle   string
	QuestionType  string
	StreakBefore  int // consecutive correct answers before this response
	Profile       string
	Config        Config
	PowerUp       string // optional claimed power-up for this question
	ShieldActive  bool   // server-validated: unused shield available
}

// Result is points + breakdown + next streak after applying this response.
type Result struct {
	Points     int
	Breakdown  Breakdown
	StreakAfter int
	ShieldUsed  bool // true when shield consumed to protect streak
}

// DefaultConfig returns built-in defaults for a profile name.
func DefaultConfig(profile string) Config {
	switch profile {
	case ProfileFormative:
		return Config{
			Base:            1000,
			SpeedWeight:     0,
			StreakStep:      0,
			StreakCap:       0,
			PowerUpsEnabled: false,
		}
	case ProfileCustom:
		return Config{
			Base:            1000,
			SpeedWeight:     1,
			StreakStep:      100,
			StreakCap:       5,
			PowerUpsEnabled: false,
		}
	default: // competitive
		return Config{
			Base:            1000,
			SpeedWeight:     1,
			StreakStep:      100,
			StreakCap:       5,
			PowerUpsEnabled: false,
		}
	}
}

// ResolveConfig merges instructor overrides onto profile defaults.
// Unknown profile falls back to competitive. Custom uses overrides with sensible floors.
func ResolveConfig(profile string, overrides Config) Config {
	base := DefaultConfig(profile)
	if profile == ProfileCustom {
		cfg := overrides
		if cfg.Base <= 0 {
			cfg.Base = base.Base
		}
		if cfg.SpeedWeight < 0 {
			cfg.SpeedWeight = 0
		}
		if cfg.StreakStep < 0 {
			cfg.StreakStep = 0
		}
		if cfg.StreakCap < 0 {
			cfg.StreakCap = 0
		}
		if cfg.ParticipationPoints < 0 {
			cfg.ParticipationPoints = 0
		}
		return cfg
	}
	// Built-ins ignore most overrides except powerUps / participation for formative.
	out := base
	out.PowerUpsEnabled = overrides.PowerUpsEnabled
	if profile == ProfileFormative && overrides.ParticipationPoints > 0 {
		out.ParticipationPoints = overrides.ParticipationPoints
	}
	if overrides.Base > 0 && profile == ProfileCompetitive {
		// Allow dialing base on competitive without becoming "custom".
		out.Base = overrides.Base
	}
	return out
}

// NormalizeProfile returns a known profile id.
func NormalizeProfile(p string) string {
	switch p {
	case ProfileFormative, ProfileCustom, ProfileCompetitive:
		return p
	default:
		return ProfileCompetitive
	}
}

// NormalizePrivacy returns a known privacy mode.
func NormalizePrivacy(p string) string {
	switch p {
	case PrivacyNicknames, PrivacyHidden, PrivacyNames:
		return p
	default:
		return PrivacyNames
	}
}

// ParseConfigJSON decodes scoring_config JSONB (empty → zero Config).
func ParseConfigJSON(raw json.RawMessage) Config {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return Config{}
	}
	var c Config
	_ = json.Unmarshal(raw, &c)
	return c
}

// MarshalConfig encodes config for persistence.
func MarshalConfig(c Config) json.RawMessage {
	b, err := json.Marshal(c)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

// Score is the pure, versioned award function (FR-1..FR-4, FR-8, FR-9).
func Score(in Input) Result {
	cfg := in.Config
	if cfg.Base <= 0 {
		cfg.Base = 1000
	}

	// Opinion / collection types never award points and do not grow streaks.
	if in.QuestionType == "poll" || in.QuestionType == "word_cloud" {
		return Result{
			Points:      0,
			Breakdown:   Breakdown{StyleMultiplier: 1, PowerUpFactor: 1, Total: 0},
			StreakAfter: 0,
		}
	}
	// Ungraded style: 0 points, but correctness still advances/resets streak.
	if in.PointsStyle == StyleNoPoints {
		streakAfter := 0
		if in.IsCorrect {
			streakAfter = in.StreakBefore + 1
		}
		return Result{
			Points:      0,
			Breakdown:   Breakdown{StyleMultiplier: 0, PowerUpFactor: 1, Total: 0},
			StreakAfter: streakAfter,
		}
	}

	streakAfter := 0
	shieldUsed := false
	if in.IsCorrect {
		streakAfter = in.StreakBefore + 1
	} else if in.PowerUp == PowerUpShield && in.ShieldActive && in.Config.PowerUpsEnabled {
		// Shield protects streak on a miss (does not award points).
		streakAfter = in.StreakBefore
		shieldUsed = true
	}

	if !in.IsCorrect {
		// Optional formative participation for answering incorrectly.
		part := 0
		if cfg.ParticipationPoints > 0 {
			part = cfg.ParticipationPoints
		}
		bd := Breakdown{
			Base:            part,
			StyleMultiplier: styleMultiplier(in.PointsStyle),
			PowerUpFactor:   1,
			Total:           part,
		}
		if shieldUsed {
			bd.PowerUp = PowerUpShield
		}
		return Result{Points: part, Breakdown: bd, StreakAfter: streakAfter, ShieldUsed: shieldUsed}
	}

	styleMult := styleMultiplier(in.PointsStyle)
	baseAward := cfg.Base

	speedBonus := 0
	if cfg.SpeedWeight > 0 && in.TimeLimitMs > 0 {
		speedFactor := 1.0 - float64(in.ResponseMs)/float64(in.TimeLimitMs)
		if speedFactor < 0 {
			speedFactor = 0
		}
		if speedFactor > 1 {
			speedFactor = 1
		}
		speedBonus = int(math.Round(float64(baseAward) * speedFactor * cfg.SpeedWeight))
	}

	streakBonus := 0
	if cfg.StreakStep > 0 && cfg.StreakCap > 0 {
		// Bonus uses streak count before this answer (first correct → 0).
		steps := in.StreakBefore
		if steps > cfg.StreakCap {
			steps = cfg.StreakCap
		}
		if steps > 0 {
			streakBonus = steps * cfg.StreakStep
		}
	}

	subtotal := baseAward + speedBonus + streakBonus
	total := int(math.Round(float64(subtotal) * styleMult))

	powerUp := ""
	powerFactor := 1.0
	if in.Config.PowerUpsEnabled && in.PowerUp == PowerUpDoubleOrNothing {
		powerUp = PowerUpDoubleOrNothing
		powerFactor = 2
		total = total * 2
	}

	bd := Breakdown{
		Base:            baseAward,
		SpeedBonus:      speedBonus,
		StreakBonus:     streakBonus,
		StyleMultiplier: styleMult,
		PowerUp:         powerUp,
		PowerUpFactor:   powerFactor,
		Total:           total,
	}
	return Result{Points: total, Breakdown: bd, StreakAfter: streakAfter}
}

func styleMultiplier(style string) float64 {
	switch style {
	case StyleDouble:
		return 2
	case StyleNoPoints:
		return 0
	default:
		return 1
	}
}

// Recompute applies Score from stored raw fields (AC-7 reproducibility).
func Recompute(profile string, profileVer int, cfg Config, isCorrect bool, responseMs, timeLimitMs int, pointsStyle, questionType string, streakBefore int, powerUp string, shieldActive bool) Result {
	_ = profileVer // reserved for future formula branches
	resolved := ResolveConfig(NormalizeProfile(profile), cfg)
	return Score(Input{
		IsCorrect:    isCorrect,
		ResponseMs:   responseMs,
		TimeLimitMs:  timeLimitMs,
		PointsStyle:  pointsStyle,
		QuestionType: questionType,
		StreakBefore: streakBefore,
		Profile:      NormalizeProfile(profile),
		Config:       resolved,
		PowerUp:      powerUp,
		ShieldActive: shieldActive,
	})
}
