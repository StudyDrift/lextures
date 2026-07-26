package contenttools

import (
	"errors"
	"os"
	"strings"
)

const (
	// EnvKillSwitch is the ops-only emergency kill-switch env var (default disengaged).
	EnvKillSwitch = "CONTENT_TOOLS_KILL_SWITCH"

	// EnvRuntimeReadonly forces every tool read-only during an incident (CT.3).
	EnvRuntimeReadonly = "CONTENT_TOOLS_RUNTIME_READONLY"

	// PlatformMaxStateBytes is the hard ceiling for any tool's storage.maxStateBytes.
	PlatformMaxStateBytes = 256 * 1024

	// DefaultMaxConfigBytes matches the DB CHECK on config_json.
	DefaultMaxConfigBytes = 256 * 1024

	// DefaultMaxStateBytes matches the DB CHECK on state_json.
	DefaultMaxStateBytes = 64 * 1024

	// DefaultAutosaveDebounceMs is the host default when a tool does not override.
	DefaultAutosaveDebounceMs = 1500
	MinAutosaveDebounceMs     = 500
	MaxAutosaveDebounceMs     = 10000

	// DefaultActionRateLimitPerMin is used when an action omits rateLimitPerMin.
	DefaultActionRateLimitPerMin = 20
	// DefaultAIActionRateLimitPerMin is the default for AI-backed actions.
	DefaultAIActionRateLimitPerMin = 10
	// StateWriteRateLimitPerMin is the per-user/instance state write budget.
	StateWriteRateLimitPerMin = 120

	EventFlagToggled        = "flag_toggled"
	EventSettingsUpdated    = "settings_updated"
	EventInstanceCreated    = "instance_created"
	EventInstanceUpdated    = "instance_updated"
	EventInstanceArchived   = "instance_archived"
	EventInstanceDuplicated = "instance_duplicated"
	EventInstanceDeleted    = "instance_deleted"
	EventStateSaved         = "state_saved"
	EventStateSubmitted     = "state_submitted"
	EventActionRan          = "action_ran"
	EventStateReset         = "state_reset"
	EventStateResetRestored = "state_reset_restored"
)

// Learner state status values (server-enforced transitions).
const (
	StatusNotStarted = "not_started"
	StatusInProgress = "in_progress"
	StatusSubmitted  = "submitted"
	StatusCompleted  = "completed"
)

// Conflict policy values (client-applied on 409).
const (
	ConflictServerWins = "server_wins"
	ConflictClientWins = "client_wins"
	ConflictMerge      = "merge"
)

var (
	ErrKillSwitchEngaged      = errors.New("Content Tools are temporarily unavailable.")
	ErrFeatureDisabled        = errors.New("Content Tools are not enabled for this course.")
	ErrToolNotFound           = errors.New("tool not found")
	ErrToolNotAllowed         = errors.New("tool is not on the course allowlist")
	ErrInvalidHostKind        = errors.New("hostKind must be content_page, assignment, quiz, syllabus, or portfolio_artifact")
	ErrInvalidStatus          = errors.New("status must be active or archived")
	ErrStructureItemRequired  = errors.New("structureItemId is required for this hostKind")
	ErrStructureItemForbidden = errors.New("structureItemId must be null for syllabus hostKind")
	ErrItemNotInCourse        = errors.New("Referenced structure item does not belong to this course.")
	ErrConfigTooLarge         = errors.New("config payload exceeds size limit")
	ErrStateTooLarge          = errors.New("state payload exceeds size limit")
	ErrMaxInstances           = errors.New("maximum instances per item exceeded")
	ErrRuntimeReadonly        = errors.New("Content Tools runtime is temporarily read-only.")
	ErrInvalidStateStatus     = errors.New("invalid state status transition")
	ErrActionUnknown          = errors.New("unknown action")
	ErrActionNotAllowed       = errors.New("action not allowed for this tool")
)

// KillSwitchEngaged reports whether the ops emergency kill-switch is on.
func KillSwitchEngaged() bool {
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

// AvailableForCourse is true when Content Tools endpoints should respond (not 404).
// Kill-switch and flag-off both yield unavailable (HTTP 404 per FR-14 / rollout plan).
func AvailableForCourse(courseFlag bool) bool {
	return ActiveForCourse(courseFlag)
}

// RuntimeReadonly reports whether CONTENT_TOOLS_RUNTIME_READONLY is engaged.
func RuntimeReadonly() bool {
	v := strings.TrimSpace(os.Getenv(EnvRuntimeReadonly))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// CanTransitionStateStatus returns true when from→to is a forward-only transition
// (backwards only via CT.4 reset). Empty to means "no change requested".
func CanTransitionStateStatus(from, to string) bool {
	if to == "" || to == from {
		return true
	}
	order := map[string]int{
		StatusNotStarted: 0,
		StatusInProgress: 1,
		StatusSubmitted:  2,
		StatusCompleted:  3,
	}
	a, okA := order[from]
	b, okB := order[to]
	if !okA || !okB {
		return false
	}
	return b >= a
}

// NextStatusOnSave advances not_started → in_progress; otherwise keeps current.
func NextStatusOnSave(current string, requested string) (string, error) {
	cur := current
	if cur == "" {
		cur = StatusNotStarted
	}
	if requested == "" {
		if cur == StatusNotStarted {
			return StatusInProgress, nil
		}
		return cur, nil
	}
	if requested != StatusInProgress && requested != StatusSubmitted {
		return "", ErrInvalidStateStatus
	}
	if !CanTransitionStateStatus(cur, requested) {
		return "", ErrInvalidStateStatus
	}
	return requested, nil
}
