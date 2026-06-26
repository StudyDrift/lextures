package statuspage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientSummary_UsesCache(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"page": {"id":"page-1","url":"https://status.example.com"},
			"incidents": [{"id":"1","name":"Outage","status":"investigating","impact":"major"}]
		}`))
	}))
	defer srv.Close()

	client := NewClient(Config{
		Enabled:      true,
		PageURL:      "https://status.lextures.io",
		APIKey:       "test-key",
		PageID:       "page-1",
		CacheTTL:     time.Minute,
		APIBaseURL:   srv.URL,
		HTTPClient:   srv.Client(),
		ComponentMap: ComponentMap{},
	})

	ctx := context.Background()
	first, err := client.Summary(ctx)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	second, err := client.Summary(ctx)
	if err != nil {
		t.Fatalf("summary cached: %v", err)
	}
	if calls != 1 {
		t.Fatalf("upstream calls=%d want 1", calls)
	}
	if len(first.Incidents) != 1 || first.Incidents[0].Name != "Outage" {
		t.Fatalf("first=%+v", first)
	}
	if second.Status != "major" {
		t.Fatalf("status=%q", second.Status)
	}
}

func TestClientSummary_NotConfigured(t *testing.T) {
	client := NewClient(Config{Enabled: false})
	out, err := client.Summary(context.Background())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if out.Configured || len(out.Incidents) != 0 {
		t.Fatalf("out=%+v", out)
	}
}