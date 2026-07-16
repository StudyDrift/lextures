// Package contentsimplificationai rewrites instructor content to a target reading level via LLM.
package contentsimplificationai

import (
	"context"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const DefaultModel = "openai/gpt-4o-mini"

// Simplify asks the LLM to rewrite text to the target Flesch-Kincaid grade level.
func Simplify(ctx context.Context, ai aiprovider.ScopedCompleter, model, text string, targetGrade int) (string, aiprovider.CallMeta, error) {
	if ai == nil {
		return "", aiprovider.CallMeta{}, fmt.Errorf("contentsimplificationai: nil completer")
	}
	if strings.TrimSpace(text) == "" {
		return "", aiprovider.CallMeta{}, fmt.Errorf("contentsimplificationai: empty text")
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
	out, meta, err := ai.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: text},
	})
	if err != nil {
		return "", meta, err
	}
	return strings.TrimSpace(out.Text), meta, nil
}
