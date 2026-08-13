package marketingpublish

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGitHubDispatchPayloadContainsOnlyBuildMetadata(t *testing.T) {
	var got map[string]any
	c := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/repos/acme/site/dispatches" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatal("missing authorization")
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
	})}
	b := Build{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Reason: "publish", Paths: []string{"/z", "/a"}}
	if err := (&GitHubDispatcher{Client: c}).Dispatch(context.Background(), Settings{Repository: "acme/site", Token: "secret"}, b); err != nil {
		t.Fatal(err)
	}
	if got["event_type"] != EventType {
		t.Fatalf("payload=%v", got)
	}
	p := got["client_payload"].(map[string]any)
	if p["paths"].([]any)[0].(string) != "/a" {
		t.Fatalf("paths not sorted: %v", p["paths"])
	}
	if _, ok := p["token"]; ok {
		t.Fatal("token leaked into payload")
	}
}

func TestFindRunMapsTerminalStatus(t *testing.T) {
	now := time.Now().UTC()
	c := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		body := `{"workflow_runs":[{"id":42,"html_url":"https://github/run/42","status":"completed","conclusion":"success","created_at":"` + now.Format(time.RFC3339) + `"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	id, url, status, err := (&GitHubDispatcher{Client: c}).FindRun(context.Background(), Settings{Repository: "acme/site"}, Build{DispatchedAt: &now})
	if err != nil || id != "42" || url == "" || status != "succeeded" {
		t.Fatalf("%s %s %s %v", id, url, status, err)
	}
}
