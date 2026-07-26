package contenttools

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Registry is an in-memory tool registry keyed by immutable tool_id (FR-2).
type Registry struct {
	byID map[string]*CompiledManifest
}

// NewRegistry builds a registry from compiled manifests. Duplicate ids fail.
func NewRegistry(manifests []*CompiledManifest) (*Registry, error) {
	r := &Registry{byID: make(map[string]*CompiledManifest, len(manifests))}
	for _, m := range manifests {
		if m == nil {
			return nil, fmt.Errorf("nil manifest")
		}
		if _, exists := r.byID[m.ID]; exists {
			return nil, fmt.Errorf("duplicate tool_id %q", m.ID)
		}
		r.byID[m.ID] = m
	}
	return r, nil
}

// Get returns a compiled manifest by id, or nil.
func (r *Registry) Get(toolID string) *CompiledManifest {
	if r == nil {
		return nil
	}
	return r.byID[toolID]
}

// List returns all manifests sorted by id.
func (r *Registry) List() []*CompiledManifest {
	if r == nil {
		return nil
	}
	out := make([]*CompiledManifest, 0, len(r.byID))
	for _, m := range r.byID {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Size returns the number of registered tools.
func (r *Registry) Size() int {
	if r == nil {
		return 0
	}
	return len(r.byID)
}

var (
	defaultReg  *Registry
	defaultErr  error
	defaultOnce sync.Once
)

// MustDefault returns the process-wide default registry, building it once.
// Startup fails (panic) on malformed tools so the process never serves silently
// with a broken registry (FR-4 / AC-8).
func MustDefault() *Registry {
	defaultOnce.Do(func() {
		start := time.Now()
		reg, err := BuildBuiltinRegistry()
		defaultErr = err
		defaultReg = reg
		if err == nil && reg != nil {
			SetRegistrySizeGauge(float64(reg.Size()))
			_ = start // startup budget asserted in TestRegistryStartupBudget
		}
	})
	if defaultErr != nil {
		panic(fmt.Sprintf("contenttools registry: %v", defaultErr))
	}
	return defaultReg
}
