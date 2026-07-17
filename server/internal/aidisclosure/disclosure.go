package aidisclosure

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

const (
	retentionDays = 30
	dpaStatus     = "sub_processor_under_institutional_DPA"
	optOutPath    = "/settings/account"
	cacheTTL      = time.Hour
)

// FeatureCard describes an AI-powered product feature for the public disclosure page.
type FeatureCard struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ModelCard is one row in the public AI disclosure document.
type ModelCard struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Provider      string   `json:"provider"`
	Purposes      []string `json:"purposes"`
	DataSent      string   `json:"dataSent"`
	RetentionDays int      `json:"retentionDays"`
	DPAStatus     string   `json:"dpaStatus"`
	OptOutPath    string   `json:"optOutPath"`
}

// PublicDisclosure is the machine-readable AI disclosure payload (plan 10.17 / AP.6).
type PublicDisclosure struct {
	Version    string        `json:"version"`
	Provider   string        `json:"provider"`             // primary / first configured (compat)
	Providers  []string      `json:"providers"`            // configured backends (AP.6 FR-4)
	Models     []ModelCard   `json:"models"`
	Features   []FeatureCard `json:"features"`
}

// DisclosureOptions customizes public disclosure assembly (AP.6).
type DisclosureOptions struct {
	// ConfiguredProviders are backends with usable credentials (platform and/or org).
	ConfiguredProviders []string
}

type modelBinding struct {
	modelID  string
	alias    aiprovider.ModelAlias
	purposes []string
	dataSent string
}

var disclosureFeatures = []FeatureCard{
	{Key: "ai_tutor", Label: "AI Tutor", Description: "Conversational tutoring within enrolled courses."},
	{Key: "rag_notebook", Label: "Notebook AI", Description: "Answers questions using your course notebook content."},
	{Key: "syllabus_generation", Label: "Syllabus generation", Description: "Instructor tool to draft syllabus sections."},
	{Key: "translation", Label: "Translation", Description: "Translates user-selected text via an AI model."},
	{Key: "content_translation", Label: "Course content translation", Description: "Translates course materials for multilingual learners."},
	{Key: "reading_level_simplification", Label: "Reading level simplification", Description: "Rewrites content to a simpler reading level."},
	{Key: "quiz_generation", Label: "Adaptive quiz generation", Description: "Generates quiz items from course materials."},
	{Key: "live_quiz_kit_generation", Label: "Live quiz kit generation", Description: "Drafts timed live-quiz questions from a topic, passage, or course content for instructor review."},
	{Key: "vibe_generation", Label: "Interactive activity generation", Description: "Drafts self-contained HTML learning activities."},
	{Key: "grader_agent", Label: "Grading agent", Description: "Suggests scores and feedback on student submissions."},
	{Key: "lesson_generation", Label: "Lesson generator", Description: "Generates lesson plans, activities, and formative assessments from learning objectives."},
	{Key: "ai_study_buddy", Label: "AI study buddy", Description: "Standalone study companion for self-learners."},
	{Key: "alt_text_suggestion", Label: "Alt-text suggestions", Description: "Suggests accessible image descriptions for course media."},
}

// platformModelBindings lists default models and the features that use them.
// User- and org-specific overrides may select other models from the platform catalog.
var platformModelBindings = []modelBinding{
	{
		modelID: user.DefaultCourseSetupModelID,
		alias:   aiprovider.AliasCourseSetup,
		purposes: []string{
			"ai_tutor", "rag_notebook", "syllabus_generation", "quiz_generation", "live_quiz_kit_generation", "lesson_generation", "ai_study_buddy",
		},
		dataSent: "Course context, prompts, and user questions necessary for the feature; PII is redacted where configured.",
	},
	{
		modelID:  user.DefaultVibeActivityModelID,
		alias:    aiprovider.AliasVibeActivity,
		purposes: []string{"vibe_generation"},
		dataSent: "Activity prompts and course context needed to draft the interactive activity.",
	},
	{
		modelID:  user.DefaultGraderAgentModelID,
		alias:    aiprovider.AliasGraderDefault,
		purposes: []string{"grader_agent"},
		dataSent: "Assignment prompts, rubrics, and student submission text for grading suggestions.",
	},
	{
		modelID:  "openai/gpt-4o-mini",
		alias:    aiprovider.AliasTranslation,
		purposes: []string{"translation", "content_translation", "reading_level_simplification"},
		dataSent: "Text submitted for translation or reading-level adjustment only.",
	},
	{
		modelID:  "openai/gpt-4o",
		alias:    aiprovider.AliasAltText,
		purposes: []string{"alt_text_suggestion"},
		dataSent: "Image references and surrounding context for alt-text generation.",
	},
}

type disclosureCache struct {
	mu      sync.RWMutex
	expires time.Time
	key     string
	payload []byte
}

var publicDisclosureCache disclosureCache

var fetchModelNames = fetchOpenRouterModelNames

// BuildPublicDisclosure assembles the disclosure document using configured providers + model metadata.
func BuildPublicDisclosure(ctx context.Context, opts DisclosureOptions) (PublicDisclosure, error) {
	names, err := fetchModelNames(ctx)
	if err != nil {
		names = map[string]string{}
	}
	return assembleDisclosure(names, opts.ConfiguredProviders), nil
}

// PublicDisclosureJSON returns cached JSON for GET /api/v1/public/ai-disclosure.
func PublicDisclosureJSON(ctx context.Context, opts DisclosureOptions) ([]byte, error) {
	cacheKey := strings.Join(normalizeProviders(opts.ConfiguredProviders), ",")
	publicDisclosureCache.mu.RLock()
	if len(publicDisclosureCache.payload) > 0 &&
		publicDisclosureCache.key == cacheKey &&
		time.Now().Before(publicDisclosureCache.expires) {
		out := publicDisclosureCache.payload
		publicDisclosureCache.mu.RUnlock()
		return out, nil
	}
	publicDisclosureCache.mu.RUnlock()

	doc, err := BuildPublicDisclosure(ctx, opts)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("aidisclosure: marshal: %w", err)
	}

	publicDisclosureCache.mu.Lock()
	publicDisclosureCache.payload = raw
	publicDisclosureCache.key = cacheKey
	publicDisclosureCache.expires = time.Now().Add(cacheTTL)
	publicDisclosureCache.mu.Unlock()
	return raw, nil
}

// InvalidatePublicDisclosureCache clears the in-memory disclosure cache (for tests).
func InvalidatePublicDisclosureCache() {
	publicDisclosureCache.mu.Lock()
	publicDisclosureCache.payload = nil
	publicDisclosureCache.key = ""
	publicDisclosureCache.expires = time.Time{}
	publicDisclosureCache.mu.Unlock()
}

func assembleDisclosure(names map[string]string, configured []string) PublicDisclosure {
	providers := normalizeProviders(configured)
	viaOpenRouter := containsProvider(providers, string(aiprovider.ProviderOpenRouter))
	routing := primaryRoutingProvider(providers)

	byID := map[string]*ModelCard{}
	order := make([]string, 0, len(platformModelBindings))

	for _, bind := range platformModelBindings {
		id, providerLabel := resolveDisclosureModel(bind, routing, viaOpenRouter)
		if id == "" {
			continue
		}
		card, ok := byID[id]
		if !ok {
			card = &ModelCard{
				ID:            id,
				Name:          modelDisplayName(id, names),
				Provider:      providerLabel,
				Purposes:      []string{},
				DataSent:      bind.dataSent,
				RetentionDays: retentionDays,
				DPAStatus:     dpaStatus,
				OptOutPath:    optOutPath,
			}
			byID[id] = card
			order = append(order, id)
		}
		card.Purposes = appendUnique(card.Purposes, bind.purposes...)
		if len(bind.dataSent) > len(card.DataSent) {
			card.DataSent = bind.dataSent
		}
	}

	models := make([]ModelCard, 0, len(order))
	for _, id := range order {
		models = append(models, *byID[id])
	}

	primary := ""
	if len(providers) > 0 {
		primary = providers[0]
	}

	return PublicDisclosure{
		Version:   time.Now().UTC().Format("2006-01-02"),
		Provider:  primary,
		Providers: providers,
		Models:    models,
		Features:  append([]FeatureCard(nil), disclosureFeatures...),
	}
}

func resolveDisclosureModel(bind modelBinding, routing aiprovider.ProviderName, viaOpenRouter bool) (modelID, providerLabel string) {
	fallbackID := strings.TrimSpace(bind.modelID)
	if routing != "" && routing != aiprovider.ProviderOpenRouter && bind.alias != "" {
		if id, err := aiprovider.ResolveModelID(string(bind.alias), routing); err == nil && strings.TrimSpace(id) != "" {
			return id, providerDisplayName(string(routing))
		}
	}
	if fallbackID == "" {
		return "", ""
	}
	return fallbackID, modelProviderLabel(fallbackID, viaOpenRouter)
}

func primaryRoutingProvider(configured []string) aiprovider.ProviderName {
	for _, p := range configured {
		switch aiprovider.ProviderName(p) {
		case aiprovider.ProviderOpenRouter,
			aiprovider.ProviderAnthropic,
			aiprovider.ProviderOpenAI,
			aiprovider.ProviderAzureOpenAI,
			aiprovider.ProviderBedrock,
			aiprovider.ProviderVertex:
			return aiprovider.ProviderName(p)
		}
	}
	return ""
}

func normalizeProviders(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, p := range in {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func containsProvider(list []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, p := range list {
		if p == want {
			return true
		}
	}
	return false
}

func fetchOpenRouterModelNames(ctx context.Context) (map[string]string, error) {
	textModels, err := openrouter.ListModelsByOutputModality(ctx, nil, openrouter.DefaultBaseURL, "text")
	if err != nil {
		return nil, err
	}
	imageModels, err := openrouter.ListModelsByOutputModality(ctx, nil, openrouter.DefaultBaseURL, "image")
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(textModels)+len(imageModels))
	for _, m := range textModels {
		names[m.ID] = m.Name
	}
	for _, m := range imageModels {
		names[m.ID] = m.Name
	}
	return names, nil
}

func modelDisplayName(id string, names map[string]string) string {
	if n := strings.TrimSpace(names[id]); n != "" {
		return n
	}
	return id
}

func modelProviderLabel(id string, viaOpenRouter bool) string {
	id = strings.TrimSpace(id)
	// Aliases and native ids: describe the backend family without assuming OpenRouter (AP.3/AP.6).
	if !strings.Contains(id, "/") {
		switch {
		case strings.HasPrefix(id, "claude-"):
			return "Anthropic"
		case strings.HasPrefix(id, "gpt-") || strings.HasPrefix(id, "o1") || strings.HasPrefix(id, "o3") || strings.HasPrefix(id, "dall-e"):
			return "OpenAI"
		case strings.HasPrefix(id, "gemini-") || strings.HasPrefix(id, "imagen-"):
			return "Google"
		case strings.Contains(id, "."):
			return "AWS Bedrock"
		default:
			return "AI provider"
		}
	}
	before, _, ok := strings.Cut(id, "/")
	if !ok || strings.TrimSpace(before) == "" {
		if viaOpenRouter {
			return "OpenRouter"
		}
		return "AI provider"
	}
	label := titleWords(strings.ReplaceAll(before, "-", " "))
	if viaOpenRouter {
		return label + " (via OpenRouter)"
	}
	return label
}

func providerDisplayName(id string) string {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case string(aiprovider.ProviderOpenRouter):
		return "OpenRouter"
	case string(aiprovider.ProviderAnthropic):
		return "Anthropic"
	case string(aiprovider.ProviderOpenAI):
		return "OpenAI"
	case string(aiprovider.ProviderAzureOpenAI):
		return "Azure OpenAI"
	case string(aiprovider.ProviderBedrock):
		return "AWS Bedrock"
	case string(aiprovider.ProviderVertex):
		return "Google Vertex AI"
	default:
		if id == "" {
			return "AI provider"
		}
		return titleWords(strings.ReplaceAll(id, "_", " "))
	}
}

func titleWords(s string) string {
	parts := strings.Fields(s)
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func appendUnique(dst []string, items ...string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, s := range dst {
		seen[s] = struct{}{}
	}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		dst = append(dst, item)
	}
	return dst
}
