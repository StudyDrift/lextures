package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestAPIBaseURL_HostedRequestOverridesLoopbackDefault(t *testing.T) {
	t.Parallel()
	d := Deps{Config: config.Config{
		LTIAPIBaseURL:   "http://localhost:8080",
		PublicWebOrigin: "http://localhost:5173",
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/integrations/mcp", nil)
	req.Host = "self.lextures.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	if got := d.apiBaseURL(req); got != "https://self.lextures.com" {
		t.Fatalf("apiBaseURL=%q", got)
	}
}

func TestAPIBaseURL_ExplicitPublicLTIBaseWins(t *testing.T) {
	t.Parallel()
	d := Deps{Config: config.Config{LTIAPIBaseURL: "https://api.example.edu"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/integrations/mcp", nil)
	req.Host = "self.lextures.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	if got := d.apiBaseURL(req); got != "https://api.example.edu" {
		t.Fatalf("apiBaseURL=%q", got)
	}
}

func TestAPIBaseURL_LocalRequestStaysOnLoopbackAPI(t *testing.T) {
	t.Parallel()
	d := Deps{Config: config.Config{
		LTIAPIBaseURL:   "http://localhost:8080",
		PublicWebOrigin: "http://localhost:5173",
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/integrations/mcp", nil)
	req.Host = "localhost:8080"

	if got := d.apiBaseURL(req); got != "http://localhost:8080" {
		t.Fatalf("apiBaseURL=%q", got)
	}
}

func TestAPIBaseURL_PublicWebOriginWhenRequestHostIsLoopback(t *testing.T) {
	t.Parallel()
	d := Deps{Config: config.Config{
		LTIAPIBaseURL:   "http://localhost:8080",
		PublicWebOrigin: "https://self.lextures.com/",
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/integrations/mcp", nil)
	req.Host = "127.0.0.1:8080"

	if got := d.apiBaseURL(req); got != "https://self.lextures.com" {
		t.Fatalf("apiBaseURL=%q", got)
	}
}

func TestAPIBaseURL_CloudflareVisitorUpgradesHTTPHop(t *testing.T) {
	t.Parallel()
	d := Deps{Config: config.Config{LTIAPIBaseURL: "http://localhost:8080"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/integrations/mcp", nil)
	req.Host = "self.lextures.com"
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("CF-Visitor", `{"scheme":"https"}`)

	if got := d.apiBaseURL(req); got != "https://self.lextures.com" {
		t.Fatalf("apiBaseURL=%q", got)
	}
}

func TestAPIBaseURL_RejectsInjectedHost(t *testing.T) {
	t.Parallel()
	d := Deps{Config: config.Config{
		LTIAPIBaseURL:   "http://localhost:8080",
		PublicWebOrigin: "https://self.lextures.com",
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/integrations/mcp", nil)
	req.Host = "evil.example\r\nX-Ignore: 1"

	if got := d.apiBaseURL(req); got != "https://self.lextures.com" {
		t.Fatalf("apiBaseURL=%q", got)
	}
}
