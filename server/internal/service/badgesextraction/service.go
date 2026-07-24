package badgesextraction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/outcomesextraction"
)

// MaxBadges caps how many draft badges are returned.
const MaxBadges = 30

// MaxNameRunes / MaxDescriptionRunes limit individual draft fields.
const (
	MaxNameRunes        = 60
	MaxDescriptionRunes = 4000
)

// OutcomeInput is a course learning outcome used as badge source material.
type OutcomeInput struct {
	ID          string
	Title       string
	Description string
}

// DraftBadge is a proposed badge definition (not persisted).
type DraftBadge struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	OutcomeID   *string `json:"outcomeId,omitempty"`
}

// OutcomesSystemPrompt instructs the model to turn outcomes into short badge names.
const OutcomesSystemPrompt = `You create competency micro-badge drafts from course learning outcomes for an LMS.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object: {"badges":[{"outcomeId":"...","name":"...","description":"..."}]}.

Rules:
- Produce exactly one badge object for each provided learning outcome.
- outcomeId must match the provided outcome id exactly.
- name: a short badge-ready title (ideally under 40 characters, max 60). Shorten long outcome titles while keeping meaning; do not invent a different competency.
- description: a clear 1–3 sentence description of what earning the badge means (learner-facing).
- Prefer concrete verbs and competencies over vague praise.
- If optional syllabus context is provided, use it only to clarify wording — do not invent unrelated badges.`

// SyllabusSystemPrompt instructs the model to extract badge drafts from a syllabus.
const SyllabusSystemPrompt = `You extract competency micro-badge drafts from a course syllabus for an LMS.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object: {"badges":[{"name":"...","description":"..."}]}.

Rules:
- Create one badge per learning outcome / learning objective found (or reasonably inferred) in the syllabus.
- name: a short badge-ready title (ideally under 40 characters, max 60).
- description: a clear 1–3 sentence description of what earning the badge means (learner-facing).
- Prefer outcomes already stated in the syllabus. Do not invent unrelated topics.
- Return between 1 and 30 badges. Prefer quality over quantity.
- If the syllabus has no usable content for badges, return {"badges":[]}.`

// ExtractFromOutcomes asks the model for one short badge draft per learning outcome.
func ExtractFromOutcomes(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model string,
	outcomes []OutcomeInput,
	syllabusMarkdown string,
) ([]DraftBadge, aiprovider.CallMeta, error) {
	if len(outcomes) == 0 {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("outcomes are required")
	}
	var user strings.Builder
	user.WriteString("Create one badge draft for each learning outcome below.\n\n")
	user.WriteString("Learning outcomes (JSON):\n")
	type row struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	rows := make([]row, 0, len(outcomes))
	validIDs := make(map[string]struct{}, len(outcomes))
	for _, o := range outcomes {
		id := strings.TrimSpace(o.ID)
		title := strings.TrimSpace(o.Title)
		if id == "" || title == "" {
			continue
		}
		validIDs[id] = struct{}{}
		rows = append(rows, row{ID: id, Title: title, Description: strings.TrimSpace(o.Description)})
	}
	if len(rows) == 0 {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("outcomes are required")
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return nil, aiprovider.CallMeta{}, err
	}
	user.Write(encoded)
	if md := strings.TrimSpace(syllabusMarkdown); md != "" {
		if utf8.RuneCountInString(md) > outcomesextraction.MaxSyllabusRunes {
			md = string([]rune(md)[:outcomesextraction.MaxSyllabusRunes])
		}
		user.WriteString("\n\nOptional syllabus context:\n")
		user.WriteString(md)
	}

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: OutcomesSystemPrompt},
		{Role: "user", Content: user.String()},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return nil, meta, err
	}
	badges, err := ParseDraftBadgesJSON(res.Text, validIDs)
	if err != nil {
		return nil, meta, err
	}
	return badges, meta, nil
}

// ExtractFromSyllabus asks the model for badge drafts from syllabus markdown.
func ExtractFromSyllabus(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, syllabusMarkdown string,
) ([]DraftBadge, aiprovider.CallMeta, error) {
	md := strings.TrimSpace(syllabusMarkdown)
	if md == "" {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("syllabus content is required")
	}
	if utf8.RuneCountInString(md) > outcomesextraction.MaxSyllabusRunes {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("syllabus is too long (max %d characters)", outcomesextraction.MaxSyllabusRunes)
	}
	user := "Extract competency badge drafts from this course syllabus:\n\n" + md
	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: SyllabusSystemPrompt},
		{Role: "user", Content: user},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return nil, meta, err
	}
	badges, err := ParseDraftBadgesJSON(res.Text, nil)
	if err != nil {
		return nil, meta, err
	}
	return badges, meta, nil
}

// ParseDraftBadgesJSON parses and normalizes model JSON into draft badges.
// When validOutcomeIDs is non-nil, outcomeId values outside the set are dropped (badge kept without link).
func ParseDraftBadgesJSON(raw string, validOutcomeIDs map[string]struct{}) ([]DraftBadge, error) {
	text := stripJSONFences(raw)
	var payload struct {
		Badges []struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			OutcomeID   *string `json:"outcomeId"`
		} `json:"badges"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, fmt.Errorf("parse badges JSON: %w", err)
	}
	out := make([]DraftBadge, 0, len(payload.Badges))
	seenNames := make(map[string]struct{})
	seenOutcomes := make(map[string]struct{})
	for _, b := range payload.Badges {
		name := strings.TrimSpace(b.Name)
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > MaxNameRunes {
			name = string([]rune(name)[:MaxNameRunes])
		}
		desc := strings.TrimSpace(b.Description)
		if utf8.RuneCountInString(desc) > MaxDescriptionRunes {
			desc = string([]rune(desc)[:MaxDescriptionRunes])
		}
		nameKey := strings.ToLower(name)
		if _, ok := seenNames[nameKey]; ok {
			continue
		}
		var outcomeID *string
		if b.OutcomeID != nil {
			oid := strings.TrimSpace(*b.OutcomeID)
			if oid != "" {
				if validOutcomeIDs != nil {
					if _, ok := validOutcomeIDs[oid]; !ok {
						oid = ""
					}
				}
				if oid != "" {
					if _, dup := seenOutcomes[oid]; dup {
						continue
					}
					seenOutcomes[oid] = struct{}{}
					outcomeID = &oid
				}
			}
		}
		seenNames[nameKey] = struct{}{}
		out = append(out, DraftBadge{Name: name, Description: desc, OutcomeID: outcomeID})
		if len(out) >= MaxBadges {
			break
		}
	}
	return out, nil
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
