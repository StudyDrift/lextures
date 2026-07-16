package aiprovider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brttypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"golang.org/x/oauth2"
)

func TestResolveAzureDeployment(t *testing.T) {
	extra := map[string]any{
		"deployments":         map[string]any{"gpt-4o": "gpt4o-prod"},
		"default_deployment":  "fallback-dep",
	}
	if got := ResolveAzureDeployment("gpt-4o", extra); got != "gpt4o-prod" {
		t.Fatalf("map lookup: %q", got)
	}
	if got := ResolveAzureDeployment("other", extra); got != "fallback-dep" {
		t.Fatalf("default: %q", got)
	}
	if got := ResolveAzureDeployment("raw-id", nil); got != "raw-id" {
		t.Fatalf("passthrough: %q", got)
	}
}

func TestAzureOpenAI_DeploymentPath(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		if r.Header.Get("api-key") == "" {
			t.Fatal("missing api-key header")
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"azure ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer srv.Close()

	p := NewAzureOpenAIProvider("key", srv.URL, AzureOptions{
		APIVersion:        "2024-10-21",
		Deployments:       map[string]string{"gpt-4o": "gpt4o-prod"},
		DefaultDeployment: "unused",
	})
	got, err := p.Complete(context.Background(), "gpt-4o", []Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "azure ok" {
		t.Fatalf("text: %q", got.Text)
	}
	if !strings.Contains(gotPath, "/openai/deployments/gpt4o-prod/chat/completions") {
		t.Fatalf("path: %s", gotPath)
	}
	if !strings.Contains(gotPath, "api-version=2024-10-21") {
		t.Fatalf("api-version missing: %s", gotPath)
	}
	if _, ok := gotBody["model"]; ok {
		t.Fatal("Azure body must not include model")
	}
}

func TestFactory_AzureRequiresBaseURL(t *testing.T) {
	f := Factory{}
	_, err := f.Build(ProviderAzureOpenAI, "k", nil)
	if err == nil || ClassifyError(err) != ErrorTypeConfig {
		t.Fatalf("want config error, got %v", err)
	}
}

func TestFactory_BedrockAuthModes(t *testing.T) {
	f := Factory{}
	_, err := f.BuildWithAuth(ProviderBedrock, AuthMaterial{}, map[string]any{"auth_mode": "access_key"})
	if err == nil || !strings.Contains(err.Error(), "aws_access_key_id") {
		t.Fatalf("want access_key validation, got %v", err)
	}
	_, err = f.BuildWithAuth(ProviderBedrock, AuthMaterial{
		Secrets: map[string]string{
			secretKeyAWSAccessKeyID:     "AKIATEST",
			secretKeyAWSSecretAccessKey: "secret",
		},
	}, map[string]any{"auth_mode": "access_key", "aws_region": "us-west-2"})
	if err != nil {
		t.Fatalf("access_key build: %v", err)
	}
	p, err := f.BuildWithAuth(ProviderBedrock, AuthMaterial{}, map[string]any{"auth_mode": "iam_role", "aws_region": "us-east-1"})
	if err != nil {
		t.Fatalf("iam_role build: %v", err)
	}
	if bp, ok := p.(*BedrockProvider); !ok || bp.authMode != AuthModeIAMRole {
		t.Fatalf("want iam_role provider, got %#v", p)
	}
}

func TestFactory_VertexAuthModes(t *testing.T) {
	f := Factory{}
	_, err := f.BuildWithAuth(ProviderVertex, AuthMaterial{}, map[string]any{
		"auth_mode": "api_key", "gcp_project": "p", "gcp_location": "us-central1",
	})
	if err == nil || ClassifyError(err) != ErrorTypeConfig {
		t.Fatalf("want api_key required, got %v", err)
	}
	_, err = f.BuildWithAuth(ProviderVertex, AuthMaterial{APIKey: "k"}, map[string]any{
		"auth_mode": "api_key", "gcp_project": "p", "gcp_location": "us-central1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.BuildWithAuth(ProviderVertex, AuthMaterial{}, map[string]any{
		"auth_mode": "service_account", "gcp_project": "p", "gcp_location": "us-central1",
	})
	if err == nil || !strings.Contains(err.Error(), "service_account_json") {
		t.Fatalf("want SA required, got %v", err)
	}
}

type mockBedrockSDK struct {
	lastModel string
}

func (m *mockBedrockSDK) Converse(ctx context.Context, params *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	_ = ctx
	if params.ModelId != nil {
		m.lastModel = *params.ModelId
	}
	return &bedrockruntime.ConverseOutput{
		Output: &brttypes.ConverseOutputMemberMessage{
			Value: brttypes.Message{
				Role: brttypes.ConversationRoleAssistant,
				Content: []brttypes.ContentBlock{
					&brttypes.ContentBlockMemberText{Value: "bedrock sdk ok"},
				},
			},
		},
		Usage: &brttypes.TokenUsage{InputTokens: aws.Int32(2), OutputTokens: aws.Int32(3)},
	}, nil
}

func TestBedrockProvider_SDKComplete(t *testing.T) {
	mock := &mockBedrockSDK{}
	p := NewBedrockProviderWithSDK(mock, "us-east-1")
	got, err := p.Complete(context.Background(), "amazon.nova-lite-v1:0", []Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "bedrock sdk ok" || mock.lastModel != "amazon.nova-lite-v1:0" {
		t.Fatalf("got=%q model=%q", got.Text, mock.lastModel)
	}
	if got.Usage.PromptTokens != 2 || got.Usage.CompletionTokens != 3 {
		t.Fatalf("usage: %+v", got.Usage)
	}
}

type staticTokenSource struct{ token string }

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: s.token}, nil
}

func TestVertexProvider_TokenSourceAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"vertex ok"}]}}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1,"totalTokenCount":2}}`))
	}))
	defer srv.Close()
	p := NewVertexProviderWithTokenSource(srv.URL, staticTokenSource{token: "ya29.test"})
	got, err := p.Complete(context.Background(), "gemini-1.5-flash", []Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "vertex ok" {
		t.Fatalf("text: %q", got.Text)
	}
	if gotAuth != "Bearer ya29.test" {
		t.Fatalf("auth: %q", gotAuth)
	}
}

func TestClassifyError_Types(t *testing.T) {
	if ClassifyError(newConfigError(ProviderAzureOpenAI, "missing base")) != ErrorTypeConfig {
		t.Fatal("config")
	}
	if ClassifyError(newAuthError(ProviderBedrock, 403, "denied")) != ErrorTypeAuth {
		t.Fatal("auth")
	}
	if ClassifyError(newProviderError(ProviderOpenAI, 429, "rate")) != ErrorTypeQuota {
		t.Fatal("quota")
	}
	if IsRetryable(newAuthError(ProviderBedrock, 403, "denied")) {
		t.Fatal("auth must not be retryable")
	}
	if !IsRetryable(newProviderError(ProviderOpenAI, 503, "down")) {
		t.Fatal("5xx should be retryable")
	}
}

func TestAuthModeFromSettings(t *testing.T) {
	if AuthModeFromSettings(ProviderBedrock, nil) != AuthModeAPIKey {
		t.Fatal("default")
	}
	if AuthModeFromSettings(ProviderBedrock, map[string]any{"auth_mode": "IAM_ROLE"}) != AuthModeIAMRole {
		t.Fatal("normalize")
	}
}
