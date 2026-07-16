package aiprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListCatalog_AnthropicCuratedWithoutKey(t *testing.T) {
	ClearCatalogCache()
	models, err := ListCatalog(context.Background(), ProviderAnthropic, CatalogKindText, CatalogOptions{SkipLive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) == 0 {
		t.Fatal("expected non-empty curated Anthropic catalog")
	}
	for _, m := range models {
		if m.ID == "" || m.Name == "" {
			t.Fatalf("incomplete model: %+v", m)
		}
	}
}

func TestListCatalog_OpenAIImageKind(t *testing.T) {
	models, err := ListCatalog(context.Background(), ProviderOpenAI, CatalogKindImage, CatalogOptions{SkipLive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) == 0 {
		t.Fatal("expected image models")
	}
	for _, m := range models {
		if !kindMatches(m.Modalities, CatalogKindImage) {
			t.Fatalf("non-image model in image catalog: %+v", m)
		}
	}
}

func TestListCatalog_VisionKind(t *testing.T) {
	models, err := ListCatalog(context.Background(), ProviderAnthropic, CatalogKindVision, CatalogOptions{SkipLive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) == 0 {
		t.Fatal("expected vision models")
	}
}

func TestListCatalog_LiveEnrichmentOpenAI(t *testing.T) {
	ClearCatalogCache()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "gpt-4o"},
				{"id": "gpt-4o-mini"},
				{"id": "dall-e-3"},
				{"id": "whisper-1"},
			},
		})
	}))
	defer srv.Close()

	models, err := ListCatalog(context.Background(), ProviderOpenAI, CatalogKindText, CatalogOptions{
		APIKey:     "sk-test",
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) == 0 {
		t.Fatal("expected models")
	}
	seen := map[string]bool{}
	for _, m := range models {
		seen[m.ID] = true
	}
	if !seen["gpt-4o"] || !seen["gpt-4o-mini"] {
		t.Fatalf("missing expected ids: %v", seen)
	}
	if seen["dall-e-3"] {
		t.Fatal("image model should not appear in text catalog")
	}
}

func TestListCatalog_LiveFailureFallsBack(t *testing.T) {
	ClearCatalogCache()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	models, err := ListCatalog(context.Background(), ProviderAnthropic, CatalogKindText, CatalogOptions{
		APIKey:     "key",
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) == 0 {
		t.Fatal("expected curated fallback")
	}
}

func TestListCatalog_OpenRouter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "arcee-ai/trinity-mini:free", "name": "Trinity Mini"},
				{"id": "other/text", "name": "Other"},
			},
		})
	}))
	defer srv.Close()

	models, err := ListCatalog(context.Background(), ProviderOpenRouter, CatalogKindText, CatalogOptions{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models", len(models))
	}
}

func TestParseCatalogKind(t *testing.T) {
	k, err := ParseCatalogKind("")
	if err != nil || k != CatalogKindText {
		t.Fatalf("default: %v %v", k, err)
	}
	if _, err := ParseCatalogKind("audio"); err == nil {
		t.Fatal("expected error")
	}
}
