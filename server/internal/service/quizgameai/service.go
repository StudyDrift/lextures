// Package quizgameai drafts live-quiz kit questions via the AP provider layer (plan IQ.10).
package quizgameai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/aitutor"
)

const (
	PromptKey     = "live_quiz_kit_generation"
	DefaultPrompt = `You generate draft quiz questions for a live classroom game.
Respond with ONLY valid JSON: {"questions":[...],"suggestedSubject":"...","suggestedGradeBand":"..."}.
Each question uses camelCase: questionType, prompt, options[{id,text,isCorrect}], correctAnswer, timeLimitSeconds, explanation, confidence (0-1).
Allowed questionType values: mc_single, mc_multiple, true_false, type_answer, numeric, poll, ordering, word_cloud.
Ground answers in any provided passage/content. Never invent unsupported facts for grounded sources.`
	LowConfidenceThreshold = 0.55
)

// SourceMaterial is the redacted generation context.
type SourceMaterial struct {
	SourceType   string
	Topic        string
	Passage      string
	ContentID    string
	ContentTitle string
	LikePrompt   string
	LikeType     string
}

// DraftQuestion is one model output item before IQ.2 insertion.
type DraftQuestion struct {
	QuestionType     string            `json:"questionType"`
	Prompt           string            `json:"prompt"`
	Options          []quizgame.Option `json:"options"`
	CorrectAnswer    json.RawMessage   `json:"correctAnswer"`
	TimeLimitSeconds int               `json:"timeLimitSeconds"`
	Explanation      *string           `json:"explanation"`
	Confidence       *float64          `json:"confidence"`
	Difficulty       string            `json:"difficulty"`
}

// ModelPayload is the top-level JSON object from the model.
type ModelPayload struct {
	Questions          []DraftQuestion `json:"questions"`
	SuggestedSubject   string          `json:"suggestedSubject"`
	SuggestedGradeBand string          `json:"suggestedGradeBand"`
}

// ParseResult is the outcome of parsing + validating model JSON.
type ParseResult struct {
	Inputs    []quizgame.CreateQuestionInput
	Repaired  int
	Dropped   int
	Subject   string
	GradeBand string
}

// GenerateOptions configures the AI call.
type GenerateOptions struct {
	ModelID string
	Prompt  string
}

// Generate calls the provider for structured questions.
func Generate(ctx context.Context, client aiprovider.ScopedCompleter, src SourceMaterial, params quizgame.GenerationParams, opts GenerateOptions) (ModelPayload, aiprovider.CallMeta, error) {
	src = RedactSource(src)
	sys := strings.TrimSpace(opts.Prompt)
	if sys == "" {
		sys = DefaultPrompt
	}
	user := buildUserPrompt(src, params)
	model := strings.TrimSpace(opts.ModelID)
	if model == "" {
		return ModelPayload{}, aiprovider.CallMeta{}, fmt.Errorf("quizgameai: model required")
	}
	text, meta, err := chatJSON(ctx, client, model, sys, user)
	if err != nil {
		return ModelPayload{}, meta, err
	}
	payload, err := ParseModelJSON(text)
	if err != nil {
		repairUser := user + "\n\nYour previous response was invalid JSON. Return ONLY valid JSON matching the schema."
		text2, meta2, err2 := chatJSON(ctx, client, model, sys, repairUser)
		meta = mergeMeta(meta, meta2)
		if err2 != nil {
			return ModelPayload{}, meta, err
		}
		payload, err = ParseModelJSON(text2)
		if err != nil {
			return ModelPayload{}, meta, err
		}
	}
	return payload, meta, nil
}

// RedactSource applies PII redaction to instructor-supplied text.
func RedactSource(src SourceMaterial) SourceMaterial {
	out := src
	out.Topic = aitutor.RedactPII(strings.TrimSpace(src.Topic))
	out.Passage = aitutor.RedactPII(strings.TrimSpace(src.Passage))
	out.ContentTitle = aitutor.RedactPII(strings.TrimSpace(src.ContentTitle))
	out.LikePrompt = aitutor.RedactPII(strings.TrimSpace(src.LikePrompt))
	return out
}

// ParseModelJSON unmarshals and lightly cleans model output.
func ParseModelJSON(text string) (ModelPayload, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	if text == "" {
		return ModelPayload{}, fmt.Errorf("quizgameai: empty model response")
	}
	var payload ModelPayload
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return ModelPayload{}, fmt.Errorf("quizgameai: parse JSON: %w", err)
	}
	return payload, nil
}

// ValidateAndFilter converts drafts into CreateQuestionInput, dropping invalid items.
func ValidateAndFilter(drafts []DraftQuestion, allowedTypes []string, includeExplanations bool, jobID string) ParseResult {
	allowed := map[string]bool{}
	for _, t := range allowedTypes {
		allowed[strings.TrimSpace(strings.ToLower(t))] = true
	}
	out := ParseResult{}
	for _, d := range drafts {
		in, repaired, ok := draftToCreateInput(d, allowed, includeExplanations)
		if !ok {
			out.Dropped++
			continue
		}
		in.Source = quizgame.QuestionSourceAIGenerated
		needs := true
		in.NeedsReview = &needs
		if jobID != "" {
			jid := jobID
			in.GenerationJobID = &jid
		}
		in.GenerationConfidence = d.Confidence
		if repaired {
			out.Repaired++
		}
		out.Inputs = append(out.Inputs, in)
	}
	return out
}

func draftToCreateInput(d DraftQuestion, allowed map[string]bool, includeExplanations bool) (quizgame.CreateQuestionInput, bool, bool) {
	repaired := false
	qtype := strings.TrimSpace(strings.ToLower(d.QuestionType))
	if qtype == "" {
		qtype = quizgame.QTypeMCSingle
		repaired = true
	}
	if len(allowed) > 0 && !allowed[qtype] {
		return quizgame.CreateQuestionInput{}, false, false
	}
	in := quizgame.CreateQuestionInput{
		QuestionType:     qtype,
		Prompt:           d.Prompt,
		Options:          append([]quizgame.Option(nil), d.Options...),
		CorrectAnswer:    d.CorrectAnswer,
		TimeLimitSeconds: d.TimeLimitSeconds,
		PointsStyle:      quizgame.PointsStandard,
	}
	if includeExplanations && d.Explanation != nil {
		e := *d.Explanation
		in.Explanation = &e
	}
	if err := quizgame.NormalizeCreateInput(&in); err != nil {
		return quizgame.CreateQuestionInput{}, false, false
	}
	optsJSON, err := json.Marshal(in.Options)
	if err != nil {
		return quizgame.CreateQuestionInput{}, false, false
	}
	tmp := quizgame.Question{
		QuestionType:     in.QuestionType,
		Prompt:           in.Prompt,
		Options:          optsJSON,
		CorrectAnswer:    in.CorrectAnswer,
		TimeLimitSeconds: in.TimeLimitSeconds,
		PointsStyle:      in.PointsStyle,
		Explanation:      in.Explanation,
	}
	for _, iss := range quizgame.ValidateQuestionReady(tmp) {
		switch iss.Code {
		case "missing_correct", "invalid_option_count", "empty_option", "invalid_options",
			"missing_accepted", "invalid_numeric", "invalid_order", "missing_prompt":
			return quizgame.CreateQuestionInput{}, false, false
		}
	}
	return in, repaired, true
}

func buildUserPrompt(src SourceMaterial, params quizgame.GenerationParams) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Generate exactly %d questions.\n", params.Count)
	fmt.Fprintf(&b, "Allowed question types: %s\n", strings.Join(params.Types, ", "))
	fmt.Fprintf(&b, "Difficulty: %s\n", params.Difficulty)
	if params.GradeBand != "" {
		fmt.Fprintf(&b, "Grade band: %s\n", params.GradeBand)
	}
	fmt.Fprintf(&b, "Language: %s\n", params.Language)
	if params.IncludeExplanations {
		b.WriteString("Include a short explanation (rationale) for each correct answer.\n")
	} else {
		b.WriteString("Omit explanations unless needed for clarity.\n")
	}
	switch src.SourceType {
	case quizgame.GenSourcePassage:
		b.WriteString("Source type: pasted passage. Ground every question in this passage:\n---\n")
		b.WriteString(src.Passage)
		b.WriteString("\n---\n")
	case quizgame.GenSourceCourseContentRef:
		b.WriteString("Source type: course content")
		if src.ContentTitle != "" {
			fmt.Fprintf(&b, " (%s)", src.ContentTitle)
		}
		b.WriteString(". Ground every question in this content:\n---\n")
		b.WriteString(src.Passage)
		b.WriteString("\n---\n")
	default:
		b.WriteString("Source type: topic/prompt.\nTopic: ")
		b.WriteString(src.Topic)
		b.WriteString("\n")
	}
	if src.LikePrompt != "" {
		b.WriteString("\nGenerate questions similar in style/topic to this example question:\n")
		fmt.Fprintf(&b, "Type: %s\nPrompt: %s\n", src.LikeType, src.LikePrompt)
	}
	return b.String()
}

func chatJSON(ctx context.Context, client aiprovider.ScopedCompleter, model, sysPrompt, user string) (string, aiprovider.CallMeta, error) {
	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: user},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return "", meta, err
	}
	text := strings.TrimSpace(res.Text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text), meta, nil
}

func mergeMeta(a, b aiprovider.CallMeta) aiprovider.CallMeta {
	out := a
	if out.Provider == "" {
		out.Provider = b.Provider
	}
	if out.ModelID == "" {
		out.ModelID = b.ModelID
	}
	out.Usage.PromptTokens += b.Usage.PromptTokens
	out.Usage.CompletionTokens += b.Usage.CompletionTokens
	out.Usage.TotalTokens += b.Usage.TotalTokens
	out.Usage.CostUSD += b.Usage.CostUSD
	if b.Usage.CostEstimated {
		out.Usage.CostEstimated = true
	}
	return out
}
