package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func fetchLiveCatalog(ctx context.Context, provider ProviderName, kind CatalogKind, opts CatalogOptions) ([]CatalogModel, error) {
	switch provider {
	case ProviderOpenAI:
		return fetchOpenAIModels(ctx, opts, kind)
	case ProviderAnthropic:
		return fetchAnthropicModels(ctx, opts, kind)
	default:
		// Azure/Bedrock/Vertex live lists are deployment/region specific; curated only for v1.
		return nil, fmt.Errorf("aiprovider: live catalog not supported for %s", provider)
	}
}

func fetchOpenAIModels(ctx context.Context, opts CatalogOptions, kind CatalogKind) ([]CatalogModel, error) {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = openAIDefaultBase
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(opts.APIKey))
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("openai: list models status %d", res.StatusCode)
	}
	var top struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, err
	}
	out := make([]CatalogModel, 0, len(top.Data))
	for _, row := range top.Data {
		id := strings.TrimSpace(row.ID)
		if id == "" || !openAIIDMatchesKind(id, kind) {
			continue
		}
		out = append(out, CatalogModel{
			ID:         id,
			Name:       id,
			Modalities: modalitiesForOpenAIID(id),
		})
	}
	return out, nil
}

func openAIIDMatchesKind(id string, kind CatalogKind) bool {
	mods := modalitiesForOpenAIID(id)
	return kindMatches(mods, kind)
}

func modalitiesForOpenAIID(id string) []string {
	lower := strings.ToLower(id)
	if strings.Contains(lower, "dall-e") || strings.HasPrefix(lower, "gpt-image") {
		return []string{"image"}
	}
	if strings.HasPrefix(lower, "gpt-4o") || strings.HasPrefix(lower, "gpt-4.1") || strings.HasPrefix(lower, "chatgpt-") {
		return []string{"text", "vision"}
	}
	if strings.HasPrefix(lower, "gpt-") || strings.HasPrefix(lower, "o1") || strings.HasPrefix(lower, "o3") || strings.HasPrefix(lower, "o4") {
		return []string{"text"}
	}
	return []string{"text"}
}

func fetchAnthropicModels(ctx context.Context, opts CatalogOptions, kind CatalogKind) ([]CatalogModel, error) {
	if kind == CatalogKindImage {
		return nil, nil
	}
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = anthropicDefaultBase
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", strings.TrimSpace(opts.APIKey))
	req.Header.Set("anthropic-version", "2023-06-01")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("anthropic: list models status %d", res.StatusCode)
	}
	var top struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, err
	}
	out := make([]CatalogModel, 0, len(top.Data))
	for _, row := range top.Data {
		id := strings.TrimSpace(row.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(row.DisplayName)
		if name == "" {
			name = id
		}
		mods := []string{"text", "vision"}
		if !kindMatches(mods, kind) {
			continue
		}
		out = append(out, CatalogModel{
			ID:         id,
			Name:       name,
			Modalities: mods,
		})
	}
	return out, nil
}
