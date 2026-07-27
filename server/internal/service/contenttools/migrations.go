package contenttools

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// DocMigration is a pure (doc) => doc transform keyed by the source schema version
// it upgrades from (FR-5). Applying fromVersion upgrades the document to fromVersion+1.
type DocMigration func(doc json.RawMessage) (json.RawMessage, error)

// MigrationTable holds ordered state/config migrations for one tool.
type MigrationTable struct {
	ToolID             string
	StateSchemaVersion int // current target
	ConfigSchemaVersion int
	State              map[int]DocMigration // keyed by from-version
	Config             map[int]DocMigration
}

// MigrationRegistry maps tool_id → migration table.
type MigrationRegistry struct {
	mu   sync.RWMutex
	byID map[string]*MigrationTable
}

// NewMigrationRegistry builds an empty registry.
func NewMigrationRegistry() *MigrationRegistry {
	return &MigrationRegistry{byID: map[string]*MigrationTable{}}
}

// Register adds or replaces a tool's migration table.
func (r *MigrationRegistry) Register(t *MigrationTable) {
	if r == nil || t == nil || t.ToolID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if t.State == nil {
		t.State = map[int]DocMigration{}
	}
	if t.Config == nil {
		t.Config = map[int]DocMigration{}
	}
	if t.StateSchemaVersion <= 0 {
		t.StateSchemaVersion = 1
	}
	if t.ConfigSchemaVersion <= 0 {
		t.ConfigSchemaVersion = 1
	}
	r.byID[t.ToolID] = t
}

// Get returns the table for toolID, or nil.
func (r *MigrationRegistry) Get(toolID string) *MigrationTable {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byID[toolID]
}

// CurrentStateSchemaVersion returns the target state schema version for a tool (default 1).
func (r *MigrationRegistry) CurrentStateSchemaVersion(toolID string) int {
	t := r.Get(toolID)
	if t == nil || t.StateSchemaVersion <= 0 {
		return 1
	}
	return t.StateSchemaVersion
}

var (
	defaultMigReg  *MigrationRegistry
	defaultMigOnce sync.Once
)

// DefaultMigrations returns the process-wide migration registry (built-in tools).
func DefaultMigrations() *MigrationRegistry {
	defaultMigOnce.Do(func() {
		defaultMigReg = NewMigrationRegistry()
		registerBuiltinMigrations(defaultMigReg)
	})
	return defaultMigReg
}

func registerBuiltinMigrations(r *MigrationRegistry) {
	// noop_probe stays at state schema v1; a demo chain is registered for unit tests
	// via Register in tests. sandbox_probe also stays at v1.
	r.Register(&MigrationTable{
		ToolID:              "noop_probe",
		StateSchemaVersion:  1,
		ConfigSchemaVersion: 1,
	})
	r.Register(&MigrationTable{
		ToolID:              "sandbox_probe",
		StateSchemaVersion:  1,
		ConfigSchemaVersion: 1,
	})
}

// MigrateResult is the outcome of applying migrations to one document.
type MigrateResult struct {
	Doc          json.RawMessage
	FromVersion  int
	ToVersion    int
	Quarantine   bool
	Error        error
	Unchanged    bool
}

// ApplyStateMigrations lazily upgrades doc from fromVersion toward the table's current version.
// On failure the original doc is returned with Quarantine=true (FR-6 / FR-8 / AC-3 / AC-4).
func ApplyStateMigrations(table *MigrationTable, fromVersion int, doc json.RawMessage) MigrateResult {
	if len(doc) == 0 {
		doc = json.RawMessage(`{}`)
	}
	if fromVersion <= 0 {
		fromVersion = 1
	}
	target := 1
	if table != nil && table.StateSchemaVersion > 0 {
		target = table.StateSchemaVersion
	}
	if fromVersion >= target {
		return MigrateResult{Doc: doc, FromVersion: fromVersion, ToVersion: fromVersion, Unchanged: true}
	}
	cur := append(json.RawMessage(nil), doc...)
	v := fromVersion
	for v < target {
		if table == nil || table.State == nil {
			return MigrateResult{
				Doc: doc, FromVersion: fromVersion, ToVersion: fromVersion,
				Quarantine: true, Error: fmt.Errorf("no migration registered from state schema v%d", v),
			}
		}
		fn, ok := table.State[v]
		if !ok || fn == nil {
			return MigrateResult{
				Doc: doc, FromVersion: fromVersion, ToVersion: fromVersion,
				Quarantine: true, Error: fmt.Errorf("missing state migration from v%d to v%d", v, v+1),
			}
		}
		next, err := fn(cur)
		if err != nil {
			return MigrateResult{
				Doc: doc, FromVersion: fromVersion, ToVersion: fromVersion,
				Quarantine: true, Error: err,
			}
		}
		if len(next) == 0 {
			next = json.RawMessage(`{}`)
		}
		cur = next
		v++
	}
	return MigrateResult{Doc: cur, FromVersion: fromVersion, ToVersion: target}
}

// ApplyConfigMigrations mirrors ApplyStateMigrations for config documents.
func ApplyConfigMigrations(table *MigrationTable, fromVersion int, doc json.RawMessage) MigrateResult {
	if table == nil {
		return MigrateResult{Doc: doc, FromVersion: fromVersion, ToVersion: fromVersion, Unchanged: true}
	}
	// Reuse state applicator with a shallow copy pointing at Config map.
	tmp := *table
	tmp.State = table.Config
	tmp.StateSchemaVersion = table.ConfigSchemaVersion
	return ApplyStateMigrations(&tmp, fromVersion, doc)
}

// MigrationFromVersions returns sorted from-versions present in a map.
func MigrationFromVersions(m map[int]DocMigration) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
