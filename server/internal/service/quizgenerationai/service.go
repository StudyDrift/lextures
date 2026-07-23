package quizgenerationai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// Service provides AI-backed quiz question generation and markdown import.
type Service struct {
	Name string
}

func New() Service {
	return Service{Name: "quizgenerationai"}
}

// Health returns a stable service heartbeat string for wiring/tests.
func (s Service) Health(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}
	return s.Name + ":ok", nil
}

// MaxMarkdownImportRunes caps pasted markdown size for AI import.
const MaxMarkdownImportRunes = 80_000

// DefaultSystemPrompt matches settings.system_prompts key quiz_generation when the row is missing.
const DefaultSystemPrompt = `You generate quiz questions for an LMS. You respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object: {"questions":[...]}.

Each question object uses camelCase keys and must match this app schema:
- prompt (string, required)
- questionType (string, required): one of exactly: multiple_choice, fill_in_blank, essay, true_false, short_answer
- choices (array of strings): for multiple_choice supply 3–5 distinct options; for true_false use ["True","False"] in that order; for fill_in_blank, essay, short_answer use []
- correctChoiceIndex (number or null): for multiple_choice and true_false, 0-based index into choices when a single best answer exists; otherwise null
- multipleAnswer (boolean, default false)
- answerWithImage (boolean, default false)
- required (boolean, default true)
- points (integer, default 1)
- estimatedMinutes (integer, default 2)

Rules:
- Use a mix of question types across the batch when the requested count allows (at least two different types when count >= 2).
- Keep prompts clear and appropriate for the instructor topic.
- For multiple_choice, ensure correctChoiceIndex refers to a valid choice index when set.`

// DefaultMarkdownImportSystemPrompt instructs the model to parse author markdown into quiz JSON.
const DefaultMarkdownImportSystemPrompt = `You convert instructor-authored quiz questions written in Markdown into Lextures quiz JSON.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object: {"questions":[...]}.

Each question object uses camelCase keys and must match this app schema:
- prompt (string, required) — the question stem; may include inline markdown when helpful
- questionType (string, required): one of exactly: multiple_choice, fill_in_blank, essay, true_false, short_answer, matching, ordering, numeric
- choices (array of strings): for multiple_choice supply the options in order; for true_false use ["True","False"]; otherwise []
- correctChoiceIndex (number or null): 0-based index into choices when a single best answer is clear; otherwise null
- multipleAnswer (boolean, default false) — true when several choices are correct
- answerWithImage (boolean, default false)
- required (boolean, default true)
- points (integer, default 1)
- estimatedMinutes (integer, default 2)
- typeConfig (object, optional) — use for matching pairs, ordering items, numeric tolerance, etc. when the markdown implies them

Rules:
- Preserve the author's intent, wording, and correct answers when present.
- Infer questionType from structure (numbered lists with A/B/C, True/False, blanks, etc.).
- Do not invent topics that are not in the markdown; only parse what is provided.
- Skip decorative headings that are not questions.
- If the markdown is empty or contains no questions, return {"questions":[]}.`

// GenerateFromPrompt asks the model for N new questions from a free-form topic prompt.
func GenerateFromPrompt(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt, userPrompt string,
	questionCount int,
) ([]coursemodulequiz.QuizQuestion, aiprovider.CallMeta, error) {
	if questionCount < 1 {
		questionCount = 1
	}
	if questionCount > 30 {
		questionCount = 30
	}
	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}
	user := fmt.Sprintf(
		"Create exactly %d quiz questions for this instructor request:\n\n%s",
		questionCount,
		strings.TrimSpace(userPrompt),
	)
	return completeQuestions(ctx, client, model, sys, user)
}

// ParseMarkdown asks the model to turn pasted markdown into QuizQuestion objects.
func ParseMarkdown(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt, markdown string,
) ([]coursemodulequiz.QuizQuestion, aiprovider.CallMeta, error) {
	md := strings.TrimSpace(markdown)
	if md == "" {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("markdown is required")
	}
	if utf8.RuneCountInString(md) > MaxMarkdownImportRunes {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("markdown is too long (max %d characters)", MaxMarkdownImportRunes)
	}
	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultMarkdownImportSystemPrompt
	}
	user := "Parse the following markdown into quiz questions JSON:\n\n" + md
	return completeQuestions(ctx, client, model, sys, user)
}

func completeQuestions(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, sysPrompt, user string,
) ([]coursemodulequiz.QuizQuestion, aiprovider.CallMeta, error) {
	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: user},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return nil, meta, err
	}
	text := stripJSONFences(res.Text)
	var payload struct {
		Questions []coursemodulequiz.QuizQuestion `json:"questions"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, meta, fmt.Errorf("parse quiz JSON: %w", err)
	}
	normalized := normalizeQuestions(payload.Questions)
	return normalized, meta, nil
}

func stripJSONFences(raw string) string {
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```JSON")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}

func normalizeQuestions(in []coursemodulequiz.QuizQuestion) []coursemodulequiz.QuizQuestion {
	allowed := make(map[string]struct{}, len(coursemodulequiz.QuizQuestionTypes))
	for _, t := range coursemodulequiz.QuizQuestionTypes {
		allowed[t] = struct{}{}
	}
	out := make([]coursemodulequiz.QuizQuestion, 0, len(in))
	for _, q := range in {
		prompt := strings.TrimSpace(q.Prompt)
		if prompt == "" {
			continue
		}
		q.Prompt = prompt
		if q.ID == "" {
			q.ID = uuid.New().String()
		}
		if _, ok := allowed[q.QuestionType]; !ok || q.QuestionType == "" {
			q.QuestionType = "short_answer"
		}
		if q.Choices == nil {
			q.Choices = []string{}
		}
		if q.ChoiceIDs == nil {
			q.ChoiceIDs = []string{}
		}
		if len(q.TypeConfig) == 0 {
			q.TypeConfig = json.RawMessage(`{}`)
		}
		if q.QuestionType == "true_false" && len(q.Choices) == 0 {
			q.Choices = []string{"True", "False"}
		}
		if q.Points <= 0 {
			q.Points = 1
		}
		if q.EstimatedMinutes <= 0 {
			q.EstimatedMinutes = 2
		}
		q.Required = true
		if q.CorrectChoiceIndex != nil {
			idx := int(*q.CorrectChoiceIndex)
			if idx < 0 || idx >= len(q.Choices) {
				q.CorrectChoiceIndex = nil
			}
		}
		out = append(out, q)
		if len(out) >= coursemodulequiz.MaxQuizQuestions {
			break
		}
	}
	return out
}
