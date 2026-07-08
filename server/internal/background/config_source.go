package background

import "github.com/lextures/lextures/server/internal/config"

// ConfigSource supplies the current merged platform configuration. Implementations
// must be safe for concurrent use (e.g. platformstate.Platform after Reload).
type ConfigSource interface {
	Config() config.Config
}

// StaticConfigSource wraps a fixed config for tests and one-shot registration.
type StaticConfigSource struct {
	C config.Config
}

func (s StaticConfigSource) Config() config.Config { return s.C }
