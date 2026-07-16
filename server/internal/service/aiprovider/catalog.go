package aiprovider

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lextures/lextures/server/internal/service/openrouter"
)

// CatalogKind filters models by capability for settings pickers (AP.3 FR-3).
type CatalogKind string

const (
	CatalogKindText   CatalogKind = "text"
	CatalogKindImage  CatalogKind = "image"
	CatalogKindVision CatalogKind = "vision"
)

const catalogCacheTTL = 5 * time.Minute

// CatalogModel is a row for GET /api/v1/settings/ai/models (AP.3 §9).
type CatalogModel struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name"`
	ContextLength            *uint64  `json:"contextLength,omitempty"`
	InputPricePerMillionUSD  *float64 `json:"inputPricePerMillionUsd,omitempty"`
	OutputPricePerMillionUSD *float64 `json:"outputPricePerMillionUsd,omitempty"`
	ModalitiesSummary        *string  `json:"modalitiesSummary,omitempty"`
	Modalities               []string `json:"modalities,omitempty"`
}

// CatalogOptions configures ListCatalog.
type CatalogOptions struct {
	// APIKey enables live list enrichment when the provider supports it.
	APIKey string
	// BaseURL overrides the provider API base for live lists (tests / Azure).
	BaseURL string
	// HTTPClient overrides the client used for live / OpenRouter fetches.
	HTTPClient *http.Client
	// SkipLive forces curated-only (tests).
	SkipLive bool
}

type catalogCacheEntry struct {
	models    []CatalogModel
	expiresAt time.Time
}

var (
	catalogCacheMu sync.Mutex
	catalogCache   = map[string]catalogCacheEntry{}
)

// ParseCatalogKind validates kind=text|image|vision.
func ParseCatalogKind(s string) (CatalogKind, error) {
	k := CatalogKind(strings.ToLower(strings.TrimSpace(s)))
	if k == "" {
		k = CatalogKindText
	}
	switch k {
	case CatalogKindText, CatalogKindImage, CatalogKindVision:
		return k, nil
	default:
		return "", fmt.Errorf("aiprovider: invalid catalog kind %q (use text, image, or vision)", s)
	}
}

// ListCatalog returns models for a provider+kind. OpenRouter uses the live public
// catalog; other providers use a curated list and MAY enrich from live APIs when
// credentials are present. Live failures fall back to curated (AP.3 NFR Reliability).
func ListCatalog(ctx context.Context, provider ProviderName, kind CatalogKind, opts CatalogOptions) ([]CatalogModel, error) {
	if _, ok := NormalizeProviderName(string(provider)); !ok || provider == ProviderDryRun {
		return nil, fmt.Errorf("aiprovider: unsupported catalog provider %q", provider)
	}
	switch kind {
	case CatalogKindText, CatalogKindImage, CatalogKindVision:
	default:
		return nil, fmt.Errorf("aiprovider: invalid catalog kind %q", kind)
	}

	if provider == ProviderOpenRouter {
		models, err := listOpenRouterCatalog(ctx, kind, opts)
		if err != nil {
			RecordCatalogFetch(string(provider), "error")
			return nil, err
		}
		RecordCatalogFetch(string(provider), "live")
		return models, nil
	}

	curated := curatedCatalog(provider, kind)
	if opts.SkipLive || strings.TrimSpace(opts.APIKey) == "" {
		RecordCatalogFetch(string(provider), "curated")
		return curated, nil
	}

	cacheKey := string(provider) + "|" + string(kind) + "|" + opts.BaseURL
	if cached, ok := getCachedCatalog(cacheKey); ok {
		RecordCatalogFetch(string(provider), "cached")
		return cached, nil
	}

	live, err := fetchLiveCatalog(ctx, provider, kind, opts)
	if err != nil || len(live) == 0 {
		RecordCatalogFetch(string(provider), "live_fallback")
		return curated, nil
	}
	merged := mergeCatalog(curated, live)
	putCachedCatalog(cacheKey, merged)
	RecordCatalogFetch(string(provider), "live")
	return merged, nil
}

func listOpenRouterCatalog(ctx context.Context, kind CatalogKind, opts CatalogOptions) ([]CatalogModel, error) {
	modality := "text"
	if kind == CatalogKindImage {
		modality = "image"
	}
	// Vision: OpenRouter has no separate output modality; use text catalog (many multimodal).
	listed, err := openrouter.ListModelsByOutputModality(ctx, opts.HTTPClient, opts.BaseURL, modality)
	if err != nil {
		return nil, err
	}
	out := make([]CatalogModel, 0, len(listed))
	for _, m := range listed {
		out = append(out, CatalogModel{
			ID:                       m.ID,
			Name:                     m.Name,
			ContextLength:            m.ContextLength,
			InputPricePerMillionUSD:  m.InputPricePerMillionUSD,
			OutputPricePerMillionUSD: m.OutputPricePerMillionUSD,
			ModalitiesSummary:        m.ModalitiesSummary,
			Modalities:               modalitiesForKind(kind),
		})
	}
	return out, nil
}

func curatedCatalog(provider ProviderName, kind CatalogKind) []CatalogModel {
	rows := curatedCatalogAll[provider]
	out := make([]CatalogModel, 0, len(rows))
	for _, row := range rows {
		if !kindMatches(row.Modalities, kind) {
			continue
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func kindMatches(mods []string, kind CatalogKind) bool {
	want := string(kind)
	for _, m := range mods {
		if strings.EqualFold(m, want) {
			return true
		}
	}
	// text kind also includes vision-capable chat models
	if kind == CatalogKindText {
		for _, m := range mods {
			if strings.EqualFold(m, "text") || strings.EqualFold(m, "vision") {
				return true
			}
		}
	}
	return false
}

func modalitiesForKind(kind CatalogKind) []string {
	switch kind {
	case CatalogKindImage:
		return []string{"image"}
	case CatalogKindVision:
		return []string{"text", "vision"}
	default:
		return []string{"text"}
	}
}

func mergeCatalog(curated, live []CatalogModel) []CatalogModel {
	byID := map[string]CatalogModel{}
	order := make([]string, 0, len(curated)+len(live))
	for _, m := range curated {
		byID[m.ID] = m
		order = append(order, m.ID)
	}
	for _, m := range live {
		if existing, ok := byID[m.ID]; ok {
			// Prefer curated display name / modalities; keep live pricing when present.
			if m.InputPricePerMillionUSD != nil {
				existing.InputPricePerMillionUSD = m.InputPricePerMillionUSD
			}
			if m.OutputPricePerMillionUSD != nil {
				existing.OutputPricePerMillionUSD = m.OutputPricePerMillionUSD
			}
			if m.ContextLength != nil {
				existing.ContextLength = m.ContextLength
			}
			byID[m.ID] = existing
			continue
		}
		byID[m.ID] = m
		order = append(order, m.ID)
	}
	out := make([]CatalogModel, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func getCachedCatalog(key string) ([]CatalogModel, bool) {
	catalogCacheMu.Lock()
	defer catalogCacheMu.Unlock()
	e, ok := catalogCache[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return append([]CatalogModel(nil), e.models...), true
}

func putCachedCatalog(key string, models []CatalogModel) {
	catalogCacheMu.Lock()
	defer catalogCacheMu.Unlock()
	catalogCache[key] = catalogCacheEntry{
		models:    append([]CatalogModel(nil), models...),
		expiresAt: time.Now().Add(catalogCacheTTL),
	}
}

// ClearCatalogCache drops live catalog cache (tests).
func ClearCatalogCache() {
	catalogCacheMu.Lock()
	clear(catalogCache)
	catalogCacheMu.Unlock()
}

// curatedCatalogAll is the in-process seed catalog (AP.3 Maintainability).
var curatedCatalogAll = map[ProviderName][]CatalogModel{
	ProviderAnthropic: {
		{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Modalities: []string{"text"}},
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Modalities: []string{"text", "vision"}},
		{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Modalities: []string{"text", "vision"}},
		{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Modalities: []string{"text", "vision"}},
	},
	ProviderOpenAI: {
		{ID: "gpt-4o-mini", Name: "GPT-4o mini", Modalities: []string{"text", "vision"}},
		{ID: "gpt-4o", Name: "GPT-4o", Modalities: []string{"text", "vision"}},
		{ID: "gpt-4.1", Name: "GPT-4.1", Modalities: []string{"text", "vision"}},
		{ID: "gpt-4.1-mini", Name: "GPT-4.1 mini", Modalities: []string{"text", "vision"}},
		{ID: "o3-mini", Name: "o3-mini", Modalities: []string{"text"}},
		{ID: "dall-e-3", Name: "DALL·E 3", Modalities: []string{"image"}},
	},
	ProviderAzureOpenAI: {
		{ID: "gpt-4o-mini", Name: "GPT-4o mini (deployment)", Modalities: []string{"text", "vision"}},
		{ID: "gpt-4o", Name: "GPT-4o (deployment)", Modalities: []string{"text", "vision"}},
		{ID: "dall-e-3", Name: "DALL·E 3 (deployment)", Modalities: []string{"image"}},
	},
	ProviderBedrock: {
		{ID: "anthropic.claude-3-5-haiku-20241022-v1:0", Name: "Claude 3.5 Haiku", Modalities: []string{"text"}},
		{ID: "anthropic.claude-3-5-sonnet-20241022-v2:0", Name: "Claude 3.5 Sonnet", Modalities: []string{"text", "vision"}},
		{ID: "amazon.titan-text-premier-v1:0", Name: "Amazon Titan Text Premier", Modalities: []string{"text"}},
	},
	ProviderVertex: {
		{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", Modalities: []string{"text", "vision"}},
		{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", Modalities: []string{"text", "vision"}},
		{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", Modalities: []string{"text", "vision"}},
		{ID: "imagen-3.0-generate-001", Name: "Imagen 3", Modalities: []string{"image"}},
	},
}
