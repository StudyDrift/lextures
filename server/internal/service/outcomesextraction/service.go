package outcomesextraction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// MaxSyllabusRunes caps syllabus material sent to the model.
const MaxSyllabusRunes = 80_000

// MaxOutcomes caps how many draft outcomes are returned.
const MaxOutcomes = 30

// MaxTitleRunes / MaxDescriptionRunes limit individual draft fields.
const (
	MaxTitleRunes       = 500
	MaxDescriptionRunes = 4000
)

// DraftOutcome is a proposed learning outcome (not persisted).
type DraftOutcome struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// DefaultSystemPrompt instructs the model to return structured outcome JSON only.
const DefaultSystemPrompt = `You extract learning outcomes from a course syllabus for an LMS.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object: {"outcomes":[{"title":"...","description":"..."}]}.

Rules:
- Extract measurable learner-facing learning outcomes (what students will be able to do or demonstrate).
- Prefer outcomes already stated in the syllabus (learning objectives, goals, competencies). If none are explicit, infer a concise set from the course description and topics — do not invent unrelated topics.
- title: short outcome statement (typically starts with a verb such as Analyze, Apply, Explain).
- description: optional brief elaboration; use "" when the title is sufficient.
- Return between 1 and 30 outcomes. Prefer quality over quantity.
- If the syllabus has no usable content for outcomes, return {"outcomes":[]}.`

// SyllabusPromptMaterial concatenates syllabus sections into markdown for the model.
func SyllabusPromptMaterial(sections []course.SyllabusSection) string {
	var b strings.Builder
	for _, s := range sections {
		heading := strings.TrimSpace(s.Heading)
		md := strings.TrimSpace(s.Markdown)
		if heading == "" && md == "" {
			continue
		}
		if heading != "" {
			fmt.Fprintf(&b, "## %s\n\n", heading)
		}
		if md != "" {
			b.WriteString(md)
			b.WriteString("\n\n")
		}
	}
	return strings.TrimSpace(b.String())
}

// ExtractFromSyllabus asks the model for draft outcomes from syllabus markdown.
func ExtractFromSyllabus(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt, syllabusMarkdown string,
) ([]DraftOutcome, aiprovider.CallMeta, error) {
	md := strings.TrimSpace(syllabusMarkdown)
	if md == "" {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("syllabus content is required")
	}
	if utf8.RuneCountInString(md) > MaxSyllabusRunes {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("syllabus is too long (max %d characters)", MaxSyllabusRunes)
	}
	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}
	user := "Extract learning outcomes from this course syllabus:\n\n" + md
	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return nil, meta, err
	}
	outcomes, err := ParseDraftOutcomesJSON(res.Text)
	if err != nil {
		return nil, meta, err
	}
	return outcomes, meta, nil
}

// ParseDraftOutcomesJSON parses and normalizes model JSON into draft outcomes.
func ParseDraftOutcomesJSON(raw string) ([]DraftOutcome, error) {
	text := stripJSONFences(raw)
	var payload struct {
		Outcomes []DraftOutcome `json:"outcomes"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, fmt.Errorf("parse outcomes JSON: %w", err)
	}
	return normalizeDraftOutcomes(payload.Outcomes), nil
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

func normalizeDraftOutcomes(in []DraftOutcome) []DraftOutcome {
	out := make([]DraftOutcome, 0, len(in))
	seen := make(map[string]struct{})
	for _, o := range in {
		title := strings.TrimSpace(o.Title)
		if title == "" {
			continue
		}
		if utf8.RuneCountInString(title) > MaxTitleRunes {
			title = string([]rune(title)[:MaxTitleRunes])
		}
		desc := strings.TrimSpace(o.Description)
		if utf8.RuneCountInString(desc) > MaxDescriptionRunes {
			desc = string([]rune(desc)[:MaxDescriptionRunes])
		}
		key := strings.ToLower(title)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, DraftOutcome{Title: title, Description: desc})
		if len(out) >= MaxOutcomes {
			break
		}
	}
	return out
}
