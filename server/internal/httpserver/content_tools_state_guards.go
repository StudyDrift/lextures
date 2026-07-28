package httpserver

import (
	"encoding/json"

	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func contentToolsGuardStatePut(toolID string, current, next json.RawMessage) (bool, string) {
	if blocked, msg := ctsvc.GuardPredictRevealStatePut(toolID, current); blocked {
		return true, msg
	}
	if blocked, msg := ctsvc.GuardClassPulseStatePut(toolID, current, next); blocked {
		return true, msg
	}
	if blocked, msg := ctsvc.GuardInlineDiscussionStatePut(toolID, current, next); blocked {
		return true, msg
	}
	return false, ""
}
