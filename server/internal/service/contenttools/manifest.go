package contenttools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	toolIDPattern   = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)
	semverPattern   = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
)

// AllowedCategories is the closed set of manifest categories.
var AllowedCategories = map[string]struct{}{
	"assess": {}, "explore": {}, "reflect": {}, "discuss": {}, "practice": {}, "read": {},
}

// AllowedCapabilities is the closed set of capability tokens.
var AllowedCapabilities = map[string]struct{}{
	"state": {}, "scoring": {}, "ai": {}, "network": {}, "media": {}, "realtime": {}, "aggregate": {},
}

// AllowedScoringModes is the closed set of scoring modes.
var AllowedScoringModes = map[string]struct{}{
	"none": {}, "auto": {}, "manual": {}, "external": {},
}

// AllowedInteractRoles is the closed set of interact roles.
var AllowedInteractRoles = map[string]struct{}{
	"student": {}, "instructor": {}, "observer": {},
}

// AllowedHostKinds is the closed set of instance host kinds.
var AllowedHostKinds = map[string]struct{}{
	"content_page": {}, "assignment": {}, "quiz": {}, "syllabus": {}, "portfolio_artifact": {},
}

// AllowedConflictPolicies is the closed set of conflict resolution policies (CT.3).
var AllowedConflictPolicies = map[string]struct{}{
	ConflictServerWins: {}, ConflictClientWins: {}, ConflictMerge: {},
}

// AllowedSandboxModes is the closed set of mount isolation modes (CT.5).
var AllowedSandboxModes = map[string]struct{}{
	SandboxInProcess: {}, SandboxIframe: {},
}

// Manifest is the declarative contract every Content Tool obeys (FR-3).
type Manifest struct {
	ID              string          `json:"id"`
	Version         string          `json:"version"`
	Name            string          `json:"name"`
	Category        string          `json:"category"`
	Capabilities    []string        `json:"capabilities"`
	ConfigSchema    json.RawMessage `json:"configSchema"`
	StateSchema     json.RawMessage `json:"stateSchema"`
	Scoring         ScoringDecl     `json:"scoring"`
	AI              *AIDecl         `json:"ai,omitempty"`
	Network         *NetworkDecl    `json:"network,omitempty"`
	Storage         StorageDecl     `json:"storage"`
	Roles           RolesDecl       `json:"roles"`
	A11y            A11yDecl        `json:"a11y"`
	I18nNamespace   string          `json:"i18nNamespace"`
	UI              UIDecl          `json:"ui"`
	Actions         []ActionDecl    `json:"actions,omitempty"`
	ConflictPolicy  string          `json:"conflictPolicy,omitempty"`
	AutosaveMs      int             `json:"autosaveMs,omitempty"`
	RespectsDueDate bool            `json:"respectsDueDate,omitempty"`
	AllowsSelfReset bool            `json:"allowsSelfReset,omitempty"`
	// CT.5 fields
	Sandbox              string `json:"sandbox,omitempty"` // inprocess | iframe
	Deprecated           bool   `json:"deprecated,omitempty"`
	SunsetAt             string `json:"sunsetAt,omitempty"` // RFC3339 date
	Contract             int    `json:"contract,omitempty"`
	StateSchemaVersion   int    `json:"stateSchemaVersion,omitempty"`
	ConfigSchemaVersion  int    `json:"configSchemaVersion,omitempty"`
	MaxBundleBytesGzip   int    `json:"maxBundleBytesGzip,omitempty"`
}

// ActionDecl declares a server-side action a tool may invoke (CT.3).
type ActionDecl struct {
	Name             string `json:"name"`
	RateLimitPerMin  int    `json:"rateLimitPerMin,omitempty"`
	RequiresAI       bool   `json:"requiresAi,omitempty"`
	Description      string `json:"description,omitempty"`
}

// ScoringDecl is the scoring block of a manifest.
type ScoringDecl struct {
	Mode     string   `json:"mode"`
	MaxScore *float64 `json:"maxScore,omitempty"`
}

// AIDecl declares AI usage.
type AIDecl struct {
	FeatureID string `json:"featureId"`
	Required  bool   `json:"required"`
}

// NetworkDecl declares allowed hosts.
type NetworkDecl struct {
	AllowedHosts []string `json:"allowedHosts"`
}

// StorageDecl declares state size budget.
type StorageDecl struct {
	MaxStateBytes int `json:"maxStateBytes"`
}

// RolesDecl declares interact roles.
type RolesDecl struct {
	Interact []string `json:"interact"`
}

// A11yDecl is mandatory accessibility metadata.
type A11yDecl struct {
	KeyboardOperable bool   `json:"keyboardOperable"`
	SRPattern        string `json:"srPattern"`
	WCAGNotes        string `json:"wcagNotes,omitempty"`
}

// UIDecl is renderer metadata.
type UIDecl struct {
	Renderer string `json:"renderer"`
	Icon     string `json:"icon"`
	Group    string `json:"group"`
}

// CompiledManifest is a validated manifest with compiled JSON Schemas.
type CompiledManifest struct {
	Manifest
	ConfigCompiled *jsonschema.Schema
	StateCompiled  *jsonschema.Schema
	I18nBundle     map[string]string
}

// KnownAIFeatureIDs returns the closed set of aigateway feature ids tools may declare.
// Kept here to avoid an import cycle with aigateway; must stay in sync with Feature* consts.
func KnownAIFeatureIDs() map[string]struct{} {
	return map[string]struct{}{
		"ai_tutor":                       {},
		"modules_ai_assistant":           {},
		"rag_notebook":                   {},
		"syllabus_generation":            {},
		"content_page_generation":        {},
		"outcomes_extraction":            {},
		"quiz_outcome_mapping":           {},
		"badges_extraction":              {},
		"translation":                    {},
		"quiz_generation":                {},
		"assignment_rubric_generation":   {},
		"live_quiz_kit_generation":       {},
		"reading_level_simplification":   {},
		"content_translation":            {},
		"alt_text_suggestion":            {},
		"vibe_generation":                {},
		"grader_agent":                   {},
		"lesson_generation":              {},
		"ai_study_buddy":                 {},
		"report_card_comment":            {},
		"adaptive_content":               {},
		"content_tool":                   {},
	}
}

// ValidateManifest checks the manifest contract (FR-3 / FR-4) without compiling schemas.
func ValidateManifest(m Manifest, i18n map[string]string) error {
	if !toolIDPattern.MatchString(m.ID) {
		return fmt.Errorf("tool id %q must be snake_case (immutable)", m.ID)
	}
	if !semverPattern.MatchString(strings.TrimSpace(m.Version)) {
		return fmt.Errorf("tool %s: version %q is not valid semver", m.ID, m.Version)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("tool %s: name is required", m.ID)
	}
	if _, ok := AllowedCategories[m.Category]; !ok {
		return fmt.Errorf("tool %s: unknown category %q", m.ID, m.Category)
	}
	for _, c := range m.Capabilities {
		if _, ok := AllowedCapabilities[c]; !ok {
			return fmt.Errorf("tool %s: unknown capability %q", m.ID, c)
		}
	}
	if _, ok := AllowedScoringModes[m.Scoring.Mode]; !ok {
		return fmt.Errorf("tool %s: unknown scoring mode %q", m.ID, m.Scoring.Mode)
	}
	if m.AI != nil {
		fid := strings.TrimSpace(m.AI.FeatureID)
		if fid == "" {
			return fmt.Errorf("tool %s: ai.featureId is required when ai is set", m.ID)
		}
		if _, ok := KnownAIFeatureIDs()[fid]; !ok {
			return fmt.Errorf("tool %s: unknown ai.featureId %q", m.ID, fid)
		}
	}
	if m.Storage.MaxStateBytes <= 0 {
		return fmt.Errorf("tool %s: storage.maxStateBytes must be > 0", m.ID)
	}
	if m.Storage.MaxStateBytes > PlatformMaxStateBytes {
		return fmt.Errorf("tool %s: storage.maxStateBytes %d exceeds platform ceiling %d", m.ID, m.Storage.MaxStateBytes, PlatformMaxStateBytes)
	}
	if len(m.Roles.Interact) == 0 {
		return fmt.Errorf("tool %s: roles.interact must be non-empty", m.ID)
	}
	for _, r := range m.Roles.Interact {
		if _, ok := AllowedInteractRoles[r]; !ok {
			return fmt.Errorf("tool %s: unknown interact role %q", m.ID, r)
		}
	}
	if !m.A11y.KeyboardOperable {
		return fmt.Errorf("tool %s: a11y.keyboardOperable must be true", m.ID)
	}
	if strings.TrimSpace(m.A11y.SRPattern) == "" {
		return fmt.Errorf("tool %s: a11y.srPattern is required", m.ID)
	}
	if strings.TrimSpace(m.I18nNamespace) == "" {
		return fmt.Errorf("tool %s: i18nNamespace is required", m.ID)
	}
	if len(i18n) == 0 {
		return fmt.Errorf("tool %s: missing i18n bundle", m.ID)
	}
	if strings.TrimSpace(m.UI.Renderer) == "" || strings.TrimSpace(m.UI.Icon) == "" || strings.TrimSpace(m.UI.Group) == "" {
		return fmt.Errorf("tool %s: ui.renderer, ui.icon, and ui.group are required", m.ID)
	}
	if len(m.ConfigSchema) == 0 {
		return fmt.Errorf("tool %s: configSchema is required", m.ID)
	}
	if len(m.StateSchema) == 0 {
		return fmt.Errorf("tool %s: stateSchema is required", m.ID)
	}
	if m.ConflictPolicy != "" {
		if _, ok := AllowedConflictPolicies[m.ConflictPolicy]; !ok {
			return fmt.Errorf("tool %s: unknown conflictPolicy %q", m.ID, m.ConflictPolicy)
		}
	}
	if m.AutosaveMs != 0 && (m.AutosaveMs < MinAutosaveDebounceMs || m.AutosaveMs > MaxAutosaveDebounceMs) {
		return fmt.Errorf("tool %s: autosaveMs must be between %d and %d", m.ID, MinAutosaveDebounceMs, MaxAutosaveDebounceMs)
	}
	seenActions := map[string]struct{}{}
	for _, a := range m.Actions {
		name := strings.TrimSpace(a.Name)
		if name == "" {
			return fmt.Errorf("tool %s: action name is required", m.ID)
		}
		if !toolIDPattern.MatchString(name) {
			return fmt.Errorf("tool %s: action name %q must be snake_case", m.ID, name)
		}
		if _, dup := seenActions[name]; dup {
			return fmt.Errorf("tool %s: duplicate action %q", m.ID, name)
		}
		seenActions[name] = struct{}{}
		if a.RateLimitPerMin < 0 {
			return fmt.Errorf("tool %s: action %s rateLimitPerMin must be >= 0", m.ID, name)
		}
	}
	if m.Sandbox != "" {
		if _, ok := AllowedSandboxModes[m.Sandbox]; !ok {
			return fmt.Errorf("tool %s: unknown sandbox mode %q", m.ID, m.Sandbox)
		}
	}
	if m.Contract != 0 && !ContractSupported(m.Contract) {
		return fmt.Errorf("tool %s: unsupported contract %d (supported %d–%d)", m.ID, m.Contract, SupportedContractMin, SupportedContractMax)
	}
	if m.StateSchemaVersion < 0 || m.ConfigSchemaVersion < 0 {
		return fmt.Errorf("tool %s: schema versions must be >= 0", m.ID)
	}
	return nil
}

// EffectiveConflictPolicy returns the tool's policy or server_wins default.
func EffectiveConflictPolicy(m *CompiledManifest) string {
	if m == nil || m.ConflictPolicy == "" {
		return ConflictServerWins
	}
	return m.ConflictPolicy
}

// EffectiveAutosaveMs returns the tool override or the host default.
func EffectiveAutosaveMs(m *CompiledManifest) int {
	if m == nil || m.AutosaveMs == 0 {
		return DefaultAutosaveDebounceMs
	}
	return m.AutosaveMs
}

// FindAction returns the declared action or nil.
func FindAction(m *CompiledManifest, name string) *ActionDecl {
	if m == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	for i := range m.Actions {
		if m.Actions[i].Name == name {
			return &m.Actions[i]
		}
	}
	return nil
}

// EffectiveActionRateLimit returns per-minute limit for an action.
func EffectiveActionRateLimit(m *CompiledManifest, a *ActionDecl) int {
	if a != nil && a.RateLimitPerMin > 0 {
		return a.RateLimitPerMin
	}
	if a != nil && a.RequiresAI {
		return DefaultAIActionRateLimitPerMin
	}
	if m != nil && m.AI != nil {
		return DefaultAIActionRateLimitPerMin
	}
	return DefaultActionRateLimitPerMin
}

// CompileManifest validates and compiles JSON Schemas for a manifest.
func CompileManifest(m Manifest, i18n map[string]string) (*CompiledManifest, error) {
	if err := ValidateManifest(m, i18n); err != nil {
		return nil, err
	}
	cfgSchema, err := compileSchema(m.ID+".config", m.ConfigSchema)
	if err != nil {
		return nil, fmt.Errorf("tool %s: invalid configSchema: %w", m.ID, err)
	}
	stateSchema, err := compileSchema(m.ID+".state", m.StateSchema)
	if err != nil {
		return nil, fmt.Errorf("tool %s: invalid stateSchema: %w", m.ID, err)
	}
	return &CompiledManifest{
		Manifest:       m,
		ConfigCompiled: cfgSchema,
		StateCompiled:  stateSchema,
		I18nBundle:     i18n,
	}, nil
}

func compileSchema(name string, raw json.RawMessage) (*jsonschema.Schema, error) {
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(name+".json", doc); err != nil {
		return nil, err
	}
	return c.Compile(name + ".json")
}
