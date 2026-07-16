package aiprovider

import (
	"fmt"
	"strings"
)

// RegistryVersion is bumped when alias → provider mappings change meaningfully.
const RegistryVersion = 1

// ModelAlias is a stable, provider-agnostic model identifier (AP.3 FR-1/FR-6).
type ModelAlias string

// Role aliases (preferred for new settings) and feature aliases (FR-2).
const (
	AliasTextFast   ModelAlias = "text-fast"
	AliasTextStrong ModelAlias = "text-strong"
	AliasVision     ModelAlias = "vision"
	AliasImageGen   ModelAlias = "image-gen"

	AliasCourseSetup         ModelAlias = "course-setup"
	AliasNotebookFlashcards  ModelAlias = "notebook-flashcards"
	AliasVibeActivity        ModelAlias = "vibe-activity"
	AliasGraderDefault       ModelAlias = "grader-default"
	AliasTranslation         ModelAlias = "translation"
	AliasTutor               ModelAlias = "tutor"
	AliasStudyBuddy          ModelAlias = "study-buddy"
	AliasSyllabus            ModelAlias = "syllabus"
	AliasLessonPlan          ModelAlias = "lesson-plan"
	AliasAltText             ModelAlias = "alt-text"
	AliasSimplification      ModelAlias = "simplification"
	AliasImageGeneration     ModelAlias = "image-generation"

	// Legacy aliases retained for admin settings / existing rows.
	AliasClaude35Sonnet ModelAlias = "claude-3-5-sonnet"
	AliasGPT4o          ModelAlias = "gpt-4o"
	AliasGemini15Pro    ModelAlias = "gemini-1.5-pro"
)

// AliasInfo is richer alias metadata for admin UIs (AP.3 §9).
type AliasInfo struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Capabilities []string `json:"capabilities"`
}

type modelMapping struct {
	OpenRouter  string
	Anthropic   string
	OpenAI      string
	AzureOpenAI string
	Bedrock     string
	Vertex      string
}

// aliasCanonical maps feature aliases onto role aliases so mappings stay honest and DRY.
var aliasCanonical = map[ModelAlias]ModelAlias{
	AliasCourseSetup:        AliasTextFast,
	AliasNotebookFlashcards: AliasTextFast,
	AliasVibeActivity:       AliasTextFast,
	AliasTranslation:        AliasTextFast,
	AliasStudyBuddy:         AliasTextFast,
	AliasSimplification:     AliasTextFast,
	AliasGraderDefault:      AliasTextStrong,
	AliasTutor:              AliasTextStrong,
	AliasSyllabus:           AliasTextStrong,
	AliasLessonPlan:         AliasTextStrong,
	AliasAltText:            AliasVision,
	AliasImageGeneration:    AliasImageGen,
}

// modelRegistry maps canonical aliases → per-provider model ids.
// Empty string means the alias is not available for that provider (honest mapping; AP.3 §11).
//
// # Registering a new alias
//
//  1. Add a ModelAlias constant and (if feature-scoped) an aliasCanonical entry.
//  2. Add a modelRegistry row with honest ids for every ListProviders() backend.
//  3. If dual-reading legacy OpenRouter ids, extend openRouterIDToAlias.
//  4. Bump RegistryVersion when mappings change for existing aliases.
var modelRegistry = map[ModelAlias]modelMapping{
	AliasTextFast: {
		OpenRouter:  "arcee-ai/trinity-mini:free",
		Anthropic:   "claude-3-5-haiku-20241022",
		OpenAI:      "gpt-4o-mini",
		AzureOpenAI: "gpt-4o-mini",
		Bedrock:     "anthropic.claude-3-5-haiku-20241022-v1:0",
		Vertex:      "gemini-1.5-flash",
	},
	AliasTextStrong: {
		OpenRouter:  "anthropic/claude-3.5-sonnet",
		Anthropic:   "claude-3-5-sonnet-20241022",
		OpenAI:      "gpt-4o",
		AzureOpenAI: "gpt-4o",
		Bedrock:     "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Vertex:      "gemini-1.5-pro",
	},
	AliasVision: {
		OpenRouter:  "openai/gpt-4o",
		Anthropic:   "claude-3-5-sonnet-20241022",
		OpenAI:      "gpt-4o",
		AzureOpenAI: "gpt-4o",
		Bedrock:     "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Vertex:      "gemini-1.5-pro",
	},
	AliasImageGen: {
		OpenRouter:  "black-forest-labs/flux.2-flex",
		Anthropic:   "",
		OpenAI:      "dall-e-3",
		AzureOpenAI: "dall-e-3",
		Bedrock:     "",
		Vertex:      "imagen-3.0-generate-001",
	},
	AliasClaude35Sonnet: {
		OpenRouter:  "anthropic/claude-3.5-sonnet",
		Anthropic:   "claude-3-5-sonnet-20241022",
		OpenAI:      "",
		AzureOpenAI: "",
		Bedrock:     "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Vertex:      "",
	},
	AliasGPT4o: {
		OpenRouter:  "openai/gpt-4o",
		Anthropic:   "",
		OpenAI:      "gpt-4o",
		AzureOpenAI: "gpt-4o",
		Bedrock:     "",
		Vertex:      "",
	},
	AliasGemini15Pro: {
		OpenRouter:  "google/gemini-pro-1.5",
		Anthropic:   "",
		OpenAI:      "",
		AzureOpenAI: "",
		Bedrock:     "",
		Vertex:      "gemini-1.5-pro",
	},
}

var aliasLabels = map[ModelAlias]string{
	AliasTextFast:            "Text (fast)",
	AliasTextStrong:          "Text (strong)",
	AliasVision:              "Vision",
	AliasImageGen:            "Image generation",
	AliasCourseSetup:         "Course setup",
	AliasNotebookFlashcards:  "Notebook flashcards",
	AliasVibeActivity:        "Vibe activity",
	AliasGraderDefault:       "Grader agent",
	AliasTranslation:         "Translation",
	AliasTutor:               "Tutor",
	AliasStudyBuddy:          "Study buddy",
	AliasSyllabus:            "Syllabus",
	AliasLessonPlan:          "Lesson plan",
	AliasAltText:             "Alt-text",
	AliasSimplification:      "Simplification",
	AliasImageGeneration:     "Image generation (feature)",
	AliasClaude35Sonnet:      "Claude 3.5 Sonnet",
	AliasGPT4o:               "GPT-4o",
	AliasGemini15Pro:         "Gemini 1.5 Pro",
}

// openRouterIDToAlias dual-reads legacy OpenRouter model ids stored in user_ai_settings (FR-7).
var openRouterIDToAlias = map[string]ModelAlias{
	"arcee-ai/trinity-mini:free":       AliasTextFast,
	"black-forest-labs/flux.2-flex":    AliasImageGen,
	"anthropic/claude-3.5-sonnet":      AliasTextStrong,
	"openai/gpt-4o":                    AliasVision,
	"openai/gpt-4o-mini":               AliasTextFast,
	"google/gemini-pro-1.5":            AliasGemini15Pro,
}

func canonicalizeAlias(a ModelAlias) ModelAlias {
	if c, ok := aliasCanonical[a]; ok {
		return c
	}
	return a
}

// mappingForProvider returns the mapped id and whether the provider is known.
// An empty id with ok=true means the alias is intentionally unavailable for that provider.
func mappingForProvider(m modelMapping, provider ProviderName) (string, bool) {
	switch provider {
	case ProviderOpenRouter:
		return m.OpenRouter, true
	case ProviderAnthropic:
		return m.Anthropic, true
	case ProviderOpenAI:
		return m.OpenAI, true
	case ProviderAzureOpenAI:
		return m.AzureOpenAI, true
	case ProviderBedrock:
		return m.Bedrock, true
	case ProviderVertex:
		return m.Vertex, true
	case ProviderDryRun:
		return "", true
	default:
		return "", false
	}
}

// ResolveModelID maps an alias (or dual-read OpenRouter id / native id) to a provider-specific model id.
//
// Rules (AP.3 FR-6 / FR-7 / AC-3 / AC-4):
//   - Known alias → provider mapping (error if unavailable for that provider)
//   - Known OpenRouter id + OpenRouter provider → pass through stored id
//   - Known OpenRouter id + other provider → resolve via alias table
//   - Native provider id → pass through
//   - Unknown OpenRouter-shaped id on a non-OpenRouter provider → actionable error
//   - Otherwise → unknown alias error
func ResolveModelID(model string, provider ProviderName) (string, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", fmt.Errorf("aiprovider: empty model")
	}
	if provider == ProviderDryRun {
		return model, nil
	}

	if id, err, ok := resolveAlias(ModelAlias(model), provider); ok {
		return id, err
	}

	if alias, ok := openRouterIDToAlias[model]; ok {
		if provider == ProviderOpenRouter {
			return model, nil
		}
		return resolveAliasRequired(alias, provider)
	}

	if isNativeModelID(model, provider) {
		return model, nil
	}

	if strings.Contains(model, "/") && provider != ProviderOpenRouter {
		return "", fmt.Errorf(
			"aiprovider: model %q looks like an OpenRouter id but has no alias mapping for provider %q; use an alias (e.g. text-fast) or a native model id",
			model, provider,
		)
	}

	return "", fmt.Errorf("aiprovider: unknown model alias %q", model)
}

func resolveAlias(alias ModelAlias, provider ProviderName) (string, error, bool) {
	canon := canonicalizeAlias(alias)
	m, ok := modelRegistry[canon]
	if !ok {
		// Also allow looking up non-canonical keys that are registered directly.
		m, ok = modelRegistry[alias]
		if !ok {
			return "", nil, false
		}
		canon = alias
	}
	if provider == ProviderDryRun {
		return string(alias), nil, true
	}
	id, ok := mappingForProvider(m, provider)
	if !ok {
		return "", fmt.Errorf("aiprovider: unsupported provider %q", provider), true
	}
	if id == "" {
		return "", fmt.Errorf("aiprovider: alias %q is not available for provider %q", canon, provider), true
	}
	return id, nil, true
}

func resolveAliasRequired(alias ModelAlias, provider ProviderName) (string, error) {
	id, err, ok := resolveAlias(alias, provider)
	if !ok {
		return "", fmt.Errorf("aiprovider: unknown model alias %q", alias)
	}
	return id, err
}

// isNativeModelID reports whether model looks like a provider-native id (pass-through).
func isNativeModelID(model string, provider ProviderName) bool {
	switch provider {
	case ProviderOpenRouter:
		return strings.Contains(model, "/")
	case ProviderAnthropic:
		return strings.HasPrefix(model, "claude-")
	case ProviderOpenAI, ProviderAzureOpenAI:
		return strings.HasPrefix(model, "gpt-") ||
			strings.HasPrefix(model, "o1") ||
			strings.HasPrefix(model, "o3") ||
			strings.HasPrefix(model, "o4") ||
			strings.HasPrefix(model, "text-") ||
			strings.HasPrefix(model, "dall-e") ||
			strings.HasPrefix(model, "chatgpt-")
	case ProviderBedrock:
		return strings.Contains(model, ".") && !strings.Contains(model, "/")
	case ProviderVertex:
		return strings.HasPrefix(model, "gemini-") ||
			strings.HasPrefix(model, "claude-") ||
			strings.HasPrefix(model, "imagen-")
	default:
		return false
	}
}

// AliasForOpenRouterID returns the dual-read alias for a known OpenRouter model id, if any.
func AliasForOpenRouterID(modelID string) (ModelAlias, bool) {
	a, ok := openRouterIDToAlias[strings.TrimSpace(modelID)]
	return a, ok
}

// ListModelAliases returns registered stable aliases (including feature aliases).
func ListModelAliases() []string {
	seen := map[ModelAlias]struct{}{}
	out := make([]string, 0, len(modelRegistry)+len(aliasCanonical))
	for alias := range modelRegistry {
		seen[alias] = struct{}{}
		out = append(out, string(alias))
	}
	for alias := range aliasCanonical {
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		out = append(out, string(alias))
	}
	return out
}

// ListModelAliasInfos returns aliases with labels and capability badges for admin UI.
func ListModelAliasInfos() []AliasInfo {
	aliases := ListModelAliases()
	out := make([]AliasInfo, 0, len(aliases))
	for _, id := range aliases {
		a := ModelAlias(id)
		label := aliasLabels[a]
		if label == "" {
			label = id
		}
		out = append(out, AliasInfo{
			ID:           id,
			Label:        label,
			Capabilities: aliasCapabilities(canonicalizeAlias(a)),
		})
	}
	return out
}

func aliasCapabilities(canon ModelAlias) []string {
	switch canon {
	case AliasImageGen:
		return []string{"image"}
	case AliasVision:
		return []string{"text", "vision"}
	case AliasTextFast, AliasTextStrong:
		return []string{"text"}
	case AliasClaude35Sonnet, AliasGPT4o, AliasGemini15Pro:
		return []string{"text", "vision"}
	default:
		return []string{"text"}
	}
}

// ListProviders returns supported provider names for admin UI.
func ListProviders() []ProviderName {
	return []ProviderName{
		ProviderOpenRouter,
		ProviderAnthropic,
		ProviderOpenAI,
		ProviderAzureOpenAI,
		ProviderBedrock,
		ProviderVertex,
	}
}

// NormalizeProviderName parses a provider query/path segment.
func NormalizeProviderName(s string) (ProviderName, bool) {
	p := ProviderName(strings.ToLower(strings.TrimSpace(s)))
	switch p {
	case ProviderOpenRouter, ProviderAnthropic, ProviderOpenAI, ProviderAzureOpenAI, ProviderBedrock, ProviderVertex, ProviderDryRun:
		return p, true
	default:
		return "", false
	}
}
