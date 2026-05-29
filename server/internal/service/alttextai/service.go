// Package alttextai generates alt-text suggestions via OpenRouter vision models (plan 12.5).
package alttextai

import (
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/aitutor"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

const (
	DefaultModel = "openai/gpt-4o"
	systemPrompt = "Describe this image in 1–2 sentences suitable as screen-reader alt text for educational content. Return only the alt text, no quotes or preamble."
)

// Suggest returns an alt-text suggestion and a confidence score in [0,1].
func Suggest(client *openrouter.Client, model, imageURL, courseLanguage string) (string, float64, error) {
	if client == nil {
		return "", 0, fmt.Errorf("alttextai: nil client")
	}
	m := strings.TrimSpace(model)
	if m == "" {
		m = DefaultModel
	}
	url := strings.TrimSpace(imageURL)
	if url == "" {
		return "", 0, fmt.Errorf("alttextai: missing image URL")
	}
	langHint := strings.TrimSpace(courseLanguage)
	userText := "Generate alt text for this image."
	if langHint != "" {
		userText = fmt.Sprintf("Generate alt text for this image in %s.", langHint)
	}
	userText = aitutor.RedactPII(userText)
	text, err := client.VisionCompletion(m, systemPrompt, userText, url)
	if err != nil {
		return "", 0, err
	}
	suggestion := strings.TrimSpace(text)
	if suggestion == "" {
		return "", 0, fmt.Errorf("alttextai: empty suggestion")
	}
	suggestion = aitutor.RedactPII(suggestion)
	confidence := 0.85
	if len(suggestion) < 8 {
		confidence = 0.5
	}
	return suggestion, confidence, nil
}
