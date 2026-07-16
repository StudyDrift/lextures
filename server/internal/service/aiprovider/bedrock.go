package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brttypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// bedrockConverseAPI is the subset of bedrockruntime.Client used for Converse (mockable).
type bedrockConverseAPI interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

// BedrockProvider calls the AWS Bedrock Converse API.
// auth_mode=api_key uses bearer HTTP (gateways/tests); access_key/iam_role use the AWS SDK (AP.8 FR-2).
type BedrockProvider struct {
	authMode string
	client   *httpClient
	sdk      bedrockConverseAPI
	region   string
}

// NewBedrockProvider builds a Bedrock provider using bearer-token HTTP (api_key / test mode).
func NewBedrockProvider(apiKey, baseURL string) *BedrockProvider {
	headers := map[string]string{}
	if strings.TrimSpace(apiKey) != "" {
		headers["Authorization"] = "Bearer " + strings.TrimSpace(apiKey)
	}
	return &BedrockProvider{
		authMode: AuthModeAPIKey,
		client:   newHTTPClient(apiKey, baseURL, headers),
	}
}

// NewBedrockProviderWithSDK is for tests injecting a mocked Converse client.
func NewBedrockProviderWithSDK(sdk bedrockConverseAPI, region string) *BedrockProvider {
	return &BedrockProvider{
		authMode: AuthModeIAMRole,
		sdk:      sdk,
		region:   region,
	}
}

func newBedrockProviderFromAuth(mode string, auth AuthMaterial, extra map[string]any) (*BedrockProvider, error) {
	region := bedrockRegion(extra)
	switch mode {
	case AuthModeAPIKey:
		return NewBedrockProvider(auth.Secret(secretKeyAPIKey), bedrockBaseURL(extra)), nil
	case AuthModeAccessKey, AuthModeIAMRole:
		sdk, err := newBedrockRuntimeClient(context.Background(), mode, auth, region)
		if err != nil {
			return nil, err
		}
		return &BedrockProvider{authMode: mode, sdk: sdk, region: region}, nil
	default:
		return nil, newConfigError(ProviderBedrock, fmt.Sprintf("bedrock unsupported auth_mode %q", mode))
	}
}

func newBedrockRuntimeClient(ctx context.Context, mode string, auth AuthMaterial, region string) (*bedrockruntime.Client, error) {
	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(region))
	if mode == AuthModeAccessKey {
		accessKey := auth.Secret(secretKeyAWSAccessKeyID)
		secretKey := auth.Secret(secretKeyAWSSecretAccessKey)
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, newAuthError(ProviderBedrock, 0, "failed to load AWS credentials: "+err.Error())
	}
	return bedrockruntime.NewFromConfig(cfg), nil
}

func (p *BedrockProvider) Name() ProviderName { return ProviderBedrock }

func (p *BedrockProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	if p == nil {
		return ChatResult{}, newConfigError(ProviderBedrock, "bedrock not configured")
	}
	opt := firstChatOptions(opts)
	ctx, cancel := withChatTimeout(ctx, opt)
	defer cancel()

	if p.sdk != nil {
		return p.completeSDK(ctx, modelID, messages, opt)
	}
	return p.completeHTTP(ctx, modelID, messages, opt)
}

func (p *BedrockProvider) completeHTTP(ctx context.Context, modelID string, messages []Message, opt ChatOptions) (ChatResult, error) {
	if p.client == nil || p.client.baseURL == "" {
		return ChatResult{}, newConfigError(ProviderBedrock, "bedrock not configured")
	}
	client := p.client.withTimeout(opt.Timeout)
	body := bedrockConverseBody(messages, opt)
	path := "/model/" + modelID + "/converse"
	b, _, err := client.postJSON(ctx, ProviderBedrock, path, body)
	if err != nil {
		return ChatResult{}, err
	}
	return parseBedrockHTTPResponse(b)
}

func (p *BedrockProvider) completeSDK(ctx context.Context, modelID string, messages []Message, opt ChatOptions) (ChatResult, error) {
	input := &bedrockruntime.ConverseInput{
		ModelId: aws.String(modelID),
	}
	var system []brttypes.SystemContentBlock
	var msgs []brttypes.Message
	for _, m := range messages {
		switch m.Role {
		case "system":
			system = append(system, &brttypes.SystemContentBlockMemberText{Value: m.TextContent()})
		default:
			role := brttypes.ConversationRoleUser
			if m.Role == "assistant" {
				role = brttypes.ConversationRoleAssistant
			}
			msgs = append(msgs, brttypes.Message{
				Role: role,
				Content: []brttypes.ContentBlock{
					&brttypes.ContentBlockMemberText{Value: m.TextContent()},
				},
			})
		}
	}
	input.Messages = msgs
	if len(system) > 0 {
		input.System = system
	}
	inf := &brttypes.InferenceConfiguration{}
	hasInf := false
	if opt.MaxTokens > 0 {
		inf.MaxTokens = aws.Int32(int32(opt.MaxTokens))
		hasInf = true
	}
	if opt.Temperature != nil {
		inf.Temperature = aws.Float32(float32(*opt.Temperature))
		hasInf = true
	}
	if hasInf {
		input.InferenceConfig = inf
	}
	out, err := p.sdk.Converse(ctx, input)
	if err != nil {
		return ChatResult{}, classifyBedrockSDKError(err)
	}
	return parseBedrockSDKResponse(out)
}

func classifyBedrockSDKError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "accessdenied"),
		strings.Contains(lower, "unauthorized"),
		strings.Contains(lower, "invalidsignature"),
		strings.Contains(lower, "expiredtoken"),
		strings.Contains(lower, "unrecognizedclient"):
		return newAuthError(ProviderBedrock, 403, msg)
	case strings.Contains(lower, "throttl"),
		strings.Contains(lower, "too many requests"):
		return &ProviderError{Provider: ProviderBedrock, StatusCode: 429, Message: msg, Type: ErrorTypeQuota}
	case strings.Contains(lower, "validation"),
		strings.Contains(lower, "resourcenotfound"),
		strings.Contains(lower, "model not ready"):
		return &ProviderError{Provider: ProviderBedrock, StatusCode: 400, Message: msg, Type: ErrorTypeConfig}
	default:
		return &ProviderError{Provider: ProviderBedrock, StatusCode: 502, Message: msg, Type: ErrorTypeServer}
	}
}

func bedrockConverseBody(messages []Message, opt ChatOptions) map[string]any {
	var system []map[string]any
	var msgs []map[string]any
	for _, m := range messages {
		switch m.Role {
		case "system":
			system = append(system, map[string]any{"text": m.TextContent()})
		default:
			msgs = append(msgs, map[string]any{
				"role":    m.Role,
				"content": []map[string]string{{"text": m.TextContent()}},
			})
		}
	}
	body := map[string]any{"messages": msgs}
	if len(system) > 0 {
		body["system"] = system
	}
	if opt.MaxTokens > 0 {
		body["inferenceConfig"] = map[string]any{"maxTokens": opt.MaxTokens}
	}
	if opt.Temperature != nil {
		cfg, _ := body["inferenceConfig"].(map[string]any)
		if cfg == nil {
			cfg = map[string]any{}
		}
		cfg["temperature"] = *opt.Temperature
		body["inferenceConfig"] = cfg
	}
	return body
}

func parseBedrockHTTPResponse(b []byte) (ChatResult, error) {
	var parsed struct {
		Output struct {
			Message struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		} `json:"output"`
		Usage struct {
			InputTokens  int `json:"inputTokens"`
			OutputTokens int `json:"outputTokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return ChatResult{}, fmt.Errorf("aiprovider: parse bedrock response: %w", err)
	}
	var text string
	for _, block := range parsed.Output.Message.Content {
		if block.Text != "" {
			text = block.Text
			break
		}
	}
	return ChatResult{
		Text: text,
		Usage: UsageInfo{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
			TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}, nil
}

func parseBedrockSDKResponse(out *bedrockruntime.ConverseOutput) (ChatResult, error) {
	if out == nil {
		return ChatResult{}, fmt.Errorf("aiprovider: empty bedrock response")
	}
	var text string
	if out.Output != nil {
		if msg, ok := out.Output.(*brttypes.ConverseOutputMemberMessage); ok {
			for _, block := range msg.Value.Content {
				if t, ok := block.(*brttypes.ContentBlockMemberText); ok && t.Value != "" {
					text = t.Value
					break
				}
			}
		}
	}
	usage := UsageInfo{}
	if out.Usage != nil {
		if out.Usage.InputTokens != nil {
			usage.PromptTokens = int(*out.Usage.InputTokens)
		}
		if out.Usage.OutputTokens != nil {
			usage.CompletionTokens = int(*out.Usage.OutputTokens)
		}
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return ChatResult{Text: text, Usage: usage}, nil
}

func (p *BedrockProvider) CompleteStream(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	_ = modelID
	_ = messages
	_ = onChunk
	_ = opts
	return ChatResult{}, notSupported("stream")
}

func (p *BedrockProvider) CompleteVision(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	_ = modelID
	_ = messages
	_ = opts
	return ChatResult{}, notSupported("vision")
}

func (p *BedrockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, notSupported("embed")
}
