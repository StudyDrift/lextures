package aiprovider

import (
	"strings"
	"testing"
)

func TestResolveModelID_TextFastOpenAI(t *testing.T) {
	id, err := ResolveModelID("text-fast", ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if id != "gpt-4o-mini" {
		t.Fatalf("got %q", id)
	}
}

func TestResolveModelID_EveryAliasEveryProvider(t *testing.T) {
	for _, alias := range ListModelAliases() {
		for _, p := range ListProviders() {
			id, err := ResolveModelID(alias, p)
			if err != nil {
				// Image aliases may be unavailable; legacy vendor aliases may be provider-scoped.
				if strings.Contains(err.Error(), "not available") {
					continue
				}
				t.Fatalf("alias=%s provider=%s: %v", alias, p, err)
			}
			if strings.TrimSpace(id) == "" {
				t.Fatalf("alias=%s provider=%s: empty id", alias, p)
			}
		}
	}
}

func TestResolveModelID_UnknownAlias(t *testing.T) {
	_, err := ResolveModelID("not-a-real-alias", ProviderOpenAI)
	if err == nil || !strings.Contains(err.Error(), "unknown model alias") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveModelID_PassThroughOpenRouterID(t *testing.T) {
	const stored = "arcee-ai/trinity-mini:free"
	id, err := ResolveModelID(stored, ProviderOpenRouter)
	if err != nil {
		t.Fatal(err)
	}
	if id != stored {
		t.Fatalf("want pass-through, got %q", id)
	}
}

func TestResolveModelID_OpenRouterIDOnAnthropicMapsViaAlias(t *testing.T) {
	id, err := ResolveModelID("arcee-ai/trinity-mini:free", ProviderAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if id != "claude-3-5-haiku-20241022" {
		t.Fatalf("got %q", id)
	}
}

func TestResolveModelID_UnmappedOpenRouterIDOnAnthropic(t *testing.T) {
	_, err := ResolveModelID("some-vendor/unknown-model:free", ProviderAnthropic)
	if err == nil || !strings.Contains(err.Error(), "no alias mapping") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveModelID_NativePassThrough(t *testing.T) {
	id, err := ResolveModelID("claude-sonnet-4-20250514", ProviderAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if id != "claude-sonnet-4-20250514" {
		t.Fatalf("got %q", id)
	}
}

func TestResolveModelID_ImageGenUnavailableOnAnthropic(t *testing.T) {
	_, err := ResolveModelID("image-gen", ProviderAnthropic)
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveModelID_LegacyClaude(t *testing.T) {
	id, err := ResolveModelID("claude-3-5-sonnet", ProviderAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if id != "claude-3-5-sonnet-20241022" {
		t.Fatalf("got %q", id)
	}
}

func TestResolveModelID_LegacyGPT4oNotOnAnthropic(t *testing.T) {
	_, err := ResolveModelID("gpt-4o", ProviderAnthropic)
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("got %v", err)
	}
}

func TestListModelAliasInfos(t *testing.T) {
	infos := ListModelAliasInfos()
	if len(infos) < 12 {
		t.Fatalf("expected feature+role aliases, got %d", len(infos))
	}
	var foundFast bool
	for _, info := range infos {
		if info.ID == "text-fast" {
			foundFast = true
			if info.Label == "" || len(info.Capabilities) == 0 {
				t.Fatalf("incomplete info: %+v", info)
			}
		}
	}
	if !foundFast {
		t.Fatal("missing text-fast")
	}
}

func TestFeatureAliasCanonicalizes(t *testing.T) {
	a, err := ResolveModelID("course-setup", ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ResolveModelID("text-fast", ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("course-setup=%q text-fast=%q", a, b)
	}
}
