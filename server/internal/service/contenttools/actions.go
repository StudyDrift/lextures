package contenttools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// ActionContext is the server-side context passed to action handlers.
type ActionContext struct {
	Ctx          context.Context
	CourseID     uuid.UUID
	CourseCode   string
	InstanceID   uuid.UUID
	EnrollmentID uuid.UUID
	PrincipalID  uuid.UUID
	ToolID       string
	ConfigJSON   json.RawMessage
	StateJSON    json.RawMessage
	Status       string
	Revision     int64
	Input        json.RawMessage
	// InteractRole is the caller's content-tools interact role (student|instructor|ta).
	InteractRole string

	// Optional AI runtime deps for AI-capable actions (CT.10+).
	Pool         *pgxpool.Pool
	OrgID        *uuid.UUID
	Completer    aiprovider.Completer
	GatewayCfg   aigateway.Config
	Model        string
	ReadingLevel string

	// CodeExecutionEnabled gates CT.17 run/check when set; nil means enabled (unit tests).
	CodeExecutionEnabled *bool

	// SRSPracticeEnabled gates CT.23 SRS writes when set; nil means enabled (unit tests).
	SRSPracticeEnabled *bool
}

// ActionResult is returned by an action handler.
type ActionResult struct {
	Result     map[string]any
	StatePatch json.RawMessage // optional full replacement state document
	Status     string          // optional status advance (submitted/completed)
	ScoreRaw   *float64
	ScoreMax   *float64
}

// ActionHandler executes a named server action for a tool.
type ActionHandler func(ctx ActionContext) (*ActionResult, error)

var (
	actionMu       sync.RWMutex
	actionHandlers = map[string]ActionHandler{}
)

func actionKey(toolID, action string) string {
	return toolID + "::" + action
}

// RegisterActionHandler registers a server action for a tool id.
// Intended for package init of built-in tools.
func RegisterActionHandler(toolID, action string, h ActionHandler) {
	if h == nil {
		return
	}
	actionMu.Lock()
	defer actionMu.Unlock()
	actionHandlers[actionKey(toolID, action)] = h
}

// LookupActionHandler returns a registered handler, or nil.
func LookupActionHandler(toolID, action string) ActionHandler {
	actionMu.RLock()
	defer actionMu.RUnlock()
	return actionHandlers[actionKey(toolID, action)]
}

// DispatchAction resolves the manifest action + handler and runs it.
func DispatchAction(m *CompiledManifest, action string, ctx ActionContext) (*ActionResult, error) {
	if m == nil {
		return nil, ErrToolNotFound
	}
	action = strings.TrimSpace(action)
	decl := FindAction(m, action)
	if decl == nil {
		return nil, ErrActionUnknown
	}
	h := LookupActionHandler(m.ID, action)
	if h == nil {
		return nil, fmt.Errorf("%w: %s/%s has no handler", ErrActionNotAllowed, m.ID, action)
	}
	return h(ctx)
}

// MergeStateJSON shallow-merges patch keys into base (objects only).
func MergeStateJSON(base, patch json.RawMessage) (json.RawMessage, error) {
	baseMap := map[string]any{}
	patchMap := map[string]any{}
	if len(base) > 0 {
		if err := json.Unmarshal(base, &baseMap); err != nil {
			return nil, err
		}
	}
	if len(patch) > 0 {
		if err := json.Unmarshal(patch, &patchMap); err != nil {
			return nil, err
		}
	}
	for k, v := range patchMap {
		baseMap[k] = v
	}
	return json.Marshal(baseMap)
}
