// Effective lockdown delivery (port of server/src/services/quiz_lockdown.rs).
package quizlockdown

import "strings"

// Standard lockdown mode tokens (see course + quiz settings).
const (
	LockdownStandard   = "standard"
	LockdownOneAtATime = "one_at_a_time"
	LockdownKiosk      = "kiosk"
)

// QuizRowLockdown models the per-quiz field used by [EffectiveLockdownMode].
type QuizRowLockdown struct {
	LockdownMode string
}

// ParseLockdownModeSetting returns a known mode token, or false when invalid.
func ParseLockdownModeSetting(raw string) (string, bool) {
	t := strings.TrimSpace(raw)
	switch t {
	case LockdownStandard:
		return LockdownStandard, true
	case LockdownOneAtATime:
		return LockdownOneAtATime, true
	case LockdownKiosk:
		return LockdownKiosk, true
	default:
		return "", false
	}
}
