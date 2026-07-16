package aiprovider

import (
	"context"
	"time"
)

func firstChatOptions(opts []ChatOptions) ChatOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return ChatOptions{}
}

func firstImageOptions(opts []ImageOptions) ImageOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return ImageOptions{}
}

// withChatTimeout derives a child context when ChatOptions.Timeout is set.
func withChatTimeout(ctx context.Context, opt ChatOptions) (context.Context, context.CancelFunc) {
	if opt.Timeout > 0 {
		return context.WithTimeout(ctx, opt.Timeout)
	}
	return ctx, func() {}
}

func effectiveMaxTokens(opt ChatOptions, defaultN int) int {
	if opt.MaxTokens > 0 {
		return opt.MaxTokens
	}
	return defaultN
}

func applyTemperature(body map[string]any, opt ChatOptions) {
	if opt.Temperature != nil {
		body["temperature"] = *opt.Temperature
	}
}

// ensureJSONSystem appends a JSON-only instruction to a system prompt when JSONMode is set.
func ensureJSONSystem(system string, jsonMode bool) string {
	if !jsonMode {
		return system
	}
	const instr = "Respond with valid JSON only."
	if system == "" {
		return instr
	}
	return system + "\n\n" + instr
}

const defaultHardTimeout = 120 * time.Second
