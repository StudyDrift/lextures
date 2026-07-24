package contentpagegeneration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// MaxPromptRunes caps the instructor prompt sent to the model.
const MaxPromptRunes = 8_000

// MaxExistingMarkdownRunes caps optional existing draft markdown sent for revision context.
const MaxExistingMarkdownRunes = 80_000

// MaxSections caps how many draft sections are returned.
const MaxSections = 20

// MaxHeadingRunes / MaxMarkdownRunes limit individual draft fields.
const (
	MaxHeadingRunes  = 200
	MaxMarkdownRunes = 20_000
)

// DraftSection is a proposed content-page section (not persisted).
type DraftSection struct {
	Heading  string `json:"heading"`
	Markdown string `json:"markdown"`
}

// DefaultSystemPrompt instructs the model to return structured section JSON only.
const DefaultSystemPrompt = `You write course content pages for an LMS block editor.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object: {"sections":[{"heading":"...","markdown":"..."}]}.

Rules:
- Produce learner-facing instructional content that matches the instructor's topic description.
- heading: short section title without markdown # prefixes; use "" for a lead-in block with no heading.
- markdown: body content in Markdown only (paragraphs, lists, emphasis, links). Do NOT put ## headings inside markdown — use separate section objects instead.
- Prefer 2–8 clear sections when the topic supports it; return between 1 and 20 sections.
- Write in a professional, accessible tone suitable for students.
- If existing draft content is provided, revise or restructure it to fit the prompt rather than ignoring it.
- If the prompt has no usable topic, return {"sections":[]}.`

// GenerateFromPrompt asks the model for draft content-page sections.
func GenerateFromPrompt(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt, prompt, existingMarkdown, pageTitle string,
) ([]DraftSection, aiprovider.CallMeta, error) {
	p := strings.TrimSpace(prompt)
	if p == "" {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("prompt is required")
	}
	if utf8.RuneCountInString(p) > MaxPromptRunes {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("prompt is too long (max %d characters)", MaxPromptRunes)
	}
	existing := strings.TrimSpace(existingMarkdown)
	if utf8.RuneCountInString(existing) > MaxExistingMarkdownRunes {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("existing markdown is too long (max %d characters)", MaxExistingMarkdownRunes)
	}
	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}

	var user strings.Builder
	if title := strings.TrimSpace(pageTitle); title != "" {
		fmt.Fprintf(&user, "Page title: %s\n\n", title)
	}
	fmt.Fprintf(&user, "Instructor description of the content:\n%s", p)
	if existing != "" {
		fmt.Fprintf(&user, "\n\nExisting draft content to revise or replace:\n%s", existing)
	}

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user.String()},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return nil, meta, err
	}
	sections, err := ParseDraftSectionsJSON(res.Text)
	if err != nil {
		return nil, meta, err
	}
	return sections, meta, nil
}

// ParseDraftSectionsJSON parses and normalizes model JSON into draft sections.
func ParseDraftSectionsJSON(raw string) ([]DraftSection, error) {
	text := stripJSONFences(raw)
	var payload struct {
		Sections []DraftSection `json:"sections"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, fmt.Errorf("parse content page sections JSON: %w", err)
	}
	return normalizeDraftSections(payload.Sections), nil
}

func stripJSONFences(raw string) string {
	text := strings.TrimSpace(raw)
	if idx := strings.Index(text, "```json"); idx != -1 {
		text = text[idx+7:]
		if endIdx := strings.Index(text, "```"); endIdx != -1 {
			text = text[:endIdx]
		}
	} else if idx := strings.Index(text, "```"); idx != -1 {
		text = text[idx+3:]
		if endIdx := strings.Index(text, "```"); endIdx != -1 {
			text = text[:endIdx]
		}
	}
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "{") {
		if start := strings.Index(text, "{"); start != -1 {
			if end := strings.LastIndex(text, "}"); end > start {
				text = text[start : end+1]
			}
		}
	}
	return strings.TrimSpace(text)
}

func normalizeDraftSections(in []DraftSection) []DraftSection {
	out := make([]DraftSection, 0, len(in))
	for _, s := range in {
		heading := strings.TrimSpace(s.Heading)
		heading = strings.TrimLeft(heading, "#")
		heading = strings.TrimSpace(heading)
		if utf8.RuneCountInString(heading) > MaxHeadingRunes {
			heading = string([]rune(heading)[:MaxHeadingRunes])
		}
		md := strings.TrimSpace(s.Markdown)
		if utf8.RuneCountInString(md) > MaxMarkdownRunes {
			md = string([]rune(md)[:MaxMarkdownRunes])
		}
		if heading == "" && md == "" {
			continue
		}
		out = append(out, DraftSection{Heading: heading, Markdown: md})
		if len(out) >= MaxSections {
			break
		}
	}
	return out
}
