package aiprovider

import (
	"context"

	"github.com/google/uuid"
)

// ToolSpec describes a model-callable tool (CT.6 FR-9).
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema object
}

// ToolCall is one tool invocation requested by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON object
}

// ToolHandler executes a tool call and returns a string result for the model.
type ToolHandler func(call ToolCall) (string, error)

// ToolCallingCompleter is an optional Completer capability for providers that
// support tool/function calling. Providers lacking it use CT.6 orchestrated fallback.
type ToolCallingCompleter interface {
	CompleteWithTools(
		ctx context.Context,
		orgID *uuid.UUID,
		modelOverride string,
		messages []Message,
		tools []ToolSpec,
		handler ToolHandler,
		opts ...ChatOptions,
	) (ChatResult, CallMeta, error)
}

// SupportsToolCalling reports whether c implements ToolCallingCompleter.
func SupportsToolCalling(c Completer) bool {
	_, ok := c.(ToolCallingCompleter)
	return ok
}
