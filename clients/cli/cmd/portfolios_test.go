package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPortfoliosPublish_RequiresYes(t *testing.T) {
	err := portfoliosPublishCmd.RunE(portfoliosPublishCmd, []string{"p1"})
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err=%v", err)
	}
}

func TestPortfoliosList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me/portfolios" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"portfolios": []any{map[string]any{"id": "p1", "title": "Showcase", "isPublic": false}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	setCfg(srv.URL, "tok")
	if err := portfoliosListCmd.RunE(portfoliosListCmd, nil); err != nil {
		t.Fatalf("list: %v", err)
	}
}