package engine

// Reduce applies an arbitration action to state. actorID is the user performing the action.
// targetID is used for grant/revoke (the user being granted/revoked).
func Reduce(s State, action Action, actorID, targetID string, nowPayload map[string]any) (State, []Event, error) {
	if s.Status == StatusEnded || s.Status == StatusAbandoned {
		return s, nil, ErrIllegalTransition{From: s.Status, Action: action, Reason: "session closed"}
	}

	switch action {
	case ActionRequestPresent:
		return reduceRequest(s, actorID)
	case ActionGrantPresent:
		return reduceGrant(s, actorID, targetID)
	case ActionRevokePresent:
		return reduceRevoke(s, actorID, targetID)
	case ActionSelfPromote:
		return reduceSelfPromote(s, actorID)
	case ActionStopPresent:
		return reduceStop(s, actorID)
	case ActionEndSession:
		return reduceEnd(s, actorID, nowPayload)
	case ActionAbandon:
		return reduceAbandon(s, nowPayload)
	case ActionSetPolicy:
		return reduceSetPolicy(s, actorID, targetID) // targetID carries policy string
	default:
		return s, nil, ErrIllegalTransition{From: s.Status, Action: action}
	}
}

func bump(s State) State {
	s.Seq++
	return s
}

func removePending(pending []string, userID string) []string {
	out := pending[:0:0]
	for _, id := range pending {
		if id != userID {
			out = append(out, id)
		}
	}
	return out
}

func contains(pending []string, userID string) bool {
	for _, id := range pending {
		if id == userID {
			return true
		}
	}
	return false
}

func reduceRequest(s State, actorID string) (State, []Event, error) {
	if actorID == "" {
		return s, nil, ErrIllegalTransition{From: s.Status, Action: ActionRequestPresent, Reason: "missing actor"}
	}
	if s.Policy == PolicyHostOnly {
		return s, nil, ErrIllegalTransition{From: s.Status, Action: ActionRequestPresent, Reason: "host_only"}
	}
	if s.Policy == PolicyFreeForAll {
		// Free-for-all: request is treated as self-promote.
		return reduceSelfPromote(s, actorID)
	}
	if s.ActivePresenterID == actorID {
		return s, nil, nil // already presenting — idempotent
	}
	if contains(s.PendingRequests, actorID) {
		return s, nil, nil // already queued — idempotent
	}
	next := bump(s)
	next.PendingRequests = append(append([]string{}, s.PendingRequests...), actorID)
	return next, []Event{{
		Type:    "present_request",
		ActorID: actorID,
		Payload: map[string]any{"userId": actorID},
	}}, nil
}

func reduceGrant(s State, hostID, targetID string) (State, []Event, error) {
	if targetID == "" {
		return s, nil, ErrIllegalTransition{From: s.Status, Action: ActionGrantPresent, Reason: "missing target"}
	}
	// Idempotent: already presenting as this user.
	if s.ActivePresenterID == targetID {
		next := bump(s)
		next.PendingRequests = removePending(s.PendingRequests, targetID)
		return next, nil, nil
	}
	next := bump(s)
	prev := s.ActivePresenterID
	next.ActivePresenterID = targetID
	next.Status = StatusPresenting
	next.PendingRequests = removePending(s.PendingRequests, targetID)
	evs := []Event{{
		Type:    "present_grant",
		ActorID: hostID,
		Payload: map[string]any{"userId": targetID},
	}}
	if prev != "" && prev != targetID {
		evs = append(evs, Event{
			Type:    "present_change",
			ActorID: hostID,
			Payload: map[string]any{"from": prev, "to": targetID},
		})
	} else {
		evs = append(evs, Event{
			Type:    "present_change",
			ActorID: hostID,
			Payload: map[string]any{"presenterId": targetID},
		})
	}
	return next, evs, nil
}

func reduceRevoke(s State, hostID, targetID string) (State, []Event, error) {
	if targetID == "" {
		targetID = s.ActivePresenterID
	}
	if targetID == "" || s.ActivePresenterID != targetID {
		// Not active — just clear pending if any.
		if contains(s.PendingRequests, targetID) {
			next := bump(s)
			next.PendingRequests = removePending(s.PendingRequests, targetID)
			return next, []Event{{
				Type:    "present_revoke",
				ActorID: hostID,
				Payload: map[string]any{"userId": targetID},
			}}, nil
		}
		return s, nil, nil
	}
	next := bump(s)
	next.ActivePresenterID = ""
	if next.Status == StatusPresenting {
		next.Status = StatusOpen
	}
	return next, []Event{
		{Type: "present_revoke", ActorID: hostID, Payload: map[string]any{"userId": targetID}},
		{Type: "present_change", ActorID: hostID, Payload: map[string]any{"presenterId": nil}},
		{Type: "present_stop", ActorID: hostID, Payload: map[string]any{"userId": targetID, "reason": "revoked"}},
	}, nil
}

func reduceSelfPromote(s State, actorID string) (State, []Event, error) {
	if actorID == "" {
		return s, nil, ErrIllegalTransition{From: s.Status, Action: ActionSelfPromote, Reason: "missing actor"}
	}
	switch s.Policy {
	case PolicyFreeForAll:
		// ok
	case PolicyHostOnly, PolicyRequest:
		return s, nil, ErrIllegalTransition{From: s.Status, Action: ActionSelfPromote, Reason: "policy_denies"}
	default:
		return s, nil, ErrIllegalTransition{From: s.Status, Action: ActionSelfPromote, Reason: "unknown_policy"}
	}
	return reduceGrant(s, actorID, actorID)
}

func reduceStop(s State, actorID string) (State, []Event, error) {
	if s.ActivePresenterID == "" {
		return s, nil, nil
	}
	// Presenter may stop themselves; host may stop anyone (caller enforces host).
	if actorID != "" && actorID != s.ActivePresenterID {
		// Allowed only when host stops — treated as revoke of active.
		return reduceRevoke(s, actorID, s.ActivePresenterID)
	}
	prev := s.ActivePresenterID
	next := bump(s)
	next.ActivePresenterID = ""
	if next.Status == StatusPresenting {
		next.Status = StatusOpen
	}
	return next, []Event{
		{Type: "present_stop", ActorID: actorID, Payload: map[string]any{"userId": prev}},
		{Type: "present_change", ActorID: actorID, Payload: map[string]any{"presenterId": nil}},
	}, nil
}

func reduceEnd(s State, actorID string, payload map[string]any) (State, []Event, error) {
	next := bump(s)
	next.Status = StatusEnded
	next.ActivePresenterID = ""
	next.PendingRequests = nil
	p := map[string]any{}
	for k, v := range payload {
		p[k] = v
	}
	return next, []Event{{Type: "session_end", ActorID: actorID, Payload: p}}, nil
}

func reduceAbandon(s State, payload map[string]any) (State, []Event, error) {
	next := bump(s)
	next.Status = StatusAbandoned
	next.ActivePresenterID = ""
	next.PendingRequests = nil
	p := map[string]any{"reason": "idle"}
	for k, v := range payload {
		p[k] = v
	}
	return next, []Event{{Type: "session_end", ActorID: "", Payload: p}}, nil
}

func reduceSetPolicy(s State, actorID, policy string) (State, []Event, error) {
	p := Policy(policy)
	switch p {
	case PolicyHostOnly, PolicyRequest, PolicyFreeForAll:
	default:
		return s, nil, ErrIllegalTransition{From: s.Status, Action: ActionSetPolicy, Reason: "invalid_policy"}
	}
	if s.Policy == p {
		return s, nil, nil
	}
	next := bump(s)
	next.Policy = p
	return next, []Event{{
		Type:    "policy_change",
		ActorID: actorID,
		Payload: map[string]any{"policy": string(p)},
	}}, nil
}

// CanJoinViewer returns false when the viewer cap would be exceeded.
func CanJoinViewer(s State) bool {
	capN := s.ViewerCap
	if capN <= 0 {
		capN = DefaultViewerCap
	}
	return s.ViewerCount < capN
}
