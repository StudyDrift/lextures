// Package contentsimplificationai rewrites instructor content to a target reading level via LLM.
package contentsimplificationai

import (
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/openrouter"
)

const DefaultModel = "openai/gpt-4o-mini"

// Simplify asks the LLM to rewrite text to the target Flesch-Kincaid grade level.
func Simplify(client *openrouter.Client, model, text string, targetGrade int) (string, error) {
	if client == nil {
		return "", fmt.Errorf("contentsimplificationai: nil client")
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("contentsimplificationai: empty text")
	}
	if model == "" {
		model = DefaultModel
	}
	system := fmt.Sprintf(
		"Rewrite the following text to a Flesch-Kincaid Grade Level of %d. "+
			"Preserve all factual content. Use short sentences and common words. "+
			"Return only the rewritten text with no preamble.",
		targetGrade,
	)
	out, err := client.ChatCompletion(model, []openrouter.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: text},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
