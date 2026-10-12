package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/service/starterlibrary"
)

func TestStarterLibrary_NoAuth_Returns401(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{}, JWTSigner: nil})
	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/starter-library"},
		{http.MethodGet, "/api/v1/starter-library/math-k-2"},
		{http.MethodPost, "/api/v1/starter-library/math-k-2/use"},
		{http.MethodPost, "/api/v1/starter-library/generate"},
	}
	for _, tc := range paths {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: got %d, want 401 (%s)", tc.method, tc.path, rr.Code, rr.Body.String())
		}
	}
}

func TestStarterDetailJSONIncludesPages(t *testing.T) {
	course, ok := starterlibrary.Get("math-3-5")
	if !ok {
		t.Fatal("missing catalog course")
	}
	raw, err := json.Marshal(detailStarter(course))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	mods, ok := body["modules"].([]any)
	if !ok || len(mods) < 1 {
		t.Fatalf("modules: %#v", body["modules"])
	}
	first, ok := mods[0].(map[string]any)
	if !ok {
		t.Fatalf("module: %#v", mods[0])
	}
	pages, ok := first["pages"].([]any)
	if !ok || len(pages) < 1 {
		t.Fatalf("pages: %#v", first)
	}
	if body["estimatedMinutes"] == nil || body["gradeBand"] != "3-5" {
		t.Fatalf("summary fields: %#v", body)
	}
}
