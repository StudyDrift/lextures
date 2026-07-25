package adaptivecontent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
)

const (
	// EnvKillSwitch is the ops-only emergency kill-switch env var (default disengaged).
	EnvKillSwitch = "ADAPTIVE_CONTENT_KILL_SWITCH"

	EventSettingsUpdated = "settings_updated"
	EventUnitCreated     = "unit_created"
	EventUnitUpdated     = "unit_updated"
	EventUnitDeleted     = "unit_deleted"
	EventFlagToggled     = "flag_toggled"
)

var (
	ErrInvalidAxis           = errors.New("allowedAxes contains an unknown axis; allowed: emphasis, scaffolding, reading_level, misconception, modality.")
	ErrInvalidStrategy       = errors.New("defaultStrategy must be gentle, balanced, or aggressive.")
	ErrHoldoutOutOfRange     = errors.New("holdoutPercent must be between 0 and 50.")
	ErrNegativeBudget        = errors.New("monthlyTokenBudget must be >= 0.")
	ErrInvalidTargetKind     = errors.New("targetKind must be module or outcome.")
	ErrInvalidUnitStatus     = errors.New("status must be draft, active, paused, or archived.")
	ErrTargetShape           = errors.New("target shape must match targetKind: module requires targetModuleItemId; outcome requires targetOutcomeId.")
	ErrBaseContentRequired   = errors.New("baseContentItemId is required.")
	ErrItemNotInCourse       = errors.New("Referenced structure item does not belong to this course.")
	ErrOutcomeNotInCourse    = errors.New("Referenced learning outcome does not belong to this course.")
	ErrKillSwitchEngaged     = errors.New("Adaptive Content Engine is temporarily unavailable (kill-switch engaged).")
	ErrPreAssessmentNotQuiz  = errors.New("preAssessmentItemId must reference a quiz structure item in this course.")
	ErrPostAssessmentNotQuiz = errors.New("postAssessmentItemId must reference a quiz structure item in this course.")
	ErrInvalidFreshnessDays  = errors.New("masteryFreshnessDays must be >= 0.")
)

// AllowedAxes is the closed set of adaptation axes.
var AllowedAxes = map[string]struct{}{
	"emphasis":       {},
	"scaffolding":    {},
	"reading_level":  {},
	"misconception":  {},
	"modality":       {},
}

// AllowedStrategies is the closed set of default strategies.
var AllowedStrategies = map[string]struct{}{
	"gentle":     {},
	"balanced":   {},
	"aggressive": {},
}

// AllowedUnitStatuses is the closed set of unit statuses.
var AllowedUnitStatuses = map[string]struct{}{
	"draft":    {},
	"active":   {},
	"paused":   {},
	"archived": {},
}

// killSwitchOverride allows tests to force the kill-switch without mutating process env.
var (
	killSwitchMu       sync.RWMutex
	killSwitchOverride *bool
)

// SetKillSwitchForTest forces kill-switch state in tests. Pass nil to clear.
func SetKillSwitchForTest(engaged *bool) {
	killSwitchMu.Lock()
	defer killSwitchMu.Unlock()
	killSwitchOverride = engaged
}

// KillSwitchEngaged reports whether the ops emergency kill-switch is on.
// Default is disengaged. True values: 1, true, yes, on (case-insensitive).
func KillSwitchEngaged() bool {
	killSwitchMu.RLock()
	if killSwitchOverride != nil {
		v := *killSwitchOverride
		killSwitchMu.RUnlock()
		return v
	}
	killSwitchMu.RUnlock()

	v := strings.TrimSpace(os.Getenv(EnvKillSwitch))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ActiveForCourse is true when the course flag is on and the kill-switch is disengaged.
// There is no separate required global "enabled" flag.
func ActiveForCourse(courseFlag bool) bool {
	return courseFlag && !KillSwitchEngaged()
}

// ValidateSettings checks settings field bounds and enums.
func ValidateSettings(allowedAxes []string, defaultStrategy string, holdoutPercent int16, monthlyTokenBudget int64) error {
	if holdoutPercent < 0 || holdoutPercent > 50 {
		return ErrHoldoutOutOfRange
	}
	if monthlyTokenBudget < 0 {
		return ErrNegativeBudget
	}
	strategy := strings.TrimSpace(defaultStrategy)
	if strategy == "" {
		strategy = "balanced"
	}
	if _, ok := AllowedStrategies[strategy]; !ok {
		return ErrInvalidStrategy
	}
	for _, a := range allowedAxes {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := AllowedAxes[a]; !ok {
			return ErrInvalidAxis
		}
	}
	return nil
}

// NormalizeAxes returns a de-duplicated, trimmed axis list (empty slice when none).
func NormalizeAxes(axes []string) []string {
	if len(axes) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(axes))
	out := make([]string, 0, len(axes))
	for _, a := range axes {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	return out
}

// ValidateUnitTargetShape checks targetKind vs module/outcome id pairing.
func ValidateUnitTargetShape(targetKind string, moduleItemID, outcomeID *uuid.UUID) error {
	kind := strings.TrimSpace(targetKind)
	switch kind {
	case "module":
		if moduleItemID == nil || *moduleItemID == uuid.Nil {
			return ErrTargetShape
		}
		if outcomeID != nil && *outcomeID != uuid.Nil {
			return ErrTargetShape
		}
	case "outcome":
		if outcomeID == nil || *outcomeID == uuid.Nil {
			return ErrTargetShape
		}
		if moduleItemID != nil && *moduleItemID != uuid.Nil {
			return ErrTargetShape
		}
	default:
		return ErrInvalidTargetKind
	}
	return nil
}

// ValidateUnitStatus checks a unit status enum.
func ValidateUnitStatus(status string) error {
	s := strings.TrimSpace(status)
	if s == "" {
		return nil // default draft
	}
	if _, ok := AllowedUnitStatuses[s]; !ok {
		return ErrInvalidUnitStatus
	}
	return nil
}

// ValidateAxesList validates axes (empty is allowed = inherit course settings).
func ValidateAxesList(axes []string) error {
	for _, a := range axes {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := AllowedAxes[a]; !ok {
			return ErrInvalidAxis
		}
	}
	return nil
}

// ProfileSignature returns a stable hex SHA-256 of the canonical JSON of payload.
// Used as a cache key for (unit, signature) variants (AC.2/AC.3).
func ProfileSignature(payload any) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("profile signature: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
