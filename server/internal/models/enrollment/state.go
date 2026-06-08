package enrollment

import (
	"fmt"
	"strings"
	"time"
)

// State is the HE enrollment lifecycle state (plan 14.3).
type State string

const (
	StateWaitlist   State = "waitlist"
	StateActive     State = "active"
	StateDropped    State = "dropped"
	StateWithdrawn  State = "withdrawn"
	StateAudit      State = "audit"
	StateNoCredit   State = "no_credit"
	StateIncomplete State = "incomplete"
)

// AllStates lists every valid enrollment state.
var AllStates = []State{
	StateWaitlist,
	StateActive,
	StateDropped,
	StateWithdrawn,
	StateAudit,
	StateNoCredit,
	StateIncomplete,
}

// ParseState normalizes and validates a state string.
func ParseState(raw string) (State, error) {
	s := State(strings.ToLower(strings.TrimSpace(raw)))
	switch s {
	case StateWaitlist, StateActive, StateDropped, StateWithdrawn, StateAudit, StateNoCredit, StateIncomplete:
		return s, nil
	default:
		return "", fmt.Errorf("invalid enrollment state %q", raw)
	}
}

// FormerStates are terminal/non-active roster states shown in gradebook "Former Students".
func FormerStates() []State {
	return []State{StateDropped, StateWithdrawn, StateNoCredit}
}

// IsFormer returns true when the student belongs in the gradebook former-students section.
func (s State) IsFormer() bool {
	switch s {
	case StateDropped, StateWithdrawn, StateNoCredit:
		return true
	default:
		return false
	}
}

// SetsActiveEnrollment returns whether the enrollment row should remain `active = true`.
func (s State) SetsActiveEnrollment() bool {
	switch s {
	case StateActive, StateAudit, StateWaitlist, StateIncomplete:
		return true
	default:
		return false
	}
}

// LISStatusCode returns the IMS LIS enrollment status code for grade passback (plan 14.1 / 14.5).
func (s State) LISStatusCode() string {
	switch s {
	case StateActive:
		return "Active"
	case StateDropped:
		return "Dropped"
	case StateWithdrawn:
		return "Withdrawn"
	case StateAudit:
		return "Auditor"
	case StateNoCredit:
		return "NoCredit"
	case StateIncomplete:
		return "Incomplete"
	case StateWaitlist:
		return "Waitlist"
	default:
		return "Active"
	}
}

// DeadlineContext carries term deadlines for transition enforcement.
type DeadlineContext struct {
	AddDropDeadline    *time.Time
	WithdrawalDeadline *time.Time
	OverrideDeadlines  bool
	Now                time.Time
}

// ValidateTransition checks whether a state change is permitted at the given time.
func ValidateTransition(from, to State, dc DeadlineContext) error {
	if from == to {
		return fmt.Errorf("enrollment is already %s", to)
	}
	now := dc.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if !dc.OverrideDeadlines {
		switch to {
		case StateDropped:
			if dc.AddDropDeadline != nil && today.After(*dc.AddDropDeadline) {
				return fmt.Errorf("add/drop deadline was %s; dropped transitions are no longer permitted", dc.AddDropDeadline.Format("January 2, 2006"))
			}
		case StateWithdrawn:
			if dc.AddDropDeadline != nil && today.Before(*dc.AddDropDeadline) {
				return fmt.Errorf("withdrawals are only permitted after the add/drop deadline (%s)", dc.AddDropDeadline.Format("January 2, 2006"))
			}
			if dc.WithdrawalDeadline != nil && today.After(*dc.WithdrawalDeadline) {
				return fmt.Errorf("withdrawal deadline was %s; withdrawn transitions are no longer permitted", dc.WithdrawalDeadline.Format("January 2, 2006"))
			}
		}
	}
	return nil
}
