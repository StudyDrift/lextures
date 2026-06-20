package aidisclosure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

func TestAssembleDisclosure_UsesOpenRouterNames(t *testing.T) {
	t.Parallel()
	doc := assembleDisclosure(map[string]string{
		user.DefaultCourseSetupModelID: "Trinity Mini",
		"openai/gpt-4o-mini":           "GPT-4o mini",
	})
	if len(doc.Models) == 0 {
		t.Fatal("expected models")
	}
	if doc.Models[0].Name != "Trinity Mini" {
		t.Fatalf("name=%q", doc.Models[0].Name)
	}
	if doc.Provider != "openrouter" {
		t.Fatalf("provider=%q", doc.Provider)
	}
	if len(doc.Features) != len(disclosureFeatures) {
		t.Fatalf("features=%d", len(doc.Features))
	}
}

func TestAssembleDisclosure_FallsBackToModelID(t *testing.T) {
	t.Parallel()
	doc := assembleDisclosure(nil)
	found := false
	for _, m := range doc.Models {
		if m.ID == "openai/gpt-4o-mini" {
			found = true
			if m.Name != "openai/gpt-4o-mini" {
				t.Fatalf("name=%q", m.Name)
			}
			if m.Provider != "Openai (via OpenRouter)" {
				t.Fatalf("provider=%q", m.Provider)
			}
		}
	}
	if !found {
		t.Fatal("expected translation model")
	}
}

func TestBuildPublicDisclosure_OpenRouterMock(t *testing.T) {
	body := `{"data":[{"id":"` + user.DefaultCourseSetupModelID + `","name":"Trinity Mini"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	orig := fetchModelNames
	fetchModelNames = func(ctx context.Context) (map[string]string, error) {
		return listModelNamesWithBase(ctx, srv.URL)
	}
	defer func() { fetchModelNames = orig }()

	doc, err := BuildPublicDisclosure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Models) == 0 {
		t.Fatal("expected models")
	}
	if doc.Models[0].Name != "Trinity Mini" {
		t.Fatalf("name=%q", doc.Models[0].Name)
	}
}

func TestPublicDisclosureJSON_Caches(t *testing.T) {
	InvalidatePublicDisclosureCache()
	calls := 0
	body := `{"data":[{"id":"` + user.DefaultCourseSetupModelID + `","name":"Cached Model"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	orig := fetchModelNames
	fetchModelNames = func(ctx context.Context) (map[string]string, error) {
		return listModelNamesWithBase(ctx, srv.URL)
	}
	defer func() { fetchModelNames = orig }()

	ctx := context.Background()
	first, err := PublicDisclosureJSON(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PublicDisclosureJSON(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("expected cached payload")
	}
	if calls != 2 {
		t.Fatalf("openrouter calls=%d want 2 (text + image once)", calls)
	}

	var doc PublicDisclosure
	if err := json.Unmarshal(first, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Models[0].Name != "Cached Model" {
		t.Fatalf("name=%q", doc.Models[0].Name)
	}
}

func listModelNamesWithBase(ctx context.Context, baseURL string) (map[string]string, error) {
	textModels, err := openrouter.ListModelsByOutputModality(ctx, nil, baseURL, "text")
	if err != nil {
		return nil, err
	}
	imageModels, err := openrouter.ListModelsByOutputModality(ctx, nil, baseURL, "image")
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