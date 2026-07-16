package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

func TestResolveCatalogRequest_ExplicitProvider(t *testing.T) {
	d := Deps{Config: config.Config{}}
	provider, configured, opts := d.resolveCatalogRequest(context.Background(), "anthropic")
	if provider != aiprovider.ProviderAnthropic {
		t.Fatalf("provider: %s", provider)
	}
	if configured {
		t.Fatal("expected unconfigured without credentials")
	}
	if opts.APIKey != "" {
		t.Fatal("expected empty api key")
	}
	models, err := aiprovider.ListCatalog(context.Background(), provider, aiprovider.CatalogKindText, aiprovider.CatalogOptions{SkipLive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) == 0 {
		t.Fatal("AC-1: Anthropic text catalog must be non-empty without OpenRouter")
	}
}

func TestActivePlatformProvider_DefaultsOpenRouter(t *testing.T) {
	d := Deps{Config: config.Config{}}
	if got := d.activePlatformProvider(context.Background()); got != aiprovider.ProviderOpenRouter {
		t.Fatalf("got %s", got)
	}
}

func TestListAIModels_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/ai/models?provider=anthropic&kind=text", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d body=%s", rec.Code, rec.Body.String())
	}
}
