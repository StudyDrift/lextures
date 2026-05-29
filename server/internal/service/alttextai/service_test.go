package alttextai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/service/openrouter"
)

func TestSuggest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "A red circle on a white background."}},
			},
		})
	}))
	defer srv.Close()

	client := openrouter.NewClientWithBaseURL("test-key", srv.URL)
	suggestion, confidence, err := Suggest(client, DefaultModel, "https://example.com/img.png", "English")
	if err != nil {
		t.Fatal(err)
	}
	if suggestion == "" {
		t.Fatal("expected suggestion")
	}
	if confidence <= 0 {
		t.Fatalf("expected confidence > 0, got %v", confidence)
	}
}
