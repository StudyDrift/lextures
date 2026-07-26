package contenttools

import (
	"encoding/json"
	"fmt"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools"
)

// BuildBuiltinRegistry loads every first-party tool from the tools package
// (generated index) and validates the contract (FR-4).
func BuildBuiltinRegistry() (*Registry, error) {
	entries := tools.All()
	compiled := make([]*CompiledManifest, 0, len(entries))
	for _, e := range entries {
		var m Manifest
		if err := json.Unmarshal(e.ManifestJSON, &m); err != nil {
			return nil, fmt.Errorf("tool %s: parse manifest: %w", e.ID, err)
		}
		if m.ID == "" {
			m.ID = e.ID
		}
		if m.ID != e.ID {
			return nil, fmt.Errorf("tool folder %s: manifest id %q mismatch", e.ID, m.ID)
		}
		cm, err := CompileManifest(m, e.I18nEN)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, cm)
	}
	return NewRegistry(compiled)
}
