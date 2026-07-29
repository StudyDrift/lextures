package inlinequestionsai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_questions"
)

// PromptKey is the settings.system_prompts row used when present.
const PromptKey = "inline_questions_generation"

// MaxPageMarkdownRunes caps host page body context.
const MaxPageMarkdownRunes = 80_000

// MaxQuestions is the Inline Questions config limit (manifest maxItems).
const MaxQuestions = 3

// DefaultSystemPrompt matches settings.system_prompts key inline_questions_generation when the row is missing.
const DefaultSystemPrompt = `You generate low-stakes formative "inline questions" (checks for understanding) for an LMS content tool.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object with camelCase keys:
{
  "label": string optional (short heading shown above the check, e.g. "Check your understanding"),
  "questions": [
    {
      "type": "single" | "multi" | "true_false" | "short_text" | "numeric",
      "prompt": string (required, the question stem),
      "options": [ // required for single, multi, true_false
        { "text": string, "correct": boolean, "feedback": string optional }
      ],
      "acceptedAnswers": [string] // required for short_text (1–5 accepted spellings/synonyms),
      "correctValue": number // required for numeric,
      "tolerance": { "kind": "absolute" | "relative", "value": number } // optional for numeric,
      "unit": string optional // display-only unit for numeric,
      "explanation": string optional // general explanation after scoring,
      "points": number optional // default 1
    }
  ]
}

Rules:
- Return 1 to 3 questions that check understanding of the provided page/assignment content only.
- Prefer a mix of types when the material supports it (e.g. one single-choice + one short_text).
- For single: exactly one option with correct=true; supply 3–4 options with brief feedback on wrong options when helpful.
- For multi: at least one correct option; 3–5 options.
- For true_false: exactly two options with text "True" and "False" (in that order); mark one correct.
- For short_text: provide acceptedAnswers; do not invent obscure synonyms.
- For numeric: provide correctValue; use a small absolute tolerance when values are approximate.
- Prompts must be answerable from the provided content without external knowledge.
- Keep language clear and age-appropriate for the material.
- If the content is empty or too thin to write fair questions, return {"questions":[]}.`

// Service provides AI-backed Inline Questions drafting.
type Service struct {
	Name string
}

func New() Service {
	return Service{Name: "inlinequestionsai"}
}

// Health returns a stable service heartbeat string for wiring/tests.
func (s Service) Health(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}
	return s.Name + ":ok", nil
}

// GenerateInput is the host page context for a draft.
type GenerateInput struct {
	PageTitle    string
	PageMarkdown string
	QuestionCount int // 1–3; default 2
}

// GenerateResult is draft config fields (not persisted).
type GenerateResult struct {
	Label     string                      `json:"label,omitempty"`
	Questions []inline_questions.Question `json:"questions"`
}

type aiEnvelope struct {
	Label     *string       `json:"label"`
	Questions []aiQuestion  `json:"questions"`
}

type aiQuestion struct {
	Type            string     `json:"type"`
	Prompt          string     `json:"prompt"`
	Options         []aiOption `json:"options"`
	AcceptedAnswers []string   `json:"acceptedAnswers"`
	CorrectValue    *float64   `json:"correctValue"`
	Tolerance       *struct {
		Kind  string  `json:"kind"`
		Value float64 `json:"value"`
	} `json:"tolerance"`
	Unit        string   `json:"unit"`
	Explanation string   `json:"explanation"`
	Points      *float64 `json:"points"`
}

type aiOption struct {
	Text     string `json:"text"`
	Correct  bool   `json:"correct"`
	Feedback string `json:"feedback"`
}

// Generate asks the model for draft Inline Questions from host page content.
func Generate(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt string,
	input GenerateInput,
) (*GenerateResult, aiprovider.CallMeta, error) {
	md := strings.TrimSpace(input.PageMarkdown)
	if md == "" {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("page content is required")
	}
	if utf8.RuneCountInString(md) > MaxPageMarkdownRunes {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("page content is too long (max %d characters)", MaxPageMarkdownRunes)
	}

	count := input.QuestionCount
	if count < 1 {
		count = 2
	}
	if count > MaxQuestions {
		count = MaxQuestions
	}

	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}

	title := strings.TrimSpace(input.PageTitle)
	if title == "" {
		title = "(untitled)"
	}

	userBody := fmt.Sprintf(
		"Create exactly %d formative inline questions grounded in this page/assignment.\n\nPage title: %s\n\nPage content (Markdown):\n---\n%s\n---\n\nRespond with ONLY a JSON object as described in your system instructions (camelCase).",
		count,
		title,
		md,
	)

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: userBody},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return nil, meta, err
	}
	text := strings.TrimSpace(res.Text)
	if text == "" {
		return nil, meta, fmt.Errorf("the model returned an empty response")
	}

	out, err := parseModelJSON(text)
	if err != nil {
		return nil, meta, err
	}
	return out, meta, nil
}

func parseModelJSON(text string) (*GenerateResult, error) {
	slice := extractJSONObject(text)
	if slice == "" {
		return nil, fmt.Errorf("could not find JSON in the model response")
	}
	var raw aiEnvelope
	if err := json.Unmarshal([]byte(slice), &raw); err != nil {
		return nil, fmt.Errorf("parse inline questions JSON: %w", err)
	}
	return normalize(raw), nil
}

func extractJSONObject(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```JSON")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return ""
	}
	return s[start : end+1]
}

func normalize(raw aiEnvelope) *GenerateResult {
	out := &GenerateResult{Questions: make([]inline_questions.Question, 0, MaxQuestions)}
	if raw.Label != nil {
		if t := strings.TrimSpace(*raw.Label); t != "" {
			out.Label = t
		}
	}
	for _, aq := range raw.Questions {
		q, ok := normalizeQuestion(aq)
		if !ok {
			continue
		}
		out.Questions = append(out.Questions, q)
		if len(out.Questions) >= MaxQuestions {
			break
		}
	}
	return out
}

func normalizeQuestion(aq aiQuestion) (inline_questions.Question, bool) {
	prompt := strings.TrimSpace(aq.Prompt)
	if prompt == "" {
		return inline_questions.Question{}, false
	}
	typ := inline_questions.QuestionType(strings.TrimSpace(aq.Type))
	switch typ {
	case inline_questions.TypeSingle, inline_questions.TypeMulti, inline_questions.TypeTrueFalse,
		inline_questions.TypeShortText, inline_questions.TypeNumeric:
	default:
		typ = inline_questions.TypeSingle
	}

	q := inline_questions.Question{
		ID:          "q_" + shortID(),
		Type:        typ,
		Prompt:      prompt,
		Explanation: strings.TrimSpace(aq.Explanation),
		Unit:        strings.TrimSpace(aq.Unit),
		Points:      1,
	}
	if aq.Points != nil && *aq.Points > 0 {
		q.Points = *aq.Points
	}

	switch typ {
	case inline_questions.TypeTrueFalse:
		trueCorrect := false
		falseCorrect := true
		// Prefer model marking; default True correct when unclear.
		sawCorrect := false
		for _, o := range aq.Options {
			text := strings.TrimSpace(strings.ToLower(o.Text))
			if text == "true" && o.Correct {
				trueCorrect = true
				falseCorrect = false
				sawCorrect = true
			}
			if text == "false" && o.Correct {
				trueCorrect = false
				falseCorrect = true
				sawCorrect = true
			}
		}
		if !sawCorrect {
			trueCorrect = true
			falseCorrect = false
		}
		var trueFB, falseFB string
		for _, o := range aq.Options {
			text := strings.TrimSpace(strings.ToLower(o.Text))
			if text == "true" {
				trueFB = strings.TrimSpace(o.Feedback)
			}
			if text == "false" {
				falseFB = strings.TrimSpace(o.Feedback)
			}
		}
		q.Options = []inline_questions.Option{
			{ID: "true", Text: "True", Correct: trueCorrect, Feedback: trueFB},
			{ID: "false", Text: "False", Correct: falseCorrect, Feedback: falseFB},
		}

	case inline_questions.TypeSingle, inline_questions.TypeMulti:
		opts := make([]inline_questions.Option, 0, len(aq.Options))
		correctCount := 0
		for _, o := range aq.Options {
			text := strings.TrimSpace(o.Text)
			if text == "" {
				continue
			}
			opt := inline_questions.Option{
				ID:       "opt_" + shortID(),
				Text:     text,
				Correct:  o.Correct,
				Feedback: strings.TrimSpace(o.Feedback),
			}
			if opt.Correct {
				correctCount++
			}
			opts = append(opts, opt)
		}
		if len(opts) < 2 {
			return inline_questions.Question{}, false
		}
		if typ == inline_questions.TypeSingle {
			if correctCount == 0 {
				opts[0].Correct = true
			} else if correctCount > 1 {
				// Keep first correct only.
				seen := false
				for i := range opts {
					if opts[i].Correct {
						if seen {
							opts[i].Correct = false
						}
						seen = true
					}
				}
			}
		} else if correctCount == 0 {
			opts[0].Correct = true
		}
		q.Options = opts

	case inline_questions.TypeShortText:
		answers := make([]string, 0, len(aq.AcceptedAnswers))
		for _, a := range aq.AcceptedAnswers {
			t := strings.TrimSpace(a)
			if t != "" {
				answers = append(answers, t)
			}
		}
		if len(answers) == 0 {
			return inline_questions.Question{}, false
		}
		if len(answers) > 10 {
			answers = answers[:10]
		}
		q.AcceptedAnswers = answers

	case inline_questions.TypeNumeric:
		if aq.CorrectValue == nil {
			return inline_questions.Question{}, false
		}
		v := *aq.CorrectValue
		q.CorrectValue = &v
		if aq.Tolerance != nil && aq.Tolerance.Value >= 0 {
			kind := inline_questions.ToleranceAbsolute
			if aq.Tolerance.Kind == string(inline_questions.ToleranceRelative) {
				kind = inline_questions.ToleranceRelative
			}
			q.Tolerance = &inline_questions.Tolerance{Kind: kind, Value: aq.Tolerance.Value}
		}
	}

	return q, true
}

func shortID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
}
