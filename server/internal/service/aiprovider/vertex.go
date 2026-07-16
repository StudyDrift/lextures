package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const vertexOAuthScope = "https://www.googleapis.com/auth/cloud-platform"

// vertexTokenSource is mockable for tests (AP.8 AC-5).
type vertexTokenSource interface {
	Token() (*oauth2.Token, error)
}

// VertexProvider calls Google Vertex AI Gemini generateContent.
type VertexProvider struct {
	authMode    string
	client      *httpClient
	tokenSource vertexTokenSource
}

// NewVertexProvider builds a Vertex provider with API-key / bearer auth (legacy path).
func NewVertexProvider(apiKey, baseURL string) *VertexProvider {
	headers := map[string]string{}
	if strings.TrimSpace(apiKey) != "" {
		headers["x-goog-api-key"] = strings.TrimSpace(apiKey)
	}
	return &VertexProvider{
		authMode: AuthModeAPIKey,
		client:   newHTTPClient("", baseURL, headers),
	}
}

// NewVertexProviderWithTokenSource is for tests injecting a mocked OAuth token source.
func NewVertexProviderWithTokenSource(baseURL string, ts vertexTokenSource) *VertexProvider {
	return &VertexProvider{
		authMode:    AuthModeADC,
		client:      newHTTPClient("", baseURL, nil),
		tokenSource: ts,
	}
}

func newVertexProviderFromAuth(mode string, auth AuthMaterial, extra map[string]any) (*VertexProvider, error) {
	base, err := vertexBaseURL(extra)
	if err != nil {
		return nil, err
	}
	switch mode {
	case AuthModeAPIKey:
		return NewVertexProvider(auth.Secret(secretKeyAPIKey), base), nil
	case AuthModeServiceAccount:
		jsonKey := []byte(auth.Secret(secretKeyServiceAccountJSON))
		creds, err := google.CredentialsFromJSON(context.Background(), jsonKey, vertexOAuthScope)
		if err != nil {
			return nil, newConfigError(ProviderVertex, "invalid service_account_json: "+err.Error())
		}
		return &VertexProvider{
			authMode:    AuthModeServiceAccount,
			client:      newHTTPClient("", base, nil),
			tokenSource: creds.TokenSource,
		}, nil
	case AuthModeADC:
		creds, err := google.FindDefaultCredentials(context.Background(), vertexOAuthScope)
		if err != nil {
			return nil, newAuthError(ProviderVertex, 0, "ADC unavailable: "+err.Error())
		}
		return &VertexProvider{
			authMode:    AuthModeADC,
			client:      newHTTPClient("", base, nil),
			tokenSource: creds.TokenSource,
		}, nil
	default:
		return nil, newConfigError(ProviderVertex, fmt.Sprintf("vertex unsupported auth_mode %q", mode))
	}
}

func (p *VertexProvider) Name() ProviderName { return ProviderVertex }

func (p *VertexProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	if p == nil || p.client == nil || p.client.baseURL == "" {
		return ChatResult{}, newConfigError(ProviderVertex, "vertex not configured")
	}
	opt := firstChatOptions(opts)
	ctx, cancel := withChatTimeout(ctx, opt)
	defer cancel()
	client := p.client.withTimeout(opt.Timeout)

	if p.tokenSource != nil {
		tok, err := p.tokenSource.Token()
		if err != nil {
			return ChatResult{}, newAuthError(ProviderVertex, 401, "failed to obtain Vertex access token: "+err.Error())
		}
		if tok == nil || tok.AccessToken == "" {
			return ChatResult{}, newAuthError(ProviderVertex, 401, "empty Vertex access token")
		}
		headers := map[string]string{"Authorization": "Bearer " + tok.AccessToken}
		client = &httpClient{
			http:    client.http,
			apiKey:  "",
			baseURL: client.baseURL,
			headers: headers,
		}
	}

	body := vertexGenerateBody(messages, opt)
	path := "/" + strings.TrimPrefix(modelID, "/") + ":generateContent"
	b, _, err := client.postJSON(ctx, ProviderVertex, path, body)
	if err != nil {
		return ChatResult{}, err
	}
	return parseVertexResponse(b)
}

func vertexGenerateBody(messages []Message, opt ChatOptions) map[string]any {
	var system string
	var contents []map[string]any
	for _, m := range messages {
		switch m.Role {
		case "system":
			system = m.TextContent()
		case "assistant":
			contents = append(contents, map[string]any{
				"role":  "model",
				"parts": []map[string]string{{"text": m.TextContent()}},
			})
		default:
			contents = append(contents, map[string]any{
				"role":  "user",
				"parts": []map[string]string{{"text": m.TextContent()}},
			})
		}
	}
	system = ensureJSONSystem(system, opt.JSONMode)
	body := map[string]any{"contents": contents}
	if system != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]string{{"text": system}},
		}
	}
	genCfg := map[string]any{}
	if opt.MaxTokens > 0 {
		genCfg["maxOutputTokens"] = opt.MaxTokens
	}
	if opt.Temperature != nil {
		genCfg["temperature"] = *opt.Temperature
	}
	if opt.JSONMode {
		genCfg["responseMimeType"] = "application/json"
	}
	if len(genCfg) > 0 {
		body["generationConfig"] = genCfg
	}
	return body
}

func parseVertexResponse(b []byte) (ChatResult, error) {
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return ChatResult{}, fmt.Errorf("aiprovider: parse vertex response: %w", err)
	}
	var text string
	if len(parsed.Candidates) > 0 {
		for _, part := range parsed.Candidates[0].Content.Parts {
			if part.Text != "" {
				text = part.Text
				break
			}
		}
	}
	total := parsed.UsageMetadata.TotalTokenCount
	if total == 0 {
		total = parsed.UsageMetadata.PromptTokenCount + parsed.UsageMetadata.CandidatesTokenCount
	}
	return ChatResult{
		Text: text,
		Usage: UsageInfo{
			PromptTokens:     parsed.UsageMetadata.PromptTokenCount,
			CompletionTokens: parsed.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      total,
		},
	}, nil
}

func (p *VertexProvider) CompleteStream(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	_ = modelID
	_ = messages
	_ = onChunk
	_ = opts
	return ChatResult{}, notSupported("stream")
}

func (p *VertexProvider) CompleteVision(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	_ = modelID
	_ = messages
	_ = opts
	return ChatResult{}, notSupported("vision")
}

func (p *VertexProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, notSupported("embed")
}
