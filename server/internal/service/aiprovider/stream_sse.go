package aiprovider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// parseOpenAICompatibleSSE reads an OpenAI/OpenRouter-style SSE body and invokes onChunk
// for each content delta. Returns concatenated text and any usage from the stream.
func parseOpenAICompatibleSSE(body io.Reader, onChunk ChunkHandler) (ChatResult, error) {
	var sb strings.Builder
	var usage UsageInfo
	scanner := bufio.NewScanner(body)
	// Allow larger SSE frames (vision/tool payloads can exceed the default 64KiB token).
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(payload) == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content *string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int     `json:"prompt_tokens"`
				CompletionTokens int     `json:"completion_tokens"`
				TotalTokens      int     `json:"total_tokens"`
				Cost             float64 `json:"cost"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		u := UsageInfo{
			PromptTokens:     chunk.Usage.PromptTokens,
			CompletionTokens: chunk.Usage.CompletionTokens,
			TotalTokens:      chunk.Usage.TotalTokens,
			CostUSD:          chunk.Usage.Cost,
		}
		if u.TotalTokens == 0 {
			u.TotalTokens = u.PromptTokens + u.CompletionTokens
		}
		if u.HasData() {
			usage = u
		}
		if len(chunk.Choices) == 0 || chunk.Choices[0].Delta.Content == nil {
			continue
		}
		text := *chunk.Choices[0].Delta.Content
		if text == "" {
			continue
		}
		sb.WriteString(text)
		if onChunk != nil {
			if err := onChunk(text); err != nil {
				return ChatResult{Text: sb.String(), Usage: usage}, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ChatResult{Text: sb.String(), Usage: usage}, fmt.Errorf("aiprovider: scan stream: %w", err)
	}
	return ChatResult{Text: sb.String(), Usage: usage}, nil
}

func streamOpenAICompatible(
	ctx context.Context,
	client *httpClient,
	provider ProviderName,
	path string,
	body map[string]any,
	onChunk ChunkHandler,
) (ChatResult, error) {
	if client == nil {
		return ChatResult{}, fmt.Errorf("aiprovider: nil http client")
	}
	body["stream"] = true
	res, err := client.postJSONRaw(ctx, provider, path, body)
	if err != nil {
		return ChatResult{}, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		msg := string(b)
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		return ChatResult{}, newProviderError(provider, res.StatusCode, msg)
	}
	wrapped := onChunk
	if ctx != nil {
		wrapped = func(text string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if onChunk == nil {
				return nil
			}
			return onChunk(text)
		}
	}
	return parseOpenAICompatibleSSE(res.Body, wrapped)
}
