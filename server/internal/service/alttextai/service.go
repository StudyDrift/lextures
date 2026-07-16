// Package alttextai generates alt-text suggestions via vision-capable AI models (plan 12.5).
package alttextai

import (
	"context"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/aitutor"
)

const (
	DefaultModel = "openai/gpt-4o"
	systemPrompt = "Describe this image in 1–2 sentences suitable as screen-reader alt text for educational content. Return only the alt text, no quotes or preamble."
)

// Suggest returns an alt-text suggestion and a confidence score in [0,1].
func Suggest(ctx context.Context, ai aiprovider.ScopedVisionCompleter, model, imageURL, courseLanguage string) (string, float64, aiprovider.CallMeta, error) {
	if ai == nil {
		return "", 0, aiprovider.CallMeta{}, fmt.Errorf("alttextai: nil completer")
	}
	m := strings.TrimSpace(model)
	if m == "" {
		m = DefaultModel
	}
	url := strings.TrimSpace(imageURL)
	if url == "" {
		return "", 0, aiprovider.CallMeta{}, fmt.Errorf("alttextai: missing image URL")
	}
	langHint := strings.TrimSpace(courseLanguage)
	userText := "Generate alt text for this image."
	if langHint != "" {
		userText = fmt.Sprintf("Generate alt text for this image in %s.", langHint)
	}
	userText = aitutor.RedactPII(userText)
	text, meta, err := ai.CompleteVision(ctx, m, aiprovider.VisionMessages(systemPrompt, userText, []string{url}))
	if err != nil {
		return "", 0, meta, err
	}
	suggestion := strings.TrimSpace(text.Text)
	if suggestion == "" {
		return "", 0, meta, fmt.Errorf("alttextai: empty suggestion")
	}
	suggestion = aitutor.RedactPII(suggestion)
	confidence := 0.85
	if len(suggestion) < 8 {
		confidence = 0.5
	}
	return suggestion, confidence, meta, nil
}
