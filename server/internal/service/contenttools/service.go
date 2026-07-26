package contenttools

import (
	"errors"
	"os"
	"strings"
)

const (
	// EnvKillSwitch is the ops-only emergency kill-switch env var (default disengaged).
	EnvKillSwitch = "CONTENT_TOOLS_KILL_SWITCH"

	// PlatformMaxStateBytes is the hard ceiling for any tool's storage.maxStateBytes.
	PlatformMaxStateBytes = 256 * 1024

	// DefaultMaxConfigBytes matches the DB CHECK on config_json.
	DefaultMaxConfigBytes = 256 * 1024

	// DefaultMaxStateBytes matches the DB CHECK on state_json.
	DefaultMaxStateBytes = 64 * 1024

	EventFlagToggled        = "flag_toggled"
	EventSettingsUpdated    = "settings_updated"
	EventInstanceCreated    = "instance_created"
	EventInstanceUpdated    = "instance_updated"
	EventInstanceArchived   = "instance_archived"
	EventInstanceDuplicated = "instance_duplicated"
	EventInstanceDeleted    = "instance_deleted"
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
