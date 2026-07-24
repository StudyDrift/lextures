package quizoutcomesmapping

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const (
	MaxSuggestions      = 120
	MaxPromptMaterial   = 80_000
	MaxRationaleRunes   = 400
	MaxQuestionPrompt   = 800
	defaultMeasurement  = "formative"
	defaultIntensity    = "medium"
)

// OutcomeInput is a course learning outcome offered for mapping.
type OutcomeInput struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// QuestionInput is a quiz question offered for mapping.
type QuestionInput struct {
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
}

// SuggestInput is the material sent to the model.
type SuggestInput struct {
	QuizTitle string
	QuizIntro string
	Outcomes  []OutcomeInput
	Questions []QuestionInput
}

// DraftSuggestion is a proposed outcome link (not persisted).
type DraftSuggestion struct {
	TargetKind       string `json:"targetKind"`
	QuizQuestionID   string `json:"quizQuestionId"`
	OutcomeID        string `json:"outcomeId"`
	MeasurementLevel string `json:"measurementLevel"`
	IntensityLevel   string `json:"intensityLevel"`
	Rationale        string `json:"rationale"`
}

// DefaultSystemPrompt instructs the model to return structured mapping JSON only.
const DefaultSystemPrompt = `You map a quiz and its questions to course learning outcomes for an LMS.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object:
{"suggestions":[{"targetKind":"quiz"|"quiz_question","quizQuestionId":"...","outcomeId":"...","measurementLevel":"diagnostic"|"formative"|"summative"|"performance","intensityLevel":"low"|"medium"|"high","rationale":"..."}]}

Rules:
- Use only outcomeId values from the provided outcomes list (exact match).
- For whole-quiz mappings use targetKind "quiz" and quizQuestionId "".
- For per-question mappings use targetKind "quiz_question" and a quizQuestionId from the provided questions list (exact match).
- Prefer the strongest relevant outcomes; do not force weak matches.
- Map the whole quiz when it clearly assesses one or more outcomes overall.
- Map individual questions when the prompt clearly aligns with an outcome.
- A quiz or question may link to multiple outcomes when justified; avoid redundant near-duplicates.
- measurementLevel: diagnostic (pre-check), formative (practice/feedback), summative (graded mastery), performance (authentic/applied).
- intensityLevel: how strongly this item evidences the outcome (low/medium/high).
- rationale: short instructor-facing reason (one sentence).
- If nothing maps well, return {"suggestions":[]}.
- Return at most 120 suggestions.`

// Suggest asks the model for draft quiz/question → outcome links.
func Suggest(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt string,
	in SuggestInput,
) ([]DraftSuggestion, aiprovider.CallMeta, error) {
	if len(in.Outcomes) == 0 {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("outcomes are required")
	}

	validOutcomes := make(map[string]struct{}, len(in.Outcomes))
	outcomes := make([]OutcomeInput, 0, len(in.Outcomes))
	for _, o := range in.Outcomes {
		id := strings.TrimSpace(o.ID)
		title := strings.TrimSpace(o.Title)
		if id == "" || title == "" {
			continue
		}
		validOutcomes[id] = struct{}{}
		desc := strings.TrimSpace(o.Description)
		outcomes = append(outcomes, OutcomeInput{ID: id, Title: title, Description: desc})
	}
	if len(outcomes) == 0 {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("outcomes are required")
	}

	validQuestions := make(map[string]struct{}, len(in.Questions))
	questions := make([]QuestionInput, 0, len(in.Questions))
	for _, q := range in.Questions {
		id := strings.TrimSpace(q.ID)
		if id == "" {
			continue
		}
		prompt := strings.TrimSpace(q.Prompt)
		if utf8.RuneCountInString(prompt) > MaxQuestionPrompt {
			prompt = string([]rune(prompt)[:MaxQuestionPrompt]) + "…"
		}
		validQuestions[id] = struct{}{}
		questions = append(questions, QuestionInput{ID: id, Prompt: prompt})
	}

	payload := map[string]any{
		"quizTitle": strings.TrimSpace(in.QuizTitle),
		"quizIntro": truncateRunes(strings.TrimSpace(in.QuizIntro), 4000),
		"outcomes":  outcomes,
		"questions": questions,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, aiprovider.CallMeta{}, err
	}
	user := "Suggest outcome mappings for this quiz. Input JSON:\n" + string(encoded)
	if utf8.RuneCountInString(user) > MaxPromptMaterial {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("quiz mapping prompt is too long (max %d characters)", MaxPromptMaterial)
	}

	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return nil, meta, err
	}
	suggestions, err := ParseSuggestionsJSON(res.Text, validOutcomes, validQuestions)
	if err != nil {
		return nil, meta, err
	}
	return suggestions, meta, nil
}

// ParseSuggestionsJSON parses and normalizes model JSON into draft suggestions.
func ParseSuggestionsJSON(
	raw string,
	validOutcomeIDs map[string]struct{},
	validQuestionIDs map[string]struct{},
) ([]DraftSuggestion, error) {
	text := stripJSONFences(raw)
	var payload struct {
		Suggestions []DraftSuggestion `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, fmt.Errorf("parse outcome mapping JSON: %w", err)
	}
	return normalizeSuggestions(payload.Suggestions, validOutcomeIDs, validQuestionIDs), nil
}

func normalizeSuggestions(
	in []DraftSuggestion,
	validOutcomeIDs map[string]struct{},
	validQuestionIDs map[string]struct{},
) []DraftSuggestion {
	out := make([]DraftSuggestion, 0, len(in))
	seen := make(map[string]struct{})
	for _, s := range in {
		kind := strings.TrimSpace(s.TargetKind)
		outcomeID := strings.TrimSpace(s.OutcomeID)
		if outcomeID == "" {
			continue
		}
		if _, ok := validOutcomeIDs[outcomeID]; !ok {
			continue
		}
		qID := strings.TrimSpace(s.QuizQuestionID)
		switch kind {
		case "quiz":
			qID = ""
		case "quiz_question":
			if qID == "" {
				continue
			}
			if _, ok := validQuestionIDs[qID]; !ok {
				continue
			}
		default:
			continue
		}
		measurement := strings.TrimSpace(strings.ToLower(s.MeasurementLevel))
		if !containsString(courseoutcomes.MeasurementLevels, measurement) {
			measurement = defaultMeasurement
		}
		intensity := strings.TrimSpace(strings.ToLower(s.IntensityLevel))
		if !containsString(courseoutcomes.IntensityLevels, intensity) {
			intensity = defaultIntensity
		}
		rationale := strings.TrimSpace(s.Rationale)
		if utf8.RuneCountInString(rationale) > MaxRationaleRunes {
			rationale = string([]rune(rationale)[:MaxRationaleRunes])
		}
		key := kind + "|" + qID + "|" + outcomeID + "|" + measurement + "|" + intensity
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, DraftSuggestion{
			TargetKind:       kind,
			QuizQuestionID:   qID,
			OutcomeID:        outcomeID,
			MeasurementLevel: measurement,
			IntensityLevel:   intensity,
			Rationale:        rationale,
		})
		if len(out) >= MaxSuggestions {
			break
		}
	}
	return out
}

func containsString(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
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
