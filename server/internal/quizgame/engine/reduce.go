package engine

import "time"

// HostGraceDefault is the recommended host disconnect grace window (open question 2).
const HostGraceDefault = 90 * time.Second

// Reduce applies a host action to state using the server clock `now`.
// It returns the next state and zero or more events to persist/broadcast.
func Reduce(s State, action HostAction, now time.Time) (State, []Event, error) {
	if s.Status == StatusEnded || s.Status == StatusAbandoned || s.Phase == PhaseEnded {
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: action}
	}

	switch action {
	case ActionOpen:
		return reduceOpen(s, now)
	case ActionLock:
		return reduceLock(s, now)
	case ActionReveal:
		return reduceReveal(s, now)
	case ActionNext:
		return reduceNext(s, now)
	case ActionSkip:
		return reduceSkip(s, now)
	case ActionPause:
		return reducePause(s, now)
	case ActionResume:
		return reduceResume(s, now)
	case ActionEnd:
		return reduceEnd(s, now)
	default:
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: action}
	}
}

// ReduceDeadlineLock auto-locks when the server clock passes the deadline.
func ReduceDeadlineLock(s State, now time.Time) (State, []Event, error) {
	if s.Phase != PhaseQuestionOpen || s.Deadline == nil {
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionLock}
	}
	if now.Before(*s.Deadline) {
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionLock}
	}
	return reduceLock(s, now)
}

// ReduceHostDisconnect marks the game paused / waiting for host.
func ReduceHostDisconnect(s State, now time.Time) (State, []Event) {
	if s.Status == StatusEnded || s.Status == StatusAbandoned {
		return s, nil
	}
	next := s
	next.ResumePhase = s.Phase
	if next.ResumePhase == PhaseWaitingForHost {
		next.ResumePhase = PhaseLobby
	}
	next.Phase = PhaseWaitingForHost
	next.Status = StatusPaused
	next.HostPaused = true
	return next, []Event{{
		Type: "host_disconnect",
		Payload: map[string]any{
			"at":          now.UTC().Format(time.RFC3339Nano),
			"resumePhase": string(next.ResumePhase),
		},
	}}
}

// ReduceHostReconnect restores the prior phase if still within grace.
func ReduceHostReconnect(s State, disconnectedAt *time.Time, now time.Time, grace time.Duration) (State, []Event, bool) {
	if !s.HostPaused && s.Phase != PhaseWaitingForHost {
		// Already active — no-op resume.
		return s, nil, true
	}
	if disconnectedAt != nil && grace > 0 && now.Sub(*disconnectedAt) > grace {
		ended, ev, _ := reduceEnd(s, now)
		ended.Status = StatusAbandoned
		ev = append(ev, Event{Type: "abandoned", Payload: map[string]any{"reason": "host_grace_expired"}})
		return ended, ev, false
	}
	next := s
	resume := s.ResumePhase
	if resume == "" || resume == PhaseWaitingForHost {
		resume = PhaseLobby
	}
	next.Phase = resume
	next.HostPaused = false
	if next.Phase == PhaseLobby {
		next.Status = StatusLobby
	} else {
		next.Status = StatusRunning
	}
	return next, []Event{{
		Type: "host_reconnect",
		Payload: map[string]any{
			"at":    now.UTC().Format(time.RFC3339Nano),
			"phase": string(next.Phase),
		},
	}}, true
}

func reduceOpen(s State, now time.Time) (State, []Event, error) {
	switch s.Phase {
	case PhaseLobby, PhaseQuestionIntro:
		// ok
	default:
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionOpen}
	}

	idx := s.QuestionIndex
	if s.Phase == PhaseLobby {
		if s.QuestionCount <= 0 {
			return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionOpen}
		}
		idx = 0
	}
	if idx < 0 || idx >= s.QuestionCount {
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionOpen}
	}

	next := s
	next.QuestionIndex = idx
	next.Phase = PhaseQuestionOpen
	next.Status = StatusRunning
	opened := now.UTC()
	next.OpenedAt = &opened
	// Deadline filled by caller with question time limit via ApplyDeadline.
	next.Deadline = nil
	next.HostPaused = false

	return next, []Event{{
		Type: "question_open",
		Payload: map[string]any{
			"questionIndex": idx,
			"openedAt":      opened.Format(time.RFC3339Nano),
		},
	}}, nil
}

// ApplyDeadline sets deadline = openedAt + limit (server clock).
func ApplyDeadline(s State, timeLimitSeconds int) State {
	if s.OpenedAt == nil || timeLimitSeconds <= 0 {
		return s
	}
	d := s.OpenedAt.Add(time.Duration(timeLimitSeconds) * time.Second)
	s.Deadline = &d
	return s
}

func reduceLock(s State, now time.Time) (State, []Event, error) {
	if s.Phase != PhaseQuestionOpen {
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionLock}
	}
	next := s
	next.Phase = PhaseQuestionLocked
	return next, []Event{{
		Type: "question_lock",
		Payload: map[string]any{
			"questionIndex": s.QuestionIndex,
			"at":            now.UTC().Format(time.RFC3339Nano),
		},
	}}, nil
}

func reduceReveal(s State, now time.Time) (State, []Event, error) {
	if s.Phase != PhaseQuestionLocked && s.Phase != PhaseQuestionOpen {
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionReveal}
	}
	next := s
	// Auto-lock if revealing from open.
	if next.Phase == PhaseQuestionOpen {
		next.Phase = PhaseQuestionLocked
	}
	next.Phase = PhaseQuestionReveal
	return next, []Event{{
		Type: "question_reveal",
		Payload: map[string]any{
			"questionIndex": s.QuestionIndex,
			"at":            now.UTC().Format(time.RFC3339Nano),
		},
	}}, nil
}

func reduceNext(s State, now time.Time) (State, []Event, error) {
	switch s.Phase {
	case PhaseLobby:
		return reduceOpen(s, now)
	case PhaseQuestionReveal, PhaseLeaderboard, PhaseQuestionLocked:
		// advance
	default:
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionNext}
	}

	nextIdx := s.QuestionIndex + 1
	if s.Phase == PhaseLobby {
		return reduceOpen(s, now)
	}
	if nextIdx >= s.QuestionCount {
		return reducePodium(s, now)
	}
	next := s
	next.QuestionIndex = nextIdx
	next.Phase = PhaseQuestionIntro
	next.OpenedAt = nil
	next.Deadline = nil
	ev := []Event{{
		Type: "question_intro",
		Payload: map[string]any{
			"questionIndex": nextIdx,
		},
	}}
	// Auto-open immediately for smoother UX (intro is instantaneous unless pacing needs pause).
	opened, openEv, err := reduceOpen(next, now)
	if err != nil {
		return next, ev, nil
	}
	return opened, append(ev, openEv...), nil
}

func reduceSkip(s State, now time.Time) (State, []Event, error) {
	switch s.Phase {
	case PhaseQuestionOpen, PhaseQuestionLocked, PhaseQuestionReveal, PhaseQuestionIntro, PhaseLeaderboard:
		next := s
		next.Phase = PhaseQuestionReveal // treat as revealed/skipped
		skipped, ev, err := reduceNext(next, now)
		if err != nil {
			return s, nil, err
		}
		ev = append([]Event{{
			Type: "question_skip",
			Payload: map[string]any{"questionIndex": s.QuestionIndex},
		}}, ev...)
		return skipped, ev, nil
	default:
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionSkip}
	}
}

func reducePause(s State, now time.Time) (State, []Event, error) {
	if s.Phase == PhaseEnded || s.Phase == PhasePodium || s.Phase == PhaseWaitingForHost {
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionPause}
	}
	next, ev := ReduceHostDisconnect(s, now)
	return next, ev, nil
}

func reduceResume(s State, now time.Time) (State, []Event, error) {
	next, ev, ok := ReduceHostReconnect(s, nil, now, 0)
	if !ok {
		return s, nil, ErrIllegalTransition{From: s.Phase, Action: ActionResume}
	}
	return next, ev, nil
}

func reducePodium(s State, now time.Time) (State, []Event, error) {
	next := s
	next.Phase = PhasePodium
	next.OpenedAt = nil
	next.Deadline = nil
	return next, []Event{{
		Type: "podium",
		Payload: map[string]any{
			"at": now.UTC().Format(time.RFC3339Nano),
		},
	}}, nil
}

func reduceEnd(s State, now time.Time) (State, []Event, error) {
	next := s
	next.Phase = PhaseEnded
	next.Status = StatusEnded
	next.OpenedAt = nil
	next.Deadline = nil
	next.HostPaused = false
	return next, []Event{{
		Type: "ended",
		Payload: map[string]any{
			"at": now.UTC().Format(time.RFC3339Nano),
		},
	}}, nil
}

// ResponseTiming computes server-side response_ms and late flag.
func ResponseTiming(openedAt time.Time, deadline *time.Time, receivedAt time.Time) (responseMs int, late bool) {
	if receivedAt.Before(openedAt) {
		receivedAt = openedAt
	}
	ms := int(receivedAt.Sub(openedAt) / time.Millisecond)
	if ms < 0 {
		ms = 0
	}
	if deadline != nil && receivedAt.After(*deadline) {
		return ms, true
	}
	return ms, false
}
