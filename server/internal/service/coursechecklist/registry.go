package coursechecklist

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Category order is declared here so evaluation/result ordering never depends on map iteration (FR-12).
const (
	CategoryFoundations   CategoryID = "foundations"
	CategoryOrientation   CategoryID = "orientation"
	CategoryStructure     CategoryID = "structure"
	CategoryOutcomes      CategoryID = "outcomes"
	CategoryAssessment    CategoryID = "assessment"
	CategoryFeedback      CategoryID = "feedback"
	CategoryAccessibility CategoryID = "accessibility"
	CategoryLaunch        CategoryID = "launch"
	CategoryReference     CategoryID = "reference" // CC.1 reference rules only
)

// CategoryOrder is the deterministic category display order.
var CategoryOrder = []CategoryID{
	CategoryFoundations,
	CategoryOrientation,
	CategoryStructure,
	CategoryOutcomes,
	CategoryAssessment,
	CategoryFeedback,
	CategoryAccessibility,
	CategoryLaunch,
	CategoryReference,
}

// ItemIDPattern is the stable ID regex (FR-3).
var ItemIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z0-9-]+){1,3}$`)

// ITEM_ID_ALIASES maps retired-or-renamed IDs to their canonical ItemID (FR-4).
//
//nolint:revive // exported name matches the CC.1 plan contract.
var ITEM_ID_ALIASES = map[ItemID]ItemID{}

// RETIRED_ITEM_IDS lists IDs that must resolve to unknown (tombstone) rather than a live rule (FR-4).
//
//nolint:revive // exported name matches the CC.1 plan contract.
var RETIRED_ITEM_IDS = map[ItemID]struct{}{
	ItemCourseSections: {}, // replaced by people.sections (CC.3)
}

// Registry is an ordered checklist rule registry keyed by ItemID (FR-1).
type Registry struct {
	ordered []ItemDescriptor
	byID    map[ItemID]int
}

// NewRegistry builds a registry from descriptors in declared order. Duplicate IDs fail.
func NewRegistry(items []ItemDescriptor) (*Registry, error) {
	r := &Registry{
		ordered: make([]ItemDescriptor, 0, len(items)),
		byID:    make(map[ItemID]int, len(items)),
	}
	for _, it := range items {
		if err := validateDescriptor(it); err != nil {
			return nil, err
		}
		if _, exists := r.byID[it.ID]; exists {
			return nil, fmt.Errorf("duplicate item id %q", it.ID)
		}
		r.byID[it.ID] = len(r.ordered)
		r.ordered = append(r.ordered, it)
	}
	return r, nil
}

func validateDescriptor(it ItemDescriptor) error {
	if !ItemIDPattern.MatchString(string(it.ID)) {
		return fmt.Errorf("item id %q does not match %s", it.ID, ItemIDPattern.String())
	}
	if strings.TrimSpace(it.TitleDefault) == "" {
		return fmt.Errorf("item %q: empty TitleDefault", it.ID)
	}
	if len([]rune(it.TitleDefault)) > 90 {
		return fmt.Errorf("item %q: TitleDefault exceeds 90 chars", it.ID)
	}
	if strings.TrimSpace(it.WhyDefault) == "" {
		return fmt.Errorf("item %q: empty WhyDefault", it.ID)
	}
	if len(it.Sources) == 0 {
		return fmt.Errorf("item %q: Sources required", it.ID)
	}
	if it.Tier != TierEssential && it.Tier != TierRecommended {
		return fmt.Errorf("item %q: invalid tier %q", it.ID, it.Tier)
	}
	if it.Evaluate == nil {
		return fmt.Errorf("item %q: Evaluate required", it.ID)
	}
	if it.TitleKey == "" {
		return fmt.Errorf("item %q: TitleKey required", it.ID)
	}
	if it.WhyKey == "" {
		return fmt.Errorf("item %q: WhyKey required", it.ID)
	}
	return nil
}

// Get returns a descriptor by ID, or nil.
func (r *Registry) Get(id ItemID) *ItemDescriptor {
	if r == nil {
		return nil
	}
	i, ok := r.byID[id]
	if !ok {
		return nil
	}
	it := r.ordered[i]
	return &it
}

// List returns descriptors in declared registry order.
func (r *Registry) List() []ItemDescriptor {
	if r == nil {
		return nil
	}
	out := make([]ItemDescriptor, len(r.ordered))
	copy(out, r.ordered)
	return out
}

// Size returns the number of registered items.
func (r *Registry) Size() int {
	if r == nil {
		return 0
	}
	return len(r.ordered)
}

// ItemsForEvaluate returns the ordered descriptors selected by opt.Only (or all).
func (r *Registry) ItemsForEvaluate(opt EvaluateOptions) []ItemDescriptor {
	if r == nil {
		return nil
	}
	if len(opt.Only) == 0 {
		return r.List()
	}
	seen := make(map[ItemID]struct{}, len(opt.Only))
	var out []ItemDescriptor
	for _, raw := range opt.Only {
		id, ok := r.ResolveItemID(string(raw))
		if !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if it := r.Get(id); it != nil {
			out = append(out, *it)
		}
	}
	// Preserve registry order among selected items.
	sort.SliceStable(out, func(i, j int) bool {
		return r.byID[out[i].ID] < r.byID[out[j].ID]
	})
	return out
}

// ResolveItemID returns the canonical ItemID against this registry, or false for
// unknown/retired IDs (FR-4).
func (r *Registry) ResolveItemID(raw string) (ItemID, bool) {
	id := ItemID(raw)
	if _, retired := RETIRED_ITEM_IDS[id]; retired {
		return "", false
	}
	if canonical, aliased := ITEM_ID_ALIASES[id]; aliased {
		if _, retired := RETIRED_ITEM_IDS[canonical]; retired {
			return "", false
		}
		id = canonical
	}
	if r != nil && r.Get(id) != nil {
		return id, true
	}
	return "", false
}

// ResolveItemID returns the canonical ItemID against the default registry, or
// false for unknown/retired IDs (FR-4).
func ResolveItemID(raw string) (ItemID, bool) {
	return MustDefault().ResolveItemID(raw)
}

// CatalogVersion returns a hash over the sorted set of (ItemID, Tier, Category) (FR-10).
func CatalogVersion() string {
	return catalogVersionFor(Default())
}

func catalogVersionFor(r *Registry) string {
	if r == nil || r.Size() == 0 {
		sum := sha256.Sum256(nil)
		return hex.EncodeToString(sum[:8])
	}
	type trip struct {
		id   string
		tier string
		cat  string
	}
	trips := make([]trip, 0, r.Size())
	for _, it := range r.List() {
		trips = append(trips, trip{id: string(it.ID), tier: string(it.Tier), cat: string(it.Category)})
	}
	sort.Slice(trips, func(i, j int) bool { return trips[i].id < trips[j].id })
	var b strings.Builder
	for _, t := range trips {
		b.WriteString(t.id)
		b.WriteByte('|')
		b.WriteString(t.tier)
		b.WriteByte('|')
		b.WriteString(t.cat)
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

// EngineVersion returns the evaluator contract version (FR-10).
func EngineVersion() int { return EngineVersionConst }

var (
	defaultReg  *Registry
	defaultErr  error
	defaultOnce sync.Once
)

// Default returns the process-wide builtin registry.
func Default() *Registry {
	defaultOnce.Do(func() {
		defaultReg, defaultErr = BuildBuiltinRegistry()
	})
	return defaultReg
}

// MustDefault returns Default or panics if the builtin registry is malformed.
func MustDefault() *Registry {
	r := Default()
	if defaultErr != nil {
		panic(fmt.Sprintf("coursechecklist registry: %v", defaultErr))
	}
	if r == nil {
		panic("coursechecklist registry: nil")
	}
	return r
}
