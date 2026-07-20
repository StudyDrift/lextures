package engine

import "time"

// Status mirrors screenshare.session_status.
type Status string

const (
	StatusOpen       Status = "open"
	StatusPresenting Status = "presenting"
	StatusEnded      Status = "ended"
	StatusAbandoned  Status = "abandoned"
)

// Policy mirrors screenshare.present_policy.
type Policy string

const (
	PolicyHostOnly    Policy = "host_only"
	PolicyRequest     Policy = "request"
	PolicyFreeForAll  Policy = "free_for_all"
)

// Role mirrors screenshare.participant_role.
type Role string

const (
	RoleHost      Role = "host"
	RolePresenter Role = "presenter"
	RoleViewer    Role = "viewer"
	RoleDisplay   Role = "display"
)

// State is the pure in-memory presenter-arbitration state for one session.
type State struct {
	Status            Status
	Policy            Policy
	ActivePresenterID string // empty when none
	PendingRequests   []string
	ViewerCount       int
	ViewerCap         int
	Seq               int
}

// Event is emitted by Reduce for persistence + broadcast.
type Event struct {
	Type    string
	ActorID string
	Payload map[string]any
}

// Action is a host/presenter arbitration command.
type Action string

const (
	ActionRequestPresent Action = "present_request"
	ActionGrantPresent   Action = "present_grant"
	ActionRevokePresent  Action = "present_revoke"
	ActionSelfPromote    Action = "self_promote"
	ActionStopPresent    Action = "present_stop"
	ActionEndSession     Action = "session_end"
	ActionAbandon        Action = "abandon"
	ActionSetPolicy      Action = "set_policy"
)

// ErrIllegalTransition is returned when an action is not valid for the current state.
type ErrIllegalTransition struct {
	From   Status
	Action Action
	Reason string
}

func (e ErrIllegalTransition) Error() string {
	if e.Reason != "" {
		return "screenshare: illegal transition: " + e.Reason
	}
	return "screenshare: illegal transition from " + string(e.From) + " via " + string(e.Action)
}

// DefaultViewerCap is the GA default fan-out bound (FR scalability).
const DefaultViewerCap = 50

// PresenterGrace is how long a disconnected presenter may reconnect before auto-stop.
const PresenterGrace = 45 * time.Second

// IdleMaxAge abandons open sessions with no activity.
const IdleMaxAge = 4 * time.Hour
