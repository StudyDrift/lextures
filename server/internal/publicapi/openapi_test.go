package publicapi

import (
	"encoding/json"
	"testing"
)

func TestOpenAPISpec_IsValidJSONAndVersion(t *testing.T) {
	t.Parallel()
	var doc map[string]any
	if err := json.Unmarshal(SpecBytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Fatalf("openapi version: %v", doc["openapi"])
	}
	info, ok := doc["info"].(map[string]any)
	if !ok || info["title"] == "" {
		t.Fatalf("missing info.title")
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) < 5 {
		t.Fatalf("expected paths, got %v", doc["paths"])
	}
}
