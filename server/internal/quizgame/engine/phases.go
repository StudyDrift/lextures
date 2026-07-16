// Package engine is a pure, unit-testable state machine for live quiz games (plan IQ.3).
package engine

import "fmt"

// Phase is the authoritative per-question / session phase.
type Phase string

const (
	PhaseLobby          Phase = "lobby"
	PhaseQuestionIntro  Phase = "question_intro"
	PhaseQuestionOpen   Phase = "question_open"
	PhaseQuestionLocked Phase = "question_locked"
	PhaseQuestionReveal Phase = "question_reveal"
	PhaseLeaderboard    Phase = "leaderboard"
	PhasePodium         Phase = "podium"
	PhaseEnded          Phase = "ended"
	PhaseWaitingForHost Phase = "waiting_for_host"
)

// Status is the durable session status (DB enum).
type Status string

const (
	StatusLobby     Status = "lobby"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusEnded     Status = "ended"
	StatusAbandoned Status = "abandoned"
)

// Pacing controls whether the host or timers advance phases.
type Pacing string

const (
	PacingManual Pacing = "manual"
	PacingAuto   Pacing = "auto"
)

// HostAction is a host-driven command.
type HostAction string

const (
	ActionOpen   HostAction = "open"
	ActionLock   HostAction = "lock"
	ActionReveal HostAction = "reveal"
	ActionNext   HostAction = "next"
	ActionSkip   HostAction = "skip"
	ActionPause  HostAction = "pause"
	ActionResume HostAction = "resume"
	ActionEnd    HostAction = "end"
)

// ErrIllegalTransition is returned when a host action is invalid for the current phase.
type ErrIllegalTransition struct {
	From   Phase
	Action HostAction
}

func (e ErrIllegalTransition) Error() string {
	return fmt.Sprintf("illegal transition: %s + %s", e.From, e.Action)
}
