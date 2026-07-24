package assignmentrubricai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// PromptKey is the settings.system_prompts row used for assignment rubric generation.
const PromptKey = "assignment_rubric_generation"

// MaxPromptRunes caps instructor instructions for rubric generation.
const MaxPromptRunes = 8_000

// MaxAssignmentMarkdownRunes caps optional assignment body context.
const MaxAssignmentMarkdownRunes = 200_000

// DefaultSystemPrompt matches settings.system_prompts key assignment_rubric_generation when the row is missing.
const DefaultSystemPrompt = `You generate grading rubrics for LMS assignments. Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object with camelCase keys:
{
  "title": string optional (short heading shown above the rubric),
  "criteria": [
    {
      "title": string (non-empty criterion name),
      "description": string optional (what students should demonstrate),
      "levels": [
        { "label": string (rating column name), "points": number (non-negative, finite), "description": string optional (what this band means for this criterion) }
      ]
    }
  ]
}

Rules:
- Include at least 3 criteria unless the instructor explicitly asks for fewer.
- Every criterion must have the SAME number of "levels" in the SAME ORDER (lowest points first, highest last is typical).
- For each rating column index, the "label" must be the SAME across all criteria (shared column headers).
- Within each criterion, points should usually be non-decreasing as quality improves.
- When assignment points are provided, the sum of each criterion's maximum level points must equal that total exactly.`

// Service provides AI-backed assignment rubric generation.
type Service struct {
	Name string
}

func New() Service {
	return Service{Name: "assignmentrubricai"}
}

// Health returns a stable service heartbeat string for wiring/tests.
func (s Service) Health(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}
	return s.Name + ":ok", nil
}

// GenerateInput is the instructor + assignment context for a rubric draft.
type GenerateInput struct {
	Prompt             string
	AssignmentTitle    string
	PointsWorth        *int
	AssignmentMarkdown string
}

type aiRubricEnvelope struct {
	Title    *string          `json:"title"`
	Criteria []aiCriterionRaw `json:"criteria"`
}

type aiCriterionRaw struct {
	Title       string       `json:"title"`
	Description *string      `json:"description"`
	Levels      []aiLevelRaw `json:"levels"`
}

type aiLevelRaw struct {
	Label       string  `json:"label"`
	Points      float64 `json:"points"`
	Description *string `json:"description"`
}

// Generate asks the model for a draft rubric (not persisted).
func Generate(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt string,
	input GenerateInput,
) (*assignmentrubric.RubricDefinition, aiprovider.CallMeta, error) {
	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("instructions are required")
	}
	if utf8.RuneCountInString(prompt) > MaxPromptRunes {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("instructions are too long (max %d characters)", MaxPromptRunes)
	}
	md := strings.TrimSpace(input.AssignmentMarkdown)
	if utf8.RuneCountInString(md) > MaxAssignmentMarkdownRunes {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("assignment body is too long (max %d characters)", MaxAssignmentMarkdownRunes)
	}

	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}

	pointsLine := "Assignment points worth: not set in the gradebook — choose sensible level points and a coherent total.\n"
	if input.PointsWorth != nil && *input.PointsWorth > 0 {
		pointsLine = fmt.Sprintf(
			"Assignment points worth (the rubric max total must match this exactly): %d points.\n",
			*input.PointsWorth,
		)
	}

	var assignmentBlock string
	if md != "" {
		assignmentBlock = fmt.Sprintf("Full assignment instructions (Markdown):\n---\n%s\n---\n\n", md)
	}

	userBody := fmt.Sprintf(
		"Assignment title (context): %s\n%s\n%sInstructor instructions for the rubric:\n---\n%s\n---\n\nRespond with ONLY a JSON object as described in your system instructions (camelCase).",
		strings.TrimSpace(input.AssignmentTitle),
		pointsLine,
		assignmentBlock,
		prompt,
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

	rubric, err := parseModelJSON(text)
	if err != nil {
		return nil, meta, err
	}
	if err := assignmentrubric.ValidateRubricDefinition(rubric); err != nil {
		return nil, meta, err
	}
	if err := validateAgainstPoints(rubric, input.PointsWorth); err != nil {
		return nil, meta, err
	}
	return rubric, meta, nil
}

func validateAgainstPoints(r *assignmentrubric.RubricDefinition, pointsWorth *int) error {
	if pointsWorth == nil {
		return nil
	}
	v := int32(*pointsWorth)
	return assignmentrubric.ValidateRubricAgainstPointsWorth(r, &v)
}

func parseModelJSON(text string) (*assignmentrubric.RubricDefinition, error) {
	slice := extractJSONObject(text)
	if slice == "" {
		return nil, fmt.Errorf("could not find JSON in the model response")
	}
	var raw aiRubricEnvelope
	if err := json.Unmarshal([]byte(slice), &raw); err != nil {
		return nil, fmt.Errorf("could not parse rubric JSON: %w", err)
	}
	return rawToDefinition(raw)
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

func rawToDefinition(raw aiRubricEnvelope) (*assignmentrubric.RubricDefinition, error) {
	if len(raw.Criteria) == 0 {
		return nil, fmt.Errorf("model returned no rubric criteria")
	}
	title := trimOptional(raw.Title)
	criteria := make([]assignmentrubric.RubricCriterion, 0, len(raw.Criteria))
	for _, c := range raw.Criteria {
		ct := strings.TrimSpace(c.Title)
		if ct == "" {
			return nil, fmt.Errorf("model returned a criterion with an empty title")
		}
		if len(c.Levels) == 0 {
			return nil, fmt.Errorf("model returned a criterion with no rating levels")
		}
		levels := make([]assignmentrubric.RubricLevel, 0, len(c.Levels))
		for _, l := range c.Levels {
			levels = append(levels, assignmentrubric.RubricLevel{
				Label:       strings.TrimSpace(l.Label),
				Points:      l.Points,
				Description: trimOptional(l.Description),
			})
		}
		criteria = append(criteria, assignmentrubric.RubricCriterion{
			ID:          uuid.New(),
			Title:       ct,
			Description: trimOptional(c.Description),
			Levels:      levels,
		})
	}
	out := normalizeRubricGrid(assignmentrubric.RubricDefinition{
		Title:    title,
		Criteria: criteria,
	})
	return &out, nil
}

func trimOptional(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// normalizeRubricGrid pads criteria so every row has the same number of levels and syncs
// level labels from the first row (matches the web editor grid).
func normalizeRubricGrid(r assignmentrubric.RubricDefinition) assignmentrubric.RubricDefinition {
	if len(r.Criteria) == 0 {
		return r
	}
	max := 1
	for _, c := range r.Criteria {
		if len(c.Levels) > max {
			max = len(c.Levels)
		}
	}
	for i := range r.Criteria {
		for len(r.Criteria[i].Levels) < max {
			n := len(r.Criteria[i].Levels) + 1
			r.Criteria[i].Levels = append(r.Criteria[i].Levels, assignmentrubric.RubricLevel{
				Label:  fmt.Sprintf("Rating %d", n),
				Points: 0,
			})
		}
		if len(r.Criteria[i].Levels) > max {
			r.Criteria[i].Levels = r.Criteria[i].Levels[:max]
		}
	}
	refLabels := make([]string, len(r.Criteria[0].Levels))
	for i, lvl := range r.Criteria[0].Levels {
		refLabels[i] = lvl.Label
	}
	for i := 1; i < len(r.Criteria); i++ {
		for j := range r.Criteria[i].Levels {
			if j < len(refLabels) {
				r.Criteria[i].Levels[j].Label = refLabels[j]
			}
		}
	}
	return r
}
